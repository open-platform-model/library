# Proposal — library-compat-comparator

## Why

Enhancement 0011's publish gate (D9) refuses a catalog build that breaks a contract it already published: for every gated primitive (`#Resource`, `#Trait`, `#Blueprint` — never `#ComponentTransformer`, which has no `apiVersion` to key on), publish pulls the last published build carrying that `name` + `apiVersion` and compares under 0010 D27's additive-only rule. That comparison cannot be `cue.Value.Subsume`: measured across 14 cases covering every change class D27 names (`enhancements/0011/experiments/03-d27-compat-gate`), `next.Subsume(prev)` agrees on 10/14 and `prev.Subsume(next)` on 8/14, failing on disjoint sets — adding a struct field makes a value more specific while adding a disjunct makes it less specific, and D27 calls both "additive". A changed default is invisible to both directions. The field-wise walk the experiment demonstrates scores 14/14.

The comparator is pure `cue.Value` logic and belongs in `library`, shared by three consumers: the `opm catalog publish` gate, `opm catalog registry check --compat` (0011 D7 — same comparator, "the marginal cost is plumbing"), and any CI action. It is level-aware per 0010 D34: the gate runs only at beta and GA, read off each primitive's own `apiVersion` — not the catalog's release version, which is an independent axis (both mainline catalogs publish `1.0.0-alpha.*` releases while carrying `v1beta1` contracts).

Two mechanical companions ride along, each with a hard ordering reason:

- **Predecessor selection moves out of `opm/materialize/filter.go` before 0010 D14 deletes the file.** `filter.go:112`'s `highestStable` implements exactly the selection D9 needs (skip prereleases, fall back to highest overall). Both plans declare this as a move, not a delete-and-rewrite — `enhancements/0010/plan.yaml` makes `library-acquire-and-subscription` depend on this slice for precisely this reason.
- **The D30 provenance strip ships here, not in `library-matching` where the decision is filed.** Every primitive carries `metadata.catalogVersion`, which changes on every catalog release by construction (provenance only, D25) — a comparator fed unstripped operands reports a violation on every single comparison, so the gate would always fire. `library-matching`'s unify rung needs the identical strip (D30's denylist: `metadata.catalogVersion` + `metadata.description`, reaching definition and instance). Shipping it once here, in the package both consumers import, avoids two implementations and a cross-slice dangle. This is a recorded scope addition against 0010's plan (D30 is listed under `library-matching`); the implementation lands here, the unify-rung *application* stays there.

This change is 0011's `library-compat-comparator` slice (see `enhancements/0011/plan.yaml`). Its decisions: D9, plus 0010:D14 (ordering), 0010:D34 (level-awareness), 0010:D30 (strip mechanism, shared), 0010:D27 (the rule being enforced).

## What Changes

- **New public package `opm/compat`** — pure logic, no I/O, no schema-cache dependency:
  - `Check(prev, next cue.Value) []Violation` — the D27 three-rule field-wise walk: recurse structs applying removed-field and must-be-optional-or-defaulted; forward subsume at leaves for the value domain; defaults compared explicitly at every level. Violations are path-located (`spec.retention: field removed`).
  - `CheckAtLevel(apiVersion string, prev, next cue.Value) ([]Violation, error)` — parses the level and returns `(nil, nil)` when the level is alpha (not enforced, D34). An unparseable `apiVersion` is an **error**, not a violation — the experiment smuggled it through the violation channel; the shipped API separates the channels.
  - `Level` / `ParseLevel` / `(Level).Enforced` — the `vNalphaM | vNbetaM | vN` ladder, regex-aligned with core's shipped `#APIVersionType` (`core/src/types.cue:83`, digit required after `alpha`/`beta` — the experiment's looser regex is rejected). This is also the parser `library-matching`'s D34 kube-aware comparator reuses, so the ladder is stated once in Go.
  - `HighestStable(published []string) string` — moved verbatim from `materialize/filter.go:112` with its four test cases; exported.
  - `StripProvenance(v cue.Value) (cue.Value, error)` — the D30 denylist round-trip: `Syntax(cue.All(), cue.InlineImports(true))` → delete `catalogVersion` and `description` from every `metadata` block → `BuildExpr`. Reaches the definition as well as the instance (removing only the instance's field leaves a required field nothing satisfies).
- **`opm/materialize/filter.go` loses `highestStable`**: `filterVersions` calls `compat.HighestStable`; `TestHighestStable` moves to the compat package. Behavior byte-identical — this is the move half of the move-before-delete seam; `library-acquire-and-subscription` later deletes `filterVersions` and the import.
- **No kernel method.** The consumer is the publish-side CLI and CI, not the render pipeline; `Kernel` gains nothing. `cli-catalog-gates` imports `opm/compat` directly.
- **Explicitly out of scope, deferred to 0011's `cli-catalog-gates`:** member enumeration over a catalog tree (needs D49's `<kind>/<apiVersion>/` filing and D42's `kindPrefix`), the pull of the predecessor build, and the backward-scan question ("last build that *carried this member*" vs "last stable build" — `HighestStable` answers which build is the predecessor *candidate*; the gate decides how absence of the member in it is treated). The comparator takes two values and a version list; I/O stays at the edges (Principle I).

## Capabilities

### New Capabilities

- `catalog-compatibility`: the D27 comparison walk, level classification, predecessor selection, and provenance stripping as a pure library surface.

### Modified Capabilities

<!-- none — the materialize call redirection is behavior-preserving; platform-materialization's spec text remains accurate until library-acquire-and-subscription's delta -->

## Impact

- **`opm/` public surface:** one new package (`opm/compat`). Permanent SemVer surface, justified under Principle VII by three named consumers (publish gate, `--compat` check, `library-matching`'s D34/D30 reads). No existing signature changes.
- **SemVer: MINOR** — purely additive; the `filter.go` change is an internal call redirection with identical behavior.
- **Packages touched:** `opm/compat` (new), `opm/materialize` (one call site + test move).
- **Dependencies:** `github.com/Masterminds/semver/v3` (already direct) gains a second importer; `materialize` will lose its semver import when D14 deletes `filterVersions`.
- **Downstream:** nothing consumes `opm/compat` until 0011's `cli-catalog-gates`; `library-matching` consumes `Level` ordering and `StripProvenance`.
- **Ordering:** must land before `library-acquire-and-subscription` (declared edge in both plans) and before `library-matching` (consumes this package).
