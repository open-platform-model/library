## Why

`LoadPlatformFile` is the last loader that breaks CUE-package semantics: it loads a single named `.cue` file via `load.Instances([]string{filename})` instead of resolving a directory as a CUE package. Change `7c435f2` already unified module and release loaders onto `(ctx, dir, opts) -> (cue.Value, apiversion.Version, error)`; the platform loader is the remaining asymmetry. The library is not yet consumed by any downstream implementation, so we can replace and remove cleanly without deprecation shims or alias methods.

## What Changes

- **BREAKING** Remove `loaderfile.LoadPlatformFile` and its kernel wrapper `(k *Kernel).LoadPlatformFile`.
- Add `loaderfile.LoadPlatformPackage(ctx *cue.Context, dirPath string, opts LoadOptions) (cue.Value, apiversion.Version, error)` — loads a directory as a single CUE package (`load.Instances([]string{"."})`), mirroring `LoadModulePackage` / `LoadReleasePackage` exactly.
- Add kernel wrapper `(k *Kernel).LoadPlatformPackage`.
- Drop the `resolvePlatformFile` helper and the file-path / `platform.cue`-filename affordance. Platforms are identified by a directory whose package declares a `#Platform`, consistent with modules and releases.
- The second return value changes from the resolution directory (`string`, discarded by every current caller) to the detected `apiversion.Version`, via `apiversion.Detect`.
- No deprecation notices, no alias methods — direct replacement.

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities

- `platform-artifact`: the "Platform Loader" requirement changes — `LoadPlatformFile` (file-or-directory, `(cue.Value, string, error)`) becomes `LoadPlatformPackage` (directory-only CUE package, `(cue.Value, apiversion.Version, error)`); the kernel wrapper is renamed accordingly.
- `helper-packages`: the `opm/helper/loader/file/` public API listing changes — the `LoadPlatformFile` entry becomes `LoadPlatformPackage` with the package-loader signature, making all three loaders symmetric.

## Impact

- **SemVer**: MAJOR — breaking removal and signature change in the `opm/` public surface. Acceptable: no downstream consumers exist yet.
- **Affected code**: `opm/helper/loader/file/platform.go` (rewrite), `opm/helper/loader/file/platform_test.go` (rewrite — delete `resolvePlatformFile`-specific tests), `opm/helper/loader/file/release.go` (the shared `LoadOptions` doc comment references `LoadPlatformFile`), `opm/kernel/wrappers.go` (wrapper rename), `opm/kernel/doc.go`.
- **Affected callers** (all pass directories already, all discard the 2nd return value — migration is mechanical): `cmd/flow-inspect/main.go`, `opm/kernel/flow_integration_test.go`, `opm/kernel/flow_synth_integration_test.go`.
- **Capability dropped**: pointing the loader at an arbitrary single `.cue` file is removed deliberately, in exchange for symmetry with the module and release loaders.
