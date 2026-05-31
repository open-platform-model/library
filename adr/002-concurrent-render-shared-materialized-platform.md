# ADR-002: Concurrent rendering against a shared materialized platform (CUE v0.17)

## Status

Accepted — records the findings of the `spike-concurrent-render-v0170` change and gates a follow-on recontract change.

The v0.16-landable half of this decision is implemented by the `concurrent-render-recontract` OpenSpec change (`openspec/changes/concurrent-render-recontract/`): the compile pipeline now builds in the caller Kernel's `*cue.Context` and consumes the materialized platform as read-only input, removing both platform-sourced `Value.Context()` sites. The v0.17 pin, the Goroutine-Safety-Contract / MaterializedPlatform spec rewrites, and the permanent concurrent `-race` regression test remain a gated follow-up (blocked on a stable/accepted-risk v0.17 and re-published v0.17-parseable catalogs).

## Context

The `opm-operator` rewrite onto the kernel needs to render many `ModuleRelease`s concurrently against a single, long-lived `*MaterializedPlatform` — materialize once (per Platform-CR generation), reuse across all releases. Every other operator need is already satisfied by the library (Kernel lifetime, `sync.Once` schema cache, `Materialize` + concurrency-safe `LRU`). The render path is the one gap.

Two constraints block this on the shipped CUE version (`v0.16.1`):

1. **Same-context coupling.** A `*MaterializedPlatform`'s `Package` is built with its owning Kernel's `*cue.Context`. The compile pipeline takes its context from the platform (`opm/compile/module.go:112`: `r.platform.Package.Context()`) and `FillPath`s release components into platform transformers. When the release was built by a *different* Kernel, that is a cross-context combination.
2. **Single-threaded Kernel.** `kernel/doc.go`: "NOT safe for concurrent use … one Kernel per goroutine."

CUE `v0.17` is reported to relax both: cross-context `Unify`/`FillPath` becomes legal (`Value.Context()` deprecated as a consequence) and `cue.Value` reads become race-safe. But the verified v0.17 guarantee is reads-only; the OPM render path is construction-heavy (`FillPath`/`Unify`); "safe" does not imply "parallel"; and v0.17 is alpha. The spike measured the real behavior before committing.

**Spike evidence.** An isolated raw-CUE module (`.spike/crosscontext/`, 32 goroutines × 200 iters, `-race`):

- **v0.16.1:** concurrent cross-context `FillPath` panics `values are not from the same runtime`.
- **v0.17.0-alpha.1:** race-clean; a serial-vs-`RunParallel` benchmark shows ~2.5× throughput (saturating ~4 cores, allocator-bound at ~145 allocs/op) — genuine parallelism, not correctness-only.

**Kernel code analysis + full-flow probe.** The compile pipeline already sources its context from the platform, not the calling Kernel. That is harmless today (single-Kernel callers make the two contexts identical) but is wrong for the concurrent design: it would funnel every concurrent render through the platform's one shared context (serializing the work) and relies on the v0.17-deprecated `Value.Context()`. A build-tagged cross-kernel test (`opm/kernel/spike_crosskernel_test.go`, since removed) confirmed at the kernel level: on v0.16.1, a release built by `K1` compiled against a platform materialized by `K0` fails — surfacing in `compile.Match` → `unifyIntersection` (`opm/compile/match.go:263`, the always-unify step) and in Execute's `FillPath`. So the cross-context boundary lives at **both** the Match and Execute stages of the compile pipeline.

A second, independent finding emerged when running the full real-fixture flow on v0.17.0-alpha.1: the **library code compiles cleanly** on v0.17 (no removed-API breakage), **but the published catalog `opmodel.dev/catalogs/opm@v0.4.0` fails to parse** under v0.17's stricter parser (`missing ',' in argument list`). The real-fixture full-flow confirmation on v0.17 is therefore blocked at the fixture layer, not the kernel — the cross-context *mechanism* is proven by the raw-CUE keystone; the kernel end-to-end path on v0.17 needs a v0.17-compatible catalog (in-memory `registrytest` fixture, or a re-published catalog).

## Decision

**Adopt the v0.17 concurrent-render model:** per-goroutine Kernels, one shared read-only `*MaterializedPlatform`, no mutex, no re-materialize. The spike answers its question affirmatively — concurrent cross-context rendering on v0.17 is both race-clean and genuinely parallel.

The production change (a separate, spec-bearing recontract) is **narrow, not a redesign**. Its crux is localized to `opm/compile`: build/unify in the **caller-Kernel's** context and only cross-*read* the shared platform, at both cross-context points (`compile.Match`/`unifyIntersection` and Execute's `FillPath`), replacing `r.platform.Package.Context()` with an explicitly-threaded `*cue.Context`. That change is simultaneously the deprecation fix and the parallelism enabler. The recontract also: pins CUE to v0.17 (once it leaves alpha, or under explicit alpha-risk acceptance), audits `Value.Context()` fallout across `compile`/`materialize`/`schema`, rewrites `kernel-runtime` §"Goroutine Safety Contract" and `platform-materialization` §"MaterializedPlatform Output Shape", lands a permanent `-race` regression test against a v0.17-compatible in-memory catalog, and **must not pin v0.17 until the published catalogs/modules are re-vetted and re-published to parse under v0.17** (a cross-repo precondition surfaced by the spike, not a library-only change).

Alternatives rejected: **shared Kernel + caller mutex** (correct, but serializes rendering and yields none of the measured ~2.5×); **per-goroutine Kernels each re-`Materialize`** (defeats materialize-once-reuse-many, the basis of the Platform CRD). Both remain the v0.16 fallback if the v0.17 adoption is deferred.

## Consequences

**Positive:** Concurrent rendering against a single materialized platform becomes possible and parallel; the operator's render path needs no mutex and the materialized platform is reused, not rebuilt. The kernel's concurrency story becomes "per-goroutine Kernel, shared read-only platform" — simpler than making a single Kernel internally thread-safe. The required kernel change is small and contained to `opm/compile`.

**Negative / Trade-off:** Production adoption pins CUE `v0.17`, currently alpha; the workspace is on `v0.16.1`, so the recontract either waits for a stable v0.17 or explicitly accepts alpha risk. Measured speedup saturates around four cores (allocator-bound on this workload) — real, but sub-linear; throughput beyond that needs allocation work, not more goroutines. The raw-CUE keystone proves the CUE primitive; the kernel end-to-end path is confirmed by code analysis and the `spike-concurrent-render-v0170` full-flow test, not yet by production code.

**Decoupling:** The operator does not wait on any of this. It ships on `v0.16.1` with a render-path mutex and drops the mutex later, only if and when this ADR's recontract lands. Nothing here is on the operator's critical path.
