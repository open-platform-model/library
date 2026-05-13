# Design — `#Platform` Construct

## Design Goals

- `#Platform` is a single CUE construct that models a deployment target's identity, context, and registered extensions.
- The composition unit is `#Module`. There is no `#providers` field on `#Platform`.
- The platform's extension surface is a single map — `#registry` — typed to accept `#Module` values via `#ModuleRegistration` entries. Id keys are kebab-case (`#NameType`, D16).
- `#registry` is fillable by the platform CUE file (static) and by the runtime (dynamic) using the same schema field. CUE unification merges both sources; concrete-value disagreement at the same Id produces `_|_` and is surfaced by the `opm-operator` reconciler in `ModuleRelease.status.conditions` (D15).
- `#ModuleRegistration.enabled: false` hides every projection of an entry — types and transformers alike (D14). Use case: stage a runtime-injected entry without activating it.
- All outward platform views at this layer (`#knownResources`, `#knownTraits`, `#composedTransformers`, `#matchers`) are computed projections over `#registry`. No view duplicates state. 005 adds `#knownClaims` once `#defines.claims` exists.
- Matcher logic lives on `#Platform` directly. `#matchers` is a per-FQN reverse index over `#composedTransformers`; `#PlatformMatch` is a per-deploy walker that resolves a consumer Module's FQN demand against the index. `#Provider` is retired (D12) — the matcher consumes `#composedTransformers` + `#matchers` directly.
- One transformer primitive at this layer: `#ComponentTransformer` (D17). Fires once per matching `#Component`. 005 introduces a sibling `#ModuleTransformer` for per-module fan-out (Claim-bound). Runtime always passes a fully concrete `#ModuleRelease` to `#transform` (D18).
- Multi-fulfiller is allowed (D13 revised). Two registered transformers whose `requiredResources` / `requiredTraits` overlap on the same FQN are listed as parallel candidates in `#matchers.{resources,traits}`. The Go runtime matcher resolves per-component via predicate evaluation (label match, optional-trait satisfaction). The CUE schema carries no `_invalid` / `_noMultiFulfiller` guard. 005 extends the same model to Claims.
- Module installation is a single operator-driven step: a `ModuleRelease` CR triggers `opm-operator` to install `#components` *and* FillPath the Module value into `#registry`. `#ModuleRegistration` carries no install metadata (D11); registration is a pure projection of `#defines`.
- FQN collisions across registered Modules surface as CUE unification errors at platform-evaluation time for *types* (D3). Fulfilment overlap (two transformers requiring the same FQN) is allowed and resolved by the Go runtime matcher (D13 revised).

## Non-Goals

- `#Environment` construct, `#ContextBuilder`, and `#ModuleRelease` integration — defined in 004, used unchanged.
- `#PlatformContext` / `#ctx` schema — defined in 004, referenced unchanged.
- Runtime-fill mechanism for `#registry` — schema declared here; mechanism (Strategy B–style Go injection) deferred to a follow-up enhancement.
- Self-service catalog runtime API surface (`opm catalog list`, web UI, deploy-time match resolver) — declarative shape only; runtime is the platform's choice (consistent with 005 design).
- `#PolicyTransformer` registration — deferred until policy redesign converges (`enhancements/012`).
- Migration of `opmodel.dev/opm/v1alpha2/providers/kubernetes` and other provider packages into `#Module` form — separate enhancement.
- `#Claim` primitive, `#ModuleTransformer`, status writeback (`#resolution`), `#defines.claims`, `#knownClaims`, `#matchers.claims`, and the Claim halves of `_demand` / `matched` / `unmatched` / `ambiguous` on `#PlatformMatch` — all introduced as extensions in [005](../005-claims/).
- Multi-fulfiller resolution policy beyond predicate evaluation. Today: D13 (revised) allows multi-fulfiller; the Go runtime matcher disambiguates per consumer component via predicate evaluation. Edge cases where predicates do not disambiguate (two transformers with identical predicates fulfil the same FQN for the same component) are deferred to a future enhancement.
- Policy for unmatched FQNs (Resource / Trait types used by a deployed module with no renderer registered, plus 005's Claim extension). Detection is deterministic (D8); the response — fail / warn / drop — is a platform-team policy concern deferred until the catalog `#Policy` redesign (012) converges.

## High-Level Approach

```text
#Platform
├── metadata        identity
├── type            target type ("kubernetes" | ...)
├── #ctx            (from 004 — #PlatformContext)
└── #registry       single dynamic ingress: [Id]: #ModuleRegistration
                     │
                     ▼ projection (CUE comprehensions)
                    #knownResources
                    #knownTraits
                    #composedTransformers      (003: [FQN]: #ComponentTransformer; 005 widens)
                    #matchers       (D12 — reverse index per demand FQN; resources + traits)

#PlatformMatch       per-deploy walker (D12)
├── platform        the #Platform whose #matchers is consulted
├── module          the consumer Module being deployed
└── matched / unmatched / ambiguous   render plan + diagnostics (resources + traits)

005 extends:
├── #knownClaims (from #defines.claims)
├── #matchers.claims (union of #ComponentTransformer | #ModuleTransformer fulfillers)
└── #PlatformMatch.{matched,unmatched,ambiguous}.claims
```

The platform CUE file lists statically-known Module registrations (e.g. OPM core, vendor operators the admin has installed). The runtime fills additional entries via `FillPath` into `#registry`. The two sources unify by Id key.

Module installation flow (D11):

```text
ModuleRelease CR (with #defines)
   │
   ▼
opm-operator
   ├── installs #components against the cluster
   └── FillPath: #registry[id].#module = <Module>
                          │
                          ▼
                   #knownResources / #knownTraits
                   #composedTransformers / #matchers
                   recompute automatically
                   (005 adds #knownClaims and #matchers.claims to the same recompute)
```

Registration is a *consequence* of release. `#ModuleRegistration` carries no install metadata — the operator already has the full `#Module` value via the CR.

### Concurrent registration writes (D15)

A platform CUE file may declare `#registry["k8up"]` statically while `opm-operator` simultaneously `FillPath`s `#registry["k8up"]` from a `ModuleRelease` CR. CUE unification merges both writes by Id. Three outcomes:

1. **Identical values** — idempotent.
2. **Disjoint fields** (static sets `presentation`, runtime sets `#module`) — both populate.
3. **Concrete-value disagreement** (static `version: "1.0.0"` vs. runtime-injected `2.0.0`) — `_|_` at platform-evaluation time.

The schema does **not** add an override hierarchy. The opm-operator reconciler catches the bottom value and writes a structured condition to the offending `ModuleRelease.status.conditions` (e.g. `type: "RegistryConflict", message: "#registry[\"k8up\"].#module.metadata.version conflict: declared 1.0.0, injected 2.0.0"`). Admins resolve by aligning the static declaration with the CR or removing one of the two writes. The kebab-case Id constraint (D16) is what keeps the merge unambiguous in the first place — case-mismatch duplicates cannot exist.

### Multi-fulfiller is allowed (D13 revised)

Two registered Modules whose `#defines.transformers` produce transformers with overlapping `requiredResources` / `requiredTraits` are listed as parallel candidates in `#matchers.{resources,traits}`. The Go runtime matcher resolves per consumer `#Component` via predicate evaluation — label match, optional-trait satisfaction, etc. The CUE schema therefore carries no `_invalid` / `_noMultiFulfiller` guard, and `apis/core/v1alpha2/platform.cue:30-35` documents this directly. 005 extends the same model to Claim overlap. Predicate-insufficient cases (two transformers fulfilling the same FQN and passing identical predicates against the same component) are a future-enhancement concern.

### Compatibility detection lives in the matcher

A consumer Module declares no platform requirement. At deploy time the matcher walks the module body for FQN usage — Resource and Trait FQNs from `#components[].#resources` / `#components[].#traits` — and looks each up in `#composedTransformers`. Unmatched FQNs are surfaced as a platform-level signal. There is no `#requires` field on `#Module` (see 005 MS-D5). 005 extends the matcher's body walk to module-level (`#claims`) and component-level (`#components[].#claims`) Claim FQNs.

What to do about an unmatched FQN — fail the deploy, warn and drop, criticality-based escalation — is a platform-team policy concern deferred until the catalog `#Policy` redesign (012) converges. Detection (this enhancement, D8) and policy (future) are independent. Non-Kubernetes runtimes lean on the same mechanism: a compose runtime registers its own transformer Module in `#registry`; resources whose FQN has no compose renderer surface as unmatched (see D9 and 004 D29).

A single registered `#Module` contributes everywhere at once:

| Module slot | Platform view | What it surfaces |
| --- | --- | --- |
| `#defines.resources`    | `#knownResources`         | Catalog of Resource types |
| `#defines.traits`       | `#knownTraits`            | Catalog of Trait types |
| `#defines.transformers` | `#composedTransformers`   | Active rendering registry. Capability fulfilment for Resources/Traits is registered by each `#ComponentTransformer`'s `requiredResources` / `requiredTraits` (D7). 005 widens this to Claims via `requiredClaims`. |
| `#components`           | (consumed at deploy)      | Operator workload to install |

005 adds `#defines.claims → #knownClaims` (Claim-type catalog) and `#claims` (module-level demand). `#components` and `#claims` are not aggregated at the platform level — they belong to the registered Module's own deployment, not to the platform's catalog surface.

## Schema / API Surface

See [03-schema.md](03-schema.md) for the full CUE definitions. The construct introduces:

- `#Platform` — replaces the earlier `#Platform` schema. Inherits `#ctx` typing from enhancement 004's `#PlatformContext`.
- `#ModuleRegistration` — the value type of `#registry` entries.
- `#PlatformMatch` — per-deploy match construct.
- `#ComponentTransformer` + `#TransformerMap` — sole transformer primitive at this layer (D17). Full design in [05-component-transformer-and-matcher.md](05-component-transformer-and-matcher.md).

Lives in: `apis/core/v1alpha2/platform.cue` (new file — `#Platform`, `#ModuleRegistration`, `#PlatformMatch`) and `apis/core/v1alpha2/transformer.cue` (new file — `#ComponentTransformer`, `#TransformerMap`). 005 extends the transformer file with `#ModuleTransformer` and widens `#TransformerMap`.

## Before / After

### Before

```cue
#Platform: core.#Platform & {
    metadata: name: "kind-opm-dev"
    type: "kubernetes"
    #providers: [opm.#Provider, k8up.#Provider, pgop.#Provider]
    #ctx: {
        runtime: cluster: domain: "cluster.local"
        platform: { ... }
    }
}
```

`#providers` is a list. Each entry is a `#Provider` value carrying only transformers. The Postgres operator's `#components` and `#defines` (under 005) have no platform-level home.

### After (003)

```cue
#Platform: core.#Platform & {
    metadata: name: "kind-opm-dev"
    type: "kubernetes"

    #ctx: {
        runtime: cluster: domain: "cluster.local"
        platform: { appDomain: "dev.local" }
    }

    #registry: {
        // OPM core — Module form, registered statically by every k8s platform.
        "opm-core": { #module: opmCore.#Module }

        // K8up — Module form, transformer-only contribution.
        "k8up": { #module: k8up.#Module }

        // Postgres operator — full surface (components + defines.transformers).
        "postgres": {
            #module: pgop.#Module
            presentation: operator: description: "Postgres operator"
        }

        // A golden-path template surfaced to users.
        "stateless-web": {
            #module: webtmpl.#Module
            presentation: template: {
                category: "web"
                examples: small: values: replicas: 2
            }
        }

        // Runtime-discovered modules land here at deploy time. The schema
        // is the same; only the source differs.
    }
}
```

After evaluation:

- `#composedTransformers` contains every transformer from every enabled registration's `#defines.transformers`. Capability fulfilment for Resources/Traits is implicit: any `#ComponentTransformer` whose `requiredResources` / `requiredTraits` includes a primitive FQN is the supply registration for that primitive (D7).
- `#matchers` is the per-FQN reverse index. The Go pipeline / `opm-operator` instantiates `#PlatformMatch` per deploy, gets back `matched` / `unmatched` / `ambiguous`, and dispatches transformers accordingly (D12). Algorithm details: see [05-component-transformer-and-matcher.md](05-component-transformer-and-matcher.md).
- 005 layers `#knownClaims`, the Claim halves of `#matchers` and `#PlatformMatch`, and the `#ComponentTransformer | #ModuleTransformer` widening over the same shape.

## File Layout

```text
apis/core/v1alpha2/
├── platform.cue        // #Platform, #ModuleRegistration, #PlatformMatch
└── transformer.cue     // #ComponentTransformer, #TransformerMap (005 extends with #ModuleTransformer)
```

The earlier `#Platform` definition is replaced wholesale. The context artefacts (`#PlatformContext`, `#EnvironmentContext`, `#Environment`, `#ContextBuilder`) live in enhancement 004 and are referenced from `#Platform.#ctx`. v1alpha1's `#Transformer` is replaced by `#ComponentTransformer` (D17).
