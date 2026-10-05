## Why

The cli and the operator each carry their own inventory code. Each has an entry type, an entry
constructor from a live object, a stale-set function, an inventory digest and a render digest.
The beta-1 kernel-plan walkthrough (owner, 2026-10-03, task e3) decided where that code goes:

> opm/k8s/inventory holds one Entry, one stale-set function, the render and inventory digests.
> Inventory digest hashes a NEW canonical field-by-field encoding (independent of JSON tags);
> both frontends change their stored digest once, with a migration note. Close 0012:OQ7 as
> component-blind (no behaviour change; drop the CLI rename filter).

Enhancement 0012 records this as 0012:D7 (stale set and inventory digest) and 0012:D6 (render
digest). This change is the library half. It adds the package. The frontends adopt it in their
own changes.

Three defects in today's copies show why one definition is needed. Checked at cli `origin/main`
`bd4d1a7c` and opm-operator `origin/main` `dd0d798`:

- **The two inventory digests disagree today.** Both sort the entries and hash
  `json.Marshal` of their own entry struct (cli `pkg/inventory/entry.go` `ComputeDigest`,
  operator `internal/inventory/digest.go` `ComputeDigest`). The structs carry different JSON
  tags. The cli writes `group` and `namespace` even when empty. The operator's CRD type omits
  them. So a core-group or cluster-scoped entry hashes differently in each frontend (0012:D1:R1
  is not met). The hash also depends on tags, which have nothing to do with the inventory.
- **The two render digests can never agree.** Both hash each object's CUE-order JSON, the
  managed-by label value included (cli `internal/inventory/digest.go`, operator
  `internal/status/digests.go` `RenderDigestJSON`). The cli stamps `opm-cli` and the operator
  stamps `opm-controller`, so one render gives two digests (0012:D6:R2 is not met). A comment
  asks maintainers to keep the two copies in sync by hand.
- **Two stale-set rules give one outcome.** The cli compares entries component-aware and then
  rescues renamed objects with `ApplyComponentRenameSafetyCheck` (cli
  `internal/inventory/stale.go`, called from `internal/workflow/apply/apply.go`
  `ComputeStaleInventorySet`). The operator compares component-blind
  (`internal/inventory/stale.go`). Both return the previous entries whose group, kind,
  namespace and name are absent from the current set, in previous order. 0012:D7 keeps the
  component-blind rule and drops the filter, and no behaviour changes.

## What Changes

- **`opm/k8s/inventory`** (new; imports the standard library, `k8s.io/apimachinery` and
  `opm/k8s/object` and `opm/k8s/labels`):
  - `Entry`: a plain struct with `Group`, `Kind`, `Namespace`, `Name`, `Version` and
    `Component`. It has no struct tags. The library never owns the wire shape: each frontend
    keeps mapping to its CRD or record fields.
  - `NewEntry(*unstructured.Unstructured) Entry` reads group, version and kind, namespace,
    name, and the `component.opmodel.dev/name` label (`labels.ComponentName`).
  - `SameObject(a, b Entry) bool` compares group, kind, namespace and name only. It ignores
    component and version.
  - `StaleSet(previous, current []Entry) []Entry` returns the previous entries that no
    current entry is the same object as. It keeps previous order and returns a non-nil empty
    slice when nothing is stale. This is the operator's rule and the cli's outcome
    (0012:D7:R1).
  - `Digest([]Entry) string` is the inventory digest, `sha256:<hex>`. It hashes a versioned
    canonical encoding. The encoding is a tag line, then each entry, sorted, with each field
    length-prefixed in a fixed order. It depends only on field values and not on input order
    or on any serialisation (0012:D7:R2/R3). The name drops the stutter of the walkthrough's
    working name `InventoryDigest`.
  - `RenderDigest([]object.Exported) (string, error)` is the render digest, `sha256:<hex>`. It
    takes the one export `opm/k8s/object.Export` already makes, so it reads no CUE value. A
    caller can drop its Resources first, and the cli no longer exports twice. It decodes each
    object's JSON again, keeping number literals. It blanks only the value of the managed-by
    label. It hashes a sorted-key encoding of each object in group, kind, namespace and name
    order, after a versioned tag line (0012:D6:R2/R3).
  - Golden tests pin both encodings. The expected bytes are spelled out in the test, so a
    change of encoding fails as a change of definition and not only as a changed hex string.
- **`opm/kernel` test**: a render of one fixture instance under `opm-cli` and under
  `opm-controller` yields objects that are equal except for the managed-by label value. With
  the tier test that the render digest ignores exactly that value, this proves the cross-runtime
  parity of 0012:D6:R2. The fence keeps the end-to-end render out of the tier's own tests (the
  tier may not import `opm/internal/registrytest`, and the kernel may not import the tier).
- **`opm/k8s/labels`**: the `ComponentName` doc comment stops saying inventory reads the
  label "to keep a component rename safe". Inventory records the label as provenance, and the
  stale set ignores it (0012:D7). Doc comment only.
- **Docs**: `README.md`, `AGENTS.md` and `CONSTITUTION.md` list `inventory` among the tier's
  packages. ADR-011's Status records the package.
- **Specs**: `kubernetes-tier` gains four requirements: entries, the component-blind stale
  set, the canonical inventory digest and the shared render digest.

Not **BREAKING**. Every symbol is new, and nothing is removed, renamed or changed in behaviour.
SemVer class: MINOR. Release class of the PR: `feat`.

The observable change comes later, in each frontend's adoption change. Both frontends'
`status.inventory.digest` and render digests (`status.lastAppliedRenderDigest`, and the
operator's `lastAttemptedRenderDigest`) change once, when each first records the new values.
The operator's no-op check then fails once per instance, so every instance applies once
after the upgrade. Those changes carry the migration note (0012:D7:R4), as a `feat!` with a
`BREAKING CHANGE:` footer in the squash body.

## Not in this change

- **Frontend adoption** (op-e3 and cli-e3). The frontends delete `ComputeDigest`,
  `ComputeStaleSet`, `IdentityEqual`, `K8sIdentityEqual`, `NewEntryFromResource`, the render
  digest copies and the cli's `ApplyComponentRenameSafetyCheck` together with its call and the
  `component rename detected` debug line. Each writes the new digests and ships the
  migration note.
- **The wire shape.** `Entry` has no tags. The CRD `InventoryEntry` type and the cli's
  record mapping stay in the frontends (0012:OQ9 is still open).
- **Prune safety exclusions, ownership verdicts and the adopt annotation** (lib-e4), **the
  deletion protocol** (lib-f2) and **health** (lib-f5).
- **An identity key type or admit set.** The cli's `K8sIdentity` and `AdmitSet` serve its
  install path. That path is ownership work, and lib-e4 decides it.
- **A compatibility path for old digest values.** 0012:D7 rejects it.
- **Closing 0012:OQ7 in the enhancement.** 0012:D7 already resolved it in enhancements#88.
  This change implements the rule and touches no enhancement file.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `kubernetes-tier`: inventory entries built from objects, the component-blind stale set,
  the canonical inventory digest, and the render digest that both runtimes compute equally.

## Impact

- Packages: new `opm/k8s/inventory`. `opm/k8s/labels` changes one doc comment. One new test
  file under `opm/kernel`. No kernel code changes.
- Repo files: `README.md`, `AGENTS.md`, `CONSTITUTION.md`,
  `adr/011-kubernetes-tier-beside-the-kernel.md`. No change to `go.mod` (apimachinery is
  already required), to the lint configuration or to the depguard rules. The package fits the
  existing tier allow list.
- Downstream: cli and opm-operator compile unchanged against this tree, and the
  consumer-build job must stay green. Their adoption changes gate on the first library
  release that contains this change. That release also carries lib-e4, so each frontend
  changes its stored digests and its guards in one release.
- Mixed-version window: until both frontends record the new digests, a cli-to-operator
  handoff compares an old-format digest with a new-format one. Nothing compares the two across
  frontends today, so this is a note for the adoption changes. It is not a defect of this one.
- Parallel work: lib-e4 and lib-f5 are written beside this change and edit the same docs
  lines (the tier's package list in `README.md`, `AGENTS.md` and `CONSTITUTION.md`, and the
  ADR-011 Status). The second of them to merge rebases its docs section. lib-e4 also edits
  `opm/k8s/labels/labels.go` (a new `AnnotationAdopt` block) where this change edits the
  `ComponentName` doc comment; the hunks are far apart and are expected to merge on their
  own.
- `enhancement.yaml` declares 0012 and claims no decision. 0012:D6 and 0012:D7 are delivered
  only once both frontends compute through this package (0012:D1, ADR-011 item 3). The claim
  belongs to the adoption changes, and under-claiming is the safe direction.
