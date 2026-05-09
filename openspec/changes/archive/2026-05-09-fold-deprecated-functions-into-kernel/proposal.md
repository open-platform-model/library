## Why

Slices 01–11 set up `Kernel` as the public anchor type but kept the canonical implementation of four functions in their original packages (`pkg/validate/`, `pkg/module/parse.go`, `pkg/compile/compile_module.go`), with `Kernel` methods as thin wrappers carrying `//nolint:staticcheck` exemptions. The wrapper-with-deprecation step was always transitional — this slice closes that arc by moving the implementations into `pkg/kernel`, deleting the deprecated free functions and the `pkg/validate/` package entirely, so `Kernel` is the *only* place the canonical impls live.

A second motivation: `module.ParseModuleRelease` undersells what the function does (validate values, fill spec, check concreteness, decode metadata). The kernel-side equivalent is renamed to `Kernel.ProcessModuleRelease` to match. The old name lives on for one cycle as a deprecated alias on the kernel.

## What Changes

- **BREAKING**: `pkg/validate/` package deleted in full. Callers (none remain in this repo after the migration) must use `Kernel.ValidateConfig` / `Kernel.ValidateConfigPartial`.
- **BREAKING**: `module.ParseModuleRelease` (free function) deleted. Migration: `Kernel.ProcessModuleRelease`.
- **BREAKING**: `compile.CompileModuleRelease` (free function) deleted. Migration: `Kernel.Compile`.
- **Added**: `Kernel.ValidateConfigPartial` — promotes the previously package-private partial-validation entry point onto the kernel surface.
- **Added**: `Kernel.ProcessModuleRelease` — canonical name; carries the impl moved out of `pkg/module/parse.go`.
- **Deprecated**: `Kernel.ParseModuleRelease` — kept for one cycle as a thin alias delegating to `Kernel.ProcessModuleRelease`. Marked with `// Deprecated:` doc comment.
- **Interface widening**: `pkg/helper/values.KernelOwner` gains a `ValidateConfigPartial` method so the helper can call the kernel without re-introducing the package import cycle (`helper/values → kernel → helper/values`).
- **Constitution edit**: `openspec/config.yaml` package list drops `pkg/validate/`.
- **Spec deltas**: `kernel-runtime` removes the wrapper-pattern requirement and the `validate.Config` / `ParseModuleRelease` scenarios that name free functions; replaces them with kernel-method scenarios. `values-validation` modifies the `ValidateAndUnify` requirement's reference from `validate.ConfigPartial` to `Kernel.ValidateConfigPartial` via the `KernelOwner` interface.

## Capabilities

### New Capabilities

(none — this change consolidates existing surface, it does not introduce new capability boundaries)

### Modified Capabilities

- `kernel-runtime`: the wrapper-pattern requirement is removed; the four canonical impls now live on the kernel. Adds the `Kernel.ValidateConfigPartial` and `Kernel.ProcessModuleRelease` surface; deprecates `Kernel.ParseModuleRelease`.
- `values-validation`: the helper SHALL call `KernelOwner.ValidateConfigPartial`, not the deleted `validate.ConfigPartial`. The `KernelOwner` interface widens by one method.

## Impact

- **Affected packages**:
  - `pkg/kernel/` — gains `validate.go`, `process.go`, `compile.go` (impl files); `wrappers.go` shrinks (loses validate + parse + compile wrapper methods, retains the helper-shaped wrappers).
  - `pkg/validate/` — deleted.
  - `pkg/module/parse.go` — deleted.
  - `pkg/compile/compile_module.go` — deleted (impl absorbed into `pkg/kernel/compile.go`).
  - `pkg/helper/values/values.go` — `KernelOwner` interface widens; helper calls `owner.ValidateConfigPartial(...)` instead of `validate.ConfigPartial(...)`.
  - Tests move with their impl: `pkg/validate/config_test.go` → `pkg/kernel/validate_test.go`; parity-against-deprecated-fn tests in `pkg/kernel/kernel_test.go` collapse to single-path tests against the kernel methods.
- **Downstream consumers** (CLI, opm-operator, future Crossplane fn) using the old free functions must migrate. `Kernel.ParseModuleRelease` alias buys one cycle for the rename; the package deletions are immediate.
- **SemVer**: MAJOR. Three deletions, one method rename (with deprecated alias), one interface widening.
- **Helper layer is preserved**. `pkg/helper/loader/file/*`, `pkg/helper/platform/Compose`, `pkg/helper/values/ValidateAndUnify`, and the `*FromValue` constructors remain helper-shaped (taking `CueContextOwner` / `KernelOwner`) per the slice 02 / 05 design intent. This change touches only the four explicitly-deprecated free functions.
