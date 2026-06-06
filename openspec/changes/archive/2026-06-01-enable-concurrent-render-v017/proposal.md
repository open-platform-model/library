## Why

The `concurrent-render-recontract` change landed the v0.16-safe half of ADR-002: the compile pipeline now builds in the caller Kernel's `*cue.Context` and reads the materialized platform as input. The CUE v0.17 toolchain (which makes cross-context `FillPath`/`Unify` legal and `cue.Value` reads race-safe) is now merged, and the republished `core` / `catalogs/opm` parse under it. The two cross-repo preconditions ADR-002 gated on are met — so the deferred enablement half can now land: assert and prove that one `*MaterializedPlatform`, materialized once, is safe to render against concurrently from many per-goroutine Kernels.

## What Changes

- **Rewrite the goroutine-safety contract** to document the v0.17 concurrent-render model: "one Kernel per goroutine" stays true, but a `*MaterializedPlatform` materialized once by one Kernel is now safe to be **read concurrently** by other per-goroutine Kernels' `Compile` calls — no mutex, no re-materialize. `kernel/doc.go` gains a concurrent-render-against-a-shared-platform example.
- **Add a permanent concurrent `-race` regression test** (`opm/kernel`): N goroutines, each its own Kernel, all rendering distinct `ModuleRelease`s against one shared `*MaterializedPlatform` (built by a separate Kernel), asserting race-clean execution and correct per-release output. This is the in-tree, real-pipeline successor to the raw-CUE `.spike` keystone — now buildable because the catalog parses under v0.17.
- **No Go public API change.** Additive documentation + a test + spec-language strengthening only. The `Kernel` surface, `compile.NewModule`, and `Materialize` signatures are unchanged.

### Out of scope

- Any further CUE-version movement (v0.17.0-alpha.1 is already merged; this change does not bump it).
- Removing the in-tree `.spike/crosscontext` experiment — handled as a separate `chore` (its findings are captured in ADR-002 and superseded by the test added here).
- The operator dropping its render-path mutex — that is downstream `opm-operator` work, not the library.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `kernel-runtime`: the **Goroutine Safety Contract** requirement is rewritten — it MUST now also state that a `*MaterializedPlatform` is safe to share read-only across goroutines and Kernels under the v0.17 toolchain, enabling concurrent rendering from per-goroutine Kernels against one shared platform.
- `platform-materialization`: the **MaterializedPlatform Output Shape** requirement is extended — the `Package` MUST be safe for concurrent read-only consumption by multiple Kernels' compile pipelines (the basis of the materialize-once-reuse-many Platform-CR model).

## Impact

- **Affected `opm/` packages:** `kernel` (`doc.go` documentation; new `*_test.go` concurrent-render regression test). No non-test source behavior changes — the enabling code (caller-Kernel context threading) already landed in `concurrent-render-recontract`.
- **Affected specs:** `kernel-runtime` (Goroutine Safety Contract), `platform-materialization` (MaterializedPlatform Output Shape).
- **SemVer:** MINOR — a newly documented and test-enforced concurrency guarantee; no signature or behavior change for existing single-Kernel callers.
- **Downstream:** unblocks `opm-operator` to render concurrently against one materialized platform and to drop its interim render mutex (a separate operator change). Nothing here is on the operator's critical path.
- **CUE version:** unchanged (`v0.17.0-alpha.1`, already merged). The guarantee is explicitly contingent on the v0.17 toolchain.
