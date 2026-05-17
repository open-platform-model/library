# Design Package: Platform Capabilities

## Summary

Introduces `#Capability` — an FQN-identified, schema-bearing construct for platform-supplied context, a sibling to `#Resource` and `#Trait`. A `#Platform` `#provides` concrete capability instances; a `#Module` declares which capabilities it `#consumes`; the matched provider value is unified back into `#consumes` so module bodies read `#consumes.required[fqn].spec.X` directly. This replaces the untyped `#ctx.platform` open struct that earlier drafts of 004 carried (extracted by 004 D36). Capabilities get schema, validation, collision detection, and discovery **without** a monolithic central schema — each capability is independently FQN-versioned, exactly as `#Resource` and `#Trait` are today.

`#Capability` is *not* a reuse of `#Resource`. A `#Resource` is a render **output** — transformers match it and emit Kubernetes. A `#Capability` is a render **input** — the platform supplies it, the module reads it, nothing renders it. The pattern transfers (FQN identity, OpenAPIv3 `spec`, FQN-keyed map unification); the type does not.

`#ModuleRelease` gains `#platform: #Platform` as a **kernel-populated** field — end-users authoring a release never write `#platform`; the runtime fills it at apply time. Same precedent as `#TransformerContext.#runtimeName!` (`transformer.cue:121`). The release artifact stays portable; the binding is a runtime decision.

The `#Environment` construct is **not** reintroduced in this enhancement (it was removed by 004 D36). Per-platform variation is handled via plain CUE unification of `#Platform` values (`#KindDev: #KindBase & {#provides: {...}}`); formalizing that pattern is OQ6.

## Documents

1. [01-problem.md](01-problem.md) — The `#ctx.platform` open struct has no schema, no identity, no contract; platform extensions are convention-only, with no discovery and no collision handling
2. [02-design.md](02-design.md) — `#Capability` construct; `#Platform.#provides`; `#Module.#consumes`; kernel-populated `#ModuleRelease.#platform`; `#ContextBuilder` capability-matching step; `#consumes` as both declaration and read surface
3. [03-schema.md](03-schema.md) — CUE definitions for `#Capability`, `#CapabilityMap`, `#consumes`, `#provides`, the `#ContextBuilder` extension, and the `#ModuleRelease` field
4. [04-decisions.md](04-decisions.md) — Decision log

## Scope

### In scope

- `#Capability` construct (FQN identity, OpenAPIv3 `spec`) — new file `apis/core/v1alpha2/capability.cue`.
- `#Module.#consumes` — `required` / `optional` capability declarations; doubles as the read surface once matched.
- `#Platform.#provides` — concrete capability instances (touches 003's `#Platform`).
- `#ModuleRelease.#platform: #Platform` — kernel-populated; end-users do not author it (touches 004's `#ModuleRelease`).
- `#ContextBuilder` capability-matching step — extends 004's slimmed builder with `#platform` + `#consumes` inputs and an `out.consumes` output that `#ModuleRelease` unifies back into `#module.#consumes`.

### Out of scope

- `#ctx.runtime` (release/module identity, per-component DNS names) — owned by 004; identity-only after the slim (004 D36).
- `#ctx.capabilities` as a typed second `#ctx` layer. Not introduced — `#consumes` is both declaration and resolved read surface, mirroring `#Component.#resources` precedent. `#ModuleContext` stays single-layer.
- `#Environment` construct. 004 D36 removed it; 006 does not reintroduce it. Per-platform variation uses CUE unification of `#Platform` values (OQ6).
- Reintroducing `route` and the cluster-domain override as shipped capability definitions — deferred (OQ1).
- Publishing capabilities through `#Module.#defines` / `#Platform.#registry` — deferred (OQ2).
- Component-scoped ("Trait-flavoured", `appliesTo`) capabilities — deferred (OQ3).
- Transformers consuming capabilities / `#TransformerContext` unification — deferred (OQ4).
- Bundle-level capability provision (module A provides for module B) — deferred (OQ5).
- Go pipeline changes — none anticipated; matching is entirely CUE-side.

## Cross-References

| Document | Purpose |
| -------- | ------- |
| `CONSTITUTION.md` (repo root) | Core design principles |
| `enhancements/004-module-context/` | Parent — owns identity-only `#ctx.runtime`, `#ComponentNames`, the `#ContextBuilder` core, and the `#ModuleRelease` 3-step flow. 006 extends `#ContextBuilder` (adds `#platform` + `#consumes` inputs) and adds the kernel-populated `#platform` field on `#ModuleRelease`. 004 D36 records the slim that made room for both. |
| `enhancements/003-platform-construct/` | `#Platform` gains a `#provides` map |
| `apis/core/v1alpha2/resource.cue`, `trait.cue` | Prior art — the FQN-identified, schema-bearing primitive pattern that `#Capability` mirrors |
| `apis/core/v1alpha2/component.cue` | Prior art — `#resources` as both declaration and read surface; the pattern `#consumes` follows |
| `apis/core/v1alpha2/transformer.cue` | Prior art — `#TransformerContext.#runtimeName!` as the kernel-populated-field pattern |

## Applicability Checklist

- [x] `03-schema.md` — New CUE definitions for `#Capability`, `#consumes`, `#provides`, kernel-populated `#platform`, and the `#ContextBuilder` extension
- [ ] `NN-pipeline-changes.md` — Go pipeline modifications. **Status revised by [experiments/02 finding F1](experiments/02-read-portability-fillpath/README.md#f1--contextbuilder-cannot-be-invoked-inside-modulerelease-revises-d5):** the runtime must orchestrate the `#ContextBuilder` call at top level and `FillPath` the matched `#consumes` entries (not just `#platform`). Small Go surface, but no longer "none anticipated".
- [ ] `NN-module-integration.md` — Module-author migration of `#ctx.platform`-style reads onto `#consumes` (deferred to a follow-up)
- [x] `experiments/` — Two experiments concluded 2026-05-15. [01-matcher-mechanics](experiments/01-matcher-mechanics/README.md) validates the outcome matrix and OQ6 inheritance. [02-read-portability-fillpath](experiments/02-read-portability-fillpath/README.md) validates D7 read surface, D13 `#`-prefix exclusion + FillPath, and surfaces F1 (the in-CUE inline-CB-then-unify-back chain cannot be evaluated due to a fixed-point cycle).
