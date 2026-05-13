## Context

This slice is the foundation of the kernel-redesign umbrella ([001-kernel-redesign-around-platform](../../../enhancements/001-kernel-redesign-around-platform/README.md)). It introduces a single public type — `Kernel` — that owns its `cue.Context` and its dependency injection (logger, tracer, clock). Subsequent slices add phase methods (`Compile`, `Validate`, `Match`, `Plan`), reorganize helpers, and change input shapes — all on top of this foundation.

The change is intentionally additive. No free function signature is removed; no behavior changes; no test fixtures need updating. Downstream binaries (`cli`, `opm-operator`) continue to compile against the current API. Once this slice lands, slice 06 (`add-phase-methods-and-rename-compile`) builds the public phase methods on the same struct.

Coordination: `add-multi-apiversion-support` lands first. Slice 06 (`add-phase-methods-and-rename-compile`) builds the public phase methods on the same struct and is free to add a binding-registry hook then — slice 01 does not pre-bake one, since adding unused state without a consumer goes against the constitution's bias against speculative scaffolding.

## Goals / Non-Goals

**Goals:**

- Single `Kernel` type as the public mental anchor for downstream consumers.
- `cue.Context` owned by `Kernel`, never exposed in public method signatures (D8 in umbrella decisions).
- Functional options pattern for DI (`WithLogger`, `WithTracer`, `WithClock`).
- Backwards compatibility — every current public function continues to work.
- Documented goroutine-safety contract: not safe across method calls; one Kernel per goroutine.

**Non-Goals:**

- Removing existing free functions. Defer to a future MAJOR release.
- Adding phase methods (`Compile`, `Validate`, `Match`, `Plan`). That is slice 06.
- Reorganizing packages under `opm/helper/`. That is slice 07.
- Changing input/output shapes. Those are slices 02, 04, 08.

## Decisions

**Functional options vs. options struct.** Choose **functional options** (`kernel.New(WithLogger(l), WithTracer(t))`). Rationale: lets us add new options in MINOR releases without breaking call sites; standard idiomatic Go pattern. Options struct would require versioning the struct itself.

**`cue.Context` lifetime = Kernel lifetime.** Constructed in `New`, never replaced. If a caller needs to "reset" the kernel (e.g. drop CUE evaluation cache), they construct a new Kernel. There is no `k.Reset()`. Goes back to Constitution Principle I (Determinism) and umbrella D8.

**`k.CueContext()` accessor exists but is documented as advanced.** Most callers never need it. Tests that build CUE values programmatically need it. The doc comment says: "Use only when building `cue.Value`s outside the kernel; values from this context can be passed back into kernel methods. Most callers should not need this."

**Wrapper methods delegate to existing free functions.** During this slice, kernel methods are thin shims:

```go
func (k *Kernel) LoadModulePackage(_ context.Context, dirPath string) (cue.Value, error) {
    return loader.LoadModulePackage(k.cueCtx, dirPath)
}
```

This keeps both call paths exercised by existing tests; the eventual removal happens in a future MAJOR.

**No `Logger() *slog.Logger` accessor on Kernel.** Logging is for the kernel's internal use; consumers should not reach in to use the kernel's logger as their own. If a consumer needs a logger, they own one and inject it into the kernel.

**Clock interface is minimal.** `type Clock interface { Now() time.Time }`. No frequency, no timer creation. Render is intentionally not time-dependent today; this slot exists so future slices can be deterministic if rendering ever consults a clock.

## Risks / Trade-offs

**Risk — two ways to call the same operation.** During this slice and until a MAJOR, both `loader.LoadModulePackage(cueCtx, dir)` and `k.LoadModulePackage(ctx, dir)` work. Documentation must steer new consumers to the method form. Mitigation: `// Deprecated:` doc comment on every retained free function pointing to the method.

**Risk — `Kernel` accumulates fields too quickly.** Every later slice adds something to the struct (binding registry, render caches). Mitigation: keep fields private; expose via methods, not embedded structs.

**Risk — goroutine-safety footgun.** A user constructs one Kernel and shares it across N goroutines, runs into a CUE concurrency issue. Mitigation: explicit doc comment on the type and on `New`; example in the package doc showing one-Kernel-per-goroutine pattern; vet check or test that confirms documented contract.

**Trade-off — slight indirection cost.** Every method goes through the Kernel struct rather than a direct package call. Negligible at the latencies CUE evaluation operates at; not measurable.
