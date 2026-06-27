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
- Debug-overlay policy. `#ModuleDebug` is **not** a kernel artifact; the kernel accepts only `Module`, `ModuleInstance`, and `Platform` (see "Artifact types" below). Debug values live as a `debugValues` field on `Module` itself; whether the frontend layers them into the values stack is policy that lives in the helper layer (CLI / operator / XR fn).

## Artifact types

The kernel accepts exactly three artifact types — every input ultimately resolves to one of them:

| Artifact         | Schema definition          | Go type              | Role                                                                                          |
| ---------------- | -------------------------- | -------------------- | --------------------------------------------------------------------------------------------- |
| `Module`         | `#Module` (v1alpha2)       | `*module.Module`     | Author-defined application blueprint (components, `#config` schema, `debugValues` field).     |
| `ModuleInstance`  | `#ModuleInstance`           | `*module.Instance`    | Per-deployment instantiation of a `Module` with concrete user values.                         |
| `Platform`       | `#Platform`                | `*platform.Platform` | Composed registry of Modules; supplies `#composedTransformers` and `#matchers` to the kernel. |

`#ModuleDebug` was previously contemplated as a fourth top-level artifact and has been **retired**; `debugValues` is now a field on `Module`. The migration is one line: read `mod.Package.LookupPath(schema.DebugValues)` and feed the result into the helper-side values stack at the layer your frontend prefers. The kernel itself never observes the distinction.

See `CONSTITUTION.md` for the full set of principles.

## Layout

```
opm/
  core/                   Platform-neutral primitives — Compiled, Resource, Identity
  errors/                 Sentinels, structured errors, grouped CUE diagnostics
  schema/                 OPM core schema loader (OCILoader, Cache), CUE path inventory, metadata decoders
  kernel/                 Public Kernel struct — single entry point for the OPM runtime
  module/                 Module / Instance model and value-validation accessors
  platform/               Platform artifact model — kernel's sole input for matching and execution
  compile/                finalize -> match -> execute -> emit pipeline
  materialize/            Resolve a Platform's #registry subscriptions into a sealed MaterializedPlatform
  helper/                 Opt-in frontend convenience layer (a frontend MAY skip these)
    loader/file/          Filesystem loading (modules, releases, platforms)
    loader/registry/      Load a published module from an OCI registry by path@version
    loader/bytes/         In-memory loading (skeleton; deferred implementation)
    loader/internal/shape Shared artifact shape gate (single-sourced across loaders)
    synth/                Instance / Platform synthesis from typed inputs (no file / no bytes)
  internal/               Test-only cross-package internals (schematest, registrytest)
cmd/
  flow-inspect/           Internal diagnostic CLI for the compile pipeline
adr/                      Architecture decision records
enhancements/             Long-form design proposals (umbrella + slices)
openspec/                 OpenSpec proposals, specs, archives
modules/                  Test-only OPM modules used by integration tests
testdata/                 CUE module fixtures consumed by package tests
Taskfile.yml              fmt / vet / lint / test entry points
```

The OPM core schema is no longer vendored or embedded — it is fetched at runtime from `CUE_REGISTRY` via `opm/schema` (the `apis/` tree and the old `opm/api` / `opm/apiversion` packages were removed). The `opm/loader/` deprecation shim is also gone; the canonical import path is `opm/helper/loader/file` (or `opm/helper/loader/registry` for published modules). A standalone `opm/validate/` package was contemplated but never landed — validation primitives live on `*kernel.Kernel` (`ValidateConfig`, `ValidateConfigPartial`, `ValidateConfigDetailed`) plus the typed shortcuts in `opm/kernel/validate_typed.go`.

## Compile pipeline

```
loaderfile.LoadInstancePackage  ->  cue.Value (release artifact)
Kernel.ProcessModuleInstance    ->  *module.Instance          (validated, concrete)
Kernel.Compile                 ->  *kernel.CompileResult    (rendered + provenance)
        |
        +-- compile.FinalizeValue   strip schema constraints from components
        +-- compile.Match           component <-> transformer pairing (paired output)
        +-- compile.Module.Execute  per-pair transformer execution
                |
                +-- FillPath #component, #context.{moduleInstanceMetadata, componentMetadata, runtimeName}
                +-- decode `output` (kind-based dispatch: ListKind | StructKind)
                +-- emit []*core.Compiled carrying Instance/Component/Transformer FQN provenance
```

The kernel exposes four phase-explicit methods that map onto frontend subcommands: `Kernel.Validate` (vet), `Kernel.Match` (match), `Kernel.Plan` (plan / preview), and `Kernel.Compile` (apply / render). The old free-function entry points (`compile.CompileModuleInstance`, `compile.ProcessModuleInstance`, `module.ParseModuleInstance`) have been removed — construct a `Kernel` and call its methods directly.

`*core.Compiled` is the kernel's terminal output. Adapters in downstream implementations wrap each `Compiled` with a platform-specific `core.Resource` that fills `core.Identity`.

## Quick start

See [`docs/getting-started.md`](docs/getting-started.md) for an end-to-end walkthrough — constructing a `Kernel`, loading a Module, layered values validation, Platform composition, and compiling a Instance into rendered `*core.Compiled` values.

## API stability

The library follows SemVer 2.0.0. The public surface is everything under `opm/`. Two distinct compatibility tracks coexist and must not be confused:

- **Go module SemVer** governs the Go types and function signatures consumed by downstream binaries. A breaking change here is a major bump of the library.
- **OPM schema versioning** governs the CUE shapes consumed at runtime — `#Module`, `#ModuleInstance`, `#Platform`, `#Component`, transformer contracts. The kernel MUST be able to load and render older schema versions seamlessly so that downstream implementations inherit multi-version support without per-implementation effort.

The two tracks are independent: within an OPM schema major (`@v0`), additive shape changes are absorbed by floating-major resolution and require no Go-side bump; a shape break in the schema is itself a coordinated library-breaking event.

## OPM schema resolution

The library does NOT vendor or embed the OPM core schema. At runtime the kernel resolves `opmodel.dev/core@v1` through CUE's module system against `CUE_REGISTRY`, then memoizes the built `cue.Value` in a per-`Kernel` `*schema.Cache`.

Key pieces:

- `opm/schema` — schema loader (`Loader` interface, `OCILoader` sole public implementation), per-instance memoization (`Cache`), CUE path inventory, metadata decoders, and the `PublicRegistry` const (`opmodel.dev=ghcr.io/open-platform-model,registry.cue.works`).
- `opm/kernel` — `kernel.WithSchemaLoader(schema.Loader)` configures which Loader the Kernel's cache wraps; `(*Kernel).SchemaCache()` exposes the cache to release-synthesis and other callers.

Frontends (CLI, operator, future Crossplane fn) set `CUE_REGISTRY` (typically to `schema.PublicRegistry`) before constructing the Kernel. The library auto-applies no default; this keeps Principle I (kernel neutrality) intact and avoids hidden lookups. See `docs/getting-started.md` and `MIGRATIONS.md` for the deployment pattern, including the warm-cache pre-seeding pattern for restricted environments.

## Helper boundary (`opm/helper/`)

Anything under `opm/helper/` is opt-in convenience for embedding the kernel; a frontend MAY skip it and call the kernel directly. Anything outside `opm/helper/` is part of the kernel contract.

Today this layer holds:

- `opm/helper/loader/file` — filesystem-coupled loaders: `LoadModulePackage`, `LoadInstancePackage`, `LoadPlatformFile`. Modules and releases both load as CUE packages (unified in commit `7c435f2`); only platforms still load from a single `.cue` file.
- `opm/helper/loader/bytes` — in-memory loader. **Skeleton only**, no exported functions yet. The full implementation lands when a concrete consumer (Crossplane composition fn, fuzzing harness) pulls on the design.
- `opm/helper/platform` — Platform composition (`Compose`): takes a shell Platform plus a slice of `*module.Module` and `FillPath`-injects each into `#registry` so the schema's computed views resolve.
- `opm/helper/synth` — Instance synthesis (`Instance`): build a `ModuleInstance` CUE value from typed inputs (name, namespace, module reference, values, labels, annotations) without round-tripping through a file. Pairs with `Kernel.SynthesizeInstance`, which chains synth + validate in one call.

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
- `opmodel.dev/core@v1` — current OPM schema, published as an OCI CUE module (sources live in the workspace `core/` repo).
- `docs/getting-started.md` — end-to-end embedding walkthrough.
- `docs/design/` — flow diagrams and pipeline notes (`kernel-validate-flow.md`, `compile-pipeline-known-gaps.md`).
- `enhancements/` — long-form design proposals (kernel redesign, compiler/runtime split, platform construct, module context, claims).
- `adr/` — architecture decision records.
- `CHANGELOG.md` — released-version history (generated by release-please).
- `MIGRATIONS.md` — pre-release API evolution and breaking-change migration recipes.
