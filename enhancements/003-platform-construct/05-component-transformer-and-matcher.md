# `#ComponentTransformer` and the Matcher Algorithm

| Field       | Value      |
| ----------- | ---------- |
| **Status**  | Draft      |
| **Created** | 2026-05-02 |
| **Targets** | `apis/core/v1alpha2/transformer.cue` |

## Why this doc exists

Retiring `#Provider` (D12) leaves two artefacts the matcher needs but `#Platform` schema doesn't fully describe on its own:

1. **The transformer primitive itself.** v1alpha1's `#Transformer` matched a single `#Component` and had a flat `requiredX`/`optionalX` shape. v1alpha2 needs a redesigned primitive that fits the matcher described in `02-design.md` and the schema in `03-schema.md`.
2. **The dispatch algorithm.** `#matchers` is the reverse index; `#PlatformMatch` is the per-deploy walker; neither describes how the Go pipeline iterates `#composedTransformers` and produces output. The algorithm belongs alongside the matcher schema.

This enhancement introduces **one** transformer primitive, `#ComponentTransformer`, that fires once per matching `#Component`. Match keys read against a single Component. Sibling enhancement [005](../005-claims/) extends this with `#ModuleTransformer` (per-module fan-out) and widens `#composedTransformers` to a union — that addition is structurally separable from the work here, so the component-scope shape lands cleanly first.

## Runtime guarantee

The runtime always invokes `#transform` with a **fully concrete `#ModuleRelease`** — every `#components`, `#config`, `#ctx` value resolved before the transformer fires. The transformer body can index into `#moduleRelease` freely. This guarantee shapes the schema: there is no need for the matcher to pre-filter components into a map for the body's convenience; the body indexes the matched `#component` directly and may walk `#moduleRelease.#components` itself for cross-component reads.

This guarantee also applies unchanged to 005's `#ModuleTransformer` once that primitive lands.

## Design constraints

- **CL-D11 (005)**: `#Resource` and `#Trait` stay component-only — never module-level. Match keys for component-scope transformers cover labels, resources, and traits.
- **DEF-D1 / DEF-D2 / DEF-D3 (003/005)**: transformers ship through `#Module.#defines.transformers` keyed by FQN. The map's value type is introduced as `[FQN]: #ComponentTransformer` here; 005 widens it to `[FQN]: #ComponentTransformer | #ModuleTransformer`.
- **D17 (this enhancement)**: there is exactly one transformer primitive at this layer. The single→two split is 005's extension.
- **D18 (this enhancement)**: runtime guarantee above — concrete `#ModuleRelease` on every fire.

## Schema — `#ComponentTransformer`

Fires **once per matching component**. Match keys read against a single `#Component`. The render body receives the `#ModuleRelease` plus the matched `#Component`.

```cue
// apis/core/v1alpha2/transformer.cue
package v1alpha2

#ComponentTransformer: {
    apiVersion: "opmodel.dev/core/v1alpha2"
    kind:       "ComponentTransformer"

    metadata: {
        modulePath!: #ModulePathType   // "opmodel.dev/opm/v1alpha2/providers/kubernetes"
        version!:    #MajorVersionType
        name!:       #NameType
        #definitionName: (#KebabToPascal & {"in": name}).out
        fqn: #FQNType & "\(modulePath)/\(name)@\(version)"
        description!: string
        labels?:      #LabelsAnnotationsType
        annotations?: #LabelsAnnotationsType
    }

    // Match keys — read against the candidate #Component.
    requiredLabels?:    #LabelsAnnotationsType
    optionalLabels?:    #LabelsAnnotationsType
    requiredResources?: [FQN=string]: _
    optionalResources?: [FQN=string]: _
    requiredTraits?:    [FQN=string]: _
    optionalTraits?:    [FQN=string]: _

    // Optional declarative metadata for catalog UIs and pipeline diff.
    readsContext?:  [...string]
    producesKinds?: [...string]

    // Render function. Runtime always supplies both inputs concretely (D18).
    #transform: {
        #moduleRelease: _              // fully concrete #ModuleRelease
        #component:     _              // the matched #Component (singular)
        #context:       #TransformerContext

        output: {...}
    }
}

#TransformerMap: [#FQNType]: #ComponentTransformer
```

005 widens `#TransformerMap` to `[#FQNType]: #ComponentTransformer | #ModuleTransformer` and adds `requiredClaims`/`optionalClaims` to `#ComponentTransformer`'s match keys.

## Publication slot

`#Module.#defines.transformers` (introduced in 003, extended in 005) is keyed by FQN with the same FQN-binding constraint as the other `#defines` sub-maps:

```cue
transformers?: [FQN=string]: #ComponentTransformer & {
    metadata: fqn: FQN
}
```

005 widens the value type to the union once `#ModuleTransformer` exists.

## Matcher algorithm

The matcher iterates `#composedTransformers` once per Module under render. With one primitive, dispatch is trivial: every transformer is a `#ComponentTransformer`, so the matcher fans out per matching component.

```text
function renderModule(moduleRelease, transformers, runtimeContext) -> [Output]:
    outputs = []
    for t in transformers:
        outputs.extend(matchAndRender(moduleRelease, t, runtimeContext))
    return outputs

function matchAndRender(moduleRelease, t, runtimeContext) -> [Output]:
    outputs = []
    for (name, cmp) in moduleRelease.#components.items():
        if not satisfiesComponent(cmp, t):
            continue
        ctx = buildContext(moduleRelease, runtimeContext, t, cmp)
        outputs.append(runRender(t, moduleRelease, cmp, ctx))
    return outputs
```

### `satisfiesComponent`

```text
function satisfiesComponent(cmp, t) -> bool:
    for (k, v) in t.requiredLabels or {}:
        if cmp.metadata.labels.get(k) != v: return False
    for fqn in t.requiredResources or {}:
        if fqn not in fqnsOf(cmp.#resources): return False
    for fqn in t.requiredTraits or {}:
        if fqn not in fqnsOf(cmp.#traits or {}): return False
    return True
```

005 extends `satisfiesComponent` with a `requiredClaims` check and adds a parallel `satisfiesModule` plus an `anyComponentMatches` gate for the new `#ModuleTransformer` kind. Dispatch in `matchAndRender` then branches on transformer `kind` to choose component-fanout vs once-per-module.

## Worked example — Deployment

```cue
#DeploymentTransformer: transformer.#ComponentTransformer & {
    metadata: { ... }

    requiredLabels:    "core.opmodel.dev/workload-type": "stateless"
    requiredResources: (workload.#ContainerResource.metadata.fqn): _

    #transform: {
        #moduleRelease: _
        #component:     _
        #context:       #TransformerContext

        output: {
            apiVersion: "apps/v1"
            kind:       "Deployment"
            metadata:   { name: #component.metadata.name, ... }
            spec:       { ... uses #component.spec ... }
        }
    }
}
```

`#component` is singular and concrete — body indexes into it directly.

## Migration impact (modules/opm)

When the well-known catalog is rebuilt under v1alpha2:

| Source | Change |
|---|---|
| Existing component-scope transformers (Deployment, Service, ConfigMap, ...) | Wrap as `#ComponentTransformer`; the v1alpha1 `#transform.#component` field is already singular; replace v1alpha1 imports |
| Module-scope transformers (Hostname, ExternalDNS, K8up backup, cert-manager, Gateway-API routing) | Wait for 005's `#ModuleTransformer` — those depend on the `#Claim` primitive that 005 introduces |

`cue vet` will flag any transformer in the v1alpha2 catalog that does not match `#ComponentTransformer` (or, after 005 lands, the union).

## What the 005 extension adds

For readers tracing the full picture: 005 layers on top of this with no rewrites — it adds a sibling primitive and widens the union. Concretely:

- `#Claim` primitive with `#spec` / `#status`.
- `#ModuleTransformer` schema (fires once per Module; match keys against module top level + optional `requiresComponents` pre-fire gate for dual-scope renders).
- `requiredClaims` / `optionalClaims` added to `#ComponentTransformer` for component-level Claim fulfilment.
- `#TransformerMap` widened to the union `#ComponentTransformer | #ModuleTransformer`.
- `#composedTransformers`, `#matchers.claims`, and `#PlatformMatch` claim-related fields land alongside the union widening.
- `#resolution` channel for transformer-to-Claim status writeback (CL-D15 / CL-D16).
- D7 / D8 / D13 each pick up Claim-shaped extensions in 005's decision log.

See [`005/06-claim-primitive.md`](../005-claims/06-claim-primitive.md) and [`005/07-claim-fulfilment.md`](../005-claims/07-claim-fulfilment.md) for the extension narratives.

## Decisions

Live in `04-decisions.md`:

- **D17** — `#ComponentTransformer` schema redesigned; single transformer primitive at this layer (005 introduces `#ModuleTransformer` as a sibling).
- **D18** — runtime always passes a fully concrete `#ModuleRelease` to `#transform`.

Cross-references: D7 (capability fulfilment for Resources/Traits via transformer match keys), D8 (matcher detects unmatched Resource/Trait FQNs), D12 (`#matchers` reverse index), D13 revised (multi-fulfiller allowed; runtime predicate evaluation disambiguates).
