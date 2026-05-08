## Why

Slice 04 (`accept-single-values-input`) narrowed the kernel to take a single pre-unified `cue.Value` for values. That left a gap: who layers and who produces source-positioned diagnostics? Frontends that skip layering get poor error UX, and each frontend reinventing layering means three versions of the same logic across `cli`, `opm-operator`, and the planned Crossplane fn.

This is slice 05 of the kernel-redesign umbrella ([001-kernel-redesign-around-platform](../../../enhancements/001-kernel-redesign-around-platform/README.md)). It introduces `pkg/helper/values/` — a kernel-shipped, opt-in helper that performs Tier-1 source-positioned validation per layer and returns a single unified `cue.Value` ready for the kernel. The kernel keeps its Tier-2 correctness safety net (slice 04). Together they realize the two-tier validation pattern (umbrella D1, D5).

Once shipped, the temporary `validate.UnifyAndValidate` helper from slice 04 is removed.

## What Changes

- Add new package `pkg/helper/values/` with:
  - `type Layer struct { Name string; Source string; Value cue.Value }` — a single labeled values source.
  - `type Stack []Layer` — ordered layer stack; later layers override earlier.
  - `func ValidateAndUnify(k *kernel.Kernel, schema cue.Value, layers Stack) (cue.Value, *MultiSourceError)` — Tier-1 validates each layer independently, then unifies in order, then returns the unified value plus per-layer diagnostics.
- Add `MultiSourceError` carrying per-layer `*ConfigError` instances with source attribution (file/line position from CUE) and the layer name.
- Each frontend (CLI, operator, XR fn) constructs a `Stack` from its sources (CLI flag stack, K8s ConfigMap/Secret/CR overlay, composition input). The Kernel never sees layers — it sees the unified result.
- Remove `validate.UnifyAndValidate` (deprecated in slice 04) — frontends migrate to `pkg/helper/values`.
- This is a MINOR change to the public API. `pkg/helper/values/` is new; `validate.UnifyAndValidate` removal is the only deletion.

## Capabilities

### New Capabilities

- `values-validation`: The Tier-1 helper for layering and source-positioned diagnostics. Defines `Layer`, `Stack`, `ValidateAndUnify`, and `MultiSourceError`. The kernel's Tier-2 validation (slice 04) is the safety net; this helper is the user-facing diagnostic surface.

### Modified Capabilities

None.

## Impact

- **`pkg/helper/values/` (new)** — `Layer`, `Stack`, `ValidateAndUnify`, `MultiSourceError`.
- **`pkg/validate/`** — `UnifyAndValidate` is removed. `Config` is unchanged from slice 04.
- **`pkg/kernel/`** — optionally adds a thin convenience method `(k *Kernel) ValidateAndUnify(schema, layers)` delegating to the helper.
- **Downstream consumers** — recommended migration: replace `validate.UnifyAndValidate(vs)` with `helper/values.ValidateAndUnify(k, schema, stack)` and propagate the per-layer errors as appropriate (CLI: print with source; operator: surface in CRD status conditions; XR: surface in composition status).
- **Constitution Principle II (Type Safety)** — input contract gains diagnostic shape.
- **Constitution Principle VII (Simplicity)** — kernel does not gain layering policy; helper does. Each frontend is unchanged in behavior, gains better error messages via shared code.
