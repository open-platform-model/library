## 1. Codebase Sweep

- [ ] 1.1 `grep -rn "ModuleDebug" pkg/ apis/` and review every hit
- [ ] 1.2 Remove any exported Go type named `ModuleDebug` if present
- [ ] 1.3 Remove any `LoadModuleDebug` function or equivalent loader path
- [ ] 1.4 Remove any kernel input field, parameter, or option referring to debug as a top-level artifact

## 2. Version Binding Cleanup

- [ ] 2.1 Inspect `pkg/api/v1alpha2/` (from `add-multi-apiversion-support`); confirm no `DecodeModuleDebugMetadata` or similar exists
- [ ] 2.2 Confirm `Paths()` exposes `DebugValues` (a path within Module.Package) but NOT a separate `Debug` artifact path; if mistake exists, correct it
- [ ] 2.3 Add or confirm a unit test that asserts the binding has no debug-artifact decoder

## 3. Documentation Updates

- [ ] 3.1 Update `library/README.md` to list the three kernel artifact types (Module, ModuleRelease, Platform) and explicitly state `#ModuleDebug` is retired
- [ ] 3.2 Update `pkg/module/` godoc to mention `debugValues` is a Module field, not an artifact
- [ ] 3.3 Add a CHANGELOG entry noting the retirement and pointing to the migration recipe (read `Module.Package`'s `debugValues` field directly)

## 4. Cross-Reference Verification

- [ ] 4.1 Verify `enhancements/001-kernel-redesign-around-platform/01-problem.md` mentions `#ModuleDebug` retirement
- [ ] 4.2 Verify `enhancements/001-kernel-redesign-around-platform/03-decisions.md` D6 is consistent with the implemented behavior

## 5. Validation

- [ ] 5.1 Run `task fmt`
- [ ] 5.2 Run `task vet`
- [ ] 5.3 Run `task lint`
- [ ] 5.4 Run `task test`
- [ ] 5.5 Final grep: `grep -rn "ModuleDebug" pkg/ apis/ enhancements/ openspec/` returns only the umbrella's expected mentions
