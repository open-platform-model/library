# Tasks — enable-concurrent-render-v017

Scope: document and test-enforce the v0.17 shared-read-only-platform concurrency guarantee. The enabling code (caller-Kernel context threading) already landed in `concurrent-render-recontract`; the v0.17.0-alpha.1 toolchain is already merged. No Go public-API change here. The `.spike/crosscontext` removal is a separate `chore`, NOT listed here.

## 0. Pre-flight

- [x] 0.1 Confirm preconditions: `grep cuelang.org/go go.mod` shows `v0.17.0-alpha.1`, and the caller-Kernel context-threading code is present — `kernel/compile.go` uses `k.cueCtx` for `FinalizeValue` + `compile.NewModule`, and `grep -rn 'Package.Context()' opm/compile opm/kernel | grep -v _test` returns nothing.

## 1. Documentation (`opm/kernel/doc.go`)

- [x] 1.1 Rewrite the `# Goroutine safety` section: keep "a single Kernel is not safe across its own method calls — one Kernel per goroutine"; add that, under the v0.17 toolchain, a `*MaterializedPlatform` materialized once is safe to be read concurrently by many per-goroutine Kernels' `Compile` calls (no mutex, no re-materialize), because the pipeline builds in the caller's `*cue.Context` and only cross-*reads* the shared platform.
- [x] 1.2 Add a godoc example showing the shared-platform model: one Kernel materializes a platform once; N goroutines each construct their own `kernel.New()` and `Compile` a distinct release against the single shared platform. Note (per ADR-002) that speedup is real but sub-linear (allocator-bound, ~4 cores).

## 2. Concurrent `-race` regression test (`opm/kernel`)

- [x] 2.1 Add `TestKernel_ConcurrentRender_SharedPlatform`: build one `*MaterializedPlatform` in a dedicated Kernel `K0` (inline fixture mirroring `newPhaseFixture`, no OCI registry), then spawn N goroutines — each constructs its own `kernel.New()`, builds a **distinct** `ModuleRelease` in its own context, and calls `Compile` against the single shared `K0` platform.
- [x] 2.2 Assert correctness, not just race-silence: each goroutine's `CompileResult` echoes the output for its **own** release (e.g. distinct `#context.#moduleReleaseMetadata.name`), with no cross-contamination between concurrent renders.
- [x] 2.3 Leave the existing one-Kernel-per-goroutine test (`kernel_test.go:~263`, each Kernel doing independent work) in place as the doc example — do not replace it; the new test is the cross-context shared-platform case it never covered.

## 3. Validation gates

- [x] 3.1 `task fmt`
- [x] 3.2 `task vet`
- [x] 3.3 `task lint`
- [x] 3.4 `task test` — full suite green under v0.17.
- [x] 3.5 `go test -race ./opm/kernel/... -count=1` — MUST be race-clean; the new shared-platform test proves the v0.17 guarantee (and would panic `values are not from the same runtime` on v0.16, confirming it exercises the cross-context path).
