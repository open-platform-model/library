## 1. Helper loader

- [x] 1.1 Rewrite `opm/helper/loader/file/platform.go`: replace `LoadPlatformFile` with `LoadPlatformPackage(ctx *cue.Context, dirPath string, opts LoadOptions) (cue.Value, apiversion.Version, error)`, mirroring `LoadReleasePackage` (abs path, directory `os.Stat` check, `load.Instances([]string{"."})`, `ctx.BuildInstance`, `apiversion.Detect`, platform-flavored error strings)
- [x] 1.2 Delete the `resolvePlatformFile` helper
- [x] 1.3 Update the `LoadOptions` doc comment in `opm/helper/loader/file/release.go` to name `LoadPlatformPackage` instead of `LoadPlatformFile`

## 2. Kernel wrapper

- [x] 2.1 Rename `(k *Kernel).LoadPlatformFile` to `(k *Kernel).LoadPlatformPackage` in `opm/kernel/wrappers.go`, delegating to `loaderfile.LoadPlatformPackage`, and update its doc comment
- [x] 2.2 Update the `LoadPlatformFile` reference in `opm/kernel/doc.go`

## 3. Callers

- [x] 3.1 Update `cmd/flow-inspect/main.go` to call `k.LoadPlatformPackage`
- [x] 3.2 Update `opm/kernel/flow_integration_test.go` to call `k.LoadPlatformPackage`
- [x] 3.3 Update `opm/kernel/flow_synth_integration_test.go` to call `k.LoadPlatformPackage`

## 4. Tests

- [x] 4.1 Rewrite `opm/helper/loader/file/platform_test.go`: rename remaining tests to `LoadPlatformPackage`, assert the returned `apiversion.Version`, and delete the `_DirectFilePath` and `_DirectoryWithoutPlatformCue` scenarios that test removed behavior
- [x] 4.2 Verify `TestKernelWrapper_*` and `_RepoFixture` tests still pass against the directory-based fixture under `testdata/platform/v1alpha2/`

## 5. Validation

- [x] 5.1 Run `task check` (fmt, vet, lint, test) and confirm all gates pass
- [x] 5.2 Update `CHANGELOG.md` with the breaking `LoadPlatformFile` → `LoadPlatformPackage` entry
