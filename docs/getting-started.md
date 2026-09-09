# Getting started

This guide walks through embedding the OPM kernel in a Go program: loading a Module, validating user values against its `#config` schema, acquiring an instance and a platform module, and rendering the instance down to `*kernel.Compiled` values.

The recommended entry point is the `kernel.Kernel` struct, which owns the schema cache used by every operation and no build context: every operation creates its own `cue.Context`, builds in it, and drops it when the call returns, so an artifact you hold pins the context that built it and the Kernel retains nothing (ADR-005, ADR-007). **A single Kernel is safe for concurrent use across its method calls**; construct one per process and share it between goroutines, with nothing else shared.

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

The Kernel owns a single `*schema.Cache` for its lifetime. The first `SchemaCache().Get()` triggers one `OCILoader.Load` call; subsequent calls on the same Kernel reuse the cached value. Long-running consumers (operators, servers) MUST keep the Kernel alive across operations to preserve memoization. No kernel verb loads the schema on a pinned kernel (the default): `SynthesizeInstance` reads the core import major off the pin, and acquisition and `Render` resolve core through the module's own `cue.mod` inside the build. Only a bare-major loader (`opmodel.dev/core@v2`) makes synthesis load the schema to learn the release.

### Pin a specific schema version

`WithSchemaLoader` configures the underlying `schema.Loader`. The default is `schema.OCILoader{}`, which resolves `schema.DefaultSchemaModule` (the pinned core release the kernel was built against). To pin a different reproducible version:

```go
import "github.com/open-platform-model/library/opm/schema"

k := kernel.New(kernel.WithSchemaLoader(schema.OCILoader{
    Module: "opmodel.dev/core@v2.0.0-alpha.7",
}))

// After a schema load (SchemaCache().Get(); no verb runs one on a pinned kernel):
log.Printf("resolved schema: %s", k.SchemaCache().ResolvedVersion())
// → "v2.0.0-alpha.7"
```

## Acquire a module

Two verbs return a typed, source-carrying `*module.Module`: `AcquireModuleFromDir` reads a CUE package directory, `AcquireModuleFromRegistry` fetches a published module by `path@version`. Both run the same shape gate and stamp `Module.Source` as an in-memory overlay of the module's `.cue` files, which is what makes the result a valid `SynthesizeInstance` input. Read `Module.Package` when only the raw `cue.Value` is wanted; there is no separate load-then-construct tier, and no verb takes a registry argument — `kernel.WithRegistry` is the one mapping.

```go
mod, err := k.AcquireModuleFromDir(ctx, "./module/")
if err != nil {
    return err
}

// …or, for a published module:
mod, err = k.AcquireModuleFromRegistry(ctx, "example.com/modules/hello@v0", "v0.0.2")
if err != nil {
    return err
}
```

A module acquired from a subdirectory of its CUE module is fine to read, but not to synthesize from: the synthesized instance imports the module by its module path, which resolves to the root package, so `SynthesizeInstance` refuses it.

Need the acquired module's tree on disk (scaffolding from a published template)? `mod.Source.WriteTo(dest)` writes every file under `dest` at its module-relative path and returns the paths it wrote, sorted — no second fetch, no walk of your own.

## Validate user values (layered)

Layered validation compiles every values source in the schema's own context, unifies them in stack order, then validates the merged value against the module's `#config` schema. A `kernel.Source` is CUE source bytes plus their origin, bound to no context; the loader helpers check syntax and evaluate nothing, and the kernel compiles each source with `cue.Filename(Origin)` where it is used, so error positions report the originating file (or a stable identifier for non-file sources). You need no `cue.Context` of your own.

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
inst, err := k.SynthesizeInstance(ctx, kernel.InstanceInput{
    Module:    mod,
    Name:      "web-app-demo",
    Namespace: "default",
    Values:    []kernel.Source{defaults, user, prod},
})
if err != nil {
    return err
}
```

`InstanceInput.Values` is the same `[]kernel.Source` `ValidateConfigDetailed` takes, so a frontend wraps its values once and passes the same slice everywhere. The kernel owns the schema cache; `InstanceInput` carries none.

**From an authored package on disk** (a `ModuleInstance` package that is already fully concrete): `Kernel.AcquireInstanceFromDir` loads it through the shape gate, processes it as authored (extra values are opt-in, below), and stamps its `Source`.

```go
inst, err := k.AcquireInstanceFromDir(ctx, "./instance/")
if err != nil {
    return err
}
```

**From an authored package plus extra values** (a frontend layering `-f` files onto an instance package): pass them as trailing `Source` arguments — the same values `ValidateConfigDetailed` takes. The kernel reads the package's on-disk files into an in-memory overlay, renders the unified sources as a package file declaring `values` beside them, and builds the package once through the instance shape gate, so the merge is the schema's own values unification. The caller's directory is never written to; the returned `Source` is overlay mode and renders like any other.

```go
prod, _ := k.LoadSourceFromFile("./prod.cue")

inst, err := k.AcquireInstanceFromDir(ctx, "./instance/", prod)
if err != nil {
    // A source conflicting with the package's own values or the module's
    // #config fails here, attributed to the source's Origin.
    return err
}
```

Only the two acquirers above produce a `*module.Instance`, and every instance they return carries a `Source`; the raw value is `Instance.Package`.

## Acquire a platform module

A platform is a CUE module on disk that imports its catalogs: every `#registry` entry embeds a catalog by import, and core derives the entry's version and the platform's `#composedTransformers` from it. The kernel acquires such a module with `Kernel.AcquirePlatformFromDir`, which loads the package through the platform shape gate and stamps its `Source`.

```go
plat, err := k.AcquirePlatformFromDir(ctx, "./platform/")
if err != nil {
    return err
}
```

The shape gate refuses a `#registry` entry that embeds no catalog (the pre-0019 subscription shape with a `version` scalar) as `oerrors.ErrMissingRequiredField`, the sentinel declared in `opm/errors` alongside `ErrInvalidPackage` and `ErrWrongKind`. There is no materialize step and no platform synthesis: the platform's catalogs are resolved by the render build, through `CUE_REGISTRY` or `WithRegistry`, exactly as any other CUE import.

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
plat, err := k.AcquirePlatformFromDir(ctx, platformDir)
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
    // marks the path's ResolvedVersions row Newer; kernel.SkewRefuse fails
    // before evaluation with *oerrors.SkewError.
})
if err != nil {
    var rerr *kernel.RenderError
    if errors.As(err, &rerr) {
        // The build ran and the fail-closed gate refused: rerr.Diagnostics
        // carries every verdict (Pairs, Unmatched, Unresolved, Unify,
        // UnhandledTraits, OverSubscribed, ResolvedVersions) and rerr.Err the
        // typed cause (*oerrors.UnresolvedDemandsError,
        // *oerrors.UnmatchedComponentsError, *oerrors.OverSubscribedContractsError,
        // *oerrors.TransformError), reachable through errors.As.
        var unmatched *oerrors.UnmatchedComponentsError
        if errors.As(rerr.Err, &unmatched) {
            // unmatched.Components: one row per component, each carrying its
            // CandidateVerdicts (transformer, matched, missing labels).
        }
    }
    return err
}
// Advisory facts are rows, not messages: the frontend words them.
for comp, traits := range result.Diagnostics.UnhandledTraits {
    _, _ = comp, traits // an optional trait no matched transformer handles
}
for _, r := range result.Diagnostics.ResolvedVersions {
    if r.Newer {
        // the module requires r.ModuleVersion, the platform carries r.PlatformVersion
    }
}
for _, r := range result.Compiled {
    // r.Value is concrete, fully evaluated CUE — encode to YAML/JSON
}
```

Each `*kernel.Compiled` carries Instance / Component / Transformer FQN provenance. Platform identity for compiled output is the frontend's concern — each consumer wraps `Compiled` in its own platform-specific resource type.

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
| `Kernel.AcquireModuleFromDir`         | (module load)                | Module package from disk, `Source` stamped as a byte overlay                 |
| `Kernel.AcquireModuleFromRegistry`    | (module fetch)               | Published module by `path@version`, `Source` stamped                         |
| `Kernel.AcquirePlatformFromDir`       | (platform load)              | Platform module from disk (hand-written or `platformmodule`-generated), `Source` stamped |
| `Kernel.AcquireInstanceFromDir`       | (instance load)              | Authored instance package, validated, `Source` stamped; trailing `Source` values layer on |
| `Kernel.SynthesizeInstance`           | (typed inputs)               | Instance from module + values, validated, `Source` stamped                   |
| `Kernel.Render`                       | `render` / `apply` / dry run | One CUE build — rendered `[]*kernel.Compiled` plus `RenderDiagnostics`         |

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
| `(*Kernel).ProcessModuleInstance`                | `(*Kernel).AcquireInstanceFromDir` (with trailing values) / `(*Kernel).SynthesizeInstance` |
| `(*Kernel).ValidateConfig`, `ValidateConfigPartial`, `kernel.Partial` | `(*Kernel).ValidateConfigDetailed` with a one-element `[]kernel.Source`; no partial-mode entry |
| `(*Kernel).LoadSourceFromString`                 | `(*Kernel).LoadSourceFromBytes(origin, []byte(s))`                          |
| `(*Kernel).NewInstanceFromValue`, `module.NewInstanceFromValue` | `(*Kernel).AcquireInstanceFromDir` / `(*Kernel).SynthesizeInstance` |
| `module.CueContextOwner`, `platform.CueContextOwner` | `module.NewModuleFromValue(v)` / `platform.NewPlatformFromValue(v)` take the value only |
| `(*Kernel).CueContext()`                         | `cuecontext.New()` for a value that stands alone; `v.Context()` of the value it will unify with; the schema's is `k.SchemaCache().Get()` then `.Context()` |
| `kernel.Source{Value: v, Origin: o}`             | `kernel.Source{Origin: o, Data: b}` (`LoadSourceFromBytes` / `LoadSourceFromFile`; the kernel compiles the bytes where it uses them) |
| `(*schema.Cache).Get(ctx)`                       | `(*schema.Cache).Get()`; the cache owns a private context                  |
| `loaderfile.LoadInstanceFile`                    | `Kernel.AcquireInstanceFromDir`                                             |
| `opm/loader/` shim, `opm/helper/loader/**`       | `Kernel.Acquire*` (the loader is `opm/internal/loader`)                     |
| `loaderfile.LoadOptions{Registry: r}`            | `kernel.WithRegistry(r)` at construction                                    |
| `synth.InstanceInput{…}`                         | `kernel.InstanceInput{…, Values: []kernel.Source{…}}`                       |
| `(*Kernel).LoadModulePackage` + `NewModuleFromValue` | `(*Kernel).AcquireModuleFromDir`                                        |
| `(*Kernel).LoadPlatformPackage` + `NewPlatformFromValue` | `(*Kernel).AcquirePlatformFromDir`                                  |
| `(*Kernel).LoadInstancePackage`                  | `(*Kernel).AcquireInstanceFromDir`                                          |
| `kernel.AcquireOption`, `kernel.WithValues`      | trailing `...kernel.Source` on `AcquireInstanceFromDir`                     |

## Further reading

- [`README.md`](../README.md) — kernel scope, layout, the render pipeline.
- [`CONSTITUTION.md`](../CONSTITUTION.md) — design principles.
- [`adr/005-shares-nothing-renders.md`](../adr/005-shares-nothing-renders.md) — the render concurrency contract and pool sizing.
- [`docs/design/`](design/) — CUE evaluator notes.
