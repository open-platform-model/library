## Context

See proposal.md, Why. Design-local decisions are numbered KI1 to KI8 so they collide with no
other numbering. Line references are at library `origin/main` `ca7c56b` (after
add-kubernetes-object-packages, library#196), cli `origin/main` `bd4d1a7c` and opm-operator
`origin/main` `dd0d798`, all fetched 2026-10-05. Evidence comes from the wave-2 research entry
e3 (`claude-stuff/kernel-plan-beta1/wave2-plan-result.json`), re-checked at those heads. Since
that research, the cli's `pkg/inventory/entry.go` has gained `K8sIdentity`, `IdentityOf` and
`AdmitSet` (the operator-install admission). They are not part of this change (proposal, Not in
this change). The digest and stale-set code it describes is unchanged.

## Goals / Non-Goals

**Goals:**
- One inventory entry type and one constructor from a live or rendered object.
- One component-blind stale-set function.
- One inventory digest over a canonical encoding that no serialiser can move.
- One render digest that both runtimes compute equally for one render, and that moves on any
  other change.
- Encodings pinned by golden tests that spell out the bytes.

**Non-Goals:**
- Any frontend edit, or the migration note itself (the adoption changes carry it).
- A wire type for the CRD or the cli record.
- Ownership, prune safety, deletion or health.

## Research & Decisions

### KI1: The package surface is six symbols

**Context**: The owner decision names "one Entry, one stale-set function, the render and
inventory digests". The frontends today also export a component-aware `IdentityEqual`, a
component-blind `K8sIdentityEqual` and an entry constructor.

**Explored**: The importers of each frontend symbol at the heads above. The cli uses
`IdentityEqual` only inside `ComputeStaleSet`. It uses `K8sIdentityEqual` in the rename filter
and in tests. The operator uses `K8sIdentityEqual` inside `ComputeStaleSet`, and in its
`IdentityEqual` doc as the rule to prefer. Both build entries from `*unstructured.Unstructured`
(cli `apply.go` `NewEntryFromResource` loop; operator `internal/render/module.go`).

**Decision**: `opm/k8s/inventory` exports exactly `Entry`, `NewEntry`, `SameObject`,
`StaleSet`, `Digest` and `RenderDigest`. No component-aware comparison is exported, because
0012:D7 leaves no rule that needs one. No list helper (`Entries`) is added: a frontend's loop
over `[]object.Exported` is three lines, and the frontends keep their own mapping to the wire
anyway. The inventory digest is `Digest` and not `InventoryDigest`, because
`inventory.InventoryDigest` stutters. `RenderDigest` keeps its name, since `inventory.Digest`
would not say which digest it is.

A test lists the package's exported identifiers and pins these six, so a later exported
comparison fails a test and not only review.

`ownership.Object` (add-kubernetes-ownership-package, written beside this change) is the
tier's exported four-field identity value. This change keeps its own key unexported, and a
later change may express `SameObject` over `ownership.Object`. No API change here.

**Rationale**: Principle VII. Each symbol has a consumer in both frontends' adoption changes.
`SameObject` is the stale set's relation, made public so a frontend that needs "is this the
same object" (the cli's prune and existence checks) asks the same question.

### KI2: Entry is a plain value with no tags

**Context**: The two frontends' entry structs differ only in JSON tags (cli
`pkg/inventory/types.go`: `group`, `namespace` without `omitempty`; operator
`api/v1alpha1/common_types.go`: with `omitempty`). That difference is what makes their digests
disagree.

**Decision**:

```go
// Entry is one object an instance owns, as the tier compares and digests it.
type Entry struct {
    Group, Kind, Namespace, Name string // identity (SameObject)
    Version   string // API version; recorded, never part of identity
    Component string // component.opmodel.dev/name at apply; provenance, never identity
}

// NewEntry reads obj's group, version, kind, namespace, name and its
// labels.ComponentName label. obj MUST NOT be nil.
func NewEntry(obj *unstructured.Unstructured) Entry
```

`Entry` carries no struct tags. A test asserts that with `reflect`, so a later edit that adds
JSON tags (suggesting the library owns the wire) is caught. `NewEntry` reads the same fields as
both frontends' `NewEntryFromResource`, and a missing label gives an empty `Component`.

**Rationale**: 0012:D7:R2. The digest must not depend on serialisation, and the library must
not own the CRD wire while 0012:OQ9 is open. With no tags there is nothing to depend on.

### KI3: StaleSet is component-blind and keeps previous order

**Context**: 0012:D7:R1. The cli computes stale entries component-aware and then removes every
stale entry for which some current entry has the same group, kind, namespace and name under
another component. The operator computes it component-blind in one step.

**Explored**: Whether the two outcomes are equal. Take a previous entry `p`. The cli keeps `p`
if no current entry matches it on the four identity fields and component, and if no current
entry matches the four fields with a different component. Together that means no current entry
matches the four fields at all. That is the operator's condition. Both iterate `previous` in
order, so the order is equal too. The outcome is equal, and dropping the filter changes no
behaviour, as 0012:D7 states.

**Decision**:

```go
func SameObject(a, b Entry) bool // Group, Kind, Namespace, Name equal

// StaleSet returns, in previous order, every previous entry no current entry
// is the SameObject as. It returns a non-nil empty slice when none is stale,
// including when previous is empty. It never changes its inputs.
func StaleSet(previous, current []Entry) []Entry
```

The implementation indexes `current` in a map keyed by the four identity fields, so it costs
O(n+m) where both copies cost O(n·m). It iterates `previous`, so map order never reaches the
output. Duplicate previous entries are each returned.

**Rationale**: One rule, the one that needs no rescue filter (0012:D7). The non-nil empty
result matches both frontends, whose callers range over it and log its length.

### KI4: The inventory digest hashes a versioned, length-prefixed field encoding

**Context**: 0012:D7:R2/R3 and the owner's "NEW canonical field-by-field encoding (independent
of JSON tags)".

**Decision**: `Digest(entries)` returns `"sha256:" + lowercase hex` of SHA-256 over these
bytes:

```text
"opm-inventory-v1\n"
for each entry, in sorted order:
    for each field in order Group, Kind, Namespace, Name, Version, Component:
        uint64 big-endian byte length of the field (8 bytes)
        the field's bytes
```

The sort order is Group, Kind, Namespace, Name, Component, Version, compared as byte strings.
That is the frontends' current order, and it uses every field, so equal keys mean equal
entries and the output does not depend on input order. Entries are not de-duplicated: the
digest is over the multiset, and the frontends never record duplicates (`object.Duplicates`
refuses them before apply). An empty or nil inventory hashes the tag line alone. The function
copies before sorting and never changes its input. The code writes each field explicitly, with
no `fmt` of structs, no map iteration and no JSON, so neither Go releases nor struct edits move
the value.

**Explored**: Delimiters instead of lengths (NUL bytes, newlines). Rejected: Kubernetes names
cannot hold NUL, but a `Component` label value is not a name, and length prefixes need no
argument about what a field may contain. A count prefix is not needed, because fixed six-field
records with length prefixes already parse unambiguously. The tag line makes a future encoding
a new tag, and that change would be a stored-digest change with its own migration note.

**Rationale**: The encoding is defined field by field, so the digest is a property of the
definition and not of an encoder (0012:D7, Alternatives).

### KI5: The render digest re-decodes the export, blanks one value and hashes sorted-key JSON

**Context**: 0012:D6:R2/R3. The input has to be the one export `object.Export` makes. Its doc
comment (`opm/k8s/object/export.go`) states the rule: "The caller drops its Resources after
the export to release the CUE build they pin." The operator renders inside a bounded render
slot and drops the CUE values when it leaves it.

**Decision**:

```go
// RenderDigest returns "sha256:<hex>" over the canonical form of objs.
func RenderDigest(objs []object.Exported) (string, error)
```

For each `Exported` it:

1. Decodes `JSON` again with `json.Decoder.UseNumber`, so every number keeps the literal CUE
   emitted. `Exported.Object` is not reused: it was decoded into `float64`, which merges
   integers above 2^53 and would let two different renders share a digest (R3). Trailing data
   after the object, a non-object value (the literal `null` included, which decodes to a nil
   map without an error) and invalid JSON are errors naming the object's index.
   `object.Export` never produces them, but a hand-built `Exported` can.
2. If `metadata` is an object, `metadata.labels` is an object and it holds the key
   `app.kubernetes.io/managed-by` (`labels.ManagedBy`), it replaces that value with the empty
   string. The key stays, so an object with the label and one without it still differ. Nothing
   else is touched.
3. Encodes the result with `encoding/json` (`SetEscapeHTML(false)`). Object keys come out
   sorted, numbers keep their literals, and each object ends with the encoder's newline, which
   also separates the objects in the hash.

It then sorts the encoded objects by group (from `apiVersion`, empty for the core group), kind,
`metadata.namespace` and `metadata.name`, read from the decoded object, with the encoded bytes
as the final tie-break. A sort field that is missing or not a string reads as `""`; the byte
tie-break keeps the order deterministic then too. It hashes `"opm-render-v1\n"` followed by each encoded object. An empty
render hashes the tag line alone. It never changes `objs`, its `JSON` or its `Object`.

**Explored**: Hashing the CUE-order JSON as today. Rejected: the managed-by value cannot be
removed from it without decoding, and once decoded the canonical form is what we can state.
Deleting the label key. Rejected: it hides a change that adds or removes the label. Sorting
with a stable sort on input order as today. Rejected: two objects with one identity would then
hash in input order. The duplicates check refuses that render, but the digest does not need to
rely on it.

**Rationale**: Exactly one label value is ignored and every other byte of every object counts
(0012:D6:R3). The input is bytes, so the digest needs no CUE value and the caller may release
the build first.

### KI6: Cross-runtime parity is proved in two tests, one on each side of the fence

**Context**: The plan entry asks for a test that renders one instance as `opm-cli` and as
`opm-controller` and gets equal render digests. A full render needs the in-process registry
`opm/internal/registrytest`. The kubernetes-tier spec forbids any file under `opm/k8s/`, tests
included, to import `opm/internal/`. It forbids any file under `opm/kernel/`, tests included,
to import `opm/k8s/`.

**Explored**: (a) A lint exemption for one tier test file, as `objectset_parity_test.go` has
for the helper. Rejected: it widens a fence for a test that a split proves equally, and that
precedent is a deletion-bound parity shim, not a pattern. (b) A test package outside `opm/`.
Not possible: Go's internal rule keeps `opm/internal/registrytest` out of reach. (c) The split.

**Decision**: The split.
- `opm/kernel/render_runtime_name_test.go` renders the existing render fixture (the platform
  and instance `TestRender_HappyOnDiskInputs` uses) twice, with `RuntimeName` `opm-cli` and
  `opm-controller`. It exports each compiled value to JSON and decodes it. It asserts that
  each object's managed-by label value is its runtime name. It asserts that, with that one
  value blanked, the two renders are deep-equal object by object. The label key is a string
  literal in the test, since the kernel may not import `opm/k8s/labels`.
  A second case renders the shipped-catalog parity instance (`testdata/parity/instance`
  against `testdata/parity/opm_platform`, the inputs of `TestParity_ShippedCatalog`) the same
  two ways, with the same assertions and the same GHCR gating. Core passes
  `#context.#runtimeName` to every transformer (core `src/transformer.cue`), so whether the name
  reaches only the label is a property of the catalog, not of the kernel. Today
  `catalog_opm`'s transformers reach it only through `#context.labels` into `metadata.labels`;
  selectors and pod templates use the component labels.
- `opm/k8s/inventory` tests assert that `RenderDigest` is equal for two object sets that
  differ only in the managed-by value, and different for every other change (KI7).

Together: a runtime name changes only that value, and the digest ignores only that value, so
the two runtimes' digests of one render are equal (0012:D6:R2). The tier owns only the second
half. The first half holds for the catalogs the kernel test renders, and a catalog that wrote
`#runtimeName` anywhere but `#context.labels` would break it. The spec states the tier's
property and names that condition. Section 1 of tasks.md writes
the kernel test first, because the premise ("the two frontends render the same instance into
objects that differ in exactly one label value", ADR-011 Context) has not been tested before.

**Rationale**: It keeps the fence as the spec states it and still tests both halves of the
claim on real code.

### KI7: Golden tests spell out the bytes

**Decision**: For each digest, one test builds the expected input of the hash by hand, from the
KI4 or KI5 definition written out in the test (tag line, length prefixes, sorted-key JSON
literals). It compares `"sha256:" + hex(sha256(expected))` with the function's result. A second
assertion pins the hex string itself, as a committed constant. The fixtures:

- Inventory: a core-group, cluster-scoped entry with empty `Group` and `Namespace` (the case
  where the frontends disagree today), a namespaced `apps` entry, and one entry with an empty
  `Component`, given out of order.
- Render: a Deployment with a managed-by label and an integer above 2^53, a core-group
  Service, and a cluster-scoped Namespace, given out of order.

Behaviour tests, one per spec scenario: order independence, sensitivity to each single field
and to adding or removing an entry, managed-by value ignored, managed-by key presence counted,
another label counted, a large-integer change counted, the input left unchanged, and the empty
sets.

**Rationale**: A changed hex string alone says only "something moved". A spelled-out encoding
says what the definition is, and it fails when the code drifts from it. The committed constant
is the value a frontend stores, and the test comment says that changing it is a stored-digest
change that needs a new tag and a migration note.

### KI8: Docs list the package

**Decision**: Locate edits by text. `README.md` (Layout tree and the tier paragraph),
`AGENTS.md` (the rules line `opm/k8s/` (`labels`, `object`) and the layout block) and
`CONSTITUTION.md` (Principle III's tier bullet) add `inventory`. ADR-011's Status gets one
sentence: the `inventory` package has landed, with the component-blind stale set and the two
digests of 0012:D6 and 0012:D7. The `labels.ComponentName` doc comment says the label is
recorded in inventory entries as provenance and that the stale set ignores it.

## Risks / Trade-offs

- **The managed-by redaction widens no-op detection by exactly one value.** A change that only
  flips the runtime name does not move the digest. That is intended (ADR-011 Trade-off). The
  test "another label counts" pins that nothing else is ignored.
- **The render digest decodes each object a second time.** The cost is one JSON decode per
  object at digest time, on bytes the caller already holds. Rendered objects are small next to
  the CUE build that produced them, so the peak stays the CUE evaluation.
- **Digest values change once in both frontends.** This is decided (0012:D7, Alternatives).
  The adoption changes carry the migration note. The operator applies every instance once after
  its upgrade, bounded by the shared render slots.
- **Mixed-version window.** Until both frontends pin a release with this package, a handoff
  compares digests of two formats. Nothing compares them across frontends today (0006:D7's
  handoff check is not implemented).
- **Docs conflict with lib-e4 and lib-f5.** They edit the same package lists. The docs edits
  are their own section, so a rebase touches one commit.

## Migration Plan

None for this change, which is additive. The adoption changes in the frontends carry the
migration note: `status.inventory.digest` and the render digest fields change once, and the
operator applies every instance once on its first reconcile after the upgrade.

## Open Questions

None.
