## 1. Spike (GATING — Phase 2+ does not start until this resolves)

- [ ] 1.1 Build a minimal virtual package (overlay) importing the platform source + one catalog known to exercise output-local hidden fields (the `transformer-output-hidden-field-scope-bug.md` §12 repro / `opmodel.dev/catalogs/opm`), evaluate in a single build, and read a transformer's `#transform` output directly off the closed result.
- [ ] 1.2 Compare single-build output against today's `Composed`-routed output for the same transformer; record whether hidden-field resolution is concrete.
- [ ] 1.3 Classify the outcome — **green** (single-build concrete) / **partial** (concrete but multi-version bucket collision per §10.5) / **red** (not expressible). Write the finding into design.md and decide scope before continuing.
- [ ] 1.4 Measure cost delta (memory/wall-time) of single-build composition for a wide SemVer range vs. current path (open question 1).

## 2. materialize: single-build composition (green/partial only)

- [ ] 2.1 Keep subscription resolution + version enumeration + range/allow/deny filtering + stable-default selection unchanged; isolate the index-then-`FillPath` step as the only thing replaced.
- [ ] 2.2 Replace it with single-build composition: synthesize the package importing the platform + selected catalog `path@version`s; let CUE build `#composedTransformers` / `#matchers`.
- [ ] 2.3 Confirm `schema.ComposedTransformers` / `schema.MatchersResources` / `schema.MatchersTraits` resolve on the single-build `Package`.
- [ ] 2.4 Preserve `Source`, resolved-version-per-path map, `MaterializeError` (`Kind`, path, version, cause), non-mutation, idempotency, and the opt-in cache — no changes to those.

## 3. materialize + compile: remove the Composed hatch

- [ ] 3.1 Remove `MaterializedPlatform.Composed` (or deprecate as an alias of `Package` for one release per the SemVer decision); delete the WARNING in `opm/materialize/types.go`.
- [ ] 3.2 `opm/compile/execute.go`: read `#transform` off `Package`; `opm/compile/module.go`: stop threading `r.platform.Composed`.
- [ ] 3.3 Grep-guard: add a test/lint that fails if any new `#transform` read goes through a removed `Composed` path (or, on a red spike, a guard that fails on a `Package` `#transform` read — see 5.3).

## 4. Tests

- [ ] 4.1 Regression: a transformer with output-local hidden fields renders concrete read directly off `mp.Package` (the case `Composed` existed for) — proves the seam is gone.
- [ ] 4.2 Preserve materialize behavior: existing range/allow/deny, stable-default, indexing, `MaterializeError`, idempotency, and cache tests pass unchanged.
- [ ] 4.3 Concurrency: the v0.17 concurrent read-only scenario still holds on the single-build `Package` (no race, no re-materialization) — open question 2.
- [ ] 4.4 Flow integration (`task cue:test:flow`) renders end-to-end against the rewritten materialize.

## 5. Records

- [ ] 5.1 `MIGRATIONS.md`: `MaterializedPlatform.Composed` removed/deprecated (MAJOR or one-release deprecation); note the behavioral contract is preserved.
- [ ] 5.2 Update `docs/design/transformer-output-hidden-field-scope-bug.md` — mark §12 resolved by single-build composition, point at this change and ADR-003.
- [ ] 5.3 If the spike is **red**: do NOT remove `Composed`; instead convert this change to "contain the seam" — keep `Composed`, add the enforced guard (3.3 fallback), update proposal/design/spec to reflect the descoped outcome, and record the CUE limitation upstream.
- [ ] 5.4 Confirm ADR-003 status line references this change.

## 6. Verify

- [ ] 6.1 `task fmt && task vet && task lint`
- [ ] 6.2 `task test` and `task cue:test:flow`
- [ ] 6.3 Confirm `cli/` and `opm-operator/` build; if `Composed` was a consumed surface, update consumers + `MIGRATIONS.md` migration recipe.
