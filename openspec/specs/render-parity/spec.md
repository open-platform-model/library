# render-parity Specification

## Purpose
The differential contract between the kernel's render path and pure-CUE unification of the same inputs: which inputs a parity case names, what it means for the two rendered values to agree, how a known divergence is recorded and retired, and the rule that a divergence is closed by removing kernel behaviour rather than by weakening the comparison.
## Requirements
### Requirement: Pure-CUE unification is the render oracle

The test suite SHALL include a differential parity harness that renders each parity case twice, once through the kernel's compile path and once by plain CUE unification of the transformer's `#transform` with its declared inputs in a single CUE build, and SHALL treat the CUE result as the reference. When the two disagree the harness SHALL report the kernel as the defective side, naming the case, the (component, transformer) pair, and the first differing path.

#### Scenario: Agreement on a shipped transformer

- **WHEN** a case pairs a fixture component with a published `catalogs/opm` transformer that reads no definition field
- **THEN** the kernel's rendered objects for that pair and the oracle's `output` for the same pair compare equal, and the case passes

#### Scenario: Disagreement is attributed to the kernel

- **WHEN** the kernel's rendered value for a pair differs from the oracle's at any path
- **THEN** the failure message names the case, the pair, and the first differing path, and describes the difference as a kernel divergence from unification

### Requirement: Both renderers resolve identical inputs

Every parity case SHALL name its instance, component, and transformer by fixture, and the harness SHALL supply the same fixture bytes and the same dependency pins to both renderers. The instance fixture SHALL name its module by import so that cross-references inside the instance are bound identically on both sides; the harness MUST NOT construct the kernel-side instance by extracting and re-filling values from a separately compiled parent.

#### Scenario: Instance enters both sides by import

- **WHEN** a case is rendered
- **THEN** the kernel loads the instance from the same on-disk package the oracle build imports, and both resolve `core` and the catalog from the same pinned versions

#### Scenario: Divergent inputs are refused

- **WHEN** a case would require the kernel side and the oracle side to construct an input differently
- **THEN** the case is not admitted to the table; the harness carries no per-case input-construction override

### Requirement: Equality is stated per case and is order-sensitive

Each case SHALL declare its equality mode as either `structural` (the whole rendered value) or `output-fields-only` (the transformer's `output` value, excluding the `#context` projection). Comparison SHALL be order-sensitive: two values whose fields or list elements appear in different orders SHALL compare unequal. The harness MUST NOT sort, canonicalise, or otherwise reorder either value before comparing.

#### Scenario: Reordered output fails

- **WHEN** the kernel emits the same fields as the oracle for a pair but in a different order
- **THEN** the case fails, naming the first path whose order differs

#### Scenario: Interim mode excludes only the context projection

- **WHEN** a case declares `output-fields-only`
- **THEN** the comparison covers the entire `output` value of the pair and excludes nothing else

### Requirement: Known divergences are recorded and asserted to reproduce

A case MAY carry an expected divergence naming the kernel behaviour that causes it. A case with an expected divergence SHALL be asserted to diverge in the named way; the suite SHALL fail when such a case unexpectedly agrees, so that a change closing the divergence must also retire the entry. A case without an expected divergence SHALL be asserted to agree.

#### Scenario: Expected divergence reproduces

- **WHEN** a case expecting divergence because the kernel strips definition fields is rendered, and the kernel side fails or differs as named
- **THEN** the case passes and the divergence remains recorded

#### Scenario: Expected divergence stops reproducing

- **WHEN** a kernel change makes a case with an expected divergence agree with the oracle
- **THEN** the case fails with a message directing the author to remove the expected-divergence entry

### Requirement: Probe transformers expose definition and instance inputs

The parity fixtures SHALL include transformers that read `#component.#names` and `#moduleInstance` respectively, so that whether the kernel supplies those inputs is observable in rendered output. Each probe case SHALL pin the value the oracle renders for the probed input, and SHALL carry no expected divergence: the kernel supplies both inputs, and a case that stops agreeing fails the harness.

#### Scenario: Names probe under the oracle

- **WHEN** the names probe is unified with a component whose `#names.dns.fqdn` is computed
- **THEN** the oracle renders an object carrying that fqdn concretely

#### Scenario: Instance probe agrees

- **WHEN** the instance probe is rendered through the kernel and through the oracle
- **THEN** both render an object carrying the instance's name and namespace concretely, and the harness reports the case as agreeing

### Requirement: Matched pair sets agree, with one stated exemption

For each case the harness SHALL compare the set of (component, transformer) pairs the kernel matched against the pairs the oracle's predicate matching produced, and SHALL require them equal except where the kernel's always-unify rung disqualified a candidate on a body conflict, which the harness SHALL record as the single named exemption rather than as agreement.

#### Scenario: Pair sets agree on the fixtures

- **WHEN** the shipped fixtures are matched on both sides
- **THEN** the two pair sets are equal and no exemption is exercised

### Requirement: The harness changes no kernel behaviour

Adding the harness SHALL NOT alter any value the kernel renders, any error it returns, or any public symbol under `opm/`. The harness SHALL live in test code and test fixtures only.

#### Scenario: Existing suite unchanged

- **WHEN** the harness is added
- **THEN** every previously passing test passes with identical output and no `opm/` package gains or changes an exported identifier
