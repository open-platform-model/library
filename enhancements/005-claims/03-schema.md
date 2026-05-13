# Schema — `#Module` Flat Shape with `#Claim` Primitive and `#defines` Channel

This file is the CUE-level reference: type definitions, field tables, file locations. Design rationale, examples, supersession history, and decisions live in the topical narrative files (see [`02-design.md`](02-design.md) for the index).

This enhancement extends 003's schema. See [003/03-schema.md](../003-platform-construct/03-schema.md) for `#Platform`, `#ModuleRegistration`, `#PlatformMatch`, and `#ComponentTransformer`. The 005 deltas to those constructs (Claim half of `#matchers` / `#PlatformMatch`, `#composedTransformers` union widening, `#ComponentTransformer` `requiredClaims` / `optionalClaims` extension) are noted inline below where the parent shape lives in 003.

## File Locations

```text
apis/core/v1alpha2/
├── module.cue          // #Module (modified — eight slots, #defines, #ctx, CL-D18 constraint)
├── component.cue       // #Component (modified — adds metadata.resourceName, #names, #claims)
├── claim.cue           // #Claim (new primitive — this enhancement)
└── transformer.cue     // 003 introduces #ComponentTransformer + #TransformerMap;
                        // this enhancement adds #ModuleTransformer, widens #TransformerMap,
                        // and extends #ComponentTransformer with requiredClaims / optionalClaims
```

Concrete commodity Claim definitions ship under `modules/opm/claims/`; vendor specialty Claims ship in their own packages. See [`06-claim-primitive.md`](06-claim-primitive.md) for the triplet / quartet pattern (and a worked `#ManagedDatabaseClaim` definition).

### Lifecycle / Workflow / Action — left as stubs

`#Module.#lifecycles` and `#Module.#workflows` reference `#Lifecycle` and `#Workflow` constructs. Neither is fully designed in this enhancement; they are declared as slots so the eight-slot shape (MS-D2) is complete and so operator-Module authors can populate them when the constructs land. For now, treat them as opaque `_` types in any implementation pass — the slots are there, the schemas are intentionally unspecified. `#Action` is consumed by `#Lifecycle` and `#Workflow` (MS-D3) and inherits the same stub status.

A follow-up enhancement (or the policy redesign in 012) will pin the construct shapes. Until then, modules that need lifecycle / workflow behaviour either inline `_` values into the slots or omit them.

### Pipeline-level concerns — see `12-pipeline-changes.md`

Decisions whose schema reserves a channel but whose algorithm lives in the Go renderer:

- `#resolution` — the writeback channel from `#ComponentTransformer.#transform` / `#ModuleTransformer.#transform` (CL-D16).
- Topological-sort ordering for `#status` writeback (CL-Q7 — multi-fulfiller half closed by 003 D13; cycle detection + missing fulfiller delegated).
- Deploy-time validation of `#Claim.#spec` / `#status` against registered Claim definitions (CL-Q3 — delegated).

See [`12-pipeline-changes.md`](12-pipeline-changes.md) for the contracts the pipeline must implement.

## `#Claim` (primitive)

```cue
package v1alpha2

// #Claim: Defines the shape of an ecosystem-supplied need.
// The same primitive serves as both the type definition (when authored in
// a catalog or vendor package) and the request (when used in a Module's
// #claims). Identity is the metadata FQN — there is no string type field.
//
// apiVersion is left open so concrete Claim definitions (e.g.
// #ManagedDatabaseClaim) set their own apiVersion. The base #Claim does
// not pin one.
#Claim: {
    apiVersion!: string                          // open — set by concrete claim
    kind:        "Claim"

    metadata: {
        modulePath!: #ModulePathType              // "opmodel.dev/opm/v1alpha2/claims/data"
        version!:    #MajorVersionType            // "v1"
        name!:       #NameType                    // "managed-database"
        #definitionName: (#KebabToPascal & {"in": name}).out

        fqn: #FQNType & "\(modulePath)/\(name)@\(version)"

        description?: string
        labels?:      #LabelsAnnotationsType
        annotations?: #LabelsAnnotationsType
    }

    // MUST be an OpenAPIv3 compatible schema.
    // The field name is the camelCase of metadata.name (kebab-case names
    // are converted: "managed-database" => "managedDatabase").
    #spec!: ((#KebabToCamel & {"in": metadata.name}).out): _

    // Open shape — concrete Claim definitions pin their own #status schema.
    // Populated by the fulfilling transformer at deploy time.
    // Module authors read via #claims.<id>.#status.<field>.
    #status?: _
}

#ClaimMap: [string]: _
```

`#Claim` lives in the flat `v1alpha2` package alongside `#Resource`, `#Trait`, `#Blueprint`. Helper types (`#ModulePathType`, `#MajorVersionType`, `#NameType`, `#FQNType`, `#KebabToPascal`, `#KebabToCamel`, `#LabelsAnnotationsType`) all live in the same package — no `t.` prefix needed.

The `#status?` channel and its writeback semantics are documented in [`06-claim-primitive.md`](06-claim-primitive.md) and [`07-claim-fulfilment.md`](07-claim-fulfilment.md).

## Two transformer primitives

[003 D17](../003-platform-construct/04-decisions.md) introduces `#ComponentTransformer` (per-component fan-out) — see [003/03-schema.md](../003-platform-construct/03-schema.md) and [003/05-component-transformer-and-matcher.md](../003-platform-construct/05-component-transformer-and-matcher.md). This enhancement adds:

1. **`#ModuleTransformer`** — per-module fan-out, with optional `requiresComponents` pre-fire gate for dual-scope renders.
2. **`#ComponentTransformer` widening** — `requiredClaims` and `optionalClaims` match keys for component-level Claim fulfilment.
3. **`#TransformerMap` widening** — value type becomes `#ComponentTransformer | #ModuleTransformer`.
4. **`#resolution`** — writeback channel on both `#transform` shapes.

Both kinds ship through `#Module.#defines.transformers`. CRD installation continues to live in `#components` via `#CRDsResource` — unchanged.

**Canonical schema for `#ModuleTransformer`, the widening details, status writeback channel, and worked examples live in [`07-claim-fulfilment.md`](07-claim-fulfilment.md).** This doc does not duplicate them.

## Updated `#Component`

`#Module.#components` references `#Component`. The current `core/v1alpha2/component.cue` (57 lines) covers `metadata`, `#resources`, `#traits`, `#blueprints`, and the `_allFields` / `spec` merging. This enhancement (in concert with 004) adds three slots:

- `metadata.resourceName?` — single point of override for the Kubernetes resource base name (004 D13).
- `#names: #ComponentNames` — runtime-injected per-component computed names; component bodies read `#names.dns.fqdn` etc. without retyping their map key (004 D32).
- `#claims?: [string]: #Claim` — per-component data-plane Claims (CL-D10). Per-component `#claims` instances are independent across components (CL-D17) — two components shipping `cache: ...` of the same Claim type each get their own fulfilment.

Proposed shape:

```cue
package v1alpha2

#LabelWorkloadType: "core.opmodel.dev/workload-type"

#Component: {
    apiVersion: "opmodel.dev/core/v1alpha2"
    kind:       "Component"

    metadata: {
        name!: #NameType

        // Override the Kubernetes resource base name for this component.
        // When absent, resourceName defaults to "{release}-{component}"
        // via #ComponentNames in #ctx (004 D13). All DNS variants in
        // #ctx.runtime.components and #names.dns.* cascade from this.
        resourceName?: #NameType

        labels?:      #LabelsAnnotationsType
        annotations?: #LabelsAnnotationsType
    }

    // Resources applied for this component (catalog-fixed primitives).
    #resources: #ResourceMap

    // Traits applied to this component.
    #traits?: #TraitMap

    // Blueprints applied to this component (CUE-import composition only;
    // not published through #defines per DEF-D6).
    #blueprints?: #BlueprintMap

    // Component-level Claims — per-component data-plane needs (CL-D10).
    // Independent fulfilment per instance per CL-D17. Two components
    // shipping the same Claim type get two #status values.
    #claims?: [string]: #Claim

    // Per-component computed names — injected by #ContextBuilder (004 D32).
    // Equal to #ctx.runtime.components[<this component's key>].
    // Components read #names.resourceName / #names.dns.* without retyping
    // their map key. Cross-component reads still go through
    // #ctx.runtime.components[<otherKey>].
    #names: #ComponentNames

    _allFields: {
        for _, resource in #resources {
            if resource.spec != _|_ {resource.spec}
        }
        if #traits != _|_ {
            for _, trait in #traits {
                if trait.spec != _|_ {trait.spec}
            }
        }
        if #blueprints != _|_ {
            for _, blueprint in #blueprints {
                if blueprint.spec != _|_ {blueprint.spec}
            }
        }
    }

    // Fields exposed by this component, merged from resources / traits /
    // blueprints. Closed — must be made concrete by the user.
    spec: close({
        _allFields
    })
}

#ComponentMap: [string]: #Component
```

Notes:

- `#claims` is keyed by author-chosen id (e.g. `db`, `cache`), not by FQN. The `metadata.fqn` on the embedded Claim value carries identity. The matcher resolves a transformer's `requiredClaims` (FQN-keyed) by walking the consumer's `#claims` map and matching on `claim.metadata.fqn`.
- `#claims` does **not** participate in `_allFields` / `spec` merging. Claims are demand declarations, not spec contributions.
- `#blueprints` stays internal to `#Component` composition; it does not flow up to a platform-level view (DEF-D6 / 003 D10 — paired drops of `#defines.blueprints` and `#knownBlueprints`).

## Updated `#Module`

```cue
package v1alpha2

import (
    cue_uuid "uuid"
)

// `#ComponentTransformer`, `#ModuleTransformer`, and `#TransformerMap`
// are siblings in the flat v1alpha2 package — no cross-package import
// needed. (Earlier draft used `transformer "opmodel.dev/core/v1alpha2:transformer"`
// which doesn't resolve in flat layout — see 003/03-schema.md for the
// matching fix.)
//
// #Module: The portable application/API/operator blueprint created by
// developers, vendors, or platform teams.
#Module: {
    apiVersion: "opmodel.dev/core/v1alpha2"
    kind:       "Module"

    metadata: {
        modulePath!: #ModulePathType
        name!:       #NameType
        version!:    #VersionType
        fqn:         #ModuleFQNType & "\(modulePath)/\(name):\(version)"
        uuid:        #UUIDType & cue_uuid.SHA1(OPMNamespace, fqn)
        #definitionName: (#KebabToPascal & {"in": name}).out

        defaultNamespace?: string
        description?:      string
        labels?:           #LabelsAnnotationsType
        annotations?:      #LabelsAnnotationsType

        labels: {
            "module.opmodel.dev/name":    "\(name)"
            "module.opmodel.dev/version": "\(version)"
            "module.opmodel.dev/uuid":    "\(uuid)"
        }
    }

    // Nucleus
    #config:     _
    debugValues: _

    // Runtime-injected counterpart to #config. Computed by #ContextBuilder
    // (enhancement 004) and unified into the module by #ModuleRelease before
    // components evaluate. Module authors read it inside #components but
    // never assign it directly. Schema: #ModuleContext.
    #ctx: #ModuleContext

    #components: [Id=string]: #Component & {
        metadata: {
            name: string | *Id
            labels: "component.opmodel.dev/name": name
        }
    }

    // Inward — operate on the module itself.
    // (#policies omitted from v1alpha2 entirely — MS-D4. Policy redesign in 012.)
    #lifecycles?: [Id=string]: #Lifecycle
    #workflows?:  [Id=string]: #Workflow

    // Outward — instances visible to the platform and other modules.
    // Author-keyed map; identity travels via the embedded Claim's
    // metadata.fqn. CL-D18 forbids two entries sharing the same FQN at
    // module level — schema constraint sketched below; final form to be
    // pinned by experiment / first implementation pass.
    #claims?: [Id=string]: #Claim

    // CL-D18: no duplicate module-level Claim FQN. The hidden
    // _moduleClaimFqnCounts map counts entries per FQN; the
    // _noDuplicateModuleClaimFqn constraint unifies each count with
    // concrete 1, producing _|_ when any FQN appears more than once.
    let _moduleClaimFqnCounts = {
        if #claims != _|_
        for _, c in #claims {
            (c.metadata.fqn): int
            (c.metadata.fqn): 1 + (*_moduleClaimFqnCounts[c.metadata.fqn] | 0)
        }
    }
    _noDuplicateModuleClaimFqn: {
        for fqn, n in _moduleClaimFqnCounts {
            (fqn): n & 1
        }
    }

    // Outward — publication channel.
    // Type definitions and rendering extensions this Module ships to the
    // ecosystem. Keyed by FQN. Map key MUST equal value.metadata.fqn —
    // CUE unification enforces this via the inline & {metadata: fqn: FQN}
    // constraint on each sub-map.
    #defines?: {
        resources?:    [FQN=string]: #Resource & {metadata: fqn: FQN}
        traits?:       [FQN=string]: #Trait & {metadata: fqn: FQN}
        claims?:       [FQN=string]: #Claim & {metadata: fqn: FQN}
        transformers?: [FQN=string]: (#ComponentTransformer | #ModuleTransformer) & {metadata: fqn: FQN}
    }
}

#ModuleMap: [string]: #Module
```

`v1alpha2`'s flat package layout means `#Component`, `#Resource`, `#Trait`, `#Blueprint`, `#Claim`, `#ComponentTransformer`, `#ModuleTransformer`, `#TransformerMap`, `#ModuleContext`, `#Lifecycle`, `#Workflow`, and the helper types are all unprefixed inside the package. No cross-package import alias is needed.

The eight-slot rationale, the supersession history (drop `#policies`, drop `#apis`), and the no-`#requires` decision live in [`04-module-shape.md`](04-module-shape.md). The `#defines` shape and FQN-binding rule live in [`05-defines-channel.md`](05-defines-channel.md).

## Field Documentation

### `#Claim`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `apiVersion` | `string` | yes | Open string set by concrete Claim definitions (e.g. `opmodel.dev/opm/v1alpha2`, `vendor.com/x/v1alpha2`) |
| `kind` | `"Claim"` | fixed | Always `"Claim"` |
| `metadata.modulePath` | `#ModulePathType` | yes | CUE module path the Claim definition lives in |
| `metadata.version` | `#MajorVersionType` | yes | Major version of the Claim definition |
| `metadata.name` | `#NameType` | yes | Kebab-case name of the Claim |
| `metadata.fqn` | `#FQNType` | computed | `\(modulePath)/\(name)@\(version)` — the deploy-time identity |
| `metadata.description` | `string` | no | Human-readable summary |
| `#spec` | `_` (camelCase field name from `metadata.name`) | yes | OpenAPIv3 schema for the request |
| `#status` | `_` (open; pinned by concrete Claims) | no | Resolution data written by the fulfilling transformer at deploy time. Open on the base; concrete Claims (e.g. `#ManagedDatabaseClaim`) pin a `#status` schema. Empty when fulfilment is side-effect only. |

### `#ComponentTransformer` — fields (v1alpha2)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `kind` | `"ComponentTransformer"` | fixed | Type identity — matcher dispatches per-component fan-out |
| `requiredLabels` | `#LabelsAnnotationsType` | no | Component MUST have these labels |
| `optionalLabels` | `#LabelsAnnotationsType` | no | Component MAY have these labels |
| `requiredResources` | `[FQN=string]: _` | no | Component MUST include these `#Resource` types |
| `optionalResources` | `[FQN=string]: _` | no | Component MAY include these |
| `requiredTraits` | `[FQN=string]: _` | no | Component MUST include these `#Trait` types |
| `optionalTraits` | `[FQN=string]: _` | no | Component MAY include these |
| `requiredClaims` | `[FQN=string]: _` | no | Component-level Claims the transformer fulfils |
| `optionalClaims` | `[FQN=string]: _` | no | Optional component-level Claim FQNs |
| `readsContext` | `[...string]` | no | Declarative list of `#ctx` paths the render reads |
| `producesKinds` | `[...string]` | no | Declarative list of output kinds |
| `#transform.#moduleRelease` | `_` | yes | Fully concrete `#ModuleRelease` — runtime always supplies this |
| `#transform.#component` | `_` | yes | The matched `#Component` (singular) |
| `#transform.#context` | `#TransformerContext` | yes | Inherited labels/annotations + runtime identity |
| `#transform.output` | `{...}` | yes | Provider-specific output |

### `#ModuleTransformer` — fields (v1alpha2)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `kind` | `"ModuleTransformer"` | fixed | Type identity — matcher dispatches per-module fan-out |
| `requiredLabels` | `#LabelsAnnotationsType` | no | Module MUST have these labels |
| `optionalLabels` | `#LabelsAnnotationsType` | no | Module MAY have these labels |
| `requiredClaims` | `[FQN=string]: _` | no | Module-level Claims the transformer fulfils |
| `optionalClaims` | `[FQN=string]: _` | no | Optional module-level Claim FQNs |
| `requiresComponents.resources` | `[FQN=string]: _` | no | Pre-fire gate: at least one component MUST carry these resources |
| `requiresComponents.traits` | `[FQN=string]: _` | no | Pre-fire gate: at least one component MUST carry these traits |
| `requiresComponents.claims` | `[FQN=string]: _` | no | Pre-fire gate: at least one component MUST carry these component-level claims |
| `readsContext` | `[...string]` | no | Declarative list of `#ctx` paths the render reads |
| `producesKinds` | `[...string]` | no | Declarative list of output kinds |
| `#transform.#moduleRelease` | `_` | yes | Fully concrete `#ModuleRelease` — body iterates `#components` itself when needed |
| `#transform.#context` | `#TransformerContext` | yes | Inherited labels/annotations + runtime identity |
| `#transform.output` | `{...}` | yes | Provider-specific output (typically a struct keyed by intent) |

### `#Module` — added/changed slots

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `#lifecycles` | `[Id=string]: #Lifecycle` | no | Inward — state transitions |
| `#workflows` | `[Id=string]: #Workflow` | no | Inward — on-demand operations |
| `#claims` | `[Id=string]: #Claim` | no | Outward — module-level needs (instances) |
| `#defines.resources` | `[FQN=string]: #Resource` | no | Outward — Resource type definitions published, FQN-keyed |
| `#defines.traits` | `[FQN=string]: #Trait` | no | Outward — Trait type definitions published, FQN-keyed |
| `#defines.claims` | `[FQN=string]: #Claim` | no | Outward — Claim type definitions published, FQN-keyed |
| `#defines.transformers` | `[FQN=string]: #ComponentTransformer \| #ModuleTransformer` | no | Outward — Transformer values published, FQN-keyed |

### `#defines` placement vs instance slots

`#Claim` is the primitive most affected by `#defines`. The same primitive serves three roles, distinguished by placement (see [`05-defines-channel.md`](05-defines-channel.md) for the full table and [`06-claim-primitive.md`](06-claim-primitive.md) for the consumer-side mechanics):

| Placement | Role |
|-----------|------|
| `#defines.claims["fqn"]: #Claim` | Module **publishes** this Claim type (catalog vocabulary) |
| `#claims.id: #Claim & {#spec: ...}` (component or module level) | Module **requests** an instance (demand) |
| `#defines.transformers["fqn"]: #ComponentTransformer & {requiredClaims …}` (component-level) or `#ModuleTransformer & {requiredClaims …}` (module-level) | Module **fulfils** this Claim type (supply, via the transformer's match keys) |

`#Resource` and `#Trait` types are publishable through `#defines`; instances of `#Resource` and `#Trait` continue to live inside `#Component` (component-internal). `#Blueprint` is **not** publishable through `#defines` (DEF-D6) — it is a CUE composition consumed by Components via direct package import, with no platform-level aggregation.

`#defines.transformers` is the only outward home for transformer values. `#PolicyTransformer` is excluded from this enhancement — see DEF-D5 in [`10-decisions.md`](10-decisions.md) and the policy redesign (012).
