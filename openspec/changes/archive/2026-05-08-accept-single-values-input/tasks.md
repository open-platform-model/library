## 1. validate.Config Signature Change

- [x] 1.1 Change `validate.Config` signature from `(schema cue.Value, values []cue.Value, context, name string)` to `(schema cue.Value, values cue.Value, context, name string)`
- [x] 1.2 Remove the slice merge loop from `validate.Config` body
- [x] 1.3 Adjust the "no values" branch to test for zero-value `cue.Value` (not slice length)
- [x] 1.4 Keep the existing schema-error and fieldNotAllowed walking logic unchanged

## 2. ParseModuleRelease Signature Change

- [x] 2.1 Change `module.ParseModuleRelease` signature: replace `values []cue.Value` parameter with `values cue.Value`
- [x] 2.2 Pass `values` directly to `validate.Config`; remove any slice handling
- [x] 2.3 Confirm the FillPath / concrete validation steps still work with the unified value

## 3. Migration Helper

- [x] 3.1 Add `validate.UnifyAndValidate(values []cue.Value) cue.Value` performing the previous slice-merge loop
- [x] 3.2 Mark the helper `// Deprecated: use opm/helper/values for layering and pass the unified result to validate.Config`
- [x] 3.3 Add a unit test that confirms `UnifyAndValidate` produces the same unified value the previous `Config` slice form would have produced

## 4. Kernel Wrapper Updates

- [x] 4.1 Update `(k *Kernel) ValidateConfig(...)` (slice 01 wrapper) to take `values cue.Value`
- [x] 4.2 Update `(k *Kernel) ParseModuleRelease(...)` to take `values cue.Value`
- [x] 4.3 Confirm wrapper godoc still references the underlying functions correctly

## 5. Test Migration

- [x] 5.1 Update `opm/validate/` tests that previously built `[]cue.Value` literals to build a single unified `cue.Value` (via `UnifyAndValidate` for parity, or directly)
- [x] 5.2 Update `opm/module/` tests for `ParseModuleRelease` accordingly
- [x] 5.3 Add a regression test that confirms zero-value `cue.Value{}` is accepted as "no values"
- [x] 5.4 Add a parity test confirming new single-value `Config` produces equivalent output to old slice form (using `UnifyAndValidate` to bridge)

## 6. Documentation and Migration Notes

- [x] 6.1 CHANGELOG entry documenting the signature breaking change with before/after recipe
- [x] 6.2 Update `library/README.md` Quick Start to use the single-value form
- [x] 6.3 Cross-reference slice 05 in the CHANGELOG as the recommended migration target

## 7. Validation

- [x] 7.1 Run `task fmt`
- [x] 7.2 Run `task vet`
- [x] 7.3 Run `task lint`
- [x] 7.4 Run `task test`
- [x] 7.5 Run `task check`
