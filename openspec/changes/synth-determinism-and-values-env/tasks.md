## 1. Deterministic synthesized metadata (opm/internal/synth)

- [x] 1.1 In `opm/internal/synth/render.go` `writeStringMap`, iterate the keys in ascending order (`slices.Sorted(maps.Keys(m))`) instead of ranging the map; verify `go build ./...`
- [x] 1.2 Add a synth test that calls `Instance` twice with four or more labels and four annotations inserted in different map orders and asserts the `instance.cue` bytes on both returned `Source.Overlay` trees are identical and list keys ascending; verify `go test ./opm/internal/synth` passes and fails when the sort is reverted
- [x] 1.3 `task check` green, then commit `fix(synth): emit labels and annotations in sorted key order`

## 2. Registry mapping for file-backed values sources (opm/kernel)

- [x] 2.1 Add an `env []string` parameter to `compileSource`, `compileSources`, `mergeSources` and `validateSources`; set `load.Config.Env` in the file-backed branch; pass `k.loadEnv()` from `ValidateConfigDetailed`, `loadInstanceWithValues`, `attributeValuesError` and `SynthesizeInstance` (both the merge and the attribution call in each); verify `go build ./...` and `go test ./opm/kernel` still pass
- [x] 2.2 Add a kernel test: serve a small module through `registrytest` (its constructor returns the mapping and sets the process `CUE_REGISTRY` to it), write a values file that imports the served module, then `t.Setenv("CUE_REGISTRY", schema.PublicRegistry)` so the served prefix routes only through `WithRegistry`, and assert a kernel built with `WithRegistry(mapping)` compiles it through `ValidateConfigDetailed`, as a trailing value to `AcquireInstanceFromDir`, and in `InstanceInput.Values` to `SynthesizeInstance`, while a kernel without the option fails at the import; verify `go test ./opm/kernel -run Source` passes and the positive cases fail when the `Env` line is removed
- [x] 2.3 Update the doc comments on `compileSource`, `LoadSourceFromFile`, `WithRegistry` and the package doc to name values-source compilation among the loads the mapping covers; verify `go vet ./...`
- [x] 2.4 `task check` green, then commit `fix(kernel): compile file-backed values sources with the kernel registry mapping`

## 3. One evaluate-and-shape-gate routine for registry modules (opm/internal/loader)

- [ ] 3.1 In `FetchModule`, replace the overlay wrap, `load.Instances`, single-instance check, build and `gate` with one `LoadDir(cueCtx, synthRoot, ".", overlay, env, ModuleSpec)` call after the empty-overlay refusal, keeping `verifyModuleIdentity`; verify `go test ./opm/internal/loader ./opm/kernel`
- [ ] 3.2 Confirm no test asserts the old "loading module package `<mv>`" wrap text (none does at alpha.30; re-grep), keep every sentinel assertion; add or extend a test that acquires the same malformed module from a registry and from a directory and asserts both wrap the same sentinel; verify `go test ./opm/internal/loader ./opm/kernel`
- [ ] 3.3 Update the `FetchModule` and `LoadDir` doc comments and the `internal/loader` line in CLAUDE.md to say registry modules build through `LoadDir`; verify `go vet ./...` and `grep -rn 'load.Instances(' opm --include='*.go' | grep -v _test` lists exactly four sites (kernel/source_loader.go, schema/loader.go, renderstage/stage.go, loader/load.go)
- [ ] 3.4 `task check` green, then commit `refactor(loader): build registry modules through LoadDir`
