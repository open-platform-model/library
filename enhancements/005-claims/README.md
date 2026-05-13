# Design Package: `#Claim` Primitive, `#ModuleTransformer`, and Module Extension Surface

| Field       | Value            |
| ----------- | ---------------- |
| **Status**  | Draft            |
| **Created** | 2026-04-28       |
| **Authors** | OPM Contributors |

## Summary

> **Implementation status (2026-05-13).** Of this enhancement, only the `#defines` slot ships in v1alpha2 — and only its three non-Claim sub-maps. `apis/core/v1alpha2/module.cue:51-60` exposes `#defines.{resources, traits, transformers}`. `module.cue:50` carries an explicit comment deferring `#defines.claims` to a follow-up. Everything else described below remains design-only: the `#Claim` primitive, `#ModuleTransformer`, `requiredClaims` / `optionalClaims` on `#ComponentTransformer`, the `#resolution` writeback channel, the `#claims` / `#lifecycles` / `#workflows` module slots, and the `#Component` extensions (`metadata.resourceName`, `#names`, `#claims`). The eight-slot `#Module` shape is partly realised — five slots are present (`metadata`, `#components`, `#defines`, `#config`, `debugValues`); the other three (`#claims`, `#lifecycles`, `#workflows`) are not yet wired.

Restructures `#Module` into a flat, bounded set of eight fields and introduces the `#Claim` primitive. Extends sibling enhancement [003's](../003-platform-construct/) `#defines` publication channel with a `claims` sub-map, `#ComponentTransformer` with `requiredClaims` / `optionalClaims`, and `#TransformerMap` to the `#ComponentTransformer | #ModuleTransformer` union. Adds `#ModuleTransformer` (per-module fan-out, dual-scope via `requiresComponents` gate) and the `#resolution` writeback channel. Together these layer the demand and ecosystem-publication surfaces for OPM's commodity and specialty service ecosystem on top of 003's matcher.

`#Module` keeps a small nucleus (`metadata`, `#config`, `debugValues`, `#components`), two inward slots (`#lifecycles`, `#workflows`), one outward instance slot (`#claims`), and one outward publication slot (`#defines`). `#Action` is removed from the top level since it is consumed by Lifecycle and Workflow internally. `#policies` is removed — `#Policy`, `#PolicyRule`, and `#Directive` are deferred to the policy redesign (enhancement 012). `#apis` is removed — capability fulfilment now flows through transformer `requiredClaims`. There is no `#requires` slot — platform compatibility is detected by the matcher (003 D8).

`#Claim` is a primitive that defines the shape of a need (`#spec`) and the resolution channel that flows back from the fulfilling transformer (`#status`). The same primitive serves as the type definition (when authored in a catalog or vendor package), the request (when used inside `#claims`), and the published vocabulary entry (when listed in `#defines.claims`) via CUE unification. `#Claim` carries `apiVersion` and `path` metadata for traceability across module boundaries. There is no `type` string field — identity is structural through CUE references plus the metadata FQN. `#status` is the cross-runtime portability surface — same module, different fulfillers, target-appropriate resolution data.

Capability fulfilment is registered by transformers, not by a wrapper primitive. Two transformer kinds (003's `#ComponentTransformer` for component-level Claims, this enhancement's `#ModuleTransformer` for module-level Claims) carry a `requiredClaims` field declaring which Claim FQNs they fulfil. The platform's render pipeline matches consumer `#claims` requests to fulfilling transformers via that field, then injects the transformer's `#resolution` back into the Claim's `#status` for downstream consumption. There is no separate `#Api` primitive — APIs in OPM are expressed either as a set of `#Resource`/`#Trait` definitions (consumed via component composition) or as a `#Claim` definition (resolved at deploy time by a fulfilling transformer). Both forms are shipped via `#defines`.

003 introduces `#defines.{resources, traits, transformers}`; this enhancement adds the `claims` sub-map. Map keys are FQNs and are bound to the value's `metadata.fqn` by CUE unification (DEF-D2). Consumer Modules still import CUE packages directly to reference definitions; `#defines` is the discovery surface, not the consumption surface. The OPM core catalog (`opmodel.dev/opm/v1alpha2`) becomes a single `#Module` whose `#defines` aggregates every primitive the catalog ships, with `claims` filled by 005's published commodity vocabulary.

`#Blueprint` is intentionally not a `#defines` sub-kind — Blueprints are CUE-import sugar with no platform-level consumer (DEF-D6).

This enhancement targets `apis/core/v1alpha2/` and `modules/opm/`. The existing `v1alpha1` tree is frozen; nothing in this enhancement modifies it.

CRDs remain part of `#components` via the existing `#CRDsResource` pattern — operators ship CRDs through their `#components` slot exactly as they do today.

## Documents

The enhancement is split into high-level overview docs (`01`–`03`), four topical narrative files (`04`–`07`), a cross-cutting examples file (`08`), litmus updates (`09`), and the central decision/question logs (`10`–`11`).

1. [01-problem.md](01-problem.md) — Module field-bloat risk; Resource/Claim litmus overlap; missing publication channel for ecosystem participants
2. [02-design.md](02-design.md) — High-level overview pointing at topical narratives
3. [03-schema.md](03-schema.md) — CUE definitions for `#Module`, `#Claim`, `#ModuleTransformer`, plus the `#defines.claims` extension and `#ComponentTransformer` widening (`requiredClaims` / `optionalClaims`); field documentation tables
4. [04-module-shape.md](04-module-shape.md) — The eight-slot flat `#Module` shape: rationale, what was removed, why no `#requires`
5. [05-defines-channel.md](05-defines-channel.md) — `#defines` publication channel narrative: 003 introduces `resources` / `traits` / `transformers`; 005 extends with `claims`; FQN binding; what's excluded
6. [06-claim-primitive.md](06-claim-primitive.md) — `#Claim` primitive: identity, placement, triplet/quartet pattern, `#status` resolution, capability fulfilment, supersession history of `#Api`
7. [07-claim-fulfilment.md](07-claim-fulfilment.md) — `#ModuleTransformer` schema (per-module fan-out), `requiresComponents` pre-fire gate, `#ComponentTransformer` Claim-key extension, `#TransformerMap` widening, `#resolution` channel + writeback flow, worked module-scope and dual-scope examples. Component-scope schema and matcher algorithm live in [003/05-component-transformer-and-matcher.md](../003-platform-construct/05-component-transformer-and-matcher.md).
8. [08-examples.md](08-examples.md) — Seven worked Modules: app-with-claim, module-level-claim, operator-with-transformer, specialty-vendor, claim-only, OPM-core publication-only, operational-commodity (backup)
9. [09-litmus-updates.md](09-litmus-updates.md) — Updates to `docs/core/definition-types.md` litmus questions and decision flowchart
10. [10-decisions.md](10-decisions.md) — All design decisions, grouped by topic (`MS-`, `DEF-`, `CL-`, `TR-` prefixes); `(was D#)` cross-refs to the prior chronological numbering
11. [11-open-questions.md](11-open-questions.md) — Unresolved items, grouped by topic, with revisit triggers
12. [12-pipeline-changes.md](12-pipeline-changes.md) — Go pipeline contracts: `#resolution` channel, topological sort for `#status` writeback (CL-Q7), deploy-time `#spec` / `#status` validation (CL-Q3)

## Applicability Checklist

- [x] `03-schema.md` — CUE definitions for `#Claim`, the revised `#Module` (with CL-D18 constraint), the proposed `#Component` shape (adds `metadata.resourceName`, `#names`, `#claims`)
- [x] `08-examples.md` — Worked examples for App, Operator, Specialty, Claim-only, OPM-core Modules, operational commodity
- [x] `09-litmus-updates.md` — Documentation updates to `definition-types.md`
- [x] `10-decisions.md` — D17 (independent Claim fulfilment) + D18 (no duplicate module-level Claim FQN) added 2026-05-02
- [x] `11-open-questions.md` — CL-Q1 + CL-Q8 closed by D17/D18; TR-Q3 closed by 003 D12
- [x] `12-pipeline-changes.md` — Go pipeline contracts for `#resolution`, topological sort, deploy-time validation
- [ ] `NN-module-integration.md` — Migration guidance for existing modules (deferred — MS-Q1)

## Cross-References

| Document | Purpose |
| -------- | ------- |
| `CONSTITUTION.md` (repo root) | Core design principles governing all changes in this repository |
| `apis/core/v1alpha2/module.cue` | `#Module` definition — restructured by this enhancement (no `#policies` slot in v1alpha2) |
| `apis/core/v1alpha2/claim.cue` | `#Claim` primitive — introduced by this enhancement |
| `apis/core/v1alpha2/transformer.cue` | `#ComponentTransformer` introduced by 003; this enhancement adds `#ModuleTransformer`, widens `#TransformerMap`, and extends `#ComponentTransformer` with `requiredClaims` / `optionalClaims` (see 07-claim-fulfilment.md) |
| `apis/core/v1alpha2/resource.cue` | `#Resource` primitive — sibling primitive whose litmus is sharpened here |
| `apis/core/v1alpha1/primitives/directive.cue` | `#Directive` primitive (v1alpha1) — pattern followed by `#Claim` (apiVersion + metadata + `#spec`); `#Directive` itself is deferred to policy redesign |
| `modules/opm/resources/extension/crd.cue` | `#CRDsResource` — canonical CRD-shipping path for operator Modules |
| `docs/core/definition-types.md` | Litmus test and decision flowchart updated by this enhancement |
| `docs/core/primitives.md` | Primitive reference — gains `#Claim` entry |
| `enhancements/012-policy-redesign/` | Concurrent exploration of cross-component noun grammar; this enhancement provides the noun answer at module/component scope |
| `enhancements/003-platform-construct/` | **Parent** — introduces `#Platform`, `#registry`, `#composedTransformers`, `#matchers.{resources,traits}`, `#PlatformMatch`, `#ComponentTransformer`, the `#defines.{resources,traits,transformers}` slot, and the matcher algorithm (component-scope). This enhancement extends every one of those views with the Claim half. |
| `enhancements/004-module-context/` | Sibling — defines `#ctx` (`#ModuleContext`, `#PlatformContext`, `#EnvironmentContext`, `#ContextBuilder`, `#Environment`). `#Module.#ctx` is typed by 004's `#ModuleContext` and injected at release time by `#ContextBuilder`. |
