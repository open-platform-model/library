## Why

Four prune rounds (`library-slot-prune`, `library-phase-and-values-prune`, `library-render-cutover`, and the dead-export pass in PR 95) left a residue: about twenty exported symbols with no non-test caller in the library, `cli` or `opm-operator` at `v1.0.0-alpha.26`. Each is a SemVer promise (Principle VI) and a detour for a reader deciding which tier they are on (Principle VII). This change removes them before the alpha line hardens, so consumers re-pin once and the surface that reaches GA is the surface something calls.

This is slice 1 of the eight-slice simplification plan reviewed on 2026-09-05 (dead surface, one API tier, internal dedupe, CUE-owned verdicts, moving `compat` and `core`, Kernel-context retention, cli copies, tests and docs). It is deliberately the smallest slice: pure removal, no new mechanism, no consumer code change required to compile.

**Scope statement (Principle VIII).** Every item below is a deletion or an unexport with zero non-test callers outside the library, verified by grep against both consumers. Anything that needs a replacement mechanism (per-source partial validation for the cli's `vet`, a per-call `cue.Context`, an `AcquireModuleFromDir` verb) is out of scope and named in the later slices.

## What Changes

**`opm/kernel` (BREAKING, `refactor(kernel)!:`):**

- **BREAKING** `Kernel.ValidateConfig` and `Kernel.ValidateConfigPartial` are removed. `ValidateConfigDetailed(schema, sources)` is the one validation entry; it always enforces concreteness. `ValidateOption` and `Partial()` are removed with them; the internal per-source pass under `WithValues` (`acquire.go:219`) calls the unexported `runValidate` with `requireConcrete=false` directly. The cli's `module vet` keeps its own per-source partial pass until slice 7 designs a kernel spelling for it; it does not call the kernel today.
- **BREAKING** `Kernel.ProcessModuleInstance` becomes the unexported `processInstance(ctx, spec)`. Its `values` branch is dead (both callers, `AcquireInstanceFromDir` and `SynthesizeInstance`, pass `cue.Value{}` because values are already unified inside the build) and its `mod module.Module` parameter feeds only a name fallback the instance shape gate makes unreachable (`metadata.name` is required concrete). Values enter through `WithValues` and `synth.InstanceInput.Values`, both validated where they are applied.
- **BREAKING** `Kernel.NewInstanceFromValue` is removed. Instances are constructed only by `AcquireInstanceFromDir` and `SynthesizeInstance`; `module.NewInstanceFromValue` goes with it (no non-test caller once the wrapper is gone).
- **BREAKING** `Kernel.LoadSourceFromString` is removed; `LoadSourceFromBytes` stays (slice 2 gives it its first consumer when `synth.InstanceInput.Values` becomes `[]Source`).
- **BREAKING** `Source.Name` is removed. Every constructor set it and nothing read it; `Origin` is the attribution key CUE positions carry.
- `ValuesFileName` becomes unexported (`opm-values.cue` is a kernel-internal reserved name; no consumer reads the constant).
- `Kernel.LoadInstancePackage` **stays** (deliberate: the instance twin of `LoadModulePackage` / `LoadPlatformPackage`, which the cli uses, and the loader `attributeValuesError` reads).

**`opm/module`, `opm/platform` (BREAKING):**

- **BREAKING** `module.CueContextOwner` and `platform.CueContextOwner` are removed; `module.NewModuleFromValue(v)` and `platform.NewPlatformFromValue(v)` take only the value. The argument has been ignored since the constructors were written ("preserved so future kernel-scoped state can be threaded"); no such state arrived in six alphas, and the kernel wrappers `Kernel.NewModuleFromValue` / `Kernel.NewPlatformFromValue` that hid the argument keep their signatures, so the cli's two call sites compile unchanged.
- `Instance.Components()` and `Instance.ConfigSchema()` **stay** (deliberate: symmetry with `Module.ConfigSchema()`, which the cli uses).

**`opm/schema`:**

- **BREAKING** `DecodeModuleMetadata`, `DecodeInstanceMetadata`, `DecodePlatformMetadata` become unexported. Their only callers are the three constructors and `processInstance`. `ModuleMetadata`, `InstanceMetadata`, `PlatformMetadata` stay exported; consumers read them through `Module.Metadata` and friends.
- `PublicRegistry` and `DefaultSchemaModule` **stay** (deliberate: documented in `CLAUDE.md`, `README.md` and `docs/getting-started.md` as the constants a frontend sets `CUE_REGISTRY` from; zero maintenance cost).
- The `schema.Loader` / `versionedLoader` two-interface shape is **not** touched here; it is a design change, not a removal, and a later slice may fold it.

**`opm/errors` (BREAKING):**

- **BREAKING** `UnresolvedDemand.UnstatedPosture` is removed. The kernel never sets it: an unstated trait posture is a build error (`render_decode.go:299-318`, `TestRender_UnstatedPostureIsBuildError`), not a diagnostics row.
- **BREAKING** `IdentityError.Artifact` is removed. It is always `"module"`; the doc on the type already says no catalog read site produces it. `Error()` drops the artifact prefix. The operator only type-checks the error (`opm-operator/internal/reconcile/resolution.go:26`) and reads no field.

**`opm/helper` (helper tier, BREAKING inside `opm/helper`):**

- **BREAKING** `platformmodule.WithCoreVersion` and `RootOption` are removed. One option, test-only; both consumers call `Roots(entries)`. Tests that pin a different core build `[]Dep` directly.
- **BREAKING** `loaderregistry.StagedSource` is removed; `LoadModulePackageWithSource` returns `(cue.Value, *module.Source, error)`, the shape the kernel unpacks it into on the next line. `loaderregistry.LoadOptions` stays until slice 2 merges the two loaders.
- `synth.ErrSchemaUnavailable` **stays** (the earlier review called it unreferenced; it is wrapped with `%w` at `synth/instance.go:180,184` and leaves with the core-schema build in slice 6).

**Tests and docs:**

- Tests that reached the removed symbols move onto the surviving entry points: `ValidateConfigDetailed` with one `Source` for the single-value cases, `AcquireInstanceFromDir` / `SynthesizeInstance` for the processing cases, a test-local helper over `LoadSourceFromBytes` for the string cases, the constructor calls drop their first argument, the `stubOwner` in `opm/helper/synth/instance_test.go` goes.
- `TestKernel_PrunedSurface` gains the removed method names; its comment and `CLAUDE.md` § Kernel API surface stop saying "values enter through `ProcessModuleInstance`". `opm/kernel/doc.go` and `README.md` follow.
- Deferred on purpose: `glueDiagnostics.Missing` / `.Resolved` (slice 4 reshapes the glue output), `compat.walkStruct`'s unused parameter (slice 5 moves the package), the `appendSchemaErrors` bool and `normalizeFieldPath` prefix branch (slice 7, when the cli copy that still reads the bool is deleted).

## SemVer classification

MAJOR (Principle VI): exported methods, fields, a type and an interface leave `opm/`. Pre-GA, so no migration fragment (ADR-004). Downstream migration cost: **zero source changes** in `cli` and `opm-operator`. Neither calls a removed symbol; the constructor wrappers they use keep their signatures; `IdentityError` and `UnresolvedDemand` are only type-checked. Both consumers re-pin in the next `fix(deps)` wave.

## Affected packages and downstream consumers

- `opm/kernel` (`wrappers.go`, `process.go`, `validate.go`, `source.go`, `source_loader.go`, `acquire.go`, `synth.go`, `doc.go`), `opm/module` (`module.go`, `instance.go`), `opm/platform` (`platform.go`), `opm/schema` (`decode.go`), `opm/errors` (`match.go`, `identity.go`), `opm/helper/platformmodule` (`generate.go`), `opm/helper/loader/registry` (`module.go`), `opm/helper/synth` (tests only).
- `cli`: no source change. `opm-operator`: no source change.
- `catalog_opm`, `modules`, `core`: no impact.

## Complexity justification

Net deletion of roughly 200 non-test lines and one interface declared twice; no new type, option or mechanism is introduced. The one internal addition is a boolean argument on an already-unexported function.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `config-validation`: the "three validation primitives" requirement becomes one (`ValidateConfigDetailed`, concrete-only, no option type); the `Source` requirement drops `Name`; the source-loader requirement lists `LoadSourceFromFile` and `LoadSourceFromBytes` only; the `ProcessModuleInstance` wrapping requirement is removed; the "typed convenience methods" scenarios spell composition with `ValidateConfigDetailed`.
- `kernel-runtime`: the wrapper requirement no longer names `CueContextOwner` or a `NewInstanceFromValue` wrapper; the single-pre-unified-values, canonical-implementations and Tier-2 requirements are restated without `ValidateConfig` / `ValidateConfigPartial` / `ProcessModuleInstance` (values are validated where applied: the `WithValues` build and `SynthesizeInstance`); the `SynthesizeInstance` documentation requirement drops the "call `synth.Instance` then `ProcessModuleInstance`" alternative.
- `artifact-types`: constructor helpers take only a `cue.Value`; the `NewInstanceFromValue` scenarios are removed (instances are constructed by the kernel's acquire paths only).
- `schema-dispatch`: the metadata decoders are internal to the library; the constructor requirement is restated against the unexported decoders.
- `helper-packages`: the registry loader returns `(cue.Value, *module.Source, error)` in place of `StagedSource`.
