# OPM kernel

The reference implementation of the Open Platform Model runtime, packaged as a Go library. Every OPM front-end — the `opm` CLI, the `opm-operator` controller, the planned Crossplane composition function, and any future runtime — embeds this kernel and inherits its behaviour.

The kernel owns:

- Loading OPM artifacts (modules, platforms, releases) from CUE module directories and `.cue` files.
- Resolving CUE module references through the native CUE module system (OCI registries, `cue.mod`).
- Validating user-supplied values against `#config` schemas with grouped, position-aware diagnostics.
- Matching component requirements against the active Platform's `#matchers` index.
- Executing matched transformers (resolved by FQN against `Platform.#composedTransformers`) and emitting platform-neutral rendered values with full provenance.

The kernel does **not** own:

- Process model, command flags, exit codes, stdout/stderr formatting (lives in CLI / controller).
- Logging output (loggers are passed in by the caller).
- Cluster reconciliation, status reporting, GitOps wiring (lives in `opm-operator`).
- Platform-native identity beyond the `core.Identity` tuple — adapters wrap rendered values into platform-specific resources.
- Debug-overlay policy. `#ModuleDebug` is **not** a kernel artifact; the kernel accepts only `Module`, `ModuleRelease`, and `Platform` (see "Artifact types" below). Debug values live as a `debugValues` field on `Module` itself; whether the frontend layers them into the values stack is policy that lives in the helper layer (CLI / operator / XR fn).

## Artifact types

The kernel accepts exactly three artifact types — every input ultimately resolves to one of them:

| Artifact         | Schema definition          | Go type              | Role                                                                                          |
| ---------------- | -------------------------- | -------------------- | --------------------------------------------------------------------------------------------- |
| `Module`         | `#Module` (v1alpha2)       | `*module.Module`     | Author-defined application blueprint (components, `#config` schema, `debugValues` field).     |
| `ModuleRelease`  | `#ModuleRelease`           | `*module.Release`    | Per-deployment instantiation of a `Module` with concrete user values.                         |
| `Platform`       | `#Platform`                | `*platform.Platform` | Composed registry of Modules; supplies `#composedTransformers` and `#matchers` to the kernel. |

`#ModuleDebug` was previously contemplated as a fourth top-level artifact and has been **retired**. The migration is one line: read `mod.Package.LookupPath(b.Paths().DebugValues)` (with `b, _ := api.Lookup(mod.APIVersion)`) and feed the result into the helper-side values stack at the layer your frontend prefers. The kernel itself never observes the distinction.

See `CONSTITUTION.md` for the full set of principles.

## Layout

```
apis/                     Versioned OPM schema (CUE)
  core/                   CUE module rooted here (cue.mod/module.cue) + embed.go for all versions
    v1alpha2/             Schema for v1alpha2 — single source of truth for that version
opm/
  api/                    Per-schema-version Binding interface, process-wide registry, embed plumbing
  api/v1alpha2/           v1alpha2 binding (registers itself in init())
  apiversion/             apiVersion enum + Detect helper
  core/                   Platform-neutral primitives — Compiled, Resource, Identity
  errors/                 Sentinels, structured errors, grouped CUE diagnostics
  kernel/                 Public Kernel struct — single entry point for the OPM runtime
  module/                 Module / Release model and value-validation accessors
  platform/               Platform artifact model — kernel's sole input for matching and execution
  compile/                Match -> finalize -> execute -> emit pipeline
  helper/                 Opt-in frontend convenience layer (a frontend MAY skip these)
    loader/file/          Filesystem-coupled loading (modules, releases, platforms)
    loader/bytes/         In-memory loading (skeleton; deferred implementation)
    platform/             Platform composition (shell + modules -> composed Platform)
    synth/                Release synthesis from typed inputs (no file / no bytes)
cmd/
  flow-inspect/           Internal diagnostic CLI for the compile pipeline
adr/                      Architecture decision records
enhancements/             Long-form design proposals (umbrella + slices)
openspec/                 OpenSpec proposals, specs, archives
modules/                  Test-only OPM modules used by integration tests
testdata/                 CUE module fixtures consumed by package tests
Taskfile.yml              fmt / vet / lint / test entry points
```

The `opm/loader/` deprecation shim has been removed; the canonical import path is `opm/helper/loader/file`. A standalone `opm/validate/` package was contemplated but never landed — validation primitives live on `*kernel.Kernel` (`ValidateConfig`, `ValidateConfigPartial`, `ValidateConfigDetailed`) plus the typed shortcuts in `opm/kernel/validate_typed.go`.

## Compile pipeline

```
loaderfile.LoadReleasePackage  ->  cue.Value (release artifact)
Kernel.ProcessModuleRelease    ->  *module.Release          (validated, concrete)
Kernel.Compile                 ->  *kernel.CompileResult    (rendered + provenance)
        |
        +-- compile.FinalizeValue   strip schema constraints from components
        +-- compile.Match           component <-> transformer pairing (paired output)
        +-- compile.Module.Execute  per-pair transformer execution
                |
                +-- FillPath #component, #context.{moduleReleaseMetadata, componentMetadata, runtimeName}
                +-- decode `output` (kind-based dispatch: ListKind | StructKind)
                +-- emit []*core.Compiled carrying Release/Component/Transformer FQN provenance
```

The kernel exposes four phase-explicit methods that map onto frontend subcommands: `Kernel.Validate` (vet), `Kernel.Match` (match), `Kernel.Plan` (plan / preview), and `Kernel.Compile` (apply / render). The old free-function entry points (`compile.CompileModuleRelease`, `compile.ProcessModuleRelease`, `module.ParseModuleRelease`) have been removed — construct a `Kernel` and call its methods directly.

`*core.Compiled` is the kernel's terminal output. Adapters in downstream implementations wrap each `Compiled` with a platform-specific `core.Resource` that fills `core.Identity`.

## Quick start

See [`docs/getting-started.md`](docs/getting-started.md) for an end-to-end walkthrough — constructing a `Kernel`, loading a Module, layered values validation, Platform composition, and compiling a Release into rendered `*core.Compiled` values.

## API stability

The library follows SemVer 2.0.0. The public surface is everything under `opm/`. Two distinct compatibility tracks coexist and must not be confused:

- **Go module SemVer** governs the Go types and function signatures consumed by downstream binaries. A breaking change here is a major bump of the library.
- **OPM schema versioning** governs the CUE shapes consumed at runtime — `#Module`, `#ModuleRelease`, `#Platform`, `#Component`, transformer contracts. The kernel MUST be able to load and render older schema versions seamlessly so that downstream implementations inherit multi-version support without per-implementation effort.

The two tracks are independent: a kernel `v1.4.0` may simultaneously support multiple OPM schema versions. Today only `v1alpha2` is shipped — `v1alpha1` was retired in commit `3a9a9bd` — but the multi-version machinery remains in place for the next version cut.

## Multi-version OPM schema support

The kernel dispatches on each artifact's `apiVersion` literal. Adding a new schema version (`v1beta1`, `v1`, ...) is a localised change: drop a new directory under `apis/core/<vN>/` (and add it to the `//go:embed` pattern in `apis/core/embed.go`), then add a sibling Go package under `opm/api/<vN>/` that registers a `Binding` in its `init()`. The new version coexists at runtime with every other registered binding. `opm/compile`, `opm/helper/loader/file`, and `opm/module` need no edits.

Key pieces:

- `opm/apiversion` — `Version` type, registered constants, and `Detect(cue.Value)` that reads the `apiVersion` field off any artifact root.
- `opm/api` — `Binding` interface (`Paths`, decoders, `BuildTransformerContext`, `EmbeddedSchema`, `SchemaValue`) plus a process-wide registry. `Register` panics on duplicate; `Lookup` and `For` return errors that wrap `apiversion.ErrUnknownAPIVersion`.
- `opm/api/v1alpha2` — the v1alpha2 binding. Registers itself in `init()` and exposes the `apis/core/v1alpha2/` schema as a `go:embed` filesystem.
- `apis/core/embed.go` — single `//go:embed` directive at the core CUE module root that pulls in `cue.mod/module.cue` plus every versioned schema package below it, so the kernel can validate artifacts deterministically without touching `CUE_REGISTRY`.

The compile pipeline resolves the binding once per release (via `api.Lookup(rel.APIVersion)`) and threads it through `Match`, `Execute`, and the per-pair context-injection step. See `MIGRATIONS.md` and the archived OpenSpec change `add-multi-apiversion-support` for the full design notes.

## Helper boundary (`opm/helper/`)

Anything under `opm/helper/` is opt-in convenience for embedding the kernel; a frontend MAY skip it and call the kernel directly. Anything outside `opm/helper/` is part of the kernel contract.

Today this layer holds:

- `opm/helper/loader/file` — filesystem-coupled loaders: `LoadModulePackage`, `LoadReleasePackage`, `LoadPlatformFile`. Modules and releases both load as CUE packages (unified in commit `7c435f2`); only platforms still load from a single `.cue` file.
- `opm/helper/loader/bytes` — in-memory loader. **Skeleton only**, no exported functions yet. The full implementation lands when a concrete consumer (Crossplane composition fn, fuzzing harness) pulls on the design.
- `opm/helper/platform` — Platform composition (`Compose`): takes a shell Platform plus a slice of `*module.Module` and `FillPath`-injects each into `#registry` so the schema's computed views resolve.
- `opm/helper/synth` — Release synthesis (`Release`): build a `ModuleRelease` CUE value from typed inputs (name, namespace, module reference, values, labels, annotations) without round-tripping through a file. Pairs with `Kernel.SynthesizeRelease`, which chains synth + validate in one call.

Layered values validation lives on the kernel itself — see `Kernel.ValidateConfigDetailed` and the `Source` type in `opm/kernel`. See `enhancements/001-kernel-redesign-around-platform/02-design.md`.

The previous `opm/loader/` deprecation shim has been removed (commit `3a9a9bd`); the canonical import path is `opm/helper/loader/file`.

## Quality gates

```
task fmt
task vet
task lint
task test
# or all four
task check
```

## Further reading

- `CONSTITUTION.md` — design principles (kernel neutrality, type safety, separation of concerns, SemVer discipline, small batches).
- `openspec/config.yaml` — normative constitution source.
- `apis/core/v1alpha2/` — current OPM schema definitions in CUE.
- `docs/getting-started.md` — end-to-end embedding walkthrough.
- `docs/design/` — flow diagrams and pipeline notes (`kernel-validate-flow.md`, `compile-pipeline-known-gaps.md`).
- `enhancements/` — long-form design proposals (kernel redesign, compiler/runtime split, platform construct, module context, claims).
- `adr/` — architecture decision records.
- `CHANGELOG.md` — released-version history (generated by release-please).
- `MIGRATIONS.md` — pre-release API evolution and breaking-change migration recipes.
