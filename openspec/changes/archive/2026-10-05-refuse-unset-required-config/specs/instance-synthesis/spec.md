## MODIFIED Requirements

### Requirement: Values field is caller-supplied with no implicit fallback

`SynthesizeInstance` SHALL NOT consult `Module.debugValues` or any other implicit source when `InstanceInput.Values` is empty. When `Values` holds one or more sources, the kernel SHALL unify them in stack order, render the result into the synthesized package's values source (via `format.Node` on the value's syntax, never string-interpolating raw input) so it participates in the single build, and after the build check each source against the module's `#config` at the source's own positions. When `Values` is empty, the kernel SHALL omit the values source and enforce concreteness on the built spec as usual.

#### Scenario: Caller-supplied values participate in the build

- **WHEN** `SynthesizeInstance` is called with `Values` holding a concrete source satisfying the module's `#config`
- **THEN** the returned instance carries those values at the schema's values path
- **AND** the values entered the build as a rendered source file, not a post-build cross-build unification

#### Scenario: Layered values unify in order

- **WHEN** `SynthesizeInstance` is called with `Values == []Source{a, b}` where `b` sets a field `a` leaves open
- **THEN** the returned instance's values carry `a` unified with `b`

#### Scenario: Zero Values is not replaced by debugValues

- **WHEN** `SynthesizeInstance` is called with empty `Values` against a Module that defines `debugValues`
- **THEN** the build's values path is unfilled (does not equal `debugValues`) and the call fails on concreteness
