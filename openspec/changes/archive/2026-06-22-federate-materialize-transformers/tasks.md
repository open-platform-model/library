## 1. materialize: federate the native surfaces

- [x] 1.1 Reshape `MaterializedPlatform` (`opm/materialize/types.go`): add `Transformers cue.Value` (the open `#composedTransformers` map from `indexCatalogs`) and `Matchers cue.Value` (the open `{resources,traits}` reverse index); remove `Composed` and `Package`; keep `Source` and `Resolved`. Replace the `Package` WARNING doc with field docs for `Transformers` (concrete-by-construction) and `Matchers`.
- [x] 1.2 `opm/materialize/materialize.go`: stop `FillPath`-ing `composed`/`matchers` onto `p.Package`; return `&MaterializedPlatform{Source: p, Transformers: composed, Matchers: matchers, Resolved: resolved}`. Keep subscription resolution, enumeration, range/allow/deny filtering, multi-version selection, and `MaterializeError` paths unchanged.
- [x] 1.3 Confirm `indexCatalogs` output is consumed directly (no schema-path re-lookup needed); the FQN keys on `Transformers` and the `resources`/`traits` keys on `Matchers` are read as-is by consumers.
- [x] 1.4 Preserve `Source`, resolved-version-per-path map, `MaterializeError` (`Kind`, path, version, cause), non-mutation, idempotency, and the opt-in cache — no behavioral changes to those.

## 2. compile + flow-inspect: read native fields

- [x] 2.1 `opm/compile/module.go`: thread `r.platform.Transformers` (was `r.platform.Composed`) into `executeTransforms`; update the doc comment (drop the "NOT r.platform.Package" warning rationale).
- [x] 2.2 `opm/compile/execute.go`: read `#transform` off the passed `Transformers` map (mechanically identical to today's `Composed` read); drop the closedness-bug WARNING comment.
- [x] 2.3 `opm/compile/match.go`: read the composed map and reverse index from `mp.Transformers` / `mp.Matchers` (was `mp.Package.LookupPath(schema.ComposedTransformers / MatchersResources / MatchersTraits)`). Matching semantics (exact version-bearing FQN lookup → unify → predicate) unchanged.
- [x] 2.4 `cmd/flow-inspect/main.go`: print FQN keys / matcher index from `mp.Transformers` / `mp.Matchers`; optionally print `mp.Source.Package` `#registry`/metadata for parity (design Open Question 2).
- [x] 2.5 Add a guard test that fails if `Materialize` ever fills `#composedTransformers`/`#matchers` onto the closed platform (lock the seam shut) — e.g. assert `mp.Source.Package.LookupPath("#composedTransformers")` stays empty.

## 3. Tests

- [x] 3.1 Regression: a transformer with output-local hidden fields renders concrete read directly off `mp.Transformers` (the case the `Composed` hatch existed for) — proving the seam is gone, not relocated. Rework/retarget `composed_open_test.go` onto `Transformers`.
- [x] 3.2 Preserve materialize behavior: existing range/allow/deny, multi-version selection, stable-default, indexing, divergent-FQN conflict, `MaterializeError`, idempotency, and cache tests pass (updated to read native fields) unchanged in intent.
- [x] 3.3 Multi-version: a subscription selecting two same-major versions yields distinct version-bearing FQN entries in `mp.Transformers`, and a component embedding `…@0.5.0` matches only `…deployment-transformer@0.5.0` (not `@0.5.1`) — D3 matcher contract.
- [x] 3.4 Concurrency: the v0.17 concurrent read-only scenario holds on the native `Transformers`/`Matchers` (no race, no re-materialization).
- [x] 3.5 Flow integration (`task cue:test:flow`): a multi-version subscription matches and renders end-to-end off the native index; assert §10.5 zero-pairs symptom is cleared (or scope a residual matcher follow-up per design Risk/Open Question 1).

## 4. Records

- [x] 4.1 `MIGRATIONS.md`: MAJOR break — `MaterializedPlatform.Composed`/`Package` removed. Recipe: `mp.Composed` → `mp.Transformers`; spec/`#registry` reads off `mp.Package` → `mp.Source.Package`; matcher/executor read native fields. Note the behavioral contract is preserved.
- [x] 4.2 Update `docs/design/transformer-output-hidden-field-scope-bug.md` (§12/§13) — mark resolved by federation (the closed twin is never built); point at this change and ADR-003.
- [x] 4.3 Confirm ADR-003 status line references this change. (`rewrite-materialize-single-build` and its single-build spike are already removed; the rejected approach + PARTIAL finding live in this change's `design.md`.)
- [x] 4.4 Capture the **C2 future design** (per-version build isolation: each selected version kept as its own build instance, executor routes per version) as an ADR or a design note, with the C1→C2 trigger (transforms needing re-evaluation against their full native build at execution time).

## 5. Verify

- [x] 5.1 `task fmt && task vet && task lint`
- [x] 5.2 `task test` and `task cue:test:flow`
- [x] 5.3 Confirm `cli/` and `opm-operator/` build unchanged (they treat `*MaterializedPlatform` as an opaque handle; no `mp.Package`/`mp.Composed` reads).
