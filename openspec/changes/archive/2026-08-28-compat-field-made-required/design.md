## Context

See proposal.md. `walkStruct` (`opm/compat/compat.go`) iterates the prior definition's fields with `Fields(cue.All())`, looks each up on the new side by selector, and recurses; a second loop over the new side catches additions and applies the optional-or-defaulted rule via `sel.ConstraintType()&cue.OptionalConstraint` and `Value().Default()`. The CUE SDK (v0.17.1, `cue/path.go:52-55`) exposes both `OptionalConstraint` and `RequiredConstraint` on a selector. The prior-side loop today never inspects the constraint type.

## Goals / Non-Goals

**Goals:** `y?`/defaulted → `y!`/plain-no-default at the same path is a named violation.
**Non-Goals:** changing what counts as narrowing at leaves; touching the level classification, predecessor selection or the prerelease exemption (cli).

## Decisions

**D1. Posture is computed from the selector and the default, on each side, with one helper.**
```go
// required reports D27's posture: a `!` field, or a regular field with no
// default. `?` and defaulted fields are optional.
func required(sel cue.Selector, v cue.Value) bool {
	ct := sel.ConstraintType()
	if ct&cue.OptionalConstraint != 0 { return false }
	if ct&cue.RequiredConstraint != 0 { return true }
	_, hasDefault := v.Default()
	return !hasDefault
}
```
The addition rule in the new-side loop becomes `required(sel, nit.Value())`, so both rules share one definition of posture on the new side. On the prior side only the optional marker is tested: a defaulted regular field that becomes required is already `default removed` (checkDefaults), and reporting it twice would give one cause two kinds. `y?: string` → `y: string | *"z"` reports no made-required (the leaf subsume's default-sensitive narrowing verdict on it is pre-existing and unchanged).

**D2. Where the prior-side loop learns the new selector.** `next.LookupPath(lookupSelector(sel))` returns the value, not the selector. Iterate `nit` once into `map[name]struct{sel; val}` before the prior-side loop (the second loop already has to skip `seen` names, so the map replaces the second `nit.Next()` pass). No behavior change for the existing rules.

**D3. Report at the field's path, kind `field made required`, and continue the walk.** Same shape as `default changed`: a posture change that also narrows reports both. Alternative (return early on made-required) rejected: the caller aggregates every violation in one pass.

## Research & Decisions

### Why the leaf subsume cannot see this
**Context**: alpha.5 → alpha.6 of `catalogs/opm` slipped a made-required through a gate that exists for exactly this.
**Explored**: 2026-08-28: the gate log shows `0 compared … 39 prerelease-exempt` (the line is exempt, cli 0011 D26); independently, `Subsume(prev, cue.Schema(), cue.Raw())` on `string` vs `string` with different constraint markers reports nothing, because subsumption is over value domains and the marker is a property of the selector.
**Decision**: a struct-walk rule, beside the added-field rule.
**Rationale**: the walk already has both selectors in hand; this is the one place the marker is visible.

## Risks / Trade-offs

- [False positive on a field that was `?` because it was truly optional and is now `!` deliberately] → that is the D27 break the rule exists for; the escape hatch is a new `apiVersion` (0010 D4), which the gate does not compare against the old one.
- [`Fields(cue.All())` iteration of `!` fields] → measured by the existing "add required field" case, which iterates `y: string`; add the `y!:` spelling to the table to pin it.
