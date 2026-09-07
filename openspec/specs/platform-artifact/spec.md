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

### Requirement: Platform acquisition from a directory returns a source-carrying artifact

The kernel SHALL expose `AcquirePlatformFromDir(ctx, dir)`, which loads a `#Platform` CUE package from a directory through the acquisition shape gate, constructs the typed platform via `platform.NewPlatformFromValue`, and stamps `Source` in on-disk mode: `Overlay` nil, `Root` = the absolute path of the enclosing module root (the nearest ancestor holding `cue.mod/module.cue`, the directory itself when it is the root), and `Pkg` = the package directory relative to `Root` (empty for the root package). A directory with no enclosing module is its own root with an empty `Pkg`. The registry mapping used for the platform's catalog imports SHALL be the kernel's (`WithRegistry`), applied through the load configuration's environment and never through `os.Setenv`; the verb takes no per-call override.

#### Scenario: Acquired platform carries its source

- **WHEN** a caller invokes `AcquirePlatformFromDir(ctx, dir)` on a module root directory holding a valid platform package
- **THEN** the returned `*Platform` has decoded `Metadata`, a `Package` holding the built platform value, `Source.Root` equal to the directory's absolute path, an empty `Source.Pkg`, and a nil `Overlay`

#### Scenario: Acquired subpackage names its module root

- **WHEN** a caller invokes `AcquirePlatformFromDir` on a subdirectory of a module
- **THEN** the returned `*Platform` has `Source.Root` equal to the module root's absolute path and `Source.Pkg` equal to the subdirectory's slash-separated path relative to it

#### Scenario: Registry override honored

- **WHEN** a kernel constructed with `WithRegistry(mapping)` acquires a platform whose `cue.mod` names catalogs under `opmodel.dev`
- **THEN** the catalog imports resolve through `mapping` with no per-call argument
- **AND** the process environment is not mutated

#### Scenario: Shape-gate failures propagate

- **WHEN** the directory's package fails the platform shape gate (wrong kind, missing concrete `metadata.name` or `type`, incomplete `#registry` entry)
- **THEN** the error wraps the corresponding `opm/errors` sentinel (`ErrWrongKind` or `ErrMissingRequiredField`), and no partial `*Platform` is returned

#### Scenario: Frontends stop composing load and construct

- **WHEN** a frontend needs a typed platform from a directory
- **THEN** it calls `AcquirePlatformFromDir` and reads `Platform.Package` for the raw value
- **AND** no `LoadPlatformPackage` method exists on the kernel to compose with a constructor

### Requirement: Platform carries its render source

`platform.Platform` SHALL expose `Source *platform.Source`, where `platform.Source` is a type alias of `module.Source` (the same re-export pattern as `platform.PlatformMetadata`). `Source` SHALL be nil when the platform was constructed from a bare `cue.Value`. `Source` is the render input: `Render` imports the platform package from it, so a platform without `Source` cannot be rendered against (`single-build-render`, "Render inputs are source-carrying artifacts"). No other kernel operation reads the field.

#### Scenario: Value-constructed platform has no source

- **WHEN** a caller builds a platform via `NewPlatformFromValue`
- **THEN** the returned `*Platform` has `Source == nil`

#### Scenario: Source-less platform cannot render

- **WHEN** a platform built via `NewPlatformFromValue` is passed to `Render`
- **THEN** `Render` refuses it with an error naming the missing source, before any build is staged
