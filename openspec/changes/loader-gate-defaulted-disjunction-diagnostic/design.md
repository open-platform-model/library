## Context

See proposal.md § Why. The gate is `shape.Gate` → `requireConcrete` (`opm/helper/loader/internal/shape/shape.go`), shared by `loader/file` and `loader/registry`. `requireConcrete` tests `f.Exists()`, then `f.IsConcrete()`, then non-emptiness for strings. On a defaulted disjunction `IsConcrete()` is false while `f.Default()` returns the default with `hasDefault == true` (measured against `cuelang.org/go v0.17.1`, the pinned SDK). The three sentinels are re-exported from `loader/file` and matched with `errors.Is` by the CLI and the controller.

## Goals / Non-Goals

**Goals:**
- The not-concrete refusal on a defaulted disjunction names the default and the rule, and points the author at the declaration.
- The spec states that a defaulted disjunction is not concrete for the gate.

**Non-Goals:**
- Changing the gate's verdict. Applying `Default()` before `IsConcrete()` would accept the form; that is rejected below.
- Any change to sentinels, signatures, `loader/registry`'s identity verification, or `kind` handling.
- Fixing the producers of the form (`cli` scaffold and templates, fixtures): sibling changes in their repos.

## Decisions

**D1. Keep `IsConcrete()` as the verdict; add `Default()` only on the failing branch.**
The alternative is to resolve defaults first (`if d, ok := f.Default(); ok { f = d }`) so the fleet's form passes. Rejected: (a) identity is "the two values a release moves" (core SPEC § 5.2), and a value that can be unified away by a consumer is not that; (b) the spec's own wording, "fields the schema never defaults", was written on the assumption that a non-concrete identity field means unfinished authorship, and an author-side default is exactly the case it did not anticipate, so the fix is to say so, not to widen the gate; (c) widening ships in a kernel release every consumer re-pins against and would let the `cli` scaffold keep emitting a form that only works because the loader forgives it. The message change is PATCH; the verdict change would be a behavior change with no consumer asking for it.

**D2. Message shape.** On `!IsConcrete()`: if `f.Default()` reports a default, the error is
`required field %q is a defaulted disjunction (default %v), not a concrete value: identity fields must be concrete literals: %w`; otherwise the existing `is not concrete` text. Same `%w` sentinel. The default is rendered with `%v` on the default `cue.Value` so a string default prints quoted. No new exported symbol.

**D3. Where the test lives.** `opm/helper/loader/file/validate_test.go` already builds malformed packages from inline CUE (`TestShapeGate_RejectsMalformedPackages`); add a case whose `metadata.version` is `#T | *"1.0.1"` and assert both `errors.Is(err, ErrMissingRequiredField)` and the message substring naming `"1.0.1"`. A second case with a plain-literal identity confirms the pass path. No new fixture directory.

## Research & Decisions

### Why `IsConcrete()` and `String()` disagree
**Context**: `cue vet -c` and the CLI's publish gates accept the form the loader refuses.
**Explored**: 2026-08-28, Go probe against `cuelang.org/go v0.17.1`: on `Version: #T | *"1.0.1"`, `IsConcrete()` = false, `Kind()` = `_|_`, `Default()` = `"1.0.1"` (hasDefault true), `String()` = `"1.0.1"`; on `"\(Version)"`, `IsConcrete()` = true. The `cue` CLI finalizes before printing, so both forms print `"1.0.1"`.
**Decision**: the gate keeps asking `IsConcrete()`; the message explains the gap.
**Rationale**: `String()` resolving defaults is why every consumer downstream of the gate works, and why publish did not catch the form; the gate is the one place that asks the strict question, and it should say what it is asking.

## Risks / Trade-offs

- [A consumer matches the old message text] → sentinels are the documented contract (`errors.Is`); the `helper-packages` spec's existing scenarios assert sentinel and field path, not the full string. Grep `cli` and `opm-operator` for `"is not concrete"` before merging; measured 2026-08-28: no Go source in either matches.
- [`Default()` on an already-failed value costs an evaluation] → only on the error path; negligible.
