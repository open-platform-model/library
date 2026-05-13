# Litmus Updates — `#Module` Flat Shape with `#Claim` Primitive and `#defines` Channel

Updates required in `docs/core/definition-types.md`, `docs/core/primitives.md`, and `docs/core/constructs.md`.

## Sharpened Litmus Questions

The current table phrases both `#Resource` and (future) `#Claim` as answering "What must exist?" — that is the line that obscures their distinction. Replace with the sharpened questions below.

### Updated Summary Table (`docs/core/definition-types.md`)

| Type | Family | Question It Answers | Level |
|------|--------|---------------------|-------|
| **Resource** | Primitive | "What well-known thing must be rendered?" | Component |
| **Trait** | Primitive | "How does it behave?" | Component |
| **Blueprint** | Primitive | "What is the reusable pattern?" | Component |
| **Op** | Primitive | "What is the unit of work?" | Action |
| **Action** | Primitive | "What is the composed operation?" | Lifecycle/Workflow |
| **Claim** | Primitive | "What ecosystem-supplied thing must be fulfilled?" | Component / Module |
| **Component** | Construct | "What composes primitives?" | Module |
| **Module** | Construct | "What is the application, API, or operator?" | Top-level |
| **ModuleRelease** | Construct | "What is being deployed?" | Deployment |
| **Bundle** | Construct | "What modules are grouped?" | Top-level |
| **BundleRelease** | Construct | "What bundle is being deployed?" | Deployment |
| **Transformer** | Construct | "How are components rendered?" | Rendering |
| **Status** | Construct | "What is the computed state?" | Module |
| **Lifecycle** | Construct | "What happens on transitions?" | Component/Module |
| **Workflow** | Construct | "What runs on-demand?" | Module |

### Sharpened Module question

Current: "What is the application?"
New: **"What is the application, API, or operator?"**

Reflects the deliberate App / API / Operator triple-duty of `#Module`.

## Updated Decision Flowchart

`docs/core/definition-types.md` flowchart additions:

```text
1. Does it define a reusable `#spec` that gets composed?
    Yes → It's a Primitive. Continue:
        1. Is this a standalone deployable thing? → Resource
        2. Does this modify how a Resource operates? → Trait
        3. Is this a reusable composition of Resources/Traits? → Blueprint
        4. Is this an atomic unit of work? → Op
        5. Is this a composed operation built from Ops/Actions? → Action
        6. Is this an ecosystem-supplied need (catalog commodity or vendor specialty)? → Claim   [new]
    No → It's a Construct. See constructs.md.
```

## Sharpening Rationale

### Why "What well-known thing must be rendered?" for `#Resource`

`#Resource` types are catalog-fixed. The catalog ships a Transformer for each Resource type that renders it to provider-native output (k8s YAML for the Kubernetes provider). Adding a new Resource type requires a catalog PR plus a Transformer. The author of a `#Resource` is **describing a thing in known catalog vocabulary** so a known Transformer can render it.

### Why "What ecosystem-supplied thing must be fulfilled?" for `#Claim`

`#Claim` types are ecosystem-extended. Anyone — the catalog (commodities), or a vendor (specialty services) — can publish a `#Claim` definition in a CUE package. The platform fulfils via whatever transformer's `requiredClaims` declares the matching FQN. The author of a `#claim` is **declaring intent** that the ecosystem will satisfy at deploy time.

### How capability fulfilment is registered (no separate primitive)

A Module that fulfils a `#Claim` ships a transformer in `#defines.transformers` whose `requiredClaims` includes the Claim's FQN. The transformer is the registration. There is no `#Api` wrapper primitive — supply is expressed by the renderer that does the actual work.

## Mirror in `docs/core/primitives.md`

Add one new section:

### `#Claim`

> **Definition Type:** Primitive
>
> **Litmus:** "What ecosystem-supplied thing must be fulfilled?"
>
> **Distinguished from `#Resource`:** Resources are **catalog-fixed and transformer-rendered** (the catalog ships a Transformer per Resource type). Claims are **ecosystem-extended and provider-fulfilled** (any CUE package — catalog or vendor — may publish a `#Claim` definition; the platform routes to whichever transformer's `requiredClaims` includes a matching FQN).
>
> **Identity:** `apiVersion` + `metadata.fqn`. No string `type` field. Matching is structural at the CUE level and metadata-driven at deploy time.
>
> **Placement:** Component-level for data-plane needs; Module-level for platform-relationship needs. Type definitions may be published via `#Module.#defines.claims`.
>
> **Pattern:** Concrete Claims follow the catalog triplet pattern: `#X` (schema) + `#XDefaults` (defaults) + `#XClaim` (`#Claim` wrapper).
>
> **Fulfilment:** A Module that fulfils a Claim ships a transformer whose `requiredClaims` lists the Claim's FQN. The transformer is the supply registration; there is no separate primitive.
>
> **Resolution surface:** `#status` carries the fulfilment data written by the matching transformer at render time — module authors read `#claims.<id>.#status.<field>` to wire connection strings, secret refs, etc. Claims fulfilled purely by side-effect (e.g. backup orchestration) may leave `#status` empty.

## Mirror in `docs/core/constructs.md`

Update the `#Module` entry to reflect the flat shape:

> **Question:** "What is the application, API, or operator?"
>
> **Slots:** Nucleus (`metadata`, `#config`, `debugValues`, `#components`); inward (`#lifecycles`, `#workflows`); outward (`#claims`, `#defines`).
>
> **Triple-duty:** Same type covers Applications (components-led), API descriptions (`#config`-led or `#defines`-led), and Operators (components + lifecycles + transformer in `#defines.transformers`).

## Affected Cross-References

| File | Change |
|------|--------|
| `docs/core/definition-types.md` | Update summary table, decision flowchart, mermaid diagram (add Claim primitive) |
| `docs/core/primitives.md` | Add `#Claim` reference section; sharpen `#Resource` litmus |
| `docs/core/constructs.md` | Update `#Module` entry with the flat shape and triple-duty framing |
| `v1alpha2/INDEX.md` | Regenerate via `task generate:index` after primitive files land |
