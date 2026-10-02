## ADDED Requirements

### Requirement: The shipped case table names the catalog build the parity module pins

The shipped case table SHALL name each transformer by its catalog-relative name only. The
harness SHALL build every transformer id from the `opmodel.dev/catalogs/opm` version the parity
module's own `cue.mod/module.cue` pins. That module is the one the oracle builds in. No case row,
helper or assertion in the parity harness SHALL spell a `catalogs/opm` version as a literal. The
harness SHALL also check, before comparing any case, that the catalog build the oracle reports
equals that pin; when the two differ, the failure SHALL name both versions and SHALL NOT report
the difference as a kernel divergence.

#### Scenario: A catalog bump edits only module files

- **WHEN** the catalog pin moves to a newer published build in the parity module, its platform module, and the other test modules that pin the catalog, and no other file changes
- **THEN** the shipped group resolves the new build on both sides, and every row is compared against the new build's transformer ids

#### Scenario: The oracle reports another build than the pin

- **WHEN** the catalog build the oracle reports differs from the parity module's pin
- **THEN** the harness fails before comparing any case, with a message naming the build the oracle reports and the pinned version

#### Scenario: A pin naming no shipped transformer

- **WHEN** the pinned build no longer ships a transformer that a case row names
- **THEN** the row-coverage check fails, naming the row that is not a pair the oracle matches, as it does today
