## 1. errors

- [x] 1.1 Add `MissingFQN{Release, Component, FQN, Alternatives}` to `opm/errors` with `Error()`.
- [x] 1.2 Add `UnifyError{Component, FQN, Cause}` to `opm/errors` with `Error()` + `Unwrap()`; `Cause` carries the CUE error tree verbatim (reachable via `errors.As` for `cuelang.org/go/cue/errors.Error`).
- [x] 1.3 Unit-test both: `MissingFQN.Alternatives` population shape; `UnifyError` verbatim-message + `errors.As` reachability.

## 2. compile/match.go — always-unify rung

- [x] 2.1 Change `Match(components, plat *platform.Platform)` → `Match(components, mp *materialize.MaterializedPlatform, releaseName string)`; read `#composedTransformers` / `#matchers` off `mp.Package`.
- [x] 2.2 Insert the unify rung before predicate: for each candidate, unify over the `component.#resources` ∩ `transformer.requiredResources` intersection (and `#traits` ∩ `requiredTraits`); on failure append `UnifyError` and skip the candidate (D1).
- [x] 2.3 Resolve unify validation mode (Q3): use `cue.Concrete(false)` for structural agreement; confirm no concrete fields are expected pre-render.
- [x] 2.4 Replace empty-bucket handling: a demanded FQN with no candidates appends one `MissingFQN` per `(release, component, fqn)`; compute `Alternatives` by same-`modulePath`/`name` walk of `#composedTransformers` keys, sorted by SemVer. Accumulate all misses in one pass (no fail-fast).
- [x] 2.5 Keep multi-fulfiller / predicate disambiguation and `requiredLabels` `MissingLabels` handling unchanged (D17); confirm `MissingLabels` stays the soft non-match, distinct from `MissingFQN`.

## 3. MatchPlan shape

- [x] 3.1 Add `Missing []oerrors.MissingFQN` and `Unify []oerrors.UnifyError` fields to `MatchPlan`; remove `MatchResult.MissingResources` / `MissingTraits` (keep `MissingLabels`).
- [x] 3.2 Reconcile `Warnings()` / `UnhandledTraits` with `MissingFQN` (Q1): keep both, document the distinction (unhandled trait vs absent FQN).

## 4. kernel wiring

- [x] 4.1 Retype `MatchInput.Platform` / `PlanInput.Platform` / `CompileInput.Platform` to `*materialize.MaterializedPlatform` in `opm/kernel/inputs.go`.
- [x] 4.2 Update `phases.go` `Match` / `Plan` / `Compile` bodies to pass `mp` + `in.ModuleRelease.Metadata.Name` into `compile.Match`.
- [x] 4.3 Confirm `compile.Module` / Execute paths read `#composedTransformers` off the materialized platform (no behavior change beyond the input type).

## 5. Retire Compose

- [x] 5.1 Delete `opm/helper/platform/compose.go` + `compose_test.go`; remove `MultiFulfillerError` if it has no remaining referent.
- [x] 5.2 Remove `(*Kernel).ComposePlatform`; update `opm/helper/platform/doc.go`.
- [x] 5.3 Grep for `Compose` / `ComposePlatform` / `MultiFulfillerError` callsites and migrate or delete.

## 6. Caller migration + fixtures

- [x] 6.1 Migrate `cmd/flow-inspect` to build `*Platform` → `Materialize` → `Match(*MaterializedPlatform)`.
- [x] 6.2 Rewire `opm/compile` and `opm/helper/platform` tests onto `add-platform-materialize`'s `modregistrytest` fixtures.
- [x] 6.3 Add match tests: clean unify pair; divergent body → `UnifyError`; absent FQN → `MissingFQN` with alternatives; multi-fulfiller still disambiguates by predicate.

## 7. Docs + validation gates

- [x] 7.1 `MIGRATIONS.md`: `Match(*Platform)` → `Materialize` + `Match(*MaterializedPlatform)`; `Compose` → subscription `#registry` + `Materialize`; `MatchResult.MissingResources` → `MatchPlan.Missing`.
- [x] 7.2 `task fmt`.
- [x] 7.3 `task vet`.
- [x] 7.4 `task lint`.
- [x] 7.5 `task test`.
