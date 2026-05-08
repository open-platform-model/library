# Design Package: Kernel Redesign Around `#Platform`

| Field       | Value            |
| ----------- | ---------------- |
| **Status**  | Draft            |
| **Created** | 2026-05-07       |
| **Authors** | OPM Contributors |

## Summary

Restructures the OPM kernel into a self-contained reference runtime that the CLI, `opm-operator`, and the planned Crossplane composition function can all embed without divergence. The kernel becomes a `Kernel` struct that owns its `cue.Context`, exposes phase-explicit methods (`Validate`, `Match`, `Plan`, `Compile`), accepts a uniform `(APIVersion, Metadata, Package)` shape across every OPM artifact type, takes a single pre-unified values `cue.Value` instead of a slice, and matches against the new `#Platform` construct (catalog enhancement 014) instead of the retired `#Provider`. Loading, layering, and composition move into opt-in helper packages under `pkg/helper/`. Tier-1 source-positioned validation lives in helpers; the kernel keeps a Tier-2 correctness safety net.

`#Claim` (catalog enhancement 015) is intentionally deferred — the matcher rewrite covers Resources/Traits only in this design package; Claims fold in once 015 lands in the catalog.

This enhancement does not itself land code. It is the umbrella reference for a sequence of small, independently shippable OpenSpec changes (slices). Each slice has its own proposal, design, and tasks under `openspec/changes/`. This document lists them and tracks dependencies so the work stays coherent.

## Documents

1. [01-problem.md](01-problem.md) — Why the current kernel shape blocks multi-frontend embedding; what each frontend (CLI, operator, XR fn) actually needs
2. [02-design.md](02-design.md) — Cross-cutting design — Kernel struct, uniform artifact shape, two-tier validation, helper layout, Platform integration, slice dependency graph
3. [03-decisions.md](03-decisions.md) — Settled decisions (D1–D10) with rationale and alternatives considered

## Slicing Plan

Each slice is an independent OpenSpec change at `openspec/changes/<slice-name>/`. Slices are ordered roughly by dependency, not by time. Land them in any topologically valid order.

| #  | Slice (kebab-case)                       | Scope                                                                                                  | Depends on    | Risk     |
| -- | ---------------------------------------- | ------------------------------------------------------------------------------------------------------ | ------------- | -------- |
| 01 | `add-kernel-struct`                      | Introduce `Kernel` type owning `cue.Context` + DI; route existing free functions through it            | apiversion    | low      |
| 02 | `unify-artifact-shape`                   | All artifact types collapse to `(ApiVersion, Metadata, Package cue.Value)`                             | apiversion    | medium   |
| 03 | `retire-module-debug`                    | Remove top-level `#ModuleDebug`; document `Module.debugValues` as the only debug surface               | —             | trivial  |
| 04 | `accept-single-values-input`             | Kernel signatures take one `cue.Value` for values, not `[]cue.Value`                                   | —             | low      |
| 05 | `introduce-tiered-validation`            | Add `helper/values` for Tier-1 source-positioned validation; kernel becomes Tier-2 safety net          | 04            | medium   |
| 06 | `add-phase-methods-and-rename-compile`   | Expose `Validate` / `Match` / `Plan` / `Compile` as Kernel methods; rename Render → Compile end-to-end | 01            | low      |
| 07 | `reorganize-helpers-under-helper`        | Move `pkg/loader/*` → `pkg/helper/loader/{file,bytes}`; flatten optional code under `pkg/helper/`      | 01, 06        | low      |
| 08 | `add-platform-construct`                 | Add `Platform` type with the uniform shape; loader for `platform.cue`; kernel input adds `*Platform`   | 02            | medium   |
| 09 | `rewrite-match-around-platform`          | Replace `pkg/render/match.go` to consume `Platform.#composedTransformers` + `Platform.#matchers`       | 08            | high     |
| 10 | `add-platform-composition-helper`        | `pkg/helper/platform.Compose(shell, modules)` for operator + CLI + XR shared composition               | 08            | low      |

`apiversion` = the in-flight `add-multi-apiversion-support` change. That lands first; every slice in this enhancement assumes the binding interface (`pkg/api/<v>` + `pkg/apiversion`) is available.

## Applicability Checklist

- [x] `01-problem.md` — Multi-frontend embedding requirements; failure modes of current shape
- [x] `02-design.md` — Cross-cutting design + slice dependency graph
- [x] `03-decisions.md` — Numbered decisions D1–D10
- [ ] `NN-schema.md` — Deferred. No CUE schema changes in the kernel repo; schema work is in `catalog/enhancements/014-platform-construct/` and `015-claims/`
- [ ] `NN-pipeline-changes.md` — Deferred. Per-slice `design.md` files capture pipeline changes for that slice
- [ ] `NN-notes.md` — Deferred. Open questions captured per-slice or in this README's open-questions footer

## Open Questions

- **OQ1 — `Resolve` collision.** 015 introduces a `#resolution` writeback channel for Claims. The Kernel public method `Resolve` (if introduced later) would collide. Once Claim support lands in the kernel, choose a different verb for the writeback surface (current candidate: leave it as a `Resolution map[string]cue.Value` field on `CompileResult`, no method named Resolve).
- **OQ2 — `Plan` semantics.** Resolved in slice 06 as "Compile with the rendered slice dropped." Plan delegates to Compile internally and copies `MatchPlan`, `Components`, `Unmatched`, `Ambiguous`, and `Warnings` into a `*PlanResult` — no separate execution path. This pins Plan and Compile to one pipeline, so any error a future Compile would surface (transformer evaluation, finalization) also surfaces at Plan time. The cost is that Plan is no cheaper than Compile; if a frontend later needs a faster preview, Plan can grow a stop-after-match flag without changing its return type.
- **OQ3 — Concurrent `Kernel` reuse.** The `Kernel` struct holds a `cue.Context` and is documented as goroutine-unsafe across compile calls. Operator workers should construct one Kernel per goroutine. If real-world operator throughput proves this expensive, consider a `KernelPool` helper later — not in scope for any current slice.

## Cross-References

| Document                                                                                | Purpose                                                                                                                                                        |
| --------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `library/CONSTITUTION.md`                                                               | Kernel constitutional principles (Kernel Neutrality, Type Safety, Separation of Concerns, Stable Contracts, CUE-Native Resolution, SemVer, YAGNI, Small Batch) |
| `library/openspec/changes/add-multi-apiversion-support/`                                | In-flight prerequisite — version binding interface every slice assumes                                                                                         |
| `catalog/enhancements/014-platform-construct/`                                          | Source-of-truth design for `#Platform`, `#PlatformMatch`, `#ComponentTransformer`, `#TransformerMap`, `#registry`, `#matchers`                                 |
| `catalog/enhancements/015-claims/`                                                      | Source-of-truth design for `#Claim`, `#ModuleTransformer`, `#defines.claims`, `#resolution` writeback (deferred in kernel until catalog lands)                 |
| `catalog/enhancements/016-module-context/`                                              | `#ctx` / `#PlatformContext` / `#ModuleContext` / `#ContextBuilder` — relevant when slice 09 wires per-deploy context injection                                 |
| `cli/`                                                                                  | Downstream consumer — one-shot CLI embedding the kernel                                                                                                        |
| `opm-operator/`                                                                         | Downstream consumer — Kubernetes controller embedding the kernel                                                                                               |

<!--
## Status Lifecycle

- **Draft** — initial design, actively being written
- **Accepted** — design agreed upon, ready for slice implementation
- **Implemented** — every slice has been merged
- **Superseded by NNN** — replaced by a newer enhancement
-->
