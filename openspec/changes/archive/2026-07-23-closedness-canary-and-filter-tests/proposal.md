# Proposal: closedness-canary-and-filter-tests

Test-only change from the 2026-07-21 workspace fixture audit. No production code changes; one fixture deletion.

## Why

Two detection gaps and one hygiene item:

1. **The CUE closedness-regression canary is dead.** The evaluator defect (spurious `field not allowed` on comprehension-guarded nested struct fields — present in `cuelang.org/go` v0.17.0 *and* v0.17.1; upstream #4423-as-filed was a sibling bug) is mitigated only by the hoisted-guard authoring rule in catalog_opm. The former detection path, `TestIntegration_Live_ValidateRealConfig`, now PASSES on known-bad CUE versions because that same workaround removed the trigger pattern from the catalog it resolves — so nothing in any suite would notice if a future CUE bump reintroduced exposure, and nothing signals when upstream actually fixes our shape. A minimal two-fixture standalone reproducer exists (`docs/design/cue-closedness-regression-alpha2.md`).
2. **The pre-release `range` opt-in is specified but untested.** `openspec/specs/platform-materialization` (Version Enumeration and Filtering) states that a `filter.range` carrying a pre-release identifier admits pre-releases, but `opm/materialize/filter_test.go` only covers the `allow` opt-in and `highestStable`. This exact mechanism is how open ranges resolve CI `-dev.*` tags over `-alpha.*` releases (enhancement 0006 OQ18) — it should be pinned by tests before OQ18 is decided.
3. **Orphaned fixtures**: `testdata/platform/v1alpha2/platform.cue` claims consumers it does not have, and `testdata/out.cue` has none (verified by grep 2026-07-21).

## What Changes

- New hermetic **closedness canary test pair** (embedded CUE, no registry/network/cue.mod): the *trigger form* is asserted to still produce `field not allowed` — when a future CUE bump makes it evaluate clean, the test fails loudly with instructions to re-run the regression matrix and re-evaluate the hoisted-guard rule (it must not be silently deleted); the *hoisted form* is asserted to evaluate clean, guarding the shipped workaround on every CUE bump.
- New `filterVersions` table rows: range-with-pre-release-identifier selection, and the mixed `-alpha`/`-dev` family case pinning that dev tags out-sort alpha releases within an admitting range (the OQ18 mechanism).
- Delete `testdata/platform/` and `testdata/out.cue`.

## Capabilities

### New Capabilities

- `cue-regression-canary`: the evaluator-level regression guard — trigger-form expected-fail semantics, hoisted-form pass, hermeticity.

### Modified Capabilities

- `platform-materialization`: no behavior change — the Version Enumeration and Filtering requirement gains scenarios for the pre-release-range opt-in and dev-over-alpha ordering it already specifies.

## Impact

- **Packages**: new test-only package (e.g. `opm/internal/cueregression`); `opm/materialize/filter_test.go`; `testdata/` deletions.
- **SemVer**: none (tests + dead fixtures). Commit type `test:`.
- **Dependencies**: none. Pure unit tier — runs under `-short`, offline.
