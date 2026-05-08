## Why

The current matcher (`pkg/render/match.go`) walks a flat `*provider.Provider` transformer list and pairs each component against transformers via `requiredLabels`, `requiredResources`, `requiredTraits`, `optionalTraits`. With catalog enhancement [014-platform-construct](../../../../catalog/enhancements/014-platform-construct/) retiring `#Provider` and replacing it with `#Platform` — a register that publishes `#composedTransformers` and `#matchers.{resources,traits}` as CUE-computed views — the matcher's input contract no longer fits. Slice 08 introduced the `Platform` Go type and binding paths; this slice rewires the matcher to consume them.

This is slice 09 of the kernel-redesign umbrella ([001-kernel-redesign-around-platform](../../../enhancements/001-kernel-redesign-around-platform/README.md)) and the highest-risk slice. It is the only slice that materially changes runtime matching behavior. By the time it lands, slices 01, 02, 06, 08 have established the foundation; this slice is purely about matching logic.

`#Claim` and the corresponding `claims` sub-map of `#matchers` are deferred per umbrella scope. The matcher walks Resources and Traits only.

## What Changes

- Rewrite `pkg/render/match.go` to consume `Platform.#composedTransformers` and `Platform.#matchers.{resources, traits}` instead of a `*provider.Provider`. Match phase walks consumer Module's components, collects required Resource/Trait FQNs, looks each up in `Platform.#matchers` via the binding paths from slice 08.
- New return shape: `*MatchPlan` retains its purpose but adopts a new internal model — matched pairs are `(component, transformer)` where the transformer is identified by FQN against `Platform.#composedTransformers`.
- **BREAKING** to `pkg/render/match.go` API. Old `Match(components, p)` signature removed; replaced by `Match(in MatchInput) (*MatchPlan, error)` (already shipped in slice 06; slice 09 changes implementation, not signature).
- Phase input structs (`MatchInput`, `PlanInput`, `CompileInput`) flip the `Platform` field from optional to required. The `Provider` field is removed.
- `pkg/provider/` package is deleted in this slice. Final retirement.
- `loader.LoadProvider` (now `pkg/helper/loader/file/provider.go`) is removed. The deprecation shim at `pkg/loader/` removes its `LoadProvider` re-export.
- The matcher does NOT use catalog 014's `#PlatformMatch` CUE construct. Per umbrella decision (Q1: match in Go), the kernel implements the walk in Go but mirrors the CUE construct's semantics.
- This is a MAJOR change. Bump kernel module version. Multiple packages retired (`pkg/provider/`, `pkg/loader/file/provider.go`).

## Capabilities

### New Capabilities

- `platform-matching`: The matcher's algorithm and input contract. Defines how the kernel walks consumer Module demand against `Platform.#matchers`, surfaces matched / unmatched / ambiguous, and returns the plan.

### Modified Capabilities

- `kernel-runtime`: Phase input structs change — `Platform` becomes required, `Provider` removed.
- `artifact-types`: `Provider` is retired; the artifact set narrows to Module, ModuleRelease, Platform.
- `helper-packages`: `loader.LoadProvider` removed.

## Impact

- **`pkg/render/match.go`** — algorithm rewritten. Input model swaps Provider for Platform. Output `*MatchPlan` shape preserved at the public surface; internal data shape adapted.
- **`pkg/render/execute.go`** — transformer execution now resolves transformers by FQN from `Platform.#composedTransformers` rather than indexing a provider's transformer list.
- **`pkg/render/module.go`** — `render.Module` (the per-render execution helper) takes `*Platform` instead of `*Provider`. `NewModule(p, runtimeName)` → `NewModule(plat, runtimeName)`.
- **`pkg/provider/`** — directory and package deleted.
- **`pkg/helper/loader/file/provider.go`** — deleted. Shim at `pkg/loader/LoadProvider` removed.
- **`pkg/kernel/`** — phase input structs lose `Provider` field; `Platform` field becomes required. Wrapper methods relating to Provider removed.
- **Downstream consumers** — `cli` and `opm-operator` migrate from constructing `*Provider` to constructing `*Platform`. The migration is non-trivial: previously they loaded a single Provider artifact; now they compose a Platform from a registry of Modules. Slice 10 (`add-platform-composition-helper`) ships the `pkg/helper/platform/Compose` helper to make this a one-liner.
- **Constitution Principle V (CUE-Native Module Resolution)** — fully aligns: matching consumes CUE-computed views; the kernel does no flat-list walking.
- **Constitution Principle VI (SemVer)** — MAJOR bump. Multiple removed packages.
