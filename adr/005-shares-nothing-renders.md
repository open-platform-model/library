# ADR-005: Shares-nothing renders

## Status

Accepted (2026-09-03). Supersedes ADR-002. Records enhancement 0019 D8 (workspace root, `enhancements/0019/03-decisions.md`) together with the `cue.Context` lifetime rule that resolves its OQ12. Implemented by `library-render-build` (`Kernel.Render`) and `library-render-cutover` (`Render` as the sole render path; `opm/materialize` and `opm/compile` deleted).

## Context

ADR-002 adopted "per-goroutine Kernels, one shared read-only `*MaterializedPlatform`, no mutex" for concurrent rendering, on the strength of a raw-CUE keystone that exercised reads. It recorded its own caveat: the verified v0.17 guarantee was reads-only, while the render path was construction-heavy. The render path was not read-only. `executePair` filled `#moduleInstance`, `#component` and `#context` into a `#transform` reached through the shared platform, and a fill is a write to that value's evaluation state. Enhancement 0019 experiment 06 ran the adopted shape against the real catalog under the race detector and measured 2321 reports, 1540 after pre-evaluating the platform first, so laziness was not the cause. No wrong output was observed across 2000 renders; the behaviour was undefined rather than demonstrably corrupting.

Two things then changed the question rather than the answer. 0019 D9 made a render one CUE build: the instance package, the platform package and the catalogs it imports enter a generated render module by import, and matching and execution run inside that build as CUE (D10). There is no Go loop over pairs, no `FillPath` into a platform value and no materialized twin, so the value ADR-002 shared no longer exists. And the operator's question moved from "may several renders read one platform" to "what may a long-lived render worker hold between renders" (0019 OQ12), because a worker that keeps a `cue.Context` keeps every value that context ever built.

The measurements that decide the lifetime question come from 0019 experiments 07 and 08 (`enhancements/0019/experiments/`), on 2-, 9-, 33- and 129-component modules at four workers:

| Worker shape | Retained per render |
| --- | --- |
| Fresh `cue.Context` per render (`S2`) | 35 KB to 117 KB, flat in render count |
| One `cue.Context` per worker, reused across renders (`S1`) | 41.9 MB to 581.8 MB, growing with render count; 23.4 GB resident through 32 renders at 129 components |
| One shared held platform, renders serialised behind a mutex (`SB`, ADR-002's own fallback) | 348 MB at 129 components |

Against the serialised shared-platform path, the fresh-context worker delivers 2.48x the throughput at 2 components and 5.49x at 129: there is no module size at which sharing wins once safety is accounted for. A single render is single-threaded. With the collector disabled a 129-component render uses 1.04 cores; forcing `rendered` concrete after the build call returns costs 3.0 ms of an 1831 ms render, so every pair is already evaluated; and rendering one component still costs 46% to 76% of rendering all of them.

## Decision

Two rules, both mechanical in `Kernel.Render`:

1. **Each render is its own CUE build in its own `cue.Context`.** `Render` stages one generated render module and evaluates it in a context created for that call. The Kernel's own context, which acquisition, synthesis and validation use, does not participate. Nothing built by one render is visible to another.
1. **The context does not outlive the render.** `Render` returns decoded Go values (`[]*core.Compiled`, `RenderDiagnostics`) and drops the context, the built value and the staging directory before returning. The Kernel holds no built value between calls, and a caller cannot obtain one to hold.

Concurrency is therefore across renders, never within one. A consumer that renders from several goroutines gives each goroutine its own Kernel and calls `Render`, with no shared platform value and no mutex. A single Kernel remains single-threaded across its own method calls, because the context-owning methods share its context.

Rejected alternatives:

- **Amend ADR-002 with a mutex around the shared platform.** Correct, and 2.48x to 5.49x slower than a shares-nothing worker across the measured range, while still retaining 348 MB per render because the process holds one context for its life. A correction that is slower and keeps the other defect is a stopgap, not an architecture.
- **Give each worker a private copy of the materialized platform.** Its safety rests on the copy being genuinely independent, the assumption ADR-002 already made and lost. It is also the reused-context shape at a different granularity, which experiment 08 measures at 41.9 MB to 581.8 MB retained per render.
- **Reuse one `cue.Context` per worker for the single build.** Race-clean, no measured throughput advantage, and it retains everything the worker ever rendered. Rejected on memory alone.
- **Build the instance once and execute the matched pairs in parallel.** There is no phase left to schedule: CUE has evaluated every pair by the time the build call returns, and the Amdahl bound on splitting the pairs across builds is 1.31x to 2.16x for K times the working set.
- **Leave ADR-002 standing and let the collapse not use it.** An ADR describing a model nothing runs is worse than none; the next person to touch the render path would implement it again.

## Consequences

**Positive:** Races are excluded by construction rather than avoided by discipline: no built value is reachable from two goroutines, and `task test` runs `opm/kernel` and `opm/internal/renderstage` under `-race` so the claim is checked on every push. Retention is bounded: a worker holds 35 KB to 117 KB per render and does not grow with render count, so a long-lived operator process does not grow until restarted. Concurrent throughput is 2.48x to 5.49x that of the serialised shared-platform path and is independent of module size within 5%. The operator's held `*MaterializedPlatform` slot, its kernel-serialisation stopgap and its cache-invalidation question disappear, because nothing is held across a reconcile.

**Negative:** Every render rebuilds the platform and its catalogs. Sequentially, at two components, the single build costs 1.74x the old per-render time (experiment 07), and it only wins sequentially from roughly five to fourteen components upward. There is no within-render parallelism to harvest and no supported way to cache a built platform across renders; the only reuse a consumer gets is CUE's on-disk module cache.

**Trade-off:** A render pool is sized by memory, not by core count. Peak resident memory per concurrent render fits `61 MB + 7.75 MB x components` (experiment 08, R^2 = 0.9997), and throughput saturates at about `physical cores / 1.6` renders in flight, because each render is one evaluator thread plus roughly 0.6 of a collector. A pod rendering 10- to 25-component modules at four concurrent renders wants about 1 GB and is comfortable at 2 GB; a 129-component fleet at eight concurrent renders wants 12 GB. Size against the largest module the pool will see: the pool has no admission control that stops several large renders coinciding. The full table is in `enhancements/0019/06-operational.md`.

**Relation to ADR-003:** the no-cross-build-fill principle stands and is now satisfied by one tactic. The federation application ADR-003 named (`indexCatalogs`, `MaterializedPlatform.Transformers` / `.Matchers`) was deleted with the old pipeline; the single build is the only construction path.
