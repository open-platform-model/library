## ADDED Requirements

### Requirement: Matching stays in the glue and derived rules move into core one at a time

The matching algorithm (which transformers a component is paired with, the demand buckets, and the verdicts the glue emits) SHALL stay in the library's render glue, as 0019:D10 and 0019:D17 decide. A single derived rule MAY move into core when core can compute it from core shapes alone. Each such move SHALL be one rule at a time, SHALL be an additive core release, and SHALL ship with a parity test proving that the core field and the derivation it replaces agree over the served fixtures. `#contracts.providedBy`, read by the glue and checked by `opm/kernel/render_inventory_parity_test.go`, is the precedent. ADR-012 records this.

#### Scenario: ADR-012 records where matching lives

- **WHEN** a developer reads `adr/012-matching-stays-in-the-library-glue.md`
- **THEN** it states that the matching algorithm stays in the library glue under 0019:D10 and 0019:D17, that single derived rules move into core one at a time with a parity test, and that `#contracts.providedBy` is the precedent
- **AND** it names moving `#Match` whole into core (core#62) and a core reverse index as rejected alternatives with reasons

## MODIFIED Requirements

### Requirement: Each render is its own build in its own context

`Render` SHALL create a fresh `cue.Context` for the render, evaluate the staged render module exactly once with it, and release its own references to it when `Render` returns; a `*kernel.Compiled` the caller holds keeps that build alive until the caller releases it. No built value SHALL be shared between renders, and the render SHALL NOT use any long-lived context. The staging directory SHALL hold only the generated render module (its `cue.mod` and glue); an overlay-mode input SHALL be served from memory and an on-disk input from its own directory, so nothing of either input is copied. The staging directory SHALL be removed when the render completes.

#### Scenario: Repeated renders share nothing

- **WHEN** `Render` is invoked twice with the same inputs
- **THEN** each invocation stages, builds and decodes independently, and the results are byte-identical

#### Scenario: The staging directory holds no input files

- **WHEN** a render of an overlay-mode instance against an overlay-mode platform is staged
- **THEN** the staging directory contains `cue.mod/module.cue`, `cue.mod/local-module.cue` and the glue file, and no `instance/` or `platform/` directory
