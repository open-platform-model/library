## MODIFIED Requirements

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

## REMOVED Requirements

### Requirement: Platform Loader

**Reason**: `LoadPlatformPackage` was the raw half of `AcquirePlatformFromDir`; its one consumer call site composed it with `NewPlatformFromValue` to rebuild what the acquire verb returns, minus the source stamp. The loader is a kernel internal.

**Migration**: `k.LoadPlatformPackage(ctx, dir, opts)` + `k.NewPlatformFromValue(val)` becomes `k.AcquirePlatformFromDir(ctx, dir)`; the raw value is `Platform.Package`. The registry override is `kernel.WithRegistry`. Sentinel checks read `opm/errors`.
