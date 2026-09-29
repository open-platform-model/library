## Context

See proposal.md for why. The behaviour is specified in `specs/single-build-render/spec.md`; this document covers where it lives in the render pipeline.

Matching and the fail-closed gate are CUE inside the one render build (`opm/internal/renderstage/render.cue.tmpl`). The Go side stages the build, reads `diagnostics` and `rendered` off the built value, and raises typed causes from the decoded rows (`opm/kernel/render.go`, `opm/kernel/render_decode.go`). The pieces this change touches:

- `#Match.verdicts[cid]` computes, per component, `_resOutcome` / `unresolvedResources` and `_traitOutcome` / `_handled` / `unresolvedTraits`. Each unresolved row carries `kind`, `fqn`, `definedBy`, `alternatives` and `disqualified`.
- `#Match` flattens them into `pairs`, `unmatched`, `unresolved`, `warnings`, `unifyFailures` and `resolved: len(unresolved) == 0 && len(unmatched) == 0`.
- `guard._providerSuppliers` maps every contract key that an enabled entry's transformer requires with `fulfilment: "provider"` to the set of registry keys supplying it. `guard.overSubscribed` is built from it.
- `rendered` and `diagnostics.failedPairs` iterate `match.pairs`. `gate` is `match.resolved & (len(guard.overSubscribed) == 0) & true`.
- The glue template has three Go-filled slots (`<<.InstanceImport>>`, `<<.PlatformImport>>`, `<<.RuntimeName>>`), filled by `renderstage.RenderGlue(GlueInputs)` and called from `renderstage.Stage`.
- `decodeRenderDiagnostics` decodes `diagnostics` into `glueDiagnostics` and copies the rows onto `RenderDiagnostics`. `gateErrors` raises a cause per non-empty refusal list.

## Goals / Non-Goals

**Goals:**

- The skip decision is made inside the build, from the same data the gate reads, so a frontend never re-derives which demand is unprovided.
- The default path (switch off) produces the same pairs, rows, refusals and output as today; only the new `unprovided` marker and message suffix appear on rows.
- The render module stays self-refusing: its own `gate` agrees with the kernel under both switch values.

**Non-Goals:**

- Choosing, installing or naming a provider. The kernel reports what it skipped; the frontend words it.
- Skipping anything but an unprovided, provider-fulfilled demand (see the spec's refusals that stand).
- A per-contract or per-component skip list. The switch is one bool per render; nothing asks for finer control.
- Touching the platform's contract inventory (`platform.#contracts`), the single-provider guard, or `opm/helper`.

## Decisions

### The unprovided predicate reuses the guard's provider count

Inside `#Match` a new parameter `#providers` is bound to `guard._providerSuppliers`. A demand is unprovided when the component's attached primitive declares `fulfilment: "provider"` and `#providers[fqn]` does not exist:

```cue
#Match: {
	#transformers: [string]: _
	#components: [string]:   _
	#definedBy: [string]:    string
	// Contract key -> registry keys whose transformers REQUIRE it with
	// provider fulfilment: the single-provider guard's own count.
	#providers: [string]: _
	// The caller's switch, a literal in the generated module.
	#skipUnprovided: bool
	...
}

match: #Match & {
	#transformers:   _transformers
	#components:     _components
	#definedBy:      platform.#contracts.definedBy
	#providers:      guard._providerSuppliers
	#skipUnprovided: _skipUnprovided
}
```

`guard` reads only `platform.#registry`, never `match`, so binding it into `#Match` adds no cycle.

**Alternative:** derive "no transformer requires the key" inside `#Match` from `#transformers`. That is equivalent for provider-fulfilled keys, but it would be a second count of the same fact. The core rule is defined as the guard's count being zero, so the predicate reads the guard's own map.

### Each unresolved row carries `unprovided`; the refusal keeps only the rows the switch did not skip

The row builders gain one field, and each component gains its split:

```cue
unresolvedResources: [
	for _, _f in _resFqns if !_resOutcome[_f].satisfied {
		{
			kind: "resource"
			fqn:  _f
			// ...definedBy, alternatives, disqualified as today...
			unprovided: [
				if comp.#resources[_f].fulfilment == "provider" && #providers[_f] == _|_ {true},
				false,
			][0]
		}
	},
]
// unresolvedTraits likewise, reading comp.#traits[_f].fulfilment.

_skipResources: [for u in unresolvedResources if #skipUnprovided && u.unprovided {u}]
_skipTraits:    [for u in unresolvedTraits if #skipUnprovided && u.unprovided {u}]
refusedResources: [for u in unresolvedResources if !(#skipUnprovided && u.unprovided) {u}]
refusedTraits:    [for u in unresolvedTraits if !(#skipUnprovided && u.unprovided) {u}]
omitted: len(_skipResources) > 0
```

The flattened verdicts read the split:

```cue
pairs: [
	for cid, v in verdicts if !v.omitted
	for tfqn, _ in v.matched {{component: cid, transformer: tfqn}},
]
unmatched: list.Sort([
	for cid, v in verdicts if len(v.matched) == 0 && !v.omitted {{...as today...}},
], ...)
unresolved: [
	for cid, v in verdicts for _, u in v.refusedResources {{u, component: cid}},
	for cid, v in verdicts for _, u in v.refusedTraits {{u, component: cid}},
]
skipped: [
	for cid, v in verdicts for _, u in v._skipResources {{
		component: cid, kind: u.kind, fqn: u.fqn, definedBy: u.definedBy,
		alternatives: u.alternatives, componentOmitted: v.omitted
	}},
	for cid, v in verdicts for _, u in v._skipTraits {{...the same fields...}},
]
warnings: [for cid, v in verdicts if !v.omitted for _, _f in v.unhandledWarnings {{component: cid, fqn: _f}}]
resolved: len(unresolved) == 0 && len(unmatched) == 0
```

`diagnostics` gains `skipped: match.skipped`. `rendered` and `failedPairs` already iterate `match.pairs`, so an omitted component neither executes nor reports a failed pair. `gate` is unchanged, because `resolved` already excludes skipped rows. The build order of `skipped` (all components' resources, then all components' traits) mirrors `unresolved`, so the kernel re-sorts nothing.

With the switch off, `_skip*` are empty, `omitted` is false and `refused*` equal `unresolved*`. The pairs, unmatched rows, unresolved rows and warnings are therefore the same as today, plus the `unprovided` field. The parity harness is untouched.

**Why not cycle-prone:** `omitted` depends on `_resOutcome` (buckets, `_unify`, `_pred`) and on `#providers`. `pairs` depends on `matched` and `omitted`. `matched` never reads `omitted`, `pairs` or anything flattened, so the dependency graph stays a DAG. The spike (tasks section 1) confirms this, and also confirms that the `fulfilment` default (`*"catalog"`) resolves inside the `if` guard the marker is built from, as the posture guards already rely on.

**Alternatives:**

- Filter in Go after decode. Rejected: the gate lives in the build and must agree with the kernel (spec "The render module's own gate agrees"), and `rendered` would still evaluate the omitted component's pairs.
- Omit the component from `_components` before matching. Rejected: that needs the verdicts that come from matching, which is a cycle.

### An omitted component drops its warnings but keeps every other refusal

An omitted component renders nothing, so an unhandled optional trait on it is not news and leaves `warnings`. Its other unresolved rows stay in `unresolved`: a catalog-fulfilled or provided-but-unmatched demand on that component still refuses. Its trait rows that are unprovided are skipped rows with `componentOmitted: true`, so the frontend can say "not rendered" once per component.

### Go surface: one input field, one diagnostics field, one row type, one row field

```go
// opm/kernel
type RenderInput struct {
	// ...
	// SkipUnprovided renders what the platform can when a component demands
	// a provider-fulfilled contract that no enabled catalog provides.
	// Off, the default, refuses such a render.
	SkipUnprovided bool
}

type RenderDiagnostics struct {
	// ...
	// Skipped is every demand skipped under RenderInput.SkipUnprovided, in
	// build order; empty when the switch is off. Advisory: a frontend words it.
	Skipped []SkippedDemand
}

// SkippedDemand is one provider-fulfilled demand skipped because nothing on
// the platform provides it. Data, like every diagnostics row.
type SkippedDemand struct {
	Component        string
	FQN              string
	Kind             string // "resource" or "trait"
	DefinedBy        string
	Alternatives     []string
	ComponentOmitted bool // the component rendered nothing; set on all its skipped rows
}

// opm/errors
type UnresolvedDemand struct {
	// ...
	// Unprovided is true when the demanded contract is provider-fulfilled
	// and no enabled catalog provides it: exactly the demands
	// kernel.RenderInput.SkipUnprovided would skip.
	Unprovided bool
}
```

`SkippedDemand` lives in `opm/kernel` beside `ResolvedVersion` and `Replacement`, the other advisory rows. It is not a gate-cause row, so it does not belong beside `UnresolvedDemand` in `opm/errors`. Decoding follows the existing untagged-field convention: `componentOmitted` and `definedBy` bind case-insensitively, as `definedBy` already does on `UnresolvedDemand`.

**Switch plumbing:** `GlueInputs` gains `SkipUnprovided bool`, rendered into a new slot as the literal `true` or `false` (`_skipUnprovided: <<.SkipUnprovided>>`, `strconv.FormatBool`). `renderstage.Stage` gains the switch. It already takes `localReplacements bool` positionally, so the implementer MAY fold both into a small options struct rather than add a second positional bool. The package is internal, so its shape is not public surface.

### Message suffix for an unprovided row

`UnresolvedDemand.describe()` keeps every existing tail byte for byte, so existing `Contains` assertions in this repo and downstream stay true. For an unprovided row it appends `; provider-fulfilled, no provider on this platform` after the case text and before `; <n> candidate(s) disqualified`:

```text
component "db": unresolved trait demand "<cat>/traits/snapshot@v1": defined by "<cat>@v0" and nothing on this platform implements it; provider-fulfilled, no provider on this platform
```

### Fixtures: provider-fulfilled members in the render fixture catalog

Three members are added to `testdata/render/registry/testing.opmodel.dev_library-render_cat_v0.1.0/catalog.cue`, each listed in the catalog's contract maps:

- `#SnapshotTrait`: `fulfilment: "provider"`, `optional: bool | *false`, applies to the container resource. No transformer requires it, so it is unprovided and carries a defining catalog. This is the shape of the real `backup` trait.
- `#LedgerResource`: `fulfilment: "provider"`. No transformer requires it, so it is an unprovided resource.
- `#ArchiveTrait`: `fulfilment: "provider"`, `optional: bool | *false`, required by a new `archive-transformer` that also requires a label (`render.test/archive: "on"`). A component attaching it without the label has a provider that exists and does not match.

New scenario packages under `testdata/render/scenarios`:

- `unprovided`: component `app` (container plus snapshot) and component `ledger` (container plus ledger), beside a healthy component.
- `provided_unmatched`: archive attached without the label.

Existing scenarios and their assertions stay as they are. The scenarios README gains the rows.

**Alternative:** author the members inline in the scenario, as `unlisted` does. Rejected for the main case: inline contracts carry no defining catalog, while the real case (catalog_opm lists `backup` and ships no transformer) always does. The catalog route tests the `definedBy` copy onto skipped rows.

## Research & Decisions

### Where the skip decision lives

**Context**: the switch changes which demands refuse, and the refusal is decided in CUE (`gate`) and again in Go (`gateErrors`).
**Explored**: `render.cue.tmpl` (`#Match`, `guard`, `diagnostics`, `gate`), `render_decode.go` (`decodeRenderDiagnostics`, `gateErrors`), the spec requirement "The render module's own gate agrees with the kernel's refusal".
**Decision**: decide in the glue, as a verdict. The Go side only decodes the new rows; `gateErrors` is unchanged because `unresolved` and `unmatched` arrive already filtered.
**Rationale**: one decision site keeps the self-refusing module and the kernel in agreement. Filtering in Go would need a second implementation of "omitted component" and would still execute the omitted pairs.

### What counts as zero providers

**Context**: core's rule says zero providers is the unresolved case, and the switch applies to exactly that case.
**Explored**: `guard._providerSuppliers` (required maps only, provider fulfilment only, enabled entries only); core `SPEC.md` §2.1 ("A transformer naming the contract among its optional demands MUST NOT count as a provider"); `_optionalCovered` (an optional listing by a matched transformer already handles a trait).
**Decision**: unprovided means that `guard._providerSuppliers` has no entry for the key and the attached primitive declares `fulfilment: "provider"`.
**Rationale**: it reuses the count that already defines over-subscription, so "one provider", "two providers" and "zero providers" are three readings of one map. A provider that exists but is disqualified is excluded by construction.

### Where the new row type lives

**Context**: `opm/errors` holds the gate-cause rows. `opm/kernel` holds the advisory rows (`ResolvedVersion`, `Replacement`).
**Decision**: `kernel.SkippedDemand`, plus one field on `oerrors.UnresolvedDemand`.
**Rationale**: a skipped demand never refuses, so it is an advisory row. `UnresolvedDemand` gains the fact a frontend needs to hint the switch without re-deriving fulfilment.

## Risks / Trade-offs

- [An unprovided demand is skipped although a provider is installed on the real cluster, because the platform the caller rendered against does not carry it (for example a module-deps platform)] → That is the switch's purpose. The frontend owns the choice to set it and must word every skipped row. The kernel never sets it.
- [A new field on a row a downstream test compares by value (`assert.Equal` on a whole `UnresolvedDemand`) fails after the bump] → Checked in tasks section 3 by building and running the cli and opm-operator unit tests against the branch. Such a test gains the field in the consumer's own bump PR.
- [The `fulfilment` default does not resolve inside the marker's `if` guard the way it does in the posture guards] → The spike measures it before any decode work lands.
- [Warnings dropped for an omitted component hide an advisory fact] → The component rendered nothing, and its skipped row already says so. Recorded here so the choice is visible.

## Migration Plan

Additive and pre-GA, so no migration fragment is written. Consumers pick it up with their next library bump; nothing changes for a caller that does not set the switch. Rollback is reverting the change: no persisted state and no published schema depend on it.
