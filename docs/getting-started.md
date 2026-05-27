# Getting started

This guide walks through embedding the OPM kernel in a Go program: loading a Module, validating user values against its `#config` schema, composing a Platform, and compiling a Release down to rendered `*core.Compiled` values.

The recommended entry point is the `kernel.Kernel` struct, which owns its `*cue.Context` and threads cross-cutting dependencies (logger, tracer, clock) through every operation. **Construct one Kernel per goroutine** — the underlying `*cue.Context` is not safe for concurrent use.

## Prerequisites

- Go 1.22+
- A CUE module containing a `Module` artifact (and optionally a `ModuleRelease` artifact and a `Platform` artifact).
- `CUE_REGISTRY` configured. The kernel resolves the OPM core schema (`opmodel.dev/core@v0`) at runtime through CUE's module system, and your own modules go through the same mechanism. The library does NOT auto-set `CUE_REGISTRY`; configure it explicitly before constructing the Kernel.

## Configure CUE_REGISTRY

The library exports `schema.PublicRegistry` as the documented mapping for the canonical GHCR location:

```go
import (
    "os"

    "github.com/open-platform-model/library/opm/schema"
)

// Once at startup, before kernel.New:
os.Setenv("CUE_REGISTRY", schema.PublicRegistry)
// → "opmodel.dev=ghcr.io/open-platform-model,registry.cue.works"
```

Operators in air-gapped environments set `CUE_REGISTRY` to an internal mirror, or pre-seed `$CUE_CACHE_DIR` with the extracted `opmodel.dev/core@v0` module. See [`MIGRATIONS.md`](../MIGRATIONS.md) for the warm-cache deployment pattern.

## Construct a kernel

```go
import "github.com/open-platform-model/library/opm/kernel"

k := kernel.New()
// or:
k := kernel.New(kernel.WithLogger(myLogger))
```

`kernel.New` accepts functional options for logger, tracer, clock, and schema loader. None are required; defaults are no-op implementations.

The Kernel owns a single `*schema.Cache` for its lifetime. The first method that needs the schema (validation, release synthesis, compile) triggers one `OCILoader.Load` call; subsequent operations on the same Kernel reuse the cached value. Long-running consumers (operators, servers) MUST keep the Kernel alive across operations to preserve memoization.

### Pin a specific schema version

`WithSchemaLoader` configures the underlying `schema.Loader`. The default is `schema.OCILoader{}`, which resolves the floating major `opmodel.dev/core@v0`. To pin a reproducible version:

```go
import "github.com/open-platform-model/library/opm/schema"

k := kernel.New(kernel.WithSchemaLoader(schema.OCILoader{
    Module: "opmodel.dev/core@v0.3.0",
}))

// After any schema-touching call:
log.Printf("resolved schema: %s", k.SchemaCache().ResolvedVersion())
// → "v0.3.0"
```

## Load a module package

`LoadModulePackage` reads a CUE package directory, builds a `cue.Value`, and returns the detected `apiVersion`. `NewModuleFromValue` wraps it into a typed `*module.Module`.

```go
import loaderfile "github.com/open-platform-model/library/opm/helper/loader/file"

moduleVal, _, err := k.LoadModulePackage(ctx, "./module/", loaderfile.LoadOptions{})
if err != nil {
    return err
}
mod, err := k.NewModuleFromValue(moduleVal)
if err != nil {
    return err
}
```

## Validate user values (layered)

Layered validation unifies every values source in stack order, then validates the merged value against the module's `#config` schema. Per-source attribution flows through `cue.Filename(Origin)` baked at load time, so error positions report the originating file (or a stable identifier for non-file sources).

```go
defaults, _ := k.LoadSourceFromString("embedded", "defaults", `replicas: 1`)
user, _    := k.LoadSourceFromFile("./values.cue")
prod, _    := k.LoadSourceFromFile("./prod.cue")

userValues, vErr := k.ValidateModuleValuesDetailed(mod, []kernel.Source{
    defaults, user, prod,
})
if vErr != nil {
    // CUE-native error tree — walk via cueerrors.Errors / Positions, or
    // print with cueerrors.Print. The kernel ships no formatter; the
    // frontend owns presentation.
    cueerrors.Print(os.Stderr, vErr, nil)
    return vErr
}
```

See [`docs/design/kernel-validate-flow.md`](design/kernel-validate-flow.md) for the full validation primitives surface (`ValidateConfig`, `ValidateConfigPartial`, `ValidateConfigDetailed`).

## Load and process a release

Releases load as CUE packages (unified with module loading in commit `7c435f2`). The release's `Package` embeds the source `#module` reference; `ProcessModuleRelease` uses it to validate user values against `#module.#config` without a separate schema argument (Tier-2 safety net).

```go
releaseVal, _, err := k.LoadReleasePackage(ctx, "./release/", loaderfile.LoadOptions{})
if err != nil {
    return err
}
rel, err := k.ProcessModuleRelease(ctx, releaseVal, *mod, userValues)
if err != nil {
    return err
}
```

If your frontend has typed inputs in hand rather than a release package on disk, use `Kernel.SynthesizeRelease` (from `opm/helper/synth`) instead. It unifies the typed inputs against the `#ModuleRelease` schema (resolved via the kernel's `*schema.Cache`) and chains into `ProcessModuleRelease` in one call. The kernel-owned cache is plumbed through `synth.ReleaseInput.SchemaCache` automatically when omitted; pass `k.SchemaCache()` explicitly if you want to share a cache across release synthesis and other schema-touching code.

## Load and compose a Platform

The Platform is the kernel's matching and execution input. A *shell* is a `Platform` whose `#registry` is empty (or partial); `ComposePlatform` `FillPath`-injects each registered Module so the schema's computed views (`#composedTransformers`, `#matchers`, `#knownResources`, `#knownTraits`) resolve.

Frontends that load a fully-authored `platform.cue` (with its registry already populated) can skip `ComposePlatform` and use `NewPlatformFromValue` directly.

```go
shellVal, _, err := k.LoadPlatformFile(ctx, "./platform.cue", loaderfile.LoadOptions{})
if err != nil {
    return err
}
shell, err := k.NewPlatformFromValue(shellVal)
if err != nil {
    return err
}
plat, err := k.ComposePlatform(shell, []*module.Module{mod /* + others */})
if err != nil {
    return err
}
```

## Compile

`Kernel.Compile` runs the match → finalize → execute → emit pipeline and returns rendered values with full provenance.

```go
result, err := k.Compile(ctx, kernel.CompileInput{
    Module:        mod,
    ModuleRelease: rel,
    Values:        userValues,
    Platform:      plat,
    RuntimeName:   "opm-cli",
})
if err != nil {
    return err
}
for _, r := range result.Compiled {
    // r.Value is concrete, fully evaluated CUE — encode to YAML/JSON
}
```

Each `*core.Compiled` carries Release / Component / Transformer FQN provenance. Adapters in downstream implementations wrap each `Compiled` with a platform-specific `core.Resource` that fills `core.Identity`.

## Phase-explicit entry points

The kernel exposes four phase methods that map onto frontend subcommands:

| Method            | Frontend subcommand | Purpose                                              |
| ----------------- | ------------------- | ---------------------------------------------------- |
| `Kernel.Validate` | `vet`               | Assert values against `#config`                      |
| `Kernel.Match`    | `match`             | Pair components with transformers                    |
| `Kernel.Plan`     | `plan` / `preview`  | Match + execute without final emission               |
| `Kernel.Compile`  | `apply` / `render`  | Full pipeline — rendered `[]*core.Compiled`          |

## Removed entry points

The previous free-function entry points have all been removed. If you have old code calling any of these, migrate to the `*Kernel` methods listed in the table above:

| Removed                         | Replacement                                  |
| ------------------------------- | -------------------------------------------- |
| `compile.CompileModuleRelease`  | `(*Kernel).Compile`                          |
| `compile.ProcessModuleRelease`  | `(*Kernel).ProcessModuleRelease`             |
| `module.ParseModuleRelease`     | `(*Kernel).ProcessModuleRelease`             |
| `loaderfile.LoadReleaseFile`    | `loaderfile.LoadReleasePackage` (now a pkg)  |
| `opm/loader/` shim              | `opm/helper/loader/file`                     |

## Further reading

- [`README.md`](../README.md) — kernel scope, layout, multi-version support.
- [`CONSTITUTION.md`](../CONSTITUTION.md) — design principles.
- [`docs/design/`](design/) — flow diagrams and pipeline notes.
