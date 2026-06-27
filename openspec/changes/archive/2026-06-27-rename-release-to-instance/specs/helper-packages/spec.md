## MODIFIED Requirements

### Requirement: Loader Reorganization Under Helper

The filesystem-coupled loader SHALL live at `opm/helper/loader/file/`. The package SHALL expose the following public API:

- `LoadModulePackage(ctx, dirPath, opts) (cue.Value, apiversion.Version, error)` — loads a CUE package from a directory as a `#Module`, with registry override via `opts.Registry`.
- `LoadInstancePackage(ctx, dirPath, opts) (cue.Value, apiversion.Version, error)` — loads a CUE package from a directory as a `#ModuleInstance`, with registry override via `opts.Registry`.
- `LoadPlatformPackage(ctx, dirPath, opts) (cue.Value, apiversion.Version, error)` — loads a CUE package from a directory as a `#Platform`, with registry override via `opts.Registry`.
- `LoadProvider` and any associated types unchanged.

`LoadOptions` SHALL carry the registry override applied to `load.Config.Env`. All three package loaders SHALL accept the same `LoadOptions` value and share the same signature shape so that a caller can pass identical options through to module, instance, and platform loads.

#### Scenario: Instance package load

- **WHEN** a caller invokes `loaderfile.LoadInstancePackage(ctx, "./instance-dir", loaderfile.LoadOptions{Registry: "..."})`
- **THEN** the function loads every `.cue` file in `./instance-dir` that shares the package as a single CUE package
- **AND** detects the apiVersion via `apiversion.Detect`
- **AND** returns the evaluated `cue.Value` and detected `apiversion.Version`
- **AND** an unrecognised or missing apiVersion wraps `apiversion.ErrUnknownAPIVersion`

#### Scenario: Platform package load

- **WHEN** a caller invokes `loaderfile.LoadPlatformPackage(ctx, "./platform-dir", loaderfile.LoadOptions{Registry: "..."})`
- **THEN** the function loads every `.cue` file in `./platform-dir` that shares the package as a single CUE package and returns the detected `apiversion.Version`

#### Scenario: Module package load with registry override

- **WHEN** a caller invokes `loaderfile.LoadModulePackage(ctx, "./module", loaderfile.LoadOptions{Registry: "..."})`
- **THEN** the function loads the module's CUE package using the supplied registry override
- **AND** module imports resolved from the registry succeed when the registry serves the required dependencies

#### Scenario: Multi-file instance package

- **WHEN** an instance directory contains both `instance.cue` and `values.cue` declaring the same package name
- **AND** the caller invokes `loaderfile.LoadInstancePackage(ctx, dir, opts)`
- **THEN** the returned `cue.Value` reflects the unification of both files
- **AND** apiVersion detection succeeds against the unified value

#### Scenario: Bytes loader skeleton present

- **WHEN** a developer reads `opm/helper/loader/bytes/`
- **THEN** the package has a doc-only file describing intent (in-memory loading for Crossplane fn / tests / fuzzing)
- **AND** the package exposes no functions yet

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

The package SHALL expose `ErrInvalidPackage`, `ErrWrongKind`, and `ErrMissingRequiredField` as sentinel errors so frontends can branch programmatically. Existing loader signatures `(cue.Value, apiversion.Version, error)` SHALL be unchanged.

#### Scenario: Wrong artifact type rejected

- **WHEN** a caller invokes `LoadModulePackage(ctx, dir, opts)` and `dir` contains a package whose `kind` is `"Platform"`
- **THEN** the function returns a zero `cue.Value` and an error wrapping `ErrWrongKind`
- **AND** the error message names both the expected and the actual `kind`

#### Scenario: Missing identity field rejected

- **WHEN** a caller invokes `LoadModulePackage(ctx, dir, opts)` and the module package omits `metadata.name`
- **THEN** the function returns an error wrapping `ErrMissingRequiredField`
- **AND** the error identifies the field path `metadata.name`

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
