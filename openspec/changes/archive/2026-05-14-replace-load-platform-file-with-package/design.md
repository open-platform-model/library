## Context

`opm/helper/loader/file/` exposes three artifact loaders. After change `7c435f2` two of them — `LoadModulePackage` and `LoadReleasePackage` — share the signature `(ctx *cue.Context, dirPath string, opts LoadOptions) (cue.Value, apiversion.Version, error)` and resolve a directory as a single CUE package via `load.Instances([]string{"."}, cfg)`. `LoadPlatformFile` is the holdout: it takes a file-or-directory path, resolves a literal `platform.cue` filename through the `resolvePlatformFile` helper, loads a single named file via `load.Instances([]string{filename}, cfg)`, returns the resolution directory as a `string`, and never detects `apiVersion`.

Every current caller (`cmd/flow-inspect/main.go`, the two kernel flow integration tests) already passes a directory and discards the second return value with `_`. The library has no downstream consumers yet, so the breaking change carries no external migration cost.

## Goals / Non-Goals

**Goals:**
- Make all three artifact loaders signature- and behavior-symmetric.
- Load platforms as CUE packages (`load.Instances([]string{"."}, cfg)`), honoring CUE package semantics — a platform is a directory whose files share a `package` clause and declare a `#Platform`.
- Detect and return `apiversion.Version`, matching the module and release loaders.
- Replace and remove cleanly: no deprecation notices, no alias methods.

**Non-Goals:**
- Preserving the ability to point the loader at an arbitrary single `.cue` file — deliberately dropped.
- Preserving the `platform.cue` filename convention — package resolution keys off the `package` clause, not the filename, consistent with modules/releases.
- Multi-file platform packages are now *possible* as a side effect, but exercising or documenting that pattern is out of scope here.
- Touching `NewPlatformFromValue` or any binding/decode path — apiVersion validation already happens downstream.

## Decisions

**1. Mirror `LoadReleasePackage` exactly.** `LoadPlatformPackage` becomes a near-copy of `LoadReleasePackage`: `filepath.Abs` → `os.Stat` directory check → `load.Config{Dir: absDir, Env: registryEnv(opts.Registry)}` → `load.Instances([]string{"."})` → `ctx.BuildInstance` → `apiversion.Detect`. Error-wrap strings say "platform" instead of "release". Rationale: the three loaders are conceptually one operation over three artifact kinds; divergence is the bug being fixed. Alternative considered — a shared generic `loadPackage` helper — rejected for now under YAGNI (Principle VII); the three functions are short and the duplication is already the established pattern from `7c435f2`.

**2. Delete `resolvePlatformFile` outright.** It exists only to support the file-path and `platform.cue`-filename affordances, both of which are being removed. No replacement.

**3. Second return value becomes `apiversion.Version`.** `apiversion.Detect` works on a platform value — `apis/core/v1alpha2/platform.cue` declares `apiVersion: #ApiVersion` and fixtures carry `opmodel.dev/v1alpha2`. The old `string` (resolution dir) is redundant once the input is always a directory the caller already holds.

**4. Rename the kernel wrapper.** `(k *Kernel).LoadPlatformFile` → `(k *Kernel).LoadPlatformPackage`, delegating to `loaderfile.LoadPlatformPackage(k.cueCtx, dirPath, opts)`. Update the doc comment in `opm/kernel/wrappers.go` and the reference in `opm/kernel/doc.go`.

**5. Update the shared `LoadOptions` doc comment** in `release.go` — it currently names `LoadPlatformFile` in its "shared by" list.

## Risks / Trade-offs

- **Lost capability: single-file loads** → Mitigation: no current caller uses it; symmetry with the module/release loaders is the stated objective and those are directory-only.
- **`platform.cue` filename convention no longer enforced** → Mitigation: this matches modules/releases, which never enforced `module.cue`/`release.cue`; the `package` clause is the real identifier. Test fixtures under `testdata/platform/v1alpha2/` already live in a directory and keep working unchanged.
- **Test churn** → `platform_test.go` needs a rewrite: the `_DirectFilePath` and `_DirectoryWithoutPlatformCue` scenarios test removed behavior and are deleted; remaining tests rename to `LoadPlatformPackage` and assert the returned `apiversion.Version`. Mechanical, low risk.
