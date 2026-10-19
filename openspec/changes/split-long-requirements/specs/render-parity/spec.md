## MODIFIED Requirements

### Requirement: Pure-CUE unification is the render oracle
The test suite SHALL include a differential parity harness that renders each parity case twice, once through `Kernel.Render` and once by plain CUE unification of the transformer's `#transform` with its declared inputs in a single CUE build, and SHALL treat the CUE result as the reference. When the two disagree the harness SHALL report the kernel as the defective side, naming the case, the (component, transformer) pair, and the first differing path.

#### Scenario: Agreement on a shipped transformer

- **WHEN** a case pairs a fixture component with a published `catalogs/opm` transformer
- **THEN** `Render`'s object for that pair and the oracle's `output` for the same pair compare equal, and the case passes

#### Scenario: Disagreement is attributed to the kernel

- **WHEN** `Render`'s value for a pair differs from the oracle's at any path
- **THEN** the failure message names the case, the pair, and the first differing path, and describes the difference as a kernel divergence from unification

### Requirement: The shipped catalog stays discriminated
The shipped parity group SHALL assert that the inventory (Requirement: The discrimination read lives in a test function of its own beside the render cases) reports no comparable transformer pair and reads discriminated (enhancement 0015 ("Comparable predicates within one catalog-fulfilled bucket are refused, not arbitrated"); core derives the report and imports no catalog, so this is the one place the published catalog is checked against it).

#### Scenario: The published catalog is discriminated

- **WHEN** the shipped group runs against the `catalogs/opm` build the parity platform fixture pins
- **THEN** the inventory read off the acquired platform has an empty `Comparable` and `Discriminated` true, and the group's render cases are unaffected by the read

#### Scenario: A comparable pair in a catalog release fails the harness

- **WHEN** the pinned catalog build carries two transformers whose match predicates are comparable over a contract the catalog itself fulfils
- **THEN** the shipped group fails, and the failure names both transformer FQNs and the shared contract FQNs from the inventory row

## REMOVED Requirements

### Requirement: The old path was proven equal to Render before deletion
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The old render path was compared with Render on every parity case before it was removed" and "The proof of the comparison lives in the archive and after deletion the harness compares Render against the oracle only".

### Requirement: The shipped case table names the catalog build the parity module pins
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The shipped case table names each transformer by its catalog-relative name and the harness builds each id from the parity module's pin" and "The harness checks that the catalog build the oracle reports equals the pin".

## ADDED Requirements

### Requirement: The harness is the standing tripwire against a Go-side transformation being reintroduced
The harness SHALL be the standing tripwire against any Go-side transformation of a component, instance or context value being reintroduced.

#### Scenario: The harness catches a reintroduced Go-side transformation

- **WHEN** a Go-side transformation of a component, instance or context value is reintroduced
- **THEN** the harness is the standing tripwire against it

### Requirement: The old render path was compared with Render on every parity case before it was removed
Before the old render path was removed, the harness SHALL have rendered every parity case through both `Compile` (against the subscription-shaped platform on the prior core pin) and `Render` (against a platform module importing the same published catalog bytes, enhancement 0019) and asserted the rendered values equal per case, order-sensitively, with any ordering-only difference recorded per case.

#### Scenario: Both renderers were compared per case before the removal

- **WHEN** the old render path is about to be removed
- **THEN** the harness SHALL have rendered every parity case through both `Compile` and `Render` and asserted the rendered values equal per case, order-sensitively, with any ordering-only difference recorded per case

### Requirement: The proof of the comparison lives in the archive and after deletion the harness compares Render against the oracle only
The proof (Requirement: The old render path was compared with Render on every parity case before it was removed) SHALL live in the archive as the record; after deletion the harness SHALL compare `Render` against the oracle only.

#### Scenario: Proof gated the deletion

- **WHEN** a reader inspects this change's archive
- **THEN** the old-versus-new comparison is recorded as having passed on every parity case at the commit that deleted the old path

### Requirement: The discrimination read lives in a test function of its own beside the render cases
The shipped parity group SHALL, in a test function of its own beside the one rendering its cases, read the contract inventory off the platform fixture that imports the published `catalogs/opm` build both renderers resolve. The read SHALL NOT live inside the render-case function: a catalog release that breaks discrimination has to fail by name under `-run`, and a render-case divergence and a discrimination break must never mask each other.

#### Scenario: The inventory is read in a test function of its own

- **WHEN** the shipped parity group runs
- **THEN** the contract inventory is read off the platform fixture in a test function of its own beside the one rendering its cases, and the read does not live inside the render-case function

### Requirement: The discrimination assertion is gated as the shipped group is
The assertion (Requirement: The shipped catalog stays discriminated) SHALL be gated exactly as the shipped group is: skipped under `-short` and when the public registry is unreachable, and forced by the same variable CI sets.

#### Scenario: The discrimination assertion is skipped where the shipped group is skipped

- **WHEN** the public registry is unreachable or the tests run under `-short`
- **THEN** the assertion is skipped exactly as the shipped group is

### Requirement: A catalog build that ships comparable transformers fails the harness naming both transformers and the shared contracts
A published catalog build that ships two transformers whose predicates are comparable over a shared catalog-fulfilled contract SHALL fail the harness with a message naming both transformer FQNs and the shared contracts, so the break is attributed to the catalog release, not to a platform that later embeds it.

#### Scenario: A catalog build with two comparable transformers is attributed to the catalog release

- **WHEN** a published catalog build ships two transformers whose predicates are comparable over a shared catalog-fulfilled contract
- **THEN** the harness fails with a message naming both transformer FQNs and the shared contracts, so the break is attributed to the catalog release, not to a platform that later embeds it

### Requirement: The shipped case table names each transformer by its catalog-relative name and the harness builds each id from the parity module's pin
The shipped case table SHALL name each transformer by its catalog-relative name only. The harness SHALL build every transformer id from the `opmodel.dev/catalogs/opm` version the parity module's own `cue.mod/module.cue` pins. That module is the one the oracle builds in. No case row, helper or assertion in the parity harness SHALL spell a `catalogs/opm` version as a literal.

#### Scenario: A catalog bump edits only module files

- **WHEN** the catalog pin moves to a newer published build in the parity module, its platform module, and the other test modules that pin the catalog, and no other file changes
- **THEN** the shipped group resolves the new build on both sides, and every row is compared against the new build's transformer ids

#### Scenario: A pin naming no shipped transformer

- **WHEN** the pinned build no longer ships a transformer that a case row names
- **THEN** the row-coverage check fails, naming the row that is not a pair the oracle matches, as it does today

### Requirement: The harness checks that the catalog build the oracle reports equals the pin
The harness SHALL also check, before comparing any case, that the catalog build the oracle reports equals that pin (Requirement: The shipped case table names each transformer by its catalog-relative name and the harness builds each id from the parity module's pin); when the two differ, the failure SHALL name both versions and SHALL NOT report the difference as a kernel divergence.

#### Scenario: The oracle reports another build than the pin

- **WHEN** the catalog build the oracle reports differs from the parity module's pin
- **THEN** the harness fails before comparing any case, with a message naming the build the oracle reports and the pinned version
