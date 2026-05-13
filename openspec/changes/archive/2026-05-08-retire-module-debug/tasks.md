## 1. Codebase Sweep

- [x] 1.1 `grep -rn "ModuleDebug" opm/ apis/` and review every hit
- [x] 1.2 Remove any exported Go type named `ModuleDebug` if present
- [x] 1.3 Remove any `LoadModuleDebug` function or equivalent loader path
- [x] 1.4 Remove any kernel input field, parameter, or option referring to debug as a top-level artifact

## 2. Version Binding Cleanup

- [x] 2.1 Inspect `opm/api/v1alpha2/` (from `add-multi-apiversion-support`); confirm no `DecodeModuleDebugMetadata` or similar exists
- [x] 2.2 Confirm `Paths()` exposes `DebugValues` (a path within Module.Package) but NOT a separate `Debug` artifact path; if mistake exists, correct it
- [x] 2.3 Add or confirm a unit test that asserts the binding has no debug-artifact decoder

## 3. Documentation Updates

- [x] 3.1 Update `library/README.md` to list the three kernel artifact types (Module, ModuleRelease, Platform) and explicitly state `#ModuleDebug` is retired
- [x] 3.2 Update `opm/module/` godoc to mention `debugValues` is a Module field, not an artifact
- [x] 3.3 Add a CHANGELOG entry noting the retirement and pointing to the migration recipe (read `Module.Package`'s `debugValues` field directly)

## 4. Cross-Reference Verification

- [x] 4.1 Verify `enhancements/001-kernel-redesign-around-platform/01-problem.md` mentions `#ModuleDebug` retirement
- [x] 4.2 Verify `enhancements/001-kernel-redesign-around-platform/03-decisions.md` D6 is consistent with the implemented behavior

## 5. Validation

- [x] 5.1 Run `task fmt`
- [x] 5.2 Run `task vet`
- [x] 5.3 Run `task lint`
- [x] 5.4 Run `task test`
- [x] 5.5 Final grep: `grep -rn "ModuleDebug" opm/ apis/ enhancements/ openspec/` returns only the umbrella's expected mentions
