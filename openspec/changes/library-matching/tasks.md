# Tasks — library-matching

> Depends on `library-compat-comparator` (imports `compat.StripProvenance` + `compat.CompareAPIVersions`). Five strands, each a separately-green group; group order minimizes fixture churn (labels first — every later group's fixtures are authored in the flipped vocabulary). Breaking change — MIGRATIONS entry mandatory; `Migration:` trailer if `add-migration-guard` is live.

## 1. The matchLabels flip (D36)

- [x] 1.1 `schema/paths.go`: add `MatchLabels = cue.ParsePath("matchLabels")`.
- [x] 1.2 `registrytest`: generated resources/traits/blueprints emit `matchLabels` mirroring their current label values; keep emitting `metadata.labels` (real catalog keeps the duplicate for now — fixtures mirror reality).
- [x] 1.3 `compile/match.go:111`: label set builds from the component's `matchLabels`; predicate and `pairTransformer` follow the set unchanged.
- [x] 1.4 `compile_test.go` raw fixtures: matching keys move to `matchLabels:`; add one case proving `metadata.labels` alone no longer matches.
- [x] 1.5 `compile/module.go:29-30,197` doc comments: summary labels are descriptive, matching reads `matchLabels`; stale `core.opmodel.dev/workload-type` citation removed. `schema/context.go` untouched — add a pinning test asserting the transformer context still carries `metadata.labels` (site-4 must-not-flip).
- [x] 1.6 `testdata/modules/web_app` transitional comment rewritten. Full `task test` + GHCR flow test (`OPM_FLOW_TEST_FORCE=1`) green — the real catalog authors `matchLabels`, so the flip holds against published bytes.

## 2. The load-bearing unify rung (D26/D30/D27)

> Mechanism revised at implementation (see design): `compat.StripProvenance` cannot rebuild kernel-side operands, so the rung excludes provenance-located diagnostics from the unify verdict instead of stripping the operands.

- [x] 2.1 `compile/match.go`: exclude diagnostics at `metadata.catalogVersion`/`metadata.description` (any metadata block) from the unify verdict and the recorded cause (`excludeProvenance`); no operand round-trip, no memoisation needed.
- [x] 2.2 Tests: two fixtures differing only in `catalogVersion`/`description` unify clean; a genuine spec divergence still raises `UnifyError` (with provenance conflicts absent from the cause); closed-definition retention (module sets a field the definition closes out → still refused).
- [x] 2.3 Pin the diagnostic surface: `UnifyError.{Component,FQN}` survive and — cost reversal vs D30's strip — surviving causes keep document positions.

## 3. Demand resolution (D28 + D4 diagnostics)

- [x] 3.1 `opm/errors`: `UnresolvedDemand` (value type, fields per design) + `UnresolvedDemandsError` aggregate with `Unwrap() []error`; tests pin shape and `errors.As` walkability.
- [x] 3.2 `compile/match.go`: state (a) empty bucket and state (b) all-candidates-disqualified accumulate `UnresolvedDemand` (resources always; traits per posture) with `Alternatives` in `compat.CompareAPIVersions` order and `Disqualified` carrying the recorded `UnifyError`s.
- [x] 3.3 Trait posture: read effective `optional` off the attachment value; non-concrete fails closed with the unstated-posture diagnostic; `Warnings()` carries only effectively-optional unhandled traits. `registrytest` knobs: `*false` posture + no-posture.
- [x] 3.4 `compile/module.go` + `kernel`: `Plan`/`Compile` fail on `len(plan.Unresolved) > 0` beside `UnmatchedComponentsError` (both reported when both apply); `Match` stays phase-only. Kernel integration tests for: undemandable resource fails Compile; non-optional unhandled trait fails; optional trait warns; different-apiVersion alternative named in the error.

## 4. The comparator (D34/D4)

- [x] 4.1 `compile/match.go`: alternatives ordering delegates to `compat.CompareAPIVersions`; function renamed to its contract-key role; SemVer path retained only where build keys are ordered.
- [x] 4.2 Test: the measured pathological triple (`v1alpha1`, `v2`, `v10`) sorts identically from any input order.

## 5. The single-provider guard (D32/D37)

- [x] 5.1 `materialize/index.go`: during the existing walk, read `fulfilment` off each **required** embedded contract copy; accumulate provider-fulfilled keys → owning catalogs; error on two catalogs for one key (`MaterializeError` naming both paths + key) and on embedded copies disagreeing on `fulfilment`.
- [x] 5.2 `index_test.go` / registrytest: two-catalog provider-fulfilled conflict; same key catalog-fulfilled with many providers passes; disagreement case.

## 6. Own-graph test (D10)

- [x] 6.1 Integration fixture: platform catalog publishes a divergent definition for a contract key the module's own dependency also defines; assert `UnifyError` reports the divergence and rendered output follows the module-side definition (fails if module and platform ever share one resolution). Cite the 04-graduation gate in the test comment.

## 7. Specs, docs, migrations

- [x] 7.1 Spec deltas: `platform-matching` (MODIFIED label predicate, always-unify, alternatives ordering; ADDED unresolved-demand + trait-posture requirements; REMOVED Defensive Ambiguity Handling and the stale `MissingFQN` "release name" wording); `platform-materialization` (ADDED Single-Provider Guard).
- [x] 7.2 `MIGRATIONS.md` `## Unreleased — Breaking` entry per design (four bullets + recipes); `Migration: library-matching` trailer if the guard workflow is live.
- [x] 7.3 `docs/design/compile-pipeline-known-gaps.md`: unresolved-demand gap closed; ambiguity-collapse history note updated.

## 8. Verify & record

- [ ] 8.1 `task check` clean; GHCR flow test forced.
- [ ] 8.2 Record back in `enhancements/0010/`: slice → `done` + history event; note the catalog follow-on (duplicate `metadata.labels` droppable from catalog primitives once this ships; component-side labels stay for render reads) and the unlanded `opm.opmodel.dev/workload-type` rename for the catalog repo.
