# platform-artifact Specification

## Purpose
TBD - created by syncing change add-platform-construct. Update Purpose after archive.

## Requirements

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

### Requirement: Platform Constructor from cue.Value

The library SHALL expose `func NewPlatformFromValue(k *kernel.Kernel, v cue.Value) (*Platform, error)`. The constructor SHALL detect `apiVersion`, look up the binding, decode `Metadata`, stamp the `APIVersion` field, and set `Package` to the supplied value unchanged.

#### Scenario: Successful construction

- **WHEN** a caller invokes `NewPlatformFromValue(k, v)` with a valid v1alpha2 Platform value
- **THEN** the returned `*Platform` has `APIVersion == apiversion.V1alpha2`, populated `Metadata`, and `Package == v`

#### Scenario: Unknown apiVersion

- **WHEN** the input `cue.Value` has an unrecognized `apiVersion`
- **THEN** the function returns a non-nil error wrapping `apiversion.ErrUnknownAPIVersion`

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
