# Tasks: library-platform-source

## opm/module

- [ ] 1. Broaden `Source` (`opm/module/source.go`): add `Pkg string` (package dir relative to `Root`, empty = `.`), rewrite the doc comment to cover both modes (overlay tree; on-disk tree when `Overlay == nil`) and the artifacts that carry it. `HasSource()` untouched.
- [ ] 2. Add `Instance.Source *Source` (`opm/module/instance.go`) with a doc comment stating it is nil for value-constructed instances and naming the two stamping sites (synthesis, directory acquire). Test: `NewInstanceFromValue` leaves it nil.

## opm/platform

- [ ] 3. Add `type Source = module.Source` and `Platform.Source *Source` (`opm/platform/platform.go`), doc comments per the delta spec. Test: `NewPlatformFromValue` leaves it nil.

## opm/helper/synth

- [ ] 4. Change `Instance` to also return `*module.Source` (`Root` = staged root, `Pkg` = `synthPkgDir`, `Overlay` = the augmented clone from `buildOverlay`); nil on every error path. Update in-repo callers and tests; assert the returned overlay contains the module files plus `instance.cue` (and `values.cue` when values supplied) and does not alias the module's own overlay map.

## opm/kernel

- [ ] 5. `SynthesizeInstance` stamps the synth-returned tree onto `inst.Source` after `ProcessModuleInstance` succeeds. Test: synthesized instance carries a non-nil `Source` with the reserved `Pkg`; failure paths return no instance as before.
- [ ] 6. Add `AcquirePlatformFromDir` (new `opm/kernel/acquire.go`): `loaderfile.LoadPlatformPackage` + `NewPlatformFromValue` + `Source{Root: absDir}`. Tests: happy path carries `Source`; shape-gate and missing-dir errors wrap the existing loader sentinels; registry override threads through.
- [ ] 7. Add `AcquireInstanceFromDir`: `loaderfile.LoadInstancePackage` + `ProcessModuleInstance(spec, module.Module{}, cue.Value{})` + `Source{Root: absDir}`. Tests: happy path (use an existing concrete instance fixture dir); non-concrete package surfaces the validation error; loader sentinels propagate.

## Verification

- [ ] 8. Workspace grep: no direct `synth.Instance(` callers in `cli/` or `opm-operator/` (design risk); record the result in the PR description if non-zero.
- [ ] 9. `task check` (fmt, vet, lint, test) passes; confirm no existing test needed a behavioral edit (signature-following edits only).
