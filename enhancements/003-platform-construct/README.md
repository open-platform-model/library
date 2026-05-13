# Design Package: `#Platform` Construct

| Field       | Value            |
| ----------- | ---------------- |
| **Status**  | Draft            |
| **Created** | 2026-04-30       |
| **Authors** | OPM Contributors |

## Summary

Defines `#Platform` as the catalog construct that models a deployment target. `#Platform` carries platform identity (`metadata`, `type`), platform-level context (`#ctx`, typed `#PlatformContext` from enhancement 004), and a single dynamic ingress — `#registry` — that holds registered `#Module` values. Outward platform-level views at this layer (known resources, known traits, composed transformers, matcher index) are computed projections over `#registry`.

`#Platform` does not carry a `#providers` field. `#Provider` is retired (D12); the matcher consumes `#composedTransformers` and the new `#matchers` reverse index directly. A companion `#PlatformMatch` construct walks a consumer Module's FQN demand against `#matchers` and surfaces `matched` / `unmatched` / `ambiguous` projections per deploy. v1alpha1's single `#Transformer` is replaced by `#ComponentTransformer` (D17) — the sole transformer primitive at this layer; the runtime guarantees a fully concrete `#ModuleRelease` to every `#transform` body (D18).

`#ModuleRegistration` is a pure projection of `#defines` — no install or deploy metadata (D11). Installation is owned by `ModuleRelease` + `opm-operator`: a release CR triggers component install *and* FillPath into `#registry`, registering the Module's primitives automatically.

Multi-fulfiller is allowed at the `#matchers` layer (D13 revised) — overlapping `requiredResources` / `requiredTraits` FQNs across registered transformers list every candidate, and the runtime matcher in the Go pipeline resolves per consumer component via predicate evaluation. `#ModuleRegistration.enabled: false` hides every projection of an entry (D14). Concurrent static + runtime writes to the same Id unify; concrete-value disagreement is surfaced by the `opm-operator` reconciler in `ModuleRelease.status.conditions` (D15). Id keys are kebab-case (`#NameType`, D16). Schema is validated by the self-contained CUE harness at `experiments/002-platform-construct/`.

This enhancement is intentionally thin. `#Environment`, runtime-fill mechanism for `#registry`, self-service catalog runtime, `#PolicyTransformer` integration, topo-sort algorithm for `#status` writeback ordering, and migration of existing provider packages are deferred to follow-up enhancements. `#Claim`, `#ModuleTransformer`, status writeback, and the Claim halves of every `#Platform` view are introduced as extensions in sibling enhancement [005](../005-claims/) — see Cross-References.

> **Implementation status (2026-05-13).** `#Platform`, `#ModuleRegistration`, `#knownResources` / `#knownTraits` / `#composedTransformers`, the `#matchers.{resources,traits}` reverse index, and `#ComponentTransformer` / `#TransformerMap` are landed in `apis/core/v1alpha2/platform.cue` and `apis/core/v1alpha2/transformer.cue`. Two deliberate deviations from the original design: (1) D13 was revised — multi-fulfiller is allowed, not forbidden, and the Go runtime matcher disambiguates via predicate evaluation; (2) `#PlatformMatch` was not landed as a CUE construct — the per-deploy walker is implemented in Go (`pkg/compile/`, `pkg/platform/`). `#Platform.#ctx: #PlatformContext` is deferred to enhancement 004.

## Documents

1. [01-problem.md](01-problem.md) — The earlier `#providers` list is a static composition point; no place for module-level extension surface (`#defines`); two parallel ingress concepts (Provider + Module)
2. [02-design.md](02-design.md) — `#Platform` shape with `#registry` as sole dynamic ingress; computed `#known*` views and `#matchers` reverse index (multi-fulfiller allowed; runtime matcher resolves via predicate evaluation — D13 revised); `#PlatformMatch` design (per-deploy walker, implemented in Go rather than CUE); operator-driven registration via `ModuleRelease`; concurrent-write conflict surface
3. [03-schema.md](03-schema.md) — CUE definitions for `#Platform`, `#ModuleRegistration`, `#ComponentTransformer`, `#TransformerMap` with kebab-case Id constraint and component-scope demand walker. `#PlatformMatch` design block included for reference but not landed as CUE.
4. [04-decisions.md](04-decisions.md) — Decision log (D1–D18) + open questions (OQ5 answered by D13 revised)
5. [05-component-transformer-and-matcher.md](05-component-transformer-and-matcher.md) — `#ComponentTransformer` design narrative, runtime guarantee (D18), matcher algorithm pseudocode, worked Deployment example, v1alpha1 → v1alpha2 migration impact

## Applicability Checklist

- [x] `03-schema.md` — CUE definitions for `#Platform`, `#ModuleRegistration`, `#PlatformMatch`, `#ComponentTransformer` (D1–D18 incorporated)
- [x] `04-decisions.md` — Decision log including D13–D18 (multi-fulfiller allowed — D13 revised, enabled-hides, concurrent-write conflict, kebab Id, `#ComponentTransformer` redesign, runtime guarantee)
- [x] `05-component-transformer-and-matcher.md` — Transformer schema + matcher algorithm in one place
- [x] `experiments/002-platform-construct/` — Self-contained CUE harness validating every projection and constraint in `03-schema.md`
- [ ] `NN-pipeline-changes.md` — Go pipeline modifications (deferred — covered by follow-up runtime-fill enhancement; topo-sort algorithm is OQ6)
- [ ] `NN-module-integration.md` — Migration of existing provider packages (deferred — separate enhancement, OQ3)

## Scope

### In scope

- `#Platform` construct: identity, `type`, `#ctx` reference, `#registry`, computed views (`#knownResources`, `#knownTraits`, `#composedTransformers`, `#matchers.{resources, traits}`).
- `#ModuleRegistration` schema (pure projection of `#defines`; no install metadata).
- `#PlatformMatch` construct — per-deploy walker producing `matched` / `unmatched` / `ambiguous` against the consumer Module's Resource/Trait FQN demand.
- `#ComponentTransformer` schema and `#TransformerMap` — sole transformer primitive at this layer (D17).
- Matcher algorithm (component-scope fan-out) plus the runtime guarantee (D18).
- Static and runtime-fillable composition of `#registry` (runtime path: `opm-operator` reconciles `ModuleRelease` and FillPaths the Module value).
- Retirement of `#Provider` and the synthetic `#provider` shim — the matcher now consumes `#composedTransformers` + `#matchers` directly.

### Out of scope

- `#Environment` construct (004 — referenced from there).
- `#ctx` / `#PlatformContext` schema (004 — referenced from there).
- `#ContextBuilder` and module integration (004 — referenced from there).
- `#Claim` primitive, `#ModuleTransformer`, status writeback (`#resolution`), `#defines.claims`, `#knownClaims`, `#matchers.claims`, the Claim halves of `#PlatformMatch._demand` / `matched` / `unmatched` / `ambiguous` — all extensions in [005](../005-claims/).
- Runtime-fill mechanism (Strategy B–style Go injection) — declared in schema, mechanism in follow-up.
- Self-service catalog runtime API (`opm catalog list`, web UI, etc.).
- `#PolicyTransformer` registration (deferred — pending policy redesign).
- Migration of existing `opmodel.dev/opm/v1alpha2/providers/kubernetes` and other provider packages into `#Module` form.
- Multi-fulfiller resolution policy beyond predicate evaluation. Today: D13 (revised) allows multi-fulfiller and lets the Go runtime matcher disambiguate by per-candidate predicate evaluation. Predicate-insufficient cases (two transformers fulfil the same FQN and pass identical predicates against the same component) are a future-enhancement concern.
- Topological-sort algorithm for `#status` writeback ordering — delegated to Go pipeline (OQ6).

## Cross-References

| Document | Purpose |
| -------- | ------- |
| `CONSTITUTION.md` (repo root) | Core design principles governing all changes in this repository |
| `enhancements/004-module-context/` | Sibling — defines `#PlatformContext`, `#EnvironmentContext`, `#ModuleContext`, `#ContextBuilder`, `#Environment`. `#Platform.#ctx` is typed by 004's `#PlatformContext`. |
| `enhancements/005-claims/` | Sibling — extends this enhancement with `#Claim` primitive, `#ModuleTransformer`, status writeback, and `#defines.claims` |
| `enhancements/012-policy-redesign/` | Open exploration that will inform policy-layer integration |
| `apis/core/v1alpha2/provider.cue` | `#Provider` — **retired in this enhancement** (D12). File deleted; matcher migrates to `#composedTransformers` + `#matchers`. |
| `apis/core/v1alpha2/transformer.cue` | `#ComponentTransformer`, `#TransformerMap` — introduced in this enhancement (D17). 005 extends with `#ModuleTransformer` and widens `#TransformerMap` to the union. |
| `apis/core/v1alpha2/module.cue` | `#Module` — registered values flow through `#Platform.#registry` (Module shape introduced in 005) |
