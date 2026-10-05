# Tasks: add-kubernetes-inventory-package

Worktree `library/.claude/worktrees/add-kubernetes-inventory-package`, branch
`feat/add-kubernetes-inventory-package` (from `origin/main` `ca7c56b`). Seed `.cue-cache` by
copying the main checkout's (`cp -a`), never by symlinking. Every command runs inside the
worktree with an absolute private `TMPDIR` from `mktemp -d` under the session scratchpad. The
registry env goes on two lines:

```bash
export CUE_REGISTRY='opmodel.dev=ghcr.io/open-platform-model,testing.opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'
export OPM_REGISTRY="$CUE_REGISTRY"
```

`TestGenerate_BuildsThroughTheKernel` can fail in a full-suite run because of a known
cross-process cache race. Rerun `go test ./opm/helper/platformmodule -count=1` on its own before
treating it as a finding. Commit bodies never start a line with `word(` and carry no bare
at-sign. The only trailer is `Co-Authored-By: Claude <noreply@anthropic.com>`. Source citations
for the frontend code this replaces: cli `origin/main` `bd4d1a7c`, opm-operator `origin/main`
`dd0d798`.

## 1. Spike: a runtime name reaches only the managed-by value (kernel test; design KI6)

- [x] 1.1 `opm/kernel/render_runtime_name_test.go`, `TestRender_RuntimeNameReachesOnlyManagedBy`.
      Reuse the render fixture helpers of `TestRender_HappyOnDiskInputs` (`newRenderKernel`,
      `acquireRenderPlatform(t, k, "platform")`, `acquireRenderInstance(t, k, "instance")`).
      Render twice, with `RuntimeName` `opm-cli` and `opm-controller`. For each compiled
      object, `Value.MarshalJSON`, then decode to `map[string]any` with `UseNumber`. Assert:
      - both renders return the same number of objects, with equal provenance index by index;
      - each object's `metadata.labels["app.kubernetes.io/managed-by"]` is its render's
        runtime name (a literal key: the kernel may not import `opm/k8s/labels`);
      - with that one value set to `""` in both, the decoded objects are deep-equal index by
        index (kubernetes-tier scenario "The two runtimes digest one render equally", first
        half).

      A second case, `TestRender_RuntimeNameReachesOnlyManagedBy_ShippedCatalog`, makes the
      same two renders and the same assertions on the shipped-catalog parity instance and
      platform that `TestParity_ShippedCatalog` uses (`testdata/parity/instance`,
      `testdata/parity/opm_platform`), with the same gating: it skips under `-short` and when
      GHCR is unreachable. Core hands `#context.#runtimeName` to every transformer, so whether
      the name reaches anything but the label depends on the catalog; this case checks the
      catalog the frontends ship.

      The doc comment says the test is half of the cross-runtime parity proof, and the
      `opm/k8s/inventory` render-digest tests are the other half (design KI6). Verify:
      `go test ./opm/kernel -run TestRender_RuntimeNameReachesOnlyManagedBy -count=1` green.
      Negative check, not committed: render the second time with `opm-cli` instead of
      `opm-controller` and confirm that the managed-by assertion fails.
- [x] 1.2 If 1.1 shows that the runtime name reaches anything beyond the managed-by value,
      stop. Write the finding into design.md KI5/KI6 and report it, because 0012:D6's premise
      would then be false. Otherwise add one sentence under KI6: "Spike result: confirmed at
      `<commit>`".
- [x] 1.3 `task check` green, then commit
      `test(kernel): pin that the runtime name reaches only the managed-by label`.

## 2. Entries and the component-blind stale set (opm/k8s/inventory, opm/k8s/labels; design KI1, KI2, KI3)

- [x] 2.1 `opm/k8s/inventory/doc.go`: the package's place in the tier (ADR-011). It covers what
      it holds (entries, the stale set, the two digests) and that it owns no wire shape:
      frontends map `Entry` to their CRD or record. It also says that the stale set is
      component-blind (0012:D7, cited once at the package) and that both digests are stored
      values, whose encodings change only under a new tag line with a migration note in each
      frontend.
- [x] 2.2 `opm/k8s/inventory/entry.go`: `Entry`, `NewEntry`, `SameObject` per KI2 and KI1.
      `stale.go`: `StaleSet` per KI3, with a map keyed by an unexported identity struct and
      iteration over `previous`. Verify: `go vet ./opm/k8s/...` clean; `task lint` green (the
      existing tier allow list admits the imports with no rule change).
- [x] 2.3 `opm/k8s/labels/labels.go`: the `ComponentName` doc comment's last sentence becomes
      "Inventory records it on each entry as provenance; the stale set ignores it." Doc
      comment only.
- [x] 2.4 Tests:
      - `entry_test.go`: the scenarios "An entry reads the object's identity and component",
        "A core-group, cluster-scoped object without the label" and "The entry type carries no
        tags" (reflect: six string fields, empty `Tag`); a `SameObject` table showing that
        component and version are ignored and that each of group, kind, namespace and name
        counts.
      - `stale_test.go`: the scenarios "A component rename is not stale", "An API version
        change is not stale", "Removed objects are stale in previous order" and "Nothing stale
        is an empty list, not nil" (`NotNil` and `Len 0` for an empty previous, a nil previous
        and a fully kept previous); duplicate previous entries each returned; inputs
        unchanged. One table case ports the cli's
        `ComputeStaleSet` + `ApplyComponentRenameSafetyCheck` composition as a local oracle
        over a mixed fixture (renames, version moves, removals) and asserts `StaleSet` equals
        it. This is the "no behaviour change" claim of 0012:D7.
      - `surface_test.go`: the scenario "The package exports no other comparison". It parses
        the package's non-test files with `go/parser` and asserts that the exported top-level
        identifiers are exactly `Entry`, `NewEntry`, `SameObject`, `StaleSet`, `Digest` and
        `RenderDigest` (the last two are listed only from section 4 on, so the test grows
        with the package).

      Verify: `go test ./opm/k8s/inventory -count=1` green.
- [x] 2.5 `task check` green, then commit
      `feat(k8s): add opm/k8s/inventory entries and the component-blind stale set`.

## 3. The canonical inventory digest (opm/k8s/inventory; design KI4, KI7)

- [x] 3.1 `opm/k8s/inventory/digest.go`: `Digest` per KI4: copy, sort with
      `slices.SortFunc` and `cmp.Compare` on the six keys, write the tag line and the
      length-prefixed fields into a `sha256` hash with `binary.BigEndian.AppendUint64` and
      explicit field writes, and return `"sha256:" + hex.EncodeToString`. No `fmt` of
      structs, no JSON, no map. Verify: `go vet ./opm/k8s/...` clean.
- [x] 3.2 `digest_test.go`:
      - the scenario "The encoding is the one defined": hand-written expected bytes for the
        KI7 inventory fixture, built in the test with its own helpers, and a committed golden
        hex constant whose comment says that changing it changes every frontend's stored
        digest and needs a new tag line and a migration note;
      - "Input order does not matter" (every permutation of the fixture);
      - "Every field counts" (a table that edits each of the six fields once, plus one entry
        added, one removed, and one row where a byte moves across a field boundary: group
        `ab` and kind `` against group `a` and kind `b`, the case the length prefixes exist
        for);
      - "Empty and nil inventories agree";
      - input unchanged after the call.

      Verify: `go test ./opm/k8s/inventory -count=1` green. Negative check, not committed:
      swapping `Version` and `Component` in the write order fails the encoding test.
- [x] 3.3 `task check` green, then commit `feat(k8s): add the canonical inventory digest`.
      The body says that the digest replaces both frontends' `ComputeDigest` when they adopt
      it, and that their stored values change once then (0012:D7:R4).

## 4. The shared render digest (opm/k8s/inventory; design KI5, KI7)

- [x] 4.1 `opm/k8s/inventory/render_digest.go`: `RenderDigest` per KI5. It decodes with
      `json.Decoder` + `UseNumber` and refuses trailing data, blanks the managed-by value
      through `labels.ManagedBy`, and encodes through `json.Encoder` with
      `SetEscapeHTML(false)` into a buffer per object. The sort keys are read from the decoded
      map (group split from `apiVersion` at the last `/`), with `bytes.Compare` as the
      tie-break. It hashes the tag line and each buffer. Errors read
      `render digest: object <i>: <cause>`; a decode that yields a nil map (the literal
      `null`) is an error too. A missing or non-string sort field reads as `""`. It never
      writes to `Exported.JSON` or
      `Exported.Object`. Verify: `go vet ./opm/k8s/...` clean.
- [x] 4.2 `render_digest_test.go`, with fixtures built through `object.Export` from CUE
      literals (as `opm/k8s/object/export_test.go` does), so the digest is tested on real
      export bytes:
      - "The two runtimes digest one render equally", second half: one set rendered with
        `managed-by: "opm-cli"`, the other with `"opm-controller"`, equal digests; the test
        comment points to the kernel half (section 1);
      - "Any other label counts"; "Adding or removing the managed-by label counts"; "Large
        integers are not rounded" (`9007199254740993` against `9007199254740992`);
      - "The encoding is the one defined": hand-written sorted-key JSON lines for the KI7
        render fixture and a committed golden hex constant, with the same comment as in 3.2;
      - "The input is not changed" (deep-equal copies of `JSON` and `Object` before and
        after);
      - "A malformed object fails with its position" (a list, `null` and invalid JSON at
        index 1, built as hand-made `Exported` values);
      - input order does not matter; the empty and nil sets both hash the tag line alone.

      Verify: `go test ./opm/k8s/inventory -count=1` green. Negative check, not committed:
      deleting the managed-by key instead of blanking its value fails "Adding or removing the
      managed-by label counts"; reusing `Exported.Object` instead of re-decoding fails "Large
      integers are not rounded".
- [x] 4.3 `task check` green, then commit `feat(k8s): add the shared render digest`. The body
      says that the managed-by value is the one byte range ignored (0012:D6), and that both
      frontends' stored render digests change once when they adopt it.

## 5. Docs say what exists, and whole-tree checks (README, AGENTS, CONSTITUTION, ADR-011; design KI8)

- [x] 5.1 Locate every edit by text, not by line number:
      - `README.md`: the Layout tree gains a `k8s/inventory/` row; the Helper boundary
        paragraph's "Its packages are ..." sentence names `opm/k8s/inventory` (entries, the
        component-blind stale set and the inventory and render digests).
      - `AGENTS.md`: the rules line `opm/k8s/` (`labels`, `object`) gains `inventory`; the
        layout block gains a `inventory/` row under `k8s/` (Entry + NewEntry, SameObject,
        StaleSet component-blind, Digest canonical v1 encoding, RenderDigest over
        object.Exported with the managed-by value blanked).
      - `CONSTITUTION.md` Principle III: "Its packages today are ..." names
        `opm/k8s/inventory`.
      - `adr/011-kubernetes-tier-beside-the-kernel.md` Status: one sentence, "Amended
        <date> by `add-kubernetes-inventory-package`: `opm/k8s/inventory` holds the entry,
        the component-blind stale set and the inventory and render digests (0012:D6,
        0012:D7)." The date is the day the sentence is written; 6.1 sets it to the day of
        the archive commit, since sibling changes edit the same Status line and merge in
        an order this plan does not know.

      Verify: no file still lists the tier as `labels` and `object` alone; the layout
      block is still one code fence; `task docs:bundle:check` green.
- [x] 5.2 Whole-tree checks on the final code:
      - `task check`;
      - `go test -race ./opm/k8s/... -count=1`;
      - `task api:diff` reports only additions (the new package);
      - `openspec validate add-kubernetes-inventory-package --strict`.

      Consumer check, not committed: run `.tasks/consumer-build.sh` against fresh clones of
      cli and opm-operator at their `origin/main` (both must build and vet against this tree).
      Verify: all green.
- [x] 5.3 `task check` green, then commit
      `docs(k8s): record the opm/k8s/inventory package`.

## 6. Archive (at PR time)

- [ ] 6.1 Set the ADR-011 amendment date to today and place the Status sentence after any
      sibling amendment that merged first. `openspec archive add-kubernetes-inventory-package
      --yes` on this branch, so the archive rides the implementing PR. Verify: the main `kubernetes-tier` spec carries the
      four requirements and `openspec validate --all --strict` passes. `enhancement.yaml`
      claims no decision, so the delivery log runs with an empty claim or is skipped.
- [ ] 6.2 Commit `chore(openspec): archive add-kubernetes-inventory-package`.
