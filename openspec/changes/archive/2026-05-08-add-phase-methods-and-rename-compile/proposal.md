## Why

Slice 01 introduced the `Kernel` struct with wrapper methods that delegate to existing free functions. The phase boundaries (`Validate`, `Match`, `Plan`, `Compile`) are not yet exposed as first-class methods — consumers must compose them themselves from the wrappers. The current pipeline entry point is `ProcessModuleRelease` (rendering the full pipeline) plus internal helpers; the verb "Render" is a misfit for a transform that lowers a declarative OPM model into platform-neutral resource values.

This is slice 06 of the kernel-redesign umbrella ([001-kernel-redesign-around-platform](../../../enhancements/001-kernel-redesign-around-platform/README.md)). It exposes the four phase methods on `*Kernel` and renames the rendering verb to `Compile` end-to-end.

## What Changes

- Add four phase methods on `*Kernel`:
  - `Validate(ctx, ValidateInput) error` — Tier-2 schema validation only.
  - `Match(ctx, MatchInput) (*MatchPlan, error)` — component ↔ transformer matching; produces a plan without executing transformers.
  - `Plan(ctx, PlanInput) (*PlanResult, error)` — full pipeline up to "what would compile produce" without finalization commitments.
  - `Compile(ctx, CompileInput) (*CompileResult, error)` — full pipeline; the terminal output.
- Rename internal `Render` / `ProcessModuleRelease` flow to `Compile` end-to-end:
  - `opm/render/process_module.go` → `opm/render/compile_module.go`.
  - `render.ProcessModuleRelease` → `render.CompileModuleRelease`. The old name remains as a `// Deprecated:` alias delegating to the new name.
  - `render.NewModule` and `render.Module` (the runtime helper struct) keep their names for now — they describe a per-module render context, not the pipeline verb.
  - `*ModuleResult` → `*CompileResult`. Old name remains as a type alias.
- Define small input structs: `ValidateInput`, `MatchInput`, `PlanInput`, `CompileInput`. Each carries the Module / ModuleRelease / values / Provider (today; Platform after slice 08) as appropriate.
- Update utility methods: `(k *Kernel) DetectAPIVersion(v cue.Value) (apiversion.Version, error)` and `(k *Kernel) Finalize(v cue.Value) (cue.Value, error)`.
- This is a MINOR change — additive methods on the existing `Kernel` type. The rename keeps the old names as deprecated aliases, so MAJOR is avoided.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `kernel-runtime`: Adds the four phase methods (`Validate`, `Match`, `Plan`, `Compile`) and the utility methods (`DetectAPIVersion`, `Finalize`). Renames `Render`-flavored internal symbols to `Compile`.

## Impact

- **`opm/kernel/`** — adds phase methods and input/output types.
- **`opm/render/`** — `ProcessModuleRelease` renamed to `CompileModuleRelease`; old name retained as deprecated alias. Internal file rename. `*ModuleResult` aliased to `*CompileResult`.
- **Downstream consumers** — no breaking change. They MAY migrate from `loader/module/render` free-function composition to the new phase methods; the rename keeps old names callable.
- **Documentation** — `library/README.md` and the umbrella enhancement adopt "Compile" terminology. The terminology change is the most visible part of this slice.
- **Constitution Principle IV (Composability via Stable Contracts)** — adds new public surface; preserves old. Eligible for a future MAJOR-version cleanup.
- **Constitution Principle VII (YAGNI)** — `Plan` ships only because every frontend has at least one use case (CLI: `plan` subcommand; operator: status preview; XR: speculative composition).
