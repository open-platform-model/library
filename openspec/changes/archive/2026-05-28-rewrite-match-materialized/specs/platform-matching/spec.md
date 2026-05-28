## MODIFIED Requirements

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

### Requirement: Lookup via Platform.#matchers

For each demanded FQN, the matcher SHALL look up the FQN in the materialized platform's `#matchers.resources[FQN]` (or `.traits[FQN]`) via the binding's `Paths().Matchers` constant. A demanded FQN whose bucket is empty SHALL produce a structured `MissingFQN` diagnostic rather than a soft warning.

#### Scenario: FQN present in matchers

- **WHEN** a demanded resource FQN exists in the materialized `#matchers.resources`
- **THEN** the matcher proceeds to the always-unify rung and predicate evaluation for each candidate transformer

#### Scenario: FQN absent

- **WHEN** a demanded resource FQN is absent from the materialized `#matchers.resources`
- **THEN** the matcher records one `MissingFQN` entry on the `MatchPlan` for that `(release, component, fqn)` triple
- **AND** the matcher continues processing the remaining demanded FQNs (no fail-fast)

## ADDED Requirements

### Requirement: Always-Unify Before Pairing

Before pairing a candidate transformer with a component, the matcher SHALL unify the component's primitive value with the transformer's required primitive value for every FQN present in BOTH `component.#resources` and `transformer.requiredResources` (and the analogous `#traits` / `requiredTraits` intersection), not only the FQN that triggered the bucket lookup. A unification failure SHALL prevent the pairing and SHALL be recorded as a `UnifyError`.

#### Scenario: Bodies agree

- **WHEN** a component's resource at `<fqn>` unifies cleanly with the transformer's `requiredResources[<fqn>]`
- **THEN** the matcher proceeds to predicate evaluation for that candidate

#### Scenario: Bodies diverge

- **WHEN** a component's resource at `<fqn>` conflicts with the transformer's `requiredResources[<fqn>]`
- **THEN** the matcher records a `UnifyError` for that `(component, fqn)`
- **AND** does not pair the candidate transformer

#### Scenario: Second required primitive diverges

- **WHEN** a transformer requires both `<fqn-A>` and `<fqn-B>`, and the component agrees on `<fqn-A>` but conflicts on `<fqn-B>`
- **THEN** a `UnifyError` is recorded for `<fqn-B>`
- **AND** the pairing is rejected

### Requirement: Structured Missing-FQN Diagnostic

A missing FQN SHALL be reported as a structured `MissingFQN` value carrying the release name, component name, the missing FQN, and a list of alternative FQNs sharing the same `modulePath`/`name` at other SemVers materialized on the platform. `Match` SHALL accumulate every miss in one pass and expose them on the `MatchPlan`.

#### Scenario: Alternatives surfaced

- **WHEN** a component demands `<path>/<name>@1.0.0` which is absent, but the platform materialized `<path>/<name>@1.1.0`
- **THEN** the `MissingFQN.Alternatives` for that miss contains `<path>/<name>@1.1.0`

#### Scenario: Multiple misses accumulated

- **WHEN** two components each demand a different absent FQN
- **THEN** the `MatchPlan` carries two `MissingFQN` entries, one per `(release, component, fqn)`

### Requirement: Structured Unify-Error Diagnostic

A unification failure SHALL be reported as a structured `UnifyError` carrying the component name, the FQN, and the underlying CUE error tree. The CUE diagnostic SHALL be surfaced verbatim with no Go-side reformatting, and SHALL be reachable via `errors.As` as a `cuelang.org/go/cue/errors.Error`.

#### Scenario: Verbatim CUE conflict

- **WHEN** a unify failure occurs
- **THEN** the `UnifyError.Cause` is the CUE error reporting `conflicting values` with `file:line` positions, unmodified
- **AND** it is reachable via `errors.As` for `cuelang.org/go/cue/errors.Error`
