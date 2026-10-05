## MODIFIED Requirements

### Requirement: The provider-fulfilled contract set is derived from the catalog

The catalog artifact SHALL expose the set of contracts it implements as a provider: every contract required by one of the catalog's own `#transformers` whose value carries `fulfilment: "provider"` (0015:D11). When the catalog was built against a core that derives `provides` (core `v2.0.0-beta.3` and later) and carries that field, the set SHALL be read from it. The set SHALL instead be derived from the catalog's `#transformers` by the library's fallback, which is deprecated and removed before GA, when the catalog's committed `cue.mod` pins a core older than `v2.0.0-beta.3`, whether or not the catalog authors a `provides` field, or when the catalog carries no `provides` field at all. For any catalog built against a core that derives `provides`, both sources SHALL yield the same set. The result SHALL be deterministically ordered and deduplicated, so two derivations of one catalog compare equal without the caller sorting. Derivation SHALL be a report: a catalog implementing no provider-fulfilled contract SHALL return an empty set and no error. A provider set that cannot be read SHALL be an error with no partial set returned, from either source: a `provides` field that is not a concrete list of strings, or a demand whose `fulfilment` is present but not a concrete string.

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

- **WHEN** the derivation runs over a catalog that carries a `provides` field and does not pin a core older than `v2.0.0-beta.3`
- **THEN** the returned set is that field's contracts, sorted and deduplicated
- **AND** the catalog's `#transformers` are not walked to produce it

#### Scenario: Core's provider set and the fallback agree

- **WHEN** a provider catalog is built against the core the library pins, so it carries `provides`
- **THEN** the set read from `provides` equals, element for element, the set the fallback derives from the same catalog's `#transformers`

#### Scenario: A catalog built against an older core falls back

- **WHEN** the derivation runs over a catalog whose committed `cue.mod` pins a core older than `v2.0.0-beta.3`
- **THEN** it returns the set derived from the catalog's `#transformers`, with no error
- **AND** a `provides` field the catalog authors beside its embedded `#Catalog` is not read, because an older core does not derive it

#### Scenario: An unreadable provider set is reported

- **WHEN** the derivation runs over a catalog whose `provides` field is present but is not a concrete list of strings
- **THEN** it fails with an error naming the `provides` field, and no set is returned
