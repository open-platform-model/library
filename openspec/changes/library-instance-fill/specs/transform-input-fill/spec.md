## ADDED Requirements

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
