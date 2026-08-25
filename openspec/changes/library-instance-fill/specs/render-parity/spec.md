## MODIFIED Requirements

### Requirement: Probe transformers expose definition and instance inputs

The parity fixtures SHALL include transformers that read `#component.#names` and `#moduleInstance` respectively, so that whether the kernel supplies those inputs is observable in rendered output. Each probe case SHALL pin the value the oracle renders for the probed input, and SHALL carry no expected divergence: the kernel supplies both inputs, and a case that stops agreeing fails the harness.

#### Scenario: Names probe under the oracle

- **WHEN** the names probe is unified with a component whose `#names.dns.fqdn` is computed
- **THEN** the oracle renders an object carrying that fqdn concretely

#### Scenario: Instance probe under the kernel today

- **WHEN** the instance probe is rendered through the kernel and through the oracle
- **THEN** both render an object carrying the instance's name and namespace concretely, and the harness reports the case as agreeing
