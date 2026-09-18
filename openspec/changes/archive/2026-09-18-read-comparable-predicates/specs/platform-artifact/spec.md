## MODIFIED Requirements

### Requirement: Platform Type Shape

The library SHALL expose `Platform` in `opm/platform/` with the uniform artifact shape: `{ Metadata *PlatformMetadata; Package cue.Value; Source *Source }`. `Package` is the source of truth; `Metadata` is a decoded cache of the platform-level metadata; `Source` is the staged source tree a render imports the platform from. The platform's derived CUE views (`#composedTransformers`, `#contracts`) SHALL NOT be decoded at construction. The contract inventory core derives on `#Platform.#contracts` (enhancement 0015 D1, D2, D5, D18) SHALL be readable on demand through a `Contracts()` accessor returning a decoded `ContractInventory`: `DefinedBy` (contract FQN to the registry key of the enabled catalog listing it), `RequiredBy` (contract FQN to the implementation FQNs of the enabled transformers requiring it), `Unfulfilled` and `OverSubscribed` (provider-fulfilled contract FQNs required by nothing, and by transformers from more than one catalog), `Comparable` (every pair of enabled transformers whose match predicates are comparable over at least one shared catalog-fulfilled contract, each row naming the broader transformer, which matches every component the narrower one matches, the narrower transformer, and the shared contracts), and the booleans `Fulfilled`, `Routable` and `Discriminated`. The accessor SHALL NOT decode `defined`: its values are the catalogs' member schemas, which a caller reads off `Package`. The accessor SHALL NOT refuse on `Fulfilled: false`, `Routable: false` or `Discriminated: false`: all three are reports (D18, D5); whether a generation step withholds a platform package on `Routable: false` or `Discriminated: false` is that step's decision, outside the accessor. The accessor SHALL NOT default a report the value does not carry: a platform whose `#contracts` predates a report field is refused with an error naming the missing field and the first core release carrying it, never returned as a partial inventory whose missing verdict reads as pass or fail.

#### Scenario: Platform struct fields

- **WHEN** a developer reads the `platform.Platform` struct
- **THEN** the struct has exactly three exported fields: `Metadata` (typed `*PlatformMetadata`), `Package` (typed `cue.Value`) and `Source` (typed `*Source`), and no field holds a decoded `#composedTransformers` or `#contracts`

#### Scenario: PlatformMetadata fields

- **WHEN** a developer reads `platform.PlatformMetadata`
- **THEN** the struct has at minimum: `Name`, `Type`, `Description`, `Labels`, `Annotations`
- **AND** the field set mirrors `#Platform.metadata` plus the top-level `type`

#### Scenario: The inventory of a healthy platform reads as fulfilled and routable

- **WHEN** an acquired platform embeds one enabled catalog whose contract maps list a provider-fulfilled contract (a resource or a trait) and whose transformers require it, and whose transformers sharing a catalog-fulfilled contract each add a required label value or a required trait the others do not
- **THEN** `Contracts()` returns `DefinedBy` mapping that contract to the catalog's registry key, `RequiredBy` listing the requiring transformer, empty `Unfulfilled`, `OverSubscribed` and `Comparable`, and `Fulfilled`, `Routable` and `Discriminated` all true

#### Scenario: An over-subscribed platform is reported, not refused

- **WHEN** an acquired platform embeds two enabled catalogs whose transformers both require one provider-fulfilled contract that an enabled catalog lists
- **THEN** the platform acquires, `Contracts()` returns `OverSubscribed` naming the contract and `Routable` false, and no error is returned from acquisition or from the accessor

#### Scenario: An undiscriminated platform is reported, not refused

- **WHEN** an acquired platform embeds two enabled catalogs, one carrying a transformer that requires a catalog-fulfilled resource alone and the other carrying transformers that require the same resource plus a required label or a required trait
- **THEN** the platform acquires, `Contracts()` returns one `Comparable` row per such pair naming the resource-only transformer as broader, the other as narrower, and the resource as the shared contract, `Discriminated` is false, `Routable` is unaffected by the report, and no error is returned from acquisition or from the accessor

#### Scenario: An unlisted demand leaves the inventory empty

- **WHEN** an acquired platform embeds catalogs whose contract maps are empty, however many transformers they carry
- **THEN** `Contracts()` returns empty maps and lists with `Fulfilled`, `Routable` and `Discriminated` all true, and `defined` is not part of the returned value

#### Scenario: An inventory that predates the comparable-predicate report is refused

- **WHEN** a platform value carries `#contracts` with the six inventory fields and no `comparable` or `discriminated` (a value built against core `2.0.0-alpha.9`)
- **THEN** `Contracts()` returns a nil inventory and an error naming the missing field and the first release carrying it, and no six-field inventory is returned in its place
