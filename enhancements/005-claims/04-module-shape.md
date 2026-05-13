# `#Module` — Eight-Slot Flat Shape

> **Implementation status (2026-05-13).** `apis/core/v1alpha2/module.cue` currently exposes 5 of the 8 slots: `metadata`, `#components`, `#defines`, `#config`, `debugValues`. The three remaining slots — `#claims`, `#lifecycles`, `#workflows` — and the `#ctx` runtime-injected field are design-only. Inside `#defines`, three of the four sub-maps ship (`resources`, `traits`, `transformers`); `defines.claims` is explicitly deferred (`module.cue:50` comment).

This file is the topical narrative for the `#Module` construct's top-level shape: which slots exist, why the count is fixed at eight, what was rejected, and what a Module *is not* required to declare. Decisions referenced here live in `10-decisions.md` under the `MS-` prefix; open questions in `11-open-questions.md`.

For the `#defines` slot (one of the eight), see [`05-defines-channel.md`](05-defines-channel.md).
For the `#claims` slot and `#Claim` primitive, see [`06-claim-primitive.md`](06-claim-primitive.md).
For the transformer types that ship under `#defines.transformers`, see [`07-claim-fulfilment.md`](07-claim-fulfilment.md).

## Goals

- One Module type that covers Applications, API descriptions, and Operators (no kind discrimination).
- Bounded top-level field set that cannot grow when a new ecosystem primitive lands.
- Each slot name predicts what's inside; no grouping nesting.
- Module body declares what it *uses*; the platform detects what it *needs*.

## The Eight Slots

```text
nucleus       metadata        # identity
              #config         # parameter / API schema
              debugValues     # example concrete values
              #components     # body — what is built

inward        #lifecycles     # state transitions for the module / its components
              #workflows      # on-demand operations

outward       #claims         # ecosystem-supplied needs (instance form)
              #defines        # publication channel — types and rendering extensions
                              #   this module ships to the ecosystem
                              #   (resources, traits, claims, transformers)
```

Three groups are descriptive, not structural — the eight fields are flat siblings on `#Module`. Bare app fills four (`metadata`, `#config`, `debugValues`, `#components`); operator Module fills six (adds `#lifecycles` + `#defines.transformers`); publication-only Module fills two (`metadata` + `#defines`).

`#Module` also carries `#ctx: ctx.#ModuleContext` as a runtime-injected definition field (enhancement 004). It is **not** a top-level slot in the same sense — it is the runtime-supplied counterpart to `#config`, computed by `#ContextBuilder` and unified into the module before `#components` evaluate. Authors read `#ctx.runtime.*` and `#ctx.platform.*` inside `#components` but never assign it directly.

## Triple-Duty Framing

Same `#Module` type covers three usage patterns:

| Pattern | Filled slots | Example |
|---|---|---|
| Application | `metadata`, `#config`, `debugValues`, `#components` (+ `#claims` for needs) | A web app deploys workloads, optionally claims a database |
| API description | `metadata`, `#config`, `#defines.{resources,traits,claims,…}` | A vendor publishes type definitions for the ecosystem |
| Operator | `metadata`, `#components` (controller + CRDs + RBAC), `#lifecycles`, `#defines.transformers` (the rendering rule that fulfils its claim) | Postgres operator ships its CRDs as components and the transformer that renders `#ManagedDatabaseClaim` requests |

A single `#Module` may do all three at once. A Postgres operator that also publishes the `#ManagedDatabaseClaim` *type* and ships an example app inline is a legitimate, expected case. Splitting into discriminated kinds (`#AppModule` / `#APIModule` / `#OperatorModule`) was rejected — see **MS-D1**.

## Why Flat Instead Of Grouped

The early design considered grouping the slots — `#aspects` (open kind-discriminated map), `#contract` + `#runtime`, governance / contracts / operations. All rejected. The rationale (**MS-D2**):

- The kind set is bounded (eight, after the supersession history). Grouping earns its keep when the list is unbounded.
- Each field name already predicts what's inside (`#components` is the body, `#claims` is needs, `#defines` is publication).
- Nesting adds an authoring step (`#aspects.something.X` instead of `#X`) without adding clarity.

The flat rule has one controlled exception: `#defines` is itself a sub-map keyed by primitive kind (`resources`, `traits`, `claims`, `transformers`). The exception is justified because `#defines` is *meta* (about the module-as-publisher) and the kind set inside it is bounded — see **DEF-D1**. (Blueprints are deliberately absent — see **DEF-D6**.)

## What Was Removed

- **`#policies` slot (was field nine).** Removed by **MS-D4**. `#Policy`, `#PolicyRule`, `#Directive` move with the policy redesign in enhancement 012; this enhancement does not surface a forwarding slot. Modules previously authoring `#policies` must drop them; behaviour reattaches via 012.
- **`#apis` slot.** Removed by **CL-D14**. `#Api` was the supply-side wrapper paired with `#Claim`; with transformer `requiredClaims` now carrying capability registration (TR-D5), the wrapper is redundant. See [`06-claim-primitive.md`](06-claim-primitive.md) for the supersession chain.
- **`#Action` from top level.** Removed by **MS-D3**. `#Action` is a primitive consumed by `#Lifecycle` and `#Workflow` constructs; module authors do not write Actions in isolation.

The ninth-slot count from MS-D2's original wording (which listed `#policies` and `#apis`) reduces to eight after MS-D4 + CL-D14.

## What Authors Don't Declare — `#requires` Is Absent By Design

Module authors do **not** declare platform compatibility — no `platformType`, no `resourceTypes`, no `criticality`. The eight slots stay at eight; there is no `#requires` field (**MS-D5**).

Mechanism: at deploy time the platform's matcher walks the module body for FQN usage —

- Resource / Trait FQNs from `#components[].#resources` / `#components[].#traits`
- Claim FQNs from `#claims` (module-level) and `#components[].#claims` (component-level)

— and looks each up in the platform's `#composedTransformers` (003). Unmatched FQNs are reported. The module is not asked to predict what platforms it might run on; the platform tells it what it can and cannot fulfil.

Policy for unmatched FQNs (deploy fails, warns, drops, criticality-driven escalation) is platform-team concern, not module-author concern. That policy is deferred until the catalog `#Policy` redesign (012) converges. See **MS-D5** for the full rationale and 003 D8 for the matcher mechanism.

## Worked Shapes

### Bare application

```cue
webApp: core.#Module & {
    metadata: {
        modulePath: "example.com/apps"
        name:       "web-app"
        version:    "0.1.0"
    }
    #config: { replicas: int | *2 }
    #components: {
        web: {...}
    }
}
```

Four slots filled. The other four are absent.

### Operator (controller + CRDs + transformer)

```cue
postgresOperator: core.#Module & {
    metadata: { modulePath: "vendor.com/operators", name: "postgres", version: "0.5.0" }
    #components: {
        controller: {...}     // operator Deployment
        crds:       {...}     // #CRDsResource
        rbac:       {...}
    }
    #lifecycles: install: {...}
    #defines: transformers: {
        (mdt.metadata.fqn): mdt   // ManagedDatabaseTransformer
    }
}
```

Six slots filled. CRDs ship in `#components` via `#CRDsResource` (CL-D8 operative half), not via the deprecated `#Api` wrapper.

### Publication-only catalog

```cue
opmCore: core.#Module & {
    metadata: { modulePath: "opmodel.dev/opm/v1alpha2", name: "opm-kubernetes-core", version: "0.1.0" }
    #defines: {
        resources:    { ... }
        traits:       { ... }
        transformers: { ... }
    }
}
```

Two slots filled. No runtime footprint; pure type publication.

See [`08-examples.md`](08-examples.md) for the full set of seven worked examples covering apps, operators, claim-only modules, publication-only modules, and operational commodities.

## Decisions Referenced

- **MS-D1** — Single `#Module` type, no kind discrimination.
- **MS-D2** — Flat field structure (eight slots).
- **MS-D3** — Remove `#Action` from top level.
- **MS-D4** — Remove `#policies` slot (defer to 012).
- **MS-D5** — No `#requires` slot; matcher detects compatibility.

Cross-topic decisions that refine the slot list:

- **CL-D14** — Remove `#Api` primitive (drops `#apis` slot).
- **DEF-D1** — Add `#defines` slot.

Full text in [`10-decisions.md`](10-decisions.md).

## Open Questions

- **MS-Q1** — Migration plan for existing Modules carrying `#policies` (deferred until 012 lands).

Full list in [`11-open-questions.md`](11-open-questions.md).
