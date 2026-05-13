## MODIFIED Requirements

### Requirement: Loader Reorganization Under Helper

The filesystem-coupled loader SHALL live at `opm/helper/loader/file/`. The package SHALL expose the following public API:

- `LoadModulePackage(ctx, dirPath, opts) (cue.Value, apiversion.Version, error)` — loads a CUE package from a directory as a `#Module`, with registry override via `opts.Registry`.
- `LoadReleasePackage(ctx, dirPath, opts) (cue.Value, apiversion.Version, error)` — loads a CUE package from a directory as a `#ModuleRelease`, with registry override via `opts.Registry`.
- `LoadPlatformFile(ctx, path, opts) (cue.Value, string, error)` — loads a `#Platform` from a single `.cue` file (or `platform.cue` from a directory).
- `LoadProvider` and any associated types unchanged.

`LoadOptions` SHALL carry the registry override applied to `load.Config.Env`. Both package loaders SHALL accept the same `LoadOptions` value so that a caller can pass identical options through to module and release loads.

#### Scenario: Release package load

- **WHEN** a caller invokes `loaderfile.LoadReleasePackage(ctx, "./release-dir", loaderfile.LoadOptions{Registry: "..."})`
- **THEN** the function loads every `.cue` file in `./release-dir` that shares the package as a single CUE package
- **AND** detects the apiVersion via `apiversion.Detect`
- **AND** returns the evaluated `cue.Value` and detected `apiversion.Version`
- **AND** an unrecognised or missing apiVersion wraps `apiversion.ErrUnknownAPIVersion`

#### Scenario: Module package load with registry override

- **WHEN** a caller invokes `loaderfile.LoadModulePackage(ctx, "./module", loaderfile.LoadOptions{Registry: "..."})`
- **THEN** the function loads the module's CUE package using the supplied registry override
- **AND** module imports resolved from the registry succeed when the registry serves the required dependencies

#### Scenario: Multi-file release package

- **WHEN** a release directory contains both `release.cue` and `values.cue` declaring the same package name
- **AND** the caller invokes `loaderfile.LoadReleasePackage(ctx, dir, opts)`
- **THEN** the returned `cue.Value` reflects the unification of both files
- **AND** apiVersion detection succeeds against the unified value

## REMOVED Requirements

### Requirement: Single-file release loader

**Reason**: `LoadReleaseFile` is removed in favor of `LoadReleasePackage`. Releases are CUE artifacts and load the same way as modules: as a package from a directory. The single-file path forced releases into a single-file pattern that prevented multi-file release packages.

**Migration**: Callers move from `loaderfile.LoadReleaseFile(ctx, file, opts)` to `loaderfile.LoadReleasePackage(ctx, dir, opts)` where `dir` is the directory containing the release CUE files.

### Requirement: Standalone values-file loader

**Reason**: `LoadValuesFile` was a thin file-load + auto-unwrap-`values`-field helper used only by `Kernel.LoadSourceFromFile`. The auto-unwrap behavior now lives directly on `Kernel.LoadSourceFromFile`, which is the only caller that needs it. The standalone helper is unused and removed.

**Migration**: Callers of `loaderfile.LoadValuesFile` move to `Kernel.LoadSourceFromFile`, which preserves the auto-unwrap semantics. Callers that do not want auto-unwrap can call `load.Instances` directly with their preferred `LoadOptions`.

### Requirement: Deprecation Shim at opm/loader/

**Reason**: The shim was already scheduled for removal in a prior change and the `opm/loader/` directory has been deleted. This requirement is retired as part of normalising the loader spec; it no longer describes any code.
