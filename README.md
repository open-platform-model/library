# OPM kernel

The reference implementation of the Open Platform Model runtime, packaged as a Go library. Every OPM front-end — the `opm` CLI, the `opm-operator` controller, the planned Crossplane composition function, and any future runtime — embeds this kernel and inherits its behaviour.

The kernel owns:

- Loading and acquiring OPM artifacts (modules, module instances, platforms, catalogs) from CUE module directories and OCI registries.
- Resolving CUE module references through the native CUE module system (OCI registries, `cue.mod`).
- Validating user-supplied values against `#config` schemas with grouped, position-aware diagnostics.
- Rendering: one CUE build per render that imports the instance, the platform and its catalogs, runs matching and transformer execution as CUE inside that build, reports the verdicts as data, and emits platform-neutral rendered values with full provenance.

The kernel does **not** own:

- Process model, command flags, exit codes, stdout/stderr formatting (lives in CLI / controller).
- Logging output (the kernel logs nothing; any logging lives with the caller).
- Cluster reconciliation, status reporting, GitOps wiring (lives in `opm-operator`).
- Platform-native identity — frontends wrap rendered values into their own platform-specific resource types.
- Platform directory lifecycle. A platform is a CUE module on disk that imports its catalogs; the frontend writes it by hand or generates it from coordinates with the opt-in `opm/helper/platformmodule` helper, owns where it lives (generations, caching), and the kernel acquires and renders against it.
- Debug-overlay policy. `#ModuleDebug` is **not** a kernel artifact; the kernel accepts only `Module`, `ModuleInstance`, `Platform` and `Catalog` (see "Artifact types" below). Debug values live as a `debugValues` field on `Module` itself; whether the frontend layers them into the values stack is policy that lives in the helper layer (CLI / operator / XR fn).

## Artifact types

The kernel accepts exactly four artifact types — every input ultimately resolves to one of them:

| Artifact         | Schema definition (`opmodel.dev/core@v2`) | Go type              | Role                                                                                                                                   |
| ---------------- | ----------------------------------------- | -------------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| `Module`         | `#Module`                                 | `*module.Module`     | Author-defined application blueprint (components, `#config` schema, `debugValues` field).                                              |
| `ModuleInstance` | `#ModuleInstance`                         | `*module.Instance`   | Per-deployment instantiation of a `Module` with concrete user values.                                                                  |
| `Platform`       | `#Platform`                               | `*platform.Platform` | A CUE module importing its catalogs; core derives `#composedTransformers`, which the render glue reads inside the build.               |
| `Catalog`        | `#Catalog`                                | `*catalog.Catalog`   | The contracts a catalog defines beside the transformers implementing them. Acquired, read and derived from — never rendered (ADR-009). |

`#ModuleDebug` was previously contemplated as a fourth top-level artifact and has been **retired**; `debugValues` is now a field on `Module`. The migration is one line: read `mod.DebugValues()` and feed the result into the helper-side values stack at the layer your frontend prefers. The kernel itself never observes the distinction.

See `CONSTITUTION.md` for the full set of principles.

## Layout

```text
opm/
  errors/                 Structured errors, grouped CUE diagnostics, typed render-gate causes
  schema/                 OPM core schema loader (OCILoader, Cache), CUE path inventory, metadata types
  kernel/                 Public Kernel struct — single entry point for the OPM runtime (acquire, synthesize, validate, Render) and `Compiled`, its terminal output
  module/                 Module / Instance model and value-validation accessors
  platform/               Platform artifact model — a CUE module importing its catalogs; Render's sole platform input
  catalog/                Catalog artifact model (ADR-009) — Metadata, Package, Source, plus the on-demand derivations Provides() (reads core's derived `provides`; a deprecated Go fold answers catalogs built against an older core) and Requires(). Read and derived from; never rendered
  helper/                 Opt-in frontend convenience layer (a frontend MAY skip these; lint-enforced)
    platformmodule/       Platform CUE module generation from catalog coordinates (files + dependency closure)
    objectset/            DEPRECATED: use k8s/object. Duplicate rendered object identities, frozen and kept until both frontends migrate
  k8s/                    The Kubernetes tier beside the kernel (ADR-011): mandatory for a Kubernetes frontend, fenced by lint
    labels/               OPM label keys and values + IsOPMManagedBy; standard library only; names labels, never stamps them
    object/               Resource (a Compiled with Kubernetes accessors), Export (one CUE export per object), the kind-class Weight table, Sort, Stages, Duplicates + DuplicateIdentitiesError
  internal/loader/        The kernel's one artifact loader: shape gate, LoadDir (the one build-and-gate step, directory or overlay), FetchArtifact (a published artifact by path@version: fetch, stage as an overlay, build through LoadDir like a directory artifact, gated to the shape the caller names) with FetchModule the #Module entry over it, adding the coordinate identity check
  internal/synth/         Instance synthesis from typed inputs, built inside the module's own staged tree
  internal/renderstage/   Single-build render staging: promoted cue.mod, skew, embedded render glue, one cue/load build
  internal/               Internal runtime packages (cueenv, modversion, sourcetree, valuesfile) plus test-only helpers (schematest, registrytest) and the CUE closedness canary (cueregression)
adr/                      Architecture decision records
enhancements/             Frozen historical proposals (cite as legacy:NNN; new work lives in the workspace enhancements/)
openspec/                 OpenSpec proposals, specs, archives
modules/                  Test-only OPM modules used by integration tests
testdata/                 CUE module fixtures consumed by package tests
Taskfile.yml              fmt / vet / lint / test entry points
```

The OPM core schema is no longer vendored or embedded — it is fetched at runtime from `CUE_REGISTRY` via `opm/schema` (the `apis/` tree and the old `opm/api` / `opm/apiversion` packages were removed). Artifact loading and instance synthesis are kernel internals (`opm/internal/loader`, `opm/internal/synth`) reached through the acquire verbs and `Kernel.SynthesizeInstance`; their sentinels are declared in `opm/errors`. A standalone `opm/validate/` package was contemplated but never landed — the one validation primitive lives on `*kernel.Kernel` (`ValidateConfigDetailed`; a single value is a one-element `[]Source`), composed with the `ConfigSchema()` accessors on `*module.Module` / `*module.Instance`.

## Render

```text
Kernel.AcquireInstanceFromDir | Kernel.SynthesizeInstance  ->  *module.Instance   (validated, carries Source)
Kernel.AcquirePlatformFromDir                             ->  *platform.Platform (carries Source)
Kernel.Render(RenderInput{Instance, Platform, RuntimeName, Skew, LocalReplacements, SkipUnprovided})
        stage one generated render module (cue.mod promoted from both inputs; each input imported by directory replacement)
        promote the inputs' own local-module.cue replacements under LocalReplacements (platform whole, instance on
        instance-only paths) -> RenderDiagnostics.Replacements rows; refuse an input carrying one when the opt-in is off
        verify every OPM-namespace path either input requires is covered; apply the skew policy (SkewWarn | SkewRefuse)
        build once in a fresh cue.Context, dropped on return
        decode `diagnostics` -> RenderDiagnostics (pairs, unmatched, unresolved, skipped, unify, unhandled traits, over-subscribed, resolved versions, required contracts)
        fail-closed gate     -> *RenderError carrying the diagnostics and typed causes (errors.As)
        decode `rendered`    -> []*kernel.Compiled with instance / component / transformer provenance
```

`Render` is the kernel's single render verb. Matching and transformer execution are CUE inside the build (the glue in `opm/internal/renderstage/render.cue.tmpl`), not Go; the build reports its verdicts as data and the kernel decodes them. A dry run is `Render` with `Compiled` discarded: the build evaluates every pair regardless, and `RenderDiagnostics` carries the pairing diagnosis. A developer's `cue.mod/local-module.cue` (a dependency redirected to a directory or another module) reaches the render only under `RenderInput.LocalReplacements`; off, an input carrying a replacement is refused rather than silently rendered against the published pin. A provider-fulfilled demand that no enabled catalog provides is marked `Unprovided` on its unresolved row; under `RenderInput.SkipUnprovided` the build skips exactly those demands (a skipped trait leaves its component rendering, a skipped resource omits the whole component) and reports each as a `RenderDiagnostics.Skipped` row, while every other refusal stands. Values are validated where they are applied: `AcquireInstanceFromDir` and `SynthesizeInstance` unify them inside the instance build and assert concreteness on the result, and `Render` performs no validation pass of its own.

Each render is its own CUE build in its own `cue.Context` that does not outlive the call (ADR-005), and every other verb works the same way (ADR-007): the Kernel holds no build context, an acquired artifact's `Package` pins the context of the call that built it for as long as the caller holds the artifact, and nothing else is retained. A single Kernel is safe for concurrent use across its method calls, so a consumer shares one Kernel per process; a render pool is sized by memory rather than by core count; see the `opm/kernel` package documentation.

`*kernel.Compiled` is the kernel's terminal output. Platform identity for compiled output is the frontend's concern — each consumer wraps `Compiled` in its own platform-specific resource type.

## Quick start

See [`docs/getting-started.md`](docs/getting-started.md) for an end-to-end walkthrough — constructing a `Kernel`, loading a Module, layered values validation, acquiring an instance and a platform module, and rendering the instance into `*kernel.Compiled` values.

## API stability

The library follows SemVer 2.0.0. The public surface is everything under `opm/`. Two distinct compatibility tracks coexist and must not be confused:

- **Go module SemVer** governs the Go types and function signatures consumed by downstream binaries. Before GA, a breaking Go API change is a `feat!` commit that advances the prerelease counter; from GA, a breaking change here is a major bump of the library.
- **OPM schema versioning** governs the CUE shapes consumed at runtime — `#Module`, `#ModuleInstance`, `#Platform`, `#Component`, transformer contracts. The kernel MUST be able to load and render older schema versions seamlessly so that downstream implementations inherit multi-version support without per-implementation effort.

The two tracks are independent. The kernel's schema loader pins an exact core release by default (`schema.DefaultSchemaModule`), and a bare major (`opmodel.dev/core@v2`) is opt-in; acquisition and `Render` resolve core through each artifact's own `cue.mod`. An additive shape change within an OPM schema major therefore needs no Go API change: artifacts take it up by re-pinning core, and `DefaultSchemaModule` moves separately, in the release cascade's `fix(deps)` PR, which re-verifies the render glue. A shape break in the schema is itself a coordinated library-breaking event.

### Beta promise

From its first beta, the library is on the path to GA, together with the other prerelease lines (`opmodel.dev/core@v2`, cli, opm-operator). A breaking change is still allowed during beta, but only as a `feat!` commit whose `BREAKING CHANGE:` footer is the migration note the [CHANGELOG](CHANGELOG.md) shows ([ADR-010](adr/010-beta-migration-notes-in-changelog.md)). It advances the `-beta.N` counter and never moves the Go module path to a new major (no `v2` of the library during beta). Stable lines (`opmodel.dev/catalogs/opm@v4` and the module fleets) keep the normal SemVer rule: a break is a new major. A core beta break that would force a catalogs/opm major needs owner sign-off. GA drops the suffix: `prerelease: false` plus a visible carrier commit per package, in dependency order. GA is cut only when the GA exit criteria in 0021:D8 (workspace root, `enhancements/0021/03-decisions.md`) hold, among them the library's API-quality criteria, 0021:D8:R11 to 0021:D8:R14: no `cue.Value` in Render output, typed fetch and resolution errors, docs and specs that match the code, and three consecutive library betas with no breaking change.

Pin an explicit `v1.0.0-beta.N` (`go get github.com/open-platform-model/library@v1.0.0-beta.N`): `go get ...@latest` resolves the retired `v0.7.0`.

## OPM schema resolution

The library does NOT vendor or embed the OPM core schema. At runtime the kernel resolves `opmodel.dev/core@v2` through CUE's module system against `CUE_REGISTRY`, then memoizes the built `cue.Value` in a per-`Kernel` `*schema.Cache`.

Key pieces:

- `opm/schema` — schema loader (`Loader` interface, `OCILoader` sole public implementation), per-instance memoization (`Cache`), CUE path inventory, metadata types, and the `PublicRegistry` const (`opmodel.dev=ghcr.io/open-platform-model,registry.cue.works`).
- `opm/kernel` — `kernel.WithSchemaLoader(schema.Loader)` configures which Loader the Kernel's cache wraps; `(*Kernel).SchemaCache()` exposes the cache to callers (a bare-major loader makes instance synthesis resolve the core release through it; a pinned loader, the default, needs no load). `kernel.WithRegistry(string)` sets the ONE registry mapping every kernel operation resolves through: the render build's catalog imports, registry module acquisition, directory acquisition, instance synthesis, the compilation of file-backed values sources (a values file that imports a registry module) and the default schema cache.

Frontends (CLI, operator, future Crossplane fn) set `CUE_REGISTRY` (typically to `schema.PublicRegistry`) before constructing the Kernel. The library auto-applies no default; this keeps Principle I (kernel neutrality) intact and avoids hidden lookups. See `docs/getting-started.md` for the deployment pattern, including the warm-cache pre-seeding pattern for restricted environments.

## Helper boundary (`opm/helper/`)

Anything under `opm/helper/` is opt-in convenience for embedding the kernel; a frontend MAY skip it and call the kernel directly. Outside it, the library has two tiers that are not optional: the kernel, which binds every frontend, and the Kubernetes tier `opm/k8s/` ([ADR-011](adr/011-kubernetes-tier-beside-the-kernel.md)), which binds every frontend that targets Kubernetes. Its packages are `opm/k8s/labels`, the label vocabulary, and `opm/k8s/object`, which converts compiled objects, orders them by kind and finds duplicate apply identities.

The boundary is enforced by `task lint`, not just documented: a `depguard` rule in `.golangci.yml` forbids `opm/kernel`, `opm/module`, `opm/platform`, `opm/catalog`, `opm/schema`, `opm/errors` and every package under `opm/internal/` from importing anything under `opm/helper/`. Six more fence `opm/k8s/`: no other `opm/` package imports it, it imports no Flux, helper or internal package, of the Kubernetes modules it imports only `k8s.io/apimachinery`, its non-test files import only the standard library, `cuelang.org/go/cue` and its subpackages, `k8s.io/apimachinery` and the library's own `opm/` packages (a strict allow list), `opm/k8s/labels` imports only the standard library, and `opm/k8s/health` imports only the standard library and `k8s.io/apimachinery`. Two keep every kernel package and every helper package off `k8s.io` and `sigs.k8s.io`, a tenth keeps client-go, controller-runtime and Flux out of every file under `opm/`, and an eleventh keeps `golang.org/x/mod/semver` (and any second SemVer library) out of every `opm/` package but `opm/internal/modversion`.

Today this layer holds exactly two subpackages:

- `opm/helper/platformmodule` — Platform module generation from catalog coordinates: `Roots` + `Closure` derive the tidied dependency list from published module files (through a caller-configured `ModFileSource`), `Generate` renders `cue.mod/module.cue` and `platform.cue` deterministically, `Files.WriteTo` writes them into a caller-owned directory for `Kernel.AcquirePlatformFromDir`. The core pin defaults to `schema.DefaultSchemaVersion()`.
- `opm/helper/objectset` — **Deprecated: use `opm/k8s/object`.** Duplicate rendered object identities: `Duplicates` and `DuplicateIdentitiesError`, identical to the copy in `opm/k8s/object`, which is now the home of that Kubernetes vocabulary. This copy is frozen and stays only until the cli and the operator have moved their imports; a later change removes it.

Layered values validation lives on the kernel itself — see `Kernel.ValidateConfigDetailed` and the `Source` type in `opm/kernel`. See `enhancements/001-kernel-redesign-around-platform/02-design.md`.

The loader and synth subpackages that used to live here folded into `opm/internal/loader` and `opm/internal/synth`: the kernel imported both, so the "opt-in" tier was mandatory. Their behaviour is unchanged and reached through the acquire verbs and `Kernel.SynthesizeInstance`; the sentinels a frontend branches on (`ErrInvalidPackage`, `ErrWrongKind`, `ErrMissingRequiredField`, `ErrMissingModule`, `ErrMissingName`, `ErrMissingNamespace`, `ErrMissingSource`, `ErrSchemaUnavailable`) are declared in `opm/errors`.

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

- `CONSTITUTION.md` — design principles (kernel neutrality, type safety, separation of concerns, SemVer discipline, mergeable sections).
- `openspec/config.yaml` — normative constitution source.
- `opmodel.dev/core@v2` — current OPM schema, published as an OCI CUE module (sources live in the workspace `core/` repo).
- `docs/getting-started.md` — end-to-end embedding walkthrough.
- `docs/design/` — CUE evaluator notes: the v0.17.x closedness regression and its canary, plus historical bug records whose code no longer exists.
- `enhancements/` — frozen historical proposals; the single-build render design is workspace enhancement 0019.
- `adr/` — architecture decision records (ADR-006: one CUE build per artifact; ADR-005: shares-nothing renders; ADR-008: the kernel plans lifecycle and never runs it).
- `CHANGELOG.md` — released-version history (generated by release-please).
- `migrations/README.md` — migration-documentation policy: per-change fragments, dormant until GA (pre-GA breaking changes are recorded in `CHANGELOG.md` and the OpenSpec archive).
