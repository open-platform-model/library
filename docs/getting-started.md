# Getting started

This guide walks through embedding the OPM kernel in a Go program: loading a Module, validating user values against its `#config` schema, acquiring an instance and a platform module, and rendering the instance down to `*core.Compiled` values.

The recommended entry point is the `kernel.Kernel` struct, which owns the `*cue.Context` and schema cache used by every operation. **Construct one Kernel per goroutine** — the underlying `*cue.Context` is not safe for concurrent use. `Render` does not use that context: each render is its own CUE build in a fresh context that is dropped when the call returns, so concurrency is across renders, one Kernel per goroutine, with nothing shared (ADR-005).

## Prerequisites

- Go 1.22+
- A CUE module containing a `Module` artifact, a `ModuleInstance` package (or the typed inputs to synthesize one), and a Platform module: a CUE module whose `#registry` entries import their catalogs.
- `CUE_REGISTRY` configured. The kernel resolves the OPM core schema (`opmodel.dev/core@v2`) at runtime through CUE's module system, and your own modules and catalogs go through the same mechanism. The library does NOT auto-set `CUE_REGISTRY`; configure it explicitly before constructing the Kernel.

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

Operators in air-gapped environments set `CUE_REGISTRY` to an internal mirror, or pre-seed `$CUE_CACHE_DIR` with the extracted `opmodel.dev/core@v2` module and the catalogs the platform imports (run any schema-touching command and one render once with registry access, then ship the populated cache directory).

## Construct a kernel

```go
import "github.com/open-platform-model/library/opm/kernel"

k := kernel.New()
```

`kernel.New` accepts functional options (`WithSchemaLoader`, `WithRegistry`). None are required. `WithRegistry` sets the registry mapping the render build uses for the platform's catalog imports and that `AcquireModuleFromRegistry` uses for module pulls; without it the kernel inherits the process `CUE_REGISTRY`. The mapping is plumbed into the load configuration only, never written back to the environment.

The Kernel owns a single `*schema.Cache` for its lifetime. The first method that needs the schema (validation, instance synthesis, instance processing) triggers one `OCILoader.Load` call; subsequent operations on the same Kernel reuse the cached value. Long-running consumers (operators, servers) MUST keep the Kernel alive across operations to preserve memoization. `Render` does not read the schema cache: the render build resolves core through the staged module's own `cue.mod`.

### Pin a specific schema version

`WithSchemaLoader` configures the underlying `schema.Loader`. The default is `schema.OCILoader{}`, which resolves `schema.DefaultSchemaModule` (the pinned core release the kernel was built against). To pin a different reproducible version:

```go
import "github.com/open-platform-model/library/opm/schema"

k := kernel.New(kernel.WithSchemaLoader(schema.OCILoader{
    Module: "opmodel.dev/core@v2.0.0-alpha.7",
}))

// After any schema-touching call:
log.Printf("resolved schema: %s", k.SchemaCache().ResolvedVersion())
// → "v2.0.0-alpha.7"
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
defaults, _ := k.LoadSourceFromBytes("embedded", []byte(`replicas: 1`))
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

`ValidateConfigDetailed` is the whole validation surface: a single value is a one-element `[]kernel.Source`, and there is no partial-mode entry. Compose it with the `ConfigSchema()` accessors on `*module.Module` / `*module.Instance` — see the `opm/kernel` package documentation.

## Acquire an instance

`Render` imports the instance as a CUE package, so the instance must carry a `Source` (where its package lives). Two entry points produce one; both validate values where they are applied, inside the build, and assert concreteness on the built spec, so the result is concrete and schema-checked.

**From typed inputs** (a frontend that has the module and the values in hand): `Kernel.SynthesizeInstance` stages a virtual package that imports the module, writes the caller's values, and validates it in one build.

```go
import "github.com/open-platform-model/library/opm/helper/synth"

inst, err := k.SynthesizeInstance(ctx, synth.InstanceInput{
    Module:    mod,
    Name:      "web-app-demo",
    Namespace: "default",
    Values:    userValues,
})
if err != nil {
    return err
}
```

The kernel-owned schema cache is plumbed through `synth.InstanceInput.SchemaCache` automatically when omitted; pass `k.SchemaCache()` explicitly if you want to share a cache across instance synthesis and other schema-touching code.

**From an authored package on disk** (a `ModuleInstance` package that is already fully concrete): `Kernel.AcquireInstanceFromDir` loads it through the shape gate, processes it as authored (extra values are opt-in, below), and stamps its `Source`.

```go
inst, err := k.AcquireInstanceFromDir(ctx, "./instance/", loaderfile.LoadOptions{})
if err != nil {
    return err
}
```

**From an authored package plus extra values** (a frontend layering `-f` files onto an instance package): pass `kernel.WithValues` with the same `Source` values `ValidateConfigDetailed` takes. The kernel reads the package's on-disk files into an in-memory overlay, renders the unified sources as a package file declaring `values` beside them, and builds the package once through the instance shape gate, so the merge is the schema's own values unification. The caller's directory is never written to; the returned `Source` is overlay mode and renders like any other.

```go
prod, _ := k.LoadSourceFromFile("./prod.cue")

inst, err := k.AcquireInstanceFromDir(ctx, "./instance/", loaderfile.LoadOptions{},
    kernel.WithValues(prod))
if err != nil {
    // A source conflicting with the package's own values or the module's
    // #config fails here, attributed to the source's Origin.
    return err
}
```

`LoadInstancePackage` remains available for draft flows that want the raw value. Only the two acquirers above produce a `*module.Instance`, and every instance they return carries a `Source`.

## Acquire a platform module

A platform is a CUE module on disk that imports its catalogs: every `#registry` entry embeds a catalog by import, and core derives the entry's version and the platform's `#composedTransformers` from it. The kernel acquires such a module with `Kernel.AcquirePlatformFromDir`, which loads the package through the platform shape gate and stamps its `Source`.

```go
plat, err := k.AcquirePlatformFromDir(ctx, "./platform/", loaderfile.LoadOptions{})
if err != nil {
    return err
}
```

The shape gate refuses a `#registry` entry that embeds no catalog (the pre-0019 subscription shape with a `version` scalar) as `loaderfile.ErrMissingRequiredField`. There is no materialize step and no platform synthesis: the platform's catalogs are resolved by the render build, through `CUE_REGISTRY` or `WithRegistry`, exactly as any other CUE import.

**From catalog coordinates** (a Platform CR, a seeded local default): `opm/helper/platformmodule` generates the module. `Roots` turns the subscriptions into dependency roots (core pinned at the kernel's verified release, `schema.DefaultSchemaVersion()`; a frontend that needs another core build assembles the `[]Dep` roots itself), `Closure` derives the full dependency list from the published module files through a registry you configure explicitly (the once-at-generation tidy, 0019 D13), `Generate` renders `cue.mod/module.cue` and `platform.cue` deterministically, and `Files.WriteTo` writes them into a directory you own. The frontend keeps the directory lifecycle (generations, caching); the kernel acquires the result as above.

```go
import "github.com/open-platform-model/library/opm/helper/platformmodule"

entries := []platformmodule.Entry{
    {Path: "opmodel.dev/catalogs/opm@v4", Version: "4.0.1", Enable: true},
}
src, err := platformmodule.NewRegistry(platformmodule.RegistryConfig{
    Registry:   registry,      // CUE_REGISTRY syntax
    ClientType: "opm-cli",     // reported to registries
    Env:        os.Environ(),  // where CUE_CACHE_DIR is read from
})
if err != nil {
    return err
}
deps, err := platformmodule.Closure(ctx, src, platformmodule.Roots(entries))
if err != nil {
    return err
}
files, err := platformmodule.Generate(platformmodule.Input{
    Name:       "local",
    Type:       "kubernetes",
    ModulePath: "opmodel.dev/platforms/local@v0", // reserved, never published
    Entries:    entries,
    Deps:       deps,
})
if err != nil {
    return err
}
if err := files.WriteTo(platformDir); err != nil {
    return err
}
plat, err := k.AcquirePlatformFromDir(ctx, platformDir, loaderfile.LoadOptions{Registry: registry})
```

Each `#registry` entry stamps the subscription's version as its expected `version`; core unifies it with the imported catalog's own readout, so a catalog build that does not match fails the acquire naming the entry.

## Render

`Kernel.Render` renders the instance against the platform as one CUE build: it stages a generated render module that imports both, builds it once in a fresh `cue.Context`, decodes the matching verdicts and the rendered output, and drops the context. Matching and transformer execution run inside the build as CUE; the instance is rendered as processed, and `Render` performs no validation pass of its own.

```go
import (
    "errors"

    oerrors "github.com/open-platform-model/library/opm/errors"
)

result, err := k.Render(ctx, kernel.RenderInput{
    Instance:    inst,
    Platform:    plat,
    RuntimeName: "opm-cli",
    // Skew: kernel.SkewWarn (default) renders against the platform's build and
    // reports module-newer-than-platform catalog skew on result.Warnings;
    // kernel.SkewRefuse fails before evaluation with *oerrors.SkewError.
})
if err != nil {
    var rerr *kernel.RenderError
    if errors.As(err, &rerr) {
        // The build ran and the fail-closed gate refused: rerr.Diagnostics
        // carries every verdict (Pairs, Unmatched, Unresolved, Unify,
        // UnhandledTraits, OverSubscribed, ResolvedVersions) and rerr.Err the
        // typed cause (*oerrors.UnresolvedDemandsError,
        // *oerrors.UnmatchedComponentsError, oerrors.OverSubscribedContractError,
        // *oerrors.TransformError), reachable through errors.As.
        var unmatched *oerrors.UnmatchedComponentsError
        if errors.As(rerr.Err, &unmatched) {
            // unmatched.Components, unmatched.Matches
        }
    }
    return err
}
for _, w := range result.Warnings {
    // advisory: unhandled optional traits, version skew under SkewWarn
    _ = w
}
for _, r := range result.Compiled {
    // r.Value is concrete, fully evaluated CUE — encode to YAML/JSON
}
```

Each `*core.Compiled` carries Instance / Component / Transformer FQN provenance. Platform identity for compiled output is the frontend's concern — each consumer wraps `Compiled` in its own platform-specific resource type.

### Dry run

There is no separate match verb. A dry run is `Render` with `Compiled` discarded: the build evaluates every matched pair regardless, and `result.Diagnostics` (or `rerr.Diagnostics` on a refusal) carries the pairing diagnosis.

```go
result, err := k.Render(ctx, kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "opm-cli"})
var rerr *kernel.RenderError
switch {
case err == nil:
    report(result.Diagnostics.Pairs, result.Diagnostics.Unmatched)
case errors.As(err, &rerr):
    report(rerr.Diagnostics.Pairs, rerr.Diagnostics.Unmatched)
default:
    return err // refused before evaluation: missing Source, uncovered path, skew under SkewRefuse
}
```

## Entry points

| Method                                | Frontend subcommand          | Purpose                                                                      |
| ------------------------------------- | ---------------------------- | ---------------------------------------------------------------------------- |
| `Kernel.AcquirePlatformFromDir`       | (platform load)              | Platform module from disk (hand-written or `platformmodule`-generated), `Source` stamped |
| `Kernel.AcquireInstanceFromDir`       | (instance load)              | Authored instance package, validated, `Source` stamped; `WithValues` layers extra sources |
| `Kernel.SynthesizeInstance`           | (typed inputs)               | Instance from module + values, validated, `Source` stamped                   |
| `Kernel.Render`                       | `render` / `apply` / dry run | One CUE build — rendered `[]*core.Compiled` plus `RenderDiagnostics`         |

Values are validated where they are applied, inside the build each instance acquirer runs, and `Render` renders the instance as processed.

## Removed entry points

The previous entry points have all been removed. If you have old code calling any of these, migrate to the `*Kernel` methods listed in the table above:

| Removed                                          | Replacement                                                                 |
| ------------------------------------------------ | --------------------------------------------------------------------------- |
| `(*Kernel).Compile`, `kernel.CompileInput`       | `(*Kernel).Render`, `kernel.RenderInput`                                    |
| `(*Kernel).Match`, `kernel.MatchInput`           | `(*Kernel).Render`; read `RenderDiagnostics.Pairs` / `.Unmatched`           |
| `(*Kernel).Materialize`, `opm/materialize`       | `(*Kernel).AcquirePlatformFromDir`; the render build resolves the catalogs  |
| `(*Kernel).SynthesizePlatform`, `synth.Platform` | Write the platform as a CUE module importing its catalogs; acquire it       |
| `compile.UnmatchedComponentsError`               | `oerrors.UnmatchedComponentsError` (same shape)                             |
| `compile.CompileModuleInstance`                  | `(*Kernel).Render`                                                          |
| `compile.ProcessModuleInstance`                  | `(*Kernel).AcquireInstanceFromDir` / `(*Kernel).SynthesizeInstance`         |
| `module.ParseModuleInstance`                     | `(*Kernel).AcquireInstanceFromDir` / `(*Kernel).SynthesizeInstance`         |
| `(*Kernel).ProcessModuleInstance`                | `(*Kernel).AcquireInstanceFromDir` (with `WithValues`) / `(*Kernel).SynthesizeInstance` |
| `(*Kernel).ValidateConfig`, `ValidateConfigPartial`, `kernel.Partial` | `(*Kernel).ValidateConfigDetailed` with a one-element `[]kernel.Source`; no partial-mode entry |
| `(*Kernel).LoadSourceFromString`                 | `(*Kernel).LoadSourceFromBytes(origin, []byte(s))`                          |
| `(*Kernel).NewInstanceFromValue`, `module.NewInstanceFromValue` | `(*Kernel).AcquireInstanceFromDir` / `(*Kernel).SynthesizeInstance` |
| `module.CueContextOwner`, `platform.CueContextOwner` | `module.NewModuleFromValue(v)` / `platform.NewPlatformFromValue(v)` take the value only |
| `loaderfile.LoadInstanceFile`                    | `loaderfile.LoadInstancePackage` (now a pkg)                                |
| `opm/loader/` shim                               | `opm/helper/loader/file`                                                    |

## Further reading

- [`README.md`](../README.md) — kernel scope, layout, the render pipeline.
- [`CONSTITUTION.md`](../CONSTITUTION.md) — design principles.
- [`adr/005-shares-nothing-renders.md`](../adr/005-shares-nothing-renders.md) — the render concurrency contract and pool sizing.
- [`docs/design/`](design/) — CUE evaluator notes.
