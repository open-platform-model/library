## Context

The kernel-redesign enhancement (umbrella `enhancements/001-kernel-redesign-around-platform/`) ran a multi-slice migration that introduced `Kernel` as the public anchor type. Each slice that touched a free function preserved backward compatibility by keeping the existing free function (marked `// Deprecated:`) and adding a thin `Kernel` method that wrapped it. The wrapper carried a `//nolint:staticcheck // SA1019: kernel method wraps the deprecated free function` exemption.

Today the wrappers are still in place for four functions:

| Free function (canonical impl)        | Wrapper                          |
|---------------------------------------|----------------------------------|
| `validate.Config`                     | `Kernel.ValidateConfig`          |
| `validate.ConfigPartial`              | (none — called only by helper)   |
| `module.ParseModuleRelease`           | `Kernel.ParseModuleRelease`      |
| `compile.CompileModuleRelease`        | `Kernel.Compile`                 |

The wrapper-with-deprecation step was always transitional. This change finishes it: the impls move into `pkg/kernel`, the free functions are deleted, and `pkg/validate/` (which exists only to host two of those functions) is deleted with the package.

A second concern was raised separately: `module.ParseModuleRelease` undersells what the function does (validate values, fill spec, check concreteness, decode metadata). The kernel-side equivalent is renamed `Kernel.ProcessModuleRelease`. The old name lives on for one cycle as a deprecated method-level alias to keep the rename migration soft.

The constraint forcing the design: `pkg/helper/values` calls `validate.ConfigPartial`. If `validate` becomes a wrapper around `kernel`, the import graph closes a cycle (`helper/values → validate → kernel → helper/values`). The only way to delete `pkg/validate/` cleanly is to widen the `KernelOwner` interface in `pkg/helper/values` so the helper calls back into the kernel through the interface rather than through a sibling package.

## Goals / Non-Goals

**Goals:**

- Delete `pkg/validate/` in full, including tests.
- Delete `pkg/module/parse.go` and `pkg/compile/compile_module.go`.
- Move the four canonical impls into `pkg/kernel/` as kernel-internal files.
- Promote `ValidateConfigPartial` to a public `Kernel` method.
- Rename `Kernel.ParseModuleRelease` → `Kernel.ProcessModuleRelease` with a deprecated alias for one cycle.
- Widen `pkg/helper/values.KernelOwner` to include `ValidateConfigPartial`.
- Remove all `//nolint:staticcheck // SA1019: …wraps the deprecated free function` exemptions from `pkg/kernel`.
- Update `openspec/config.yaml` constitution to drop the `pkg/validate/` line from the package-boundary list.
- Spec deltas for `kernel-runtime` and `values-validation`.

**Non-Goals:**

- Folding the helper-shaped functions (`loaderfile.Load*`, `compile.FinalizeValue`, `module.New*FromValue`, `platform.NewPlatformFromValue`, `helperplatform.Compose`, `helpervalues.ValidateAndUnify`) into the kernel. Those take `CueContextOwner` / `KernelOwner` interfaces by design (slice 02 / 05), so downstream consumers can use them without dragging the whole kernel package. This change preserves that boundary.
- Removing the `Kernel.ParseModuleRelease` deprecated alias. That happens in a follow-up cycle.
- Folding `compile.NewModule` (the deprecated constructor inside `pkg/compile/module.go`). That belongs to a different cleanup stream.
- Touching `pkg/loader/` (the legacy directory predating `pkg/helper/loader/file/`). Out of scope.

## Decisions

### Decision 1: Hard rename for `ParseModuleRelease` → `ProcessModuleRelease` with a kernel-side deprecated alias

**Choice.** The free function `module.ParseModuleRelease` is deleted outright. On the kernel, `ProcessModuleRelease` is the canonical name; `ParseModuleRelease` becomes a thin alias method marked `// Deprecated: use Kernel.ProcessModuleRelease`.

**Rationale.** The free function has no consumers in this repo besides the kernel wrapper and a parity test. Outside consumers use the kernel method. Keeping the deprecated alias on the kernel (rather than the deleted free function) means the migration story for downstream code is:

```diff
- rel, err := k.ParseModuleRelease(ctx, spec, mod, values)
+ rel, err := k.ProcessModuleRelease(ctx, spec, mod, values)
```

… with the old call still compiling for one cycle.

**Alternatives considered:**

- *Hard rename, no alias.* Cleaner but forces every downstream caller to migrate in lock-step with the package deletions. Rejected to keep the breaking surface narrow.
- *Add `ProcessModuleRelease`, keep both indefinitely.* Carries two names for the same operation forever — exactly the kind of API bloat Principle VII discourages. Rejected.

### Decision 2: Widen `KernelOwner` in `pkg/helper/values` rather than re-route via a third shared package

**Choice.** The `KernelOwner` interface in `pkg/helper/values/values.go` grows one method:

```go
type KernelOwner interface {
    CueContext() *cue.Context
    ValidateConfigPartial(schema cue.Value, values cue.Value, contextLabel, name string) (cue.Value, *oerrors.ConfigError)
}
```

The helper's per-layer validation calls `owner.ValidateConfigPartial(...)` instead of `validate.ConfigPartial(...)`.

**Rationale.** This is the minimum surface that breaks the cycle. The helper still doesn't import `pkg/kernel` directly — it depends only on the interface. `*kernel.Kernel` automatically satisfies the wider interface because the new method already exists on it.

**Alternatives considered:**

- *Move the partial-validation impl into a new shared internal package both `kernel` and `helper/values` import.* Adds a third package solely to dodge a cycle, which contradicts Principle VII. Rejected.
- *Inline a private partial-validation implementation in `helper/values`.* Duplicates impl across the two packages. Rejected.
- *Have `helper/values` accept `*kernel.Kernel` directly instead of an interface.* Forces the cycle and contradicts the slice 05 design choice that the helper layer remains kernel-agnostic. Rejected.

### Decision 3: Three impl files under `pkg/kernel/` — `validate.go`, `process.go`, `compile.go`

**Choice.** Each canonical impl gets its own file in `pkg/kernel/`, named for the operation rather than the source package:

- `pkg/kernel/validate.go` — `ValidateConfig`, `ValidateConfigPartial`, plus the unexported helpers (`appendSchemaErrors`, `walkDisallowed`, `fieldNotAllowedError`, `normalizeFieldPath`).
- `pkg/kernel/process.go` — `ProcessModuleRelease` (and the unexported `bestEffortReleaseName` helper). `Kernel.ParseModuleRelease` (deprecated alias) lives here too so the rename is co-located.
- `pkg/kernel/compile.go` — the `Compile` impl currently in `pkg/compile/compile_module.go`. The kernel `Compile` method that exists in `pkg/kernel/phases.go` calls into this file's internal helper rather than the deleted `compile.CompileModuleRelease`.

`pkg/kernel/wrappers.go` shrinks: it loses `ValidateConfig`, `ValidateConfigPartial` (it never had this one), `ParseModuleRelease`, and any `Compile`-related delegation. It retains the helper-shaped wrappers (`LoadModulePackage`, `LoadReleaseFile`, `LoadValuesFile`, `LoadPlatformFile`, `NewPlatformFromValue`, `ComposePlatform`, `NewModuleFromValue`, `NewReleaseFromValue`, `ValidateAndUnify`) per Non-Goal 1.

**Rationale.** One file per operation reads better than one big `noncanon.go`. File names match the canonical method names rather than the deleted package names.

**Alternatives considered:**

- *Inline the impls into `phases.go` / `wrappers.go`.* `phases.go` is already the home for the four phase methods; pushing impl into it bloats the phase-orchestration file. Rejected.
- *Single `pkg/kernel/canonical.go`.* Reads as a dumping ground. Rejected.

### Decision 4: Tests move with their impl

**Choice.** Test files relocate alongside their implementation:

- `pkg/validate/config_test.go` → `pkg/kernel/validate_test.go` (rewritten to use `kernel.New()` + `k.ValidateConfig` / `k.ValidateConfigPartial`).
- The two `TestKernel_ValidateConfig_Parity*` tests in `pkg/kernel/kernel_test.go` collapse to single-path tests against `k.ValidateConfig` (the parity-against-deprecated-fn assertions vanish because the deprecated function is gone).
- `TestKernel_ParseModuleRelease_Parity` in `pkg/kernel/kernel_test.go` likewise collapses; an additional small test confirms `Kernel.ParseModuleRelease` (the deprecated alias) and `Kernel.ProcessModuleRelease` produce identical results.

**Rationale.** Tests live next to the code they exercise. The parity tests existed to give confidence the kernel wrapper matched the canonical impl during the wrapper era — once both names point at the same body, parity is structural and the test collapses to a single-path check on the canonical method.

### Decision 5: Constitution edit is in scope

**Choice.** `openspec/config.yaml`'s context block lists `pkg/validate/` as one of the focused-package boundaries. That line is removed in the same change.

**Rationale.** The constitution describes the package boundaries; deleting a package without updating the constitution leaves it out of sync. The slice that defined the wrapper-pattern requirement (slice 01) didn't add the line — it was already there from earlier work — but the requirement that referenced `pkg/validate/` did get added in slice 01's spec. Both are updated now.

## Risks / Trade-offs

- **Risk: Downstream consumers (cli/, opm-operator/) import `pkg/validate` directly.** → Mitigation: search the workspace before merge; the four-symbol surface (`validate.Config`, `validate.ConfigPartial`, `*ConfigError`, `*MultiSourceError`) has clear migration targets on the kernel. The `Kernel.ParseModuleRelease` alias additionally buys a cycle for parse-side migrations. Document the migration matrix in the CHANGELOG.
- **Risk: Wider `KernelOwner` breaks test fakes in `pkg/helper/values/`.** → Mitigation: the existing tests already use `*kernel.Kernel` (no fake implementations of `KernelOwner` exist in this repo). If downstream tests fake `KernelOwner`, those fakes must grow the new method — same compile-time signal pattern Go uses for any interface widening.
- **Risk: Importing both `oerrors` and `cue.Value` directly into `pkg/kernel/validate.go` increases the kernel package's surface API.** → Trade-off accepted; this is the natural consequence of moving the canonical impl home.
- **Risk: One-cycle deprecated alias for `ParseModuleRelease` invites confusion about which name to use.** → Mitigation: godoc on `Kernel.ParseModuleRelease` says explicitly `// Deprecated: use Kernel.ProcessModuleRelease`; CHANGELOG calls out the rename and the removal target (next breaking cycle).
