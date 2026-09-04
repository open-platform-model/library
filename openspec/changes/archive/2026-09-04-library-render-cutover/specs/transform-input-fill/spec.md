## MODIFIED Requirements

### Requirement: `#component` is filled with every field class preserved

When the render build executes a matched (component, transformer) pair, `#transform.#component` SHALL be the instance's component value as evaluated inside the same build, preserving regular fields, definition fields (including `#names`, `#resources`, `#traits`, `#blueprints` and `#instance`), hidden fields and constraints. The kernel MUST NOT export, finalize, fill across a build boundary, or otherwise re-materialise the component; the binding is plain unification in the generated glue.

#### Scenario: A transformer reads a computed name

- **WHEN** a transformer's `#transform` re-declares `#component: _` and its `output` reads `#component.#names.dns.fqdn`
- **THEN** the pair renders concretely and the rendered value carries the fqdn core computed for that component

#### Scenario: Parity with unification on the names probe

- **WHEN** the render-parity harness renders the `names-probe :: web` case
- **THEN** `Render` and the pure-CUE oracle agree, and the case carries no expected divergence

#### Scenario: Shipped transformers are unaffected

- **WHEN** a transformer reads no definition field
- **THEN** its rendered output through `Render` is byte-identical to the oracle's for every shipped catalog transformer; the parity harness classifies every shipped pair as agreeing

### Requirement: Matching and execution read one components value

The render build SHALL read the same components value for matching and for execution: both are comprehensions over the imported instance's `components` inside one build. No second, narrowed components value SHALL exist.

#### Scenario: One value through the pipeline

- **WHEN** `Kernel.Render` runs
- **THEN** the value the matching glue reads and the value each pair's `#component` binds to are the same evaluated value

### Requirement: `#moduleInstance` is filled with the whole evaluated instance

When the render build executes a matched pair, `#transform.#moduleInstance` SHALL be the imported instance's `#ModuleInstance` value as evaluated: its metadata, values, module reference and every component, including components other than the one bound to `#component`. The kernel MUST NOT narrow the value to metadata, mask sibling components, or otherwise hand over less than plain CUE unification would.

#### Scenario: A transformer reads instance metadata

- **WHEN** a transformer's `#transform` re-declares `#moduleInstance: _` and its `output` reads `#moduleInstance.metadata.name` and `#moduleInstance.metadata.namespace`
- **THEN** the pair renders concretely and the rendered value carries the instance's name and namespace

#### Scenario: A transformer reads its own component through the instance

- **WHEN** a transformer reads `#moduleInstance.components[<name>]` where `<name>` is the component bound to `#component` for the same pair
- **THEN** the pair renders concretely, with no cycle or structural error, and the value read through the instance equals the value bound to `#component`

#### Scenario: Parity with unification on the instance probe

- **WHEN** the render-parity harness renders the `instance-probe :: web` case
- **THEN** `Render` and the pure-CUE oracle agree, and the case carries no expected divergence

#### Scenario: Shipped transformers are unaffected

- **WHEN** a transformer does not read `#moduleInstance`
- **THEN** its rendered output through `Render` is unchanged for every shipped catalog transformer; the parity harness classifies every shipped pair as agreeing

## ADDED Requirements

### Requirement: `#context` is projected by core; the kernel supplies `#runtimeName` only

The kernel SHALL supply exactly one context value into the build, `#context.#runtimeName`, as a formatted literal in the generated glue. Every other `#TransformerContext` field SHALL come from core's projection of `#moduleInstance` and `#component` (core `v2.0.0-alpha.7`, 0019 D12). The kernel SHALL carry no Go-side context builder.

#### Scenario: No Go context mirror

- **WHEN** a consumer inspects `opm/schema`
- **THEN** no `BuildTransformerContext` or context-data types exist

#### Scenario: Context values match the projection

- **WHEN** a transformer reads `#context.#moduleInstanceMetadata.namespace`
- **THEN** the value equals `#moduleInstance.metadata.namespace` of the imported instance, with nothing filled by the kernel
