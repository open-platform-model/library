## ADDED Requirements

### Requirement: The shipped case table names the catalog build the parity module pins

The shipped case table SHALL name each transformer by its catalog-relative name only. The
harness SHALL build every transformer id from the `opmodel.dev/catalogs/opm` version the parity
module's own `cue.mod/module.cue` pins. That module is the one the oracle builds in. No case row,
helper or assertion in the parity harness SHALL spell a `catalogs/opm` version as a literal. The
harness SHALL also require the catalog build the oracle resolved to equal that pin. When the two
differ, the failure SHALL name both versions and say that resolution lifted the pin. It SHALL NOT
report the difference as a kernel divergence.

#### Scenario: A catalog bump edits only module files

- **WHEN** the catalog pin moves to a newer published build in the parity module, its platform module, and the other test modules that pin the catalog, and no other file changes
- **THEN** the shipped group resolves the new build on both sides, and every row is compared against the new build's transformer ids

#### Scenario: Resolution lifts the pin

- **WHEN** a dependency of the parity module requires a newer catalog build than the parity module pins, so the oracle resolves a build other than the pinned one
- **THEN** the harness fails before comparing any case, naming the pinned version and the resolved version

#### Scenario: A pin naming no shipped transformer

- **WHEN** the pinned build no longer ships a transformer that a case row names
- **THEN** the row-coverage check fails, naming the row that is not a pair the oracle matches, as it does today
