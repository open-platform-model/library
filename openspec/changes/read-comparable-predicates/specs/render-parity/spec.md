## ADDED Requirements

### Requirement: The shipped catalog stays discriminated

The shipped parity group SHALL, beside rendering its cases, read the contract inventory off the platform fixture that imports the published `catalogs/opm` build both renderers resolve, and SHALL assert that the inventory reports no comparable transformer pair and reads discriminated (enhancement 0015 D5; core derives the report and imports no catalog, so this is the one place the published catalog is checked against it). The assertion SHALL be gated exactly as the shipped group is: skipped under `-short` and when the public registry is unreachable, and forced by the same variable CI sets. A published catalog build that ships two transformers whose predicates are comparable over a shared catalog-fulfilled contract SHALL fail the harness with a message naming both transformer FQNs and the shared contracts, so the break is attributed to the catalog release, not to a platform that later embeds it.

#### Scenario: The published catalog is discriminated

- **WHEN** the shipped group runs against the `catalogs/opm` build the parity platform fixture pins
- **THEN** the inventory read off the acquired platform has an empty `Comparable` and `Discriminated` true, and the group's render cases are unaffected by the read

#### Scenario: A comparable pair in a catalog release fails the harness

- **WHEN** the pinned catalog build carries two transformers whose match predicates are comparable over a contract the catalog itself fulfils
- **THEN** the shipped group fails, and the failure names both transformer FQNs and the shared contract FQNs from the inventory row
