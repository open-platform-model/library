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

- `LoadModulePackage(ctx, dirPath, opts) (cue.Value, error)`: loads a CUE package from a directory as a `#Module`, with registry override via `opts.Registry`.
- `LoadInstancePackage(ctx, dirPath, opts) (cue.Value, error)`: loads a CUE package from a directory as a `#ModuleInstance`, with registry override via `opts.Registry`.
- `LoadPlatformPackage(ctx, dirPath, opts) (cue.Value, error)`: loads a CUE package from a directory as a `#Platform`, with registry override via `opts.Registry`.

`LoadOptions` SHALL carry the registry override applied to `load.Config.Env`. All three package loaders SHALL accept the same `LoadOptions` value and share the same signature shape so that a caller can pass identical options through to module, instance, and platform loads.

A registry-sourced loader lives beside it at `opm/helper/loader/registry/`. The helper tree SHALL NOT carry a placeholder package that exports nothing: a loader for a source that has no consumer yet is added when that consumer exists, not scaffolded ahead of it.

#### Scenario: Instance package load

- **WHEN** a caller invokes `loaderfile.LoadInstancePackage(ctx, "./instance-dir", loaderfile.LoadOptions{Registry: "..."})`
- **THEN** the function loads every `.cue` file in `./instance-dir` that shares the package as a single CUE package
- **AND** returns the evaluated `cue.Value` after the shape gate has passed

#### Scenario: Platform package load

- **WHEN** a caller invokes `loaderfile.LoadPlatformPackage(ctx, "./platform-dir", loaderfile.LoadOptions{Registry: "..."})`
- **THEN** the function loads every `.cue` file in `./platform-dir` that shares the package as a single CUE package and returns the evaluated `cue.Value`

#### Scenario: Module package load with registry override

- **WHEN** a caller invokes `loaderfile.LoadModulePackage(ctx, "./module", loaderfile.LoadOptions{Registry: "..."})`
- **THEN** the function loads the module's CUE package using the supplied registry override
- **AND** module imports resolved from the registry succeed when the registry serves the required dependencies

#### Scenario: Multi-file instance package

- **WHEN** an instance directory contains both `instance.cue` and `values.cue` declaring the same package name
- **AND** the caller invokes `loaderfile.LoadInstancePackage(ctx, dir, opts)`
- **THEN** the returned `cue.Value` reflects the unification of both files

#### Scenario: Bytes loader skeleton present

- **WHEN** a developer lists the subpackages of `opm/helper/loader/`
- **THEN** `file`, `registry` and the shared `internal/` gate are present and no `bytes` package exists

### Requirement: Helper Layout for Future Subpackages

Future opt-in helpers SHALL follow the `opm/helper/<name>/` convention. Subpackages SHALL be added by their owning slices and not as part of the originating slice that established the convention. Past examples of helper subpackages in this convention SHALL reflect the current package layout; subpackages that have been collapsed into the kernel (such as the previous `opm/helper/values/`) SHALL NOT appear as exemplars. The current subpackages are `loader/file`, `loader/registry`, `synth` and `platformmodule`.

#### Scenario: Platform helper landing place

- **WHEN** a frontend needs to generate a platform module from catalog coordinates
- **THEN** `opm/helper/platformmodule/` is the directory that helper occupies, consistent with `opm/helper/loader/file/` and `opm/helper/synth/`

#### Scenario: Values helper subpackage no longer exists

- **WHEN** a developer searches `opm/helper/` for a `values` subpackage
- **THEN** no `opm/helper/values/` directory exists
- **AND** the canonical implementation of layered values validation lives at `Kernel.ValidateConfigDetailed` in `opm/kernel/`

### Requirement: Registry Loader Under Helper

The library SHALL provide a `opm/helper/loader/registry` subpackage, sibling to `opm/helper/loader/file`, that loads a published `#Module` from an OCI registry by `path@version`. It SHALL expose:

- `LoadModulePackageWithSource(ctx, cueCtx, modPath, version string, opts) (StagedSource, error)`: fetch the module's source via CUE's native module machinery, load it in memory as the main module with registry override via `opts.Registry`, verify its declared identity against the fetched coordinate, and return the built value together with the staged source tree (synthetic root plus overlay) so a follow-on build can reuse the module's own `cue.mod`.
- `LoadOptions` carrying the registry override, the same shape as `loader/file.LoadOptions`.

The package SHALL NOT expose a second, value-only entry point: `StagedSource.Value` is the value-only result, and dropping the staged source is the caller's one-field choice.

The package SHALL be opt-in under the `opm/helper/` boundary: a frontend MAY skip it and resolve registry modules another way.

#### Scenario: Registry loader present under helper

- **WHEN** a developer reads `opm/helper/loader/registry`
- **THEN** the package exposes `LoadModulePackageWithSource` for loading a published module by path and version
- **AND** it lives under `opm/helper/`, marking it opt-in convenience over the kernel core contract

#### Scenario: Value-only loader is gone

- **WHEN** a developer searches `opm/helper/loader/registry` and `opm/kernel` for `LoadModulePackage(` and `LoadModuleFromRegistry`
- **THEN** neither symbol exists; `Kernel.AcquireModuleFromRegistry` is the single kernel entry point for a published module

### Requirement: Shared Module Shape Gate Across Loaders

The module shape gate SHALL be single-sourced so that `opm/helper/loader/file` and `opm/helper/loader/registry` validate a `#Module` identically. The sentinels `ErrInvalidPackage`, `ErrWrongKind`, and `ErrMissingRequiredField` SHALL remain exposed from `opm/helper/loader/file` with unchanged identity, so existing `errors.Is` callers are unaffected. Extracting the gate to a shared location SHALL be behavior-preserving for `loader/file`.

#### Scenario: Identical gate for both loaders

- **WHEN** `loader/file.LoadModulePackage` and `loader/registry.LoadModulePackage` each load a package whose `kind` is not `"Module"`
- **THEN** both return an error wrapping the same `ErrWrongKind` sentinel value

#### Scenario: Sentinel identity preserved

- **WHEN** a frontend that previously called `errors.Is(err, loaderfile.ErrWrongKind)` is recompiled against this slice
- **THEN** that check continues to compile and behave identically

### Requirement: No platform synthesis helper

`opm/helper/synth` SHALL expose no `Platform` function and no `PlatformInput`/`SubscriptionSpec` types, and the kernel SHALL expose no `SynthesizePlatform`. A platform is a CUE module on disk that imports its catalogs (0019 D5/D6); a frontend that starts from catalog coordinates generates that module through `opm/helper/platformmodule` (`platform-module-generation`) or writes it by hand, and acquires it with `AcquirePlatformFromDir`. No helper SHALL turn typed subscription inputs into a platform `cue.Value` without a module on disk.

#### Scenario: Synth surface is instance-only

- **WHEN** a consumer inspects the exported identifiers of `opm/helper/synth`
- **THEN** `Instance` and its input types exist and no platform-synthesis identifier exists

#### Scenario: A frontend that synthesized platforms migrates to modules

- **WHEN** a frontend previously built a platform from typed subscription inputs
- **THEN** it generates or writes a platform CUE module (a `cue.mod` pinning the catalogs and their closure, a `platform.cue` importing them) and acquires it from that directory

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
