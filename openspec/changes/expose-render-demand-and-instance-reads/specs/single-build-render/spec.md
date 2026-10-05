## ADDED Requirements

### Requirement: A render reports every contract its instance requires

The render SHALL report the instance's contract demand as data on `RenderDiagnostics.RequiredContracts`. The field holds every contract key that a component of the render requires: the keys of each component's `#resources`, plus the keys of its `#traits` when it attaches any. The keys are sorted in byte order with duplicates removed. Source: 0013:D24.

The demand SHALL be computed inside the build, from the same component set the matcher iterates. A component that is omitted under `RenderInput.SkipUnprovided` still counts. The list SHALL NOT be narrowed to provider-fulfilled contracts or to contracts the platform implements. The list SHALL be an empty, non-nil slice for an instance with no components.

The field SHALL be set on every `RenderResult` and on every `*RenderError`. A render that returns a plain error carries no demand. That covers a refusal before evaluation and a build or diagnostics-decode failure.

Computing the demand SHALL fail closed. A component's `#resources` is read with no presence test, so a component whose `#resources` is missing or does not evaluate makes the render return an error and no result. It never reads as a component with no demand. An absent `#traits` contributes nothing. `#traits` is read with the same presence test the matcher uses, so the demand covers exactly the traits matching sees.

#### Scenario: Demand on a successful render

- **WHEN** an instance renders whose components declare resources and traits, and two components declare the same resource key
- **THEN** `RenderResult.Diagnostics.RequiredContracts` lists every resource and trait key once, sorted

#### Scenario: A component without traits

- **WHEN** an instance renders whose component attaches no `#traits`
- **THEN** that component's resource keys are listed and the render does not fail on the absent traits

#### Scenario: No components

- **WHEN** an instance with no components renders
- **THEN** the render succeeds and `RequiredContracts` is an empty, non-nil slice

#### Scenario: Demand on a refusal

- **WHEN** the gate refuses a render because a resource demand is unresolved
- **THEN** the returned `*RenderError`'s `Diagnostics.RequiredContracts` lists every contract key the instance's components declare, the unresolved one included

#### Scenario: An omitted component's demand is reported

- **WHEN** `SkipUnprovided` is set and a component is omitted for an unprovided resource demand
- **THEN** that component's resource and trait keys are on `RequiredContracts`, although it renders nothing

#### Scenario: A component without resources refuses

- **WHEN** a render's instance package has a component that carries no `#resources`
- **THEN** `Render` returns an error and no result, and does not report the component as one with no demand

#### Scenario: The demand equals the declared contracts of the instance

- **WHEN** any served render fixture renders and the build reaches its diagnostics
- **THEN** `RequiredContracts` equals the sorted, deduplicated `#resources` and `#traits` keys of the instance package's own components
