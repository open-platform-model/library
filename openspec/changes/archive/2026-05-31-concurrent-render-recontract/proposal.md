## Why

The compile pipeline sources its `*cue.Context` from the **materialized platform** (`mp.Package.Context()` at `kernel/compile.go:40`; `r.platform.Package.Context()` at `compile/module.go:112`) instead of from the caller Kernel. Today that is harmless — single-Kernel callers materialize the platform in their own context, so the platform context *is* the caller context. But it funnels every render through the platform's one context (serializing concurrent renders) and relies on `cue.Value.Context()`, which CUE v0.17 deprecates.

The `opm-operator` rewrite onto the kernel needs to render many `ModuleRelease`s concurrently against one `*MaterializedPlatform` that is materialized **once** per Platform-CR generation and reused (see the spike dossier, `opm-operator/KERNEL-CONCURRENCY-SPIKE-RESULTS.md`, and `adr/002`). A keystone spike proved the per-goroutine-Kernel + one-shared-read-only-platform model is race-clean and ~2.5× faster — **on CUE v0.17**. This change lands the kernel half of that model that is correct and verifiable **today on v0.16**: thread the caller-Kernel's context through the compile pipeline so renders build in the caller's context and read the shared platform as input. Pinning v0.17 and enabling actual cross-Kernel concurrency is a gated follow-up.

## What Changes

- The compile pipeline sources its `*cue.Context` from the **caller Kernel**, not from the materialized platform. `compileModuleRelease` threads `k.cueCtx` into `compile.FinalizeValue` and into the Execute path, replacing both `mp.Package.Context()` and `r.platform.Package.Context()`.
- `compile.NewModule` gains a leading `*cue.Context` parameter so Execute builds the transformer-context (`#context.*`) and finalized values in the caller's context. **BREAKING** (exported `opm/compile` signature) — but `Kernel.Compile` is the only caller in-tree; downstream consumers (`cli`, `opm-operator`) go through `Kernel`, never `compile` directly. Migration recipe in `MIGRATIONS.md`.
- `compile.Match` is **unchanged**: its always-unify already uses the component (caller) value as the `Unify` receiver, so the result resolves in the caller's context — the fill direction the v0.17 model wants. No matching behavior changes.
- `adr/002` status `Proposed → Accepted` (the spike concluded; this change implements its v0.16-landable half).

### Out of scope (gated follow-up change)

- Pinning CUE → v0.17 (alpha; the production `go.mod` stays at v0.16.1).
- Rewriting the **Goroutine Safety Contract** / **MaterializedPlatform** specs to assert "a `*MaterializedPlatform` is safe to share across goroutines and Kernels." That is **false on v0.16** and stays unchanged here; it becomes true only under v0.17.
- The permanent concurrent `-race` regression test (needs a v0.17-compatible in-memory catalog; the published `catalogs/opm@v0.4.0` does not parse under v0.17).
- A full sweep of remaining `Value.Context()` usages across `materialize` / `schema`.

These are **blocked** on (1) a stable/accepted-risk CUE v0.17 and (2) re-published v0.17-compatible catalogs — see `design.md` § Deferred.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `kernel-runtime`: the compile pipeline MUST source its `*cue.Context` from the caller Kernel (`k.CueContext()`), not from the materialized platform. New requirement parallel to the existing "SynthesizeRelease uses the Kernel's cue.Context".
- `platform-materialization`: clarify that a `*MaterializedPlatform.Package` is consumed by Compile as **read-only input** (the fill argument / cross-read source), not as the owner of the build context.

## Impact

- **Affected `opm/` packages:** `compile` (`module.go`, `NewModule` signature), `kernel` (`compile.go`, `phases.go`).
- **SemVer:** behavior-preserving refactor for `Kernel` callers; technically a breaking change to the exported `compile.NewModule` signature (kernel-internal caller only). Document in `MIGRATIONS.md`.
- **Downstream:** none functional — `cli` and `opm-operator` use the `Kernel` surface, which is unchanged. The operator continues to ship on v0.16 + a render mutex and drops the mutex only when the v0.17 follow-up lands; nothing here is on its critical path.
- **CUE version:** unchanged (v0.16.1). No dependency bump in this change.
