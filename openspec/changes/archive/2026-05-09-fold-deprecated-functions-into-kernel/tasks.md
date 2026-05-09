## 1. pkg/kernel — Add Canonical Validate Implementation

- [x] 1.1 Create `pkg/kernel/validate.go` containing `(*Kernel).ValidateConfig` and `(*Kernel).ValidateConfigPartial` plus the unexported helpers `runValidate`, `appendSchemaErrors`, `walkDisallowed`, `fieldNotAllowedError` (with `Position`/`InputPositions`/`Error`/`Path`/`Msg` methods), and `normalizeFieldPath`. Include the `fieldNotAllowed` constant. Match the current `pkg/validate/config.go` semantics exactly: zero `cue.Value` → success-no-op; concrete-required toggle distinguishes the two entry points.
- [x] 1.2 Add godoc on both methods explaining Tier-2 vs Tier-1 framing and the zero-value contract. Cross-reference `pkg/helper/values.ValidateAndUnify` from `ValidateConfigPartial`'s godoc.
- [x] 1.3 Create `pkg/kernel/validate_test.go` covering: zero-value-as-no-op (success); schema-error returned as `*oerrors.ConfigError` with correct `Context` and `Name`; field-not-allowed reported via the closed-schema walk; partial-vs-full distinction (a missing required field passes partial but fails full). Tests construct a `kernel.New()` and call `k.ValidateConfig` / `k.ValidateConfigPartial`.

## 2. pkg/kernel — Add Canonical Process Implementation

- [x] 2.1 Create `pkg/kernel/process.go` containing `(*Kernel).ProcessModuleRelease` plus the unexported helper `bestEffortReleaseName`. Body mirrors current `module.ParseModuleRelease`: best-effort name → binding lookup → `#config` schema lookup at `paths.Module.Config` → call internal `runValidate(..., requireConcrete=true)` (or `k.ValidateConfig`) → fill validated value into spec at `paths.Values` → assert `spec.Validate(cue.Concrete(true))` → decode metadata via `b.DecodeReleaseMetadata` → return `*module.Release`. (Side change: exported `module.releaseMetadataFromAPI` as `module.ReleaseMetadataFromAPI` so the kernel can use the converter; the helper was previously private to pkg/module.)
- [x] 2.2 Add `(*Kernel).ParseModuleRelease` in the same file as a deprecated thin alias: `// Deprecated: use (*Kernel).ProcessModuleRelease.` body simply returns `k.ProcessModuleRelease(ctx, spec, mod, values)`.
- [x] 2.3 Add godoc on `ProcessModuleRelease` explaining the four-step pipeline and that `cue.Value{}` for `values` is the no-values path.

## 3. pkg/kernel — Move Compile Implementation Home

- [x] 3.1 Create `pkg/kernel/compile.go` containing the implementation currently in `pkg/compile/compile_module.go`. Extract the helper logic into an unexported function (e.g. `compileModuleRelease`) so `(*Kernel).Compile` in `pkg/kernel/phases.go` can call it directly without the `compile.CompileModuleRelease` indirection. Preserve every existing return-path / error-wrapping semantic from `compile.CompileModuleRelease`.
- [x] 3.2 Update `(*Kernel).Compile` in `pkg/kernel/phases.go` to call the new internal helper instead of `compile.CompileModuleRelease`. Drop the `//nolint:staticcheck // SA1019: ...` exemption.

## 4. pkg/kernel — Drop Wrapper Methods and Imports

- [x] 4.1 In `pkg/kernel/wrappers.go`, remove `(*Kernel).ValidateConfig` (now lives in `validate.go`) and `(*Kernel).ParseModuleRelease` (replaced by the canonical method + alias in `process.go`). Drop the `"github.com/open-platform-model/library/pkg/validate"` import. Drop the `"github.com/open-platform-model/library/pkg/module"` import only if no other wrapper still uses it (the `ComposePlatform`, `NewModuleFromValue`, `NewReleaseFromValue` wrappers still need it).
- [x] 4.2 In `pkg/kernel/phases.go`, drop the `"github.com/open-platform-model/library/pkg/validate"` import. Replace the `validate.Config(schema, in.Values, "module", name)` call inside `(*Kernel).Validate` with the kernel's internal `runValidate(schema, in.Values, "module", name, true)` helper (or call `k.ValidateConfig` directly). Drop the `//nolint:staticcheck` exemption.
- [x] 4.3 In `pkg/kernel/wrappers.go`, drop the `"github.com/open-platform-model/library/pkg/compile"` import only if no other wrapper still uses it (`Finalize` lives in `phases.go` and uses `compile.FinalizeValue`, so the import stays in `phases.go`; `wrappers.go` likely loses it). (Note: wrappers.go didn't import compile to begin with — Finalize and Compile both lived in phases.go and the new `compileModuleRelease` lives in compile.go which imports the package directly. Verified clean.)

## 5. pkg/helper/values — Widen KernelOwner

- [x] 5.1 In `pkg/helper/values/values.go`, widen the `KernelOwner` interface to include `ValidateConfigPartial(schema cue.Value, values cue.Value, contextLabel, name string) (cue.Value, *oerrors.ConfigError)`. Update the interface's godoc to explain why both methods are required.
- [x] 5.2 Replace the `validate.ConfigPartial(schema, l.Value, "values", l.Name)` call inside `ValidateAndUnify` with `owner.ValidateConfigPartial(schema, l.Value, "values", l.Name)`.
- [x] 5.3 Drop the `"github.com/open-platform-model/library/pkg/validate"` import from `pkg/helper/values/values.go`. Add `oerrors "github.com/open-platform-model/library/pkg/errors"` if not already present (interface signature names the type).
- [x] 5.4 Update `pkg/helper/values/doc.go` and any other godoc that references `validate.ConfigPartial` to reference `(*Kernel).ValidateConfigPartial` instead.

## 6. pkg/kernel — Tests

- [x] 6.1 In `pkg/kernel/kernel_test.go`, collapse `TestKernel_ValidateConfig_Parity` and `TestKernel_ValidateConfig_Parity_Error` to single-path tests that exercise only `k.ValidateConfig` (the deprecated free function it compared against no longer exists). Drop the `validate` import. (Compile parity tests `TestKernel_Compile_Parity_VersionMismatch` / `TestKernel_Compile_Parity_UnknownVersion` and `TestKernel_GoroutineIsolation`'s call to `compile.CompileModuleRelease` were also collapsed; `compile` import dropped. The cross-file `TestKernel_Compile_MatchesCompileModuleRelease` parity test in `phase_test.go` was removed too.)
- [x] 6.2 In `pkg/kernel/kernel_test.go`, collapse `TestKernel_ParseModuleRelease_Parity` to a single-path test against `k.ProcessModuleRelease`. Add a small follow-up test confirming `k.ParseModuleRelease(...)` returns identical results to `k.ProcessModuleRelease(...)` (the deprecated alias contract).
- [x] 6.3 Confirm `pkg/kernel/phase_test.go` does not import `pkg/validate` (it should not, since it exercised `k.Validate` not the free function); update if needed. (Confirmed: phase_test.go has no `pkg/validate` import; the `pkg/compile` import is retained for `compile.CompileResult`/`compile.ModuleResult` types unrelated to this change.)

## 7. Delete Deprecated Surface

- [x] 7.1 Delete `pkg/validate/config.go` and `pkg/validate/config_test.go`. Run `rm -r pkg/validate/` to remove the directory.
- [x] 7.2 Delete `pkg/module/parse.go`. Confirm no remaining file in `pkg/module/` references `ParseModuleRelease` or the unexported `bestEffortReleaseName` helper.
- [x] 7.3 Delete `pkg/compile/compile_module.go`. Confirm `pkg/compile/` still compiles (it should — `CompileResult`, `Match`, `FinalizeValue`, `ComponentSummary`, `MatchPlan` types live in other files in that package).

## 8. Constitution and Documentation

- [x] 8.1 Edit `openspec/config.yaml`: in the `context` block's "Separation of Concerns" list, remove the line `- pkg/validate/: configuration validation helpers.`. Adjust surrounding wording if needed so the list still reads cleanly. (Also updated commit-scope and task-grouping references from `validate` to `kernel`.)
- [x] 8.2 Add a CHANGELOG entry under "Unreleased — next MAJOR" with three sub-sections: **Added** for `(*Kernel).ValidateConfigPartial` and `(*Kernel).ProcessModuleRelease`; **Deprecated** for `(*Kernel).ParseModuleRelease`; **Removed (BREAKING)** with a migration table showing `validate.Config` → `(*Kernel).ValidateConfig`, `validate.ConfigPartial` → `(*Kernel).ValidateConfigPartial`, `module.ParseModuleRelease` → `(*Kernel).ProcessModuleRelease`, `compile.CompileModuleRelease` → `(*Kernel).Compile`. Note that `pkg/validate/` is deleted entirely.
- [x] 8.3 Note in the CHANGELOG that `pkg/helper/values.KernelOwner` widened by one method (`ValidateConfigPartial`) and that downstream test fakes implementing the interface must add the method.

## 9. Validation Gates

- [x] 9.1 Run `task fmt`
- [x] 9.2 Run `task vet`
- [x] 9.3 Run `task lint`
- [x] 9.4 Run `task test`
- [x] 9.5 Run `task check`
- [x] 9.6 Run `grep -rn "pkg/validate\|validate\.Config\|module\.ParseModuleRelease\|compile\.CompileModuleRelease" --include="*.go"` and confirm no matches remain (other than CHANGELOG entries). Verified: only matches are historical comments (`pkg/compile/compile_test.go:48` notes coverage moved; the `module.ParseModuleRelease` godoc comment in `phases.go:148` was updated to reference `Kernel.ProcessModuleRelease`).
- [x] 9.7 Run `grep -rn "//nolint:staticcheck // SA1019" pkg/kernel/` and confirm the wrapper-related exemptions are gone. Verified: remaining exemptions are unrelated to this change — `kernel_test.go:267` deliberately exercises the deprecated `ParseModuleRelease` alias contract; `phase_test.go:303` tests the unrelated `compile.ModuleResult` type alias; `compile.go:62` calls `compile.NewModule` which is on a separate deprecation arc.
