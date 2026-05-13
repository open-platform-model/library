## 1. Input Structs

- [x] 1.1 Define `ValidateInput`, `MatchInput`, `PlanInput`, `CompileInput` in `opm/kernel/inputs.go` (or equivalent)
- [x] 1.2 Each struct exposes the minimal fields its phase needs (per spec); add godoc for each field
- [x] 1.3 Decide whether to share a base struct (e.g. embedded `commonInput`) — current preference: keep flat for clarity

## 2. Result Types

- [x] 2.1 Define `MatchPlan` (re-export from `opm/render/`), `PlanResult`, `CompileResult` in `opm/kernel/results.go`
- [x] 2.2 Add type alias `type ModuleResult = CompileResult` in `opm/render/` for backward compatibility
- [x] 2.3 Document each result's fields with godoc

## 3. Phase Methods on Kernel

- [x] 3.1 Implement `(k *Kernel) Validate(ctx context.Context, in ValidateInput) error`; call `validate.Config` internally
- [x] 3.2 Implement `(k *Kernel) Match(ctx context.Context, in MatchInput) (*MatchPlan, error)`; call existing match logic
- [x] 3.3 Implement `(k *Kernel) Plan(ctx context.Context, in PlanInput) (*PlanResult, error)`; run Validate + Match + dry-execute
- [x] 3.4 Implement `(k *Kernel) Compile(ctx context.Context, in CompileInput) (*CompileResult, error)`; full pipeline
- [x] 3.5 Implement `(k *Kernel) DetectAPIVersion(v cue.Value) (apiversion.Version, error)` delegating to `apiversion.Detect`
- [x] 3.6 Implement `(k *Kernel) Finalize(v cue.Value) (cue.Value, error)` delegating to `render.FinalizeValue`

## 4. Render Package Rename

- [x] 4.1 `git mv opm/render/process_module.go opm/render/compile_module.go`
- [x] 4.2 Rename function: `func ProcessModuleRelease(...)` → `func CompileModuleRelease(...)`
- [x] 4.3 Add `func ProcessModuleRelease(...)` as a thin alias delegating to `CompileModuleRelease` with `// Deprecated:` doc comment
- [x] 4.4 Rename `*ModuleResult` to `*CompileResult` at definition site; add `type ModuleResult = CompileResult` alias
- [x] 4.5 Update internal references in `opm/render/` to use new names
- [x] 4.6 Confirm `opm/render/module.go` (the per-render execution helper) is unchanged

## 5. Wrapper Method Updates (slice 01 wrappers)

- [x] 5.1 Update `(k *Kernel) ProcessModuleRelease(...)` from slice 01 to be a deprecated alias delegating to the new `Compile` method (or to `render.CompileModuleRelease`)
- [x] 5.2 Confirm godoc on every wrapper method is current

## 6. Tests

- [x] 6.1 Add `opm/kernel/phase_test.go` covering `Validate`, `Match`, `Plan`, `Compile` methods against existing fixtures
- [x] 6.2 Test that `Plan` does not produce `Rendered` values; only summaries
- [x] 6.3 Test that `Compile` matches behavior of the prior `ProcessModuleRelease` flow on every fixture (golden test)
- [x] 6.4 Test that the deprecated `ProcessModuleRelease` alias still works
- [x] 6.5 Add a kernel-level utility test for `DetectAPIVersion` and `Finalize`

## 7. Documentation

- [x] 7.1 CHANGELOG entry: rename `Render`/`Process` → `Compile`; new phase methods on Kernel; deprecation aliases
- [x] 7.2 Update `library/README.md` Quick Start to use `k.Compile`
- [x] 7.3 Update umbrella enhancement (`enhancements/001-kernel-redesign-around-platform/`) to confirm Compile naming consistency
- [x] 7.4 `opm/kernel/doc.go` package doc gains a section "Phase methods" explaining Validate / Match / Plan / Compile

## 8. Validation

- [x] 8.1 Run `task fmt`
- [x] 8.2 Run `task vet`
- [x] 8.3 Run `task lint`
- [x] 8.4 Run `task test`
- [x] 8.5 Run `task check`
