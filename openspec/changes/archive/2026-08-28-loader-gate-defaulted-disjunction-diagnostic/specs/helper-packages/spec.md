## MODIFIED Requirements

### Requirement: Loader Shape Gate Validation

The three package loaders in `opm/helper/loader/file/` SHALL run a shape gate immediately after the CUE package is built and before the `cue.Value` is returned. The shape gate validates structural identity only; it SHALL NOT perform full schema validation of the artifact's configuration fields, which remains the contract of the Kernel/Binding layer.

The shape gate SHALL reject a package when any of the following hold, returning an error that wraps the corresponding sentinel:

- The built root value is not a struct, or `load.Instances` returned other than exactly one instance — wraps `ErrInvalidPackage`.
- The concrete `kind` literal does not match the loader's artifact type (`"Module"` for `LoadModulePackage`, `"ModuleInstance"` for `LoadInstancePackage`, `"Platform"` for `LoadPlatformPackage`) — wraps `ErrWrongKind`.
- A required identity field is absent or not concrete — wraps `ErrMissingRequiredField`.

Required identity fields are those the schema never defaults:

- Module: `metadata.name`, `metadata.modulePath`, `metadata.version`.
- Instance: `metadata.name`, `metadata.namespace`, and `#module` present with `#module.kind == "Module"`.
- Platform: `metadata.name`, `type`, and every `#registry[id].#module` present with `#registry[id].#module.kind == "Module"`.

A required identity field declared as a disjunction with a default (for example `#VersionType | *"1.0.1"`) is NOT concrete for the purpose of this gate: the gate judges the value as authored, before default finalization, and a default arm is a suggestion, not the value a release moved. When the gate rejects such a field, the error SHALL name the default arm's value and state that identity fields must be concrete literals, so the author is directed at the declaration rather than at the reference that copies it.

The package SHALL expose `ErrInvalidPackage`, `ErrWrongKind`, and `ErrMissingRequiredField` as sentinel errors so frontends can branch programmatically. Existing loader signatures `(cue.Value, apiversion.Version, error)` SHALL be unchanged.

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

#### Scenario: Platform registry entry pointing at a non-module rejected

- **WHEN** a caller invokes `LoadPlatformPackage(ctx, dir, opts)` and a `#registry` entry's `#module` carries a value whose `kind` is not `"Module"`
- **THEN** the function returns an error wrapping `ErrWrongKind`

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
- **AND** the function returns the evaluated `cue.Value` and detected `apiversion.Version` exactly as before this change
