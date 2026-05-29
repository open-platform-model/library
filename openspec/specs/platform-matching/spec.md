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

### Requirement: Defensive Ambiguity Handling

If a `Platform.#matchers.resources[FQN]` or `Platform.#matchers.traits[FQN]` lookup returns more than one candidate (which catalog 014 D13 forbids at the platform layer), the matcher SHALL flag the FQN as ambiguous and not pair the component with any candidate.

#### Scenario: Multi-candidate FQN

- **WHEN** a Platform somehow produces a list of two or more candidates for a single FQN
- **THEN** the matcher does not select a winner
- **AND** the FQN appears as ambiguous in the `MatchPlan`
- **AND** an error or warning explains the violation of catalog 014 D13

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

