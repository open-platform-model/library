## Why

`opm/compat.Check` reports two classes of non-violations the moment its operands differ at all: provenance carried inside member references (`appliesTo`, `composedResources`, `composedTraits` embed the referenced member's `metadata.catalogVersion`, which changes every release by construction), and the already-recorded `matchN` / pending-comprehension leaf false positives (`#Image`'s `if digest != "" && tag != ""` guards report `domain narrowed (value not an instance)` on byte-identical leaves). The CLI's member-level identity fast path hides both only while a member is unchanged; on `catalog_opm` PR 51 a single genuine tightening in `#ExposeTrait` surfaced four noise rows next to it (cli issue 165), and on a stable module line the same noise would refuse a legitimately additive change. The spec records the comparator-level fix as unowned; this change owns it.

## What Changes

- `Check` skips 0010 D30's provenance fields wherever a `metadata` block occurs in the walk, not only at the compared root: `<path>.metadata.catalogVersion` and `<path>.metadata.description` at any depth are never compared. Same denylist, same direct-children-of-metadata scope, applied per occurrence.
- `Check` walks lists element-wise when both sides have the same length, so member references inside `appliesTo` / `composedResources` are compared structurally (and reach the skip above) instead of as one opaque leaf; lists of different length stay a leaf and are judged by subsumption as today.
- `Check` short-circuits a leaf whose emitted syntax is identical on both sides: an unchanged leaf cannot have narrowed, so the subsume false positive on `matchN` / comprehension-bearing leaves no longer reports. A *changed* leaf inside such a construct may still report noise; the spec's known-limitation note shrinks to that residue.
- Fixture-backed regression tests reproduce the three shapes against real catalog member syntax.
- `Violation`, `Check`, `CheckAtLevel` signatures unchanged. PATCH per SemVer for `opm/compat`: strictly fewer reported violations, all of them measured non-violations. Downstream: the cli publish gate and `registry check --compat` inherit the fix on the next dependency bump; the cli's own `provenancePaths` filter becomes redundant and may be dropped there later.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `catalog-compatibility`: the Compatibility Comparison requirement gains the nested provenance skip, element-wise list walking and the identical-leaf short-circuit; its known-limitation note is narrowed.

## Impact

- `opm/compat/compat.go` (`walk`, `walkStruct`, new `walkList`, leaf identity check), `opm/compat/compat_test.go`, new testdata mirroring `#ExposeTrait` / `#StatefulWorkloadBlueprint` / `#Image` shapes.
- Consumers: `cli/internal/publish/compat.go` (behavioural improvement only), library-matching's unify rung (unaffected: it uses `StripProvenance`, not `Check`).
- Sequencing: lands before `cli` change `compat-prerelease-line-exemption`, which depends on a clean stable-line comparison of the real tree. Related to enhancement 0011 D26's delivery note; see `enhancement.yaml`.
