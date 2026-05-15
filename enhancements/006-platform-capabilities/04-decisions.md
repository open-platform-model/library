# Design Decisions — Platform Capabilities

## Summary

Decision log for the Platform Capabilities enhancement. Decisions are append-only — do not remove or renumber existing entries. If a decision is reversed, add a new decision that supersedes it.

This enhancement is the successor to 004's `#ctx.platform` extension layer; 004 D36 records the extraction. Several decisions here reference 004's decision log.

---

## Decisions

### D1: `#Capability` is a new construct, not a reuse of `#Resource` / `#Trait`

**Decision:** Platform-supplied context is modelled by a new construct, `#Capability`, sibling to `#Resource` and `#Trait`. It is not the same type as `#Resource` and is not keyed into `#Component.#resources` or `#Module.#defines.resources`.

**Alternatives considered:**

- Reuse `#Resource` for context primitives. Rejected: a `#Resource` is a render *output* — `#Platform.#matchers` (`platform.cue:97-134`) pairs each resource FQN with the transformers that render it. A context primitive is a render *input*; nothing renders it. Keying context into `#Resource` would feed it into the transformer matcher, which would then attempt to render it (or, at best, require every transformer to special-case context FQNs).
- Reuse `#Trait`. Rejected for the same render-output reason, plus `#Trait.appliesTo` carries component-attachment semantics that module-scoped context does not have (the component-scoped variant is OQ3).

**Rationale:** The valuable part of `#Resource` / `#Trait` is the *pattern* — FQN identity, OpenAPIv3 `spec`, FQN-keyed map unification with collision-as-bottom, a reverse-index matcher. That pattern transfers cleanly to context. The *type* does not, because render-input and render-output are different roles in the pipeline. A sibling construct keeps both roles legible.

**Source:** Design discussion 2026-05-14 (004 split brainstorm).

---

### D2: `#Capability.spec` is unwrapped — no camelCase single-field wrapper

**Decision:** `#Capability.spec` is `spec!: _` directly. It does **not** use the `spec!: (strings.ToCamel(metadata.#definitionName)): _` single-field wrapper that `#Resource`, `#Trait`, and `#Blueprint` use.

**Alternatives considered:**

- Mirror `#Resource` exactly (`spec!: (camelName): _`). Rejected: the wrapper exists on `#Resource` / `#Trait` so that many specs can be flattened into one `#Component.spec` (`component.cue:31-55` `_allFields`) without key collisions. Capabilities are never flattened — they are addressed by FQN in `#consumes`. The wrapper would only add a redundant level (`#consumes.required[fqn].spec.route.domain` instead of `#consumes.required[fqn].spec.domain`).

**Rationale:** The wrapper solves a problem capabilities do not have. Keeping `spec` unwrapped makes the read path shorter and the schema simpler. The inconsistency with `#Resource` / `#Trait` is intentional and traceable to the flatten-vs-FQN-keyed difference.

**Source:** Design discussion 2026-05-14.

---

### D3: `#consumes` is split into `required` and `optional` maps

**Decision:** `#Module.#consumes` has two FQN-keyed sub-maps, `required` and `optional`. A `required` entry with no provider is a release-time error; an `optional` entry with no provider is dropped from `#consumes.optional` entirely (the module guards reads with `if … != _|_`).

**Alternatives considered:**

- A single `#consumes` map with a per-entry `optional: bool` marker. Rejected: requires wrapping each entry in a struct (`{#capability: …, optional: …}`), which is more nesting than the two-map form and diverges from the established codebase pattern.
- A single map where required-ness is inferred from whether the consumed schema is fully concrete. Rejected: too implicit; a module author cannot see at the declaration site whether an entry is required.

**Rationale:** The transformer construct already solved required-vs-optional dependency declaration with exactly this shape — `requiredResources` / `optionalResources`, `requiredTraits` / `optionalTraits` (`transformer.cue:55-65`). Mirroring it keeps the catalog's dependency-declaration vocabulary consistent.

**Source:** Design discussion 2026-05-14; prior art `apis/core/v1alpha2/transformer.cue`.

---

### D4: Single `#Platform.#provides` source — no provider layering, no `#Environment`

**Decision:** Capability values come from a single source: `#Platform.#provides`. There is no second provider tier (no `#Environment.#provides`, no Layer 1 / Layer 2). Per-platform variation is handled by **CUE unification of `#Platform` values** — e.g. `#KindDev: #KindBase & {#provides: {...}}`. See OQ6 for whether to formalize the inheritance pattern.

**Alternatives considered:**

- Two provider tiers: `#Platform.#provides` (Layer 1) + `#Environment.#provides` (Layer 2). Rejected: requires reintroducing the `#Environment` construct (which 004 D36 removed) and a fallback-conditional in the matcher. The Layer-2 use case (per-environment overrides) is covered cleanly by CUE unification of `#Platform` values — no new construct needed.
- Environment-only `#provides` (no platform-level provisioning). Rejected: platform-wide defaults (a default storage class available on every release) are a real case; forcing every per-environment Platform to restate them is needless duplication. CUE unification handles this — base + overrides.
- Flat merge with no precedence (drop the question of who wins). Rejected: ambiguous when something has to win. With a single source, the question doesn't arise.

**Rationale:** The simplest design that delivers the mechanism. Per-platform variation is a real need but it is a CUE *value-composition* concern, not a *schema* concern — `#KindDev: #KindBase & {…}` is idiomatic CUE and needs no new construct. Reserving the right to formalize platform inheritance later (OQ6) keeps the door open without locking in a Layer-2 construct now.

**Supersedes:** an earlier draft of this decision that introduced `#Environment.#provides` as Layer 2 (since reversed). The earlier "Platform → Environment precedence" framing borrowed from 004 D24 (itself superseded by 004 D36); both precedence concepts are gone here.

**Source:** User decision 2026-05-15 (drop `#Environment`; capability provider is `#Platform` only).

---

### D5: Matching lives in `#ContextBuilder`, CUE-side

**Decision:** The `#consumes` × `#provides` match is computed in CUE, inside `#ContextBuilder`, as a new step alongside the 004 runtime-context computation. No Go pipeline change. `#ContextBuilder` gains two inputs (`#platform`, `#consumes`) and one output (`out.consumes`); `#ModuleRelease` unifies `out.consumes` back into `#module.#consumes` alongside `out.ctx` and `out.injections`.

**Alternatives considered:**

- Go-side matching with `FillPath` injection. Rejected for the same reasons 004 D7 rejected Go-side `#ctx` computation: it moves logic out of the schema, reduces CUE-native discoverability, and makes the match harder to test in isolation.
- A separate `#CapabilityMatcher` helper, runtime-orchestrated outside `#ModuleRelease`. Considered when an earlier draft tried to keep `#platform` off `#ModuleRelease` entirely; once D13 settled on a kernel-populated `#platform` field on the release, the matcher can ride inside `#ContextBuilder` and the separate helper buys nothing.
- A separate `#CapabilityMatcher` helper invoked inside `#ModuleRelease`. Rejected: needs the same inputs `#ContextBuilder` already takes; an extra helper duplicates the wiring.

**Rationale:** `#ContextBuilder` already assembles `#ctx`; capability matching is one more part of the same assembly. Keeping it there means the entire `#ModuleRelease` data flow goes through one CUE helper — testable as one value, debuggable from one place. The match itself is structurally parallel to `#Platform.#matchers` — a reverse pass over declared requirements.

**Source:** Design discussion 2026-05-15; 004 D7, D14.

**Experimental revision (2026-05-15, [exp 02 F1](../006-platform-capabilities/experiments/02-read-portability-fillpath/README.md#f1--contextbuilder-cannot-be-invoked-inside-modulerelease-revises-d5)):** the matching *algorithm* lives in `#ContextBuilder` CUE-side, as decided. What changes: the CB call **cannot be invoked from inside `#ModuleRelease`** — `_builderOut = CB(#module.#consumes)` plus `#module.#consumes = _builderOut.consumes` is a fixed-point cycle that CUE 0.16.1 does not solve (both default and `CUE_EXPERIMENT=evalv3=1`). The viable shape is: the kernel/runtime invokes `#ContextBuilder` at **top level** (one-shot call with concrete `#module.#consumes` and `#Platform.#provides` inputs) and `FillPath`s the matched entries into `#module.#consumes` before evaluating the release. The "no Go pipeline change" claim is therefore wrong — there is a small Go-side orchestration (load → top-level CB → FillPath matched entries + FillPath `#platform` → evaluate). Experiment 01 confirms the matcher's *logic* is correct (outcome matrix passes); experiment 02 shows the *placement* must move out of `#ModuleRelease`'s body.

---

### D6: Missing required capability surfaces as an incomplete `spec!`

**Decision:** When a `required` capability has no provider, `#ContextBuilder` leaves the `#consumes.required[fqn]` entry as the bare consumed `#Capability`, whose `spec!` is a required-but-non-concrete field. `cue vet -c` then reports `out.consumes.required.<fqn>.spec` as an incomplete value — a release-time error that names the missing capability. No explicit `error()`-style construct is used.

**Alternatives considered:**

- Embed `#platform.#provides[fqn]` (which is `_|_` when absent) so the entry becomes a CUE bottom. Rejected: a bottom-valued field makes the whole `_matched` struct bottom, and the diagnostic points less precisely than an incomplete `spec` on a named FQN path.
- Synthesise a human-readable error string via a CUE contradiction trick. Rejected for v1: pure-CUE error synthesis is awkward and fragile; the incomplete-`spec` path already names the FQN, which is legible enough to act on.

**Rationale:** The incomplete-`spec!` form gives a precise, FQN-named diagnostic with the simplest possible builder code (no embedded bottoms, no contradiction tricks). A nicer error message is a refinement, not a v1 requirement.

**Open follow-up:** Error-message legibility refinement — if release-time diagnostics prove confusing in practice, `#ContextBuilder` can add an explicit no-provider branch that sets a descriptive field. Tracked informally; not an OQ until a concrete confusion surfaces.

**Source:** Design discussion 2026-05-14.

**Experimental confirmation (2026-05-15, [exp 01](../006-platform-capabilities/experiments/01-matcher-mechanics/README.md)):** the matcher in isolation produces exactly this diagnostic — `out.consumes.required."opmodel.dev/.../route@v1".spec.domain: incomplete value =~"…"`. Precise, FQN-named, actionable. **End-to-end caveat ([exp 02 F4](../006-platform-capabilities/experiments/02-read-portability-fillpath/README.md#f4--unbound-release-diagnostic-is-actionable-but-suboptimal)):** under the kernel-orchestrated writeback model (see D5 experimental revision), the unbound-release diagnostic surfaces at the *component-body interpolation site* rather than at the `#consumes.required[fqn].spec` path — still actionable but pointing at the symptom (a non-concrete interpolation) instead of the cause (the missing provider). A clean release-time diagnostic ("platform X does not satisfy module Y's required capability Z") would come from a kernel-side pre-check before CUE evaluation.

---

### D7: `#consumes` is both declaration and resolved read surface

**Decision:** `#Module.#consumes` is **both** the declaration channel (FQN + schema, authored on the module) **and** the resolved read surface (matched provider specs unified back in by `#ContextBuilder`). Module bodies read straight from `#consumes`: `#consumes.required[fqn].spec.X`. There is no separate `#ctx.capabilities` mirror.

**Alternatives considered:**

- `#consumes` is pure declaration; resolved values land in a separate `#ctx.capabilities`. (This was an earlier draft of D7, since reversed.) Rejected: the separation borrowed an architectural parallel from 004 D6's `#config` (operator-supplied) vs `#ctx` (runtime-resolved), but the parallel is forced — `#config`/`#ctx` differ in *who supplies*, while `#consumes` is module-declared and runtime-matched. The right precedent is `#Component.#resources` (`component.cue:23-55`), where a component declares `#resources: { Container: workload.#Container & {spec: ...} }` and reads `#resources.Container.spec.…` from the same place. Co-locating declaration and resolution in `#consumes` mirrors that pattern.
- Inject each consumed capability into per-component scope (à la 004's `#names`). Rejected: per-component injection works for `#names` because each component has its own value; capabilities are module-scoped and shared across components — the FQN-keyed map on `#Module` is the right granularity.

**Rationale:** The simpler design. Authors see one place for "the capabilities I consume" — declared and resolved. `#ModuleContext` stays single-layer (D8). The matcher's output (`out.consumes` from `#ContextBuilder`) unifies cleanly with the declared `#consumes` because they share the same `#CapabilityMap` shape.

**Supersedes:** an earlier draft of D7 that kept `#consumes` as pure declaration with `#ctx.capabilities` as the read surface (since reversed).

**Source:** User decision 2026-05-15 (drop `#ctx.capabilities`); precedent `#Component.#resources`.

**Experimental revision (2026-05-15, [exp 02 F1](../006-platform-capabilities/experiments/02-read-portability-fillpath/README.md#f1--contextbuilder-cannot-be-invoked-inside-modulerelease-revises-d5)):** the read-surface pattern works (claim 1 in exp 02 confirms `value: "jellyfin.apps.example.com"` resolves through the lexical `#consumes.required[fqn].spec.domain` reference). What was wrong in the original D7 framing: "`#ContextBuilder` unifies provider specs back into `#consumes`" via an inline CB call inside `#ModuleRelease`. The actual writeback path is kernel-driven `FillPath` from outside, not an in-CUE unify-back from inside the release. The end result for the module author is identical (read straight from `#consumes`); the mechanism filling the values differs. The `#Component.#resources` precedent still holds — that pattern works because resource specs are supplied *directly* by the author, not derived from a self-referential function call. `#consumes` works the same way once we accept that the kernel is the supplier.

---

### D8: `#ModuleContext` stays single-layer — no `#ctx.capabilities`

**Decision:** `#ModuleContext` (defined by 004 as `{ runtime: #RuntimeContext }`) is **unchanged** by 006. There is no `capabilities` field, no `#CapabilitySet` type, no second `#ctx` layer. Capability reads go through `#consumes` (D7), not through `#ctx`.

**Alternatives considered:**

- Add `capabilities: #CapabilitySet` as a second `#ModuleContext` layer; module reads `#ctx.capabilities[fqn].spec.X`. (This was an earlier draft of D8, since reversed.) Rejected: redundant once `#consumes` carries the resolved value (D7); the second layer would just mirror what `#consumes` already holds. Removing it shrinks 006 (no `#CapabilitySet` type, no `context.cue` modification).
- A separate top-level field on `#Module` (e.g. `#capabilities`), distinct from both `#ctx` and `#consumes`. Rejected: third surface for the same data, no benefit.

**Rationale:** With `#consumes` as the read surface (D7), `#ctx.capabilities` is redundant. `#ModuleContext` is identity-only — `#ctx.runtime` is everything `#ctx` carries. Net effect: `apis/core/v1alpha2/context.cue` is untouched by 006.

**Supersedes:** an earlier draft of D8 that introduced `#ctx.capabilities` as the typed second layer of `#ModuleContext` (since reversed).

**Source:** User decision 2026-05-15 (drop `#ctx.capabilities`).

---

### D9: `#provides` and the two `#consumes` sub-maps are non-optional pattern fields

**Decision:** `#Platform.#provides` and the two maps under `#Module.#consumes` are declared as non-optional pattern fields (`[FQN=#FQNType]: …`), not optional (`?:`). An unpopulated map is `{}`.

**Alternatives considered:**

- Optional fields (`#provides?:`, `#consumes?:`). Rejected: `#ContextBuilder` iterates these maps. An optional-absent field cannot be iterated without a `!= _|_` guard at every use site — the friction 004 D33 documented for the (now-removed) optional `#EnvironmentContext.runtime.cluster`. A non-optional pattern field with no entries is a clean, iterable `{}`.

**Rationale:** Non-optional pattern fields default to empty structs, which iterate to nothing — no guard needed in the builder. This matches how `#Module.#components` (`module.cue:40`) is declared. The cost is that the fields always appear; since they are all `#`-prefixed definition fields, they are excluded from `cue export` output anyway.

**Source:** Design discussion 2026-05-14; 004 D33.

**Experimental confirmation (2026-05-15, [exp 01](../006-platform-capabilities/experiments/01-matcher-mechanics/README.md)):** cases 02-required-missing and 05-optional-missing both pass through the `_matched` comprehension with `#provides: {}` and no guard is needed. The non-optional empty-`{}` pattern iterates cleanly under CUE 0.16.1.

---

### D10: `route` and the cluster-domain override are not reintroduced in v1

**Decision:** 004 D36 removed `route` and the cluster-domain override from `#ctx.runtime` entirely. 006 does **not** reintroduce them — not as `#ctx.runtime` fields and not as shipped `#Capability` definitions. `#ComponentNames` keeps self-defaulting `_clusterDomain` to `"cluster.local"` (004's behaviour). Reintroduction is deferred to OQ1.

**Alternatives considered:**

- Ship a `route@v1` and a `cluster-domain@v1` capability as part of 006. Both fit the capability model cleanly — they require platform input and are not derivable from identity, which is the dividing-line test for capability-vs-runtime. Deferred, not rejected: 006's v1 job is to land the *mechanism* (`#Capability`, `#provides`, `#consumes`, matching, kernel-populated `#platform`). Shipping specific capability definitions and migrating modules onto them is separable follow-up work.
- Reintroduce them as plain `#ctx.runtime` fields fed by a layered hierarchy. Rejected for v1: that re-grows `#ctx.runtime` and brings back the layered shape the 004 slim deliberately removed; if these come back, the capability model is the more consistent home.

**Rationale:** Keep 006 v1 focused on the mechanism. `route` and the cluster-domain override are the obvious first capability definitions to ship, but they are content, not mechanism — OQ1 tracks them.

**Source:** Design discussion 2026-05-14.

---

### D11: Capabilities are imported directly for v1; no `#defines` / `#registry` publication

**Decision:** For v1, `#Capability` definitions are plain CUE definitions in packages, imported directly by platforms (to provide) and modules (to consume). They do not flow through `#Module.#defines` and do not appear in `#Platform.#registry`-derived views.

**Alternatives considered:**

- Add `#defines.capabilities` (parallel to `#defines.resources` / `traits` / `transformers`) and compute `#Platform.#knownCapabilities` from `#registry`. Deferred: this is a real integration — it would let a platform discover capabilities the same way it discovers resources — but it touches 005's `#defines` and 003's `#registry` views, widening the blast radius. v1 keeps the dependency graph small.

**Rationale:** Direct import is the simplest thing that works and is enough to exercise the `#provides` / `#consumes` / matching mechanics. Publication-and-discovery integration is tracked as OQ2.

**Source:** Design discussion 2026-05-14; 004 D26.

---

### D12: Component-scoped capabilities are out of scope for v1

**Decision:** Capabilities in this enhancement are module-scoped: `#Module.#consumes`, resolved into `#consumes` once per release. There is no component-scoped capability — no capability with an `appliesTo` that attaches at component granularity.

**Alternatives considered:**

- Include a "Trait-flavoured" capability now — one that, like `#Trait.appliesTo`, attaches per-component (e.g. a per-component node-pool assignment or per-component certificate). Deferred: module-scoped is the dominant case and the simpler one; component-scoped capabilities need an attachment-and-injection story (something like 004's `#names` injection) that is its own design.

**Rationale:** Start with the dominant case. Module-scoped capabilities cover the motivating examples (app domain, storage class, route). Component-scoped is a coherent extension but a separable one — tracked as OQ3.

**Source:** Design discussion 2026-05-14.

---

### D13: Kernel-populated `#platform: #Platform` field on `#ModuleRelease`

**Decision:** `#ModuleRelease` gains a single `#`-prefixed field, `#platform: #Platform`. End-users authoring a release **do not write** `#platform`. The runtime/CLI/operator unifies `#platform: <chosen>` into the release at apply time (via `FillPath` or equivalent CUE-side unification). The release artifact stays portable; the platform binding is a runtime decision, not an authored one.

`#ModuleRelease` invokes `#ContextBuilder` inline (extending the 004 D34 three-step flow), passing `#platform: #platform` and `#consumes: _withConfig.#consumes` into the builder, and unifying `out.consumes` back into `#module.#consumes` alongside `out.ctx` and `out.injections`.

**Alternatives considered:**

- No field on `#ModuleRelease`; matching done in a separate `#CapabilityMatcher` helper invoked by the runtime outside the release's CUE evaluation. (Considered when an earlier draft tried to keep `#ModuleRelease` `#Platform`-unaware.) Rejected: the mechanism would have required either reaching into `#ContextBuilder`'s `let` bindings via `FillPath` (not possible — `let`s are private) or a two-helper, two-stage runtime orchestration (more moving parts). With a kernel-populated field, the existing 004 D34 flow extends cleanly: one builder call, one unify-back step.
- Have the end-user author `#platform: <chosen>` in the release file. Rejected: couples the release artifact to a specific deployment target — releases stop being portable.
- Reintroduce `#Environment` and have the release reference it (`#env: #Environment`, with `#Environment.#platform`). (Was the "006 reintroduces `#Environment`" plan, since reversed.) Rejected: `#Environment` was a structural intermediary that earned little — the kernel-populated `#platform` field gives the same wiring with one fewer construct.

**Rationale:** Direct precedent — `#TransformerContext.#runtimeName!` (`transformer.cue:121`) is exactly this pattern: a required field the runtime fills, end-user never sees it ("Mandatory — CUE evaluation fails if the runtime forgets to fill this. The value is stamped verbatim onto every rendered resource"). The kernel-populated field is consistent with how the catalog already wires runtime-supplied state into CUE-side evaluation. The `#`-prefix excludes `#platform` from `cue export` so `#Platform` values do not leak into rendered manifests.

`cue vet -c` against an unbound release surfaces the unfilled `#platform` (and any required capability whose provider is therefore non-concrete) as incomplete-value errors — the correct diagnostic for an unbound release.

**Supersedes:** an earlier draft of D13 that reintroduced `#Environment` as the capability-provider node and put `#env: #Environment` on `#ModuleRelease` (since reversed; `#Environment` is not in 006).

**Source:** User decision 2026-05-15 (kernel-populated `#platform` field; drop `#Environment`).

**Experimental confirmation + revision (2026-05-15, [exp 02 F1+F2+F3](../006-platform-capabilities/experiments/02-read-portability-fillpath/README.md)):**

- **Confirmed (F2):** `#`-prefix exclusion works. `cue export` on a bound release emits only `apiVersion`, `kind`, `metadata`, `values`, `components`. No `#platform`, no `#module`, no `#consumes` leak.
- **Confirmed (F3):** `cue.Value.FillPath` against `#`-prefixed paths (`cue.MakePath(cue.Def("#platform"))`, `cue.MakePath(cue.Def("#module"), cue.Def("#consumes"), cue.Str("required"), cue.Str(fqn))`) works as specified. The Go harness `cmd/fillpath/main.go` ships ~50 LoC that does the full FillPath dance and resolves the component-body interpolation to the expected concrete value.
- **Revised (F1):** the kernel's responsibility expands modestly from "fill `#platform`" to "(a) invoke `#ContextBuilder` at top level to compute matched `#consumes`, (b) FillPath each matched entry into `#module.#consumes`, (c) FillPath `#platform` onto the release". The "one builder call, one unify-back step" framing in D13's rationale needs rewriting — the builder call moves out of `#ModuleRelease`'s body and into the kernel's orchestration. The release schema itself carries no CB invocation. Inline-CB-in-release plus unify-back is a self-referential evaluation cycle CUE does not solve.

---

## Open Questions

### OQ1 — Ship `route` and a cluster-domain override as capabilities

004 D36 removed `route` and the cluster-domain override from `#ctx.runtime`; D10 does not reintroduce them in 006 v1. Both fit the capability model cleanly — platform-supplied, not identity-derived. **Why this matters:** until they ship, modules that need a route domain or a non-default cluster domain have no supported channel, and `#ComponentNames` is stuck self-defaulting `_clusterDomain` to `"cluster.local"`. **Revisit trigger:** once 006's mechanism is landed and stable; the follow-up ships `route@v1` and `cluster-domain@v1` capability definitions (and decides whether `dns.fqdn` reads the cluster-domain capability or keeps the self-default).

### OQ2 — Capability publication through `#defines` and `#registry`

D11 has capabilities imported directly. A `#defines.capabilities` channel (parallel to `#defines.resources`) plus a `#Platform.#knownCapabilities` view computed from `#registry` would give capabilities the same publish-and-discover story resources and traits have. **Why this matters:** without it, "what capabilities exist in the ecosystem" has no computed answer — only a platform's `#provides` map (what one platform provides) exists. **Revisit trigger:** when a platform team needs to discover available capability *definitions* (not just provisions), or when 005's `#defines` is next revised.

### OQ3 — Component-scoped ("Trait-flavoured") capabilities

D12 scopes capabilities to the module. A component-scoped capability — one with an `appliesTo` that attaches per-component — would cover per-component context (node pools, per-component certs). **Why this matters:** module-scoped context cannot express "this specific component needs X". **Revisit trigger:** first concrete case where two components in one module need different values of the same capability.

### OQ4 — Transformers consuming capabilities; `#TransformerContext` relationship

Transformers currently receive `#TransformerContext` (`transformer.cue:99-198`), a surface that overlaps `#ctx` (004 OQ1 / D15 defers the unification). If transformers could `#consume` capabilities the same way modules do, the capability model might become the path that unifies `#ctx` and `#TransformerContext`. **Why this matters:** it could resolve 004 OQ1 rather than leaving it for a separate enhancement. **Revisit trigger:** when 004 OQ1 is picked up, or when a transformer needs a platform-supplied fact that is not in `#TransformerContext`.

### OQ5 — Bundle-level capability provision

A `#Module` could itself `#provide` capabilities for sibling modules in a bundle (module A's computed service URL consumed by module B). The `#defines`-style publication pattern would allow it. **Why this matters:** this is the cross-module wiring 004 OQ2 / D20 defers; the capability model gives it a natural mechanism instead of a separate one. **Revisit trigger:** when `#BundleRelease` is designed (004 OQ2's trigger).

### OQ6 — Formalize platform inheritance

D4 establishes a single `#Platform.#provides` source. Per-platform variation is handled today by plain CUE unification of `#Platform` values:

```cue
#KindBase: #Platform & {
    #registry: {...shared...}
    #provides: "...storage-class@v1": {spec: name: "fast-ssd"}
}
#KindDev: #KindBase & {
    metadata: name: "kind-dev"
    #provides: "...route@v1": {spec: domain: "dev.example.com"}
}
```

This works today, no construct needed. The OQ is whether to **formalize** it: a `#extends: #Platform` metadata field so tooling can show "kind-dev inherits from kind-base" without reverse-engineering the unification chain; a `#PlatformInheritance` helper that documents and validates the pattern; registry-level inheritance discovery. **Why this matters:** as ecosystems grow many environment-specialized platforms, the implicit unification chain becomes hard to audit and the structural relationship is invisible to docs/tooling. **Revisit trigger:** first concrete case where a platform team has more than ~3 environment-specialized platforms and wants tooling to surface the inheritance graph; or when a second mechanism for "platform variation" is proposed (at which point the formalized inheritance is the obvious comparison point).

**Experimental confirmation (2026-05-15, [exp 01 case 06](../006-platform-capabilities/experiments/01-matcher-mechanics/README.md)):** `#KindDev: #KindBase & {#provides: {…}}` works as plain CUE unification. Base provisions are inherited, dev provisions extend or override. One caveat to document if `#extends` is formalized: derived platforms cannot override a base-pinned literal — base must use `*default | _` for any field the derived platform may set (the example: `metadata: name: *"kind-base" | _`). Idiomatic CUE; no design change needed, just a small piece of guidance.
