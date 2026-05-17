# Design Package: Kernel Redesign Around `#Platform`

See [config.yaml](config.yaml) for metadata.

## Summary

Restructures the OPM kernel into a self-contained reference runtime that the CLI, `opm-operator`, and the planned Crossplane composition function can all embed without divergence. The kernel is a `Kernel` struct that owns its `cue.Context`, exposes phase-explicit methods (`Validate`, `Match`, `Plan`, `Compile`), accepts a uniform `(APIVersion, Metadata, Package)` shape across every OPM artifact type, takes a single pre-unified values `cue.Value` instead of a slice, and matches against the new `#Platform` construct (enhancement 003) instead of the retired `#Provider`. Loading and composition live in opt-in `opm/helper/` packages; layered validation lives on the kernel itself (see amendment to D5 below). Tier-1 source-positioned validation runs on the kernel via `Source` / `ValidateConfigDetailed`; the kernel keeps a Tier-2 correctness safety net.

`#Claim` (enhancement 005) is intentionally deferred — the matcher rewrite covers Resources/Traits only; Claims fold in once 005 stabilizes.

This enhancement did not itself land code. It is the umbrella reference for a sequence of small, independently shippable OpenSpec changes (slices); the slice list is in `config.yaml.slices`.

> **Implementation status (2026-05-14).** Complete. 20 OpenSpec change slices archived under `library/openspec/changes/archive/` (see `config.yaml.slices` for the list, and `## Slice Implementation Notes` below for per-slice notable choices). `## Known Deviations Between Design and Code` records deliberate divergences; `## Open Questions — Resolution` records OQ outcomes.

## Scope

This is an umbrella enhancement; concrete scope is carried by the 20 archived OpenSpec slices listed in `config.yaml.slices` (each under `library/openspec/changes/archive/`). Each slice has its own proposal, design, specs, and tasks.

### In scope

- Kernel API surface rewrite covered by the 20 archived slices.
- Retirement of `#Provider` and migration of matcher to `#composedTransformers` + `#matchers` (jointly with enhancement 003).
- Two-tier validation surface on the kernel itself (see D5 amendment).
- Helper boundary under `opm/helper/{loader,platform,synth}/`.

### Out of scope

- `#Claim` integration into the kernel (deferred to enhancement 005).
- `#ctx` / `#PlatformContext` injection (deferred to enhancement 004).
- Repo rename from `library` to `kernel` (D9 deferred — see Known Deviations).
- `ValidateInput.Module` alignment with the slim shape (deferred — see Known Deviations).

## Documents

1. [01-problem.md](01-problem.md) — Why the previous kernel shape blocked multi-frontend embedding; what each frontend (CLI, operator, XR fn) actually needs
2. [02-design.md](02-design.md) — Cross-cutting design — Kernel struct, uniform artifact shape, two-tier validation, helper layout, Platform integration, slice dependency graph
3. [03-decisions.md](03-decisions.md) — Settled decisions (D1–D10) with rationale, alternatives considered, and post-implementation amendments

## Slice Implementation Notes

Per-slice choices worth preserving. The slice list itself is in `config.yaml.slices`; each is archived under `library/openspec/changes/archive/`. The original plan ran 01–11; eight follow-up slices (everything dated `2026-05-09` onward beyond `slim-kernel-inputs`) shipped as the design met the codebase.

- **add-multi-apiversion-support** *(prerequisite)* — Version binding interface every slice assumes; landed before slice 01.
- **add-kernel-struct** — Functional options (`WithLogger`/`WithTracer`/`WithClock`) chosen over an `Options` struct.
- **unify-artifact-shape** — `Module`, `Release`, `Platform` all carry `(APIVersion, *Metadata, Package cue.Value)`.
- **retire-module-debug** — `Module.debugValues` exposed via `api.Paths().DebugValues`.
- **accept-single-values-input** — All phase inputs take one `cue.Value`.
- **introduce-tiered-validation** *(amended)* — Originally landed `opm/helper/values`. Follow-up `redesign-config-validation` replaced it with kernel-level `Source`, `ValidateOption`, `Partial()`, `ValidateConfigDetailed`, `ValidateModuleValues*`, `ValidateReleaseValues*`. See D5 amendment.
- **add-phase-methods-and-rename-compile** — `Render` → `Compile` end-to-end; package renamed `opm/render` → `opm/compile`.
- **reorganize-helpers-under-helper** — `opm/loader/*` → `opm/helper/loader/{file,bytes}`. `loader/bytes` ships as a skeleton (`doc.go` only) until a consumer pulls.
- **add-platform-construct** — `*platform.Platform` mirrors `*module.Module`.
- **rewrite-match-around-platform** — `opm/compile/match.go` consumes `Platform.#composedTransformers` + `Platform.#matchers`. `opm/provider/` deleted.
- **add-platform-composition-helper** — `helper/platform.Compose(shell, modules)`; surfaced as `(*Kernel).ComposePlatform`.
- **slim-kernel-inputs** — `Module` field dropped from `MatchInput`, `PlanInput`, `CompileInput`. `ValidateInput.Module` retained (see "Known Deviations" below).
- **fold-deprecated-functions-into-kernel** — Move surviving free-function entry points onto `*Kernel` so the struct is the single anchor.
- **redesign-config-validation** — Replace `opm/helper/values` with kernel-level `Source` / `ValidateConfigDetailed` / `Partial`. Amends D5.
- **add-cue-schema-test-harness** — Shared in-process schema-loading harness for `opm/api/<v>` tests.
- **add-cue-schema-test-coverage** — Coverage sweep using the harness.
- **add-release-synth-helper** — New `opm/helper/synth/` (peer of `loader/`) for typed-input release synthesis; surfaced as `(*Kernel).SynthesizeRelease`.
- **unify-loaders-as-packages** — Collapse the four loader entry points into the `LoadModulePackage` / `LoadReleasePackage` / `LoadPlatformPackage` triad.
- **add-loader-shape-gates** — Shape-gate every loader before returning the value to the kernel.
- **replace-load-platform-file-with-package** — Drop the file-based platform loader in favour of the package loader.

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
| `library/enhancements/005-claims/`                                    | Source-of-truth design for `#Claim`, `#ModuleTransformer`, `#defines.claims`, `#resolution` writeback (deferred in kernel until schema lands) |
| `library/enhancements/004-module-context/`                            | `#ctx` / `#PlatformContext` / `#ModuleContext` / `#ContextBuilder` — relevant where context injection is wired |
| `library/opm/kernel/`                                                 | The implemented `Kernel` struct, phase methods, validation surface, source-loader wrappers |
| `library/opm/{module,platform,compile,api,apiversion,core,errors}/`   | Kernel-internal packages |
| `library/opm/helper/{loader,platform,synth}/`                         | Opt-in helper boundary |
| `cli/`                                                                | Downstream consumer — one-shot CLI embedding the kernel |
| `opm-operator/`                                                       | Downstream consumer — Kubernetes controller embedding the kernel |
