# helper-packages Specification

## Purpose
The `opm/helper/` subdirectory is the opt-in convenience boundary of the OPM library. Subpackages under `opm/helper/` are opinionated frontend conveniences that wrap kernel primitives for specific embedding patterns; a frontend MAY skip them and call the kernel directly. Anything outside `opm/helper/` is part of the kernel core contract that every frontend (CLI, controller, Crossplane composition function, future runtimes) MUST honour. This boundary keeps the kernel's public surface small and lets the helper layer evolve independently. Future kernel-redesign slices add subpackages here (loaders, layered values, Platform composition); each new helper requires its own slice.
## Requirements
### Requirement: Helper Boundary at opm/helper/

The library SHALL maintain a `opm/helper/` subdirectory whose subpackages are opt-in, opinionated frontend conveniences. Anything outside `opm/helper/` SHALL be considered part of the kernel core contract.

#### Scenario: Helper boundary documented

- **WHEN** a developer reads `opm/helper/doc.go`
- **THEN** the file documents that anything under `opm/helper/` is opt-in and a frontend MAY skip it
- **AND** documents that anything outside `opm/helper/` is part of the kernel contract

### Requirement: Loader Reorganization Under Helper

The filesystem-coupled loader SHALL live at `opm/helper/loader/file/`. The package SHALL expose the following public API:

- `LoadModulePackage(ctx, dirPath, opts) (cue.Value, apiversion.Version, error)` — loads a CUE package from a directory as a `#Module`, with registry override via `opts.Registry`.
- `LoadReleasePackage(ctx, dirPath, opts) (cue.Value, apiversion.Version, error)` — loads a CUE package from a directory as a `#ModuleRelease`, with registry override via `opts.Registry`.
- `LoadPlatformPackage(ctx, dirPath, opts) (cue.Value, apiversion.Version, error)` — loads a CUE package from a directory as a `#Platform`, with registry override via `opts.Registry`.
- `LoadProvider` and any associated types unchanged.

`LoadOptions` SHALL carry the registry override applied to `load.Config.Env`. All three package loaders SHALL accept the same `LoadOptions` value and share the same signature shape so that a caller can pass identical options through to module, release, and platform loads.

#### Scenario: Release package load

- **WHEN** a caller invokes `loaderfile.LoadReleasePackage(ctx, "./release-dir", loaderfile.LoadOptions{Registry: "..."})`
- **THEN** the function loads every `.cue` file in `./release-dir` that shares the package as a single CUE package
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

#### Scenario: Multi-file release package

- **WHEN** a release directory contains both `release.cue` and `values.cue` declaring the same package name
- **AND** the caller invokes `loaderfile.LoadReleasePackage(ctx, dir, opts)`
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
- The concrete `kind` literal does not match the loader's artifact type (`"Module"` for `LoadModulePackage`, `"ModuleRelease"` for `LoadReleasePackage`, `"Platform"` for `LoadPlatformPackage`) — wraps `ErrWrongKind`.
- A required identity field is absent or not concrete — wraps `ErrMissingRequiredField`.

Required identity fields are those the schema never defaults:

- Module: `metadata.name`, `metadata.modulePath`, `metadata.version`.
- Release: `metadata.name`, `metadata.namespace`, and `#module` present with `#module.kind == "Module"`.
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

#### Scenario: Release embedding a non-module rejected

- **WHEN** a caller invokes `LoadReleasePackage(ctx, dir, opts)` and the release's `#module` field carries a value whose `kind` is not `"Module"`
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

### Requirement: Helper Layout for Future Subpackages

Future opt-in helpers SHALL follow the `opm/helper/<name>/` convention. Subpackages SHALL be added by their owning slices (e.g. `opm/helper/platform/` by slice 10) and not as part of the originating slice that established the convention. Past examples of helper subpackages in this convention SHALL reflect the current package layout; subpackages that have been collapsed into the kernel (such as the previous `opm/helper/values/`) SHALL NOT appear as exemplars.

#### Scenario: Platform helper landing place

- **WHEN** slice 10 (`add-platform-composition-helper`) is implemented
- **THEN** `opm/helper/platform/` is the directory it occupies
- **AND** the convention is consistent with `opm/helper/loader/file/`

#### Scenario: Values helper subpackage no longer exists

- **WHEN** a developer searches `opm/helper/` for a `values` subpackage
- **THEN** no `opm/helper/values/` directory exists
- **AND** the canonical implementation of layered values validation lives at `Kernel.ValidateConfigDetailed` in `opm/kernel/`

### Requirement: Registry Loader Under Helper

The library SHALL provide a `opm/helper/loader/registry` subpackage, sibling to `opm/helper/loader/file`, that loads a published `#Module` from an OCI registry by `path@version`. It SHALL expose:

- `LoadModulePackage(ctx, cueCtx, modPath, version string, opts) (cue.Value, error)` — fetch the module's source via CUE's native module machinery and load it in memory as the main module, with registry override via `opts.Registry`. (Mirrors the current `loader/file.LoadModulePackage` return shape `(cue.Value, error)`.)
- `LoadOptions` carrying the registry override, the same shape as `loader/file.LoadOptions`.

The package SHALL be opt-in under the `opm/helper/` boundary: a frontend MAY skip it and resolve registry modules another way.

#### Scenario: Registry loader present under helper

- **WHEN** a developer reads `opm/helper/loader/registry`
- **THEN** the package exposes `LoadModulePackage` for loading a published module by path and version
- **AND** it lives under `opm/helper/`, marking it opt-in convenience over the kernel core contract

### Requirement: Shared Module Shape Gate Across Loaders

The module shape gate SHALL be single-sourced so that `opm/helper/loader/file` and `opm/helper/loader/registry` validate a `#Module` identically. The sentinels `ErrInvalidPackage`, `ErrWrongKind`, and `ErrMissingRequiredField` SHALL remain exposed from `opm/helper/loader/file` with unchanged identity, so existing `errors.Is` callers are unaffected. Extracting the gate to a shared location SHALL be behavior-preserving for `loader/file`.

#### Scenario: Identical gate for both loaders

- **WHEN** `loader/file.LoadModulePackage` and `loader/registry.LoadModulePackage` each load a package whose `kind` is not `"Module"`
- **THEN** both return an error wrapping the same `ErrWrongKind` sentinel value

#### Scenario: Sentinel identity preserved

- **WHEN** a frontend that previously called `errors.Is(err, loaderfile.ErrWrongKind)` is recompiled against this slice
- **THEN** that check continues to compile and behave identically

