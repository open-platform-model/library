## MODIFIED Requirements

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
