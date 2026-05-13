# Design Decisions — `#Module` Flat Shape with `#Claim` Primitive and `#defines` Channel

## Summary

Decision log for all architectural and design choices made during this enhancement. Decisions are grouped by topic; numbering resets within each topic group. Within a group, decisions are listed chronologically. Append-only — do not remove or renumber existing entries. If a decision is reversed, add a new decision that supersedes it.

Each decision carries a `(was D#)` parenthetical pointing at the prior chronological numbering used during design conversations on 2026-04-28 / 2026-04-30 / 2026-05-01. Cross-document references in 003 and 004 point at the prefixed identifiers.

Topic groups:

- **MS** — Module shape (the eight-slot flat structure)
- **DEF** — `#defines` publication channel
- **CL** — `#Claim` primitive (identity, placement, resolution)
- **TR** — Transformer redesign (`#ComponentTransformer` + `#ModuleTransformer`)

Topical narrative files (`04-module-shape.md`, `05-defines-channel.md`, `06-claim-primitive.md`, `07-claim-fulfilment.md`) carry the design rationale and reference these decisions inline. This file is the audit-trail home.

---

## Module Shape (MS)

### MS-D1: Single `#Module` type, no kind discrimination (was D1)

**Decision:** `#Module` remains a single type that simultaneously covers Applications, API descriptions, and Operators. No `#AppModule` / `#APIModule` / `#OperatorModule` split.

**Alternatives considered:**

- Discriminated kinds (`#AppModule`, `#APIModule`, `#OperatorModule`) — clearer reading of intent at a glance, but locks the dual/triple-use at the type level and forces every CLI command and tool to know which kinds it handles.

**Rationale:** Vision goal is extreme flexibility. A Module that is both an Operator and an API Module (an operator that publishes a CRD-backed self-service API) is a legitimate, expected case. Splitting types fragments tooling and forces author choices that the type system should not require.

**Source:** User decision 2026-04-28 in design conversation.

---

### MS-D2: Flat `#Module` field structure (was D2)

**Decision:** `#Module` has a flat top-level field set: `metadata`, `#config`, `debugValues`, `#components`, `#lifecycles`, `#workflows`, `#claims`, `#defines`. No grouping (no `#aspects`, `#contract`, `#behavior`, `#ports`, etc.). The original 2026-04-28 wording listed nine fields including `#policies` and `#apis`; the field set has since been refined by MS-D4 (drop `#policies`) and CL-D14 (drop `#apis`), yielding the eight-field shape above.

**Alternatives considered:**

- One open `#aspects` map discriminated by `kind` — pinned `#Module` at four fields forever, but lost per-field schema enforcement and diluted the grouping.
- Two-bucket `#behavior` + `#ports` (or `#runtime` + `#surface`) — clean inward/outward split, but words felt muddy and added an unneeded layer of nesting.
- Nested grouping by category (governance / contracts / operations) — heavier authoring; new primitive forces a categorization decision.

**Rationale:** With `#Action` removed from top level (MS-D3) and the primitive list bounded at five adornments, the field count is stable. Each field name predicts what is inside. Grouping only earns its keep when the list is unbounded; it is not.

**Source:** User decision 2026-04-28 ("go flat").

---

### MS-D3: Remove `#Action` from `#Module` top level (was D3)

**Decision:** `#Action` is not a top-level slot on `#Module`. `#Action` is a primitive consumed by `#Lifecycle` and `#Workflow` constructs.

**Alternatives considered:**

- `#actions: [Id=string]: action.#Action` at module level — symmetric with other primitives, but redundant since Lifecycle and Workflow already compose Actions internally.

**Rationale:** `#Action` is a building block, not a module-level concern. Module authors do not write Actions in isolation; they write Lifecycles and Workflows, which compose Actions.

**Source:** User decision 2026-04-28.

---

### MS-D4: Remove `#policies` slot from `#Module`; defer `#Policy`, `#PolicyRule`, `#Directive` to enhancement 012 (was D23)

**Decision:** `#Module` no longer carries a `#policies` slot. The `policy` package import is removed from `module.cue`. `#Policy`, `#PolicyRule`, and `#Directive` are not referenced from this enhancement's Module schema. Any module-level governance / operational-orchestration concerns reattach via the policy redesign (enhancement 012) when that work converges — likely not as a `#Module.#policies` slot.

The catalog primitives `#PolicyRule` and `#Directive` and the construct `#Policy` continue to exist in their respective files (they are not deleted by this enhancement); they are simply no longer surfaced through `#Module`'s top-level shape and no longer appear in the litmus reference.

**Supersedes:** The original wording of MS-D2 ("nine flat top-level fields") which listed `#policies` as one of nine — the count is now eight after MS-D4 drops `#policies` and CL-D14 drops `#apis`. MS-D2 is restated in light of the supersession history.

Other affected primitives/constructs in the litmus reference are also dropped here for consistency: `#PolicyRule`, `#Directive`, `#StatusProbe` (primitives) and `#Policy`, `#Test`, `#Config`, `#Template`, `#Provider` (constructs) are removed from `09-litmus-updates.md`. `#Provider` is fully retired by 003 D12 — the matcher now consumes `#composedTransformers` + `#matchers` directly; no synthetic shim survives. The other entries are deferred until they are real and demonstrably useful; until then the litmus reference advertises only what exists and works.

**Alternatives considered:**

- Keep `#policies` as a forwarding slot to whatever 012 produces. Rejected: pre-commits this enhancement's Module schema to a shape that 012 may revise. Leaving the slot empty is worse than removing it cleanly and reattaching via 012.
- Remove `#PolicyRule`, `#Directive`, `#Policy` source files now. Rejected: out of scope. This enhancement only changes `#Module` and the litmus reference. The primitive/construct files are touched (or removed) by 012.

**Rationale:** "Keep only what really exists and works." Slots and litmus rows for primitives/constructs that are not yet load-bearing add cognitive overhead without paying their way. Removing them sharpens the public surface to the eight Module slots and the smaller primitive/construct set that this enhancement (with 003) actually depends on. When the deferred items mature, they reattach via their own enhancements with shapes informed by real use.

**Source:** User decision 2026-04-30 ("Lets remove: PolicyRule, Directive, Policy, Test, Config, Template, Provider, StatusProbe" and follow-up confirming `#policies` removal from `#Module`).

---

### MS-D5: No `#requires` slot on `#Module`; platform compatibility detection lives in the matcher (was D33)

**Decision:** `#Module` does not gain a `#requires` slot (no `platformType`, `resourceTypes`, or other declarative compatibility hints). Module authors do not declare what platforms or resource types they need. Platform compatibility is detected mechanically by the matcher.

Mechanism: at deploy time the matcher walks the module's body for FQN usage —

- Resource / Trait FQNs from `#components[].#resources` and `#components[].#traits`.
- Claim FQNs from `#claims` (module-level) and `#components[].#claims` (component-level).

— and looks each up in the platform's `#composedTransformers`. Unmatched FQNs are surfaced. The module slot count stays at 8.

**Alternatives considered:**

- **`#Module.#requires.platformType: "kubernetes" | "compose" | …`.** Rejected: predicts every target the author cares about; module authors cannot anticipate platforms that don't yet exist. The matcher already knows what the platform supports.
- **`#Module.#requires.resourceTypes: [...FQN]`.** Rejected: duplicates information already present in `#components[].#resources`. The matcher derives the same set automatically.
- **Resource / Trait `criticality` hints (e.g. `must` / `should` / `nice`) shipped on the type definition.** Considered but deferred — pairs naturally with platform-level matching policy (`unmatchedResource: fail|warn|drop`), which itself is deferred until the catalog `#Policy` redesign (012) converges.

**Rationale:** The matcher walks an FQN graph that's already in the module body. Adding a parallel `#requires` declaration creates two sources of truth (module says "I need X"; module body says "I use X") that can drift. Trusting the matcher keeps the module schema minimal and aligns with 003 D7's "transformer is the registration" principle — capability supply is registered by transformer presence; capability demand is read from module body, not declared.

What to do about unmatched FQNs (deploy fails / warns / drops the unrenderable resource) is a *platform-team policy*, not a *module-author concern*. That policy lives at the platform level and is deferred here (cross-ref 012 policy redesign).

**Cross-references:** 003 D8 (matcher detection mechanism); 004 D29 (k8s vocabulary canonical, non-k8s runtimes derive — relies on the same matcher mechanism for unmatched detection); CL-D4 (Claim FQN identity used by the matcher).

**Source:** User decision 2026-05-01 ("I don't want platformType or resourceTypes").

---

## `#defines` Channel (DEF)

### DEF-D1: `#defines` slot for type publication (003 introduces three sub-maps; 005 extends with `claims`) (was D17)

**Decision:** `#Module` gains a `#defines` slot, structured as a map of sub-maps keyed by primitive kind. **003 introduces** the slot with three sub-maps the matcher consumes: `resources`, `traits`, `transformers`. **005 extends** the slot with a fourth sub-map: `claims` (the Claim-type catalog). Each sub-map is keyed by FQN. `#defines` is the platform-facing publication channel for type definitions and rendering extensions a Module ships to the ecosystem.

(Original wording introduced all four sub-maps in 005 and listed `blueprints` as a fifth; the blueprints sub-map was subsequently removed by **DEF-D6**, and the `resources` / `traits` / `transformers` introduction moved to 003 as part of the 003 / 005 untangle on 2026-05-02. The Claims half stays here in 005 — `#Claim` is a 005 primitive and the publication channel for it has no consumer until 005's `#knownClaims` view exists.)

**Alternatives considered:**

- Five flat sibling slots (`#publishedResources`, `#publishedTraits`, `#publishedBlueprints`, `#publishedClaims`, `#transformers`). Faithful to MS-D2's flat rule but pushes the field count from nine to fifteen.
- Auto-derive from `#transformers` and `#apis` (no explicit publication slot). Misses publisher-only Modules (a vendor publishes a Trait that someone else's Module renders).
- Extension-by-import only (no slot). Status quo. Forces the platform to source-scan to enumerate available definitions.
- Introduce the entire `#defines` slot in 005 and have 003 cross-ref. Considered during the untangle: rejected because 003's matcher needs `#defines.transformers` (and the `#known*` views need `#defines.resources` / `#defines.traits`); it is cleaner for 003 to introduce the slots its matcher consumes and 005 to extend with claims than for 003 to dangle a forward-reference into 005.

**Rationale:** A bounded, grouped slot is the smallest delta that makes definitions discoverable at platform-evaluation time. The grouping is justified because the kind-set is bounded (matches the primitive list, not unbounded) and the slot is meta — about the Module-as-publisher, not about the Module's runtime shape. Consumer Modules continue to import CUE packages directly to reference definitions; `#defines` is the discovery surface, not the consumption surface. Carves out a controlled exception to MS-D2's flat rule rather than ballooning the top-level field count.

**Cross-references:** 003 introduces the slot mechanism + three sub-maps the matcher consumes; this enhancement adds the `claims` sub-map alongside `#Claim` itself.

**Source:** User decision 2026-04-30 ("#defines is fine ... I also want to move #transformers into #defines"). Split between 003 and 005 on 2026-05-02 as part of the untangle.

---

### DEF-D2: Map keys in `#defines` sub-maps are bound to `value.metadata.fqn` (was D18)

**Decision:** Each sub-map under `#defines` enforces `key == value.metadata.fqn` via inline CUE unification:

```cue
resources?: [FQN=string]: prim.#Resource & {metadata: fqn: FQN}
```

Mismatched key/value pairs fail CUE unification at definition time.

**Alternatives considered:**

- Convention only (no enforcement). Rejected: typos and mismatched copy-paste edits silently produce broken discovery indexes.
- Computed-key wrapper (e.g. `#defines.resources: list of values, key derived`). Rejected: violates the keyed-map contract used by the Platform's computed views (003); breaks CUE map-as-set unification.

**Rationale:** Single source of truth. The FQN that the Platform indexes by is the same FQN the value declares. CUE catches mismatches at definition time (Constitution I — Type Safety First). No runtime check needed.

**Source:** User decision 2026-04-30 ("enforce").

---

### DEF-D3: `#defines` absorbs transformer publication; no separate `#transformers` slot (was D19)

**Decision:** Transformers ship under `#defines.transformers`, alongside type definitions. There is no separate top-level `#transformers` slot on `#Module`.

**Alternatives considered:**

- Separate `#transformers` slot at module top level. Rejected: splits "what this Module ships to the ecosystem" across two locations. The Platform's aggregation logic would need two parallel walks; CLI catalog tools would need two lookup paths.
- Both `#defines.transformers` and a top-level `#transformers` for backward-compatibility. Rejected: introduces ambiguity ("which one wins?") and doubles the surface for no benefit.

**Rationale:** Transformers are a kind of definition the Module publishes to the ecosystem — the Platform consumes them through the same publication channel as types. Single channel = single aggregation path = single discovery view. The active-vs-passive distinction (transformers are consumed by the matcher; types are discovery-only) is preserved by the Platform's separate `#composedTransformers` view; the Module-side publication channel does not need to differentiate.

**Source:** User decision 2026-04-30 ("I also want to move #transformers into #defines").

---

### DEF-D4: Drop `directives` from initial `#defines` sub-kinds (was D20)

**Decision:** `#defines` initially exposes the sub-maps for primitives whose shape is settled: `resources`, `traits`, `claims`, `transformers`. `directives` is excluded.

(Original 2026-04-30 wording listed five sub-maps including `blueprints`; that sub-map was subsequently removed by **DEF-D6**. The directive-exclusion decision is unchanged.)

**Alternatives considered:**

- Include `directives` from day one. Rejected: `#Directive` exists in an earlier design, but enhancement 012 (open exploration) may revise the policy/directive model. Locking a Directive publication channel before 012 converges risks committing to a shape we are about to change.

**Rationale:** Keep this enhancement scoped to primitives whose shape is already settled. When the policy redesign converges, a follow-up enhancement adds `directives` (or its successor) to `#defines` in a single contained change. Same reasoning applies to Op/Action publication (enhancement 010, still draft).

**Source:** User decision 2026-04-30 ("Remove directive. keep 'resources, traits, blueprints, claims'").

---

### DEF-D5: `#PolicyTransformer` is excluded from `#defines.transformers` (was D21)

**Decision:** `#defines.transformers` is typed as the union `#ComponentTransformer | #ModuleTransformer` only. `#PolicyTransformer` is not registered through `#Module.#defines` until the policy redesign (enhancement 012) converges.

**Alternatives considered:**

- Type `#defines.transformers` to include `#PolicyTransformer` in the union now. Rejected: same risk as DEF-D4 — locks `#PolicyTransformer`'s shape before 012 settles.

**Rationale:** Keep transformer publication scoped to the component-scope and module-scope flavours whose shape is settled (TR-D5). Policy-scope transformers ride along with the policy redesign and land via a follow-up enhancement.

**Source:** User decision 2026-04-30 ("Remove #PolicyTransformer for now. i want to figure out a better way to handle policies").

---

### DEF-D6: `blueprints` removed from `#defines`; Blueprints have no platform-level publication or aggregation

**Decision:** `#defines` does **not** carry a `blueprints` sub-map. The `#Blueprint` primitive remains in the catalog (used by `#Component.#blueprints` for spec-field merging via `_allFields` in component.cue), but Module authors cannot publish Blueprints through `#defines`, and the platform exposes no `#knownBlueprints` view (paired drop in 003 D10). Blueprint definitions ship as plain CUE types in their declaring packages, imported directly by consumer Modules.

**Alternatives considered:**

- **Keep `#defines.blueprints` for symmetry with `resources` / `traits` / `claims`.** Rejected: symmetry alone. Blueprints have zero downstream consumer in the platform — no transformer's `requiredResources` / `requiredTraits` / `requiredClaims` references a Blueprint FQN, and the matcher never walks Blueprint FQNs. A publication channel with no aggregating consumer is dead schema.
- **Demote to a presentation-only field on `#ModuleRegistration` (`presentation.template` pointing at a Blueprint FQN).** Considered — Blueprints map naturally onto golden-path templates. Deferred: the self-service catalog runtime is itself deferred (003 OQ4); designing a presentation channel for Blueprints before the consuming tooling exists pre-empts that decision. Tooling can walk `#registry[*].#module` directly when needed.
- **Defer the cut.** Rejected: 003 and 005 are unimplemented. Removing the slot now is free; removing later is a breaking change.

**Rationale:** A Blueprint is a CUE composition of Resources + Traits + Claims with no runtime semantics. At deploy time the Blueprint has already expanded into the Component's `#resources` / `#traits` / `#claims` slots before the matcher runs — its contribution is fully captured by those slots. The Blueprint FQN never participates in transformer matching, capability fulfilment (DEF-D3 / TR-D5 / 003 D7), or unmatched-FQN detection (003 D8 / MS-D5).

The asymmetry with the other `#defines.*` sub-maps is correct, not a flaw: `resources` and `traits` are read by transformer match keys; `claims` is read by both the demand-side FQN walk and the supply-side `requiredClaims` registration (CL-D14 / TR-D5); `transformers` is the active rendering registry. Each earns its publication slot by having a downstream consumer. Blueprints earn nothing.

`#Component.#blueprints` (the spec-merging field on Component) is unaffected — it is internal to Component composition and does not flow up to a platform-level view. Future cleanup may simplify `#Component.#blueprints` from FQN-keyed map to a plain list (the keying ceremony was justified by platform-level aggregation that no longer exists), but that is out of scope for this enhancement.

**Cross-references:** 003 D10 (paired drop of `#knownBlueprints`); DEF-D3 (transformer publication via `#defines` — by contrast); TR-D5 (`requiredClaims` as the supply registration mechanism — does not reference Blueprint FQNs); 003 D7 (no separate `#apis` aggregation — same parsimony principle); 003 D8 / MS-D5 (matcher walks Resource / Trait / Claim FQNs only); core/v1alpha2/component.cue (Component retains `#blueprints` for spec merging).

**Source:** User decision 2026-05-01 ("I don't think we need to register the blueprints. A blueprint is just a composition of #Resources, #Traits, and #Claims" → Option E in 003 brainstorm).

---

## `#Claim` Primitive (CL)

### CL-D1: Replace `#Offer` with `#Api` (was D4)

**Decision:** Supply-side capability registration is named `#Api`. An earlier `#Offer` primitive is not adopted.

**Alternatives considered:**

- Keep `#Offer` (the earlier name) — symmetric with `#Claim` in language ("offer" mirrors "claim"), but less aligned with common platform vocabulary.

**Rationale:** `#Api` matches widely understood platform terminology ("API surface", "self-service API"). When a `#Module` deploys to an OPM platform, an `#Api` registers the capability — this becomes a CRD in a Kubernetes context and a self-service catalog entry in the OPM platform context.

**Source:** User decision 2026-04-28.

**Status:** SUPERSEDED by CL-D14 (Remove `#Api` primitive). Retained for the historical record of the prior shape.

---

### CL-D2: Keep both `#Resource` and `#Claim` (was D5)

**Decision:** `#Resource` and `#Claim` are distinct primitives with sharpened litmus questions. Neither is absorbed into the other.

**Alternatives considered:**

- Absorb `#Claim` into `#Resource` and allow Resource at module level — fewer primitives, but loses the runtime extensibility signal and the demand/supply asymmetry that makes `#Api` meaningful.

**Rationale:** Resources are catalog-fixed and transformer-rendered. Claims are ecosystem-extended and provider-fulfilled. The two answer different questions for different ecosystem layers. Absorbing them blurs the specialty-services innovation surface.

**Source:** User decision 2026-04-28.

---

### CL-D3: No separate `#ClaimType` primitive (was D6)

**Decision:** `#Claim` serves as both the type definition (in catalog or vendor packages) and the request (in `#claims`) via CUE unification. There is no `#ClaimType` primitive.

**Alternatives considered:**

- `#ClaimType` (defines schema) + `#Claim` (request) as separate primitives — gRPC-like split with sharper roles, but added a primitive without proportional benefit.
- Schema embedded in `#Api` (whoever publishes first owns the schema) — fragile.

**Rationale:** CUE's package system already provides type identity. Catalog-published `#Claim` definitions (e.g. `#ManagedDatabaseClaim`) are simultaneously the type and the request shape — instances unify with the definition.

**Source:** User decision 2026-04-28.

---

### CL-D4: `#Claim` carries `apiVersion` + path metadata for traceability (was D7)

**Decision:** `#Claim` includes `apiVersion` and `metadata.{modulePath, name, version, fqn}` for identity. There is no string `type` field. Matching is structural at the CUE level and metadata-driven at deploy time.

**Alternatives considered:**

- `type: string` field for runtime lookup — simpler, but loses CUE-level structural matching and creates a naming-collision risk.
- Identity solely via CUE references (no explicit metadata) — works at authoring time but breaks when claims serialize across module boundaries.

**Rationale:** Identity must travel beyond CUE references when a Module is serialized for deploy-time matching. The `apiVersion` + `fqn` pair is the carrier — same role as Kubernetes' `apiVersion` + `kind`, or Go's package path.

**Source:** User decision 2026-04-28.

---

### CL-D5: `#Api` is 1:1 with `#Claim` (was D8)

**Decision:** Each `#Api` embeds exactly one `#Claim` as its `schema` field. A Module that fulfills multiple capabilities ships multiple `#api` entries in the `#apis` map.

**Alternatives considered:**

- 1:N (`#Api` embeds a list of `#Claim`s via a `schemas` field) — more compact for multi-fulfillers, but breaks parallel structure with `#components`, `#claims`, and other map-typed slots.

**Rationale:** Map-as-set ergonomics in CUE favor 1:1. A Postgres operator that fulfills both `ManagedDatabase` and `BackupTarget` ships two clear, parallel `#api` entries — easier to read, easier to evolve, easier to remove one without touching the other.

**Source:** User decision 2026-04-28 (Q1: 1:1).

**Status:** SUPERSEDED by CL-D14 (Remove `#Api` primitive). Retained for the historical record of the prior shape.

---

### CL-D6: Triplet pattern for concrete Claim definitions (was D9)

**Decision:** Concrete `#Claim` definitions follow the existing catalog triplet pattern: `#X` (schema) + `#XDefaults` (defaults) + `#XClaim` (`#Claim` wrapper). When the Claim has resolution data (CL-D15), the triplet extends to a quartet with `#XStatus` (status shape).

**Alternatives considered:**

- Single-definition collapse — `#ManagedDatabase: claim.#Claim & {...}` — fewer definitions but inconsistent with how `#Container` / `#ContainerDefaults` / `#ContainerResource` are organized for Resources.

**Rationale:** Catalog consistency. Authors moving between Resources and Claims see the same pattern. Defaults stay separable so consumers can opt in.

**Source:** User decision 2026-04-28 (Q2: Yes).

---

### CL-D7: `apiVersion` is open on the `#Claim` base; concrete Claims set their own (was D10)

**Decision:** The `#Claim` primitive base type leaves `apiVersion` as `string!` (required, open). Concrete Claim definitions (e.g. `#ManagedDatabaseClaim`, `#VectorIndexClaim`) set their own `apiVersion`. The base does not pin one.

**Alternatives considered:**

- Pin `apiVersion: "opmodel.dev/core/v1alpha2"` on `#Claim` base — consistent with `#Resource` and `#Directive`, but forces vendor specialty Claims to live under OPM's apiVersion or break the constraint.

**Rationale:** `apiVersion` belongs to the catalog or vendor that published the Claim, not the OPM core primitive. A vendor's `#VectorIndexClaim` should carry `apiVersion: "vendor.com/vectordb/v1alpha2"` — open base allows this naturally. (This differs from `#Resource` and `#Directive` which today pin `apiVersion` to OPM core; revisit those for consistency in a follow-up.)

**Source:** User decision 2026-04-28 (Q3: Yes, mirror; allow apiVersion open).

---

### CL-D8: CRDs deploy via `#CRDsResource` in `#components`, not via `#Api` (was D11)

**Decision:** `#Api` carries no CRD installation logic. Operators continue to ship CRDs as `#CRDsResource` inside their `#components`.

**Alternatives considered:**

- Embed CRD schema in `#Api.schema` so `#Api` declaration triggers CRD installation — couples platform-level capability registration to k8s-specific CRD lifecycle.

**Rationale:** Existing `#CRDsResource` pattern works. `#Api` is a platform-level concept (capability + self-service catalog); CRDs are a Kubernetes-provider concern. Keeping them separate preserves layer boundaries.

**Source:** User decision 2026-04-28.

**Status:** Live (the operative half — CRDs ship via `#CRDsResource`). The framing clause referencing `#Api` is moot after CL-D14.

---

### CL-D9: `#Api` deploy-time semantics are platform's choice (was D12)

**Decision:** `#Api` is purely declarative. The platform may use it to populate a self-service catalog, a deploy-time match cache, both, or any equivalent mechanism. The primitive does not pin the runtime behavior.

**Alternatives considered:**

- Catalog-only ("`#Api` is documentation") — loses the deploy-time matching story.
- Match-cache-only ("`#Api` is wiring") — loses the self-service catalog story.
- Strict pinned spec — over-constrains platform implementations.

**Rationale:** `#Api` declares intent; the platform decides how to honor it. Different OPM platforms (Kubernetes, OPM-as-a-service, future bare-metal) may implement registration differently. The primitive is the contract; the runtime is open.

**Source:** User decision 2026-04-28 (Q5: Option 3 — platform doesn't care).

**Status:** SUPERSEDED by CL-D14 (Remove `#Api` primitive). Retained for the historical record of the prior shape.

---

### CL-D10: Both component-level and module-level `#Claim` placement (was D13)

**Decision:** `#Claim` may be placed at component level (data-plane needs — DB, queue, cache, secret) or at module level (platform-relationship needs — DNS, tenant admission, identity, observability backend, mesh membership). Same primitive, two scopes; placement determines semantics.

**Alternatives considered:**

- Component-only — forces module-as-unit needs onto an arbitrary "primary" component, coupling the claim to a component implementation choice.
- Module-only — forces every per-component data-plane need up to module level, decoupling claim from the component that actually uses it.
- Two distinct primitives (`#Claim` for component, `#ModuleClaim` for module) — adds vocabulary for what is structurally the same primitive.

**Rationale:** Two flavors of need genuinely exist: per-component data-plane and per-module platform-relationship. Both are "I need X from the ecosystem." Placement carries the scope information cleanly.

**Source:** User decision 2026-04-28 (Q4: Yes, both).

---

### CL-D11: `#Resource` stays component-level only (was D14)

**Decision:** `#Resource` is not allowed at module level. Shared resources should be modeled as their own `#Component`.

**Alternatives considered:**

- Allow Resource at module level for shared infra (e.g. shared ConfigMap) — convenient but blurs the component composition model.

**Rationale:** Components are the unit of composition. Shared infra is its own component, with other components depending on it. Allowing module-level Resources collapses that boundary.

**Source:** User decision 2026-04-28.

---

### CL-D12: Sharpened litmus for `#Resource` and new entries for `#Claim` / `#Api` (was D15)

**Decision:** `docs/core/definition-types.md` litmus is updated:

- `#Resource`: "What well-known thing must be rendered?" (sharpened from "What must exist?")
- `#Claim`: "What ecosystem-supplied thing must be fulfilled?" (new)
- `#Api`: "What capability does this Module register?" (new)
- `#Module`: "What is the application, API, or operator?" (sharpened from "What is the application?")

**Alternatives considered:**

- Leave Resource's litmus unchanged and add Claim with a similar phrasing — perpetuates the overlap.

**Rationale:** Without distinct litmus questions, authors cannot pick between `#Resource` and `#Claim` from the doc alone. The sharpened pair maps to the **catalog-fixed vs. ecosystem-extended** axis (CL-D2).

**Source:** Design discussion 2026-04-28.

**Status:** The `#Api` litmus row is moot after CL-D14; the rest stand.

---

### CL-D13: `#Api` primitive has its own apiVersion pinned to OPM core (was D16)

**Decision:** `#Api`'s base type pins `apiVersion: "opmodel.dev/core/v1alpha2"`. Unlike `#Claim` (CL-D7), `#Api` is not extended by vendors as a base type — vendors author `#Claim` definitions and reference them via `#Api.schema`. The `#Api` primitive itself remains an OPM core type.

**Alternatives considered:**

- Open `apiVersion` on `#Api` parallel to `#Claim` — but vendors do not redefine `#Api` as a primitive; they only embed Claims into it.

**Rationale:** `#Api` is the wrapper. The variation comes from the embedded `schema`, not from `#Api` itself.

**Source:** Design discussion 2026-04-28.

**Status:** SUPERSEDED by CL-D14 (Remove `#Api` primitive). Retained for the historical record of the prior shape.

---

### CL-D14: Remove `#Api` primitive (was D22)

**Decision:** `#Api` is removed from this enhancement. There is no top-level `#apis` slot on `#Module` and no `api.cue` primitive file. Capability fulfilment is registered by transformers via their `requiredClaims` field — the transformer that does the rendering is the registration. An "API" in OPM is now expressed in one of two forms, both via `#defines`:

- A set of `#Resource` / `#Trait` definitions (consumed via component composition; rendered by transformers in the same Module or another).
- A `#Claim` definition (resolved at deploy time by any transformer whose `requiredClaims` includes the Claim's FQN).

CRDs continue to ship via `#CRDsResource` inside `#components` — unchanged.

**Supersedes:** CL-D1 (replace `#Offer` with `#Api`), CL-D5 (`#Api` is 1:1 with `#Claim`), the framing clause of CL-D8 (CRDs deploy via `#CRDsResource` not `#Api` — the operative half stands), CL-D9 (`#Api` deploy-time semantics are platform's choice), CL-D13 (`#Api` apiVersion pinned to OPM core). The superseded decisions remain as historical record; CL-D14 declares the new shape that replaces them.

**Alternatives considered:**

- Keep `#Api` as a self-service catalog metadata wrapper (description + examples) without the schema-embedding aspect. Rejected: leaves a wrapper primitive whose only purpose is documentation metadata; that metadata can live on `#Claim` itself or in a future self-service-catalog enhancement.
- Replace `#Api` with a thinner registration record on `#Module.metadata`. Rejected: capability fulfilment is structural information about the rendering pipeline, not metadata. The transformer's `requiredClaims` is the natural home.

**Rationale:** `#Api` was the supply-side wrapper paired with `#Claim` as the demand-side. With transformers gaining `requiredClaims` as a match key (TR-D5), the same information is carried by the transformer that does the rendering, with no wrapper layer. Removing `#Api` collapses three placements of `#Claim` (publish via `#defines.claims`, request via `#claims`, fulfil via `#apis`) into two (publish via `#defines.claims`, request via `#claims`), with fulfilment expressed by the transformer pipeline. Self-service catalog metadata (description, examples) that previously lived on `#Api.metadata` is dropped from this enhancement; if a self-service catalog UI later needs per-Claim example values, it becomes a separate enhancement that extends `#Claim.metadata`.

**Source:** User decision 2026-04-30 ("Now i want to remove #Api definition. It is not required anymore. OPM handles new apis differently, a new API can be a set of #Resoruces and #Traits or it can be a #Claim. All are registered in #defines as stated.").

---

### CL-D15: `#Claim` gains a `#status?` field (was D31)

**Decision:** The `#Claim` primitive grows an optional `#status?` field, parallel to `#spec!`. The base leaves it open; concrete Claim definitions (e.g. `#ManagedDatabaseClaim`, `#VectorIndexClaim`) pin a `#ManagedDatabaseStatus` / `#VectorIndexStatus` schema as the resolution shape. The split mirrors the Kubernetes `spec` / `status` convention: `#spec` is the demand the consumer Module makes; `#status` is the resolution the fulfilling transformer writes back.

Some Claims fulfil purely by side-effect (e.g. `#BackupClaim` — the transformer creates Schedule + Backend CRs but the consumer never reads connection data). Those Claims may omit the `#status` schema; an empty `#status` is a valid by-design outcome.

**Alternatives considered:**

- **Skip `#status`; route resolutions through `#ctx.platform.claims.<id>` open struct.** Rejected: claim instances are scoped to their consumer Module; storing per-instance resolution data on the platform-team-owned open struct (004 D28) blurs ownership and clashes with cross-module name collisions. Spec/status colocation on the claim instance is the cleaner shape.
- **Keep `#status` purely on operator-side CRs and force consumers to read those CRs directly.** Rejected: defeats the portability goal. A consumer Module that wires `DATABASE_HOST` from a Postgres CR's status field becomes Postgres-CR-bound and breaks when a different fulfiller (Aiven, RDS) is registered. `#status` is the abstraction layer that lets the consumer read fulfiller-agnostic data.

**Rationale:** `#status` is the cross-runtime portability surface. The same `#Module` deployed against different fulfilling transformers receives target-appropriate resolution data without changing the consumer's code path. A `#PublicEndpointClaim` resolved by a k8s Gateway transformer or by a compose Traefik-label transformer both write `#status.url`; the consumer reads the same field on both runtimes. This makes `#status` the bridge that 003 D9 (per-runtime transformer Modules) leans on for cross-runtime portability.

**Cross-references:** CL-D4 (Claim FQN identity); TR-D5 (two transformer primitives); TR-D6 (runtime guarantee of fully concrete `#ModuleRelease`); 004 D11 / D12 (centralised computed-field injection precedent); `07-claim-fulfilment.md` (writeback channel sketch).

**Source:** Design conversation 2026-05-01 (platform-model brainstorm).

---

### CL-D16: `#status` is written by the matching transformer; injection follows 004 D12 (Strategy B / Go pipeline) (was D32)

**Decision:** A transformer whose `requiredClaims` contains a Claim FQN is the writer of that Claim instance's `#status`. Specifically:

- `#ComponentTransformer.requiredClaims: <FQN>` writes `#status` on a component-level `#claims.<id>` instance whose `metadata.fqn` matches.
- `#ModuleTransformer.requiredClaims: <FQN>` writes `#status` on a module-level `#claims.<id>` instance whose `metadata.fqn` matches.

The CUE schema reserves `#status?` on `#Claim`. The Go render pipeline injects values via `FillPath` after the fulfilling transformer fires and before consumers (transformers / component bodies that read `#status`) re-evaluate. Same Strategy B precedent as 004 D12 for content hashes.

The matcher must topologically order fulfillers before consumers. `requiredClaims` declares both a match and a write edge; `#claims.<id>.#status.<field>` reads inside another transformer's body or a component's spec declare the read edge. The dependency graph is the topological-sort input. Edge cases (cycles, missing fulfiller, multi-fulfiller — see CL-Q7, TR-Q2) remain open.

**Alternatives considered:**

- **Strategy A (two-pass CUE evaluation).** Deferred for the same reason content hashes deferred it (004 D12 Alternatives): adds CUE-evaluation complexity without a current need. Strategy A becomes desirable only if a `#status` write must self-reference its own future value (no current case).
- **Status writeback as a separate, declared `#resolution` field on the transformer.** Schema sketch landed in `07-claim-fulfilment.md`; left at convention level for now. The exact field name and shape are an implementation concern, not a schema-decision concern.

**Rationale:** Status injection mirrors the most-similar already-decided mechanism (hashes). Re-using the precedent keeps the Go pipeline simple — one injection phase, two kinds of writes. The split lifecycle (CUE schema reserves; Go pipeline populates) preserves CUE-time testability for the read shapes while letting the populating order live in Go where the matcher already owns iteration.

**Cross-references:** 004 D11 / D12 (centralised hash injection); TR-D6 (concrete `#ModuleRelease` runtime guarantee); CL-D17 (each Claim instance fulfilled independently — single writer per instance is the natural reading); CL-Q3 (delegated to `12-pipeline-changes.md`); `07-claim-fulfilment.md` CL-Q7 (matcher writeback dispatch + ordering — delegated to `12-pipeline-changes.md`); 003 D13 (multi-fulfiller forbidden — guarantees a single writer per Claim instance, closes the multi-fulfiller sub-bullet of CL-Q7 and the entirety of TR-Q2).

**Source:** Design conversation 2026-05-01.

---

### CL-D17: Each `#Claim` instance is fulfilled independently — no merge / override / share-fulfilment semantics

**Decision:** Every `#Claim` instance — whether at module level (`#Module.#claims.<id>`) or component level (`#Module.#components.<X>.#claims.<id>`) — is fulfilled independently and carries its own `#status`. No instance "merges" or "overrides" another. Specifically:

1. Module-level `#claims.db` and component-level `#components.X.#claims.db` are two distinct instances ⇒ two `#status` values. Module-level is fulfilled by a `#ModuleTransformer.requiredClaims`; component-level by a `#ComponentTransformer.requiredClaims`. They do not share a fulfilment.
2. Two component-level `#claims.db` instances on different components (same Claim FQN, different ids) are two distinct instances ⇒ two `#status` values. They do not share a fulfilment either; each component's instance is fulfilled by its own transformer match.
3. The matcher does not deduplicate fulfilments across instances. If a Module wants a single shared resource (one DB, one queue), it models that as a single Claim at the appropriate scope — not as multiple instances expecting consolidation.

**Alternatives considered:**

- **Module-level as singleton; component-level instances all share the module-level fulfilment if FQNs match.** Rejected: forces an implicit cross-scope merge that the schema cannot validate; consumer Modules can't tell at authoring time whether their component-level Claim will be honored or overridden by an unrelated module-level Claim.
- **Component-level instances of the same FQN share one fulfilment per FQN per Module.** Rejected: collapses the per-component independence that makes Components the unit of composition (CL-D11 reasoning). Also makes per-component sizing / variant impossible.
- **Defer the resolution-order question to the Go matcher.** Rejected: the schema needs a stable contract before pipeline implementation; deferring means each pipeline rev can pick a different rule.

**Rationale:** Independent fulfilment is the natural reading of CL-D15 (each instance carries its own `#status`) and removes the entire merge / override / share-fulfilment design space at the matcher layer. It also matches how Resources behave (a Component owning its own `#resources` does not merge with another Component's `#resources` of the same type). Module authors who want a shared resource have a clean expression of that intent — declare it once at the appropriate scope. Matcher implementation collapses to "one fulfiller per instance."

**Cross-references:** CL-D10 (component-level + module-level placement); CL-D11 (Components as unit of composition — same parsimony); CL-D15 (`#status` per instance); CL-D16 (status writeback per matched instance); CL-D18 (no duplicate Claim FQN at module level — paired enforcement); 003 D13 (single fulfiller per FQN at platform level — together D17 + 003 D13 give "one fulfiller per FQN per instance").

**Source:** User decision 2026-05-02 ("Elevate" — accepting the lean direction stated in CL-Q1's CL-D15 update note).

---

### CL-D18: Duplicate module-level Claim FQN is forbidden at `#Module` schema time

**Decision:** `#Module.#claims` enforces FQN uniqueness across its entries. Two `#claims` entries with the same `metadata.fqn` produce `_|_` at module-evaluation time, before the matcher runs. The constraint is a CUE-level check on `#Module`:

```cue
#Module: {
    ...
    #claims?: [string]: #Claim

    // FQN uniqueness across module-level Claims (CL-D18). The hidden
    // _claimFqnCounts map counts how many entries share each FQN; the
    // _noDuplicateClaimFqn constraint unifies every count with concrete
    // 1 (or absent), producing _|_ when any FQN appears more than once.
    let _claimFqnCounts = {
        for _, c in #claims if #claims != _|_ {
            (c.metadata.fqn): int
            (c.metadata.fqn): 1 + (*_claimFqnCounts[c.metadata.fqn] | 0)
        }
    }
    _noDuplicateClaimFqn: {
        for fqn, n in _claimFqnCounts {
            (fqn): n & 1
        }
    }
    ...
}
```

(The exact CUE form is to be validated by the experiment / first implementation pass — the principle is "fail at module-eval time when two `#claims` entries share an FQN.")

The check applies to module-level `#claims` only. Component-level `#claims` (on `#Module.#components.<X>.#claims`) are per-component scopes and are not constrained across components — two components may legitimately each ship a `cache: ...` Claim entry of the same Claim type, and both fulfilments fire independently per CL-D17.

**Alternatives considered:**

- **Defer to runtime detection** — the matcher pseudocode in `07-claim-fulfilment.md` picks the first matching claim instance via comprehension lookup; if two entries share an FQN, the body silently sees only one. Rejected: silent shadowing is exactly the misconfiguration class CUE excels at catching at definition time. Punting to runtime hides the bug from the author.
- **Allow duplicates and treat as multi-instance fulfilment** — the matcher runs per-instance and emits two fulfilments. Rejected: at module level there is no per-instance discriminator (no parent Component as scope), so two same-FQN module-level Claims are indistinguishable to downstream consumers reading `#claims.<id>.#status`. Authors who want two distinct resources should give them distinct Claim definitions or push them to component level.

**Rationale:** Module-level is the singleton scope for "platform-relationship" Claims (DNS hostname, workload identity, mesh tenant, backup orchestration — see CL-D10's table). Each of those is conceptually a single resource per Module. Allowing duplicate FQNs would either silently drop one instance (matcher bug) or fan out fulfilment in ways consumers cannot disambiguate (`#claims.dns1.#status.fqdn` vs `#claims.dns2.#status.fqdn` — both bound to the same Claim type but with no module-level discriminator). Failing at schema time forces the author to model the intent properly.

**Cross-references:** CL-D10 (module-level vs component-level placement — singleton scope rationale); CL-D17 (independent fulfilments per instance — D18 prevents the degenerate case at module level); 003 D3 (FQN-collision-on-types fails CUE unification — same parsimony, applied to instances at module scope).

**Source:** User decision 2026-05-02 ("Elevate" — accepting the lean direction stated in CL-Q8).

---

## Transformer Redesign (TR)

### TR-D1: `#Transformer` match keys grouped into `componentMatch` and `moduleMatch` buckets (was D24)

**Decision:** The v1alpha2 `#Transformer` (apis/core/v1alpha2/transformer.cue) replaces the flat `requiredX` / `optionalX` fields with two scope buckets. `componentMatch` carries everything matched against a single `#Component` (`requiredLabels`, `requiredResources`, `requiredTraits`, `requiredClaims`, plus the `optional*` parallels). `moduleMatch` carries everything matched against `#Module` top level (`requiredLabels`, `requiredClaims`, plus `optional*`). `requiredDirectives` / `optionalDirectives` are dropped pending the policy redesign (012). The full design lives in `07-claim-fulfilment.md`.

**Alternatives considered:**

- **Keep flat fields, add `requiredClaims` only.** Rejected: a Claim FQN can legitimately be placed at component or module level (CL-D10). A flat `requiredClaims` makes the matcher guess scope, and a transformer that wants the *module* flavour cannot disambiguate from one that wants the *component* flavour. Example 7 (K8up backup, dual-scope) cannot be expressed unambiguously.
- **Add a `scope: "component" | "module"` discriminator + flat fields.** Rejected: cannot express dual-scope (Example 7's transformer needs both a module-level Claim and a component-level Trait). Would force a third `"module-with-components"` value, growing the discriminator past two axes.
- **Pure matcher inference (no schema change).** Rejected: CL-D10 lets the same Claim type appear at either level, so type alone never tells the matcher placement intent. The transformer must declare its scope.
- **Two primitives `#ComponentTransformer` / `#ModuleTransformer`.** Rejected: doubles the surface in `#defines.transformers` typing (DEF-D3) and forces every catalog tool to handle both kinds. User asked for one type.

**Rationale:** Authoring intent is explicit. Resources/Traits naturally appear only under `componentMatch` (CL-D11 enforced by schema, not convention). Same Claim FQN at both levels (CL-Q1's lean) is unambiguous: each transformer says which level it expects. Dual-scope transformers (Example 7) drop out as `moduleMatch` + `componentMatch` populated together.

**Source:** Design conversation 2026-05-01.

**Status:** SUPERSEDED by TR-D5 (two transformer primitives). Retained for the historical record of the prior shape.

---

### TR-D2: Single `#Transformer` primitive — no scope-discriminated split (was D25)

**Decision:** There is exactly one `#Transformer` definition. Component-scope, module-scope, and dual-scope transformers all use the same primitive; scope is derived from match-bucket presence. There is no `#ComponentTransformer` / `#ModuleTransformer` pair.

**Alternatives considered:**

- **Two-primitive split.** Rejected: see TR-D1's "two primitives" alternative. Adds vocabulary for what is structurally the same primitive.

**Rationale:** Map-as-set ergonomics in `#defines.transformers` (DEF-D3) want one type. CUE composition (`transformer.#Transformer & { ... }`) stays uniform. Catalog UIs and the matcher get one shape to introspect.

**Source:** Design conversation 2026-05-01.

**Status:** SUPERSEDED by TR-D5 (two transformer primitives). Retained for the historical record of the prior shape.

---

### TR-D3: `_scope` is a derived field on `#Transformer` (was D26)

**Decision:** `#Transformer._scope` is a hidden, derived field with values `"component"` (default), `"module"`, or `"invalid"`. It is computed from bucket presence (`moduleMatch != _|_` ⇒ `"module"`; both buckets absent ⇒ `"invalid"`). The matcher reads `_scope` to choose iteration mode (per-component or once-per-module).

**Alternatives considered:**

- **Pure inference at matcher time** (no derived field). Rejected: re-implements the inference in every consumer (matcher, catalog UI, diff surface, `#composedTransformers` view in 003). One source of truth is cheaper.
- **Author-declared `scope` field.** Rejected: invites disagreement between declared scope and actual bucket population. A derived field is the single source of truth.

**Rationale:** Surfaces invalid transformers (`_scope == "invalid"` when neither bucket is set) at CUE evaluation time, not at render time (Constitution I — Type Safety First). Keeps the discriminator stable for tooling.

**Source:** Design conversation 2026-05-01.

**Status:** SUPERSEDED by TR-D5 (two transformer primitives — type identity replaces the derived field). Retained for the historical record of the prior shape.

---

### TR-D4: Dual-scope render fan-out semantics (was D27)

**Decision:** When `_scope == "module"` and `componentMatch` is also populated, the transformer fires **once per module** and the matcher pre-populates `#transform.#components` with every component satisfying `componentMatch.required*`. If `componentMatch` declares any `required*` key but no component satisfies it, the transformer does not fire and the platform reports an unfulfilled match. `componentMatch.optional*` keys never gate inclusion — they only widen the read surface.

**Alternatives considered:**

- **Fire per matching component even in module-scope.** Rejected: would emit one Backup Schedule per component instead of one per Module — wrong granularity for cross-component orchestration (Example 7).
- **Always fire even with no satisfying components.** Rejected: a dual-scope transformer with zero eligible components has nothing to operate on; vacuous output is worse than a clear "unfulfilled" signal.

**Rationale:** Module-scope fan-out exists precisely because the work is module-wide; the transformer needs the module's full set of relevant components in a single render call. Refusing to fire when no component qualifies turns a misconfiguration into a deploy-time error.

**Source:** Design conversation 2026-05-01.

**Status:** SUPERSEDED by TR-D5 (the body iterates `#moduleRelease.#components` itself; the matcher's role shrinks to a pre-fire gate via `requiresComponents`, see TR-D7). Retained for the historical record of the prior shape.

---

### TR-D5: Two transformer primitives — 003 owns `#ComponentTransformer`; 005 introduces `#ModuleTransformer` and widens the union (was D28)

**Decision:** Replace the single v1alpha1 `#Transformer` with two primitives in `apis/core/v1alpha2/transformer.cue`:

- **`#ComponentTransformer`** — fires once per matching `#Component`. Match keys (`requiredLabels`, `requiredResources`, `requiredTraits`) read against a single component. `#transform` receives `#moduleRelease` plus the matched `#component` (singular). **Introduced by [003 D17](../003-platform-construct/04-decisions.md).** This enhancement extends `#ComponentTransformer` with `requiredClaims` / `optionalClaims` for component-level Claim fulfilment.
- **`#ModuleTransformer`** — fires once per matching `#Module`. Match keys (`requiredLabels`, `requiredClaims`) read against `#Module` top level. `#transform` receives `#moduleRelease` only; the body iterates `#moduleRelease.#components` itself when dual-scope work is needed. An optional `requiresComponents` field carries a pre-fire gate ("at least one component must carry X"). **Introduced by this enhancement (005).**

This enhancement also widens 003's `#TransformerMap` from `[FQN]: #ComponentTransformer` to `[FQN]: #ComponentTransformer | #ModuleTransformer`. `#defines.transformers` accepts the union. The runtime guarantees `#transform` is invoked with a fully concrete `#ModuleRelease` (003 D18 — applies to both kinds equally).

**003 / 005 ownership note (2026-05-02):** the original wording introduced both primitives in 005. The 003 / 005 untangle moved `#ComponentTransformer` to 003 because the matcher needs a typed transformer primitive at the 003 layer; introducing a sibling primitive in 005 is a true extension rather than a re-shuffle. The original split-motivation (CL-D10 — same Claim FQN at component or module level) only earns its keep once Claims exist, which is also a 005 concern.

**Supersedes:** TR-D1 (scope buckets `componentMatch` / `moduleMatch`), TR-D2 (single-primitive constraint), TR-D3 (derived `_scope` discriminator), TR-D4 (dual-scope semantics via map of components). TR-D1–TR-D4 remain as historical record; TR-D5 (with the 003 split) declares the new shape that replaces them.

**Alternatives considered:**

- **Keep the bucketed single-primitive design (TR-D1–TR-D4).** Rejected: the `_scope` derived field is a crutch that exists only because one signature tries to cover three populations. With the runtime guaranteeing a concrete `#ModuleRelease`, the body can iterate components itself; pre-filtering them into a map buys no expressivity. Two types make scope explicit at the type level and remove the `_scope` derivation.
- **Single primitive with optional `#component` field.** Rejected: shifts the discrimination to nullability, which CUE evaluators and catalog tooling have to repeatedly inspect; type identity is a stronger discriminator and groups naturally in catalog UIs.
- **Pre-filtered component map for dual-scope.** Rejected on the same grounds — the body has the full `#moduleRelease` and can filter freely. The matcher's job is "should this fire?" not "which components should the body see?"
- **Keep `#ComponentTransformer` introduction in 005 (single-doc home).** Rejected during the untangle: 003's matcher and `#composedTransformers` schema need a typed primitive; deferring to 005 leaves 003 with a dangling reference. The split is cleaner.

**Rationale:** The runtime guarantee changes the design pressure. When the transformer always sees a fully concrete `#ModuleRelease`, the matcher's job collapses to two questions — *should this fire?* and *which component(s) does it fire over?* — and those questions split cleanly along type identity (component vs module). The pre-filter convenience that justified the bucketed map disappears. Two primitives is then the minimum-information shape: scope encoded by type, fan-out by matcher dispatch on `kind`, dual-scope expressed by `#ModuleTransformer` + `requiresComponents` + body iteration.

**Source:** User question 2026-05-01 ("So i wonder if `_scope` is actually needed. And i wonder if the transformer should allow two types, `#ModuleRelease` and `#Component`, as in singular."). Decision split between 003 and 005 on 2026-05-02 ("003 owns the redesign of the #Transformer (now #ComponentTransformer) and the matcher and the matcher algorithm. Then 005 extends that design with #ModuleTransformer and #Claim, and so on").

---

### TR-D6: Runtime always passes a fully concrete `#ModuleRelease` to `#transform` — *lifted to [003 D18](../003-platform-construct/04-decisions.md)* (was D29)

**Status:** Lifted to 003 D18 on 2026-05-02 as part of the 003 / 005 untangle. The runtime guarantee applies to both `#ComponentTransformer` (003's primitive) and `#ModuleTransformer` (005's extension); placing it in 003 keeps it alongside the primitive that first relies on it.

**Cross-reference:** [`003/04-decisions.md` D18](../003-platform-construct/04-decisions.md). Original wording retained below for the historical record.

> **Decision:** The OPM runtime guarantees that by the time a transformer's `#transform` body evaluates, `#moduleRelease` is fully concrete: `#config` is resolved, `#ctx` is injected (via 004's `#ContextBuilder`), every `#components[].spec` is concrete, and every `#claims[]` instance carries its complete `#spec`. Transformer bodies may index into `#moduleRelease` freely without re-resolving values.
>
> **Alternatives considered:**
>
> - **Pass partial values and let the body resolve.** Rejected: forces every transformer to know how to drive `#config`/`#ctx` resolution; couples render logic to the runtime pipeline.
>
> **Rationale:** Without this guarantee, the schema would have to allow either party (matcher or body) to perform component filtering for dual-scope, which is what dragged the original design into the bucketed map. With the guarantee, the body owns iteration and the schema stays minimal (TR-D5).
>
> **Source:** User statement 2026-05-01 ("Whenever the `#Transformer` is invoked, it will always receive a fully concrete `#ModuleRelease`. ... The runtime implementation makes sure of that.").

---

### TR-D7: `#ModuleTransformer.requiresComponents` is a pre-fire gate, not a filter (was D30)

**Decision:** `requiresComponents` declares "at least one component carrying X must exist for this transformer to fire". The matcher's `anyComponentMatches` check evaluates the gate; if it returns false, the transformer is skipped and the platform reports an unfulfilled dual-scope render. The render body still iterates `#moduleRelease.#components` itself to determine which components to act on — `requiresComponents` does not pre-select a subset.

The conjunction is single-level: `resources` AND `traits` AND `claims`. Disjunctive gates are deferred (TR-Q4 in `07-claim-fulfilment.md`).

**Alternatives considered:**

- **Pass the matching components into the body.** Rejected: re-introduces the pre-filtered map that TR-D5 dropped. Forces the schema to expose two component sources (the gate's matches and the full module's components), which the body would have to reconcile.
- **Drop the gate entirely; let the body produce empty output.** Rejected: silent no-op outputs hide misconfiguration. A Module that authored a `#BackupClaim` without a `#BackupTrait` on any component should fail the render, not produce an empty Schedule.

**Rationale:** Separating the *gate* from the *body's iteration* keeps each concern in one place. The matcher reasons about whether the transformer should fire; the body reasons about what it does with the components it sees. The gate is declarative metadata for the matcher and for catalog UIs; the body retains full control of cross-component logic.

**Source:** Design conversation 2026-05-01.
