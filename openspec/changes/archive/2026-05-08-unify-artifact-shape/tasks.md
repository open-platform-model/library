## 1. Type Refactor — Module

- [x] 1.1 Edit `pkg/module/module.go`: redefine `Module` to `{ APIVersion apiversion.Version; Metadata *ModuleMetadata; Package cue.Value }`; remove `Spec` and `Config` fields
- [x] 1.2 Update godoc on `Module` to state that `Package` is source of truth and `Metadata` is a cache
- [x] 1.3 Remove or relocate any `Module` methods that depended on removed fields

## 2. Type Refactor — Release

- [x] 2.1 Edit `pkg/module/release.go`: redefine `Release` to `{ APIVersion apiversion.Version; Metadata *ReleaseMetadata; Package cue.Value }`; remove `Module`, `Spec`, `Values` fields
- [x] 2.2 Move `Release.MatchComponents()` (or equivalent) to read from `Package` via the binding
- [x] 2.3 Update godoc on `Release` to state that `Package` is source of truth

## 3. Constructor Helpers

- [x] 3.1 Add `func NewModuleFromValue(k *kernel.Kernel, v cue.Value) (*Module, error)` to `pkg/module/module.go`
- [x] 3.2 Add `func NewReleaseFromValue(k *kernel.Kernel, v cue.Value) (*Release, error)` to `pkg/module/release.go`
- [x] 3.3 Each constructor uses `apiversion.Detect`, `api.Lookup`, then the binding's metadata decoder
- [x] 3.4 Stamp `APIVersion` on the returned struct from the detected version
- [x] 3.5 Set `Package` directly from the input `cue.Value` (do not re-evaluate)

## 4. Internal Call-Site Migration — pkg/render/

- [x] 4.1 Rewrite `pkg/render/match.go` reads of removed fields to use `Package.LookupPath(binding.Paths()....)`
- [x] 4.2 Rewrite `pkg/render/execute.go` reads of `mod.Spec`, `rel.Spec`, `rel.Values` accordingly
- [x] 4.3 Rewrite `pkg/render/finalize.go` reads accordingly
- [x] 4.4 Rewrite `pkg/render/process_module.go` reads accordingly
- [x] 4.5 Rewrite `pkg/render/module.go` reads accordingly
- [x] 4.6 Confirm no direct field reads of removed fields remain (`grep -n '\.Spec' pkg/render/`)

## 5. Internal Call-Site Migration — pkg/module/, pkg/validate/

- [x] 5.1 Rewrite `pkg/module/parse.go` (`ParseModuleRelease` and helpers) to read schema/values via the binding
- [x] 5.2 Rewrite `pkg/validate/config.go` to consume schema via the binding when called from kernel-internal paths
- [x] 5.3 Update `module.bestEffortReleaseName` and `module.decodeReleaseMetadata` to consume the binding rather than direct field access

## 6. Loader Glue

- [x] 6.1 Confirm `pkg/loader/` continues to return raw `cue.Value` (loader does not produce typed artifacts; the constructors do)
- [x] 6.2 Add `(k *Kernel) NewModuleFromValue` / `NewReleaseFromValue` thin wrappers on `pkg/kernel/` for consumer ergonomics

## 7. Test Migration

- [x] 7.1 Update `pkg/module/` tests that constructed `Module` / `Release` literals to use constructors or set `Package` directly
- [x] 7.2 Update `pkg/render/` tests that depended on removed fields
- [x] 7.3 Update `pkg/validate/` tests
- [x] 7.4 Ensure all fixtures continue to validate against the v1alpha2 binding

## 8. Documentation and Migration Notes

- [x] 8.1 Add a CHANGELOG entry documenting the breaking field removals and the migration recipe (before/after pairs)
- [x] 8.2 Update `library/README.md` Quick Start to use constructors
- [x] 8.3 Cross-reference this slice from `enhancements/001-kernel-redesign-around-platform/02-design.md`

## 9. Validation

- [x] 9.1 Run `task fmt`
- [x] 9.2 Run `task vet`
- [x] 9.3 Run `task lint`
- [x] 9.4 Run `task test`; investigate every failure; do not paper over
- [x] 9.5 Run `task check`
