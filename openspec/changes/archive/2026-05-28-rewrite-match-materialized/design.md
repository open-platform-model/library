## Context

`add-platform-materialize` yields a `*MaterializedPlatform` whose CUE value answers `#composedTransformers` and `#matchers.{resources,traits}`. The current matcher (`opm/compile/match.go`) is already FQN-keyed — it does `matchersIndex.LookupPath(cue.Str(fqn))` — but it only verifies *set-membership* (`fqnSubset`): does the demanded FQN exist, and is the transformer's required-set a subset of the component's. It never checks that the consumer's primitive *body* agrees with the transformer's `requiredResources[FQN]` / `requiredTraits[FQN]` body. Under the SemVer-FQN model (enhancement 0001, D5/D6), schema agreement MUST be enforced per paired primitive, a missing FQN MUST be a hard structured diagnostic (D20), and the kernel phase methods MUST consume the materialized platform.

This change is the breaking half of the kernel rewrite: it inserts the always-unify rung, adds `MissingFQN` / `UnifyError`, swaps the phase signatures to `*MaterializedPlatform`, and retires the obsolete `Compose` helper.

## Goals / Non-Goals

**Goals:**

- Always-on `unify` between consumer and transformer primitive bodies before predicate evaluation (D6).
- `MissingFQN` (per `(release, component, fqn)`, with `alternatives`) and `UnifyError` (verbatim CUE divergence), both accumulated in one pass.
- `Kernel.Match` / `Plan` / `Compile` (and their `*Input` structs) take `*MaterializedPlatform`.
- Remove `opm/helper/platform/Compose`.

**Non-Goals:**

- Multi-fulfiller / predicate-disambiguation logic — **unchanged** (D17). Same `[...#ComponentTransformer]` bucket shape, same label + extra-required-primitive tie-breaking, same workload-type discrimination.
- `#ctx` reads — the matcher consumes `#resources` / `#traits` / `metadata.labels` only; `#ctx` feeds the renderer, not matching. Out of scope.
- Catalog repackage and the `modules/` publish task.

## Decisions

### D1: The unify rung runs over the consumer↔transformer FQN intersection, not just the trigger key

For a candidate transformer paired to a component, unify **every FQN present in both** `component.#resources` and `transformer.requiredResources` (and the analogous traits intersection) — not only the FQN that triggered the bucket lookup. This honors "schema agreement is enforced for every paired primitive" (design 02 §High-Level Approach): a transformer requiring `container@1.4.0` *and* `volume@1.4.0` must agree with the consumer on both bodies, not just whichever one was walked first.

```go
// per (component, candidate) pair, before the existing predicate:
for fqn := range intersect(componentResourceFQNs, transformerRequiredResourceFQNs) {
    cv := component.LookupPath(append(schema.ComponentResources, cue.Str(fqn)))
    tv := candidate.LookupPath(append(schema.TransformerRequiredResources, cue.Str(fqn)))
    if err := cv.Unify(tv).Validate(cue.Concrete(false)); err != nil {
        plan.Unify = append(plan.Unify, oerrors.UnifyError{Component: comp, FQN: fqn, Cause: err})
        // unify failure ⇒ not a valid pairing; skip predicate for this candidate
    }
}
```

**Alternative considered:** unify only the keyed FQN. Rejected — it would let a divergent *second* required primitive slip through to render time, which is exactly the failure mode D6 exists to catch.

### D2: `MissingFQN` is the hard "no transformer for this FQN" diagnostic; `MissingLabels` stays the soft non-match

Two distinct outcomes, kept distinct:

- **`MissingFQN`** (NEW, hard): a demanded FQN whose `#matchers` bucket is empty — no transformer on the platform requires it. One per `(release, component, fqn)`, shape `{Release, Component, FQN, Alternatives}`.
- **`MissingLabels`** (existing, soft): a transformer was found for the FQN but its `requiredLabels` are not satisfied — a legitimate non-match (e.g. a `stateful` transformer skipped for a `stateless` component), not an error.

`alternatives` is computed by parsing the demanded FQN into `(modulePath/name, version)` and collecting every materialized FQN in `#composedTransformers` sharing that `modulePath/name`, sorted by SemVer. "Adjacent" is a presentation nuance the frontend may trim; the kernel returns the full same-name set.

### D3: `UnifyError.Cause` carries the CUE error tree verbatim

```go
type UnifyError struct {
    Component string
    FQN       string
    Cause     error // cuelang.org/go/cue/errors.Error — walkable via errors.As
}
```

No Go-side reformatting. CUE's `conflicting values "X" and "Y": ./a.cue:3:5 ./b.cue:7:9` message is authoring-grade (experiment 03); frontends walk the tree and render. `MissingFQN` + `UnifyError` land in `opm/errors` alongside the existing `TransformError`.

### D4: `MatchPlan` grows two slices; existing return shape preserved

```go
type MatchPlan struct {
    Matches         map[string]map[string]MatchResult
    Unmatched       []string
    UnhandledTraits map[string][]string
    Missing         []oerrors.MissingFQN  // NEW
    Unify           []oerrors.UnifyError   // NEW
}
```

`Match` still returns `(*MatchPlan, error)` — `error` stays for operational failures (bad inputs, CUE iteration errors); the structured match diagnostics live on the plan and accumulate in one pass (no fail-fast). `MatchResult.MissingResources` / `MissingTraits` are removed (the absent-FQN case is now `MissingFQN`); `MatchResult.MissingLabels` stays.

### D5: Signature swap + release threading

`MatchInput.Platform` / `PlanInput.Platform` / `CompileInput.Platform` retype to `*materialize.MaterializedPlatform`. `phases.go` updates the three bodies. `compile.Match` gains the materialized platform and a release identifier (needed to populate `MissingFQN.Release`):

```go
// before: func Match(components cue.Value, plat *platform.Platform) (*MatchPlan, error)
// after:  func Match(components cue.Value, mp *materialize.MaterializedPlatform, releaseName string) (*MatchPlan, error)
```

`Kernel.Match` already has the release (`in.ModuleRelease.Metadata.Name`); it passes it through. The matcher reads the same `schema.ComposedTransformers` / `MatchersResources` / `MatchersTraits` paths, now off `mp.Package` instead of `plat.Package` — minimal body change thanks to D2 of `add-platform-materialize`.

### D6: `Compose` removed outright, not deprecated

`opm/helper/platform/compose.go` + `compose_test.go` deleted. The Module-valued `#registry` composition it performs cannot produce a valid value against the subscription-shaped schema anyway. The repo's only consumer is the quarantined `modules/opm_platform/` fixture; no external consumers (per the recent archive). A deprecated stub that errors at call time costs attention without buying compatibility — delete it (consistent with the no-back-compat stance).

## Risks / Trade-offs

- **Unify cost per paired primitive** → bounded (a few CUE evaluations per pair); D6 accepted this explicitly. No `--strict` opt-out — uniform behavior is the point.
- **`MatchResult` field removal is a visible API break** → callers reading `MissingResources` migrate to `MatchPlan.Missing`; documented in `MIGRATIONS.md`.
- **Release threading changes `compile.Match`'s signature** → it was already changing for `*MaterializedPlatform`; folding the release param in is one more arg, not a second break.
- **`alternatives` over a large materialized universe** → it's a single walk of `#composedTransformers` keys per miss; negligible.

## Migration Plan

`MIGRATIONS.md` gains:

1. `Match(*Platform)` → `Materialize(*Platform)` then `Match(*MaterializedPlatform)` — the new platform-construction boundary.
2. `Compose(shell, modules...)` → build a `*Platform` with a subscription `#registry` directly, then `Materialize`.
3. `MatchResult.MissingResources` / `MissingTraits` → `MatchPlan.Missing []MissingFQN`.

`cmd/flow-inspect` and the `opm/compile` / `opm/helper/platform` tests migrate in this change, reusing `add-platform-materialize`'s `modregistrytest` fixtures — whose layout, enumerate→pull→read flow, and the read-only-cache cleanup helper (CUE writes extracts 0444) are verified by spike (see `add-platform-materialize` design.md § Research & Decisions). The transitive cross-module import path is also confirmed there, which the cross-catalog (D16) match tests rely on. Downstream `cli/` and `opm-operator/` migrate on adoption; cost concentrates at platform construction, not across the pipeline.

## Open Questions

- **Q1:** Does `MissingFQN` subsume the `UnhandledTraits` warning? They differ — `UnhandledTraits` = a component trait that no *matched* transformer consumes (soft, values ignored); `MissingFQN` = a demanded FQN with no transformer at all (hard). Lean: keep both, document the distinction. Confirm during implementation.
- **Q2:** When `compile.Match` is called directly (not via `Kernel`), the caller must supply `releaseName`. Acceptable, or should `MissingFQN.Release` be optional/blank in that path? Lean: required param, blank tolerated.
- **Q3:** Unify validation mode — `cue.Concrete(false)` (structural agreement) vs `cue.Concrete(true)` (fully concrete)? Matching happens pre-render on schema bodies, so structural (`false`) is correct; confirm no concrete fields are expected at this stage.
