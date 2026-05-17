# Design Decisions — `#Platform` Construct

## Summary

Decision log for all architectural and design choices made during this enhancement. Each decision is numbered sequentially and recorded as it is made. Decisions are append-only — do not remove or renumber existing entries. If a decision is reversed, add a new decision that supersedes it.

---

## Decisions

### D1: `#Platform` drops `#providers`; `#registry` of `#Module` values is the sole composition ingress

**Decision:** `#Platform` no longer has a `#providers: [...provider.#Provider]` field. The composition unit is `#Module`, registered through a single map field `#registry: [Id=string]: #ModuleRegistration`.

**Alternatives considered:**

- Keep `#providers` alongside `#registry` — accepts both ingress paths. Rejected: forces every ecosystem participant to ship both forms or accept partial discoverability; perpetuates the duplication described in 01-problem.md.
- Replace `#providers` only with a flat `[Id=string]: #Module` map (no `#ModuleRegistration` wrapper) — simpler shape but loses the enable flag, presentation metadata, and optional release reference that runtime tooling needs.

**Rationale:** Enhancement 005 made `#Module` the ecosystem extension unit. `#defines.transformers` makes `#Module` structurally equivalent to `#Provider` for transformer-registration purposes, while also carrying `#defines.{resources,traits,claims}`, `#components`, and `#claims` that `#Provider` cannot. A single ingress eliminates the duplication and gives every Module slot a platform-level home.

**Source:** User decision 2026-04-30 ("I don't want #providers directly in #Platform. I want it to be extremely dynamic").

---

### D2: `#registry` is fillable from both static (CUE) and runtime sources via the same schema field

**Decision:** `#registry` is declared as a normal CUE field. Platform CUE files write entries directly. The runtime fills additional entries via `FillPath`. CUE unification merges the two sources by Id key.

**Alternatives considered:**

- Two separate fields: `#registry` (static) + `#discoveredRegistry` (runtime). Rejected: forces every computed view to walk both maps and adds a precedence rule with no benefit — unification already handles the merge cleanly.
- Runtime-only: `#registry` is filled exclusively at runtime, platform CUE file uses an indirection (e.g. import a generated file). Rejected: prevents admins from declaring statically-known registrations (OPM core, K8up) at authoring time and breaks self-describing distribution (Constitution VIII).

**Rationale:** A single field with two write paths is the minimum viable schema. The runtime-fill mechanism (Strategy B–style content-hash injection) is deferred to a follow-up enhancement; this enhancement only needs the schema to permit both write paths.

**Source:** User decision 2026-04-30 ("extremely dynamic").

---

### D3: FQN collisions across registered Modules surface as CUE unification errors

**Decision:** When two registered Modules declare a definition under the same FQN (`#defines.resources`, `#defines.traits`, `#defines.claims`, or `#defines.transformers`), the platform's computed view fails CUE unification.

**Alternatives considered:**

- First-write-wins (preserve the first registered Module's value) — silently masks the conflict.
- Last-write-wins — order-dependent and surprising under dynamic registry fills.
- List of values per FQN — punts the resolution to consumers and breaks the keyed-map contract used by the matcher.

**Rationale:** FQN is constructed from `modulePath/name@version`. Two values under the same FQN means two ecosystem participants disagree on the type's shape — a genuine conflict that must be resolved by the admin (disable one registration, pin a different version, file an upstream issue). Failing loud at platform-evaluation time is correct behaviour.

**Source:** Design discussion 2026-04-30.

---

### D4: `#provider` is synthesised internally from `#composedTransformers` *(superseded by D12)*

**Decision:** `#Platform` exposes a `#provider: provider.#Provider` field whose `#transformers` map is `#composedTransformers`. Metadata is filled from the platform's own identity. The matcher reads this synthetic provider unchanged.

**Alternatives considered:**

- Remove `#Provider` entirely and update the matcher to consume `#composedTransformers` directly. Rejected for now: changes the matcher signature and Go pipeline interface; out of scope for a thin platform-construct enhancement.
- Reify `#Provider` as a first-class concept on `#Platform`. Rejected: re-introduces the duplication that D1 eliminated.

**Rationale:** Internal compatibility shim. `#Provider` survives as a transport type the matcher expects; the platform constructs it on demand without exposing it as a composition surface. Future enhancement may dissolve `#Provider` entirely once the matcher is migrated.

**Status:** Superseded by **D12** (2026-05-01). `#Provider` retired entirely; matcher consumes `#composedTransformers` + the new `#matchers` reverse index directly.

**Source:** Design discussion 2026-04-30.

---

### D5: `#registry` value type is `#ModuleRegistration`, not bare `#Module` *(narrowed by D11)*

**Decision:** Each `#registry` entry wraps a `#Module` in a `#ModuleRegistration` struct that adds `enabled` (default `true`) and `presentation?` fields.

**Alternatives considered:**

- Bare `[Id=string]: module.#Module`. Rejected: gives admins no way to disable a registration without removing it, and no place for platform-level curation metadata (self-service catalog category/tags/examples).
- Two parallel maps — one for Modules, one for registration metadata. Rejected: keeps related fields apart and complicates runtime fills.

**Rationale:** The wrapper carries platform-level metadata that is *about* the registration, not about the Module itself. The Module value is referenced via `#module`; everything else describes how the platform handles it.

**Status:** Narrowed by **D11** (2026-05-01). The original draft also carried `#release?` (deploy-state assertion) and `presentation.operator` (admin-facing install metadata); both removed because registration is now a pure projection of `#defines`. Deploy state lives in the `ModuleRelease` CRD reconciler, not in the CUE registration value. Further narrowed by **D14** — `presentation.template` wrapper flattened to a bare `presentation` struct after the only sibling (`presentation.operator`) was already gone.

**Source:** Design discussion 2026-04-30.

---

### D6: `#PolicyTransformer` registration is deferred

**Decision:** `#defines.transformers` is typed as `transformer.#Transformer` only (not `transformer.#Transformer | transformer.#PolicyTransformer`). Policy-scope transformers are not registered through the platform until the policy redesign (enhancement 012) converges.

**Alternatives considered:**

- Include `#PolicyTransformer` in the union now. Rejected: 012 is still in exploration phase and may revise the policy model in ways that change `#PolicyTransformer`'s shape or eliminate it entirely. Including it now risks committing to a shape we are about to change.

**Rationale:** Keep the platform construct thin and unblocked by the open policy work. When the policy redesign settles, a follow-up enhancement adds `#PolicyTransformer` (or its successor) to the platform's view in a single contained change.

**Source:** User decision 2026-04-30 ("Remove #PolicyTransformer for now. i want to figure out a better way to handle policies").

---

### D7: Capability fulfilment for Resources/Traits is registered via transformer `requiredResources` / `requiredTraits`; no separate aggregation map

**Decision:** Capability fulfilment for Resource and Trait requests is registered by the transformer that does the rendering, via the transformer's `requiredResources` / `requiredTraits` fields. `#Platform` exposes `#composedTransformers` and `#matchers.{resources,traits}` only; there is no separate aggregation map keyed by primitive FQN. A consumer Module's `#components[].#resources` / `#components[].#traits` are matched at deploy time by the platform's render pipeline against transformers whose `required*` fields include the FQN.

005 extends this story to Claims: a transformer's `requiredClaims` is the supply registration for component-level (`#ComponentTransformer.requiredClaims`) and module-level (`#ModuleTransformer.requiredClaims`) Claim instances. The same parsimony principle applies — fulfilment is a property of the rendering transformer, not a wrapper primitive (005 CL-D14 retires `#Api`).

**Alternatives considered:**

- Aggregate `#apis: [fqn=string]: prim.#Api` from each registered Module's `#apis` slot. Rejected: 005 CL-D14 removes the `#Api` primitive entirely. Capability supply is now expressed through the transformer that renders the request, not a wrapper primitive.
- List-of-implementations map keyed by primitive FQN, derived from transformer `required*` fields, alongside `#matchers`. Rejected for now: pre-emptively encodes a resolution policy. The matcher already iterates `#composedTransformers` per component; the same iteration resolves the request without an extra projection. Multi-fulfiller resolution is handled by D13 (revised) — overlap is allowed and the runtime matcher disambiguates via predicate evaluation.

**Rationale:** The transformer is both the renderer and the registration; aggregating that information twice (once as transformers, once as a separate map) duplicates state. Keep the platform's outward views minimal; let the matcher discover fulfillers by walking `#composedTransformers`. The same principle scales to Claims when 005 widens the union.

**Source:** User decision 2026-04-30 ("Now i want to remove #Api definition"). Original wording covered Claims; rewritten 2026-05-02 to scope this enhancement to Resources/Traits, with Claims extension delegated to 005.

---

### D8: Platform compatibility detection is the matcher's job; no `#requires` field on `#Module`

**Decision:** A consumer Module declares no platform compatibility. At deploy time the matcher walks the module body for FQN usage — Resource and Trait FQNs from `#components[].#resources` / `#components[].#traits` — and looks each up in `#composedTransformers`. Unmatched FQNs are surfaced as a platform-level signal. The module schema gains nothing — `#Module` stays at 8 slots (see 005 MS-D5).

005 extends this detection to Claims (module-level `#claims` and component-level `#components[].#claims`) once `#Claim` instances exist on `#Module`. The detection mechanism (matcher walks the body; matches against `#composedTransformers`) is unchanged — only the FQN surface widens.

What to do about unmatched FQNs (fail / warn / drop, possibly with per-FQN `criticality` hints on Resource / Trait / Claim definitions) is a separate **platform-team policy** concern that lives at the platform level. That policy is deferred until the catalog `#Policy` redesign (012) converges — detection (this decision) and response (deferred) are independent.

**Alternatives considered:**

- **`#Module.#requires.platformType: "kubernetes" | …`** — declarative platform-type pinning. Rejected: forces module authors to predict every target the module might one day deploy to; the platform already knows what it supports.
- **`#Module.#requires.resourceTypes: [...FQN]`** — declarative resource-type requirements. Rejected: duplicates information already present in `#components[].#resources`; the matcher derives the same set automatically.
- **Per-Resource `criticality` hints** (`must` / `should` / `nice`) shipped on the type definition. Considered. Deferred — pairs naturally with the platform-level matching policy that's also deferred.

**Rationale:** The matcher already walks an FQN graph derived from the module body. A parallel `#requires` declaration creates two sources of truth ("module body says X is used"; "module declares it requires X") that drift. Trusting the matcher keeps the module schema minimal and aligns with D7's "transformer presence is the registration" principle: capability supply = transformer FQN match-keys; capability demand = module-body FQN reads. Symmetric, mechanical, no module-author bookkeeping.

**Cross-references:** 005 MS-D5 (no `#requires` slot on `#Module`); 004 D29 (k8s vocabulary canonical, non-k8s runtimes derive — relies on this matcher mechanism for unmatched detection); D7 (Claim fulfilment via transformer `requiredClaims`); D3 (FQN identity used by the matcher).

**Source:** User decision 2026-05-01 ("I would like for the platform runtime implementation to be responsible for figuring this out").

---

### D9: Non-Kubernetes runtime support is achieved via per-runtime transformer Modules

**Decision:** A non-Kubernetes runtime (Docker Compose, HashiCorp Nomad, future targets) is realised as a `#Module` registered in `#Platform.#registry` whose `#defines.transformers` emit target-specific output for the same k8s-vocabulary Resource / Trait / Claim FQNs that other Modules use. Module bodies are unchanged across runtimes — they read `#ctx.runtime.cluster.domain`, `#ctx.runtime.components.<x>.dns.svc`, etc. The non-k8s runtime Module's transformers map those k8s-shaped fields to local concepts (`namespace` → compose project, `dns.svc` → network alias, see 004 D29). Cross-runtime ecosystem-supplied resolutions (URLs, peer addresses, connection strings) flow through Claim `#status` (005 CL-D15), with each runtime registering its own fulfilling transformer for the relevant Claims.

Drops, warnings, and criticality policy for FQNs the non-k8s runtime cannot render are deferred — same as D8.

**Alternatives considered:**

- **Generic `#ctx.runtime` field shapes** (`searchDomain` instead of `cluster.domain` etc.). Rejected by 004 D29 — produces lowest-common-denominator field names without earning real portability; non-k8s runtimes can map k8s vocabulary mechanically.
- **Per-target subtree split inside `#ctx.runtime`.** Rejected by 004 D30 — would force module bodies to do target-specific reads.
- **YAML-to-YAML translator** (k8s pipeline runs first, produces YAML; compose translator parses it and emits compose YAML). Rejected: bypasses the catalog's transformer architecture; a translator can't access semantic info that a transformer-pipeline can. The same architecture (transformer Modules in `#registry`) handles every runtime.

**Rationale:** A new runtime needs the same primitives as k8s — Resource definitions, Trait definitions, Claim definitions, transformers. The cleanest place for those is a `#Module` with full `#defines`. `#registry` is already the dynamic ingress (D1, D2); adding a compose-runtime Module is the same operation as adding any other transformer-shipping Module. No special-case translator pipeline; no separate runtime-registration channel. Symmetry.

**Cross-references:** 004 D29 (k8s vocabulary canonical); 004 D30 (no `target.<runtime>` split); 005 CL-D15 (Claim `#status` as cross-runtime resolution surface); D1 / D2 (registry as the sole composition ingress).

**Source:** Design conversation 2026-05-01 (platform-model brainstorm).

---

### D10: `#Blueprint` has no platform-level publication or aggregation

**Decision:** `#Platform` exposes no `#knownBlueprints` view. `#Module.#defines` drops the `blueprints` sub-map (paired drop in 005 DEF-D6). `#Blueprint` definitions remain plain CUE types developers import from their declaring packages; the platform never aggregates them.

**Alternatives considered:**

- **Keep `#knownBlueprints` for symmetry with `#knownResources` / `#knownTraits` / `#knownClaims`.** Rejected: symmetry is the only argument. Blueprints have zero downstream consumer in the platform — the matcher walks Resource / Trait / Claim FQNs (transformer.cue `requiredResources` / `requiredTraits` / `requiredClaims`), never Blueprint FQNs. A `#known*` view with no consumer is dead schema.
- **Reshape Blueprints into `presentation.template`** (Blueprints surface only as golden-path templates on a `#ModuleRegistration`). Rejected: breaks the "all type definitions live under `#defines`" pattern without removing the underlying redundancy. If Blueprints are not platform-aggregated at all, the simpler cut is to remove the publication slot entirely.
- **Defer the cut, leave the schema as-is.** Rejected: 003 is unimplemented. Removing now is free; removing after release is a breaking change.

**Rationale:** Blueprint is a CUE composition of Resources + Traits + Claims. It has no runtime semantics — the matcher never matches against a Blueprint FQN, no transformer's `requiredResources` / `requiredTraits` / `requiredClaims` references one, and at deploy time a Blueprint's contribution has already expanded into the Component's `#resources` / `#traits` slots before render begins. The Component's `#blueprints` field stays (used for spec-field merging via `_allFields` in component.cue) but is internal to Component, not a platform-aggregation surface. Discovery for hypothetical tooling (`opm new <blueprint>`) can walk `#registry[*].#module` directly without a dedicated `#known*` field — same deferral pattern as OQ4.

Asymmetry with the other `#defines.*` slots is correct, not a flaw: Resources, Traits, and Claims earn their `#known*` views because the matcher reads them (Resources / Traits via transformer match keys; Claims via demand-side FQN walk + transformer `requiredClaims` supply registration — D7). Blueprints earn nothing.

**Cross-references:** 005 DEF-D6 (paired drop of `#defines.blueprints`); D7 (capability fulfilment via transformer `requiredClaims`, no separate `#apis` aggregation — same parsimony principle); D8 (matcher walks Resource / Trait / Claim FQNs only); core/v1alpha2/component.cue (Component retains `#blueprints` for spec merging).

**Source:** User decision 2026-05-01 ("I don't think we need to register the blueprints. A blueprint is just a composition of #Resources, #Traits, and #Claims" → Option E).

---

### D11: `#ModuleRegistration` is a pure projection of `#defines`; deployment is owned by `ModuleRelease` + `opm-operator`

**Decision:** `#ModuleRegistration` carries no install or deploy metadata. The registration value reflects "this Module's primitives (`#defines`) are visible on this platform" — nothing more. Installation of `#components` is an operator-driven step:

1. User creates a `ModuleRelease` CR referencing a Module (with `#defines` populated).
2. `opm-operator` reconciles the CR: installs `#components` against the cluster *and* `FillPath`s the Module value into `#Platform.#registry[id].#module`.
3. Registration and installation are a single operator-driven step; the CUE model does not carry separate "install instructions" or release-state assertions.

Concretely this drops two fields from D5's draft `#ModuleRegistration` shape:

- **`#release?: _`** — removed. Encoded cluster state in the CUE value, conflating two concerns. Cluster state lives in the `ModuleRelease` CRD reconciler; the registration only reflects the consequence of a successful reconcile.
- **`presentation.operator: { description?, installNotes? }`** — removed. `#module.metadata.description` already covers admin-facing copy; install notes belong in Module documentation, not in platform-level schema. The platform never needs to read "how to install" — the operator already has the full Module value.

`presentation.template` survives because it carries platform-curation data (category / tags / examples for self-service surfacing) that the Module itself cannot know — it is information about how *this platform* surfaces the Module, not about the Module.

**Alternatives considered:**

- **Keep `#release?` for static asserts.** Rejected: admins who want to assert "this Module is materialised" can write a `ModuleRelease` CR; they do not need a parallel CUE channel. Two channels = drift.
- **Keep `presentation.operator` for admin-facing install hints.** Rejected: install hints are documentation, not schema. Tooling that wants admin-facing help can read `#module.metadata.description` (or a future doc surface on `#Module`); the registration does not need its own copy.
- **Drop `presentation` entirely; let tooling derive everything from `#module.metadata`.** Considered. Rejected because per-platform curation (category / tag overrides for the same Module surfaced under different platforms) is a real need that the Module cannot pre-bake.

**Rationale:** Registration ≠ deployment. Conflating them creates two sources of truth (CUE registration value vs. live `ModuleRelease`) and forces every consumer to reconcile both. Pure projection — `#registry[id].#module` carries the Module value, the operator owns the install + the FillPath, and downstream views (`#knownResources`, `#composedTransformers`, …) recompute automatically. No install metadata in CUE.

**Cross-references:** D2 (registry fillable from static + runtime via the same field); D5 (narrowed by this decision); D14 (`enabled: false` use case for staged registrations); D15 (concurrent-write conflict policy when static + runtime disagree); OQ1 (runtime-fill mechanism — `opm-operator` reconciler is the implementation path); OQ4 (self-service catalog runtime — consumes `presentation`).

**Source:** User decision 2026-05-01 ("we don't need ANY instructions for the operators. No install instructions. Installing the #Module as a ModuleRelease (with #defines defined) will automagically register the primitives").

---

### D12: Matcher logic lives on `#Platform` via `#matchers` + `#PlatformMatch`; `#Provider` retired

**Decision:** Replace the synthetic `#provider` (D4) with two native constructs in `core/v1alpha2/platform.cue`:

1. **`#Platform.#matchers`** — a computed reverse index over `#composedTransformers`. Two submaps at this layer (`resources`, `traits`) each keyed by FQN; the value is the list of transformer candidates whose `required*` field includes that FQN. Resources and Traits index `#ComponentTransformer` only (CL-D11). 005 adds a `claims` submap (indexing the union of `#ComponentTransformer` and `#ModuleTransformer` per TR-D5) once Claim-fulfilling transformers exist.
2. **`#PlatformMatch`** — a per-deploy walker. Inputs: a `#Platform` and a consumer `#Module`. Outputs: `matched`, `unmatched`, and `ambiguous` projections that the Go pipeline / `opm-operator` consumes per render pass. `_demand` is computed from the consumer Module's component bodies (`#components[*].#resources/#traits`). 005 extends `_demand` with module-level and component-level Claim FQNs.

`#Provider` and the `provider.cue` file are deleted. The matcher Go interface migrates to read `#composedTransformers` and `#matchers` directly. `#declaredResources` and `#declaredTraits` (convenience FQN lists on `#Platform`) are also dropped — `#knownResources` / `#knownTraits` map keys give the same information.

**Alternatives considered:**

- **Keep `#provider` as a permanent compat shim** (status quo from D4). Rejected: shim survives only to preserve a Go interface that this enhancement is already changing; carrying it forward bakes in dead schema. Killing it now is a one-time cost vs. perpetual confusion about why a synthetic field exists.
- **Index transformers without exposing `#matchers`** (matcher Go code rebuilds the reverse index every render). Rejected: the index is a deterministic projection of `#composedTransformers`; CUE expresses it once, the Go pipeline reads it. Rebuilding per render duplicates work and hides the contract.
- **Make `#PlatformMatch` a method on `#Platform`** (no separate construct). Rejected: CUE definitions are not parameterised functions. A separate construct that takes `platform!` and `module!` is the clean expression of "instantiate a match per deploy".
- **Surface the render plan (matched transformer per FQN) directly instead of candidate lists.** Rejected: resolution policy (OQ5 — multi-fulfiller selection) is not yet decided; surfacing candidate lists keeps the schema honest and gives policy a place to plug in.

**Rationale:** D4 deferred the matcher migration as out-of-scope. With D11 simplifying `#ModuleRegistration` and the schema otherwise stabilising, deferring `#provider` removal trades short-term scope for long-term debt. The reverse index belongs to `#Platform` because it is a deterministic projection of the platform's transformer catalog. The per-deploy walker belongs alongside it because it operationalises D8 detection (unmatched FQNs) and exposes the OQ5 hook (ambiguous candidates) without committing to a resolution policy. Schema and Go pipeline land in lockstep; no window where the two disagree.

**Cross-references:** D4 (superseded); D7 (capability fulfilment via `requiredClaims` — `#matchers.claims` is the operationalisation); D8 (matcher detects unmatched FQNs — `#PlatformMatch.unmatched` is the surface); D13 (revised — multi-fulfiller allowed; runtime predicate evaluation pairs candidates with components); 005 TR-D5 (two transformer primitives — what the index keys against).

**Source:** User decision 2026-05-01 ("Lets come up with a new set of definitions that work within the scope of #Platform. It should handle the matching logic").

---

### D13: Multi-fulfiller is allowed; runtime matcher resolves via per-candidate predicate evaluation

**Decision (revised 2026-05-13 to match landed implementation):** Two registered Modules' `#defines.transformers` may legitimately produce transformers whose `requiredResources` / `requiredTraits` overlap on the same FQN. `#Platform` evaluation does not fail; `#matchers.{resources,traits}` simply lists every candidate transformer per FQN. The runtime matcher in the Go pipeline evaluates each candidate's predicate (label match, optional-trait satisfaction, etc.) against the consumer `#Component` and pairs every surviving candidate with that component.

The CUE schema therefore carries no `_invalid` / `_noMultiFulfiller` hidden projection. The earlier guard (forbid multi-fulfiller at the CUE layer, fail platform-eval with `_|_`) was reversed during implementation in favour of letting the runtime matcher do the disambiguation it was already designed to do via predicate evaluation. `apis/core/v1alpha2/platform.cue` documents this directly in its `#Platform` header comment.

`#PlatformMatch.ambiguous` (design-only — `#PlatformMatch` itself was not landed in CUE) remains a diagnostic concept; in practice the Go pipeline's output is "every paired (component, transformer) tuple after predicate filtering", and ambiguity only matters when two candidates survive predicate evaluation for the same component — that case becomes a runtime diagnostic, not a platform-eval failure.

005 keeps the same revised semantics for Claims: multiple transformers may publish overlapping `requiredClaims` FQNs; predicate evaluation against the consumer's Claim instance disambiguates at fulfilment time.

**Alternatives considered (original D13 framing, prior to reversal):**

- **Forbid multi-fulfiller at the `#matchers` layer** (the original D13). Implemented in the experiment harness as `_invalid` + `_noMultiFulfiller`; rejected at integration time because the runtime matcher's predicate evaluation is the right place to disambiguate — it has the consumer component in hand and can apply label/optional-trait gates that the CUE-only reverse index cannot.
- **Admin-selected default fulfiller per Claim FQN, consumer-pinned, or registry priority order.** Each requires a new schema slot and a tie-breaker policy. Predicate evaluation subsumes these for the common cases.
- **Last-registered-wins.** Order-dependent and surprising — still rejected.

**Rationale:** Multi-fulfiller is the normal ecosystem case (Postgres backup transformer + filesystem backup transformer both require the `Backup` trait; the predicate picks the right one per component). Failing platform-eval on registration would force admins to choose at registration time rather than letting per-component predicates pick automatically. The cost of CUE-side forbidding (over-strict, blocks legitimate fan-out) outweighed the benefit (early failure surface).

**Supersedes:** Reopens **OQ5** ("Conflict resolution when two transformers declare overlapping `requiredClaims`") — answered by "predicate evaluation in the Go matcher". 005 TR-Q2 likewise resolves to predicate evaluation.

**Cross-references:** D3 (FQN-collision-on-types fails CUE unification — type-level uniqueness still enforced); D12 (`#matchers` reverse index lists all candidates per FQN); 005 TR-D5 (two transformer primitives — what `#matchers` indexes); 005 CL-D16 (status writeback — single fulfiller per Claim is now a runtime-matcher invariant, enforced by predicate disjointness).

**Source:** Implementation reversal landed with `apis/core/v1alpha2/platform.cue`; revised wording captured 2026-05-13.

---

### D14: `enabled: false` hides every projection of a `#ModuleRegistration` entry

**Decision:** When `#ModuleRegistration.enabled` is `false`, every computed view that walks `#registry` skips the entry — `#knownResources`, `#knownTraits`, `#composedTransformers`, and (transitively) `#matchers` (plus `#knownClaims` once 005 lands). The Module's primitives are completely hidden from the platform until the flag flips.

The previous wording ("import types but skip transformer composition") was incorrect — the schema already gates every projection on `if reg.enabled`, so disabling skips types as well as transformers.

**Alternatives considered:**

- **Half-disable: hide transformers only, keep type definitions visible** (the discarded original wording). Rejected: the platform views are conceptually one bundle ("what does this registration contribute?"); peeling them apart creates a third state ("registered, types visible, transformers dark") whose semantics no consumer needs.
- **Drop `enabled` entirely; admins remove the entry instead.** Rejected: removal loses curated `presentation` metadata and breaks runtime-injected entries that are managed by `opm-operator` reconciliation. Toggling a flag is the cleaner operator interface.

**Rationale:** Use case is staging — opm-operator may FillPath an entry from a `ModuleRelease` CR before the admin is ready to activate it; flipping `enabled` to `true` in a follow-up reconcile is cleaner than racing the runtime fill against the admin's static declaration. Full hide also matches the principle of least surprise: a disabled registration produces zero platform-side effects.

**Cross-references:** D2 (registry fillable from static + runtime); D11 (registration is consequence of release).

**Source:** User decision 2026-05-01 ("enabled = false should hide everything registered from that module").

---

### D15: Concurrent static + runtime writes to the same `#registry[Id]` use CUE unification; concrete-value conflict surfaces in `ModuleRelease.status.conditions`

**Decision:** When the platform CUE file declares `#registry["k8up"]` statically and `opm-operator` `FillPath`s `#registry["k8up"]` at runtime, CUE unification merges the two writes by Id. Three outcomes:

1. **Identical values** → idempotent; no change.
2. **Disjoint fields** (static sets `presentation`, runtime sets `#module`) → both populate.
3. **Concrete-value disagreement** (static `version: "1.0.0"`, runtime injects `2.0.0`) → `_|_` at platform-evaluation time.

The schema does **not** add an override hierarchy. The opm-operator reconciler catches the bottom value via `cue.Value.Err()`, parses the field path of the conflict, and writes a structured condition to the offending `ModuleRelease.status.conditions` (e.g. `type: "RegistryConflict", message: "#registry[\"k8up\"].#module.metadata.version conflict: declared 1.0.0, injected 2.0.0"`). Admins resolve by aligning the static declaration with the CR or removing one of the two.

**Alternatives considered:**

- **Schema-level split** (`#registry` for static, `#discoveredRegistry` for runtime). Rejected by D2 — forces every computed view to walk two maps; precedence rule offers no benefit unification doesn't already give.
- **Runtime authoritative** (operator overwrites static on conflict). Rejected: breaks unification semantics; static admin declarations become advisory, surprising readers of the platform CUE.
- **Static authoritative** (operator skips + warns when concrete static value present). Rejected: couples the reconciler to CUE-side diffing; complicates the operator without clear payoff.

**Rationale:** CUE already models the merge correctly. The only real question was *where* the failure surfaces — the schema cannot raise it without an override hierarchy that breaks D2. Putting the surfacing in the reconciler keeps CUE pure and gives the admin an actionable signal instead of an opaque field-path error.

**Cross-references:** D2 (registry fillable from static + runtime via the same field); D11 (operator-driven registration); OQ1 (runtime-fill mechanism — D15 is the conflict-handling half of that contract).

**Source:** User decision 2026-05-01 ("for 16, Option A sounds good") — selected from a three-option menu (CUE unification + reconciler diagnostic / runtime authoritative / static authoritative).

---

### D16: `#registry` Id keys MUST be kebab-case (`#NameType`)

**Decision:** `#registry: [Id=#NameType]: #ModuleRegistration`. The Id key is constrained by the existing `#NameType` regex (kebab-case). Convention is to set Id to `#module.metadata.name`. Non-kebab Ids fail CUE unification at definition time.

**Alternatives considered:**

- **Bare `string` Id** (current pre-decision shape). Rejected: opens the door to "k8up" vs "K8up" duplicates, especially under runtime injection where a CR field passes through unsanitised; admins would discover the collision only at platform-eval time, with an opaque error.
- **Default Id to `#module.metadata.name` automatically.** Considered (see new OQ7). Deferred — the cosmetic win is small and the unification rule for default-from-sibling-field is fiddly enough to deserve its own follow-up.

**Rationale:** Kebab-case matches the rest of the catalog's identifier conventions (Module names, Component names, Resource/Trait/Claim names). One identifier shape across the system reduces author cognitive load and makes runtime injection behave predictably.

**Cross-references:** D2 (registry merge contract — Id space must be unambiguous for unification to merge cleanly); D15 (concurrent-write conflicts — kebab constraint helps avoid case-mismatch collisions).

**Source:** User decision 2026-05-01 ("Yes, add constraints").

---

### D17: `#ComponentTransformer` is the sole transformer primitive at this layer

**Decision:** v1alpha1's single `#Transformer` is replaced in v1alpha2 by `#ComponentTransformer` — a primitive that fires once per matching `#Component` with match keys `requiredLabels` / `optionalLabels` / `requiredResources` / `optionalResources` / `requiredTraits` / `optionalTraits`. The render body receives the `#ModuleRelease` plus the matched `#Component` (singular) and a `#TransformerContext`. `#defines.transformers` is typed `[FQN]: #ComponentTransformer` at this layer; the publication slot accepts only this kind. Schema lives in `apis/core/v1alpha2/transformer.cue` (new file).

005 introduces a sibling primitive `#ModuleTransformer` (per-module fan-out) and widens `#TransformerMap` to `#ComponentTransformer | #ModuleTransformer`. The original motivation for splitting `#Transformer` into two kinds was Claim placement (005 CL-D10 — same Claim FQN at component or module level); without Claims, one kind is sufficient. Naming `#ComponentTransformer` here (rather than keeping the bare `#Transformer` and renaming later) makes the future extension purely additive — 005 introduces a sibling, no rename.

**Alternatives considered:**

- **Keep the bucketed single-primitive design** (original 005 TR-D1–TR-D4 with `componentMatch` / `moduleMatch` buckets and a derived `_scope` field). Rejected: at this layer the only scope is component, so the buckets carry no information; a flat shape is the minimum-information form. 005's TR-D5 already retired the bucketed design.
- **Pure inference at matcher time** (no schema change in v1alpha2 — keep v1alpha1's `#Transformer` shape, redesign only the matcher). Rejected: the matcher needs typed access to `requiredResources` / `requiredTraits` separately from the labels; v1alpha1's flat shape works but mixes axes. Cleaner to author one primitive for v1alpha2 and let 005 extend it than to graft a Claim story onto v1alpha1's vocabulary.
- **Defer the `#Transformer` redesign to 005** (let 003 reference v1alpha1's `#Transformer`). Rejected: D12 already refactors how the matcher consumes transformers; the schema needs to land in lockstep so the Go pipeline interface change is internally consistent.

**Rationale:** D12 dropped `#Provider`; the matcher consumes `#composedTransformers` + `#matchers` directly. That refactor needs a typed transformer primitive that the matcher can inspect (`requiredResources`, `requiredTraits`, `metadata.fqn`). Introducing `#ComponentTransformer` here keeps 003 a complete unit — schema, matcher, algorithm — and makes 005 a true extension rather than a bidirectional dependency.

**Cross-references:** D7 (capability fulfilment via `requiredResources` / `requiredTraits`); D12 (matcher consumes the primitive directly); D18 (runtime guarantee — concrete `#ModuleRelease`); 005 TR-D5 (the union widening); 005 CL-D10 (Claim placement, the original split motivation); `05-component-transformer-and-matcher.md` (full design narrative + matcher algorithm).

**Source:** User decision 2026-05-02 (untangle 003 / 005 ownership — "003 owns the redesign of the #Transformer (now #ComponentTransformer) and the matcher and the matcher algorithm").

---

### D18: Runtime always passes a fully concrete `#ModuleRelease` to `#transform`

**Decision:** The OPM runtime guarantees that by the time a transformer's `#transform` body evaluates, `#moduleRelease` is fully concrete: `#config` is resolved, `#ctx` is injected (via 004's `#ContextBuilder`), and every `#components[].spec` is concrete. Transformer bodies may index into `#moduleRelease` freely without re-resolving values. The guarantee is a runtime-implementation contract, not a CUE-schema check.

This guarantee shapes the schema: there is no need for the matcher to pre-filter components into a map for the body's convenience. `#ComponentTransformer.#transform` receives the matched `#component` directly; the body can also walk `#moduleRelease.#components` for cross-component reads. The same guarantee applies unchanged to 005's `#ModuleTransformer` — its body iterates `#moduleRelease.#components` for dual-scope work, and 005's `requiresComponents` field is a pre-fire gate (not a filter).

**Alternatives considered:**

- **Pass partial values and let the body resolve.** Rejected: forces every transformer to know how to drive `#config` / `#ctx` resolution; couples render logic to the runtime pipeline.

**Rationale:** Without this guarantee, the schema would have to allow either party (matcher or body) to perform component filtering for cross-component renders, which is what dragged the original 005 transformer design into the bucketed map. With the guarantee, the body owns iteration and the schema stays minimal.

**Cross-references:** D17 (the schema this guarantee makes possible); 005 TR-D5 / TR-D6 / TR-D7 (the parallel application to `#ModuleTransformer` and the `requiresComponents` pre-fire gate); 004 D11 / D12 (centralised context-injection precedent that the runtime uses to satisfy this guarantee).

**Source:** User statement 2026-05-01 ("Whenever the `#Transformer` is invoked, it will always receive a fully concrete `#ModuleRelease`. ... The runtime implementation makes sure of that."). Lifted to 003 on 2026-05-02 as part of the 003 / 005 untangle.

---

## Open Questions

Captured here while the enhancement is thin (no separate `NN-open-questions.md` yet).

### OQ1 — Runtime-fill mechanism

`#registry` is declared as fillable from runtime, but the mechanism (Go-side `FillPath` versus CUE-side discovered-registry import versus operator CRD reconciliation) is not specified here. **Revisit trigger:** when the first runtime-fill source is implemented (likely `opm-operator/`).

### OQ2 — `#Platform.type` role beyond UX / registry filtering

`#Platform.type` is **kept** as an authored field. D8 (matcher detects unmatched FQNs) subsumes the type-mismatch *detection* concern. The remaining open question is what `type` carries weight on beyond display — UX hints (catalog UIs filter "compatible Modules" by type), registry-filter shortcuts before walking FQNs, or a future enforcement contract. The field stays informational for now. **Revisit trigger:** when self-service catalog tooling ships and we discover whether the field earns enforcement.

### OQ3 — Migration of existing provider packages *(ANSWERED 2026-05-17)*

**Status:** ANSWERED. The OPM-core transformers ship as Module form at `library/modules/opm/transformers/` (deployment, service, configmap, secret, statefulset, daemonset, cronjob, job, pvc, role, crd, sa-resource, plus the route family — grpc/http/tcp/tls). No `providers/` packages remain under `library/`. Vendor packages (k8up, cert-manager, etc.) follow the same Module-form pattern.

Original text retained for the historical record:

> `opmodel.dev/opm/transformers/kubernetes`, `opmodel.dev/k8up/transformers/kubernetes`, `opmodel.dev/cert_manager/transformers/kubernetes` all currently export `#Provider` values. Each must be re-shaped as a `#Module` with the existing transformers under `#defines.transformers` (and OPM core gains the catalog of resources/traits under the rest of `#defines`). **Revisit trigger:** separate migration enhancement after this lands.

### OQ4 — Self-service catalog runtime API

`presentation` declares the metadata; the consuming surface (`opm catalog list`, web UI, deploy-time matcher) is platform-implementation territory. **Revisit trigger:** when the first self-service catalog tooling is implemented in `cli/` or `opm-operator/`. Consistent with 005 DEF-Q1.

### OQ5 — Conflict resolution when two transformers declare overlapping `requiredClaims` *(ANSWERED by D13 revised)*

**Status:** ANSWERED 2026-05-13 by **D13 (revised)** — multi-fulfiller is allowed; the runtime matcher in the Go pipeline disambiguates per consumer component via predicate evaluation (label match, optional-trait satisfaction, etc.). The CUE schema lists every candidate transformer per FQN in `#matchers.{resources,traits}`; predicate evaluation reduces the candidate list to the pair(s) that apply for a given component.

Original text retained for the historical record:

> Two registered Modules may each ship a transformer (`#ComponentTransformer` or `#ModuleTransformer`) whose `requiredClaims` includes the same Claim FQN (e.g. one Postgres operator and one Aiven operator, both fulfilling `ManagedDatabase`). The platform's render pipeline must pick one per consumer request. Candidates: admin-selected default fulfiller per Claim FQN, consumer-pinned fulfiller (transformer FQN), or registry priority order.

### OQ6 — Topological-sort algorithm for `#status` writeback ordering

`#PlatformMatch` exposes `matched.claims` per Claim FQN, but the order in which transformer fulfillers run vs. consumers that read `#claims.<id>.#status` is not modelled in CUE. The dependency graph (write edges from transformer `requiredClaims`; read edges from `#claims.<id>.#status.<field>` reads) is computed and traversed in the Go pipeline. Cycle detection, missing-fulfiller signalling, and ordering policy live in Go. **Revisit trigger:** Go pipeline implementation, or a real cycle case from a catalog Module. Paired with **005 CL-Q7**.

### OQ7 — Default `#registry` Id from `#module.metadata.name`

D16 constrains Id keys to kebab-case but does not auto-default Id to `#module.metadata.name`. Authors must currently spell the Id explicitly (`"k8up": #module: k8up.#Module`). A CUE-side default (something like `[Id=#NameType]: #ModuleRegistration & {#module: metadata: name: Id}`) would remove the duplication but adds unification ceremony with corner cases (what if author overrides `Id` to something other than the Module name?). **Revisit trigger:** experiment 002 surfaces real authoring friction, or first admin who hits the duplication.
