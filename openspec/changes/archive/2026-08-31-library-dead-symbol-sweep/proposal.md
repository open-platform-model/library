## Why

The library kernel review (2026-08-30) swept every exported identifier under `opm/` against library production code, `cli` and `opm-operator`, and found a set of symbols with zero call sites outside their own definitions and tests. Each was added ahead of a consumer that never arrived (Principle VII), and several are still required by specs that describe symbols which do not exist. Every one of them is SemVer surface the kernel has to keep stable and document for nobody.

## What Changes

All removals are **BREAKING** by SemVer rule; none has a caller in `cli` or `opm-operator` (verified 2026-08-30), so no consumer migration is needed.

- **BREAKING** `opm/errors`: remove the four sentinels `ErrValidation`, `ErrConnectivity`, `ErrPermission`, `ErrNotFound` and the `Wrap` helper. The identically aliased `oerrors` in `cli` and `opm-operator` resolve to those repos' own `pkg/errors`, which is why the symbols looked used. Remove `MaterializeKindCoreSchema`, reserved for a failure `Materialize` never emits.
- **BREAKING** `opm/compile`: remove the `ModuleResult` alias of `CompileResult`.
- **BREAKING** `opm/schema`: remove `DecodeProviderMetadata` and `ProviderMetadata` (the provider artifact was retired with the platform construct); remove the eight path variables no production code reads (`KnownResources`, `KnownTraits`, `ComposedTransformers`, `Matchers`, `MatchersResources`, `MatchersTraits`, `ContextModuleInstanceMetadata`, `ContextComponentMetadata`, `ContextRuntimeName`); remove `AnnotationDefaultNamespace` (the annotation key itself stays in core's `#Module`; ADR-001 is unaffected).
- **BREAKING** `opm/materialize/cache`: remove the package (`MaterializeCache`, `LRU`, `Key`). The one long-running consumer holds a single generation-keyed slot instead and never adopted it.
- **BREAKING** `opm/helper/loader/bytes`: remove the doc-only package; it has exported nothing since it was scaffolded.
- **BREAKING** `opm/helper/loader/registry` and `opm/kernel`: remove the value-only `LoadModulePackage` and its wrapper `Kernel.LoadModuleFromRegistry`. `LoadModulePackageWithSource` and `Kernel.AcquireModuleFromRegistry` are the acquisition path both consumers use.
- **BREAKING** `opm/compat`: remove `StripProvenance`. The comparison walk implements 0010 D30 by skipping provenance fields at every depth, and the matcher excludes provenance from the unify verdict; the standalone strip has no caller.
- Spec hygiene in the same sweep: retire the `kernel-runtime` requirement for a `ParseModuleInstance` deprecated alias that never shipped, and the `ModuleResult aliased` scenario.
- Docs: `CLAUDE.md` layout table and materialize-cache paragraph, `README.md` layout, `opm/helper/doc.go`, `MIGRATIONS.md` entry under Unreleased, Breaking.

Out of scope (owned elsewhere): `context.Context` threading and the `WithLogger` / `WithTracer` / `WithClock` slots (enhancement 0009 D9); `Plan`, `Kernel.Validate`, `CompileInput.Values` and the typed validation wrappers (`library-phase-and-values-prune`).

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `schema-dispatch`: the path inventory no longer lists the eight unused paths; the metadata decoders are the three artifact decoders; the default-namespace annotation constant is removed.
- `helper-packages`: no `loader/bytes` skeleton; the registry loader exposes the source-carrying loader only.
- `platform-materialization`: the opt-in materialize cache requirement is removed; `MaterializeError.Kind` has one kind.
- `kernel-runtime`: `LoadModuleFromRegistry` and the `ParseModuleInstance` alias requirements are removed; the `ModuleResult` alias scenario is dropped from the compile-rename requirement.
- `catalog-compatibility`: the provenance strip requirement is removed; the walk's own provenance skip remains the D30 implementation.

## Impact

- Packages: `opm/errors`, `opm/compile`, `opm/schema`, `opm/materialize/cache` (deleted), `opm/helper/loader/bytes` (deleted), `opm/helper/loader/registry`, `opm/kernel`, `opm/compat`.
- Consumers: none of the removed identifiers is referenced by `cli` or `opm-operator`; both keep compiling unchanged. MAJOR under SemVer regardless (Principle VI); recorded in `MIGRATIONS.md`.
- Tests: the tests that exercised the removed symbols go with them; no behaviour covered by a remaining test changes.
