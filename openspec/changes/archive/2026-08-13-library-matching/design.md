# Design — library-matching

## Overview

Four independent strands land in one slice because they share the match rung's files and test fleet: the `matchLabels` flip (D36), the load-bearing unify rung (D26/D27/D30), demand resolution semantics (D28 with D4's diagnostics), and the provider guard (D32/D37). The fifth strand (D10's own-graph test) is pure test work. Each strand is a separately-green task group; the breaking surface is the union of their semantic changes.

## Research & Decisions

### Which label sites flip (D36)

**Context**: 0010's measurement names four `metadata.labels` readers. They are not all matching reads.
**Decision** per site:

| Site | Role | Fate |
| --- | --- | --- |
| `compile/match.go:111` (set build) | matching | **flips** to `matchLabels` via new `schema.MatchLabels` path constant |
| `compile/match.go:350` (predicate, + `pairTransformer`'s reuse of the same set) | matching | **flips** (follows the set) |
| `compile/module.go:197` (`ComponentSummary.Labels`) | display | **stays** on `metadata.labels`; doc comment updated to say matching no longer reads it (its stale `core.opmodel.dev/workload-type` citation goes) |
| `schema/context.go:68` (transformer context) | render | **stays** — D36 explicitly keeps `matchLabels` out of `#TransformerContext`; render reads (`hpa_transformer`'s workload-type lookup off the component) depend on `metadata.labels` surviving |

**Rationale**: D36 draws exactly this line: `matchLabels` is matching vocabulary, `metadata.labels` is descriptive. Flipping site 4 would break shipped render behavior; flipping site 3 would make the summary lie about what it displays.
**Component derivation is core's job**: `#Component.matchLabels` is derived wholesale from attached primitives (`_matchLabelsAreDerived` forbids component-authored keys), so the matcher reads the component's `matchLabels` exactly as it read `metadata.labels` — one `LookupPath` swap per site.

### The unify rung's provenance exclusion (D26/D30) — mechanism revised at implementation

**Context**: `unifyIntersection` unifies the component's `#resources[fqn]`/`#traits[fqn]` against the transformer's `required*[fqn]` — both sides carry `metadata.catalogVersion`, which diverges across builds by construction. D30 specifies excluding exactly `metadata.catalogVersion` and `metadata.description` from the comparison; the original plan applied `compat.StripProvenance` (the publish-side syntax-round-trip strip) to both operands.
**Measured at implementation — the strip cannot run at the rung**: kernel-side operands are schema-derived values whose `Syntax(cue.All(), cue.InlineImports(true))` export carries let-bound references to core helper definitions (`#KebabToPascal`, comprehension machinery); the rebuild fails with unresolved references against both the real GHCR catalog and registrytest fixtures. Every export profile that does rebuild them (`Eval()` first, `cue.Final()`) was measured to OPEN closed definitions — a module-set field the definition closes out sails through, silently voiding the D27 closed-definition comparison the spec requires. The publish-side compat gate is unaffected (its operands export self-contained); `compat.StripProvenance` stays as shipped.
**Decision (revised)**: exclude provenance from the unify VERDICT instead of from the operands. Unify unstripped, validate with `cue.Concrete(false)`, then drop every CUE diagnostic located at a `metadata` block's `catalogVersion`/`description` (any depth — the "every metadata block" literal rule); if no diagnostic survives, the pair unifies clean; otherwise the surviving diagnostics are the recorded `UnifyError.Cause`. Equivalent to the strip under D25 (the denylisted fields are provenance-only; nothing derives from them), and strictly cheaper — no syntax round-trip, no memoisation needed.
**Closedness**: nothing is round-tripped, so closedness is preserved trivially; the rung keeps unifying the closed definition, which is what makes D27 enforceable at match time (cutting at `spec` was measured to silently drop module-set fields).
**Cost reversal**: D30's recorded cost (post-strip diagnostics lose document position) does not materialize — surviving diagnostics keep their positions. `UnifyError.{Component, FQN}` remain the routing surface; provenance-only conflicts no longer appear in the cause text.

### Demand resolution semantics (D28, D4 diagnostics)

**Context**: three distinct unresolved states exist today: (a) empty bucket → `MissingFQN` recorded in `plan.Missing`, unconsumed; (b) bucket present, every candidate disqualified (unify or predicate) → nothing recorded; (c) component pairs nothing at all → `Unmatched`, the only fatal state.
**Decision**:
- A demanded **resource** in state (a) or (b) is fatal. A demanded **trait** in state (a)/(b), or left unconsumed by every matched transformer's `optionalTraits`, is fatal iff its effective `optional` is false; otherwise it degrades to today's warning.
- New value-type diagnostic in `oerrors`:

```go
type UnresolvedDemand struct {
    Component    string
    FQN          string   // the contract key demanded
    Kind         string   // "resource" | "trait"
    Alternatives []string // same-base contract keys the platform does implement, kube-aware order
    Disqualified []UnifyError // state (b): the candidates that existed and why they fell out
}
```

- `MatchPlan` accumulates them (new `Unresolved []UnresolvedDemand` field beside `Missing`; `Missing` stays for compatibility, fed as today). `Kernel.Match` remains phase-only and returns the plan; `Plan` and `Compile` fail when `len(Unresolved) > 0`, through the same exit path as `UnmatchedComponentsError` (a sibling typed error, `UnresolvedDemandsError`, with `Unwrap() []error`).
- The D4 contract-key diagnostic is carried by `Alternatives`: empty → "nothing on this platform implements this contract"; non-empty → "implemented at a different apiVersion" with the alternatives listed. No `MissingFQN` shape change — the distinction is derivable and the new type owns the message.
**Trait posture read**: effective `optional` is read from the component's trait attachment value (`#traits[fqn].optional` — catalog default unified with any attachment-site override). A non-concrete `optional` **fails closed**: the trait is treated as load-bearing and the diagnostic names the unstated posture, so an un-gated catalog cannot silently weaken a render. (The publish gate — 0011 D22's `#TraitOptionalGate` — makes this unreachable for gated catalogs; the library still meets un-gated trees.)
**Rationale**: D28's amended text verbatim; fail-closed is the only reading consistent with "every declared resource is required" — absence of evidence of optionality is not optionality.

### The comparator split (D34/D4)

**Context**: `sortFQNsBySemVer` orders `MissingFQN`/`UnresolvedDemand` alternatives; measured non-transitive (rule switches per pair). Its only call site handles contract keys.
**Decision**: contract-key ordering delegates to `compat.CompareAPIVersions` (total, transitive, kube ladder); the SemVer branch survives only where build keys are ordered (currently nowhere in match — the function is renamed to say what it orders). No mixed lists exist: D4's key-shape split means a call site knows which shape it holds.

### The single-provider guard (D32/D37)

**Context**: the guard keys on declared `fulfilment` and counts required demands. `#Catalog` exposes no primitive maps — the only place materialize can read a contract's `fulfilment` is the provider's embedded copy (`transformer.requiredResources[fqn].fulfilment` / `requiredTraits[fqn].fulfilment`).
**Decision**: during `indexCatalogs`' existing walk, for each transformer and each **required** contract key, read `fulfilment` off the embedded definition. Accumulate `providerOf[key] → set of owning catalogs` for keys any provider declares `fulfilment: "provider"`. After the walk: two catalogs for one provider-fulfilled key → `MaterializeError` naming both catalog paths and the key (provenance is structural — `catalogBuild.Subscription`). Two embedded copies disagreeing on `fulfilment` for one key → also an error (divergent contract definitions for one key; unifying them would mask a catalog bug).
**Explicitly unchanged**: the match candidate loop (`match.go:138-157`) — per 02-design, D14+D32 guarantee a provider-fulfilled bucket holds at most one entry before it is read; catalog-fulfilled buckets legitimately hold many and all satisfied candidates pair.
**Open acknowledgment**: the embedded copy is the provider's claim about the contract, not the declaring catalog's word. If a provider lies about `fulfilment`, the guard sees the lie. The alternative (reading the declaring catalog's member) requires `#Catalog` to expose primitive maps — an additive core extension 0010 explicitly deferred. Recorded; the disagreement error above is the partial mitigation.

### The own-graph test (D10)

**Context**: graduation gate: "verified by a test that would fail if the module and the platform shared one resolution." The property holds structurally (`compile/module.go:137` consumes `mp.Transformers` as read-only input from a separate resolver).
**Decision**: an integration test (registrytest) where the **platform's catalog** publishes a primitive definition that *diverges* from the one the **module's own dependency** carries for the same contract key (e.g. an extra required field with a different domain). Assertions: the unify rung reports the divergence (`UnifyError` — proving the two values came from different resolutions and were actually compared), and the module's rendered output reflects the module-side definition. Under a shared resolution the two values would be one value, the unify rung would trivially pass, and the test fails on the missing `UnifyError`.
**Rationale**: asserts the observable consequence of separate resolution rather than an implementation detail; survives refactors that preserve the invariant.

### Fixture strategy

- `registrytest` generated primitives gain `matchLabels` (mirroring today's label values), a `fulfilment` field (default `"catalog"`; test knob for `"provider"` cases), and keep `optional: bool | *true` with a knob for `*false` and for stating no posture (the fail-closed case).
- `compile_test.go`'s raw-CUE fixtures (schema-bypassing) move their matching keys from `metadata: labels:` to `matchLabels:` — no derivation constraint applies to them.
- `testdata/modules/web_app`: the component's explicit `metadata.labels` **stays** (render reads; hpa-style transformers) but its transitional comment is rewritten — it is no longer what matching reads.
- The GHCR flow test is the flip's proof against real catalog bytes: the published catalog authors both `matchLabels` and the duplicate, so it passes before and after; a follow-on catalog release may drop the duplicate.

## Technical Notes

### Ordering interactions

- Hard dependency: `opm/compat` (`StripProvenance`, `CompareAPIVersions`) — `library-compat-comparator` must be merged first.
- `library-acquire-and-subscription`: no shared files (`materialize.go`/`filter.go` vs `index.go`); either merge order works. Plan convention sequences it first.
- `add-migration-guard`: if live, this PR carries `Migration: library-matching` and the MIGRATIONS entry follows its template.

### Error/exit path sketch

```
Match  → MatchPlan{Matches, Missing, Unresolved, UnifyErrors, UnhandledTraits(optional-only)}
Plan   → error: UnresolvedDemandsError | UnmatchedComponentsError (first non-empty wins? no — both reported, joined)
Compile→ same gate as Plan before execution
```

Both aggregate errors implement `Unwrap() []error`; frontends keep routing on types. Warnings() keeps carrying only effectively-optional unhandled traits.

### MIGRATIONS entry contents

- Unresolved resource demands and non-optional unhandled traits now fail `Plan`/`Compile` (previously: silent or warning). Recipe: mark genuinely-optional traits `optional: true` at the attachment site; remove or platform-support undemandable resources.
- Matching reads `matchLabels`; a module whose matching relied on hand-authored component `metadata.labels` must attach primitives that carry the keys (component `matchLabels` is derived, never authored).
- Unify verdicts now ignore provenance-only divergence (`metadata.catalogVersion`/`metadata.description`), and provenance conflicts no longer appear in `UnifyError` causes; route on `UnifyError` fields, not message patterns.
- Platforms subscribing two catalogs providing one provider-fulfilled contract now fail materialize.
