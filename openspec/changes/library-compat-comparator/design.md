# Design — library-compat-comparator

## Overview

One new pure package, `opm/compat`, carrying four related pieces of publish-side logic: the D27 comparison walk, the D34 level ladder, the D9 predecessor selection (moved from `materialize/filter.go`), and the D30 provenance strip. The package does no I/O, holds no state, and depends only on `cuelang.org/go` and `github.com/Masterminds/semver/v3`. Callers (the future `cli-catalog-gates`, `library-matching`'s unify rung) compose it with enumeration, pulling, and policy.

The reference implementation is `enhancements/0011/experiments/03-d27-compat-gate` (concluded, cue v0.17.1) — a demonstration of shape and feasibility, explicitly not code to copy. The shipped API fixes three of its shortcuts: the error/violation channel conflation, the level regex divergence from core, and the non-concrete-default false positive.

## Research & Decisions

### Why a field-wise walk and not Subsume

**Context**: D27's additive-only rule must be enforced mechanically at publish.
**Explored**: `enhancements/0011/experiments/03-d27-compat-gate` — 14 cases covering every change class D27 names, against cue v0.17.1.
**Options considered**:
1. `next.Subsume(prev)` — 10/14. Misses disjunction widening (less specific) and defaults.
2. `prev.Subsume(next)` — 8/14, failing on a disjoint set. Misses struct-field addition (more specific) and defaults.
3. Field-wise walk — 14/14, ~100 lines.
**Decision**: The walk. Recurse structs (removed-field, must-be-optional-or-defaulted); forward subsume at leaves only, where it is correct for the value domain (`next.Subsume(prev, cue.Schema(), cue.Raw())` → `domain narrowed`); compare defaults explicitly at every level.
**Rationale**: D27 calls both lattice directions "additive"; a single subsume call tests one. Defaults change nothing about what a value accepts, so no subsume direction can see them. The measured options (`cue.All()` on field iteration, `cue.Schema()`+`cue.Raw()` on the subsume) are load-bearing and MUST be preserved with a test asserting each.

### Level parsing: aligned with core, stated once

**Context**: D34 keys enforcement per primitive `apiVersion`; the experiment shipped its own regex (`^v[0-9]+(alpha|beta)?[0-9]*$`) which admits `v1alpha` (no digit). Core's shipped `#APIVersionType` (`core/src/types.cue:83`) requires the digit: `^v[0-9]+((alpha|beta)[0-9]+)?$`. `library-matching` needs the same ladder for its D34 transitive contract-key comparator.
**Options considered**:
1. Unify against core's `#APIVersionGated` via a `*schema.Cache` — one statement of the ladder, but makes a pure package schema-dependent and I/O-adjacent.
2. Go-side parse, core-aligned regex, parity-pinned by test.
**Decision**: Option 2. `ParseLevel` uses core's exact regex; a test embeds the regex from `core/src/types.cue:83` as a string literal with a comment pinning the source, so divergence is a test failure, not a silent drift.
**Rationale**: Principle I (no hidden I/O; the comparator must run in CI against two values with no registry). The ladder is small and frozen by an accepted decision; the parity test is the cheap insurance. Exposing `Level` with a total, transitive `Compare` also retires the need for a third parser when `library-matching` fixes `sortFQNsBySemVer`'s non-transitivity.

### Package placement

**Context**: where does publish-side comparison logic live?
**Options considered**:
1. New `opm/compat` — single responsibility; importable by the CLI without dragging `kernel`; pure.
2. Extend `opm/materialize` — zero new packages, enumeration/pull already there; but materialize is the package `library-acquire-and-subscription` is about to *narrow*, its doc contract is "realizes a #Platform's subscriptions", and publish has no `#Platform`.
3. `(*Kernel).CheckCompat` — matches the phase-method pattern, but the kernel is the render entry point; publish gating is a different axis.
4. `opm/helper/` — ruled out by the repo's own boundary: helper is skippable convenience, and a compat gate a frontend may skip is exactly the "convention nothing checks" D9 rejects.
**Decision**: Option 1. No kernel wrapper now (YAGNI; add one if a render-side consumer appears).
**Rationale**: The blessed precedent is the `opm/materialize` (pure package) + `opm/kernel/materialize.go` (thin delegation) split — this follows the first half and defers the second until a consumer exists. `CONSTITUTION.md` III's package list and `CLAUDE.md`'s layout gain an entry.

### The move: `highestStable` → `compat.HighestStable`

**Context**: both plans declare the move-before-delete seam; `highestStable` has zero callers outside `materialize` (verified: `filter.go:48` internal + `filter_test.go:157` only), so the move is SemVer-free today.
**Decision**: True move, no transient duplication: `compat.HighestStable` is the only copy; `filterVersions` calls it across the package boundary (`materialize` → `compat` import, no cycle — `compat` imports nothing from this repo). `TestHighestStable`'s four cases move with it (all-stable, skip-higher-prerelease, prerelease-only fallback, unparseable-trailing-entry).
**Rationale**: Two copies invite divergence during the window before `library-acquire-and-subscription` lands. The import is temporary — it dies with `filterVersions`.
**Deliberately not moved**: `enumerateVersions` and `pullCatalog`. They are I/O and their post-D14 fate (diagnostic-only vs deleted) is `library-acquire-and-subscription`'s decision; moving them here would preempt it and drag registry plumbing into a pure package. The comparator takes `published []string`.

### `StripProvenance` lands here

**Context**: `metadata.catalogVersion` changes on every catalog release by construction (D25: provenance only), so an unstripped comparison always reports a violation — the comparator is unusable without the strip. D30 (filed under `library-matching`) specifies the identical mechanism for the unify rung: denylist of exactly `metadata.catalogVersion` + `metadata.description`; syntax round-trip `Syntax(cue.All(), cue.InlineImports(true))` → AST delete from every `metadata` block → `BuildExpr`; must reach the definition as well as the instance.
**Options considered**:
1. Ship in `library-matching` — leaves this comparator broken against real catalogs until that slice lands, and the gate slice would depend on `library-matching` for a helper.
2. Ship twice — divergence risk on a subtle AST transform.
3. Ship once here; `library-matching` applies it at the unify rung.
**Decision**: Option 3, recorded as a scope addition against 0010's plan (implementation here, application there).
**Rationale**: One statement of a non-obvious transform (`InlineImports` is required; the definition-side strip is the part `Validate(cue.Concrete(false))` cannot catch if missed). D30's known cost carries over: the round-trip discards document position, so downstream CUE messages lose file/line — acceptable at the comparator, whose violations are path-located by the walk itself, not by CUE positions.

### Error channel vs violation channel

**Context**: the experiment returns an unparseable `apiVersion` as a `Violation` with empty `Path`.
**Decision**: `CheckAtLevel` returns `([]Violation, error)`. Errors: unparseable `apiVersion`, strip failure, non-struct top-level operands. Violations: D27 breaches only. `Check` (level-blind) returns `[]Violation` alone — it cannot fail given two valid values.
**Rationale**: The library's error doctrine (`opm/errors/errors.go`): diagnostics are structured results, failures are errors; frontends route on types, not strings. A gate that cannot classify its input has not found an incompatibility.

### Violation shape

**Decision**:

```go
type Violation struct {
    Path string // dotted path from the compared root, "" for top-level
    Kind string // one of the Kind* constants
    Old  string // rendered prior value; "" when not applicable
    New  string // rendered new value; "" when not applicable
}

const (
    KindFieldRemoved      = "field removed"
    KindFieldAddedStrict  = "field added without optional or default"
    KindDefaultChanged    = "default changed"
    KindDefaultRemoved    = "default removed"
    KindDomainNarrowed    = "domain narrowed"
)
```

`Old`/`New` exist so 0011's refusal message 9 can render `("daily" -> "hourly")`; the raw CUE subsume error text goes into `New` for `KindDomainNarrowed` (verbatim, consistent with `UnifyError`'s no-reformatting rule). The primitive's name, `apiVersion`, and predecessor coordinate are **caller-attached** — the walk does not know them, and message 9's grouping header is CLI rendering work. Plain slice, no `oerrors` aggregate: violations are results, not errors.

### Non-concrete defaults

**Context**: the experiment's `equalConcrete` returns false when either default is non-concrete, which would report a spurious `default changed`.
**Decision**: defaults are compared only when at least one side has one. Prev-has/next-lacks → `KindDefaultRemoved`. Both present and both concrete → equality. Both present, either non-concrete → mutual subsumption; only asymmetry reports. A test case pins each branch.

## Technical Notes

### File layout

```
opm/compat/
  compat.go        Check, CheckAtLevel, Violation, Kind constants, the walk
  level.go         Level, ParseLevel, Enforced, Compare
  predecessor.go   HighestStable (moved)
  strip.go         StripProvenance
  compat_test.go   14 experiment cases + edge additions (lists, struct disjunctions, hidden fields, non-concrete defaults)
  level_test.go    ladder cases + core-regex parity pin
  predecessor_test.go  the four moved cases
  strip_test.go    definition+instance strip, closedness preserved, InlineImports required
```

### Signatures

```go
func Check(prev, next cue.Value) []Violation
func CheckAtLevel(apiVersion string, prev, next cue.Value) ([]Violation, error)

type Level int // LevelAlpha, LevelBeta, LevelGA
func ParseLevel(apiVersion string) (major int, l Level, ok bool)
func (l Level) Enforced() bool
func CompareAPIVersions(a, b string) int // total, transitive; kube ladder ordering for library-matching

func HighestStable(published []string) string
func StripProvenance(v cue.Value) (cue.Value, error)
```

### Edge coverage beyond the 14 cases

The experiment dispatches on `StructKind` vs leaf. The shipped tests MUST additionally cover: list values at a leaf, a disjunction of structs (walked as leaf — forward subsume; a case documenting the consequence), hidden fields (excluded — `cue.All()` does not include them; documented), and closedness preservation through `StripProvenance` (both spec-body styles, per experiment 03's finding 1).

### Known deferrals (owned by cli-catalog-gates)

- Member enumeration over `<kind>/<apiVersion>/` filing (0010 D49) and `kindPrefix` (D42).
- Predecessor *fetch* (`enumerateVersions` + `pullCatalog` composition) and the member-absent-in-predecessor policy ("no predecessor → pass" vs backward scan).
- Prerelease-predecessor policy: both mainline catalogs publish only `1.0.0-alpha.*` today, so `HighestStable`'s fallback branch (highest overall) is the live path. Whether a publish gate should compare against a prerelease predecessor is gate policy; `HighestStable` keeps its measured semantics.
- Refusal message rendering (0011 06-operational message 9), including its grouping header.
