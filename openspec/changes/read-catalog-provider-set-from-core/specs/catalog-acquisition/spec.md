## MODIFIED Requirements

### Requirement: The provider-fulfilled contract set is derived from the catalog

The catalog artifact SHALL expose the set of contracts it implements as a provider: every contract required by one of the catalog's own `#transformers` whose value carries `fulfilment: "provider"` (0015:D11). When the catalog carries core's derived `provides` field (core `v2.0.0-beta.3` and later), the set SHALL be read from that field. When the catalog carries no `provides` field, as a catalog built against an older core does, the set SHALL be derived from the catalog's `#transformers` by the library's fallback, which is deprecated and removed before GA. Both sources SHALL yield the same set for any catalog. The result SHALL be deterministically ordered and deduplicated, so two derivations of one catalog compare equal without the caller sorting. Derivation SHALL be a report: a catalog implementing no provider-fulfilled contract SHALL return an empty set and no error. A provider set that cannot be read SHALL be an error with no partial set returned, from either source: a `provides` field that is not a concrete list of strings, or a demand whose `fulfilment` is present but not a concrete string.

#### Scenario: A provider catalog's contracts are derived

- **WHEN** the derivation runs over a catalog whose transformers require a contract carrying `fulfilment: "provider"`
- **THEN** that contract's FQN is in the returned set
- **AND** a contract required by those transformers that carries any other fulfilment is not

#### Scenario: A catalog implementing no provider contract derives an empty set

- **WHEN** the derivation runs over a catalog none of whose transformers require a provider-fulfilled contract
- **THEN** it returns an empty set and no error, because implementing none is a fact about the catalog and not a malformed value

#### Scenario: The derived order is stable

- **WHEN** the derivation runs twice over one catalog
- **THEN** the two results are equal element for element

#### Scenario: Core's provider set is read when the catalog carries it

- **WHEN** the derivation runs over a catalog that carries a `provides` field
- **THEN** the returned set is that field's contracts, sorted and deduplicated
- **AND** the catalog's `#transformers` are not walked to produce it

#### Scenario: Core's provider set and the fallback agree

- **WHEN** a provider catalog is built against the core the library pins, so it carries `provides`
- **THEN** the set read from `provides` equals, element for element, the set the fallback derives from the same catalog's `#transformers`

#### Scenario: A catalog built against an older core falls back

- **WHEN** the derivation runs over a catalog whose `cue.mod` pins a core older than `v2.0.0-beta.3`, so it carries no `provides` field
- **THEN** it returns the set derived from the catalog's `#transformers`, with no error

#### Scenario: An unreadable provider set is reported

- **WHEN** the derivation runs over a catalog whose `provides` field is present but is not a concrete list of strings
- **THEN** it fails with an error naming the `provides` field, and no set is returned
