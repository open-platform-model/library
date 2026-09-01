## Context

See proposal.md, Why. The 2026-09-01 review verified each removed identifier has zero production readers in the library and zero references in `cli` and `opm-operator`; the only readers are the identifiers' own tests, two spec requirements, and doc prose. Two of the removals retire surfaces that were kept "for later" by the 2026-08-31 sweeps and have since acquired an owner-level answer: `MissingFQN` was pre-softened by the D28 rework ("retained for compatibility"), and `core.Resource`/`core.Identity` is the exact subject of enhancement 0012 OQ3.

Constraints that shape the approach:

- Principle VI: MAJOR; pre-GA, consumers migrate in the same wave (here: nothing to migrate) and the record is changelog plus archive.
- Principle VII bounds the list to zero-reader identifiers: `MatchPlan.Unmatched`, `ModuleVersion()`, `TransformError.ComponentName` and `alternativesFor` all have readers and stay.
- 0019 owns `Kernel.Match`/`MatchInput` (D10 retires the Go matcher) and the platform file-load path (D6/D9); neither is touched here.
- 0012 is `draft`; its OQ3 is answered in place per the enhancements rules, with this change cited.

## Goals / Non-Goals

**Goals:**

- Remove the write-only diagnostics (`MatchPlan.Missing`/`MissingFQN`, `CompileResult.Unmatched`), the dead accessors (`TransformError.Component()`, `Instance.ModuleFQN()` and the `InstanceView.ModuleFQN` member), and the implementation-less adapter contract (`core.Resource`, `core.Identity`).
- Keep every diagnostic a consumer actually reads byte-identical: `UnresolvedDemand`(sError), `UnifyError`, `MatchPlan.Unmatched`, `UnhandledTraits`, warnings.
- Record 0012 OQ3's resolution where 0012's promotion gate reads it.

**Non-Goals:**

- Any match-verdict or render-output change. The matcher stops *recording* misses twice; it does not stop detecting them.
- Touching `Kernel.Match`, `MatchInput`, `LoadPlatformPackage`, `NewPlatformFromValue` (deferred to 0019, recorded in the exploration and the proposal's scope).
- Modernizing the stale binding-era prose in `artifact-types`/`schema-dispatch` beyond the requirements this change touches.

## Decisions

### D1: `UnresolvedDemand` fully subsumes `MissingFQN`; no information is lost

**Context**: An empty demand bucket today produces two records: a `MissingFQN` (instance, component, fqn, alternatives) appended to `plan.Missing`, and an `UnresolvedDemand` (component, fqn, kind, alternatives, disqualified, posture) appended to `plan.Unresolved`. Both call the same `alternativesFor`. Production code and both consumers read only the unresolved set (the CLI renders `UnresolvedDemandsError`; the operator classifies it via `errors.AsType`).

**Explored**: Keeping `Missing` as the "phase-only" record because `Match` alone does not fail on misses. Rejected: `MatchPlan.Unresolved` is returned by `Match` too, so the phase-only reader already has the full diagnosis; the one field `Missing` carries that `Unresolved` does not (the instance name) is known to every caller that holds the plan.

**Decision**: Remove the field and the type; the empty-bucket path records only the `UnresolvedDemand`. An effectively-optional trait with an empty bucket keeps surfacing through `UnhandledTraits` warnings exactly as today.

**Rationale**: One miss, one record, one consumer-facing shape. The spec requirement being removed said so itself: the load-bearing diagnosis is the unresolved-demand set.

### D2: Remove `CompileResult.Unmatched`; keep `MatchPlan.Unmatched`

**Context**: Since the D28 gate, an unmatched component fails `Compile` with typed errors before execution, so the result-level field is only ever the empty slice. The plan-level field is real data and is what `cli`'s log output prints.

**Explored**: Keeping the result field as a convenience mirror. Rejected: a field that is empty by construction on every reachable success path documents a state that cannot occur.

**Decision**: Delete the result field and its always-empty initialization; the plan inside the result remains the unmatched surface.

**Rationale**: The gate made the field unreachable; the spec delta states the invariant explicitly so it cannot silently return.

### D3: Delete the neutral `core.Resource`/`core.Identity` contract outright (0012 OQ3)

**Context**: The interface has zero implementations anywhere; both frontends define their own `Resource` structs and import `opm/core` only for `Compiled`. 0012 OQ3 asks delete-or-retain and blocks 0012's promotion; 0012's own alternatives analysis rejects retaining an abstraction with one hypothetical implementer whose vocabulary cannot express Kubernetes concepts (propagation policy, ownerReferences, subresources).

**Explored**: Retaining it unused until 0012 resolves OQ3 itself. Rejected by the owner (2026-09-01): the question's library half is exactly "does anything use it", which is now measured; deleting resolves OQ3 with evidence instead of leaving the blocker to re-derive it.

**Decision**: Delete `resource.go` and its test; `Compiled` is the terminal output and platform identity is stated as the frontend's concern. 0012's OQ3 and history are updated in place, citing this change.

**Rationale**: 0012 D2's direction (kernel written for Kubernetes, no portability abstraction maintained on its behalf) makes the neutral contract a dead end in every future; if 0012 is later rejected, the contract can return shaped by an actual second platform, which it never had.

### D4: Drop `ModuleFQN` from `InstanceView` rather than keeping the interface stable

**Context**: `BuildTransformerContext` fills `#moduleInstanceMetadata.fqn` from `InstanceFQN()` (the instance's own identity, 0010 D41); `ModuleFQN()` exists beside it and nothing calls it.

**Explored**: Keeping the member for interface stability. Rejected: the interface is library-owned with one production implementer; a member no caller reads is exactly the surface under removal, and a test fake carrying an extra method stays valid Go regardless.

**Decision**: Remove the accessor and the interface member; `ModuleVersion()` stays (the context builder reads it). Source-module identity remains reachable through `Instance.Package` at the module-metadata path.

**Rationale**: The interface should list what the context builder consumes, nothing more; the removed member was the one entry that lied about that.

## Risks / Trade-offs

- [A future frontend wants per-miss records without the fail-gate] → `Match` still returns the full plan without failing; `Unresolved` is that record. If a distinct "miss" shape is ever needed, it returns with its reader.
- [0012 lands and wants a shared resource type after all] → It gets a Kubernetes-native one per 0012 D2; the deleted neutral tuple would not have satisfied it (its own analysis).
- [Spec drift: the modified `artifact-types`/`schema-dispatch` requirements keep stale binding-era wording] → Accepted; wording beyond the touched clauses is a separate hygiene pass, as the 2026-08-31 sweep also recorded.

## Migration Plan

1. `library`: tasks 1 through 4; `task check`.
2. Docs and 0012 OQ3 note (task 5); no consumer edits exist.

Rollback is a single revert; no consumer, schema or registry surface is involved.
