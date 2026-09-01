# platform-matching Delta

## MODIFIED Requirements

### Requirement: Lookup via Platform.#matchers

For each demanded FQN, the matcher SHALL look up the FQN in the materialized platform's `#matchers.resources[FQN]` (or `.traits[FQN]`) via the binding's `Paths().Matchers` constant. A demanded FQN whose bucket is empty SHALL be carried by the unresolved-demand set rather than a soft warning: for a resource demand (and for a load-bearing trait demand) the matcher records an `UnresolvedDemand` with empty `Disqualified` and the same-base alternatives; an effectively-optional trait demand degrades to the unhandled-trait warning.

#### Scenario: FQN present in matchers

- **WHEN** a demanded resource FQN exists in the materialized `#matchers.resources`
- **THEN** the matcher proceeds to the always-unify rung and predicate evaluation for each candidate transformer

#### Scenario: FQN absent

- **WHEN** a demanded resource FQN is absent from the materialized `#matchers.resources`
- **THEN** the matcher records one `UnresolvedDemand` for that `(component, fqn)` with `Kind: "resource"` and the platform's same-base alternatives
- **AND** the matcher continues processing the remaining demanded FQNs (no fail-fast)

## REMOVED Requirements

### Requirement: Structured Missing-FQN Diagnostic

**Reason**: `MissingFQN` was produced on every empty-bucket miss and read by no production code in the library, `cli` or `opm-operator` (verified 2026-09-01). The requirement itself already stated the record was "retained for compatibility; the load-bearing diagnosis is the unresolved-demand set", and `UnresolvedDemand` carries the same instance-facing content: the demanded key and the contract-key-ordered alternatives, plus the disqualification causes `MissingFQN` never had.

**Migration**: Consumers of miss diagnostics read `MatchPlan.Unresolved` (and route on `*UnresolvedDemandsError` via `errors.As` from `Compile`), which no consumer needs to change because none read `MatchPlan.Missing`. The alternatives ordering requirement is unaffected; `UnresolvedDemand.Alternatives` remains its consumer.
