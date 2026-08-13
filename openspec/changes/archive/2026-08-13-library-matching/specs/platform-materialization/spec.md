# platform-materialization — Delta

## ADDED Requirements

### Requirement: Single-Provider Guard

While building the reverse matcher index, Materialize SHALL read each transformer's required contract definitions' declared `fulfilment` and, for every contract key declared `fulfilment: "provider"` by any provider, refuse a materialization in which transformers from more than one subscribed catalog supply that key. The refusal SHALL be a typed materialize error naming both catalog paths and the contract key, using the structurally-known catalog provenance (never parsed back out of an FQN). Embedded contract copies that disagree on `fulfilment` for one key SHALL also be refused as divergent contract definitions. Contract keys with `fulfilment: "catalog"` (the default) MAY be supplied by any number of transformers from any number of catalogs.

#### Scenario: Second provider refused

- **WHEN** two subscribed catalogs each supply a transformer requiring a contract declared `fulfilment: "provider"`
- **THEN** Materialize fails with an error naming both catalog paths and the key

#### Scenario: Catalog-fulfilled plurality allowed

- **WHEN** many transformers across catalogs require a contract with default fulfilment
- **THEN** Materialize succeeds and all candidates are indexed

#### Scenario: Divergent fulfilment refused

- **WHEN** two embedded copies of one contract key disagree on `fulfilment`
- **THEN** Materialize fails naming the key and both sources
