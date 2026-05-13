## 1. loader/file: add LoadReleasePackage

- [x] 1.1 Add `LoadReleasePackage(ctx *cue.Context, dirPath string, opts LoadOptions) (cue.Value, apiversion.Version, error)` in `opm/helper/loader/file/release.go`. Mirror `LoadModulePackage` body, but pass `Env: registryEnv(opts.Registry)` in `load.Config`.
- [x] 1.2 Godoc the function with the same shape and "recommended entry point: Kernel.LoadReleasePackage" pointer used elsewhere in the package.

## 2. loader/file: modify LoadModulePackage to accept LoadOptions

- [x] 2.1 Add `opts LoadOptions` parameter to `LoadModulePackage` in `opm/helper/loader/file/module.go`. Apply `Env: registryEnv(opts.Registry)` to the `load.Config`.
- [x] 2.2 Update the godoc to mention `LoadOptions.Registry`.

## 3. loader/file: remove LoadReleaseFile and LoadValuesFile

- [x] 3.1 Delete `LoadReleaseFile` from `opm/helper/loader/file/release.go`.
- [x] 3.2 Delete `LoadValuesFile` from `opm/helper/loader/file/release.go`.
- [x] 3.3 Delete `resolveReleaseFile` (becomes dead code after `LoadReleaseFile` is gone).
- [x] 3.4 Keep `registryEnv` (still used by `LoadModulePackage`, `LoadReleasePackage`, `LoadPlatformFile`).

## 4. kernel: update wrappers

- [x] 4.1 Update `Kernel.LoadModulePackage` in `opm/kernel/wrappers.go` to accept `opts loaderfile.LoadOptions` and forward it.
- [x] 4.2 Add `Kernel.LoadReleasePackage` in `opm/kernel/wrappers.go` mirroring `Kernel.LoadModulePackage`.
- [x] 4.3 Delete `Kernel.LoadReleaseFile` from `opm/kernel/wrappers.go`.
- [x] 4.4 Delete `Kernel.LoadValuesFile` from `opm/kernel/wrappers.go`.

## 5. kernel: absorb values-file auto-unwrap into LoadSourceFromFile

- [x] 5.1 Rewrite `Kernel.LoadSourceFromFile` in `opm/kernel/source_loader.go` to call `load.Instances` directly (replacing the `loaderfile.LoadValuesFile` delegate).
- [x] 5.2 Preserve the auto-extract behavior: if the loaded value has a top-level `values:` field with no error, return that field as `Source.Value`; otherwise return the whole value.
- [x] 5.3 Update godoc to document the auto-unwrap explicitly. Add a one-line `// Why:` comment naming the OPM values-file convention.
- [x] 5.4 Remove the dependency on `loaderfile` from `source_loader.go` (if no other helper functions remain in use).

## 6. tests: loader/file

- [x] 6.1 Rename `opm/helper/loader/file/release_test.go` test cases from `TestLoadReleaseFile_*` to `TestLoadReleasePackage_*`. Materialize a package directory (one `release.cue`) instead of a standalone file. Adjust assertions.
- [x] 6.2 Add `TestLoadReleasePackage_MultiFile` — a package with `release.cue` + `values.cue` in one CUE package — assert the unified result is returned and apiVersion is detected.
- [x] 6.3 Add `TestLoadReleasePackage_RegistryOption` — exercise `LoadOptions{Registry: ...}` against a fixture release that imports a module from a local registry. (Reuse the pattern from `flow_integration_test.go` if the fixture exists.) — Implemented as `TestLoadReleasePackage_RegistryOptionAccepted` (smoke test: confirms the option is plumbed without breaking the load); the full registry round trip stays in `opm/kernel/flow_integration_test.go` where `skipUnlessRegistry` already gates on the local registry.
- [x] 6.4 Update `TestLoadModulePackage_*` to pass the new `LoadOptions` parameter.
- [x] 6.5 Delete the `TestLoadValuesFile_*` test file (if separate) or test cases (if mixed into another file). The auto-unwrap behavior is now covered by `Kernel.LoadSourceFromFile` tests. — No such file existed; the only caller was `Kernel.LoadValuesFile`, exercised in `kernel_test.go` and removed under 7.2.

## 7. tests: kernel

- [x] 7.1 Rename `TestKernel_LoadReleaseFile_Parity` to `TestKernel_LoadReleasePackage_Parity`. Materialize a release directory; assert kernel wrapper matches the free function.
- [x] 7.2 Delete `TestKernel_LoadValuesFile_Parity` — replaced by source-loader tests.
- [x] 7.3 Add or extend a `TestKernel_LoadSourceFromFile_AutoUnwrapsValuesField` test verifying the auto-unwrap branch still triggers.
- [x] 7.4 Add a `TestKernel_LoadSourceFromFile_PassesThroughWithoutValuesField` test verifying the fallback branch.
- [x] 7.5 Update `flow_integration_test.go:68` (the "LoadModulePackage does not accept a Registry override" comment) — registry override now works for module loads too. Drop the workaround if no longer needed.

## 8. fixtures

- [x] 8.1 Audit existing release fixtures under `opm/helper/loader/file/testdata/` and any other test fixture dirs. Confirm each fixture file's `package` declaration is consistent within its directory so the package load succeeds. — No `testdata/` exists under `opm/helper/loader/file/`; release tests build fixtures inline via `t.TempDir()`. Real-world fixtures (`testdata/modules/web_app`, `modules/opm_platform`) are loaded by the kernel-level flow tests and are module/platform packages, not release packages.
- [x] 8.2 If any fixture needs a renamed package directive to land in a clean package layout, fix it. Document in fixture README if one exists. — N/A: no fixtures needed renaming.

## 9. spec deltas + sync

- [x] 9.1 `openspec validate unify-loaders-as-packages --strict` passes.
- [x] 9.2 After implementation merges, run `openspec sync` to fold the delta specs into `openspec/specs/helper-packages/spec.md` and `openspec/specs/kernel-runtime/spec.md`.

## 10. docs and changelog

- [x] 10.1 CHANGELOG entry under the next MAJOR: removal of `LoadReleaseFile` + `LoadValuesFile`; addition of `LoadReleasePackage`; `LoadModulePackage` signature change.
- [x] 10.2 Update `opm/helper/doc.go` if it enumerates loader entry points. — File describes loader/file at the subpackage level only; no concrete entry-point names listed, no update needed.
- [x] 10.3 Update `opm/kernel/doc.go` if it references the removed wrappers. — Updated the one-Kernel-per-goroutine example to pass `loaderfile.LoadOptions{}`.
- [x] 10.4 Edit the Unreleased CHANGELOG entry under `add-release-synth-helper` so the line *"Mirrors how `Kernel.LoadReleaseFile` is the recommended entry point for the file-driven path"* reads `Kernel.LoadReleasePackage` instead. Both sections land in the same MAJOR; keep the file coherent.

## 11. synth cross-references

- [x] 11.1 Update `opm/kernel/synth.go` godoc on `Kernel.SynthesizeRelease`: replace `[Kernel.LoadReleaseFile]` with `[Kernel.LoadReleasePackage]`.
- [x] 11.2 Audit `opm/helper/synth/doc.go` and `opm/helper/synth/release.go` for any remaining `LoadReleaseFile` / `LoadValuesFile` references; rewrite to the new names if found. (Current audit: none. Re-verify before merge.) — Re-verified: clean.
- [x] 11.3 Verify `opm/kernel/synth_test.go` and `opm/kernel/flow_synth_integration_test.go` do not depend on the removed loader wrappers. (Current audit: they do not. Re-verify before merge.) — Re-verified: clean.

## 12. validation gates

- [x] 12.1 `task fmt`
- [x] 12.2 `task vet`
- [x] 12.3 `task lint` (`0 issues.`)
- [x] 12.4 `task test` (all packages pass)
- [x] 12.5 `openspec validate unify-loaders-as-packages --strict` (`Change 'unify-loaders-as-packages' is valid`)
