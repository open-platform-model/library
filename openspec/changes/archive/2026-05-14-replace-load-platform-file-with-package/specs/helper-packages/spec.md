## MODIFIED Requirements

### Requirement: Filesystem Loader Package

The filesystem-coupled loader SHALL live at `opm/helper/loader/file/`. The package SHALL expose the following public API:

- `LoadModulePackage(ctx, dirPath, opts) (cue.Value, apiversion.Version, error)` — loads a CUE package from a directory as a `#Module`, with registry override via `opts.Registry`.
- `LoadReleasePackage(ctx, dirPath, opts) (cue.Value, apiversion.Version, error)` — loads a CUE package from a directory as a `#ModuleRelease`, with registry override via `opts.Registry`.
- `LoadPlatformPackage(ctx, dirPath, opts) (cue.Value, apiversion.Version, error)` — loads a CUE package from a directory as a `#Platform`, with registry override via `opts.Registry`.

`LoadOptions` SHALL carry the registry override applied to `load.Config.Env`. All three package loaders SHALL accept the same `LoadOptions` value and share the same signature shape so that a caller can pass identical options through to module, release, and platform loads.

#### Scenario: Release package load

- **WHEN** a caller invokes `loaderfile.LoadReleasePackage(ctx, "./release-dir", loaderfile.LoadOptions{Registry: "..."})`
- **THEN** the function loads every `.cue` file in `./release-dir` that shares the package as a single CUE package

#### Scenario: Module package load with registry override

- **WHEN** a caller invokes `loaderfile.LoadModulePackage(ctx, "./module", loaderfile.LoadOptions{Registry: "..."})`
- **THEN** the function loads the module's CUE package using the supplied registry override

#### Scenario: Platform package load

- **WHEN** a caller invokes `loaderfile.LoadPlatformPackage(ctx, "./platform-dir", loaderfile.LoadOptions{Registry: "..."})`
- **THEN** the function loads every `.cue` file in `./platform-dir` that shares the package as a single CUE package and returns the detected `apiversion.Version`

#### Scenario: Multi-file release package

- **WHEN** a release directory contains both `release.cue` and `values.cue` declaring the same package name
- **AND** the caller invokes `loaderfile.LoadReleasePackage(ctx, dir, opts)`
- **THEN** the files are unified into a single CUE package instance
