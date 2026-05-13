## Why

The kernel currently exposes its functionality as a loose collection of free functions across `opm/loader/`, `opm/module/`, `opm/render/`, `opm/validate/`. Every entry point takes `*cue.Context` as a parameter, leaking CUE plumbing into every downstream consumer (CLI, operator, planned Crossplane fn). Cross-cutting dependencies (logger, tracer, clock) cannot be added without breaking every signature. There is no single mental anchor a downstream binary can attach to and call "the kernel."

This is slice 01 of the kernel redesign umbrella ([001-kernel-redesign-around-platform](../../../enhancements/001-kernel-redesign-around-platform/README.md)). It establishes the `Kernel` struct as the single public face, owning its `cue.Context` and its DI dependencies. No behavior changes — existing free functions are retained as deprecated aliases that delegate to `Kernel` methods until later slices remove them.

## What Changes

- Introduce `opm/kernel/` package with the `Kernel` struct, `kernel.New(opts ...Option)` constructor, and functional options for logger / tracer / clock.
- `Kernel` owns one `cue.Context` for its lifetime; never appears in public method signatures.
- Add `k.CueContext()` accessor for advanced cases (tests, programmatic CUE construction).
- Existing `loader.LoadModulePackage`, `loader.LoadReleaseFile`, `loader.LoadValuesFile`, `loader.LoadProvider`, `module.ParseModuleRelease`, `render.NewModule`, `render.ProcessModuleRelease` gain `Kernel`-method equivalents that source `cue.Context` from the Kernel. Existing functions remain callable; they delegate to the kernel-method form internally and are marked `// Deprecated:` with a pointer to the new method.
- `Kernel` is documented as **not goroutine-safe across method calls**. Callers needing concurrency construct one Kernel per goroutine.
- This is a MINOR change — additive only. No existing function signature is removed.

## Capabilities

### New Capabilities

- `kernel-runtime`: The `Kernel` struct, its construction, configuration, and lifecycle. Houses cross-cutting dependencies (logger, tracer, clock). All future kernel-facing slices modify this capability.

### Modified Capabilities

None.

## Impact

- **`opm/kernel/` (new)** — `Kernel` struct, `Option` type, `New` constructor, accessor methods.
- **`opm/loader/`, `opm/module/`, `opm/render/`, `opm/validate/`** — each gains a `Kernel`-method wrapper that delegates. Existing free functions stay; gain `// Deprecated:` doc comments pointing to the new methods.
- **Downstream consumers (CLI, operator)** — no code change required. They continue to call existing functions; they MAY migrate to Kernel methods incrementally.
- **Constitution Principle IV (Composability via Stable Contracts)** — adds new public surface; no breaking change. Current free functions remain part of the SemVer contract until a future MAJOR removes them.
