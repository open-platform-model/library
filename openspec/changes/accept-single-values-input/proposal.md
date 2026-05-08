## Why

`validate.Config(schema cue.Value, values []cue.Value, ...)` and `module.ParseModuleRelease(_, spec, mod, values []cue.Value)` accept a slice of values and unify them internally. The kernel bakes in one merge order; different frontends layer values differently (CLI: `-f` flag stack; operator: ConfigMap → Secret → CR overlay; XR: composition input). No frontend gets to express its layering policy cleanly.

This is slice 04 of the kernel-redesign umbrella ([001-kernel-redesign-around-platform](../../../enhancements/001-kernel-redesign-around-platform/README.md)). It changes the kernel's values input from `[]cue.Value` to a single, pre-unified `cue.Value`. Layering becomes a helper / frontend concern (slice 05 introduces `pkg/helper/values/` for source-positioned Tier-1 validation and unification). The kernel always re-validates the unified value as a Tier-2 correctness safety net.

## What Changes

- Change `validate.Config` signature from `(schema cue.Value, values []cue.Value, context, name string)` to `(schema cue.Value, values cue.Value, context, name string)`. The single `values` argument is the unified value the caller has already merged.
- Change `module.ParseModuleRelease` signature from `(_, spec cue.Value, mod Module, values []cue.Value)` to `(_, spec cue.Value, mod Module, values cue.Value)`. Same rationale.
- The kernel re-validates the unified value as Tier 2 — schema-correctness only, position-rich diagnostics are slice 05's concern.
- **BREAKING** to two function signatures. CLI / operator that previously passed `[]cue.Value` must merge before calling.
- Provide a temporary `validate.UnifyAndValidate(schema, values []cue.Value, ...)` helper that performs caller-side merge then calls the new single-value form. This lets downstream consumers migrate incrementally.
- This is a MAJOR change for `pkg/validate/` and `pkg/module/`. Bump kernel module version.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `kernel-runtime`: Tightens the kernel's input contract. The kernel now expects a single, pre-unified `values cue.Value`; layering policy is delegated to the frontend / helper.

## Impact

- **`pkg/validate/`** — `Config` signature changes. Internal merge loop is removed. Schema-validation logic remains.
- **`pkg/module/`** — `ParseModuleRelease` signature changes. The "Validate values, then fill into spec" sequence is preserved; only the values argument shape changes.
- **`pkg/kernel/`** — wrapper methods (slice 01) update to match.
- **Downstream consumers** — `cli` and `opm-operator` must merge values before calling. The temporary `UnifyAndValidate` helper provides a one-line migration; consumers migrate to source-positioned Tier-1 validation when slice 05 ships its `pkg/helper/values/` package.
- **Constitution Principle II (Type Safety)** — input contract sharpens.
- **Constitution Principle VII (Simplicity)** — kernel does less; layering is policy and lives outside the kernel.
