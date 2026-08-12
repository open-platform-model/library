# Tasks — library-compat-comparator

> Every group lands independently green. Group 3 is the move — `materialize` compiles against `compat.HighestStable` in the same commit that deletes the package-local copy. Nothing here changes observable behavior; the whole change is additive plus one internal call redirection.

## 1. Package scaffold + level ladder

- [x] 1.1 Create `opm/compat` with package doc: pure logic, no I/O, callers compose enumeration/pull/policy; consumers are the 0011 publish gate, `opm catalog registry check --compat`, and `library-matching`'s D34/D30 reads.
- [x] 1.2 `level.go`: `Level` (`LevelAlpha`, `LevelBeta`, `LevelGA`), `ParseLevel`, `Enforced`, `CompareAPIVersions` (total, transitive). Regex identical to core `#APIVersionType` (`core/src/types.cue:83` — digit required after `alpha`/`beta`).
- [x] 1.3 `level_test.go`: ladder cases (`v1alpha1` alpha, `v1beta1` beta, `v1`/`v2` GA, `v1alpha` rejected, `V1` rejected), `Enforced` per level, `CompareAPIVersions` transitivity over the measured pathological triple (`v1alpha1`, `v2`, `v10` — same order regardless of input order), core-regex parity pin with source comment.
- [x] 1.4 `task check:fast` green.

## 2. The walk

- [x] 2.1 `compat.go`: `Violation` + `Kind*` constants, `Check`, `CheckAtLevel` with the `([]Violation, error)` split (unparseable apiVersion is an error; alpha returns `(nil, nil)`).
- [x] 2.2 The walk per design: defaults first at every level (non-concrete rules per design), struct recursion with `cue.All()` (missing-in-next → `KindFieldRemoved`; new-in-next requires optional constraint or default → `KindFieldAddedStrict`), leaf forward subsume with `cue.Schema()`+`cue.Raw()` → `KindDomainNarrowed` carrying verbatim CUE error text in `New`.
- [x] 2.3 `compat_test.go`: port all 14 experiment cases (assert each Kind and Path); add edge cases — list leaf, disjunction-of-structs, hidden field excluded, both non-concrete-default branches, `Old`/`New` rendering for a default change. *(Deviations found while measuring, pinned by test: `cue.All()` does include hidden fields — the walk skips them explicitly; open-list element narrowing is invisible to CUE Subsume under every option set, only fixed-length lists are caught; a pure default change double-reports `default changed` + `domain narrowed` because the raw-mode leaf subsume is default-sensitive by design.)*
- [x] 2.4 Options-are-load-bearing tests: one case each that fails if `cue.All()`, `cue.Schema()`, or `cue.Raw()` is dropped.
- [x] 2.5 `task check:fast` green.

## 3. The move (highestStable)

- [x] 3.1 `predecessor.go`: `HighestStable` moved verbatim from `opm/materialize/filter.go:112-123` (doc comment updated to name its publish-side role and the D14 history); delete the `materialize` copy; `filterVersions` (`filter.go:48`) calls `compat.HighestStable`.
- [x] 3.2 Move `TestHighestStable`'s four cases (`filter_test.go:127-158`) to `predecessor_test.go`; delete from `filter_test.go`.
- [x] 3.3 Full `task test` — materialize suite byte-identical behavior (no fixture or assertion changes anywhere outside the moved test).

## 4. StripProvenance

- [x] 4.1 `strip.go`: D30 round-trip — `Syntax(cue.All(), cue.InlineImports(true))` → delete `catalogVersion` + `description` from every `metadata` block → `BuildExpr`. Denylist is a package-level `var` documented as D30's exact set.
- [x] 4.2 `strip_test.go`: instance field removed AND definition's required field removed (the `Validate(cue.Concrete(false))` blind spot); closedness preserved in both spec-body styles; a fixture importing another package to prove `InlineImports` is required (test fails without it); fields outside `metadata` untouched.
- [x] 4.3 One integration-shaped test: two hand-built primitive values differing only in `catalogVersion` — `Check` reports violations unstripped, none after `StripProvenance` on both operands.

## 5. Spec + docs

- [x] 5.1 New spec delta `specs/catalog-compatibility/spec.md` (ADDED requirements: Compatibility Comparison, Level Classification, Predecessor Selection, Provenance Stripping — scenarios per design).
- [x] 5.2 `CLAUDE.md` § Repository Layout gains `opm/compat/`; remove or correct the stale `opm/apiversion/` line while touching the block. `CONSTITUTION.md` III package list gains the compat entry.
- [x] 5.3 No `MIGRATIONS.md` entry (additive, MINOR). Commit as `feat(compat): …`.

## 6. Verify & record

- [x] 6.1 `task check` clean.
- [x] 6.2 Record back in `enhancements/0011/`: slice `library-compat-comparator` → `done`, `openspec_ref: "library/library-compat-comparator"`, history event. Record the D30 scope addition in **both** entries' notes (0011 history event text + a 0010 history event noting D30's strip implementation landed here, application still owed by `library-matching`).
- [x] 6.3 Confirm `enhancements/0010/plan.yaml`'s `library-acquire-and-subscription` is now unblocked (`task plan:ready ID=0010`).
