# Design — `#Claim` Primitive, `#ModuleTransformer`, Module Extension Surface

This is the high-level overview. Topical narratives live in dedicated files; consult them when you need the rationale, schema sketches, examples, and supersession history of a specific area.

[003](../003-platform-construct/) introduces `#Platform`, `#registry`, `#composedTransformers`, `#matchers.{resources,traits}`, `#PlatformMatch`, `#ComponentTransformer`, the matcher algorithm, and the `#defines.{resources,traits,transformers}` slot. **This enhancement extends every one of those views with the Claim half** plus introduces `#Module` flat shape, `#Claim` primitive, `#ModuleTransformer`, and `#resolution`.

| Topic | Narrative file |
|---|---|
| The eight-slot flat `#Module` shape | [`04-module-shape.md`](04-module-shape.md) |
| The `#defines` publication channel (003 introduces three sub-maps; 005 extends with `claims`) | [`05-defines-channel.md`](05-defines-channel.md) |
| The `#Claim` primitive and `#status` resolution | [`06-claim-primitive.md`](06-claim-primitive.md) |
| `#ModuleTransformer` schema, `#resolution` writeback, dual-scope examples | [`07-claim-fulfilment.md`](07-claim-fulfilment.md) |
| `#ComponentTransformer` schema + matcher algorithm | [003/05-component-transformer-and-matcher.md](../003-platform-construct/05-component-transformer-and-matcher.md) |
| Worked examples (seven Modules covering apps, operators, publication-only, commodities) | [`08-examples.md`](08-examples.md) |
| Litmus / docs updates landing alongside this enhancement | [`09-litmus-updates.md`](09-litmus-updates.md) |
| Decision audit log | [`10-decisions.md`](10-decisions.md) |
| Open questions | [`11-open-questions.md`](11-open-questions.md) |

## Design Goals

- Keep `#Module` a single type that covers Applications, API descriptions, Operators, and publication-only catalogs uniformly.
- Bound `#Module`'s top-level field set to a fixed, predictable list. New ecosystem primitives must not require `#Module` schema changes.
- Provide a primitive surface for ecosystem-extensible needs (`#Claim`) that vendors can extend without catalog changes, with a `#status` channel for resolution data flowing back to consumers.
- Provide a publication channel (`#defines`) so a Module can ship type definitions and rendering extensions to the ecosystem in a discoverable way.
- Preserve the App/API duality via `#config` and make Operator Modules a natural fit (controller + CRDs + transformer ships rendering of fulfilled Claims).
- Sharpen the litmus test so that `#Resource` and `#Claim` answer distinct questions and authors can pick the right primitive without reading source.

## Non-Goals

- Splitting `#Module` into kind-discriminated variants (`#AppModule`, `#APIModule`, `#OperatorModule`).
- Defining the deploy-time runtime that matches `#claim` requests to fulfilling transformers. This design specifies declarative shape only; the platform runtime is free to populate a self-service catalog, a deploy-time match cache, both, or anything equivalent.
- CRD installation semantics. CRDs continue to deploy via `#CRDsResource` inside `#components`.
- A wrapper primitive for capability registration. Fulfilment is registered by the transformer's `requiredClaims` field (on `#ComponentTransformer` or `#ModuleTransformer` depending on Claim placement), not by a separate primitive.
- Module-level declaration of platform compatibility. Compatibility detection lives in the matcher (003 D8) — modules do not carry a `#requires` slot. Policy for unmatched FQNs is deferred until the catalog `#Policy` redesign (012) converges.
- Migration tooling. Existing `#Module`s using `#policies` must validate against the new shape; the slot is removed in this enhancement and migration is addressed in a follow-up.
- Resolving the cross-component noun grammar from enhancement 012 in full. This enhancement provides the noun answer at module/component scope; module-spanning shared nouns (mesh tenant, identity domain) may still need 012's work.

## High-Level Approach

`#Module` is a flat struct with eight slots in three descriptive groups:

```text
nucleus       metadata        # identity
              #config         # parameter / API schema
              debugValues     # example concrete values
              #components     # body — what is built

inward        #lifecycles     # state transitions for the module / its components
              #workflows      # on-demand operations

outward       #claims         # ecosystem-supplied needs (instance form)
              #defines        # publication channel — type definitions and rendering
                              #   extensions this module ships to the ecosystem
                              #   (resources, traits, claims, transformers)
```

`#Action` is not a top-level slot — it is consumed by `#Lifecycle` and `#Workflow` constructs. `#policies` is removed — `#Policy`, `#PolicyRule`, and `#Directive` are deferred to enhancement 012. `#apis` is removed — capability fulfilment now flows through transformer `requiredClaims`. The full slot rationale, the supersession history, and the no-`#requires` decision live in [`04-module-shape.md`](04-module-shape.md).

`#Claim` is a new primitive that defines the shape of a need (`#spec`) and the resolution channel that flows back from the fulfilling transformer (`#status`). The same primitive serves as both type definition (in catalog or vendor packages, or published via `#defines.claims`) and request (in `#claims`). Identity travels via `apiVersion` + `metadata.fqn`. `#Claim` may be placed at component level (data-plane needs — DB, queue, cache) or module level (platform-relationship needs — DNS, identity, mesh tenant). The full Claim story — identity, placement, triplet/quartet pattern, `#status` lifecycle, capability fulfilment — lives in [`06-claim-primitive.md`](06-claim-primitive.md).

There is no separate `#Api` wrapper primitive. An "API" in OPM is expressed either as a set of `#Resource` / `#Trait` definitions (consumed via component composition; rendered by catalog transformers) or as a `#Claim` definition (resolved at deploy time by any registered transformer whose `requiredClaims` includes the Claim's FQN). Both ship through `#defines`. Two transformer kinds carry fulfilment: `#ComponentTransformer` (component-level Claims; introduced by [003 D17](../003-platform-construct/04-decisions.md), extended here with `requiredClaims` / `optionalClaims`) and `#ModuleTransformer` (module-level Claims; introduced here). The `#ComponentTransformer` schema and matcher algorithm live in [003/05-component-transformer-and-matcher.md](../003-platform-construct/05-component-transformer-and-matcher.md); the `#ModuleTransformer` schema, dual-scope (`requiresComponents`) gate, and `#resolution` writeback channel live in [`07-claim-fulfilment.md`](07-claim-fulfilment.md).

`#defines` is the platform-facing publication channel. A Module that publishes new Resource / Trait / Claim type definitions or ships transformers lists them under `#defines`, keyed by FQN. Consumer Modules import the CUE packages directly to reference definitions; `#defines` is the discovery surface, not the consumption surface. `#Blueprint` is *not* publishable through `#defines` (DEF-D6) — Blueprints are CUE-import sugar with no platform-level consumer. Full publication-channel story lives in [`05-defines-channel.md`](05-defines-channel.md).

`#Resource` and `#Claim` differ on a sharp axis: `#Resource` is **catalog-fixed and transformer-rendered**; `#Claim` is **ecosystem-extended and provider-fulfilled**. The sharpened litmus reflects this in the docs (CL-D12); see [`09-litmus-updates.md`](09-litmus-updates.md).

`#Module` also carries `#ctx: core.#ModuleContext` as a runtime-injected definition field (enhancement 004) — the runtime-supplied counterpart to `#config`, computed by `#ContextBuilder` and unified into the module before components evaluate. Authors read `#ctx.runtime.*` and `#ctx.platform.*` inside `#components` but never assign it directly. See enhancement 004 for `#ctx` design.

## Schema Reference

The CUE definitions for `#Module`, `#Claim`, `#ComponentTransformer`, and `#ModuleTransformer` live in [`03-schema.md`](03-schema.md). That file is the type-level reference; topical narratives carry the rationale.

## File Layout

```text
apis/core/v1alpha2/
├── module.cue          // #Module (modified — eight slots, #defines, #ctx)
├── claim.cue           // #Claim (new primitive)
└── transformer.cue     // #ComponentTransformer + #ModuleTransformer (replaces v1alpha1 #Transformer)
```

Concrete commodity Claim definitions ship under `modules/opm/claims/`; vendor specialty Claims ship in their own packages following the same triplet / quartet pattern.

## Where The Major Decisions Live

- Eight-slot shape, no `#requires`: `MS-D1` … `MS-D5` in [`10-decisions.md`](10-decisions.md).
- `#defines` shape: `DEF-D1` … `DEF-D5`.
- `#Claim` identity, placement, `#status`, `#Api` removal: `CL-D1` … `CL-D16` (CL-D1, CL-D5, CL-D9, CL-D13 superseded by CL-D14).
- Two transformer primitives + matcher + runtime guarantee: `TR-D1` … `TR-D7` (TR-D1 … TR-D4 superseded by TR-D5).

Cross-document references in 003 and 004 use the same prefix scheme.
