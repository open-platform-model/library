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

### Requirement: Binding Path Constants for Platform Views

Each version binding (`opm/api/<version>/`) SHALL expose path constants for navigating a Platform package: `Paths().Registry`, `Paths().KnownResources`, `Paths().KnownTraits`, `Paths().ComposedTransformers`, `Paths().Matchers`. The binding SHALL also expose `DecodePlatformMetadata(v cue.Value) (*platform.PlatformMetadata, error)`.

#### Scenario: Registry path on v1alpha2

- **WHEN** code reads `binding.Paths().Registry` for the v1alpha2 binding
- **THEN** the path resolves to `#registry` within a Platform package

#### Scenario: Composed transformers path on v1alpha2

- **WHEN** code reads `binding.Paths().ComposedTransformers`
- **THEN** the path resolves to `#composedTransformers`

#### Scenario: Matchers path on v1alpha2

- **WHEN** code reads `binding.Paths().Matchers`
- **THEN** the path resolves to `#matchers`

#### Scenario: DecodePlatformMetadata on v1alpha2

- **WHEN** code invokes `binding.DecodePlatformMetadata(v)` on a v1alpha2 Platform value
- **THEN** the returned `*PlatformMetadata` has `Name`, `Type`, `Description`, `Labels`, `Annotations` populated from the value

### Requirement: Optional Platform Field on Phase Inputs

The phase input structs (`MatchInput`, `PlanInput`, `CompileInput`) SHALL gain an optional `Platform *Platform` field. The field SHALL be documented as optional in this slice and as becoming required after slice 09.

#### Scenario: Platform field present and optional

- **WHEN** a developer reads `MatchInput`, `PlanInput`, or `CompileInput`
- **THEN** each struct has a `Platform *platform.Platform` field
- **AND** the godoc states the field is optional today and becomes required when slice 09 lands
