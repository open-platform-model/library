## MODIFIED Requirements

### Requirement: Platform Type Shape

The library SHALL expose `Platform` in `opm/platform/` with the uniform artifact shape: `{ Metadata *PlatformMetadata; Package cue.Value; Source *Source }`. `Package` is the source of truth; `Metadata` is a decoded cache of the platform-level metadata; `Source` is the staged source tree a render imports the platform from. The platform's derived CUE views (`#composedTransformers`, `#contracts`) SHALL NOT be decoded at construction. The contract inventory core derives on `#Platform.#contracts` (enhancement 0015 D1, D2, D18) SHALL be readable on demand through a `Contracts()` accessor returning a decoded `ContractInventory`: `DefinedBy` (contract FQN to the registry key of the enabled catalog listing it), `RequiredBy` (contract FQN to the implementation FQNs of the enabled transformers requiring it), `Unfulfilled` and `OverSubscribed` (provider-fulfilled contract FQNs required by nothing, and by transformers from more than one catalog), and the booleans `Fulfilled` and `Routable`. The accessor SHALL NOT decode `defined`: its values are the catalogs' member schemas, which a caller reads off `Package`. The accessor SHALL NOT refuse on `Fulfilled: false` or `Routable: false`: both are reports (D18); whether a generation step withholds a platform package on `Routable: false` is that step's decision, outside the accessor.

#### Scenario: Platform struct fields

- **WHEN** a developer reads the `platform.Platform` struct
- **THEN** the struct has exactly three exported fields: `Metadata` (typed `*PlatformMetadata`), `Package` (typed `cue.Value`) and `Source` (typed `*Source`), and no field holds a decoded `#composedTransformers` or `#contracts`

#### Scenario: PlatformMetadata fields

- **WHEN** a developer reads `platform.PlatformMetadata`
- **THEN** the struct has at minimum: `Name`, `Type`, `Description`, `Labels`, `Annotations`
- **AND** the field set mirrors `#Platform.metadata` plus the top-level `type`

#### Scenario: The inventory of a healthy platform reads as fulfilled and routable

- **WHEN** an acquired platform embeds one enabled catalog whose contract maps list a provider-fulfilled trait and whose transformers require it
- **THEN** `Contracts()` returns `DefinedBy` mapping that trait to the catalog's registry key, `RequiredBy` listing the requiring transformer, empty `Unfulfilled` and `OverSubscribed`, and `Fulfilled` and `Routable` both true

#### Scenario: An over-subscribed platform is reported, not refused

- **WHEN** an acquired platform embeds two enabled catalogs whose transformers both require one provider-fulfilled trait that an enabled catalog lists
- **THEN** the platform acquires, `Contracts()` returns `OverSubscribed` naming the trait and `Routable` false, and no error is returned from acquisition or from the accessor

#### Scenario: An unlisted demand leaves the inventory empty

- **WHEN** an acquired platform embeds catalogs whose contract maps are empty, however many transformers they carry
- **THEN** `Contracts()` returns empty maps and lists with `Fulfilled` and `Routable` both true, and `defined` is not part of the returned value
