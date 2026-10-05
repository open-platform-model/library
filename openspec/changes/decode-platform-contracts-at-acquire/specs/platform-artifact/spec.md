## MODIFIED Requirements

### Requirement: Platform Type Shape

The library SHALL expose `Platform` in `opm/platform/` with the uniform artifact shape: `{ Metadata *PlatformMetadata; Package cue.Value; Source *Source }`. `Package` is the source of truth; `Metadata` is a decoded cache of the platform-level metadata; `Source` is the staged source tree a render imports the platform from. No exported field SHALL hold a decoded derived CUE view (`#composedTransformers`, `#contracts`), and `#composedTransformers` SHALL NOT be decoded in Go. The contract inventory core derives on `#Platform.#contracts` (enhancement 0015 D1, D2, D5, D18) SHALL be readable through a `Contracts()` accessor returning a decoded `ContractInventory`, decoded once when the platform is constructed ("A platform records its core floor and contract inventory at construction"): `DefinedBy` (contract FQN to the registry key of the enabled catalog listing it, for keys exactly one enabled entry lists), `RequiredBy` (each such defined contract FQN to the implementation FQNs of the enabled transformers requiring it), `ProvidedBy` (every provider-fulfilled contract FQN some enabled transformer requires, whether or not an enabled catalog defines it, to the sorted registry keys, path plus major, of the enabled registry entries whose transformers require it), `Unfulfilled` (defined provider-fulfilled contract FQNs no enabled entry provides), `OverSubscribed` (provider-fulfilled contract FQNs `ProvidedBy` maps to two or more registry entries, defined or not), `Comparable` (every pair of enabled transformers whose match predicates are comparable over at least one shared catalog-fulfilled defined contract, each row naming the broader transformer, which matches every component the narrower one matches, the narrower transformer, and the shared contracts), `Collisions` (every contract FQN that two or more enabled registry entries' catalogs list in their contract maps, ascending), `CollidingEntries` (each `Collisions` key to the ascending registry keys, path plus major, of the enabled entries listing it), and the booleans `Fulfilled`, `Routable` and `Discriminated`. `Routable` SHALL be true exactly when `OverSubscribed` and `Collisions` are both empty. A colliding key is in none of `DefinedBy`, `RequiredBy`, `Unfulfilled` or `Comparable`, so `Fulfilled` and `Discriminated` can read true while `Collisions` is non-empty; the accessor's documentation SHALL say so, and a caller SHALL NOT read either as safe while `Collisions` is non-empty. A disabled entry never counts as a definer. Providers are counted per registry entry, the same count the render's single-provider guard reads: two majors of one catalog are two providers, and two transformers of one entry are one. The accessor SHALL NOT decode `defined`: its values are the catalogs' member schemas, which a caller reads off `Package`. The accessor SHALL NOT refuse on `Fulfilled: false`, `Routable: false`, `Discriminated: false` or a non-empty `Collisions`: all are reports (D18, D5); whether a generation step withholds a platform package on them is that step's decision, outside the accessor. The accessor SHALL NOT default a report the value does not carry: a platform whose `#contracts` predates a report field is refused with an error naming the missing field and the first core release carrying it, never returned as a partial inventory whose missing verdict reads as pass or fail. A missing `providedBy` SHALL be refused with the typed `PlatformCoreTooOldError` (field `providedBy`, release `2.0.0-alpha.12`), the error `Kernel.Render` returns for the same platform. The one exception is the collision report: an absent `collisions` or `collidingEntries` SHALL decode as empty, because every core release carrying `#contracts` without them fails to evaluate a platform whose enabled entries share a contract key (the `definedBy` conflict), so a value that evaluated without the report provably has no collision. A present collision field that fails to decode SHALL still be refused.

#### Scenario: Platform struct fields

- **WHEN** a developer reads the `platform.Platform` struct
- **THEN** the struct has exactly three exported fields: `Metadata` (typed `*PlatformMetadata`), `Package` (typed `cue.Value`) and `Source` (typed `*Source`), and no exported field holds a decoded `#composedTransformers` or `#contracts`

#### Scenario: PlatformMetadata fields

- **WHEN** a developer reads `platform.PlatformMetadata`
- **THEN** the struct has at minimum: `Name`, `Type`, `Description`, `Labels`, `Annotations`
- **AND** the field set mirrors `#Platform.metadata` plus the top-level `type`

#### Scenario: The inventory of a healthy platform reads as fulfilled and routable

- **WHEN** an acquired platform embeds one enabled catalog whose contract maps list a provider-fulfilled contract (a resource or a trait) and whose transformers require it, and whose transformers sharing a catalog-fulfilled contract each add a required label value or a required trait the others do not
- **THEN** `Contracts()` returns `DefinedBy` mapping that contract to the catalog's registry key, `RequiredBy` listing the requiring transformer, `ProvidedBy` mapping that contract to the one registry key, empty `Unfulfilled`, `OverSubscribed`, `Comparable`, `Collisions` and `CollidingEntries`, and `Fulfilled`, `Routable` and `Discriminated` all true

#### Scenario: An over-subscribed platform is reported, not refused

- **WHEN** an acquired platform embeds two enabled catalogs whose transformers both require one provider-fulfilled contract that an enabled catalog lists
- **THEN** the platform acquires, `Contracts()` returns `OverSubscribed` naming the contract, `ProvidedBy` mapping it to both registry keys in ascending order, empty `Collisions`, and `Routable` false, and no error is returned from acquisition or from the accessor

#### Scenario: An undiscriminated platform is reported, not refused

- **WHEN** an acquired platform embeds two enabled catalogs, one carrying a transformer that requires a catalog-fulfilled resource alone and the other carrying transformers that require the same resource plus a required label or a required trait
- **THEN** the platform acquires, `Contracts()` returns one `Comparable` row per such pair naming the resource-only transformer as broader, the other as narrower, and the resource as the shared contract, `Discriminated` is false, `Routable` is unaffected by the report, and no error is returned from acquisition or from the accessor

#### Scenario: An unlisted demand leaves the inventory empty

- **WHEN** an acquired platform embeds catalogs whose contract maps are empty and whose transformers require no provider-fulfilled contract, however many transformers they carry
- **THEN** `Contracts()` returns empty maps and lists, `Collisions` and `CollidingEntries` included, with `Fulfilled`, `Routable` and `Discriminated` all true, and `defined` is not part of the returned value

#### Scenario: An inventory that predates the comparable-predicate report is refused

- **WHEN** a platform value carries `#contracts` with the six inventory fields and no `comparable` or `discriminated` (a value built against core `2.0.0-alpha.9`)
- **THEN** `Contracts()` returns a nil inventory and an error naming the missing field and the first release carrying it, and no six-field inventory is returned in its place

#### Scenario: Two majors of one provider catalog are two providers

- **WHEN** an acquired platform enables `cat@v0` and `cat@v1`, and a transformer of each requires `cat@v0`'s provider-fulfilled gateway contract, and `cat@v1` lists no contract of its own
- **THEN** `Contracts()` returns `ProvidedBy` mapping the gateway key to the registry keys `cat@v0` and `cat@v1`, `OverSubscribed` naming it, empty `Collisions` and `Routable` false, and no error

#### Scenario: A disabled definer does not hide over-subscription

- **WHEN** an acquired platform carries the defining `cat@v0` entry with `enable: false` and enables `cat2` 0.2.0 and `cat@v1`, whose transformers both require the gateway contract `cat@v0` lists
- **THEN** `Contracts()` returns `ProvidedBy` mapping the gateway key to the registry keys `cat2@v0` and `cat@v1`, `OverSubscribed` naming it, `Routable` false, and neither `DefinedBy` nor `RequiredBy` carrying the gateway key, and no error

#### Scenario: An inventory that predates the provider count is refused

- **WHEN** a platform value carries `#contracts` with every other report field and no `providedBy` (a value built against core `2.0.0-alpha.11` or older)
- **THEN** `Contracts()` returns a nil inventory and an error from which `errors.As` extracts a `PlatformCoreTooOldError` naming the field `providedBy` and the release `2.0.0-alpha.12`, and no inventory lacking `ProvidedBy` is returned in its place

#### Scenario: Two majors sharing contract keys are reported as collisions

- **WHEN** an acquired platform pinning the default core release (`schema.DefaultSchemaModule`, a release carrying the collision report) enables `maj@v0` 0.1.0 and `maj@v1` 1.4.0, both listing the container resource, the expose trait and the provider-fulfilled backup trait at the same keys, and `maj@v1` also lists a `container@v2` resource no other entry lists
- **THEN** the platform acquires, `Contracts()` returns `Collisions` naming the three shared keys in ascending order, `CollidingEntries` mapping each to the registry keys `maj@v0` and `maj@v1`, `DefinedBy` carrying `container@v2` to `maj@v1` and none of the three shared keys, empty `OverSubscribed`, and `Routable` false, and no error is returned from acquisition or from the accessor

#### Scenario: A colliding key is left out of the other reports

- **WHEN** the colliding platform above also enables `bprov@v0`, whose transformer requires the colliding provider-fulfilled backup trait
- **THEN** `Contracts()` returns `ProvidedBy` mapping the backup key to `bprov@v0`, no backup key in `RequiredBy` or `Unfulfilled`, no `Comparable` row sharing a colliding key, `Fulfilled` true, `Discriminated` true although `maj@v0`'s deployment transformer and `maj@v1`'s bridge transformer require the colliding container key under equal predicates, and `Routable` false: the stated blind spot, pinned as such

#### Scenario: Collision and over-subscription are reported together

- **WHEN** the colliding platform enables both `bprov@v0` and `bprov@v1`, whose transformers both require the colliding backup trait
- **THEN** `Contracts()` returns the same `Collisions` and `CollidingEntries`, `OverSubscribed` naming the backup key with `ProvidedBy` mapping it to `bprov@v0` and `bprov@v1`, and `Routable` false

#### Scenario: An inventory that predates the collision report reads no collision

- **WHEN** a platform module pinning core `2.0.0-alpha.12` (whose `#contracts` carries no `collisions` or `collidingEntries`) is acquired
- **THEN** `Contracts()` returns a nil error, empty `Collisions` and `CollidingEntries`, and every other field exactly as it decoded before this change

## ADDED Requirements

### Requirement: A platform records its core floor and contract inventory at construction

`platform.NewPlatformFromValue`, and therefore `Kernel.AcquirePlatformFromDir`, SHALL read the platform value's core floor (whether `#contracts.providedBy` exists) and its contract inventory once, and record them in the returned `*Platform`. Where the inventory is refused, it SHALL record the refusal (a missing `#contracts` or report field, or a `#contracts` that does not evaluate or decode) in place of the inventory. A contracts refusal SHALL NOT fail construction: the constructor still fails only for the metadata reasons "Platform constructor takes a bare value" states. Source: owner decision h4 of the beta.1 kernel checklist walkthrough (2026-10-03).

On a constructed platform, `Contracts()` SHALL return the recorded inventory or the recorded refusal without reading `Package`. Each call SHALL return a value no other call shares: an inventory whose maps and slices a caller may change without changing what the next call returns, and a fresh typed `PlatformCoreTooOldError` that names the platform as before. The library SHALL expose `(*Platform).CoreFloor() error`, which SHALL return nil when the recorded floor is met and otherwise the typed `PlatformCoreTooOldError` with field `providedBy` and release `schema.ProvidedBySince` that `Kernel.Render` refuses the platform with. It SHALL NOT read `Package` on a constructed platform.

A `Platform` that the constructor did not build (a struct literal or the zero value) SHALL decode the same facts from its `Package` on the first `Contracts()` or `CoreFloor()` call, at most once, and every later call SHALL read that result. So a hand-built platform keeps today's results, and the zero `Platform` returns the `PlatformCoreTooOldError` with field `#contracts`. `Package` stays the source of truth: the recorded facts are a cache stamped at construction, like `Metadata`, and a caller that changes `Package` re-runs the constructor. The `Platform` documentation SHALL say this, and SHALL say that a `Platform` is used through a pointer and never copied.

#### Scenario: An older-core platform constructs and is refused later

- **WHEN** a platform module pinning core `2.0.0-alpha.10` is acquired with `AcquirePlatformFromDir`
- **THEN** acquisition returns the platform and no error, `Contracts()` returns a nil inventory and the typed `PlatformCoreTooOldError` it returned before this change, and `CoreFloor()` returns the typed `PlatformCoreTooOldError` with field `providedBy`

#### Scenario: Contracts and the floor no longer read Package

- **WHEN** a current-core platform is acquired, and its `Package` is then replaced with the zero `cue.Value`
- **THEN** `Contracts()` returns an inventory equal to the one it returned before the replacement, and `CoreFloor()` returns nil

#### Scenario: Each call returns its own inventory

- **WHEN** a caller changes a map entry and truncates a slice in the inventory `Contracts()` returned, then calls `Contracts()` again
- **THEN** the second inventory equals the first one as it was returned, before the changes

#### Scenario: Concurrent reads of one platform are race-free

- **WHEN** several goroutines call `Contracts()` and `CoreFloor()` on one acquired platform, and on one struct-literal platform built over the same `Package`, at the same time
- **THEN** every call returns an equal inventory and a nil floor error, with no data race reported under the race detector

#### Scenario: A hand-built platform keeps its results

- **WHEN** a caller builds `&platform.Platform{Metadata: md, Package: v, Source: s}` over the value of an older-core platform, with `md` nil or carrying an empty name
- **THEN** `Contracts()` and `CoreFloor()` return the refusals they returned before this change, and `Kernel.Render` refuses the platform before staging with the same error as `CoreFloor()`

#### Scenario: The zero platform reports no contracts

- **WHEN** a caller calls `Contracts()` on `&platform.Platform{}`
- **THEN** it returns a nil inventory and a `PlatformCoreTooOldError` with field `#contracts`, as before this change
