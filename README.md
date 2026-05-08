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
  core/v1alpha1/          Schema for v1alpha1 — single source of truth for that version
  core/v1alpha2/          Schema for v1alpha2 — single source of truth for that version
pkg/
  api/                    Per-schema-version Binding interface and registry
  api/v1alpha2/           v1alpha2 binding (registers itself in init())
  apiversion/             apiVersion enum + Detect helper
  core/                   Platform-neutral primitives — Rendered, Resource, Identity
  errors/                 Sentinels, structured errors, grouped CUE diagnostics
  kernel/                 Public Kernel struct — single entry point for the OPM runtime
  loader/                 Deprecated re-export shim of pkg/helper/loader/file (kept for one SemVer cycle)
  module/                 Module / Release model, parsing, value validation entry point
  platform/               Platform artifact model — kernel's sole input for matching and execution
  compile/                Match -> finalize -> execute -> emit pipeline
  validate/               #config validation against supplied values
  helper/                 Opt-in frontend convenience layer (a frontend MAY skip these)
    loader/file/          Filesystem-coupled loading (modules, providers, releases)
    loader/bytes/         In-memory loading (skeleton; deferred implementation)
openspec/                 OpenSpec proposals, specs, archives
Taskfile.yml              fmt / vet / lint / test entry points
```

## Compile pipeline

```
loader.LoadReleaseFile        ->  cue.Value (release artifact)
module.ParseModuleRelease     ->  *module.Release          (validated, concrete)
kernel.Compile                ->  *kernel.CompileResult    (rendered + provenance)
        |
        +-- compile.FinalizeValue   strip schema constraints from components
        +-- compile.Match           component <-> transformer pairing
        +-- compile.executeTransforms
                |
                +-- FillPath #component, #context.{moduleReleaseMetadata, componentMetadata, runtimeName}
                +-- decode `output` (cue.ListKind | cue.StructKind)
                +-- emit []*core.Rendered carrying Release/Component/Transformer FQN provenance
```

The kernel exposes four phase-explicit methods that map onto frontend
subcommands: `Kernel.Validate` (vet), `Kernel.Match` (match),
`Kernel.Plan` (plan / preview), and `Kernel.Compile` (apply / render).
`compile.CompileModuleRelease` and `compile.ProcessModuleRelease` (an alias
for the former) remain as deprecated free-function entry points.

`*core.Rendered` is the kernel's terminal output. Adapters in downstream implementations wrap each `Rendered` with a platform-specific `core.Resource` that fills `core.Identity`.

## Quick start

The recommended entry point is the `kernel.Kernel` struct, which owns its
`*cue.Context` and threads cross-cutting dependencies (logger, tracer, clock)
through every operation. Construct one Kernel per goroutine.

```go
import (
    "context"

    "github.com/open-platform-model/library/pkg/kernel"
    "github.com/open-platform-model/library/pkg/module"
    loader "github.com/open-platform-model/library/pkg/helper/loader/file"
)

k := kernel.New() // optional: kernel.New(kernel.WithLogger(myLogger))

// Load the module CUE package and build a typed *module.Module.
moduleVal, _, err := k.LoadModulePackage(ctx, "./module/")
mod, err := k.NewModuleFromValue(moduleVal)

// Load and parse the release. The release's Package embeds the source #module
// reference; ParseModuleRelease uses it to validate user values against
// #module.#config without a separate schema argument.
releaseVal, _, _, err := k.LoadReleaseFile(ctx, "./release.cue", loader.LoadOptions{})
rel, err := k.ParseModuleRelease(ctx, releaseVal, *mod, []cue.Value{userValues})

// Load the Platform — the kernel's matching and execution input. The
// shell is a Platform whose #registry is empty (or partial); Compose
// FillPath-injects each registered Module so the schema's computed views
// (#composedTransformers, #matchers, #knownResources, #knownTraits)
// resolve. Frontends that load a fully-authored platform.cue can skip
// Compose and use NewPlatformFromValue directly.
shellVal, _, err := k.LoadPlatformFile(ctx, "./platform.cue", loader.LoadOptions{})
shell, err := k.NewPlatformFromValue(shellVal)
plat, err := k.ComposePlatform(shell, []*module.Module{mod /* + others */})

result, err := k.Compile(ctx, kernel.CompileInput{
    Module:        mod,
    ModuleRelease: rel,
    Values:        userValues,
    Platform:      plat,
    RuntimeName:   "opm-cli",
})
for _, r := range result.Rendered {
    // r.Value is concrete, fully evaluated CUE — encode to YAML/JSON
}
```

The free-function form is `// Deprecated:`-marked. New consumers should
construct a `Kernel` and call `Compile`.

```go
import (
    "cuelang.org/go/cue/cuecontext"

    loader "github.com/open-platform-model/library/pkg/helper/loader/file"
    "github.com/open-platform-model/library/pkg/compile"
    "github.com/open-platform-model/library/pkg/module"
    "github.com/open-platform-model/library/pkg/platform"
)

cueCtx := cuecontext.New()
releaseVal, _, ver, err := loader.LoadReleaseFile(cueCtx, "./release.cue", loader.LoadOptions{})
rel, err := module.ParseModuleRelease(ctx, releaseVal, mod, []cue.Value{userValues})

platformVal, _, err := loader.LoadPlatformFile(cueCtx, "./platform.cue", loader.LoadOptions{})
plat, err := platform.NewPlatformFromValue(nil, platformVal)
result, err := compile.CompileModuleRelease(ctx, rel, plat, "opm-cli")
```

## API stability

The library follows SemVer 2.0.0. The public surface is everything under `pkg/`. Two distinct compatibility tracks coexist and must not be confused:

- **Go module SemVer** governs the Go types and function signatures consumed by downstream binaries. A breaking change here is a major bump of the library.
- **OPM schema versioning** governs the CUE shapes consumed at runtime — `#Module`, `#ModuleRelease`, `#Platform`, `#Component`, transformer contracts. The kernel MUST be able to load and render older schema versions seamlessly so that downstream implementations inherit multi-version support without per-implementation effort.

The two tracks are independent: a kernel `v1.4.0` may simultaneously support OPM schema versions `v1alpha1` and `v1alpha2`.

## Multi-version OPM schema support

The kernel dispatches on each artifact's `apiVersion` literal. Adding a new schema version (`v1beta1`, `v1`, ...) is a localised change: drop a new directory under `apis/core/<vN>/` and a sibling Go package under `pkg/api/<vN>/`, and the new version coexists at runtime with every other registered binding. `pkg/compile`, `pkg/loader`, and `pkg/module` need no edits.

Key pieces:

- `pkg/apiversion` — `Version` type, registered constants, and `Detect(cue.Value)` that reads the `apiVersion` field off any artifact root.
- `pkg/api` — `Binding` interface (`Paths`, decoders, `BuildTransformerContext`, `EmbeddedSchema`) plus a process-wide registry. `Register` panics on duplicate; `Lookup` and `For` return errors that wrap `apiversion.ErrUnknownAPIVersion`.
- `pkg/api/v1alpha2` — the v1alpha2 binding. Registers itself in `init()` and exposes the `apis/core/v1alpha2/` schema as a `go:embed` filesystem.
- `apis/core/<vN>/embed.go` — embeds that version's CUE source so the kernel can validate artifacts deterministically without touching `CUE_REGISTRY`.

The compile pipeline resolves the binding once per release (via `api.Lookup(rel.APIVersion)`) and threads it through `Match`, `Execute`, and the per-pair context-injection step. See `CHANGELOG.md` and the archived OpenSpec change `add-multi-apiversion-support` for the full design notes.

## Helper boundary (`pkg/helper/`)

Anything under `pkg/helper/` is opt-in convenience for embedding the kernel; a frontend MAY skip it and call the kernel directly. Anything outside `pkg/helper/` is part of the kernel contract.

Today this layer holds the filesystem loader (`pkg/helper/loader/file`) and a skeleton for the in-memory loader (`pkg/helper/loader/bytes`). Future slices add layered values (`pkg/helper/values`) and Platform composition (`pkg/helper/platform`); see `enhancements/001-kernel-redesign-around-platform/02-design.md`.

The old `pkg/loader/` import path remains as a deprecation shim that re-exports `pkg/helper/loader/file/` symbols for one SemVer cycle. New code SHOULD import the new path.

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
- `apis/v1alpha2/core/` — current OPM schema definitions in CUE.
