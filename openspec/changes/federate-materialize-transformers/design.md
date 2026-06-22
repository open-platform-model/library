# Design — federate-materialize-transformers

Governing principle: **ADR-003** (do not construct artifacts by cross-build `FillPath` of a value into a closed, independently-built value). This change applies ADR-003 to materialize by **federating** the native transformer surfaces rather than collapsing them into the closed platform.

## Context

`Materialize` resolves a `#Platform`'s `#registry` subscriptions into a sealed `*MaterializedPlatform`. Today it:

1. enumerates + filters + selects catalog versions per subscription (multi-version possible),
2. `pullCatalog`s each selected version as its own build (shared owner `*cue.Context`),
3. `indexCatalogs` builds, **natively in the owner context**, an open `#composedTransformers` map (FQN → `#ComponentTransformer`) and an open `#matchers` reverse index,
4. **`FillPath`s both onto the closed `c.#Platform`** (`p.Package.FillPath(schema.ComposedTransformers, composed)` …).

Step 4 is the ADR-003 failure: filling the composed map into a closed, separately-built `#Platform` corrupts lazy in-expression resolution of output-local hidden fields inside transformer `#transform`s (bug doc §12 — confirmed a CUE Go-API closedness bug, not a CUE-language or schema bug). The shipped mitigation keeps the open map as `MaterializedPlatform.Composed` and routes the executor to read transforms from it, with a WARNING never to read `#transform` off `Package`. The map is *still* filled onto `Package` (dead and dangerous), so two surfaces exist and correctness depends on a comment.

The sibling change `rewrite-materialize-single-build` proposed removing the seam by composing the platform + catalogs in one CUE build. Its Phase-1 spike proved single-build renders concretely **but** classified **PARTIAL**: CUE Minimal Version Selection admits exactly one version per `path@major` per build, so single-build cannot hold multiple same-major catalog versions. OPM **requires** that (see Decisions → matcher contract), so single-build is rejected as the mechanism.

### The seam, before and after

```
PRE-CHANGE (Composed hatch)                    THIS CHANGE (federation, C1)
  indexCatalogs → composed, matchers (native)    indexCatalogs → composed, matchers (native)
        │ FillPath ↓↓ CORRUPTS                          │  (no FillPath onto closed platform)
        ▼                                                ▼
  Package := p.Package.FillPath(#composed…)        MaterializedPlatform{
  ── closed twin; #transform ⇒ non-concrete _        Source,                 (closed spec / #registry)
                                                      Transformers,  ← #transform CONCRETE, the only surface
  MaterializedPlatform{                               Matchers,      ← reverse index, native
    Source, Resolved,                                 Resolved,
    Package  (matcher reads; #transform corrupt)    }
    Composed (executor reads; concrete)            // no Package, no Composed; no wrong path to read
  }                                                // correctness BY CONSTRUCTION
```

## Goals / Non-Goals

**Goals:**
- Delete the closedness footgun *structurally*: no surface exists from which reading `#transform` corrupts. Remove `Composed` (the comment-enforced workaround) and `Package` (the corrupt twin).
- Preserve multi-version-per-major composition (required) and every other observable materialize behavior.
- Keep the diff small: the native `indexCatalogs` outputs already render concrete; the change is to *expose* them and *stop* the closed-fill.

**Non-Goals:**
- Single-build composition (rejected — MVS-incompatible with required multi-version).
- Changing subscription/filter/selection semantics, the matcher algorithm, or the cache interface.
- Per-version build isolation (C2) — recorded as a future, not built here.
- Fixing the matcher's internal behavior beyond changing the surface it reads (any §10.5 improvement is a verified side effect, not new matcher logic).

## Decisions

### D1 — Federate: expose native surfaces, never fill the closed platform
**Decision**: `MaterializedPlatform` carries `Transformers cue.Value` (the open `#composedTransformers` map from `indexCatalogs`) and `Matchers cue.Value` (the open reverse index). `Materialize` does **not** `FillPath` either onto `p.Package`.
**Why**: The native `indexCatalogs` outputs already render `#transform` concretely (proven by the retained `composed_open_test` and the sibling spike). The corruption was introduced *only* by filling them into the closed `c.#Platform`. Not doing that removes the corruption at the source — there is no closed twin to misread.
**Alternative considered**: single-build composition (collapse the two surfaces into one closed value). Rejected: MVS forbids multiple same-major versions in one build, which OPM requires.

### D2 — Remove `Composed` and `Package` (1a)
**Decision**: Delete both fields. `Source.Package` remains the closed `c.#Platform` spec (reachable for `#registry`, metadata, diagnostics). Transformer reads go through `Transformers`; matcher reads through `Matchers`/`Transformers`.
**Why**: `Composed` was a workaround for the twin; with no twin it is redundant. A field named `Package` on `MaterializedPlatform` that no longer carries the materialized data is itself a footgun (looks materialized, isn't). Verified blast radius: only `compile/match.go` and `cmd/flow-inspect` read `mp.Package` in-repo; `cli/`/`opm-operator/` treat `*MaterializedPlatform` as opaque. So 1a (drop) is clean and rides the same MAJOR bump.
**Alternative considered**: keep `Package` as a pass-through of the unfilled spec for back-compat. Rejected: no consumer needs it, and it reintroduces the "empty/which-surface" confusion.

### D3 — Matcher contract: exact version-bearing FQN (demand-side version pinning)
**Decision**: Matching stays exactly as implemented — a component demands the FQNs it embedded (`component.#resources` ∪ `component.#traits` keys) and looks each up in the reverse index; `matchers[FQN]` yields the transformers requiring that exact FQN.
**Why**: Version selection is an **authoring-time, demand-side** decision. A module pins its catalog version in `cue.mod/module.cue`; the author embeds that version's `#Resource`/`#Trait` into `#Component`s; the version travels into the FQN. So Module-A (built on `catalog@0.5.0`) embeds `…/container@0.5.0` and matches only `…/deployment-transformer@0.5.0`; Module-B (`@0.5.1`) matches only `@0.5.1`. The platform's job is to make **all** subscribed versions available; modules self-select. The matcher never chooses a version — so there is no "two versions both match" ambiguity by construction, and no highest-wins fallback is needed.
**Consequence for §10.5**: today's "multi-version subscription → zero pairs" symptom comes from the matcher reading the **corrupt closed `Package`** (D1's twin). Reading the native `Matchers` index instead is expected to clear it. This is asserted/verified in tests, not assumed; it is a side effect, not new matcher logic.

### D4 — Reuse subscription / filter / selection / cache / error / concurrency unchanged
**Decision**: Version enumeration, range/allow/deny filtering, multi-version selection, `indexCatalogs` (incl. divergent-FQN conflict → `MaterializeError`), `Resolved`, `MaterializeError` shape, non-mutation/idempotency, the opt-in `MaterializeCache`, and the v0.17 concurrent-read-only guarantee are untouched.
**Why**: Orthogonal to the seam, already specified and tested; touching them widens risk for no benefit. The federated `MaterializedPlatform` must remain safe for concurrent read-only sharing (built once, read-only thereafter) — the Platform-CR "materialize-once, reuse-many" model depends on it.

### D5 — C1 now, C2 documented as future
**Decision**: Ship **C1** — one merged native map keyed by version-bearing FQN. Record **C2** (each selected version kept as its own build instance; executor routes each matched transform back to its native build) as a future design.
**Why**: C1 is sufficient and proven: version-bearing FQNs are distinct keys (no collision), each version's deps are resolved in its own `pullCatalog` build before the merge (per-version dep isolation already holds), and execution only fills `#component`/`#context` data (no re-resolution against a native build needed). C2 earns its keep only if a transformer must *re-evaluate against its full native build* at execution time — no current flow does. C1's `Transformers`/`Matchers` fields can grow into a `version → build` collection later without breaking the "look up by FQN, render" contract.

## Risks / Trade-offs

- **§10.5 not cleared by the surface swap** → Add a flow/integration assertion that a multi-version subscription matches and renders end-to-end off the native index; if a residual matcher bug remains, scope it as a separate follow-up (this change still removes the footgun and is independently correct).
- **A hidden external reader of `mp.Package`/`mp.Composed`** → Grep-verified none in `cli/`/`opm-operator/`; `MIGRATIONS.md` documents the field removals and the `Source.Package` / `Transformers` / `Matchers` replacements so any out-of-tree consumer has a recipe.
- **Concurrency regression** → The federated value must not be mutated per-render; assert the existing v0.17 concurrent-read-only scenario against the native `Transformers`/`Matchers`.
- **Re-introducing a closed-fill later** → Add a guard test that fails if `Materialize` fills `#composedTransformers`/`#matchers` onto the closed platform (lock the seam shut).

## Migration Plan

1. Reshape `MaterializedPlatform`: add `Transformers`, `Matchers`; remove `Composed`, `Package`. `Materialize` returns the native `indexCatalogs` outputs and stops the `FillPath` onto `p.Package`.
2. Update consumers in lockstep: `compile/match.go` (read `Matchers`/`Transformers`), `compile/execute.go` + `compile/module.go` (read `Transformers`, drop the WARNING/plumbing), `cmd/flow-inspect`.
3. `MIGRATIONS.md`: record the MAJOR break + recipe (`mp.Composed` → `mp.Transformers`; `mp.Package` spec reads → `mp.Source.Package`; matcher/exec read native fields).
4. Update `docs/design/transformer-output-hidden-field-scope-bug.md` §12/§13 — mark resolved by federation (twin never built), point at this change + ADR-003.
5. Confirm ADR-003 status references this change. (`rewrite-materialize-single-build` and its single-build spike have been removed; the rejected approach and its PARTIAL finding are preserved in this document's Context + D1.)
6. Rollback: the change is internal to the library kernel + its in-repo consumers; reverting restores the `Composed` mitigation. No data or registry migration.

## Open Questions

1. Does the native-index surface swap fully clear §10.5, or does a residual matcher bug remain (→ separate follow-up)? Resolve via the multi-version flow assertion.
2. Should `cmd/flow-inspect` print `Source.Package`'s `#registry`/metadata alongside the native `Transformers`/`Matchers` for parity with today's `Package` dump, or is the native view sufficient?
