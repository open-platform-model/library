## 1. Input Structs

- [ ] 1.1 Define `ValidateInput`, `MatchInput`, `PlanInput`, `CompileInput` in `pkg/kernel/inputs.go` (or equivalent)
- [ ] 1.2 Each struct exposes the minimal fields its phase needs (per spec); add godoc for each field
- [ ] 1.3 Decide whether to share a base struct (e.g. embedded `commonInput`) — current preference: keep flat for clarity

## 2. Result Types

- [ ] 2.1 Define `MatchPlan` (re-export from `pkg/render/`), `PlanResult`, `CompileResult` in `pkg/kernel/results.go`
- [ ] 2.2 Add type alias `type ModuleResult = CompileResult` in `pkg/render/` for backward compatibility
- [ ] 2.3 Document each result's fields with godoc

## 3. Phase Methods on Kernel

- [ ] 3.1 Implement `(k *Kernel) Validate(ctx context.Context, in ValidateInput) error`; call `validate.Config` internally
- [ ] 3.2 Implement `(k *Kernel) Match(ctx context.Context, in MatchInput) (*MatchPlan, error)`; call existing match logic
- [ ] 3.3 Implement `(k *Kernel) Plan(ctx context.Context, in PlanInput) (*PlanResult, error)`; run Validate + Match + dry-execute
- [ ] 3.4 Implement `(k *Kernel) Compile(ctx context.Context, in CompileInput) (*CompileResult, error)`; full pipeline
- [ ] 3.5 Implement `(k *Kernel) DetectAPIVersion(v cue.Value) (apiversion.Version, error)` delegating to `apiversion.Detect`
- [ ] 3.6 Implement `(k *Kernel) Finalize(v cue.Value) (cue.Value, error)` delegating to `render.FinalizeValue`

## 4. Render Package Rename

- [ ] 4.1 `git mv pkg/render/process_module.go pkg/render/compile_module.go`
- [ ] 4.2 Rename function: `func ProcessModuleRelease(...)` → `func CompileModuleRelease(...)`
- [ ] 4.3 Add `func ProcessModuleRelease(...)` as a thin alias delegating to `CompileModuleRelease` with `// Deprecated:` doc comment
- [ ] 4.4 Rename `*ModuleResult` to `*CompileResult` at definition site; add `type ModuleResult = CompileResult` alias
- [ ] 4.5 Update internal references in `pkg/render/` to use new names
- [ ] 4.6 Confirm `pkg/render/module.go` (the per-render execution helper) is unchanged

## 5. Wrapper Method Updates (slice 01 wrappers)

- [ ] 5.1 Update `(k *Kernel) ProcessModuleRelease(...)` from slice 01 to be a deprecated alias delegating to the new `Compile` method (or to `render.CompileModuleRelease`)
- [ ] 5.2 Confirm godoc on every wrapper method is current

## 6. Tests

- [ ] 6.1 Add `pkg/kernel/phase_test.go` covering `Validate`, `Match`, `Plan`, `Compile` methods against existing fixtures
- [ ] 6.2 Test that `Plan` does not produce `Rendered` values; only summaries
- [ ] 6.3 Test that `Compile` matches behavior of the prior `ProcessModuleRelease` flow on every fixture (golden test)
- [ ] 6.4 Test that the deprecated `ProcessModuleRelease` alias still works
- [ ] 6.5 Add a kernel-level utility test for `DetectAPIVersion` and `Finalize`

## 7. Documentation

- [ ] 7.1 CHANGELOG entry: rename `Render`/`Process` → `Compile`; new phase methods on Kernel; deprecation aliases
- [ ] 7.2 Update `library/README.md` Quick Start to use `k.Compile`
- [ ] 7.3 Update umbrella enhancement (`enhancements/001-kernel-redesign-around-platform/`) to confirm Compile naming consistency
- [ ] 7.4 `pkg/kernel/doc.go` package doc gains a section "Phase methods" explaining Validate / Match / Plan / Compile

## 8. Validation

- [ ] 8.1 Run `task fmt`
- [ ] 8.2 Run `task vet`
- [ ] 8.3 Run `task lint`
- [ ] 8.4 Run `task test`
- [ ] 8.5 Run `task check`
