# transform-input-fill Specification

## Purpose
What the runtime owes each declared `#transform` input when it fills it: the value a transformer receives is the value plain CUE unification would give it, with no field class removed in transit.
## Requirements
### Requirement: `#component` is filled with every field class preserved

When the kernel executes a matched (component, transformer) pair, it SHALL fill `#transform.#component` with the instance's component value as evaluated, preserving regular fields, definition fields (including `#names`, `#resources`, `#traits`, `#blueprints` and `#instance`), hidden fields and constraints. It MUST NOT export, finalize, or otherwise re-materialise the component before filling it.

#### Scenario: A transformer reads a computed name

- **WHEN** a transformer's `#transform` re-declares `#component: _` and its `output` reads `#component.#names.dns.fqdn`
- **THEN** the pair renders concretely and the rendered value carries the fqdn core computed for that component

#### Scenario: Parity with unification on the names probe

- **WHEN** the render-parity harness renders the `names-probe :: web` case
- **THEN** the kernel side and the pure-CUE oracle agree, and the case carries no expected divergence

#### Scenario: Shipped transformers are unaffected

- **WHEN** a transformer reads no definition field
- **THEN** its rendered output is identical to the output before this change modulo CUE natural field order (0019 D14), for every shipped catalog transformer; the parity harness classifies every shipped pair as agreeing or ordering-only, never as differing in value

### Requirement: Matching and execution read one components value

The compile pipeline SHALL read the same components value for matching and for execution. It MUST NOT derive a second, narrowed components value for execution.

#### Scenario: One value through the pipeline

- **WHEN** `Kernel.Compile` runs
- **THEN** the value `Match` reads and the value each pair's `#component` is filled from are the same evaluated value

### Requirement: The instance fixture resolves `#instance`

The kernel flow fixture SHALL author its `#ModuleInstance` as a CUE package that names its module by import, so that every component's `#instance` and `#names` resolve. The fixture MUST NOT construct the instance by extracting `#components` from a module value and filling it into a separately compiled skeleton.

#### Scenario: fqdn resolves inside the flow fixture

- **WHEN** the flow fixture's instance is processed
- **THEN** `components.web.#names.dns.fqdn` evaluates to `web.default.svc.cluster.local` with no `required field missing` error

#### Scenario: uuid is derived, not authored

- **WHEN** the flow fixture's instance is processed
- **THEN** `metadata.uuid` is the value core derives from the instance fqn, identical to the value the parity oracle derives for the same name and namespace

### Requirement: `#moduleInstance` is filled with the whole evaluated instance

When the kernel executes a matched (component, transformer) pair, it SHALL fill `#transform.#moduleInstance` with the instance's `#ModuleInstance` value as evaluated: its metadata, values, module reference and every component, including components other than the one filled into `#component`. It MUST NOT narrow the value to metadata, mask sibling components, or otherwise hand over less than plain CUE unification would.

#### Scenario: A transformer reads instance metadata

- **WHEN** a transformer's `#transform` re-declares `#moduleInstance: _` and its `output` reads `#moduleInstance.metadata.name` and `#moduleInstance.metadata.namespace`
- **THEN** the pair renders concretely and the rendered value carries the instance's name and namespace

#### Scenario: A transformer reads its own component through the instance

- **WHEN** a transformer reads `#moduleInstance.components[<name>]` where `<name>` is the component filled into `#component` for the same pair
- **THEN** the pair renders concretely, with no cycle or structural error, and the value read through the instance equals the value filled into `#component`

#### Scenario: Parity with unification on the instance probe

- **WHEN** the render-parity harness renders the `instance-probe :: web` case
- **THEN** the kernel side and the pure-CUE oracle agree, and the case carries no expected divergence

#### Scenario: Shipped transformers are unaffected

- **WHEN** a transformer does not read `#moduleInstance`
- **THEN** its rendered output is unchanged for every shipped catalog transformer; the parity harness classifies every shipped pair as agreeing or ordering-only, never as differing in value
