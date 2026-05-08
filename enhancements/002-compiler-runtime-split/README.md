# Design Package: Compiler / Runtime Kernel Split for Op & Action Execution

| Field       | Value            |
| ----------- | ---------------- |
| **Status**  | Draft            |
| **Created** | 2026-05-08       |
| **Authors** | OPM Contributors |

## Summary

Splits the OPM kernel into two distinct concerns: a **Compiler** that performs the existing deterministic transform from `*ModuleRelease + *Provider/*Platform + values → CompileResult`, and a **Runtime** that executes the operational primitives (`#Op`, `#Action`) declared by catalog enhancement 010. The two share a common `Kernel` substrate (`cue.Context`, `pkg/api` binding registry, logger, tracer) but live in separate packages with separate import surfaces, so the Crossplane composition function can embed the Compiler without ever pulling Runtime symbols.

The Compiler emits a deterministic plan of `ActionInvocation`s as part of its result. The frontend orchestrates Runtime execution against that plan. The Runtime publishes a `RuntimeSnapshot` (a `cue.Value`) that the next Compile may read. Determinism in the Compiler is preserved by construction — the wall is enforced at the package boundary, not by docstring.

This enhancement does not itself land code. It is the umbrella reference for a sequence of small, independently shippable OpenSpec changes (slices). Each slice has its own proposal, design, and tasks under `openspec/changes/`. This document lists them and tracks dependencies so the work stays coherent.

## Documents

1. [01-problem.md](01-problem.md) — Why the kernel must split before Op/Action can execute; how Constitution I forbids effects on the existing surface
2. [02-design.md](02-design.md) — Two-part shape, package boundaries, determinism wall, Compiler↔Runtime feedback loop, embedding stories, slice graph
3. [03-decisions.md](03-decisions.md) — Settled decisions (D1–D10) with rationale and alternatives considered

## Slicing Plan

Each slice is an independent OpenSpec change at `openspec/changes/<slice-name>/`. Slices are ordered roughly by dependency, not by time. Land them in any topologically valid order.

| #  | Slice (kebab-case)                       | Scope                                                                                          | Depends on             | Risk    |
| -- | ---------------------------------------- | ---------------------------------------------------------------------------------------------- | ---------------------- | ------- |
| 01 | `drop-unused-kernel-clock`               | Remove `Kernel.Clock`, the `Clock` interface, and `WithClock` (YAGNI — currently no consumer)  | —                      | trivial |
| 02 | `add-action-decoder-paths`               | `pkg/api/v1alpha2` gains `Paths.Steps` / `Paths.After` / `Paths.OpType` and `DecodeAction`     | catalog 010 publish    | low     |
| 03 | `add-action-invocation-core-type`        | `core.ActionInvocation` Go type + `core.StepNode` + `core.ActionDAG`                           | 02                     | low     |
| 04 | `emit-action-invocations-from-compile`   | Internal Action walker — descends a Release, finds Action declarations, emits `[]*ActionInvocation`. No `CompileResult` field change yet (consumers add typed fields in slices 08/09); shipped as `compile.WalkActions` for slices 08/09 to call. | 03, 001 slice 06       | medium  |
| 05 | `add-runtime-package-skeleton`           | `pkg/runtime` — `Runtime` struct, `Executor` interface, in-memory `StateStore`, `$after` walker | 03                     | medium  |
| 06 | `add-local-op-executors`                 | `pkg/runtime/local` — reference executors for `@op("exec")`, `@op("http.*")`, `@op("wait")`    | 05                     | low     |
| 07 | `add-runtime-snapshot-feedback`          | `RuntimeSnapshot` type; Compiler accepts it via option; `Runtime.Snapshot()` produces it       | 05                     | medium  |
| 08 | `add-workflow-decoder`                   | `pkg/api/v1alpha2.DecodeWorkflow`; `CompileResult.Workflows map[string]*ActionInvocation` keyed by Workflow FQN; on-demand invocation pattern (CLI: `opm workflow run <name>`; operator: CRD-triggered) | 04, catalog Workflow   | low     |
| 09 | `add-lifecycle-decoder`                  | `pkg/api/v1alpha2.DecodeLifecycle`; `compile.LifecyclePhase` enum (preInstall/install/postInstall/preUpgrade/upgrade/postUpgrade/preUninstall/uninstall/postUninstall, optional downgrade triplet); `CompileResult.Lifecycle map[LifecyclePhase]*ActionInvocation`; `compile.CanonicalOrder()` helper | 04, catalog Lifecycle  | low     |

`001 slice 06` = `add-phase-methods-and-rename-compile` from `001-kernel-redesign-around-platform/`. That slice renames `Render → Compile` end-to-end. Slices in this enhancement use the new naming throughout. If a slice here lands before 001's slice 06, it must reconcile with the older naming explicitly in its proposal.

`catalog 010 publish` = `catalog/enhancements/010-op-action-primitives/` reaches Implemented status with `#Op`, `#Action`, `#Step` published under the catalog's `apis/core/v1alpha2/`. Library cannot decode what catalog has not published.

`catalog Workflow` and `catalog Lifecycle` = future catalog enhancements that publish the consumer constructs. Slices 08 and 09 are gated on those.

## Applicability Checklist

- [x] `01-problem.md` — Why the kernel must split before Op/Action can execute
- [x] `02-design.md` — Cross-cutting design + slice dependency graph
- [x] `03-decisions.md` — Numbered decisions D1–D10
- [ ] `NN-schema.md` — Deferred. CUE schema for Op/Action lives in `catalog/enhancements/010-op-action-primitives/`. Library introduces no schema.
- [ ] `NN-pipeline-changes.md` — Deferred. Per-slice `design.md` files capture pipeline changes for that slice.
- [ ] `NN-notes.md` — Deferred. Open questions captured per-slice or in this README's open-questions footer.

## Open Questions

- **OQ1 — Cross-step `#out` wiring.** Catalog 010 D4 deferred this; this enhancement follows. The first concrete consumer (likely the operator's restore orchestration) will demand it. When it lands, decide whether to add `$inputs: { from: "stepName.#out.value" }` syntax on the catalog side, or to evaluate references through the kernel-redesign's `#ctx` mechanism (catalog enhancement 016).
- **OQ2 — `StateStore` persistence.** Slice 05 ships an in-memory store. CLI may persist to a file under `~/.opm/state/`. Operator must persist via CRD status. XR fn never persists. Persistence backend selection is per-frontend; the kernel only defines the interface.
- **OQ3 — Concurrent Action execution.** Catalog 010 allows `$after`-derived parallelism, but the initial Runtime ships sequential execution. A `WithMaxParallel(n)` option lands when a real consumer demands the worker-pool complexity.
- **OQ4 — Dry-run.** `Compile` already produces `ActionInvocations` without executing. Whether the Runtime needs an explicit `DryRun` mode (validate executor map presence, no execution) depends on whether CLI users want to see the planned Action graph before committing. Defer until the CLI wires `opm plan`.
- **OQ5 — Failure semantics across slices.** Step failure fails the Action (catalog 010). What an Action failure means for the surrounding Compile↔Run loop (continue with remaining invocations, abort, retry the failed step) is a frontend policy choice. The Runtime must expose per-step error detail; frontends decide loop behavior. Lifecycle-phase failure semantics are particularly load-bearing: a failed `preInstall` likely halts the install entirely, while a failed `postInstall` may be recoverable. Catalog `#Lifecycle` may need to declare per-phase failure-mode hints (`continueOnFailure`, `retry`, `halt`) to inform frontends.
- **OQ6 — Downgrade phases.** Catalog `#Lifecycle` may include a downgrade triplet (`preDowngrade`, `downgrade`, `postDowngrade`) for SemVer-major rollbacks. Library defers the enum entry until catalog publishes; if catalog includes downgrade, library mirrors with `compile.PhasePreDowngrade` etc. If catalog excludes it, library has no constant to emit.
- **OQ7 — Action sharing across consumers.** A single `#Action` definition (e.g., `#DBMigration`) can appear as a step inside both a `#Workflow` and a `#Lifecycle.preUpgrade` declaration. The `ActionInvocation` produced for each is independent (different concrete inputs, different invocation IDs), but the underlying Action FQN is shared. The Runtime treats them as distinct invocations; no deduplication or coordination at the kernel level. Flagged so frontends know shared declarations are expected and the Runtime does not de-duplicate them.

## Cross-References

| Document                                                                                | Purpose                                                                                                                                                  |
| --------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `library/CONSTITUTION.md`                                                               | Constitutional principles — Principle I (Kernel Neutrality & Determinism) is the constraint that motivates the split; Principle VII (YAGNI) governs Clock removal |
| `library/enhancements/001-kernel-redesign-around-platform/`                             | Prerequisite umbrella — Kernel struct (slice 01), Compile rename (slice 06), helper layout (slice 07) all assumed                                        |
| `library/openspec/changes/archive/`                                                     | Archived `add-multi-apiversion-support` — version binding interface every slice in this enhancement assumes                                              |
| `catalog/enhancements/010-op-action-primitives/`                                        | Source-of-truth design for `#Op`, `#Action`, `#Step`, `@op("...")` attribute, `$after` ordering — this enhancement is the kernel-side counterpart        |
| `cli/`                                                                                  | Downstream consumer — first Runtime user (`opm run <action>`)                                                                                            |
| `opm-operator/`                                                                         | Downstream consumer — Runtime user via Lifecycle / Workflow controllers                                                                                  |
| (planned) catalog `#Workflow` construct                                                 | On-demand named Action graph; parallel to CUE's `cmd` scripting concept. Required before slice 08.                                                       |
| (planned) catalog `#Lifecycle` construct                                                | Phase-keyed Action map: preInstall, install, postInstall, preUpgrade, upgrade, postUpgrade, preUninstall, uninstall, postUninstall, optional downgrade triplet. Required before slice 09. |

<!--
## Status Lifecycle

- **Draft** — initial design, actively being written
- **Accepted** — design agreed upon, ready for slice implementation
- **Implemented** — every slice has been merged
- **Superseded by NNN** — replaced by a newer enhancement
-->
