# platform-artifact Specification

## Purpose
The Platform as a typed kernel artifact: a CUE package importing its catalogs, held as the built value with decoded metadata and the staged source tree a render imports it from. Covers the artifact's shape and its on-demand contract inventory, which reports fulfilment, routability and discrimination without refusing on them but refuses an inventory that predates a report field. Also covers construction from a bare CUE value, acquisition from a directory through the shape gate under the kernel's registry mapping, and the source the platform carries as its render input, absent on a value-constructed platform, which therefore cannot be rendered.

## Requirements

### Requirement: Platform Type Shape

The library SHALL expose `Platform` in `opm/platform/` with the uniform artifact shape: `{ Metadata *PlatformMetadata; Package cue.Value; Source *Source }`. `Package` is the source of truth; `Metadata` is a decoded cache of the platform-level metadata; `Source` is the staged source tree a render imports the platform from. The platform's derived CUE views (`#composedTransformers`, `#contracts`) SHALL NOT be decoded at construction. The contract inventory core derives on `#Platform.#contracts` (enhancement 0015 D1, D2, D5, D18) SHALL be readable on demand through a `Contracts()` accessor returning a decoded `ContractInventory`: `DefinedBy` (contract FQN to the registry key of the enabled catalog listing it, for keys exactly one enabled entry lists), `RequiredBy` (each such defined contract FQN to the implementation FQNs of the enabled transformers requiring it), `ProvidedBy` (every provider-fulfilled contract FQN some enabled transformer requires, whether or not an enabled catalog defines it, to the sorted registry keys, path plus major, of the enabled registry entries whose transformers require it), `Unfulfilled` (defined provider-fulfilled contract FQNs no enabled entry provides), `OverSubscribed` (provider-fulfilled contract FQNs `ProvidedBy` maps to two or more registry entries, defined or not), `Comparable` (every pair of enabled transformers whose match predicates are comparable over at least one shared catalog-fulfilled defined contract, each row naming the broader transformer, which matches every component the narrower one matches, the narrower transformer, and the shared contracts), `Collisions` (every contract FQN that two or more enabled registry entries' catalogs list in their contract maps, ascending), `CollidingEntries` (each `Collisions` key to the ascending registry keys, path plus major, of the enabled entries listing it), and the booleans `Fulfilled`, `Routable` and `Discriminated`. `Routable` SHALL be true exactly when `OverSubscribed` and `Collisions` are both empty. A colliding key is in none of `DefinedBy`, `RequiredBy`, `Unfulfilled` or `Comparable`, so `Fulfilled` and `Discriminated` can read true while `Collisions` is non-empty; the accessor's documentation SHALL say so, and a caller SHALL NOT read either as safe while `Collisions` is non-empty. A disabled entry never counts as a definer. Providers are counted per registry entry, the same count the render's single-provider guard reads: two majors of one catalog are two providers, and two transformers of one entry are one. The accessor SHALL NOT decode `defined`: its values are the catalogs' member schemas, which a caller reads off `Package`. The accessor SHALL NOT refuse on `Fulfilled: false`, `Routable: false`, `Discriminated: false` or a non-empty `Collisions`: all are reports (D18, D5); whether a generation step withholds a platform package on them is that step's decision, outside the accessor. The accessor SHALL NOT default a report the value does not carry: a platform whose `#contracts` predates a report field is refused with an error naming the missing field and the first core release carrying it, never returned as a partial inventory whose missing verdict reads as pass or fail. A missing `providedBy` SHALL be refused with the typed `PlatformCoreTooOldError` (field `providedBy`, release `2.0.0-alpha.12`), the error `Kernel.Render` returns for the same platform. The one exception is the collision report: an absent `collisions` or `collidingEntries` SHALL decode as empty, because every core release carrying `#contracts` without them fails to evaluate a platform whose enabled entries share a contract key (the `definedBy` conflict), so a value that evaluated without the report provably has no collision. A present collision field that fails to decode SHALL still be refused.

#### Scenario: Platform struct fields

- **WHEN** a developer reads the `platform.Platform` struct
- **THEN** the struct has exactly three exported fields: `Metadata` (typed `*PlatformMetadata`), `Package` (typed `cue.Value`) and `Source` (typed `*Source`), and no field holds a decoded `#composedTransformers` or `#contracts`

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

### Requirement: Platform acquisition from a directory returns a source-carrying artifact

The kernel SHALL expose `AcquirePlatformFromDir(ctx, dir)`, which loads a `#Platform` CUE package from a directory through the acquisition shape gate, constructs the typed platform via `platform.NewPlatformFromValue`, and stamps `Source` in on-disk mode: `Overlay` nil, `Root` = the absolute path of the enclosing module root (the nearest ancestor holding `cue.mod/module.cue`, the directory itself when it is the root), and `Pkg` = the package directory relative to `Root` (empty for the root package). A directory with no enclosing module is its own root with an empty `Pkg`. The registry mapping used for the platform's catalog imports SHALL be the kernel's (`WithRegistry`), applied through the load configuration's environment and never through `os.Setenv`; the verb takes no per-call override.

#### Scenario: Acquired platform carries its source

- **WHEN** a caller invokes `AcquirePlatformFromDir(ctx, dir)` on a module root directory holding a valid platform package
- **THEN** the returned `*Platform` has decoded `Metadata`, a `Package` holding the built platform value, `Source.Root` equal to the directory's absolute path, an empty `Source.Pkg`, and a nil `Overlay`

#### Scenario: Acquired subpackage names its module root

- **WHEN** a caller invokes `AcquirePlatformFromDir` on a subdirectory of a module
- **THEN** the returned `*Platform` has `Source.Root` equal to the module root's absolute path and `Source.Pkg` equal to the subdirectory's slash-separated path relative to it

#### Scenario: Registry override honored

- **WHEN** a kernel constructed with `WithRegistry(mapping)` acquires a platform whose `cue.mod` names catalogs under `opmodel.dev`
- **THEN** the catalog imports resolve through `mapping` with no per-call argument
- **AND** the process environment is not mutated

#### Scenario: Shape-gate failures propagate

- **WHEN** the directory's package fails the platform shape gate (wrong kind, missing concrete `metadata.name` or `type`, incomplete `#registry` entry)
- **THEN** the error wraps the corresponding `opm/errors` sentinel (`ErrWrongKind` or `ErrMissingRequiredField`), and no partial `*Platform` is returned

#### Scenario: Frontends stop composing load and construct

- **WHEN** a frontend needs a typed platform from a directory
- **THEN** it calls `AcquirePlatformFromDir` and reads `Platform.Package` for the raw value
- **AND** no `LoadPlatformPackage` method exists on the kernel to compose with a constructor

### Requirement: Platform carries its render source

`platform.Platform` SHALL expose `Source *platform.Source`, where `platform.Source` is a type alias of `module.Source` (the same re-export pattern as `platform.PlatformMetadata`). `Source` SHALL be nil when the platform was constructed from a bare `cue.Value`. `Source` is the render input: `Render` imports the platform package from it, so a platform without `Source` cannot be rendered against (`single-build-render`, "Render inputs are source-carrying artifacts"). No other kernel operation reads the field.

#### Scenario: Value-constructed platform has no source

- **WHEN** a caller builds a platform via `NewPlatformFromValue`
- **THEN** the returned `*Platform` has `Source == nil`

#### Scenario: Source-less platform cannot render

- **WHEN** a platform built via `NewPlatformFromValue` is passed to `Render`
- **THEN** `Render` refuses it with an error naming the missing source, before any build is staged

### Requirement: Platform constructor takes a bare value

The library SHALL expose `func NewPlatformFromValue(v cue.Value) (*Platform, error)` in `opm/platform`. The constructor SHALL take only the platform value. It SHALL decode `Metadata` from the value's `metadata` field, hoist the root-level `type` into `Metadata.Type`, set `Package` to the supplied value unchanged, and leave `Source` nil. It SHALL perform no API version detection and no binding lookup. When `metadata` is absent or does not decode, or the root `type` is present but not a string, it SHALL return an error and a nil `*Platform`, never a partial one.

#### Scenario: Successful construction

- **WHEN** a caller invokes `platform.NewPlatformFromValue(v)` with a `#Platform` value carrying `metadata.name`, `metadata.description`, labels, annotations and a root `type: "kubernetes"`
- **THEN** the returned `*Platform` has those fields in `Metadata`, `Metadata.Type == "kubernetes"`, `Package` equal to `v`, and `Source == nil`

#### Scenario: Missing metadata is refused

- **WHEN** a caller invokes `platform.NewPlatformFromValue(v)` with a value that has no `metadata` field
- **THEN** it returns an error stating that the platform metadata field is required, and a nil `*Platform`

#### Scenario: The constructor takes one argument

- **WHEN** a consumer inspects the signature of `platform.NewPlatformFromValue`
- **THEN** it takes a single `cue.Value` and no kernel, context or option argument
- **AND** no `opm/apiversion` package exists to detect a version with
