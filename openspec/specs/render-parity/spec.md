# render-parity Specification

## Purpose
The differential contract between the kernel's render path and pure-CUE unification of the same inputs: which inputs a parity case names, what it means for the two rendered values to agree, how a known divergence is recorded and retired, and the rule that a divergence is closed by removing kernel behaviour rather than by weakening the comparison.
## Requirements

### Requirement: Pure-CUE unification is the render oracle

The test suite SHALL include a differential parity harness that renders each parity case twice, once through `Kernel.Render` and once by plain CUE unification of the transformer's `#transform` with its declared inputs in a single CUE build, and SHALL treat the CUE result as the reference. When the two disagree the harness SHALL report the kernel as the defective side, naming the case, the (component, transformer) pair, and the first differing path. The harness is the standing tripwire against any Go-side transformation of a component, instance or context value being reintroduced.

#### Scenario: Agreement on a shipped transformer

- **WHEN** a case pairs a fixture component with a published `catalogs/opm` transformer
- **THEN** `Render`'s object for that pair and the oracle's `output` for the same pair compare equal, and the case passes

#### Scenario: Disagreement is attributed to the kernel

- **WHEN** `Render`'s value for a pair differs from the oracle's at any path
- **THEN** the failure message names the case, the pair, and the first differing path, and describes the difference as a kernel divergence from unification

### Requirement: Both renderers resolve identical inputs

Every parity case SHALL name its instance, component, and transformer by fixture, and the harness SHALL supply the same fixture bytes and the same dependency pins to both renderers. The instance and the platform SHALL enter the kernel side as source-carrying artifacts from the same on-disk packages the oracle build imports; the harness MUST NOT construct either kernel-side input by extracting and re-filling values from a separately compiled parent.

#### Scenario: Instance enters both sides by import

- **WHEN** a case is rendered
- **THEN** the kernel stages the instance and platform packages the oracle build imports, and both resolve `core` and the catalog from the same pinned versions

#### Scenario: Divergent inputs are refused

- **WHEN** a case would require the kernel side and the oracle side to construct an input differently
- **THEN** the case is not admitted to the table; the harness carries no per-case input-construction override

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

### Requirement: Matched pair sets agree

For each case the harness SHALL compare the set of (component, transformer) pairs `Render` reports against the pairs the oracle's predicate matching produced, and SHALL require them equal. No exemption exists: the always-unify rung is plain unification on both sides.

#### Scenario: Pair sets agree on the fixtures

- **WHEN** the shipped fixtures are matched on both sides
- **THEN** the two pair sets are equal

### Requirement: The harness changes no kernel behaviour

Adding the harness SHALL NOT alter any value the kernel renders, any error it returns, or any public symbol under `opm/`. The harness SHALL live in test code and test fixtures only.

#### Scenario: Existing suite unchanged

- **WHEN** the harness is added
- **THEN** every previously passing test passes with identical output and no `opm/` package gains or changes an exported identifier

### Requirement: The old path was proven equal to Render before deletion

Before the old render path was removed, the harness SHALL have rendered every parity case through both `Compile` (against the subscription-shaped platform on the prior core pin) and `Render` (against a D5-shaped platform importing the same published catalog bytes) and asserted the rendered values equal per case, order-sensitively, with any ordering-only difference recorded per case. The proof lives in the archive as the record; after deletion the harness compares `Render` against the oracle only.

#### Scenario: Proof gated the deletion

- **WHEN** a reader inspects this change's archive
- **THEN** the old-versus-new comparison is recorded as having passed on every parity case at the commit that deleted the old path

### Requirement: Equality is structural and order-sensitive

Each case SHALL compare the whole rendered value (`structural`). Comparison SHALL be order-sensitive: two values whose fields or list elements appear in different orders SHALL compare unequal. The harness MUST NOT sort, canonicalise, or otherwise reorder either value before comparing. The `output-fields-only` mode is retired: with the context projected by core (D12) there is no runtime-built projection to exclude, and the harness SHALL refuse a table entry that declares it.

#### Scenario: Reordered output fails

- **WHEN** the kernel emits the same fields as the oracle for a pair but in a different order
- **THEN** the case fails, naming the first path whose order differs

#### Scenario: Retired mode is refused

- **WHEN** a case declares the retired `output-fields-only` mode
- **THEN** the harness refuses the table entry: `structural` is the only admitted mode
