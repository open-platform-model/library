# Design: render-duplicate-identities

## Context

See `proposal.md` § Why. The decision is `enhancements/0015` D15 (one registration per module, refused before apply naming both carrying components) with D12 (the registration's instance-derived name, so two registrations collide on one object). The refusal site was left to the slice ("module vet or render-output validation"); this change picks render output, the one place every runtime passes through with provenance in hand.

Current state, read 2026-09-19 (library `v1.0.0-alpha.32`):

- `kernel.Compiled{Value, Instance, Component, Transformer}`: "carries no platform-native fields — keeping platform vocabulary out of the kernel keeps it platform-neutral, and each consumer wraps *Compiled in its own resource type" (`opm/kernel/render.go`). `Render` decodes each pair's output and dispatches on struct versus list; it never reads `kind` or `metadata`.
- The render glue's fail-closed gate (`gateErrors`) raises three typed causes from CUE-owned verdicts (`unresolved`, `unmatched`, `overSubscribed`); none concerns rendered object identity.
- `opm/helper/` holds one subpackage, `platformmodule`; the depguard rule in `.golangci.yml` forbids any package outside `opm/helper/` from importing under it, and `helper-packages` fixes the exported non-helper packages to five.
- The cli wraps each `Compiled` into `pkgcore.Resource` (`internal/workflow/render/render.go`); the operator into `core.Resource` and then one inventory entry per object (`internal/render/module.go`, `buildInventoryEntries`). Neither dedupes; Flux `ApplyAll` and the cli's apply write each object in turn, so the last identical identity wins.
- `opm/module` has no need of this; `testdata/render` fixtures render distinct identities today, and no fixture renders two objects with one name.

## Goals / Non-Goals

**Goals**

- One detector, one wording, called by both runtimes between render and apply.
- Rows carry provenance a module author can act on: which two components, through which transformers.
- The kernel stays platform-neutral; the helper boundary stays real in the import graph.

**Non-Goals**

- Any refusal inside `Render`, or a CUE-side duplicate verdict in the glue.
- Validating rendered objects beyond reading four identity fields.
- Arbitration or deduplication: a duplicate is refused, never merged.
- The runtimes' calls: cli and operator changes.

## Decisions

### The package

`opm/helper/objectset`:

```go
// Identity is a rendered object's apply identity. Namespace is empty for a
// cluster-scoped object or one that names none.
type Identity struct {
	APIVersion string
	Kind       string
	Namespace  string
	Name       string
}

// String renders "apps/v1 Deployment web-system/web" (no namespace: "apps/v1 Deployment web").
func (i Identity) String() string

// Producer is the (component, transformer) the kernel recorded on the object.
type Producer struct {
	Component   string
	Transformer string
}

// Duplicate is one identity two or more rendered objects share.
type Duplicate struct {
	Identity  Identity
	Producers []Producer // render order
}

// Duplicates scans a render's objects and returns every shared identity, in
// first-seen order. A value with no kind or no metadata.name is skipped.
func Duplicates(compiled []*kernel.Compiled) []Duplicate

// DuplicateIdentitiesError is the refusal a runtime raises from the rows,
// before apply. Error() names each identity once and every producer.
type DuplicateIdentitiesError struct {
	Duplicates []Duplicate
}
```

`Duplicates` reads `apiVersion`, `kind`, `metadata.namespace` and `metadata.name` through `Value.LookupPath` and `String()`; a lookup that does not exist or is not a string leaves the field empty, and an empty `Kind` or `Name` skips the object. The map from identity to producers is keyed by the `Identity` struct; the first-seen order is kept in a slice so the output does not depend on map iteration.

Message shape:

```
2 rendered objects share one identity, so the last apply would silently overwrite the first:
  opmodel.dev/v1alpha1 TransformerRegistration backup-system.k8up rendered by component "registration" (…/transformer-registration-transformer@4.4.0) and component "registration-copy" (…/transformer-registration-transformer@4.4.0)
```

### Where it runs

Between render and apply in each runtime, on the `[]*kernel.Compiled` the kernel returned, before any wrapping that would lose the two provenance fields. The cli calls it in `internal/workflow/render` so `build` refuses too (the same objects would be written by `apply`); the operator calls it in `resultFromRender` before `buildInventoryEntries`. Both are their own changes; this one ships the function they call.

### Files touched

| File | Change |
| --- | --- |
| `opm/helper/objectset/doc.go`, `objectset.go`, `objectset_test.go` | new |
| `opm/helper/doc.go` | the subpackage list |
| `CLAUDE.md` § Repository Layout | the subpackage list |
| `opm/kernel/flow_integration_test.go` | no-duplicate assertion on the shipped fixture |

## Research & Decisions

### Helper, not kernel

**Context**: D15 wants the refusal before apply; the kernel is the one place every render passes through, but `Compiled` is deliberately platform-neutral.
**Explored**:
1. `Render` refuses duplicates: one site, but it reads Kubernetes identity fields inside the kernel and returns a platform-specific error from a neutral verb, which Principle I and the `Compiled` contract both forbid.
2. A CUE verdict in the render glue (`diagnostics.duplicates`), decoded like `overSubscribed`: keeps Go neutral but moves the same vocabulary into the glue, which is still the kernel.
3. An opt-in helper both runtimes call: the vocabulary lives where a Kubernetes-applying frontend opts in, the kernel stays neutral, and there is still one implementation and one wording.
4. A copy in each runtime: two implementations, two wordings, and the operator's and the cli's refusals drift.
**Decision**: 3.
**Rationale**: `helper-packages` exists for exactly this shape ("opinionated convenience that wraps kernel primitives for a specific embedding pattern"), and the boundary is lint-enforced, so the kernel cannot come to depend on it. 4 is what the helper tier was created to avoid.

### Skip, do not refuse, a value without an identity

**Context**: transformers may render values that are not Kubernetes objects (a platform of another type, or a fixture).
**Explored**: refusing any object without `kind` and `metadata.name`; skipping it.
**Decision**: skip.
**Rationale**: the helper's contract is duplicate detection, not shape validation; a runtime that requires every object to be a Kubernetes manifest already fails at conversion (`ToUnstructured`) with a better message, and refusing here would make the helper assert something the kernel deliberately does not.

### Render order, not sorted

**Context**: rows must be deterministic so an operator's transition-gated events fire once per distinct refusal.
**Explored**: sorting rows by identity string; keeping first-seen render order.
**Decision**: first-seen render order, producers in render order.
**Rationale**: the render's pair order is already deterministic (the build's), so the output is deterministic without a sort, and the first producer named is the one that rendered first, which is the natural reading of "the second overwrites the first".

### No fixture module for the collision

**Context**: an end-to-end fixture (two components carrying the registration resource) would need catalog_opm's registration contract in the served render registry.
**Explored**: adding such a module to `testdata/render`; unit tests over hand-built `Compiled` values with `cuecontext` plus a no-duplicate assertion on the shipped fixture.
**Decision**: unit tests plus the flow assertion.
**Rationale**: the helper takes `[]*kernel.Compiled` and reads four fields; every scenario in the spec is a value-level case, and the healthy path against the published catalog is what the flow test already renders. The registration-specific end-to-end case belongs with the operator change that refuses it, where the fixture pipeline already publishes provider modules.

## Risks / Trade-offs

- [A runtime forgets to call the helper] → the two follow-up changes are named in the proposal and the operator's claims D15; until they land nothing changes, which is today's behaviour.
- [A transformer legitimately renders two values with one identity] → it does not: two writes to one object is an authoring error in every runtime; the message says which components.
- [`Identity` as a map key with an unset namespace] → a namespaced object rendered without `metadata.namespace` and a cluster-scoped object of the same kind and name compare equal, which is also how apply would treat them once the namespace defaults; a false positive there names both producers and is reviewable.

## Migration Plan

One PR, two sections, squash title `feat(helper): detect duplicate rendered object identities`. release-please cuts `v1.0.0-alpha.33`. Rollback is a revert; nothing calls the package until the runtime changes land.

## Open Questions

None.
