# Tasks: library-platform-source

## opm/module

- [x] 1. Broaden `Source` (`opm/module/source.go`): add `Pkg string` (package dir relative to `Root`, empty = `.`), rewrite the doc comment to cover both modes (overlay tree; on-disk tree when `Overlay == nil`) and the artifacts that carry it. `HasSource()` untouched.
- [x] 2. Add `Instance.Source *Source` (`opm/module/instance.go`) with a doc comment stating it is nil for value-constructed instances and naming the two stamping sites (synthesis, directory acquire). Test: `NewInstanceFromValue` leaves it nil.

## opm/platform

- [x] 3. Add `type Source = module.Source` and `Platform.Source *Source` (`opm/platform/platform.go`), doc comments per the delta spec. Test: `NewPlatformFromValue` leaves it nil.

## opm/helper/synth

- [x] 4. Change `Instance` to also return `*module.Source` (`Root` = staged root, `Pkg` = `synthPkgDir`, `Overlay` = the augmented clone from `buildOverlay`); nil on every error path. Update in-repo callers and tests; assert the returned overlay contains the module files plus `instance.cue` (and `values.cue` when values supplied) and does not alias the module's own overlay map.

## opm/kernel

- [x] 5. `SynthesizeInstance` stamps the synth-returned tree onto `inst.Source` after `ProcessModuleInstance` succeeds. Test: synthesized instance carries a non-nil `Source` with the reserved `Pkg`; failure paths return no instance as before.
- [x] 6. Add `AcquirePlatformFromDir` (new `opm/kernel/acquire.go`): `loaderfile.LoadPlatformPackage` + `NewPlatformFromValue` + `Source{Root: absDir}`. Tests: happy path carries `Source`; shape-gate and missing-dir errors wrap the existing loader sentinels; registry override threads through.
- [x] 7. Add `AcquireInstanceFromDir`: `loaderfile.LoadInstancePackage` + `ProcessModuleInstance(spec, module.Module{}, cue.Value{})` + `Source{Root: absDir}`. Tests: happy path (use an existing concrete instance fixture dir); non-concrete package surfaces the validation error; loader sentinels propagate.

## Verification

- [x] 8. Workspace grep: no direct `synth.Instance(` callers in `cli/` or `opm-operator/` (design risk); record the result in the PR description if non-zero.
- [x] 9. `task check` (fmt, vet, lint, test) passes; confirm no existing test needed a behavioral edit (signature-following edits only).
  - 2026-09-01: green after pinning `schema.DefaultSchemaModule` to core 2.0.0-alpha.6 on this branch (alpha.7, released the same day, reshapes `#Platform.#registry` and broke 25 materialize / flow / synth.Platform tests on clean `main`; the platform code follows the new shape in a later change). Every existing test edit in this change is signature-following only.
