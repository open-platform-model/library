## MODIFIED Requirements

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
