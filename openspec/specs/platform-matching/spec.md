# platform-matching Specification

## Purpose
TBD - created by syncing change rewrite-match-around-platform. Update Purpose after archive.

## Requirements

### Requirement: Match Phase Consumes Platform

The Match phase SHALL consume `*platform.Platform` exclusively. The kernel SHALL NOT accept `*provider.Provider` as a matcher input on any public method.

#### Scenario: Match against Platform

- **WHEN** a caller invokes `k.Match(ctx, MatchInput{Module, ModuleRelease, Platform})`
- **THEN** the matcher walks the consumer Module's `#components` and looks up each demanded FQN in `Platform.#matchers` via the binding paths
- **AND** returns a `*MatchPlan` describing matched pairs, unmatched FQNs, and ambiguous FQNs

#### Scenario: Provider field removed

- **WHEN** a developer reads `MatchInput`, `PlanInput`, or `CompileInput`
- **THEN** none of these structs has a `Provider` field
- **AND** the `Platform` field is non-optional (no godoc continues to mark it optional)

### Requirement: Demand Walking from Components

The matcher SHALL collect required Resource and Trait FQNs by walking each component's `#resources` and `#traits` maps.

#### Scenario: Component with required resources

- **WHEN** a consumer Module has a component with `#resources: { "<fqn-A>": ..., "<fqn-B>": ... }`
- **THEN** the demand for that component includes resource FQNs `<fqn-A>` and `<fqn-B>`

#### Scenario: Component with required traits

- **WHEN** a consumer Module has a component with `#traits: { "<fqn-X>": ... }`
- **THEN** the demand for that component includes trait FQN `<fqn-X>`

### Requirement: Lookup via Platform.#matchers

For each demanded FQN, the matcher SHALL look up `Platform.#matchers.resources[FQN]` (or `.traits[FQN]`) via the binding's `Paths().Matchers` constant.

#### Scenario: FQN present in matchers

- **WHEN** a demanded resource FQN exists in `Platform.#matchers.resources`
- **THEN** the matcher pairs the demanding component with the transformer at that FQN
- **AND** the pair appears in `MatchPlan.MatchedPairs()`

#### Scenario: FQN absent

- **WHEN** a demanded resource FQN is absent from `Platform.#matchers.resources`
- **THEN** the FQN appears in `MatchPlan.NonMatchedPairs()` (or equivalent unmatched accessor)
- **AND** an explanatory warning describes which component demanded the missing FQN

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
- **AND** proceeds with FillPath / decode / emit Rendered as before

### Requirement: Provider Package Retired

The `pkg/provider/` package SHALL be removed in this slice. The `LoadProvider` loader (in `pkg/helper/loader/file/provider.go`) and its deprecation shim at `pkg/loader/LoadProvider` SHALL also be removed.

#### Scenario: pkg/provider absent

- **WHEN** a developer searches the repository for `pkg/provider`
- **THEN** no directory or package exists at that path

#### Scenario: LoadProvider absent

- **WHEN** a developer searches for `LoadProvider`
- **THEN** the symbol exists in no `pkg/` package
- **AND** the deprecation shim previously at `pkg/loader/` is removed

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
