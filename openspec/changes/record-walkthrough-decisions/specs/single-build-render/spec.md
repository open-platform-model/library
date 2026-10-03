## ADDED Requirements

### Requirement: Matching stays in the glue and derived rules move into core one at a time

The matching algorithm (which transformers a component is paired with, the demand buckets, and the verdicts the glue emits) SHALL stay in the library's render glue, as 0019:D10 and 0019:D17 decide. A single derived rule MAY move into core when core can compute it from core shapes alone. Each such move SHALL be one rule at a time, SHALL be an additive core release, and SHALL ship with a parity test proving that the core field and the derivation it replaces agree over the served fixtures. `#contracts.providedBy`, read by the glue and checked by `opm/kernel/render_inventory_parity_test.go`, is the precedent. ADR-012 records this.

#### Scenario: ADR-012 records where matching lives

- **WHEN** a developer reads `adr/012-matching-stays-in-the-library-glue.md`
- **THEN** it states that the matching algorithm stays in the library glue under 0019:D10 and 0019:D17, that single derived rules move into core one at a time with a parity test, and that `#contracts.providedBy` is the precedent
- **AND** it names moving `#Match` whole into core (core#62) and a core reverse index as rejected alternatives with reasons
