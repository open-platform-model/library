## Why

The kernel currently treats `#ModuleDebug` as a separate top-level construct. The new design (catalog 015's eight-slot Module shape) folds debug into a `debugValues` field on `Module` itself. Centralizing debug as a kernel input forces the kernel to know about debug semantics, which is a frontend policy (operator: never in prod; CLI: when `--debug` is set; XR: per-composition). Slice 04 of the kernel-redesign umbrella ([001-kernel-redesign-around-platform](../../../enhancements/001-kernel-redesign-around-platform/README.md)) has already moved values layering out of the kernel; this slice is the corresponding cleanup for debug.

This is slice 03. It removes the `#ModuleDebug` artifact concept from the kernel and documents `Module.debugValues` as the only debug surface. The frontend reads `debugValues` from the Module's package and decides whether to layer it into the values stack.

## What Changes

- Remove any kernel code that loaded, parsed, or rendered `#ModuleDebug` as a standalone artifact. (Note: the kernel does not currently expose a `ModuleDebug` Go type — this slice is largely codebase-search and docs cleanup, plus removal of any debug-specific paths from loaders and the binding.)
- Update kernel documentation (`library/README.md`, `pkg/module/` godoc, the umbrella enhancement) to clarify that `debugValues` is a Module field, not a separate artifact, and that layering it is a frontend concern.
- Confirm via grep that no kernel-internal call site special-cases debug.
- Confirm version binding (from `add-multi-apiversion-support`) does NOT carry a `Debug` path or decoder; if it does, remove it.
- This is a **PATCH** change for code (no Go API surface removed), but doc-level it sharpens the kernel's contract. If a `ModuleDebug` Go type does exist in some intermediate state and is removed here, treat as MINOR.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `artifact-types`: Tightens the contract — `Module`, `ModuleRelease`, `Platform` are the artifact types. `#ModuleDebug` is not a kernel artifact.

## Impact

- **`pkg/module/`** — confirm no `ModuleDebug` type exists; if it does, remove it.
- **`pkg/loader/`** — confirm no `LoadModuleDebug` or equivalent; remove if present.
- **`pkg/api/v1alpha2/`** (from `add-multi-apiversion-support`) — confirm binding does not expose a debug path or decoder; remove if it does.
- **`library/README.md`** — clarify `debugValues` is a Module field; layering is helper-side.
- **`enhancements/001-kernel-redesign-around-platform/02-design.md`** — already aligned; verify.
- **Downstream consumers** — `cli` and `opm-operator` that read `debugValues` continue to do so via `Module.Package.LookupPath(binding.Paths().DebugValues)` once that path is added (or via direct CUE lookup). No removal of an existing public type is expected.
