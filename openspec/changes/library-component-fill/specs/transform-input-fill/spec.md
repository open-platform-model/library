## Purpose

What the runtime owes each declared `#transform` input when it fills it: the value a transformer receives is the value plain CUE unification would give it, with no field class removed in transit.

## ADDED Requirements

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
- **THEN** its rendered output is byte-identical to the output before this change, for every shipped catalog transformer

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
