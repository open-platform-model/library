## Why

Core `@v0.3.0` reshaped `#Platform.#registry` into a path-keyed map of catalog *subscriptions* and downgraded `#composedTransformers` / `#matchers` to optional, kernel-filled slots (enhancement 0001, D13/D14). Nothing in the library realizes those subscriptions yet: there is no step that turns a subscription spec into a usable transformer index, so the matcher has nothing to consult on the new schema. This slice adds that realization step — `Materialize` — as the additive foundation the match rewrite then builds on.

## What Changes

This change is **additive (MINOR)**. No existing signature changes; `Match` / `Plan` / `Compile` keep taking `*platform.Platform` in this slice (they move to `*MaterializedPlatform` in the follow-up `rewrite-match-materialized` change).

- **NEW `opm/materialize/`** — package with the realization flow: walk `#registry`; per subscription path, enumerate published versions via `modregistry.ModuleVersions`; filter the candidate set Go-side by `range` ∧ `allow` ∧ `deny` (SemVer constraints, D10/D11 — CUE cannot evaluate range syntax); pull each survivor via `cue/load` against the resolved registry; read each pulled `#Catalog.#transformers`; index by stamped FQN into a composed transformer map plus a `#matchers.{resources,traits}` reverse index.
- **NEW `MaterializedPlatform`** — a sealed view carrying the source `*Platform`, the kernel-filled CUE value (composed transformers + matchers present), and the resolved version per subscription path (for diagnostics).
- **NEW `MaterializeError`** in `opm/errors` — structured pull/decode failure carrying a `kind` discriminator (`"catalog"` | `"core-schema"`, D24) plus the subscription path and attempted version.
- **NEW `opm/materialize/cache/`** — opt-in `MaterializeCache` interface + reference LRU implementation + spec-content-hash key derivation (D14). The kernel holds no cache; consumers (operator, CLI) wire their own. Justified by Principle VII: the operator needs invalidation keyed to a CR generation, and the kernel must stay stateless (Principle I), so the cache cannot live on the kernel.
- **MODIFIED `Kernel`** — gains a `Registry` field + `WithRegistry` option (the single OCI discovery surface for catalogs, shared with the schema loader) and a `Materialize` method that delegates to the package. Default callers compile unchanged.

Out of scope (separate changes): the match-algorithm rewrite and the `*MaterializedPlatform` signature swap (`rewrite-match-materialized`); the catalog repackage to the D19 `#Catalog` shape and its `@0.1.0` publish; the `modules/` publish task. Tests here use an in-memory `modregistrytest` registry with inline `#Catalog` fixtures, so this slice does not depend on the real catalog being repackaged first.

## Capabilities

### New Capabilities

- `platform-materialization`: resolving a `#Platform`'s path-keyed subscriptions into a `MaterializedPlatform` — version enumeration, Go-side `range`/`allow`/`deny` filtering, OCI pull, `#Catalog.#transformers` indexing into the composed map + `#matchers` reverse index — plus the `MaterializeError` contract and the opt-in `MaterializeCache` primitives.

### Modified Capabilities

- `kernel-runtime`: `Kernel` gains a `Registry` field + `WithRegistry` option and a `Materialize` method. Additive only — existing methods and their signatures are unchanged.

## Impact

- **New packages**: `opm/materialize/` (realization flow), `opm/materialize/cache/` (opt-in cache primitives).
- **`opm/errors/`**: adds `MaterializeError`.
- **`opm/kernel/`**: adds `Registry` field, `WithRegistry` option, `Materialize` method.
- **New type**: `MaterializedPlatform` (final package home decided in design.md).
- **New dependency**: `github.com/Masterminds/semver/v3` for SemVer-range constraint parsing (D11). Justified per Principle VII — CUE evaluates exact versions but not range expressions like `">=1.0.0 <2.0.0"`; Masterminds is the de-facto Go constraint library.
- **Tests**: `mod/modregistrytest` in-memory OCI registry + inline `#Catalog` fixtures; exercises the production resolver → registry-client → loader path unchanged (no test-only `Loader` backdoor, consistent with the OCI-loader change). The version-enumeration path specifically needs a queryable registry — a pre-seeded on-disk cache cannot answer "which versions exist?". The substrate, fixture layout, and enumerate→pull→read flow are **verified by spike** (design.md § Research & Decisions); the real `#Catalog`-shape indexing and the `FillPath` approach remain to be confirmed during implementation.
- **SemVer**: **MINOR**. No breaking change; the new surface is additive and opt-in. `cli/` and `opm-operator/` are unaffected until they choose to call `Materialize`; migration cost for existing callers is zero.
- **Sequencing**: follows `core@v0.3.0` (shipped) and `replace-embedded-schema-with-oci-loader` (shipped, archived); precedes `rewrite-match-materialized`, which consumes `MaterializedPlatform` and is where the breaking signature change lands.
