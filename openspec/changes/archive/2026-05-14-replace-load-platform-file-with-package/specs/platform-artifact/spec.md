## MODIFIED Requirements

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

## REMOVED Requirements

### Requirement: ~~Platform Loader (file-based)~~

**Reason**: `LoadPlatformFile` loaded a single named `.cue` file (or a literal `platform.cue` from a directory) and returned the resolution directory as a `string`, breaking symmetry with the package-based `LoadModulePackage` and `LoadReleasePackage`. It is replaced — not deprecated — by the `LoadPlatformPackage` requirement above.

**Migration**: Replace `LoadPlatformFile(ctx, path, opts)` calls with `LoadPlatformPackage(ctx, dir, opts)`, passing the platform's directory; the second return value is now `apiversion.Version` instead of the resolution directory string. Replace `(k *Kernel).LoadPlatformFile` calls with `(k *Kernel).LoadPlatformPackage`. The library has no downstream consumers, so no compatibility shim or alias is provided.
