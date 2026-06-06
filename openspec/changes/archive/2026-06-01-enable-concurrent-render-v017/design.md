## Context

ADR-002 adopted the v0.17 concurrent-render model: per-goroutine Kernels, one shared read-only `*MaterializedPlatform`, no mutex, no re-materialize. `concurrent-render-recontract` landed the enabling code — the compile pipeline now builds in the caller Kernel's `*cue.Context` (`k.CueContext()`) and consumes the platform's `Package` as read-only input — and the v0.17.0-alpha.1 toolchain is now merged with republished, v0.17-parseable `core` / `catalogs/opm`.

What remains is to (1) state the now-true guarantee in the specs and godoc, and (2) prove it with a permanent regression test. On v0.16.1 a release built by Kernel `K1` rendered against a platform materialized by `K0` panicked `values are not from the same runtime`; v0.17 makes that cross-context combination legal and race-safe. The recontract deliberately left the **Goroutine Safety Contract** and **MaterializedPlatform Output Shape** specs unchanged because the sharing guarantee was false on v0.16. It is true now.

Constraint: no Go public-API change; no further CUE-version movement; the test must be hermetic (no registry round-trip) so it runs under `-race` in CI on every push.

## Goals / Non-Goals

**Goals:**

- The `kernel-runtime` and `platform-materialization` specs assert that one `*MaterializedPlatform`, materialized once, is safe to render against concurrently from many per-goroutine Kernels under the v0.17 toolchain.
- `kernel/doc.go` documents the model and carries a concurrent-render-against-a-shared-platform example (replacing the implication that concurrency means each goroutine does fully independent work).
- A permanent `opm/kernel` `-race` test exercises the real compile pipeline: a platform materialized by one Kernel, rendered concurrently by N other per-goroutine Kernels, asserting both race-cleanliness **and** correct per-release output.

**Non-Goals:**

- Making a single `Kernel` internally goroutine-safe across method calls. The model is one Kernel per goroutine; that line stays true.
- Any CUE-version bump (v0.17.0-alpha.1 already merged) or change to `Kernel` / `compile` / `materialize` signatures.
- Removing `.spike/crosscontext` (separate `chore`) or the operator's mutex drop (downstream `opm-operator`).

## Decisions

### D1: Strengthen the contract, do not weaken "one Kernel per goroutine"

The Goroutine Safety Contract keeps "a Kernel is not safe for concurrent use across its own method calls — one Kernel per goroutine." It **adds**: a `*MaterializedPlatform` produced by one Kernel is safe to be read concurrently by other Kernels' `Compile` calls under v0.17, because the pipeline builds in the caller's context and only cross-*reads* the platform.

- *Why:* this is the precise, minimal truth the v0.17 toolchain + recontract code already deliver. Overstating it ("a Kernel is now concurrency-safe") would be false and dangerous.
- *Alternative:* make `Kernel` internally thread-safe → larger surface, contradicts the proven model, rejected per ADR-002.

### D2: Hermetic cross-context test — platform from one Kernel, releases from N others

The regression test materializes (or hand-builds, mirroring `newPhaseFixture`) one `*MaterializedPlatform` in a dedicated Kernel `K0`'s context, then spawns N goroutines; each constructs its **own** Kernel, builds its **own** `ModuleRelease` (in its own context), and calls `Compile` against the single shared platform. It uses the inline-fixture platform (no OCI registry), so it is fast and deterministic in CI.

- *Why:* the cross-context boundary only exists when the release's context differs from the platform's. Building the platform in `K0` and releases in per-goroutine Kernels reproduces exactly the operator's materialize-once-reuse-many topology. A registry pull would make the test slow and flaky for no added coverage of the concurrency property.
- *Alternative:* reuse `kernel_test.go`'s existing goroutine test (each Kernel does independent work) → does not share a platform, so it never exercises cross-context rendering. Rejected; that test stays as the one-Kernel-per-goroutine doc example.

### D3: Assert correctness, not just race-cleanliness

The test asserts each goroutine's `CompileResult` carries the expected per-release output (e.g. the echoed `#context.#moduleReleaseMetadata.name`), not merely that `-race` is silent. Distinct releases must produce distinct, correct outputs.

- *Why:* "race-clean" does not imply "correct" — a data race could be absent while results are silently cross-contaminated through a shared context. ADR-002 §Negative explicitly flags that "safe" ≠ "parallel" ≠ "correct"; the test must close all three.

### D4: No public API change — documentation + test + spec only

The enabling code already shipped. This change adds a godoc block, an example, a test, and spec language. SemVer MINOR (a newly guaranteed behavior), no signatures touched.

## Risks / Trade-offs

- **v0.17 is alpha; the guarantee is contingent on the pinned toolchain** → the permanent `-race` test is the guard: a CUE regression that breaks cross-context safety fails CI immediately, surfacing the contingency rather than hiding it.
- **A test that does not truly cross contexts proves nothing** → D2 mitigates by building the platform and the releases in distinct Kernels and asserting (in code review) that no goroutine reuses `K0`.
- **Sub-linear scaling (allocator-bound ~4 cores, per the spike)** → not a correctness concern; documented in `doc.go`/ADR-002 so callers do not expect linear speedup. The test asserts correctness, not throughput.
- **Flake risk under `-race`** → mitigated by the hermetic inline fixture (no network), a fixed goroutine/iteration count, and `t.Parallel`-free deterministic structure.

## Migration Plan

Single self-contained change:

1. Rewrite `kernel-runtime` §Goroutine Safety Contract and `platform-materialization` §MaterializedPlatform Output Shape (delta specs).
2. Update `kernel/doc.go`: document the shared-read-only-platform model; add the concurrent-render example.
3. Add the `opm/kernel` concurrent `-race` regression test.
4. Validation gates (`task fmt|vet|lint`, `task test`, and `go test -race ./opm/kernel/...`).

Rollback: revert the commit; no data, registry, or version state to undo.

## Open Questions

None. The toolchain is merged, the enabling code is in, the two target specs and the test topology are settled (D1–D3). The `.spike` removal and the operator mutex drop are explicitly separate.
