# Getting started

This guide walks through embedding the OPM kernel in a Go program: loading a Module, validating user values against its `#config` schema, materializing a Platform, and compiling an Instance down to rendered `*core.Compiled` values.

The recommended entry point is the `kernel.Kernel` struct, which owns its `*cue.Context` and threads cross-cutting dependencies (logger, tracer, clock) through every operation. **Construct one Kernel per goroutine** — the underlying `*cue.Context` is not safe for concurrent use.

## Prerequisites

- Go 1.22+
- A CUE module containing a `Module` artifact (and optionally a `ModuleInstance` artifact and a `Platform` artifact).
- `CUE_REGISTRY` configured. The kernel resolves the OPM core schema (`opmodel.dev/core@v2`) at runtime through CUE's module system, and your own modules go through the same mechanism. The library does NOT auto-set `CUE_REGISTRY`; configure it explicitly before constructing the Kernel.

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

Operators in air-gapped environments set `CUE_REGISTRY` to an internal mirror, or pre-seed `$CUE_CACHE_DIR` with the extracted `opmodel.dev/core@v2` module (run any schema-touching command once with registry access, then ship the populated cache directory).

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

`WithSchemaLoader` configures the underlying `schema.Loader`. The default is `schema.OCILoader{}`, which resolves the floating major `opmodel.dev/core@v2`. To pin a reproducible version:

```go
import "github.com/open-platform-model/library/opm/schema"

k := kernel.New(kernel.WithSchemaLoader(schema.OCILoader{
    Module: "opmodel.dev/core@v2.0.0-alpha.4",
}))

// After any schema-touching call:
log.Printf("resolved schema: %s", k.SchemaCache().ResolvedVersion())
// → "v2.0.0-alpha.4"
```

## Load a module package

`LoadModulePackage` reads a CUE package directory and builds a `cue.Value`. `NewModuleFromValue` wraps it into a typed `*module.Module`. To load a module published in an OCI registry instead of from disk, use `k.AcquireModuleFromRegistry(ctx, modPath, version)` (or `opm/helper/loader/registry.LoadModulePackageWithSource` directly) — it returns a decoded `*module.Module` carrying its staged source, ready for synthesis; read `Module.Package` when only the raw value is wanted.

```go
import loaderfile "github.com/open-platform-model/library/opm/helper/loader/file"

moduleVal, err := k.LoadModulePackage(ctx, "./module/", loaderfile.LoadOptions{})
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

userValues, vErr := k.ValidateConfigDetailed(mod.ConfigSchema(), []kernel.Source{
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

The full validation primitives surface is `ValidateConfig`, `ValidateConfigPartial`, and `ValidateConfigDetailed`, composed with the `ConfigSchema()` accessors on `*module.Module` / `*module.Instance` — see the `opm/kernel` package documentation.

## Load and process a release

Instances load as CUE packages (unified with module loading in commit `7c435f2`). The release's `Package` embeds the source `#module` reference; `ProcessModuleInstance` uses it to validate user values against `#module.#config` without a separate schema argument (Tier-2 safety net).

```go
releaseVal, err := k.LoadInstancePackage(ctx, "./release/", loaderfile.LoadOptions{})
if err != nil {
    return err
}
rel, err := k.ProcessModuleInstance(ctx, releaseVal, *mod, userValues)
if err != nil {
    return err
}
```

If your frontend has typed inputs in hand rather than a release package on disk, use `Kernel.SynthesizeInstance` (from `opm/helper/synth`) instead. It unifies the typed inputs against the `#ModuleInstance` schema (resolved via the kernel's `*schema.Cache`) and chains into `ProcessModuleInstance` in one call. The kernel-owned cache is plumbed through `synth.InstanceInput.SchemaCache` automatically when omitted; pass `k.SchemaCache()` explicitly if you want to share a cache across release synthesis and other schema-touching code.

## Load and materialize a Platform

The Platform is the kernel's matching and execution input. Its `#registry` declares path-keyed catalog subscriptions; `Kernel.Materialize` pulls each enabled subscription's catalog build from the OCI registry and returns a sealed `*materialize.MaterializedPlatform` with the composed transformers and matcher index filled. The phase methods accept only the materialized form — a raw `*platform.Platform` is not a valid phase input.

```go
platVal, err := k.LoadPlatformPackage(ctx, "./platform/", loaderfile.LoadOptions{})
if err != nil {
    return err
}
plat, err := k.NewPlatformFromValue(platVal)
if err != nil {
    return err
}
mp, err := k.Materialize(ctx, plat)
if err != nil {
    return err
}
```

`Materialize` is explicit and caller-driven: every call performs registry I/O, and the kernel holds no materialize cache. Long-running consumers store the `*MaterializedPlatform` keyed on an invalidation signal they own; short-lived ones rely on CUE's on-disk module cache.

## Compile

`Kernel.Compile` runs the match → execute → emit pipeline and returns rendered values with full provenance. The instance is rendered as processed: values were validated and filled by `ProcessModuleInstance`, and `Compile` performs no validation pass of its own.

```go
result, err := k.Compile(ctx, kernel.CompileInput{
    ModuleInstance: rel,
    Platform:       mp,
    RuntimeName:    "opm-cli",
})
if err != nil {
    return err
}
for _, r := range result.Compiled {
    // r.Value is concrete, fully evaluated CUE — encode to YAML/JSON
}
```

Each `*core.Compiled` carries Instance / Component / Transformer FQN provenance. Platform identity for compiled output is the frontend's concern — each consumer wraps `Compiled` in its own platform-specific resource type.

## Phase-explicit entry points

The kernel exposes two phase methods that map onto frontend subcommands:

| Method            | Frontend subcommand | Purpose                                              |
| ----------------- | ------------------- | ---------------------------------------------------- |
| `Kernel.Match`    | `match`             | Pair components with transformers                    |
| `Kernel.Compile`  | `apply` / `render`  | Full pipeline — rendered `[]*core.Compiled`          |

Values are validated where they are applied: `Kernel.ProcessModuleInstance` is the validated entry point, and `Compile` renders the instance as processed. A dry run is `Match` (pairing diagnosis) or `Compile` with the rendered slice discarded.

## Removed entry points

The previous free-function entry points have all been removed. If you have old code calling any of these, migrate to the `*Kernel` methods listed in the table above:

| Removed                         | Replacement                                  |
| ------------------------------- | -------------------------------------------- |
| `compile.CompileModuleInstance`  | `(*Kernel).Compile`                          |
| `compile.ProcessModuleInstance`  | `(*Kernel).ProcessModuleInstance`             |
| `module.ParseModuleInstance`     | `(*Kernel).ProcessModuleInstance`             |
| `loaderfile.LoadInstanceFile`    | `loaderfile.LoadInstancePackage` (now a pkg)  |
| `opm/loader/` shim              | `opm/helper/loader/file`                     |

## Further reading

- [`README.md`](../README.md) — kernel scope, layout, multi-version support.
- [`CONSTITUTION.md`](../CONSTITUTION.md) — design principles.
- [`docs/design/`](design/) — flow diagrams and pipeline notes.
