# platform-matching Specification

## Purpose
TBD - created by syncing change rewrite-match-around-platform. Update Purpose after archive.
## Requirements
### Requirement: Match Phase Consumes Platform

The Match phase SHALL consume a `*materialize.MaterializedPlatform` (the realized form of `*platform.Platform`) exclusively. The kernel SHALL NOT accept a raw `*platform.Platform` or `*provider.Provider` as a matcher input on any public method; callers MUST `Materialize` the platform first.

#### Scenario: Match against MaterializedPlatform

- **WHEN** a caller invokes `k.Match(ctx, MatchInput{ModuleRelease, Platform})` where `Platform` is a `*materialize.MaterializedPlatform`
- **THEN** the matcher walks the consumer Module's `#components` and looks up each demanded FQN in the materialized `#matchers` via the binding paths
- **AND** returns a `*MatchPlan` describing matched pairs, structured missing-FQN diagnostics, and unify failures

#### Scenario: Raw Platform not accepted

- **WHEN** a developer reads `MatchInput`, `PlanInput`, or `CompileInput`
- **THEN** the `Platform` field type is `*materialize.MaterializedPlatform`, not `*platform.Platform`
- **AND** none of these structs has a `Provider` field

### Requirement: Demand Walking from Components

The matcher SHALL collect required Resource and Trait FQNs by walking each component's `#resources` and `#traits` maps.

#### Scenario: Component with required resources

- **WHEN** a consumer Module has a component with `#resources: { "<fqn-A>": ..., "<fqn-B>": ... }`
- **THEN** the demand for that component includes resource FQNs `<fqn-A>` and `<fqn-B>`

#### Scenario: Component with required traits

- **WHEN** a consumer Module has a component with `#traits: { "<fqn-X>": ... }`
- **THEN** the demand for that component includes trait FQN `<fqn-X>`

### Requirement: Lookup via Platform.#matchers

For each demanded FQN, the matcher SHALL look up the FQN in the materialized platform's `#matchers.resources[FQN]` (or `.traits[FQN]`) via the binding's `Paths().Matchers` constant. A demanded FQN whose bucket is empty SHALL produce a structured `MissingFQN` diagnostic rather than a soft warning.

#### Scenario: FQN present in matchers

- **WHEN** a demanded resource FQN exists in the materialized `#matchers.resources`
- **THEN** the matcher proceeds to the always-unify rung and predicate evaluation for each candidate transformer

#### Scenario: FQN absent

- **WHEN** a demanded resource FQN is absent from the materialized `#matchers.resources`
- **THEN** the matcher records one `MissingFQN` entry on the `MatchPlan` for that `(release, component, fqn)` triple
- **AND** the matcher continues processing the remaining demanded FQNs (no fail-fast)

### Requirement: Execute Resolves Transformers by FQN

The Execute phase SHALL resolve each matched pair's transformer by looking up the transformer's FQN in `Platform.#composedTransformers` via the binding's `Paths().ComposedTransformers` constant.

#### Scenario: Transformer body fetched by FQN

- **WHEN** Execute processes a matched `(component, transformerFQN)` pair
- **THEN** it fetches `Platform.#composedTransformers[transformerFQN]` to obtain the transformer's `#transform` body
- **AND** proceeds with FillPath / decode / emit Compiled as before

### Requirement: Provider Package Retired

The `opm/provider/` package SHALL be removed in this slice. The `LoadProvider` loader (in `opm/helper/loader/file/provider.go`) and its deprecation shim at `opm/loader/LoadProvider` SHALL also be removed.

#### Scenario: opm/provider absent

- **WHEN** a developer searches the repository for `opm/provider`
- **THEN** no directory or package exists at that path

#### Scenario: LoadProvider absent

- **WHEN** a developer searches for `LoadProvider`
- **THEN** the symbol exists in no `opm/` package
- **AND** the deprecation shim previously at `opm/loader/` is removed

### Requirement: render.Module Runtime Helper Updated

The runtime helper `render.Module` SHALL take `*platform.Platform` instead of `*provider.Provider`. `render.NewModule` SHALL be updated accordingly.

#### Scenario: NewModule signature

- **WHEN** a developer reads `render.NewModule`
- **THEN** the function signature is `func NewModule(plat *platform.Platform, runtimeName string) *Module`
- **AND** internal `Module` fields reference `Platform`, not `Provider`

### Requirement: Test Fixture Migration

Every test fixture that constructed `*provider.Provider` SHALL be migrated to construct `*platform.Platform` with a `#registry` carrying the previously-implicit Module's transformers. Behavior on each fixture SHALL be preserved (byte-equal output) for the single-fulfiller cases that constitute every existing fixture.

#### Scenario: Fixture parity

- **WHEN** the test suite runs the migrated fixtures
- **THEN** the rendered output for each fixture is byte-equal to the pre-slice-09 output
- **AND** any deviation is investigated, not silently accepted

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

### Requirement: Structured Unify-Error Diagnostic

A unification failure SHALL be reported as a structured `UnifyError` carrying the component name, the FQN, and the underlying CUE error tree. The CUE diagnostic SHALL be surfaced verbatim with no Go-side reformatting, and SHALL be reachable via `errors.As` as a `cuelang.org/go/cue/errors.Error`.

#### Scenario: Verbatim CUE conflict

- **WHEN** a unify failure occurs
- **THEN** the `UnifyError.Cause` is the CUE error reporting `conflicting values` with `file:line` positions, unmodified
- **AND** it is reachable via `errors.As` for `cuelang.org/go/cue/errors.Error`

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
