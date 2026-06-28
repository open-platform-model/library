## 1. Shared module staging helper

- [x] 1.1 Extract `overlayFromSource` + `syntheticRoot` (currently in `opm/helper/loader/registry/module.go`) into a shared spot both `loader/registry` and `helper/synth` can call (e.g. `opm/helper/loader/internal/stage`), keeping behavior identical.
- [x] 1.2 Point `registry.LoadModulePackage` at the shared helper; confirm `registry-module-loading` tests still pass (`task test:run TEST=TestLoadModulePackage`-style).

## 2. Module source carrier

- [x] 2.1 Add a source carrier to `opm/module` (e.g. `type Source struct { Root string; Overlay map[string]load.Source }`) and an optional `Source *Source` field on `*module.Module`; document that it is nil unless the module was acquired with source.
- [x] 2.2 Add accessor/predicate (e.g. `(*Module).HasSource()`), unit-tested for the nil and populated cases.

## 3. Registry acquire entrypoint surfaces source

- [x] 3.1 Add `registry`-layer support to return the staged source alongside the built value (internal helper returning `(cue.Value, *module.Source, error)` or equivalent), without changing the existing `LoadModulePackage(...) (cue.Value, error)` signature.
- [x] 3.2 Add `Kernel.AcquireModuleFromRegistry(ctx, path, version) (*module.Module, error)` that builds a `*module.Module` with `Source` populated; keep `LoadModuleFromRegistry` unchanged (additive, MINOR).
- [x] 3.3 Tests: `AcquireModuleFromRegistry` returns a module whose `Source` is populated and is the same artifact used to build the value (no second fetch); existing `LoadModuleFromRegistry` behavior unchanged (registry-module-loading spec scenarios).

## 4. Synth construction rewrite (module-root staging)

- [x] 4.1 Rewrite `buildOverlay` / `Instance` (`opm/helper/synth/instance.go`) to overlay the instance source (+ rendered values) into the module's staged source under a reserved non-`_` synthetic subdir; use the module's staged root as `ModuleRoot`, so the module's own `cue.mod/module.cue` governs. (Added `BuildInstanceOverlayAt` to the file loader for subdir-package loading.)
- [x] 4.2 Rewrite `renderInstanceFile` to import core (from the module's deps) and the module's own package (resolves locally); delete `renderModuleFile` and the fabricated-deps helpers (`deps:` block, `normalizeVersion`, `corePath`-as-dep). Retain `moduleImportPath` / `moduleSnakeName` / `major` for addressing the module package. (`renderInstanceFile` body unchanged — already import-based; `renderModuleFile`/`normalizeVersion`/`synthLanguageVersion` deleted.)
- [x] 4.3 Make `synth.Instance` require `Module.Source`: return a clear typed error (`ErrMissingSource`) when it is nil (no registry fetch inside synth), per the new `instance-synthesis` precondition requirement.
- [x] 4.4 Keep `SchemaCache` required for `#ModuleInstance` confirmation, but stop using it to pin the synth build's core version (core now comes from the module's deps). Update `synth_test.go` helper (`publishSynthModule`) + the synth-package integration helper to acquire with source.

## 5. Downstream migration

- [ ] 5.1 **(DOWNSTREAM — blocked on library release)** Update `opm-operator` acquire path (`internal/moduleacquire/acquire.go:28` `LoadModuleFromRegistry`+`NewModuleFromValue` → `Kernel.AcquireModuleFromRegistry`) so `SynthesizeInstance` receives a source-carrying module. The operator pins `library v1.0.0-alpha.1` with no replace/go.work, so this lands only after the library publishes the new version (or via a temporary `replace ../library`). Recipe in `MIGRATIONS.md`.
- [ ] 5.2 **(DOWNSTREAM — blocked on library release)** Update CLI synth/acquire call sites the same way (if any); `cd cli && task build`. Same publish/replace dependency as 5.1.
- [x] 5.3 Add `MIGRATIONS.md` entry: new `AcquireModuleFromRegistry`, the `module.Source` field, the `ErrMissingSource` precondition, and the core-version-source change (D4).

## 6. Regression coverage + gates

- [x] 6.1 Add the library#31 regression test. **Key finding:** `registrytest`'s in-memory resolver walks transitive deps, so it CANNOT reproduce #31 (verified: an old fabricated-`{core, module}` build resolves the catalog under registrytest). The faithful, **non-vacuous** guard is `TestFlow_Redis_CatalogSubpackage_Regression` (real registry, real `redis@v0.1.6` importing `catalogs/opm/blueprints/workload`): proven to FAIL under the old path (`cannot find module providing package …`) and PASS under module-root construction. Also added `TestFlow_ImportedModule_CatalogSubpackageImport_SynthToCompile` as a hermetic construction/compile smoke test (honestly documented as not-by-itself-non-vacuous).
- [x] 6.2 Within-major core-skew safety (design D4) is exercised by the hermetic catalog-import test's `Kernel.Compile` step (instance built against the module's own core, compiled under the kernel's `SchemaCache` core; both `core@v1`). Cross-patch skew is not constructible in the single-core hermetic harness — documented in the test.
- [x] 6.3 Ran `task fmt`/`vet`/`lint` (0 issues) + full `task test` (green); confirmed no old-catalog paths in changed files; `openspec validate synth-instance-in-module-root --strict` passes.
