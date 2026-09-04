# Design: library-platform-module-generator

## Context

See `proposal.md` § Why. The operator's `internal/platformmodule` (PR 118, 0019 D6) has three seams: `Generate` (pure), `Closure` (module-file walk through `modconfig.Registry.ModFile`) and `Layout` (per-generation directories, staging swap, retention, boot reset). Its design § "The dependency list is the full MVS closure" measured that CUE `v0.17.1` resolves a dependency's unqualified import through the module graph, but the *render* module promotes the platform's list verbatim (`renderstage.Promote`), so a path absent from the platform's `cue.mod` (measured: `cue.dev/x/k8s.io@v0`) falls to maximum-version selection over the instance's pins. Hence the closure must be complete at generation. The same design deferred lifting the generator to the library until the CLI needed it.

`Kernel.AcquireInstanceFromDir` (`artifact-types`) returns an on-disk-mode `Source` and supplies no values. `synth.Instance` already builds an overlay (`buildOverlay`: cloned module overlay plus `instance.cue` and a `values.cue` rendered with `format.Node`) and evaluates it with `loaderfile.BuildInstanceOverlayAt`. `Kernel.ValidateConfigDetailed` already takes `[]kernel.Source` with per-source filenames.

Constraints: kernel neutrality (no hidden env, no package singletons), helpers are opt-in under `opm/helper/<name>/`, Principle VIII (this change is two additive surfaces and touches no existing signature).

## Goals / Non-Goals

**Goals**

- One generator and one closure derivation for every frontend; byte-identical output to the operator's for the same input, so the operator's re-point is a pure re-import.
- The core pin for generated platforms is owned by the library's verified release.
- An on-disk instance package can carry caller-supplied values into `Render` without the frontend re-implementing overlay construction.

**Non-Goals**

- Directory lifecycle (`Layout`): stays in the operator; the CLI has its own cache policy (`cli-render-switch`).
- Generating a module from a *value* (a `#Platform` already evaluated): the input is coordinates, never a platform.
- Changing `synth.Instance`, `LoadInstancePackage` or `Render`.
- A tidy of an existing hand-written platform module (`cue mod tidy` exists).

## Decisions

### Lift `Generate`, `Roots`, `Closure` and the types; leave `Layout`

**Context**: the operator package mixes a pure generator with process policy.
**Options**: (1) lift everything including `Layout`; (2) lift the generator and closure, add `Files.WriteTo(dir)`, leave `Layout`; (3) lift only `Generate` and let each frontend derive its own closure.
**Decision**: option 2. `Layout` encodes Kubernetes generations, staging swaps under a render gate and boot-time reset on an ephemeral volume; none of it applies to a CLI. The closure is the part the CLI must not fork (it is the D13 tidy). `WriteTo` is the smallest write both need (create parents, refuse path escape), lifted from the operator's `writeFiles`.

### The module path is caller input; the core pin defaults from `schema.DefaultSchemaModule`

**Context**: the operator hard-codes `ModulePath = opmodel.dev/platforms/cluster@v0` and `CoreVersion` (bumped by workspace tooling). The CLI's local module is `opmodel.dev/platforms/local@v0`.
**Options**: (1) keep both as helper constants; (2) module path in `Input`, core version defaulting from the library's default schema identifier with an explicit override; (3) both as required input.
**Decision**: option 2. The module path is identity the frontend owns (both under the reserved-unpublished `platforms/` namespace, 0019 D6). The core pin is not: `DefaultSchemaModule` is defined as the release the render glue, fixtures and parity oracle were verified against, which is exactly the release a generated platform must embed. `Roots(entries, opts...)` takes an optional `WithCoreVersion`; tests use it to pin the in-process fixture core. The version is split from the identifier with `ast.SplitPackageVersion`'s counterpart on the module path (the loader already parses it in `isBareMajorVersion`; a small exported accessor `schema.DefaultSchemaVersion()` avoids re-parsing in consumers).

### Registry access through explicit `modconfig.Config`

**Context**: the operator's `NewRegistry(registry)` sets `ClientType: "opm-operator"` and lets `modconfig` read `CUE_CACHE_DIR` from the process environment, a hidden lookup the library forbids.
**Decision**: `NewRegistry(RegistryConfig{Registry, ClientType, Env []string})` maps onto `modconfig.Config{CUERegistry, ClientType, Env}`. `Env` is passed through verbatim (the loader already threads `load.Config.Env` the same way for the registry override); a nil `Env` means `modconfig`'s own default, documented as the caller's explicit choice. `ModFileSource` stays the one-method interface so `closure_test` runs on a fixture graph.

### Values overlay for `AcquireInstanceFromDir` reuses synth's pieces

**Context**: layering values onto an authored instance package must be a single CUE build (ADR-006), never a Go-side fill.
**Options**: (1) a new kernel method `AcquireInstanceFromDirWithValues`; (2) a variadic `AcquireOption` on the existing method, `WithValues(sources ...Source)`; (3) the frontend copies the package to a temp dir and appends a file.
**Decision**: option 2. One method, one source of `Source` construction, every existing caller unchanged. Implementation: unify the sources exactly as `ValidateConfigDetailed` does (per-source `cue.Filename` kept for attribution), render the unified value to `values.cue` through the same `format.Node` path `synth` uses (extract the renderer into `opm/internal/valuesfile`, importable by both `opm/kernel` and `opm/helper/synth` without adding public surface; no duplication), read the module root's on-disk files into an overlay map keyed by absolute path (`cue.mod/**` and every `.cue` under the root, matching what `load` would read), add the values file under the package directory as `opm-values.cue` (an overlay entry replaces the on-disk file of the same path, so the name is reserved rather than `values.cue`, which an authored package commonly holds) with the package's own `package` clause (parsed from the package's files), and call `BuildInstanceOverlayAt(root, pkg, overlay)`. Attribution: the schema does not bind an instance's `values` to `#module.#config` at acquisition (an authored `values: replicas: "three"` builds), so after a successful build the sources are validated against `#config` with `Partial()` (positions are the sources' Origins); when the build itself fails (a conflict with the package's own values), the authored package is loaded and the package's values plus the sources are re-validated as `ValidateConfigDetailed` does, which attributes the conflict to the source instead of the rendered overlay file. The returned `Source` is `{Root, Pkg, Overlay}`; `Render` already stages overlay-mode instances (it does for synthesized ones).
**Rationale**: a package's `values` field is regular, so a second file in the same package unifying `values: {...}` is the schema's own merge. Option 3 writes into user space; option 1 doubles the surface.

### The platform.cue header names no frontend

**Context**: the operator's generator opens `platform.cue` with `// Generated by opm-operator from the cluster Platform.`; a helper shared with the CLI cannot name a frontend (kernel neutrality), and a CLI-generated file saying "opm-operator" would be wrong.
**Decision**: the header reads `// Generated by opm/helper/platformmodule from catalog coordinates. Never edit, never publish.` Everything after the header line is byte-identical to the operator's output; the operator re-point accepts the one-line header change (nothing asserts on it).

### Byte-identical proof against the operator's generator

**Context**: the operator re-point must not change what its e2e specs see.
**Decision**: the helper's tests include the operator's `TestGenerate_TwoCatalogs` golden expectations verbatim (module file in modfile canonical format, `platform.cue` with `cat0`/`cat1` aliases and the `enable`/`version`/`#catalog` block). The operator change asserts equality against the helper for its sample Platform.

## Risks / Trade-offs

- [The overlay must mirror what `cue/load` reads from disk; a missed file class (e.g. `cue.mod/pkg`, `cue.mod/usr`) changes evaluation] → the overlay reader mirrors every `.cue` file under the module root rather than re-deciding what `load` includes (an entry `load` ignores on disk it ignores in the overlay too), covered by a test that acquires a fixture with and without values and asserts the same exported package (`cue.Value.Equals` is false even across two plain acquires of the fixture, so the comparison is the formatted syntax).
- [Extra values on a package whose values are already concrete produce a conflict, not an override] → documented; that is CUE unification and what the CLI's `-f` already meant under `ProcessModuleInstance`.
- [The core pin defaulting from the library couples generated platforms to library releases] → intended (the pin must match the glue); the override exists for fixtures and forward testing.
- [A frontend keeps its own copy after the lift] → the operator change deletes it; the CLI never had one.

## Migration Plan

Additive MINOR release. Consumers adopt at their own pace: `opm-operator` via `operator-platformmodule-lift`, `cli` via `cli-render-switch` and `cli-config-platform-module`. No migration fragment (pre-GA, ADR-004). Rollback is a revert of the release.

## Open Questions

None blocking. Whether `schema.DefaultSchemaVersion()` is the right spelling for the version accessor is settled at implementation; it changes no behaviour.
