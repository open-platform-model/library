# OPM kernel

The reference implementation of the Open Platform Model runtime, packaged as a Go library. Every OPM front-end — the `opm` CLI, the `opm-operator` controller, the planned Crossplane composition function, and any future runtime — embeds this kernel and inherits its behaviour.

The kernel owns:

- Loading and acquiring OPM artifacts (modules, module instances, platforms) from CUE module directories and OCI registries.
- Resolving CUE module references through the native CUE module system (OCI registries, `cue.mod`).
- Validating user-supplied values against `#config` schemas with grouped, position-aware diagnostics.
- Rendering: one CUE build per render that imports the instance, the platform and its catalogs, runs matching and transformer execution as CUE inside that build, reports the verdicts as data, and emits platform-neutral rendered values with full provenance.

The kernel does **not** own:

- Process model, command flags, exit codes, stdout/stderr formatting (lives in CLI / controller).
- Logging output (the kernel logs nothing; any logging lives with the caller).
- Cluster reconciliation, status reporting, GitOps wiring (lives in `opm-operator`).
- Platform-native identity — frontends wrap rendered values into their own platform-specific resource types.
- Platform directory lifecycle. A platform is a CUE module on disk that imports its catalogs; the frontend writes it by hand or generates it from coordinates with the opt-in `opm/helper/platformmodule` helper, owns where it lives (generations, caching), and the kernel acquires and renders against it.
- Debug-overlay policy. `#ModuleDebug` is **not** a kernel artifact; the kernel accepts only `Module`, `ModuleInstance`, and `Platform` (see "Artifact types" below). Debug values live as a `debugValues` field on `Module` itself; whether the frontend layers them into the values stack is policy that lives in the helper layer (CLI / operator / XR fn).

## Artifact types

The kernel accepts exactly three artifact types — every input ultimately resolves to one of them:

| Artifact         | Schema definition          | Go type              | Role                                                                                                                       |
| ---------------- | -------------------------- | -------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| `Module`         | `#Module` (v1alpha2)       | `*module.Module`     | Author-defined application blueprint (components, `#config` schema, `debugValues` field).                                  |
| `ModuleInstance` | `#ModuleInstance`          | `*module.Instance`   | Per-deployment instantiation of a `Module` with concrete user values.                                                      |
| `Platform`       | `#Platform`                | `*platform.Platform` | A CUE module importing its catalogs; core derives `#composedTransformers`, which the render glue reads inside the build. |

`#ModuleDebug` was previously contemplated as a fourth top-level artifact and has been **retired**; `debugValues` is now a field on `Module`. The migration is one line: read `mod.Package.LookupPath(schema.DebugValues)` and feed the result into the helper-side values stack at the layer your frontend prefers. The kernel itself never observes the distinction.

See `CONSTITUTION.md` for the full set of principles.

## Layout

```text
opm/
  core/                   Platform-neutral primitives — Compiled (terminal output)
  errors/                 Structured errors, grouped CUE diagnostics, typed render-gate causes
  schema/                 OPM core schema loader (OCILoader, Cache), CUE path inventory, metadata decoders
  kernel/                 Public Kernel struct — single entry point for the OPM runtime (acquire, synthesize, validate, Render)
  module/                 Module / Instance model and value-validation accessors
  platform/               Platform artifact model — a CUE module importing its catalogs; Render's sole platform input
  compat/                 Publish-side catalog compatibility (comparison walk, level ladder, predecessor selection)
  helper/                 Opt-in frontend convenience layer (a frontend MAY skip these)
    loader/file/          Filesystem loading (modules, instances, platforms) as CUE packages
    loader/registry/      Load a published module from an OCI registry by path@version
    loader/internal/shape Shared artifact shape gate (single-sourced across loaders)
    synth/                Instance synthesis from typed inputs (no file / no bytes)
    platformmodule/       Platform CUE module generation from catalog coordinates (files + dependency closure)
  internal/renderstage/   Single-build render staging: promoted cue.mod, skew, embedded render glue, one cue/load build
  internal/               Test-only cross-package internals (schematest, registrytest) and the CUE closedness canary (cueregression)
adr/                      Architecture decision records
enhancements/             Frozen historical proposals (cite as legacy:NNN; new work lives in the workspace enhancements/)
openspec/                 OpenSpec proposals, specs, archives
modules/                  Test-only OPM modules used by integration tests
testdata/                 CUE module fixtures consumed by package tests
Taskfile.yml              fmt / vet / lint / test entry points
```

The OPM core schema is no longer vendored or embedded — it is fetched at runtime from `CUE_REGISTRY` via `opm/schema` (the `apis/` tree and the old `opm/api` / `opm/apiversion` packages were removed). The `opm/loader/` deprecation shim is also gone; the canonical import path is `opm/helper/loader/file` (or `opm/helper/loader/registry` for published modules). A standalone `opm/validate/` package was contemplated but never landed — validation primitives live on `*kernel.Kernel` (`ValidateConfig`, `ValidateConfigPartial`, `ValidateConfigDetailed`), composed with the `ConfigSchema()` accessors on `*module.Module` / `*module.Instance`.

## Render

```text
Kernel.AcquireInstanceFromDir | Kernel.SynthesizeInstance  ->  *module.Instance   (validated, carries Source)
Kernel.AcquirePlatformFromDir                             ->  *platform.Platform (carries Source)
Kernel.Render(RenderInput{Instance, Platform, RuntimeName, Skew})
        stage one generated render module (cue.mod promoted from both inputs; each input imported by directory replacement)
        verify every OPM-namespace path either input requires is covered; apply the skew policy (SkewWarn | SkewRefuse)
        build once in a fresh cue.Context, dropped on return
        decode `diagnostics` -> RenderDiagnostics (pairs, unmatched, unresolved, unify, unhandled traits, over-subscribed, resolved versions)
        fail-closed gate     -> *RenderError carrying the diagnostics and typed causes (errors.As)
        decode `rendered`    -> []*core.Compiled with instance / component / transformer provenance
```

`Render` is the kernel's single render verb. Matching and transformer execution are CUE inside the build (the glue in `opm/internal/renderstage/render.cue.tmpl`), not Go; the build reports its verdicts as data and the kernel decodes them. A dry run is `Render` with `Compiled` discarded: the build evaluates every pair regardless, and `RenderDiagnostics` carries the pairing diagnosis. Values are validated where they are applied: `Kernel.ProcessModuleInstance` is the validated entry point (`AcquireInstanceFromDir` and `SynthesizeInstance` both go through it), and `Render` performs no validation pass of its own.

Each render is its own CUE build in its own `cue.Context` that does not outlive the call (ADR-005). Nothing built is shared between renders, concurrency is across renders with one Kernel per goroutine, and a render pool is sized by memory rather than by core count; see the `opm/kernel` package documentation.

`*core.Compiled` is the kernel's terminal output. Platform identity for compiled output is the frontend's concern — each consumer wraps `Compiled` in its own platform-specific resource type.

## Quick start

See [`docs/getting-started.md`](docs/getting-started.md) for an end-to-end walkthrough — constructing a `Kernel`, loading a Module, layered values validation, acquiring an instance and a platform module, and rendering the instance into `*core.Compiled` values.

## API stability

The library follows SemVer 2.0.0. The public surface is everything under `opm/`. Two distinct compatibility tracks coexist and must not be confused:

- **Go module SemVer** governs the Go types and function signatures consumed by downstream binaries. A breaking change here is a major bump of the library.
- **OPM schema versioning** governs the CUE shapes consumed at runtime — `#Module`, `#ModuleInstance`, `#Platform`, `#Component`, transformer contracts. The kernel MUST be able to load and render older schema versions seamlessly so that downstream implementations inherit multi-version support without per-implementation effort.

The two tracks are independent: within an OPM schema major, additive shape changes are absorbed by floating-major resolution and require no Go-side bump; a shape break in the schema is itself a coordinated library-breaking event.

## OPM schema resolution

The library does NOT vendor or embed the OPM core schema. At runtime the kernel resolves `opmodel.dev/core@v2` through CUE's module system against `CUE_REGISTRY`, then memoizes the built `cue.Value` in a per-`Kernel` `*schema.Cache`.

Key pieces:

- `opm/schema` — schema loader (`Loader` interface, `OCILoader` sole public implementation), per-instance memoization (`Cache`), CUE path inventory, metadata decoders, and the `PublicRegistry` const (`opmodel.dev=ghcr.io/open-platform-model,registry.cue.works`).
- `opm/kernel` — `kernel.WithSchemaLoader(schema.Loader)` configures which Loader the Kernel's cache wraps; `(*Kernel).SchemaCache()` exposes the cache to instance synthesis and other callers. `kernel.WithRegistry(string)` sets the registry mapping the render build uses for the platform's catalog imports and for registry module acquisition.

Frontends (CLI, operator, future Crossplane fn) set `CUE_REGISTRY` (typically to `schema.PublicRegistry`) before constructing the Kernel. The library auto-applies no default; this keeps Principle I (kernel neutrality) intact and avoids hidden lookups. See `docs/getting-started.md` for the deployment pattern, including the warm-cache pre-seeding pattern for restricted environments.

## Helper boundary (`opm/helper/`)

Anything under `opm/helper/` is opt-in convenience for embedding the kernel; a frontend MAY skip it and call the kernel directly. Anything outside `opm/helper/` is part of the kernel contract.

Today this layer holds:

- `opm/helper/loader/file` — filesystem-coupled loaders: `LoadModulePackage`, `LoadInstancePackage`, `LoadPlatformPackage`. All three load a CUE package directory through the shared shape gate; the platform gate refuses a `#registry` entry that embeds no catalog.
- `opm/helper/loader/registry` — `LoadModulePackageWithSource`: a published `#Module` by `path@version`, staged so a follow-on build can import it. Surfaced as `Kernel.AcquireModuleFromRegistry`.
- `opm/helper/synth` — Instance synthesis (`Instance`): build a `ModuleInstance` CUE value from typed inputs (name, namespace, module reference, values, labels, annotations) without round-tripping through a file. Pairs with `Kernel.SynthesizeInstance`, which chains synth + validate in one call. There is no platform synthesis: a platform is a CUE module on disk.
- `opm/helper/platformmodule` — Platform module generation from catalog coordinates: `Roots` + `Closure` derive the tidied dependency list from published module files (through a caller-configured `ModFileSource`), `Generate` renders `cue.mod/module.cue` and `platform.cue` deterministically, `Files.WriteTo` writes them into a caller-owned directory for `Kernel.AcquirePlatformFromDir`. The core pin defaults to `schema.DefaultSchemaVersion()`.

Layered values validation lives on the kernel itself — see `Kernel.ValidateConfigDetailed` and the `Source` type in `opm/kernel`. See `enhancements/001-kernel-redesign-around-platform/02-design.md`.

The previous `opm/loader/` deprecation shim has been removed (commit `3a9a9bd`); the canonical import path is `opm/helper/loader/file`.

## Quality gates

```text
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
- `opmodel.dev/core@v2` — current OPM schema, published as an OCI CUE module (sources live in the workspace `core/` repo).
- `docs/getting-started.md` — end-to-end embedding walkthrough.
- `docs/design/` — CUE evaluator notes: the v0.17.x closedness regression and its canary, plus historical bug records whose code no longer exists.
- `enhancements/` — frozen historical proposals; the single-build render design is workspace enhancement 0019.
- `adr/` — architecture decision records (ADR-006: one CUE build per artifact; ADR-005: shares-nothing renders).
- `CHANGELOG.md` — released-version history (generated by release-please).
- `migrations/README.md` — migration-documentation policy: per-change fragments, dormant until GA (pre-GA breaking changes are recorded in `CHANGELOG.md` and the OpenSpec archive).
