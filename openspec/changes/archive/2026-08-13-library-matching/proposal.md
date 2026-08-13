# Proposal — library-matching

## Why

The match rung becomes load-bearing. Under 0010, matching is the seam where the platform's promises meet the module's demands, and five decisions put teeth into it that today's code does not have:

- **Unresolved demands do not fail a render.** `plan.Missing` is written and never consumed in production; a bucket whose candidates all fail records nothing; only a fully-unmatched component stops `Compile`. D28: every declared resource is required, and an unhandled trait fails identically unless its effective `optional` is true — today unhandled traits degrade to warning strings unconditionally.
- **The always-unify rung compares provenance and would fire on every catalog release.** D26/D30: the unify comparison must exclude exactly `metadata.catalogVersion` and `metadata.description` (definition and instance) while keeping the closed definition in the comparison. The strip mechanism shipped in `library-compat-comparator` (`compat.StripProvenance`) was measured unable to rebuild kernel-side operands (see design), so the rung excludes provenance-located diagnostics from the unify verdict instead — equivalent under D25, closedness and positions preserved.
- **Matching still reads the deleted vocabulary.** Core v2 moved matching to `matchLabels` (D36) and the retarget kept the matcher reading `metadata.labels` against a transitional duplicate the catalog authors on every primitive. This change flips the read at the two matching sites (`match.go:111` set-build, `match.go:350` predicate) and closes the window. The two non-matching readers keep `metadata.labels`: the transformer context (D36 explicitly keeps `matchLabels` out of `#TransformerContext` — render reads like `hpa_transformer`'s workload-type lookup depend on it) and the display-only `ComponentSummary`.
- **The alternatives ordering is non-transitive.** `sortFQNsBySemVer` switches comparison rule per pair; measured: `v1alpha1 < v2`, `v2 < v10`, `v10 < v1alpha1` all true, so the same three FQNs sort differently depending on input order. D34/D4: contract keys get a kube-aware total ordering (`compat.CompareAPIVersions`), build keys keep SemVer — the two key shapes D4 split get the two comparators the split implies.
- **Nothing guards single-provider contracts.** D32 as corrected by D37: the guard keys on a contract's declared `fulfilment` and counts required demands — bucket arity was measured undetectable (catalog buckets legitimately hold 3–8 transformers). It lands in materialize's reverse-index build, where the owning catalog of every transformer is structural.

Plus D10's surviving obligation: the own-graph invariant — a module's primitives resolve through the module's own dependency graph, never one shared with the platform — holds structurally today but is untested, and under D4 a shared resolution would make the unify rung compare a value against itself and never fail. The graduation gate requires a test that fails if the two ever share resolution.

This change is 0010's `library-matching` slice (see `enhancements/0010/plan.yaml`). Its decisions: D4, D10, D26, D28, D30, D32, D34, D36, D37. It depends on `library-compat-comparator` (consumes `StripProvenance` and `CompareAPIVersions`); it is independent of `library-acquire-and-subscription` in code (disjoint files) and sequenced after it by plan convention only.

## What Changes

- **`opm/compile/match.go`:**
  - Label set builds from `matchLabels` (new `schema.MatchLabels` path constant); the predicate reads the same set — the flip is two sites plus the path constant.
  - The unify rung excludes provenance-located diagnostics (`metadata.catalogVersion`/`metadata.description` under any metadata block) from the `Unify(...).Validate(cue.Concrete(false))` verdict, keeping closed definitions in the comparison. D30's recorded position-loss cost does not materialize — surviving diagnostics keep positions; `UnifyError`'s structural fields carry component and FQN.
  - Unresolved demands become errors: an empty bucket for a demanded resource, a bucket whose every candidate is disqualified, and an unhandled non-optional trait all surface as typed unresolved-demand diagnostics that fail `Plan`/`Compile` (`Match` stays phase-only and keeps returning the full plan). The contract-key diagnostic distinguishes "nothing implements this contract" from "implemented at a different apiVersion" via the same-base alternatives set.
  - Trait posture: the effective `optional` is read off the component's trait attachment; a non-concrete `optional` (un-gated catalog that states no posture) fails closed — treated as load-bearing, with a diagnostic naming the unstated posture.
  - `sortFQNsBySemVer` splits per D4: contract keys via `compat.CompareAPIVersions`, build keys via SemVer.
- **`opm/materialize/index.go`:** the single-provider guard — for every contract key whose definition (read from the provider's embedded `required*` copy) declares `fulfilment: "provider"`, at most one subscribed catalog may supply a transformer for it; a second is a `MaterializeError` naming both catalog paths and the key. Embedded copies that disagree on `fulfilment` for one key are themselves an error (divergent contract).
- **`opm/compile/module.go` / `opm/kernel`:** `plan.Missing` gains its production consumer; the unresolved-demand failure joins `UnmatchedComponentsError`'s exit path with its own typed error.
- **Own-graph test:** a fixture where the platform's catalog carries a divergent definition for a key the module also defines — passes only while module-side resolution is the module's own; fails if the two ever share a resolver.
- **Fixtures:** `registrytest` emits `matchLabels`, `fulfilment`, and an `optional` posture on generated primitives; the `web_app` fixture's transitional duplicate-label comment is resolved (component-side `metadata.labels` stays where render reads need it); `compile_test.go`'s raw-CUE fixtures move their matching keys to `matchLabels`.

## Capabilities

### New Capabilities

<!-- none -->

### Modified Capabilities

- `platform-matching`: label predicate reads `matchLabels`; always-unify excludes the D30 denylist from both operands; unresolved demands and non-optional unhandled traits fail; the defensive-ambiguity requirement is removed (D32/D37 supersede it); alternatives ordering becomes total and transitive.
- `platform-materialization`: the reverse-index build gains the single-provider guard.

## Impact

- **`opm/` public surface:** new typed error(s) in `opm/errors`; `MatchPlan` semantics change (Missing consumed; unresolved demands fail Plan/Compile); no signature deletions.
- **SemVer: breaking** (Principle VI). Renders that previously succeeded now fail: modules with undemandable resources, non-optional unhandled traits, platforms with duplicate provider-fulfilled contracts, and modules relying on `metadata.labels` for matching without catalog-derived `matchLabels`. Ships as `feat!:` with a `MIGRATIONS.md` entry (+ `Migration:` trailer if the guard workflow is live).
- **Real-catalog compatibility:** the flip is safe against the published catalog — it authors `matchLabels` on every matching-relevant primitive (the transitional duplicate was `metadata.labels`, kept for the old read). The flow test against GHCR proves it.
- **Coordination (out of scope here):** `catalog_opm` may drop the transitional duplicate `metadata.labels` from its primitives afterward — except that component-side `metadata.labels` remains meaningful for render reads (`hpa_transformer` reads the workload-type off the component); the duplicate's removal is catalog-owned follow-on work, noted in the 0010 record. The `opm.opmodel.dev/workload-type` key rename (D36) is unlanded catalog-side; the flip here is key-neutral.
- **Packages touched:** `opm/compile`, `opm/materialize` (index only), `opm/errors`, `opm/schema` (one path constant), `opm/internal/registrytest`, kernel integration tests, `testdata/`.
- **Ordering:** after `library-compat-comparator` (hard, imports it). Lands with `add-migration-guard` in whichever order — if the guard is live first, this PR carries the trailer.
