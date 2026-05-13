# Schema — `#Platform` Construct

## Summary

Four CUE definitions land in `core/v1alpha2`:

- `#Platform` — `core/v1alpha2/platform.cue` (replaces the earlier `#Platform` schema)
- `#ModuleRegistration` — same file
- `#PlatformMatch` — same file (per-deploy match construct)
- `#ComponentTransformer` — `core/v1alpha2/transformer.cue` (replaces v1alpha1's single `#Transformer`)

`#Platform`, `#ModuleRegistration`, and `#PlatformMatch` cover Resource/Trait demand only at this layer. `#ComponentTransformer` is the sole transformer primitive introduced here — it fires once per matching `#Component`. Sibling enhancement [005](../005-claims/) extends the schema with `#Claim`, `#ModuleTransformer`, `#defines.claims`, status writeback, and the corresponding widenings of `#composedTransformers`, `#matchers`, and `#PlatformMatch`.

The `#Module` shape (eight-slot flat structure) and the `#defines.{resources, traits, transformers}` publication channel are introduced together with this enhancement and described in [005](../005-claims/05-defines-channel.md); 003 references the slots it consumes.

## `#Platform`

```cue
package v1alpha2

// #Platform: A deployment target's identity, context, and registered
// extensions. Composition unit is #Module (registered via #registry).
// All outward views are computed projections over #registry.
//
// `#ComponentTransformer` and `#TransformerMap` are siblings in the
// flat v1alpha2 package — no import needed. (005 adds `#ModuleTransformer`
// and widens `#TransformerMap` to the union.)
#Platform: {
    apiVersion: #ApiVersion
    kind:       "Platform"

    metadata: {
        name!:        #NameType
        description?: string
        labels?:      #LabelsAnnotationsType
        annotations?: #LabelsAnnotationsType
    }

    // Target type. Every registered Module's transformers must be
    // compatible with this type. (Compatibility check deferred — for now
    // type is informational; future enhancement may enforce. See OQ2.)
    type!: string

    // 004 (deferred): #ctx: #PlatformContext is not yet present on the
    // landed #Platform schema. When 004 lands, a Platform-level context
    // field will be added here.

    // The single dynamic ingress for platform extensions.
    // Filled by:
    //   1. Platform CUE file (admin-authored static registrations).
    //   2. Runtime injection — opm-operator reconciles ModuleRelease CRs
    //      and FillPaths the Module value into #registry[id].#module.
    //      Installation of #components and registration of #defines are
    //      a single operator-driven step (see D11).
    // Both sources unify by Id key.
    //
    // Id keys MUST be kebab-case (#NameType — D16). Convention is to set
    // Id to #module.metadata.name. Static + runtime writes to the same Id
    // unify; concrete-value disagreement produces _|_ at platform-eval
    // time (D15). The opm-operator reconciler surfaces such conflicts in
    // ModuleRelease.status.conditions.
    #registry: [Id=#NameType]: #ModuleRegistration

    // ---- Computed views over #registry ----

    // Catalog views — type definitions published by registered Modules.
    // Each is keyed by FQN. FQN collisions across Modules surface as CUE
    // unification errors (correct behaviour — forces conflict resolution
    // at registration time).
    //
    // The pattern constraint and the comprehension live as siblings inside
    // the struct (NOT as `[FQN=string]: T & {comprehension}` — that form
    // treats the comprehension as a per-value constraint and the map stays
    // empty). Verified by experiments/002 finding 2.
    #knownResources: {
        [FQN=string]: #Resource
        for _, reg in #registry
        if reg.enabled
        if reg.#module.#defines != _|_
        if reg.#module.#defines.resources != _|_
        for fqn, v in reg.#module.#defines.resources {
            (fqn): v
        }
    }

    #knownTraits: {
        [FQN=string]: #Trait
        for _, reg in #registry
        if reg.enabled
        if reg.#module.#defines != _|_
        if reg.#module.#defines.traits != _|_
        for fqn, v in reg.#module.#defines.traits {
            (fqn): v
        }
    }

    // #knownClaims is added by 005 once #defines.claims exists.

    // Active rendering registry — all transformers from all enabled Modules.
    // Capability fulfilment for Resources/Traits is registered implicitly:
    // a transformer whose requiredResources / requiredTraits includes a
    // primitive FQN is the supply registration for that primitive (D7).
    // 005 widens #TransformerMap to the union (#ComponentTransformer |
    // #ModuleTransformer) and adds the requiredClaims supply path.
    // Multi-fulfiller is allowed — see #matchers below; the runtime
    // matcher disambiguates per consumer component via predicate
    // evaluation (D13 revised).
    #composedTransformers: #TransformerMap & {
        for _, reg in #registry
        if reg.enabled
        if reg.#module.#defines != _|_
        if reg.#module.#defines.transformers != _|_
        for fqn, v in reg.#module.#defines.transformers {
            (fqn): v
        }
    }

    // ---- Match index (D12) ----

    // Reverse index from #composedTransformers' required* fields.
    // Key = FQN of a primitive that some transformer fulfils; value =
    // candidate transformers for that FQN. Empty/missing FQN at lookup
    // time = unmatched (D8). Multiple candidates per FQN are allowed —
    // the runtime matcher evaluates each candidate's predicate against
    // the consumer #Component and pairs every survivor (D13, revised).
    //
    // Resources and Traits are component-scope only (CL-D11); only
    // #ComponentTransformer can fulfil them. 005 adds a `claims` sub-map
    // populated from transformer requiredClaims (component-level via
    // #ComponentTransformer, module-level via #ModuleTransformer).
    //
    // Pre-compute candidate maps as `let` bindings. Iterating the published
    // `resources` / `traits` fields fails under `cue vet -c` with
    // "incomplete type list" because the field type is
    // `[FQN]: [...#ComponentTransformer]` — an open value-list — which CUE
    // refuses to range over. List comprehensions yield via `{t}` — the
    // trailing `t,` form is invalid CUE.
    #matchers: {
        let _resourceFqns = {
            for _, t in #composedTransformers
            if t.requiredResources != _|_
            for fqn, _ in t.requiredResources {
                (fqn): _
            }
        }
        let _traitFqns = {
            for _, t in #composedTransformers
            if t.requiredTraits != _|_
            for fqn, _ in t.requiredTraits {
                (fqn): _
            }
        }

        let _resourceCandidates = {
            for fqn, _ in _resourceFqns {
                (fqn): [
                    for _, t in #composedTransformers
                    if t.requiredResources != _|_
                    if t.requiredResources[fqn] != _|_ {t},
                ]
            }
        }
        let _traitCandidates = {
            for fqn, _ in _traitFqns {
                (fqn): [
                    for _, t in #composedTransformers
                    if t.requiredTraits != _|_
                    if t.requiredTraits[fqn] != _|_ {t},
                ]
            }
        }

        resources: {[FQN=string]: [...#ComponentTransformer]} & _resourceCandidates
        traits:    {[FQN=string]: [...#ComponentTransformer]} & _traitCandidates

        // D13 (revised): multi-fulfiller is allowed. Two transformers may
        // legitimately require the same primitive FQN — the runtime
        // matcher evaluates each candidate's predicate against the
        // consumer #Component and pairs every survivor. The CUE schema
        // therefore carries no `_invalid` / `_noMultiFulfiller` guard.
    }
}
```

## `#ModuleRegistration`

```cue
// #ModuleRegistration: A single entry in #Platform.#registry.
// Pure projection of "this Module's primitives are visible on this
// platform". Carries no install/deploy metadata — installation of
// #components is owned by ModuleRelease + opm-operator (D11).
#ModuleRegistration: {
    // The Module definition. Static path: imported from a CUE package.
    // Runtime path: FillPath-injected by opm-operator after a
    // ModuleRelease CR is reconciled.
    #module!: #Module

    // Enable/disable without removing the entry. Default true.
    // When false, every #Platform projection that walks #registry
    // (#knownResources, #knownTraits, #composedTransformers, #matchers —
    // and #knownClaims once 005 lands) skips this entry — the Module's
    // primitives are completely hidden from the platform until enabled
    // flips back to true (D14). Use case: stage a registration via
    // FillPath, leave it dark until a follow-up reconcile flips the flag.
    enabled: bool | *true

    // Optional self-service catalog metadata. Carries platform-curation
    // data (category/tags/examples) that #module.metadata cannot — i.e.
    // information about how this platform surfaces the Module, not about
    // the Module itself. Flat shape after D11 removed presentation.operator
    // and D14 dropped the redundant `template` wrapper.
    presentation?: {
        description?: string
        category?:    string
        tags?:        [...string]
        examples?: [Name=string]: {
            description?: string
            values:       _
        }
    }

    metadata?: {
        labels?:      #LabelsAnnotationsType
        annotations?: #LabelsAnnotationsType
    }
}
```

## `#PlatformMatch`

> **Status — design only, not in v1alpha2 CUE.** The per-deploy walker (demand collection, `matched` / `unmatched` / `ambiguous` projections) was not landed as a CUE construct. The equivalent logic is implemented in Go in `pkg/compile/` and `pkg/platform/`, which iterate the consumer `#Module` against `#Platform.#matchers` at compile time. The CUE block below is preserved as the design reference; treat it as documentation of intent, not a definition in the v1alpha2 schema.

Per-deploy match construct. The Go pipeline (or `opm-operator`) instantiates one `#PlatformMatch` per consumer `#Module` being deployed, walks the consumer's FQN demand against the platform's `#matchers`, and emits a render plan plus diagnostics.

```cue
// #PlatformMatch: Per-deploy walker. Resolves a consumer Module's FQN
// demand against #Platform.#matchers and surfaces matched / unmatched /
// ambiguous sets for the Go pipeline to act on.
#PlatformMatch: {
    platform!: #Platform
    module!:   #Module   // consumer Module being deployed

    // ---- Demand: FQNs the consumer Module reads ----
    //
    // 005 extends `_demand` with a `claims` sub-tree (module-level + per-
    // component) once #Claim instances exist on #Module.
    _demand: {
        resources: [FQN=string]: _
        resources: {
            if module.#components != _|_
            for _, c in module.#components
            if c.#resources != _|_
            for fqn, _ in c.#resources {
                (fqn): _
            }
        }

        traits: [FQN=string]: _
        traits: {
            if module.#components != _|_
            for _, c in module.#components
            if c.#traits != _|_
            for fqn, _ in c.#traits {
                (fqn): _
            }
        }
    }

    // ---- Lookup: candidate transformers per demanded FQN ----

    matched: {
        resources: [FQN=string]: [...#ComponentTransformer]
        resources: {
            for fqn, _ in _demand.resources
            if platform.#matchers.resources[fqn] != _|_ {
                (fqn): platform.#matchers.resources[fqn]
            }
        }

        traits: [FQN=string]: [...#ComponentTransformer]
        traits: {
            for fqn, _ in _demand.traits
            if platform.#matchers.traits[fqn] != _|_ {
                (fqn): platform.#matchers.traits[fqn]
            }
        }
    }

    // ---- Diagnostics ----
    //
    // FQNs the consumer demands but no transformer fulfils. D8 detection
    // signal. Response policy (fail / warn / drop) is platform-team
    // policy concern, deferred to 012.
    //
    // 005 adds `claims` to `matched`, `unmatched`, and `ambiguous` once
    // Claim demand exists on #Module.
    //
    // Membership tested against pre-built matched-FQN sets, not via
    // `matched.resources[fqn] == _|_` — that form errors with "undefined
    // field" under `cue vet -c` because `matched.resources` is typed
    // `[FQN]: [...#ComponentTransformer]`. (experiments/002 finding 5.)
    // List comprehensions yield via `{fqn}` — the trailing `fqn,` form is
    // invalid CUE (experiments/002 finding 1).
    unmatched: {
        let _matchedResourceSet = {
            for fqn, _ in matched.resources {(fqn): _}
        }
        let _matchedTraitSet = {
            for fqn, _ in matched.traits {(fqn): _}
        }
        resources: [
            for fqn, _ in _demand.resources
            if _matchedResourceSet[fqn] == _|_ {fqn},
        ]
        traits: [
            for fqn, _ in _demand.traits
            if _matchedTraitSet[fqn] == _|_ {fqn},
        ]
    }

    // FQNs with > 1 candidate. Under D13 (revised) multi-candidate is
    // normal — the Go runtime matcher resolves per consumer #Component
    // via predicate evaluation. This field is preserved in the design
    // as a diagnostic surface for cases where predicates do not
    // disambiguate (two transformers with identical predicates fulfil
    // the same FQN for the same component).
    ambiguous: {
        resources: {
            for fqn, ts in matched.resources
            if len(ts) > 1 {
                (fqn): ts
            }
        }
        traits: {
            for fqn, ts in matched.traits
            if len(ts) > 1 {
                (fqn): ts
            }
        }
    }
}
```

## `#ComponentTransformer`

Sole transformer primitive at this layer (D17). Fires once per matching `#Component`. See [`05-component-transformer-and-matcher.md`](05-component-transformer-and-matcher.md) for the full design narrative, runtime guarantee (D18), matcher algorithm, and worked example.

```cue
// apis/core/v1alpha2/transformer.cue
package v1alpha2

#ComponentTransformer: {
    apiVersion: #ApiVersion
    kind:       "ComponentTransformer"

    metadata: {
        modulePath!: #ModulePathType
        version!:    #MajorVersionType
        name!:       #NameType
        #definitionName: (#KebabToPascal & {"in": name}).out
        fqn: #FQNType & "\(modulePath)/\(name)@\(version)"
        description!: string
        labels?:      #LabelsAnnotationsType
        annotations?: #LabelsAnnotationsType
    }

    // Match keys — read against the candidate #Component.
    // Values reference the concrete primitive definition (not an
    // unconstrained marker) so authors must point at a real #Resource /
    // #Trait. 005 extends this set with requiredClaims / optionalClaims
    // for component-level Claim fulfilment.
    requiredLabels?:    #LabelsAnnotationsType
    optionalLabels?:    #LabelsAnnotationsType
    requiredResources?: [#FQNType]: #Resource
    optionalResources?: [#FQNType]: #Resource
    requiredTraits?:    [#FQNType]: #Trait
    optionalTraits?:    [#FQNType]: #Trait

    readsContext?:  [...string]
    producesKinds?: [...string]

    // Runtime always supplies both inputs concretely (D18).
    #transform: {
        #moduleRelease: _
        #component:     _
        #context:       #TransformerContext

        output: {...}
    }
}

// 005 widens this to `#ComponentTransformer | #ModuleTransformer`.
#TransformerMap: [#FQNType]: #ComponentTransformer
```

## Field Documentation

### `#Platform`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `apiVersion` | `"opmodel.dev/core/v1alpha2"` | fixed | OPM core |
| `kind` | `"Platform"` | fixed | Always `"Platform"` |
| `metadata.name` | `#NameType` | yes | Platform identifier (kebab-case) |
| `metadata.description` | `string` | no | Human-readable summary |
| `type` | `string` | yes | Target type (`"kubernetes"`, future: `"docker-compose"`, etc.) |
| `#registry` | `[Id=#NameType]: #ModuleRegistration` | yes | Registered Modules (static + runtime). Id MUST be kebab-case (D16). Runtime entries are FillPath-injected by `opm-operator` from `ModuleRelease` CRs (D11). Static + runtime writes to the same Id unify via CUE; concrete-value disagreement = `_\|_` surfaced by reconciler (D15). |
| `#knownResources` | `[FQN=string]: #Resource` | computed | Resource types from `#registry[*].#module.#defines.resources` (only entries where `enabled: true` per D14). |
| `#knownTraits` | `[FQN=string]: #Trait` | computed | Trait types from enabled `#registry[*].#module.#defines.traits`. |
| `#composedTransformers` | `#TransformerMap` | computed | All transformers from enabled `#registry[*].#module.#defines.transformers`, keyed by FQN. Value type is `[FQN]: #ComponentTransformer` at this layer; 005 widens to `#ComponentTransformer \| #ModuleTransformer`. Capability fulfilment for Resources/Traits is registered via the transformer's `requiredResources` / `requiredTraits` fields (D7). |
| `#matchers.resources` | `[FQN=string]: [...#ComponentTransformer]` | computed | Reverse index: per-Resource-FQN, transformer candidates whose `requiredResources` includes that FQN. Multiple candidates per FQN are normal — the runtime matcher resolves via per-candidate predicate evaluation against the component (D13, revised). |
| `#matchers.traits` | `[FQN=string]: [...#ComponentTransformer]` | computed | Same shape, keyed off `requiredTraits`. Multi-candidate resolution as above. |

> **Note — `#ctx` deferred.** Enhancement 004 introduces `#Platform.#ctx: #PlatformContext`. The field is not yet on the landed `#Platform` schema in `apis/core/v1alpha2/platform.cue`; it will be added when 004 lands.

### `#ModuleRegistration`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `#module` | `#Module` | yes | Registered Module value (CUE-imported or runtime-injected) |
| `enabled` | `bool` | default `true` | When false, hides every projection of this entry from the platform (`#knownResources`, `#knownTraits`, `#composedTransformers`, `#matchers` — and `#knownClaims` once 005 lands) — D14 |
| `presentation` | `{description?, category?, tags?, examples?}` | no | Self-service catalog curation metadata. Flat shape after D11/D14. |
| `metadata.labels` | `#LabelsAnnotationsType` | no | Registration labels |
| `metadata.annotations` | `#LabelsAnnotationsType` | no | Registration annotations |

### `#PlatformMatch`

> **Design-only — not in v1alpha2 CUE schema.** The walker is implemented in Go (`pkg/compile/`, `pkg/platform/`). Table below describes the original CUE design.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `platform` | `#Platform` | yes | Platform whose `#matchers` index is consulted |
| `module` | `#Module` | yes | Consumer Module being deployed |
| `matched.resources` | `[FQN=string]: [...#ComponentTransformer]` | computed | Per-FQN candidate list for Resource demand |
| `matched.traits` | `[FQN=string]: [...#ComponentTransformer]` | computed | Per-FQN candidate list for Trait demand |
| `unmatched.{resources,traits}` | `[...string]` | computed | FQNs the consumer demands with zero fulfillers (D8). 005 adds `claims`. |
| `ambiguous.{resources,traits}` | `[FQN=string]: [...transformer]` | computed | FQNs with > 1 fulfiller. Under D13 (revised) multi-candidate is normal — the Go runtime resolves via predicate evaluation; the field is retained in the design as a diagnostic surface. 005 adds `claims`. |

### `#ComponentTransformer`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `apiVersion` | `"opmodel.dev/core/v1alpha2"` | fixed | OPM core |
| `kind` | `"ComponentTransformer"` | fixed | Always `"ComponentTransformer"` |
| `metadata.modulePath` | `#ModulePathType` | yes | Module path (e.g. `"opmodel.dev/opm/v1alpha2/providers/kubernetes"`) |
| `metadata.version` | `#MajorVersionType` | yes | Major version |
| `metadata.name` | `#NameType` | yes | Transformer name (kebab-case) |
| `metadata.fqn` | `#FQNType` | computed | `\(modulePath)/\(name)@\(version)` — used as `#defines.transformers` map key |
| `metadata.description` | `string` | yes | Human-readable summary |
| `requiredLabels` / `optionalLabels` | `#LabelsAnnotationsType` | no | Component label match keys |
| `requiredResources` / `optionalResources` | `[#FQNType]: #Resource` | no | Component `#resources` FQN match keys. Map values reference concrete `#Resource` definitions. |
| `requiredTraits` / `optionalTraits` | `[#FQNType]: #Trait` | no | Component `#traits` FQN match keys. Map values reference concrete `#Trait` definitions. 005 adds `requiredClaims` / `optionalClaims`. |
| `readsContext` | `[...string]` | no | Declarative `#ctx` paths the body reads (catalog-UI hint) |
| `producesKinds` | `[...string]` | no | Output Kubernetes kinds (catalog-UI hint) |
| `#transform` | `{ #moduleRelease, #component, #context, output }` | yes | Render function. Runtime supplies all three inputs concretely (D18). |

## File Locations

### New files

| Path | Purpose |
|------|---------|
| `apis/core/v1alpha2/platform.cue` | `#Platform`, `#ModuleRegistration`, `#PlatformMatch` |
| `apis/core/v1alpha2/transformer.cue` | `#ComponentTransformer`, `#TransformerMap`. 005 extends with `#ModuleTransformer` and widens `#TransformerMap` to the union. |
| `experiments/002-platform-construct/` | Self-contained CUE harness exercising every schema in this doc — mirrors `experiments/001-module-context/`. Validates D3 (FQN collision), D14 (enabled hides), D15 (concurrent-write conflict), D16 (kebab Id), the `#PlatformMatch.unmatched` walker (design-only), and basic `#known*` / `#composedTransformers` / `#matchers` projections. (D13 revised: multi-fulfiller allowed; no CUE-side guard to validate.) |

### Removed / superseded

| Path | Status |
|------|--------|
| `apis/core/v1alpha2/provider.cue` | `#Provider` retired in this enhancement (D4 superseded). Matcher migrates to consume `#composedTransformers` + `#matchers` directly. The file is deleted; any remaining imports are repointed at the new constructs. |

### Defined elsewhere (referenced from this enhancement)

| Path | Purpose | Owning enhancement |
|------|---------|--------------------|
| `apis/core/v1alpha2/context.cue` | `#PlatformContext`, `#EnvironmentContext`, `#ModuleContext`, `#RuntimeContext`, `#ComponentNames` | 004 |
| `apis/core/v1alpha2/environment.cue` | `#Environment` | 004 |
| `apis/core/v1alpha2/context_builder.cue` | `#ContextBuilder` | 004 |
| `apis/core/v1alpha2/module.cue` | `#Module` flat shape (eight slots) | 005 |
| `apis/core/v1alpha2/claim.cue` | `#Claim` primitive, `#ClaimMap` | 005 |
| `apis/core/v1alpha2/transformer.cue` | `#ModuleTransformer` (extends this enhancement's transformer file) | 005 |

> Flat-package note: 003 / 005 / 004 all live in the single `v1alpha2` package, so `#Platform`'s schema references `#ComponentTransformer` and `#TransformerMap` (and 005's `#ModuleTransformer`) directly without a cross-package import alias.
