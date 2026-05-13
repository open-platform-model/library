# Problem Statement — `#Module` Flat Shape with `#Claim` Primitive and `#defines` Channel

## Current State

`#Module` (currently `apis/core/v1alpha1/module/module.cue`; this enhancement targets the v1alpha2 rewrite at `apis/core/v1alpha2/module.cue`) currently exposes:

- `metadata` (identity, FQN, UUID, labels)
- `#components` (developer-defined components)
- `#policies` (legacy — slot is being removed in this enhancement)
- `#config` (the value schema / API contract)
- `debugValues` (concrete example values for testing)

`#Resource`, `#Trait`, `#Blueprint`, `#Op`, and `#Action` exist as primitives. `#Component`, `#Lifecycle`, `#Workflow`, `#Status`, `#Bundle`, `#Transformer` exist as constructs. `#Policy`, `#PolicyRule`, `#Directive`, `#Provider`, `#Test`, `#Config`, `#Template`, and `#StatusProbe` are deferred — see the policy redesign (enhancement 012) for the policy/directive line, and the platform construct (enhancement 003) for the Provider replacement via `#Module.#defines`.

`#Module` doubles as both an **Application** (`#components` render to deployable resources via providers) and an **API description** (`#config` is the parameter schema; components describe what gets generated).

There is no primitive today that expresses an ecosystem-extensible "I need X" pairing. An earlier sketch introduced `#Claim` along these lines but was never landed.

## Gap / Pain

### Module field-bloat risk

OPM's vision requires several more primitives at module level: needs (claims), lifecycles for operators, workflows for on-demand operations, and a publication channel for type definitions a Module ships to the ecosystem. Adding each as its own top-level field on `#Module` produces an unboundedly growing struct. Each new field forces a `#Module` schema change and adds cognitive load for module authors who must learn the full vocabulary even to write a small module.

### Resource and Claim litmus overlap

`docs/core/definition-types.md` lists both `#Resource` and a future `#Claim` as answering "what must exist?" That phrasing does not differentiate them. Without a sharper litmus, authors cannot tell whether to model a need as a `#Resource` (catalog-fixed, transformer-rendered) or a `#Claim` (ecosystem-extended, provider-fulfilled).

### No clean ecosystem extension surface

The vision distinguishes commodity services (well-known, agreed-upon platform APIs — Managed Database, Container, Volume, Backup) from specialty services (vendor innovation surface — VectorIndex, EventBus, custom platform APIs). Today, both would have to ship as `#Resource` in the catalog or as ad-hoc CRDs. Neither path lets a vendor publish a typed contract that other modules can request and that the platform can route — without a catalog PR for every new type.

The vision also requires that the platform's `#Platform` construct (enhancement 003) discover what types are available across registered Modules. Today there is no schema-level place where a Module declares "I publish these Resource / Trait / Claim type definitions to the ecosystem", forcing the platform to source-scan to enumerate available primitives. (`#Blueprint` is intentionally not published through `#defines` — see DEF-D6: Blueprints are CUE-import sugar with no platform-level consumer.)

### App/API duality lost on adornment growth

`#Module`'s App-vs-API duality already works elegantly via `#config` (App = parameterized via values; API = `#config` is the published schema). However, every new top-level adornment (Policy, Claim, Action, Lifecycle, …) blurs which fields apply in which mode and forces all consumers — App authors, Operator authors, type publishers — to read past fields they do not use.

## Concrete Example

A developer wants to ship a stateless web app that needs a Postgres database. A vendor wants to ship a Postgres operator that fulfils the well-known `ManagedDatabase` capability. A platform team wants to ship the OPM core catalog so the platform can discover all the standard `#Resource` and `#Trait` definitions without source-scanning. (Blueprint definitions ride along in their CUE packages and are imported directly by consumers; they are not platform-aggregated — DEF-D6.)

Today, all three would use the same `#Module` shape but with no clean fit:

- The web app uses `#components` but has nowhere to express "I need a Postgres database" as an ecosystem-extensible request.
- The operator uses `#components` (controller + CRDs) but has no field to register the rendering rule that maps a `ManagedDatabaseClaim` request to a Postgres CRD instance.
- The OPM core catalog has no `#components` and no `#config`. It is a publication-only Module — but `#Module` has no slot for that role.

The `ManagedDatabase` contract itself has no canonical home — neither catalog `#Resource` nor CRD captures the typed contract that a vendor's Module fulfils. Adding `#claims`, `#lifecycles`, `#workflows`, `#actions` and a publication slot to `#Module` naïvely produces a ten-plus-field type, half of which apply to one role only.

## Why Existing Workarounds Fail

- **Modeling claims as Resources.** Resources are catalog-fixed: adding `ManagedDatabase` requires a catalog PR plus a Transformer per provider. Vendors cannot self-publish.
- **Modeling provided APIs as CRDs only.** CRDs cover the k8s registration but not the OPM-platform-level contract: the self-service catalog has no entry, deploy-time matching has nothing to match against, and the schema is duplicated between the operator's CRD and any module that wants typed parameters.
- **Treating `#policies` as a catch-all bag.** Even before this enhancement removes `#policies`, the slot was the wrong home for needs and published types: it was scoped to governance and operational orchestration. The policy redesign (enhancement 012) will reintroduce a future home for those concerns, but in a deliberately shaped construct rather than a catch-all bag on `#Module`.
- **Adding kind discrimination to `#Module` (`#AppModule`, `#APIModule`, `#OperatorModule`).** Loses the deliberate flexibility of one type covering app + API + operator simultaneously. Also fragments tooling: every CLI command must know which kinds it handles.
- **Source-scanning to discover published primitives.** Works mechanically but breaks Constitution VIII (self-describing distribution): the CUE evaluation graph stops being the dependency graph as soon as discovery requires walking files outside the import chain.
