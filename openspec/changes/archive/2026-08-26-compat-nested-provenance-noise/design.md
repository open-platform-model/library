## Context

See proposal.md. `opm/compat/compat.go`'s `walk` recurses plain structs via `walkStruct`, treats everything else (lists, disjunctions, scalars, guarded structs) as a leaf and applies `next.Subsume(prev, cue.Schema(), cue.Raw())`. `StripProvenance` exists but cannot rebuild core-typed members (`#KebabToPascal` not inlined), which is why the cli applies D30 as a path filter over `Check`'s output at the root `metadata` only. `#ExposeTrait.appliesTo: [res.#ContainerResource]` embeds a whole member, so the referenced `metadata.catalogVersion: *"2.0.0-alpha.4" | =~...` lands inside a list leaf and the subsume diagnostic names it.

Constraints: kernel neutrality (pure `cue.Value` logic, no I/O), stable public surface of `opm/compat`, Principle VIII (one package, one session).

## Goals / Non-Goals

**Goals:**
- No violation is ever reported for D30 provenance at any depth.
- Byte-identical leaves never report.
- Member references inside lists are compared structurally.
- Public signatures unchanged.

**Non-Goals:**
- Fixing `StripProvenance`'s inline-imports gap (separate, still recorded in the spec).
- Semantic evaluation of `matchN` / comprehensions on changed leaves.
- Any policy (levels, predecessors) work.

## Research & Decisions

### Provenance at depth
**Context**: D30's filter is applied by the cli only at the root; nested member references carry their own `metadata`.
**Explored**: run log of catalog_opm run 32975611857 (diagnostic text names `catalogVersion:*"2.0.0-alpha.4"` inside `appliesTo`); `provenanceDenylist` in `strip.go`; cli `provenancePaths`.
**Decision**: `walkStruct` skips the fields `catalogVersion` and `description` when the field being walked is a direct child of a field named `metadata` (tracked by passing the parent label down). Reuses `provenanceDenylist`. The skip applies to both operands, so a removed provenance field is also not a `field removed`.
**Rationale**: Same denylist and same "direct children of metadata" scope as D30, applied per occurrence; no round-trip, no rebuild, so it works on core-typed members where the strip does not.

### Lists
**Context**: A list is currently a leaf, so nested member references are opaque.
**Explored**: `cue.Value.List()` iteration; D27's rule has no explicit list clause.
**Decision**: `walkList`: when both sides are lists of equal length, recurse element-wise with paths `name[i]`; otherwise fall through to the existing leaf subsume. Defaults are still checked at the list level first.
**Rationale**: Equal-length lists are the member-reference case and the only one where structural comparison is well-defined; element addition or removal keeps its current subsume verdict rather than inventing list semantics D27 does not state.

### Identical-leaf short-circuit
**Context**: The leaf subsume false-positives on unchanged `#Image`-shaped and `matchN`-bearing leaves.
**Explored**: cli `equalModuloProvenance` (member-level syntax equality, measured necessary); `TestOptionSchemaIsLoadBearing` / `TestOptionRawIsLoadBearing` pin why the subsume options cannot simply be relaxed.
**Decision**: Before the leaf subsume, render both sides with `format.Node(v.Syntax(cue.All()))` and return without a violation when the bytes are equal. Rendering failure falls through to the subsume.
**Rationale**: An identical leaf cannot have narrowed; this is the cli's fast path pushed to the granularity where the walk actually judges. Cost is one format per leaf, acceptable for a publish-time gate. It does not address changed guarded leaves, which the spec keeps as the residue.

Signatures (internal only):

```go
func walk(path string, prev, next cue.Value, acc []Violation) []Violation        // unchanged
func walkStruct(path string, pit, nit *cue.Iterator, next cue.Value, underMetadata bool, acc []Violation) []Violation
func walkList(path string, prev, next cue.Value, acc []Violation) ([]Violation, bool)
func leafIdentical(prev, next cue.Value) bool
```

Impact: `opm/compat` only; no pipeline phase or other package changes. Public surface of `opm/`: unchanged.

## Risks / Trade-offs

- [Skipping `description` at depth hides a genuine description change] → descriptions are documentation, not contract; D30 already exempts them at the root for the same reason.
- [Element-wise list walk pairs elements by index, so a reordered list reports removals/additions] → member reference lists are declaration-ordered in the catalog and never reordered without intent; the cli's member-level fast path still short-circuits an unchanged member, and the residue is a visible path, not a silent pass.
- [Syntax rendering differs for semantically equal leaves] → then the subsume runs as today; the short-circuit can only remove reports, never add one.
- [Per-leaf formatting cost on large catalogs] → measured only on the publish path (66 members); add a benchmark task and revisit if it exceeds the fetch cost.
