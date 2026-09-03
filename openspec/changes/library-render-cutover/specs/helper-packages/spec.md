## ADDED Requirements

### Requirement: No platform synthesis helper

`opm/helper/synth` SHALL expose no `Platform` function and no `PlatformInput`/`SubscriptionSpec` types, and the kernel SHALL expose no `SynthesizePlatform`. A platform is a CUE module on disk that imports its catalogs (0019 D5/D6); frontends generate that module themselves and acquire it with `AcquirePlatformFromDir`.

#### Scenario: Synth surface is instance-only

- **WHEN** a consumer inspects the exported identifiers of `opm/helper/synth`
- **THEN** `Instance` and its input types exist and no platform-synthesis identifier exists

#### Scenario: A frontend that synthesized platforms migrates to modules

- **WHEN** a frontend previously built a platform from typed subscription inputs
- **THEN** it writes a platform CUE module (a `cue.mod` pinning the catalogs, a `platform.cue` importing them) and acquires it from that directory

### Requirement: Loader shape gate validates identity and registry completeness

The three package loaders in `opm/helper/loader/file/` SHALL run a shape gate immediately after the CUE package is built and before the `cue.Value` is returned. The shape gate validates structural identity only; it SHALL NOT perform full schema validation of the artifact's configuration fields, which remains the contract of the Kernel layer.

The shape gate SHALL reject a package when any of the following hold, returning an error that wraps the corresponding sentinel:

- The built root value is not a struct, or `load.Instances` returned other than exactly one instance: wraps `ErrInvalidPackage`.
- The concrete `kind` literal does not match the loader's artifact type (`"Module"` for `LoadModulePackage`, `"ModuleInstance"` for `LoadInstancePackage`, `"Platform"` for `LoadPlatformPackage`): wraps `ErrWrongKind`.
- A required identity field is absent or not concrete: wraps `ErrMissingRequiredField`.

Required identity fields are those the schema never defaults:

- Module: `metadata.name`, `metadata.modulePath`, `metadata.version`.
- Instance: `metadata.name`, `metadata.namespace`, and `#module` present with `#module.kind == "Module"`.
- Platform: `metadata.name`, `type`, and every `#registry` entry complete under `cue.Concrete(true)`. Core derives an entry's `version` from its embedded catalog's stamped `metadata.version` (0019 D5), so an entry with no embedded catalog, or one embedding an unstamped catalog, fails here as a missing required field naming the entry. This is the refusal `single-build-render` relies on for a subscription-shaped platform; `#registry` is a definition, so no root-level validation reaches it.

A required identity field declared as a disjunction with a default (for example `#VersionType | *"1.0.1"`) is NOT concrete for the purpose of this gate: the gate judges the value as authored, before default finalization, and a default arm is a suggestion, not the value a release moved. When the gate rejects such a field, the error SHALL name the default arm's value and state that identity fields must be concrete literals, so the author is directed at the declaration rather than at the reference that copies it.

The package SHALL expose `ErrInvalidPackage`, `ErrWrongKind`, and `ErrMissingRequiredField` as sentinel errors so frontends can branch programmatically.

#### Scenario: Wrong artifact type rejected

- **WHEN** a caller invokes `LoadModulePackage(ctx, dir, opts)` and `dir` contains a package whose `kind` is `"Platform"`
- **THEN** the function returns a zero `cue.Value` and an error wrapping `ErrWrongKind`
- **AND** the error message names both the expected and the actual `kind`

#### Scenario: Missing identity field rejected

- **WHEN** a caller invokes `LoadModulePackage(ctx, dir, opts)` and the module package omits `metadata.name`
- **THEN** the function returns an error wrapping `ErrMissingRequiredField`
- **AND** the error identifies the field path `metadata.name`

#### Scenario: Defaulted identity field rejected with the default named

- **WHEN** a caller invokes `LoadModulePackage(ctx, dir, opts)` and the module package declares `metadata.version` as a disjunction with a default (for example `#VersionType | *"1.0.1"`) rather than a concrete literal
- **THEN** the function returns an error wrapping `ErrMissingRequiredField`
- **AND** the error identifies the field path `metadata.version`, names the default value `"1.0.1"`, and states that identity fields must be concrete literals
- **AND** a concrete `metadata.version` referencing an identity package whose `Version` is a plain literal passes the gate

#### Scenario: Instance embedding a non-module rejected

- **WHEN** a caller invokes `LoadInstancePackage(ctx, dir, opts)` and the instance's `#module` field carries a value whose `kind` is not `"Module"`
- **THEN** the function returns an error wrapping `ErrWrongKind`

#### Scenario: Registry entry with no embedded catalog rejected

- **WHEN** a caller invokes `LoadPlatformPackage(ctx, dir, opts)` and a `#registry` entry declares `enable` and a `version` scalar but embeds no `#catalog`
- **THEN** the function returns an error wrapping `ErrMissingRequiredField` that names the entry and its `version` field
- **AND** the same entry with the catalog imported and embedded passes the gate

#### Scenario: Non-struct root rejected

- **WHEN** a caller invokes any `Load*Package` on a directory whose package evaluates to a scalar or list rather than a struct
- **THEN** the function returns an error wrapping `ErrInvalidPackage`

#### Scenario: Conflicting package clauses rejected

- **WHEN** a directory contains two `.cue` files declaring different `package` names
- **AND** a caller invokes any `Load*Package` on that directory
- **THEN** the function returns a non-nil error
- **AND** the error originates from the CUE loader, not a partial `instances[0]` result

#### Scenario: Valid package passes the gate unchanged

- **WHEN** a caller invokes `LoadModulePackage(ctx, dir, opts)` on a well-formed module package
- **THEN** the shape gate passes
- **AND** the function returns the evaluated `cue.Value` exactly as before this change

## REMOVED Requirements

### Requirement: Loader Shape Gate Validation

**Reason**: the platform bullet and its scenario described a `#registry[id].#module` check from the module-registration era; the D5 entry embeds a catalog, and the gate's job on a platform is completeness of every entry. Restated as "Loader shape gate validates identity and registry completeness". The `(cue.Value, apiversion.Version, error)` signature sentence was already superseded by `schema-dispatch`.

**Migration**: none; the sentinels keep their identity.
