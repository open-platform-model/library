# platform-artifact Specification

## Purpose
TBD - created by syncing change add-platform-construct. Update Purpose after archive.

## Requirements

### Requirement: Platform Type Shape

The library SHALL expose `Platform` in `opm/platform/` with the uniform artifact shape: `{ APIVersion apiversion.Version; Metadata *PlatformMetadata; Package cue.Value }`.

#### Scenario: Platform struct fields

- **WHEN** a developer reads the `platform.Platform` struct
- **THEN** the struct has exactly three exported fields: `APIVersion`, `Metadata` (typed `*PlatformMetadata`), and `Package` (typed `cue.Value`)

#### Scenario: PlatformMetadata fields

- **WHEN** a developer reads `platform.PlatformMetadata`
- **THEN** the struct has at minimum: `Name`, `Type`, `Description`, `Labels`, `Annotations`
- **AND** the field set mirrors catalog enhancement 014's `#Platform.metadata` plus the top-level `type`

### Requirement: Platform Constructor from cue.Value

The library SHALL expose `func NewPlatformFromValue(k *kernel.Kernel, v cue.Value) (*Platform, error)`. The constructor SHALL detect `apiVersion`, look up the binding, decode `Metadata`, stamp the `APIVersion` field, and set `Package` to the supplied value unchanged.

#### Scenario: Successful construction

- **WHEN** a caller invokes `NewPlatformFromValue(k, v)` with a valid v1alpha2 Platform value
- **THEN** the returned `*Platform` has `APIVersion == apiversion.V1alpha2`, populated `Metadata`, and `Package == v`

#### Scenario: Unknown apiVersion

- **WHEN** the input `cue.Value` has an unrecognized `apiVersion`
- **THEN** the function returns a non-nil error wrapping `apiversion.ErrUnknownAPIVersion`

### Requirement: Platform Loader

The library SHALL expose `LoadPlatformPackage(ctx *cue.Context, dirPath string, opts loader.LoadOptions) (cue.Value, apiversion.Version, error)` in `opm/helper/loader/file/`. The function SHALL mirror `LoadModulePackage` and `LoadReleasePackage` in signature shape and behavior, resolving `dirPath` as a single CUE package via `cuelang.org/go/cue/load` and returning the built `cue.Value` together with the detected `apiVersion`. The function SHALL NOT accept a single-file path and SHALL NOT depend on a `platform.cue` filename convention; the platform is identified by the CUE `package` clause shared across the directory's files.

#### Scenario: Directory loaded as a CUE package

- **WHEN** `LoadPlatformPackage(ctx, "/path/to/platform-dir", opts)` is invoked and the directory contains one or more `.cue` files sharing a package that declares a `#Platform`
- **THEN** the function loads the directory via `load.Instances([]string{"."}, cfg)`, builds the instance, and returns the `cue.Value` and the detected `apiversion.Version`

#### Scenario: Registry override applied

- **WHEN** `LoadPlatformPackage(ctx, dir, loader.LoadOptions{Registry: "..."})` is invoked
- **THEN** the supplied registry override is applied via `load.Config.Env` so the platform's transitive imports resolve from the override registry without mutating process state

#### Scenario: Path is not a directory

- **WHEN** `dirPath` does not exist or is not a directory
- **THEN** the function returns a non-nil error and an empty `cue.Value`

#### Scenario: Unknown or missing apiVersion

- **WHEN** the loaded platform package has a missing or unrecognised `apiVersion` field
- **THEN** the function returns a non-nil error wrapping `apiversion.ErrUnknownAPIVersion`

#### Scenario: Kernel wrapper exists

- **WHEN** a caller invokes `(k *Kernel) LoadPlatformPackage(ctx, dirPath, opts)`
- **THEN** the result is identical to calling `loaderfile.LoadPlatformPackage` with `k.CueContext()`

### Requirement: Platform acquisition from a directory returns a source-carrying artifact

The kernel SHALL expose `AcquirePlatformFromDir`, which loads a `#Platform` CUE package from a directory through the existing shape-gated loader path (identical evaluation and gating to `LoadPlatformPackage`), constructs the typed platform via `NewPlatformFromValue`, and stamps `Source` in on-disk mode: `Overlay` nil, `Root` = the absolute path of the enclosing module root (the nearest ancestor holding `cue.mod/module.cue`, the directory itself when it is the root), and `Pkg` = the package directory relative to `Root` (empty for the root package). A directory with no enclosing module is its own root with an empty `Pkg`.

#### Scenario: Acquired platform carries its source

- **WHEN** a caller invokes `AcquirePlatformFromDir` on a module root directory holding a valid platform package
- **THEN** the returned `*Platform` has decoded `Metadata`, a `Package` identical to what `LoadPlatformPackage` returns for that directory, `Source.Root` equal to the directory's absolute path, an empty `Source.Pkg`, and a nil `Overlay`

#### Scenario: Acquired subpackage names its module root

- **WHEN** a caller invokes `AcquirePlatformFromDir` on a subdirectory of a module
- **THEN** the returned `*Platform` has `Source.Root` equal to the module root's absolute path and `Source.Pkg` equal to the subdirectory's slash-separated path relative to it

#### Scenario: Registry override honored

- **WHEN** `AcquirePlatformFromDir` is invoked with a registry override in its load options
- **THEN** the override is applied via the load configuration's environment (never `os.Setenv`), exactly as `LoadPlatformPackage` applies it

#### Scenario: Shape-gate failures propagate

- **WHEN** the directory's package fails the platform shape gate (wrong kind, missing concrete `metadata.name` or `type`)
- **THEN** the error wraps the same sentinel `LoadPlatformPackage` reports today, and no partial `*Platform` is returned

### Requirement: Platform carries its render source

`platform.Platform` SHALL expose `Source *platform.Source`, where `platform.Source` is a type alias of `module.Source` (the same re-export pattern as `platform.PlatformMetadata`). `Source` SHALL be nil when the platform was constructed from a bare `cue.Value`. `Source` is the render input: `Render` imports the platform package from it, so a platform without `Source` cannot be rendered against (`single-build-render`, "Render inputs are source-carrying artifacts"). No other kernel operation reads the field.

#### Scenario: Value-constructed platform has no source

- **WHEN** a caller builds a platform via `NewPlatformFromValue`
- **THEN** the returned `*Platform` has `Source == nil`

#### Scenario: Source-less platform cannot render

- **WHEN** a platform built via `NewPlatformFromValue` is passed to `Render`
- **THEN** `Render` refuses it with an error naming the missing source, before any build is staged
