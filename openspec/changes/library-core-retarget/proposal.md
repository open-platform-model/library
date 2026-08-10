# Proposal — library-core-retarget

## Why

Enhancement 0010 moved the OPM core schema to a new major: `opmodel.dev/core@v2`, published on GHCR with `v2.0.0-alpha.4` as the first (and only) complete tag on the line — alpha.1–alpha.3 are partial cuts that resolve but lack the identity package. The library still consumes `opmodel.dev/core@v1` (retired at `v1.1.0-alpha.1`). Every remaining 0010 library slice (`library-identity-read-checks`, `library-subscription-collapse`, `library-contract-match`, `library-match-labels`) and both downstream repos (`cli`, `opm-operator`) depend on this crossing landing first.

This is an **import rewrite, not a re-pin**: a major crossing that `task update-deps` cannot perform. The runtime retarget itself is one constant (`schema.DefaultSchemaModule`), but the test fleet must move with it, because a module or catalog evaluates against the core version its own `cue.mod` declares — v1-authored fixtures cannot unify with v2-typed values at render seams.

This change is 0010's `library-core-retarget` slice (see `enhancements/0010/plan.yaml`). Its decisions: D42.

## What Changes

- **`schema.DefaultSchemaModule` flips to `"opmodel.dev/core@v2"`.** The floating-major mechanism is unchanged: the existing bare-major expansion in `OCILoader.loadVersioned` resolves the latest version within the major (today `v2.0.0-alpha.4`; the `-0.dev.*` tags sort below the alphas by SemVer, so they never win).
- **Test fixture writers become major-aware.** `opm/internal/registrytest` hardcodes `core@v1` in the deps line, the catalog import, and the module-path major; these derive from the fixture's (defaulted) core version instead, so v2-defaulted and v1-pinned fixtures coexist. `defaultCoreVersion` moves to `v2.0.0-alpha.4`.
- **A v2-shaped fixture catalog replaces the published-catalog test dependency.** Mainline kernel/flow tests currently materialize the real `opmodel.dev/catalogs/opm` (v1-built, from GHCR). No v2-built catalog can exist until 0010's `catalogs-republish` — which depends on library slices downstream of this one. The circle breaks with a library-owned fixture catalog under the `testing.opmodel.dev` prefix: tests serve it from the in-process registry; local diagnostic flows (`cmd/flow-inspect`, `task cue:publish`) serve it from `localhost:5000` per the workspace Registry Policy. CI stays GHCR-only.
- **Mainline fixtures reauthored to the v2 shape**: `testdata/` (test module + synth fixture), `testdata/modules/web_app` (core v2, catalog dep re-pointed to the fixture catalog, blueprint import flattened per D42), `modules/opm_platform` (subscription re-keyed to the fixture catalog; `filter: range:` becomes the required scalar `version:` — `#SubscriptionFilter` no longer exists in v2), kernel flow test fixtures.
- **Historical pinned tests are kept unchanged**: the v0.5.2 closedness canary (`composed_open_test.go`), the library#31 regression (pinned to `catalogs/opm@v0` v0.6.0), and the synth boundary tests (`pinnedCache("v1.0.0-alpha.1")`) — each pins both its fixture core dep and its schema cache coherently, independent of the default. The `docs/design/repro-hidden-field/` repro stays pinned to v1 (it reproduces a v1-era bug).
- **No behaviour change.** The subscription-resolution, matching, and identity-read code paths are untouched; those are the downstream slices. Two transitional invariants make that safe (see design).

**Recorded deviation from 0010's plan:** the slice concern names "two D42 import sites." The one at `instance_integration_test.go:154` does **not** move — it imports `blueprints/workload` from catalog v0.6.0, where blueprints genuinely live one segment deep; flattening it while pinned there would break the fixture. Only the `web_app` site moves. This deviation gets recorded back in 0010 when the slice lands.

## Capabilities

### New Capabilities

<!-- none -->

### Modified Capabilities

- `schema-dispatch`: the single consumed schema module becomes `opmodel.dev/core@v2`; default-resolution and resolved-version scenarios move to the v2 line. Loader mechanics (interface, OCILoader, Cache) are unchanged.

## Impact

- **`opm/` public surface:** `schema.DefaultSchemaModule` value changes (`@v1` → `@v2`). No signature changes.
- **SemVer: breaking** (Principle VI). A consumer relying on the default loader now resolves core v2, whose `#Module`/`#Catalog` shapes reject v1-authored artifacts. Ships as `feat!:` with a `Migration: library-core-retarget` trailer and a `MIGRATIONS.md` `## Unreleased — Breaking` entry (recipe: pin `schema.OCILoader{Module: "opmodel.dev/core@v1.0.0-alpha.1"}` to stay on v1; rollback is the same rewrite in reverse to `v1.1.0-alpha.1`).
- **Downstream:** `cli` and `opm-operator` pick this up via their own retarget slices (`cli-coordinate-adoption`, `operator-library-retarget`); neither consumes this release until then.
- **Packages touched:** `opm/schema` (one constant + doc comments), `opm/internal/registrytest`, `opm/internal/schematest` (comments), test files under `opm/kernel/`, `opm/helper/synth/`, `opm/materialize/`, plus CUE fixtures (`testdata/`, `modules/opm_platform`, new fixture catalog) and docs (`CLAUDE.md`, `Taskfile.yml` comments, `MIGRATIONS.md`).
- **Scope note (Principle VIII):** the crossing is atomic — the default flip and the mainline-fixture move must land together or the suite breaks — but the tasks stage every preparatory step (major-aware writers, fixture catalog authoring) as independently green commits, leaving the atomic step as small as the crossing allows. Splitting further means splitting the slice, which 0010's plan already sized as one.
