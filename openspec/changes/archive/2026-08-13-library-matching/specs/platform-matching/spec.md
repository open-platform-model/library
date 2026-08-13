# platform-matching — Delta

## ADDED Requirements

### Requirement: Label Predicate

The matcher SHALL build a component's label set from the component's `matchLabels` field — the derived union of its attached primitives' matching keys — and SHALL test each candidate transformer's `requiredLabels` against that set. The matcher SHALL NOT read `metadata.labels` for matching; descriptive labels remain readable by non-matching consumers (component summaries, the transformer render context), whose reads are unchanged.

#### Scenario: matchLabels satisfies requiredLabels

- **WHEN** a component's attached primitives contribute `matchLabels` satisfying a candidate's `requiredLabels`
- **THEN** the candidate pairs

#### Scenario: Descriptive labels alone do not match

- **WHEN** a component carries a key only in `metadata.labels` and a candidate requires it
- **THEN** the candidate does not pair on that key

#### Scenario: Render context unchanged

- **WHEN** a transformer executes
- **THEN** its context's component metadata still carries `metadata.labels`, not `matchLabels`

### Requirement: Alternatives Ordering

Diagnostic alternatives (contract keys sharing a base with an unresolved demand) SHALL be ordered by a total, transitive comparator over the `vNalphaM | vNbetaM | vN` apiVersion ladder. The ordering SHALL be identical regardless of input order.

#### Scenario: Pathological triple sorts stably

- **WHEN** alternatives carry apiVersions `v1alpha1`, `v2`, and `v10`
- **THEN** the reported order is the same for every input permutation

### Requirement: Unresolved Demand Failure

Every resource a component declares is a required demand. A demanded resource for which the platform's matcher index holds no candidate, or for which every candidate is disqualified (by unification or by predicate), SHALL be reported as a typed unresolved-demand diagnostic carrying the component, the contract key, the same-base alternatives the platform does implement, and — when candidates existed — the per-candidate disqualification causes. The plan phase SHALL return the full diagnosis; the plan-for-execution and compile phases SHALL fail on any unresolved demand through a typed aggregate error. The diagnostic SHALL distinguish "nothing on this platform implements this contract" (no alternatives) from "implemented at a different apiVersion" (alternatives listed).

#### Scenario: Undemandable resource fails compile

- **WHEN** a component demands a resource contract no subscribed catalog provides
- **THEN** compile fails with an unresolved-demand error naming the component and key with no alternatives

#### Scenario: Different apiVersion named

- **WHEN** the platform implements the same contract base at a different apiVersion only
- **THEN** the unresolved-demand error lists those keys as alternatives

#### Scenario: All candidates disqualified is fatal

- **WHEN** candidates exist but every one fails unification or the predicate
- **THEN** compile fails with an unresolved-demand error carrying the disqualification causes

### Requirement: Trait Posture

An unhandled trait's effect SHALL be governed by its effective `optional` value read from the component's trait attachment (the declaring catalog's default unified with any attachment-site override): effectively optional degrades to a warning; effectively load-bearing fails exactly as an unresolved resource. A non-concrete `optional` — a catalog that states no posture — SHALL fail closed, treated as load-bearing, with a diagnostic naming the unstated posture.

#### Scenario: Optional trait warns

- **WHEN** an unhandled trait's effective `optional` is true
- **THEN** the render proceeds and the trait is reported as a warning

#### Scenario: Load-bearing trait fails

- **WHEN** an unhandled trait's effective `optional` is false
- **THEN** compile fails with an unresolved-demand error for the trait

#### Scenario: Unstated posture fails closed

- **WHEN** a trait's `optional` is not concrete
- **THEN** the trait is treated as load-bearing and the diagnostic names the unstated posture

## MODIFIED Requirements

### Requirement: Always-Unify Before Pairing

The matcher SHALL, before testing the predicate, unify the component's demanded definition with the candidate's required definition for every intersecting contract key, validating without requiring concreteness, and record a typed unify error for any divergence. Diagnostics located at exactly the provenance denylist — `metadata.catalogVersion` and `metadata.description`, directly under any `metadata` block — SHALL be excluded from the verdict and from the recorded cause; the closed definition SHALL remain in the comparison. Identity fields and labels remain compared. The typed error's structural fields (component, contract key) are the routing surface; surviving diagnostics keep their document positions.

#### Scenario: Provenance divergence does not fail unification

- **WHEN** the demanded and required definitions differ only in `metadata.catalogVersion` or `metadata.description`
- **THEN** the rung records no unify error

#### Scenario: Substantive divergence still refused

- **WHEN** the two definitions disagree on a `spec` field's domain
- **THEN** a typed unify error is recorded naming the component and contract key

### Requirement: Structured Missing-FQN Diagnostic

A missing FQN SHALL be reported as a structured `MissingFQN` value carrying the instance name, component name, the missing FQN, and a list of alternative FQNs sharing the same `modulePath`/`name` at other versions materialized on the platform, ordered by the total contract-key comparator (see Alternatives Ordering). `Match` SHALL accumulate every miss in one pass and expose them on the `MatchPlan`. The `MissingFQN` record is retained for compatibility; the load-bearing diagnosis is the unresolved-demand set.

#### Scenario: Alternatives surfaced

- **WHEN** a component demands `<path>/<name>@v9` which is absent, but the platform materialized `<path>/<name>@v1`
- **THEN** the `MissingFQN.Alternatives` for that miss contains `<path>/<name>@v1`

#### Scenario: Multiple misses accumulated

- **WHEN** two components each demand a different absent FQN
- **THEN** the `MatchPlan` carries two `MissingFQN` entries, one per `(instance, component, fqn)`

## REMOVED Requirements

### Requirement: Defensive Ambiguity Handling

**Reason**: superseded by 0010 D32/D37 — multiple satisfied candidates for a catalog-fulfilled contract legitimately all pair (catalog buckets measured holding 3–8 transformers); exclusivity is a property only of provider-fulfilled contracts and is enforced structurally at materialize (single-provider guard), not defensively at pairing.
