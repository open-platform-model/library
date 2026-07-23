# Design: closedness-canary-and-filter-tests

## Context

The regression's full history and bisected trigger rule live in `docs/design/cue-closedness-regression-alpha2.md`; the authoring-rule mitigation in `catalog_opm/docs/cue-guard-closedness-workaround.md`. The minimal reproducer (verified 2026-07-16: v0.16.1 clean / v0.17.0-alpha.1 clean / v0.17.0 FAIL / v0.17.1 FAIL, error `x.out.a.b: field not allowed`) is two files with no deps. `language.version` does not gate the bug — only the evaluating SDK version matters — so the Go-SDK evaluation path reproduces it without any `cue.mod`.

## Goals / Non-Goals

**Goals:** restore hermetic detection in both directions (reintroduced exposure on a bump; upstream fix signal); pin the pre-release range semantics; remove dead fixtures.

**Non-Goals:** testing blueprints or any catalog primitive (deferred to the colocated-test framework, enhancements#6); filing or tracking an upstream issue; changing `filterVersions` behavior (OQ18's resolution is a separate decision).

## Decisions

### LD1: Canary lives in a dedicated test-only package with embedded fixtures

New `opm/internal/cueregression` containing only `closedness_test.go` with the two fixtures as Go string constants (the reproducer body, minus the `cue.mod` file, verbatim from the design doc). Evaluation: `cuecontext.New().CompileString(fixture)` + `Validate` — no load.Instances, no filesystem, no module resolution. A dedicated package keeps the expected-fail semantics from being diluted into an unrelated suite and gives the failure message one obvious home.

*Alternative — testdata `.cue` files + load.Instances:* rejected; drags in module-root resolution for zero fidelity gain (the bug is evaluator-level).

### LD2: Expected-fail is asserted narrowly and inverts loudly

The trigger-form test asserts the validation error exists **and** matches `field not allowed` at path `x.out.a.b` — a narrow match so an unrelated compile error cannot fake-satisfy the canary. If validation comes back clean, the test FAILS with a message that includes the running `cuelang.org/go` version (via `debug.ReadBuildInfo`) and instructs: re-run the regression matrix in `docs/design/cue-closedness-regression-alpha2.md`, re-evaluate (do not silently revert) the catalog_opm hoisted-guard rule, then update this canary's expectation deliberately. The hoisted-form test asserts clean evaluation and failure means the workaround itself broke — a release blocker for any CUE bump.

### LD3: Filter rows extend the existing table, no new harness

`opm/materialize/filter_test.go` already table-drives `filterVersions`. Added rows: (a) `Range: ">=0.6.0-dev.0 <0.7.0"` over `[v0.5.0, v0.6.0-dev.1, v0.6.0]` → selects `v0.6.0-dev.1` + `v0.6.0`, excludes `v0.5.0`; (b) the OQ18 shape — `Range: ">=1.0.0-alpha"` over `[v1.0.0-alpha, v1.0.0-alpha.1, v1.0.0-dev.1784212239.g0c11c12]` → all three selected; plus one Materialize-level assertion that the resolved (highest) version from that survivor set is the `-dev.*` tag, documenting that dev out-sorts alpha. These tests pin *current* behavior; if OQ18 later changes the semantics, they are the rows that change with it.

### LD4: Fixture deletion is verify-then-delete

Re-run the consumer grep in the change (not trusting the audit's snapshot), then delete `testdata/platform/` and `testdata/out.cue`. If a consumer surfaces, stop and record instead of deleting.

## Risks / Trade-offs

- [Upstream fixes the bug in a patch release we bump to casually] → that is the point: the canary inverts and forces the deliberate re-evaluation instead of a silent drift.
- [Narrow error-match breaks if CUE rewords the diagnostic] → acceptable; the failure message explains both interpretations (fixed vs reworded) and points at the matrix doc.
