# Design Package: Kernel Redesign Around `#Platform`

See [config.yaml](config.yaml) for metadata.

## Summary

Restructures the OPM kernel into a self-contained reference runtime that the CLI, `opm-operator`, and the planned Crossplane composition function can all embed without divergence. The kernel is a `Kernel` struct that owns its `cue.Context`, exposes phase-explicit methods (`Validate`, `Match`, `Plan`, `Compile`), accepts a uniform `(APIVersion, Metadata, Package)` shape across every OPM artifact type, takes a single pre-unified values `cue.Value` instead of a slice, and matches against the new `#Platform` construct (enhancement 003) instead of the retired `#Provider`. Loading and composition live in opt-in `opm/helper/` packages; layered validation lives on the kernel itself (see amendment to D5 below). Tier-1 source-positioned validation runs on the kernel via `Source` / `ValidateConfigDetailed`; the kernel keeps a Tier-2 correctness safety net.

`#Claim` (enhancement 005) is intentionally deferred — the matcher rewrite covers Resources/Traits only; Claims fold in once 005 stabilizes.

This enhancement did not itself land code. It is the umbrella reference for a sequence of small, independently shippable OpenSpec changes (slices). Every slice in the original plan has been archived; seven follow-up slices were added during implementation and are listed below for completeness.

## Documents

1. [01-problem.md](01-problem.md) — Why the previous kernel shape blocked multi-frontend embedding; what each frontend (CLI, operator, XR fn) actually needs
2. [02-design.md](02-design.md) — Cross-cutting design — Kernel struct, uniform artifact shape, two-tier validation, helper layout, Platform integration, slice dependency graph
3. [03-decisions.md](03-decisions.md) — Settled decisions (D1–D10) with rationale, alternatives considered, and post-implementation amendments

## Slicing Plan — Status

Every slice in the original plan is archived at `library/openspec/changes/archive/2026-05-08-<slug>/`. The "Shipped as" column shows the dated archive directory.

| #  | Slice                                    | Shipped as                                          | Status | Notes |
| -- | ---------------------------------------- | --------------------------------------------------- | ------ | ----- |
| 01 | `add-kernel-struct`                      | `2026-05-08-add-kernel-struct`                      | ✅     | Functional options (`WithLogger`/`WithTracer`/`WithClock`) chosen over an `Options` struct. |
| 02 | `unify-artifact-shape`                   | `2026-05-08-unify-artifact-shape`                   | ✅     | `Module`, `Release`, `Platform` all carry `(APIVersion, *Metadata, Package cue.Value)`. |
| 03 | `retire-module-debug`                    | `2026-05-08-retire-module-debug`                    | ✅     | `Module.debugValues` exposed via `api.Paths().DebugValues`. |
| 04 | `accept-single-values-input`             | `2026-05-08-accept-single-values-input`             | ✅     | All phase inputs take one `cue.Value`. |
| 05 | `introduce-tiered-validation`            | `2026-05-08-introduce-tiered-validation`            | ✅ (amended) | Originally landed `opm/helper/values`. Follow-up `redesign-config-validation` replaced it with kernel-level `Source`, `ValidateOption`, `Partial()`, `ValidateConfigDetailed`, `ValidateModuleValues*`, `ValidateReleaseValues*`. See D5 amendment. |
| 06 | `add-phase-methods-and-rename-compile`   | `2026-05-08-add-phase-methods-and-rename-compile`   | ✅     | `Render` → `Compile` end-to-end; package renamed `opm/render` → `opm/compile`. |
| 07 | `reorganize-helpers-under-helper`        | `2026-05-08-reorganize-helpers-under-helper`        | ✅     | `opm/loader/*` → `opm/helper/loader/{file,bytes}`. `loader/bytes` ships as a skeleton (`doc.go` only) until a consumer pulls. |
| 08 | `add-platform-construct`                 | `2026-05-08-add-platform-construct`                 | ✅     | `*platform.Platform` mirrors `*module.Module`. |
| 09 | `rewrite-match-around-platform`          | `2026-05-08-rewrite-match-around-platform`          | ✅     | `opm/compile/match.go` consumes `Platform.#composedTransformers` + `Platform.#matchers`. `opm/provider/` deleted. |
| 10 | `add-platform-composition-helper`        | `2026-05-08-add-platform-composition-helper`        | ✅     | `helper/platform.Compose(shell, modules)`; surfaced as `(*Kernel).ComposePlatform`. |
| 11 | `slim-kernel-inputs`                     | `2026-05-09-slim-kernel-inputs`                     | ✅     | `Module` field dropped from `MatchInput`, `PlanInput`, `CompileInput`. `ValidateInput.Module` retained (see "Known deviations" below). |

`add-multi-apiversion-support` (prerequisite) landed as `2026-05-08-add-multi-apiversion-support` before slice 01.

## Follow-Up Slices Added During Implementation

These were not in the original slicing plan; they shipped as the design met the codebase. Each is archived under `library/openspec/changes/archive/`.

| Shipped as                                            | Scope |
| ----------------------------------------------------- | ----- |
| `2026-05-09-fold-deprecated-functions-into-kernel`    | Move surviving free-function entry points onto `*Kernel` so the struct is the single anchor. |
| `2026-05-09-redesign-config-validation`               | Replace `opm/helper/values` with kernel-level `Source` / `ValidateConfigDetailed` / `Partial`. Amends D5. |
| `2026-05-10-add-cue-schema-test-harness`              | Shared in-process schema-loading harness for `opm/api/<v>` tests. |
| `2026-05-10-add-cue-schema-test-coverage`             | Coverage sweep using the harness. |
| `2026-05-12-add-release-synth-helper`                 | New `opm/helper/synth/` (peer of `loader/`) for typed-input release synthesis; surfaced as `(*Kernel).SynthesizeRelease`. |
| `2026-05-12-unify-loaders-as-packages`                | Collapse the four loader entry points into the `LoadModulePackage`/`LoadReleasePackage`/`LoadPlatformPackage` triad. |
| `2026-05-14-add-loader-shape-gates`                   | Shape-gate every loader before returning the value to the kernel. |
| `2026-05-14-replace-load-platform-file-with-package`  | Drop the file-based platform loader in favour of the package loader. |

## Known Deviations Between Design and Code

These survived implementation deliberately or by drift. Not blockers for acceptance; documented so future reviewers do not chase them as bugs.

- **`ValidateInput.Module` retained.** Slice 11 (`slim-kernel-inputs`) dropped `Module` from `MatchInput`/`PlanInput`/`CompileInput` but left it on `ValidateInput`. `Kernel.Compile` synthesises a transient `*module.Module` from the release's embedded `#module` reference (see `kernel/phases.go` `moduleFromRelease`) to satisfy `Validate`. Aligning `ValidateInput` with the slim shape is a future micro-slice.
- **`Plan`/`Compile` treat `Values` as optional.** Per `kernel/inputs.go` the zero `cue.Value` skips Tier-2 validation. The design implied unconditional Tier-2. The pragmatic deviation is documented at each input field.
- **`PlanResult` has no `Ambiguous` field.** The original OQ2 sketch promised `MatchPlan, Components, Unmatched, Ambiguous, Warnings`. Ambiguity is now folded into `MatchPlan.UnhandledTraits` and surfaces through `MatchPlan.Warnings()`. Resolved in slice 06 as part of writing the result types.
- **Repo not renamed.** D9 said "the repo will be renamed from `library` to `kernel`." Import path remains `github.com/open-platform-model/library/opm/...`. Rename deferred; tracked as a separate rename when downstream consumers are ready. See D9 amendment.

## Open Questions — Resolution

- **OQ1 — `Resolve` collision.** Still open. No `Resolution` field has been added to `CompileResult`; Claims remain deferred to a future enhancement. Naming choice will be made when Claim support lands in the kernel.
- **OQ2 — `Plan` semantics.** Resolved in slice 06. `Plan` delegates to `Compile` internally and copies `MatchPlan`, `Components`, `Unmatched`, and `Warnings` into `*PlanResult`. No separate execution path. See `kernel/phases.go:83-105`.
- **OQ3 — Concurrent `Kernel` reuse.** Resolved by documentation. `kernel.go:18-20` and `doc.go` document the one-Kernel-per-goroutine pattern. No `KernelPool` was needed.

## Cross-References

| Document                                                              | Purpose |
| --------------------------------------------------------------------- | ------- |
| `library/CONSTITUTION.md`                                             | Kernel constitutional principles (Kernel Neutrality, Type Safety, Separation of Concerns, Stable Contracts, CUE-Native Resolution, SemVer, YAGNI, Small Batch) |
| `library/openspec/changes/archive/2026-05-08-add-multi-apiversion-support/` | Prerequisite — version binding interface every slice assumes |
| `library/openspec/changes/archive/2026-05-08-*/`                      | The 11 planned slices, all archived |
| `library/openspec/changes/archive/2026-05-09-*/`, `2026-05-10-*/`, `2026-05-12-*/`, `2026-05-14-*/` | Seven follow-up slices |
| `library/enhancements/003-platform-construct/`                        | Source-of-truth design for `#Platform`, `#PlatformMatch`, `#ComponentTransformer`, `#TransformerMap`, `#registry`, `#matchers` |
| `library/enhancements/005-claims/`                                    | Source-of-truth design for `#Claim`, `#ModuleTransformer`, `#defines.claims`, `#resolution` writeback (deferred in kernel until schema lands) |
| `library/enhancements/004-module-context/`                            | `#ctx` / `#PlatformContext` / `#ModuleContext` / `#ContextBuilder` — relevant where context injection is wired |
| `library/opm/kernel/`                                                 | The implemented `Kernel` struct, phase methods, validation surface, source-loader wrappers |
| `library/opm/{module,platform,compile,api,apiversion,core,errors}/`   | Kernel-internal packages |
| `library/opm/helper/{loader,platform,synth}/`                         | Opt-in helper boundary |
| `cli/`                                                                | Downstream consumer — one-shot CLI embedding the kernel |
| `opm-operator/`                                                       | Downstream consumer — Kubernetes controller embedding the kernel |
