## ADDED Requirements

### Requirement: Platform carries its staged source

`platform.Platform` SHALL expose `Source *platform.Source`, where `platform.Source` is a type alias of `module.Source` (the same re-export pattern as `platform.PlatformMetadata`). `Source` SHALL be nil when the platform was constructed from a bare `cue.Value`. No existing consumer behavior changes: `Materialize`, `Match` and `Compile` do not read the field in this change.

#### Scenario: Value-constructed platform has no source

- **WHEN** a caller builds a platform via `NewPlatformFromValue`
- **THEN** the returned `*Platform` has `Source == nil`

#### Scenario: Existing pipeline unaffected

- **WHEN** a platform with a nil `Source` is materialized and compiled through the existing pipeline
- **THEN** every result is identical to the behavior before this change

### Requirement: Platform acquisition from a directory returns a source-carrying artifact

The kernel SHALL expose `AcquirePlatformFromDir`, which loads a `#Platform` CUE package from a directory through the existing shape-gated loader path (identical evaluation and gating to `LoadPlatformPackage`), constructs the typed platform via `NewPlatformFromValue`, and stamps `Source` with the directory (`Root` = absolute directory, `Overlay` nil, root package).

#### Scenario: Acquired platform carries its source

- **WHEN** a caller invokes `AcquirePlatformFromDir` on a directory holding a valid platform package
- **THEN** the returned `*Platform` has decoded `Metadata`, a `Package` identical to what `LoadPlatformPackage` returns for that directory, and `Source.Root` equal to the directory's absolute path with a nil `Overlay`

#### Scenario: Registry override honored

- **WHEN** `AcquirePlatformFromDir` is invoked with a registry override in its load options
- **THEN** the override is applied via the load configuration's environment (never `os.Setenv`), exactly as `LoadPlatformPackage` applies it

#### Scenario: Shape-gate failures propagate

- **WHEN** the directory's package fails the platform shape gate (wrong kind, missing concrete `metadata.name` or `type`)
- **THEN** the error wraps the same sentinel `LoadPlatformPackage` reports today, and no partial `*Platform` is returned
