# `#defines` — Publication Channel

This file is the topical narrative for the `#defines` slot on `#Module`: what it surfaces to the platform, why it's a single slot rather than five flat siblings, what it accepts and what it doesn't, and how it's consumed.

**Ownership:** [003](../003-platform-construct/) introduces the slot mechanism and three sub-maps (`resources`, `traits`, `transformers`) — those are what 003's matcher consumes. This enhancement extends `#defines` with a fourth sub-map, `claims`, and adds the corresponding `#knownClaims` view to the platform aggregation. The narrative here covers the full final shape; sub-section ownership notes call out which half lives where.

Decisions referenced live in `10-decisions.md` under the `DEF-` prefix; open questions in `11-open-questions.md`.

For the `#Module` slot list overall, see [`04-module-shape.md`](04-module-shape.md).
For the `#Claim` primitive that's one of the publishable kinds, see [`06-claim-primitive.md`](06-claim-primitive.md).
For the `#ComponentTransformer` schema and matcher algorithm, see [003/05-component-transformer-and-matcher.md](../003-platform-construct/05-component-transformer-and-matcher.md).
For `#ModuleTransformer` + status writeback, see [`07-claim-fulfilment.md`](07-claim-fulfilment.md).

## Goal

Make every type definition and rendering extension a Module ships discoverable at platform-evaluation time, without source-scanning, without scattered slots, without forcing every primitive into its own top-level field.

## Shape

```cue
#defines?: {
    resources?:    [FQN=string]: #Resource    & {metadata: fqn: FQN}
    traits?:       [FQN=string]: #Trait       & {metadata: fqn: FQN}
    claims?:       [FQN=string]: #Claim       & {metadata: fqn: FQN}
    transformers?: [FQN=string]: (transformer.#ComponentTransformer | transformer.#ModuleTransformer) & {metadata: fqn: FQN}
}
```

Four sub-maps under one slot. Each sub-map keyed by FQN. Each entry's `metadata.fqn` is bound by CUE unification to the map key — typos fail at definition time. **003 introduces** `resources`, `traits`, `transformers`; **this enhancement adds** `claims`.

`#Blueprint` is intentionally **not** a sub-kind here — see **DEF-D6**. Blueprints are CUE-import sugar with no platform-level consumer (no transformer match key references a Blueprint FQN); registering them buys nothing.

## Why One Slot Instead Of Four Flat Siblings

Four sibling slots (`#publishedResources`, `#publishedTraits`, `#publishedClaims`, `#transformers`) was the obvious alternative. Rejected (**DEF-D1**):

- Pushes `#Module`'s top-level field count from eight to twelve+. Cognitive cost.
- Splits "what this Module ships to the ecosystem" across four locations. Platform aggregation logic walks four maps; CLI catalog tools have four lookup paths.
- Doesn't earn the flatness it claims — every sibling has the same role (publication), distinguished only by primitive kind, which is exactly what a sub-map keys on.

`#defines` bundles all four into one slot grouped by kind. The kind set is bounded (matches the publishable primitive list), so the grouping doesn't violate the flat-rule's intent — it just collects related publications under one name.

`#defines` is also a controlled exception to the flat-slot rule (MS-D2) because it's *meta* — about the module-as-publisher, not about the module's runtime shape.

## FQN Identity

Each sub-map enforces `key == value.metadata.fqn`:

```cue
resources?: [FQN=string]: #Resource & {metadata: fqn: FQN}
```

A typo in either the key or the value's metadata produces a CUE unification error. Single source of truth: the platform indexes by the same FQN the value declares (**DEF-D2**).

Alternatives rejected:

- **Convention only** (no enforcement). Typos and copy-paste errors silently produce broken discovery indexes.
- **Computed-key wrapper** (e.g. `#defines.resources` as a list with key derived from `metadata.fqn`). Violates the keyed-map contract that the Platform's computed views (003 `#knownResources` etc.) rely on; breaks CUE map-as-set unification.

## What `#defines` Surfaces

Each sub-map maps to a Platform-side view (via 003's `#registry` aggregation):

| Module slot | Platform view (003) | Role |
| --- | --- | --- |
| `#defines.resources` | `#knownResources` | Catalog of Resource types |
| `#defines.traits` | `#knownTraits` | Catalog of Trait types |
| `#defines.claims` | `#knownClaims` | Catalog of Claim types (commodity vocabulary) |
| `#defines.transformers` | `#composedTransformers` | Active rendering registry |

`#defines` is the **discovery surface**, not the **consumption surface**. Consumer Modules continue to import CUE packages directly to reference definitions — `import data "opmodel.dev/opm/v1alpha2/claims/data"`, not via `#defines` lookups. The slot exists so the Platform (and CLI catalog tools) can enumerate without source-scanning.

## Transformers Live Here Too

`#defines.transformers` is the only outward home for transformer values. There is no separate top-level `#transformers` slot on `#Module` (**DEF-D3**).

Rationale: transformers are a kind of definition the Module publishes to the ecosystem — the Platform consumes them through the same publication channel as types. Single channel = single aggregation path = single discovery view. The active-vs-passive distinction (transformers are consumed by the matcher; types are discovery-only) is preserved by the Platform's separate `#composedTransformers` view; the Module-side publication channel does not need to differentiate.

The slot type is the union `#ComponentTransformer | #ModuleTransformer` — both transformer kinds (TR-D5) ship through the same map.

## What's Not In `#defines` Yet

### `directives` is excluded

`#defines` initially exposes four sub-maps: `resources`, `traits`, `claims`, `transformers`. `directives` is **excluded** (**DEF-D4**). `blueprints` is also absent by design — see **DEF-D6**.

Reason: `#Directive` exists in an earlier design, but enhancement 012 (open exploration) may revise the policy / directive model. Locking a Directive publication channel before 012 converges risks committing to a shape we are about to change. When the policy redesign lands, a follow-up enhancement adds `directives` (or its successor) to `#defines` in a single contained change.

Same reasoning applies to Op / Action publication (enhancement 010, still draft).

### `#PolicyTransformer` is excluded

`#defines.transformers` is typed as the union `#ComponentTransformer | #ModuleTransformer` only. `#PolicyTransformer` is **not** registered through `#Module.#defines` until 012 converges (**DEF-D5**).

Same logic as DEF-D4: the policy-scope transformer's shape is in flux; locking it in now risks committing to a shape we are about to change.

## Three Roles `#Claim` Plays Across `#defines`

`#Claim` is the primitive most affected by the publication channel because the same primitive serves three distinct roles, distinguished by placement:

| Placement | Role |
|-----------|------|
| `#defines.claims["fqn"]: #Claim` | Module **publishes** this Claim type (catalog vocabulary) |
| `#claims.id: #Claim & {#spec: ...}` (component or module level) | Module **requests** an instance (demand) |
| `#defines.transformers["fqn"]: #ComponentTransformer & {requiredClaims: …}` (component-level) or `#ModuleTransformer & {requiredClaims: …}` (module-level) | Module **fulfils** this Claim type (supply, via the transformer's match keys) |

A single Module may do all three. The OPM core Module typically only does the first; an operator Module typically does the last (and ships any new Claim types it introduces under `#defines.claims`).

`#Resource` and `#Trait` types are publishable through `#defines`; instances of `#Resource` and `#Trait` continue to live inside `#Component` (component-internal). `#Blueprint` is **not** publishable through `#defines` (DEF-D6) — it is a CUE-only composition consumed by Components, instantiated by direct package import, with no platform-level aggregation.

## Worked Shapes

### Publication-only Module — OPM core catalog

The OPM core catalog (`opmodel.dev/opm/v1alpha2`) ships every well-known `#Resource` and `#Trait` definition plus the Kubernetes-provider transformers. (Blueprint definitions ship in their CUE packages and are imported directly by consumers — they are not aggregated in `#defines`; see DEF-D6.) Under the new shape, the entire catalog is packaged as a single Module whose only filled slot (besides `metadata`) is `#defines`:

```cue
opmKubernetesCore: core.#Module & {
    metadata: {
        modulePath: "opmodel.dev/opm/v1alpha2"
        name:       "opm-kubernetes-core"
        version:    "0.1.0"
    }
    #defines: {
        resources: {
            (container.#ContainerResource.metadata.fqn): container.#ContainerResource
            ...
        }
        traits:       {...}
        transformers: {...}
    }
}
```

No `#components`, no `#config`, no `#claims`. Pure publication.

### Claim-only Module — vendor publishes a type without fulfilling it

```cue
imageRegistryClaim: core.#Module & {
    metadata: {modulePath: "example.com/platform/v1alpha2", name: "image-registry-claim", version: "0.1.0"}
    #defines: claims: {
        (#ImageRegistryClaim.metadata.fqn): #ImageRegistryClaim
    }
}
```

Once registered with a Platform, the Claim type appears in `#Platform.#knownClaims` and is discoverable. Consumer requests stay unmatched until a fulfilling operator Module is also registered.

### Mixed Module — operator with components AND publication

```cue
postgresOperator: core.#Module & {
    metadata: {...}
    #components: {
        controller: {...}
        crds:       {...}
    }
    #lifecycles: install: {...}
    #defines: {
        // Publishes the Claim type (optional — could live in a separate Claim-only Module).
        claims: {
            (data.#ManagedDatabaseClaim.metadata.fqn): data.#ManagedDatabaseClaim
        }
        // Ships the transformer that fulfils it.
        transformers: {
            (#ManagedDatabaseTransformer.metadata.fqn): #ManagedDatabaseTransformer
        }
    }
}
```

See [`08-examples.md`](08-examples.md) Examples 5, 6, and 7 for full versions.

## Decisions Referenced

- **DEF-D1** — Add `#defines` slot for type publication.
- **DEF-D2** — Map keys bound to `value.metadata.fqn`.
- **DEF-D3** — `#defines` absorbs transformer publication; no separate `#transformers` slot.
- **DEF-D4** — Drop `directives` from initial sub-kinds.
- **DEF-D5** — `#PolicyTransformer` excluded from `#defines.transformers`.
- **DEF-D6** — `blueprints` removed from `#defines`; Blueprints are CUE-import sugar with no platform-level consumer.

Full text in [`10-decisions.md`](10-decisions.md).

## Open Questions

- **DEF-Q1** — Self-service catalog API surface (CLI, Web UI, deploy-time match resolver). Platform-implementation territory, deferred.

Full list in [`11-open-questions.md`](11-open-questions.md).
