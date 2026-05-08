## 1. validate.Config Signature Change

- [ ] 1.1 Change `validate.Config` signature from `(schema cue.Value, values []cue.Value, context, name string)` to `(schema cue.Value, values cue.Value, context, name string)`
- [ ] 1.2 Remove the slice merge loop from `validate.Config` body
- [ ] 1.3 Adjust the "no values" branch to test for zero-value `cue.Value` (not slice length)
- [ ] 1.4 Keep the existing schema-error and fieldNotAllowed walking logic unchanged

## 2. ParseModuleRelease Signature Change

- [ ] 2.1 Change `module.ParseModuleRelease` signature: replace `values []cue.Value` parameter with `values cue.Value`
- [ ] 2.2 Pass `values` directly to `validate.Config`; remove any slice handling
- [ ] 2.3 Confirm the FillPath / concrete validation steps still work with the unified value

## 3. Migration Helper

- [ ] 3.1 Add `validate.UnifyAndValidate(values []cue.Value) cue.Value` performing the previous slice-merge loop
- [ ] 3.2 Mark the helper `// Deprecated: use pkg/helper/values for layering and pass the unified result to validate.Config`
- [ ] 3.3 Add a unit test that confirms `UnifyAndValidate` produces the same unified value the previous `Config` slice form would have produced

## 4. Kernel Wrapper Updates

- [ ] 4.1 Update `(k *Kernel) ValidateConfig(...)` (slice 01 wrapper) to take `values cue.Value`
- [ ] 4.2 Update `(k *Kernel) ParseModuleRelease(...)` to take `values cue.Value`
- [ ] 4.3 Confirm wrapper godoc still references the underlying functions correctly

## 5. Test Migration

- [ ] 5.1 Update `pkg/validate/` tests that previously built `[]cue.Value` literals to build a single unified `cue.Value` (via `UnifyAndValidate` for parity, or directly)
- [ ] 5.2 Update `pkg/module/` tests for `ParseModuleRelease` accordingly
- [ ] 5.3 Add a regression test that confirms zero-value `cue.Value{}` is accepted as "no values"
- [ ] 5.4 Add a parity test confirming new single-value `Config` produces equivalent output to old slice form (using `UnifyAndValidate` to bridge)

## 6. Documentation and Migration Notes

- [ ] 6.1 CHANGELOG entry documenting the signature breaking change with before/after recipe
- [ ] 6.2 Update `library/README.md` Quick Start to use the single-value form
- [ ] 6.3 Cross-reference slice 05 in the CHANGELOG as the recommended migration target

## 7. Validation

- [ ] 7.1 Run `task fmt`
- [ ] 7.2 Run `task vet`
- [ ] 7.3 Run `task lint`
- [ ] 7.4 Run `task test`
- [ ] 7.5 Run `task check`
