## Why

`opm/compat.Check` implements 0010 D27's additive-only rule with three field rules: removed, added-without-optional-or-default, default changed, plus a leaf subsumption for domains. It has no rule for a field that already existed and **changed posture**: `name?: string` becoming `name!: c.#ServiceNameType`. That transition goes through the leaf path, where `Subsume` compares value domains and is blind to the constraint marker, so the only report is `domain narrowed` when the type also tightens, and nothing at all when it does not (`y?: string` → `y!: string`). Measured 2026-08-28: `catalogs/opm` alpha.5 → alpha.6 made `#ExposeSchema.name` required inside `v1beta1`; every module compiled against ≤ alpha.5 lost its Services on an alpha.6+ platform (`cannot reference optional field: name`). D27 says a supplier at or above the build a module compiled against is unconditionally safe; a field made required is the clearest way to break that promise, and the comparator cannot name it.

## What Changes

- `Check` reports a new violation kind, `field made required`, when a field present in both definitions was optional (`?`) or defaulted in the prior definition and is required (`!`) or a plain non-defaulted regular field in the new one. The opposite direction (required → optional, or gaining a default) is widening and reports nothing.
- The existing rules are unchanged; a posture change that also narrows the domain reports both kinds at the path, as `default changed` + `domain narrowed` already do.

Not in scope: whether the prerelease line stays exempt from the gate (that is `cli`'s `internal/publish/compat.go`, 0011 D26); the transformer-side fix for the alpha.6 break (`catalog_opm`).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `catalog-compatibility`: "Compatibility Comparison" gains the made-required rule and a scenario.

## Impact

- `opm/compat` (`compat.go` `walkStruct`, one new `Kind*` constant); `compat_test.go` table cases. No signature change: `Violation.Kind` is a string discriminator, so a new kind is additive for `cli` (`opm catalog publish` / `registry check --compat` print kinds verbatim).
- SemVer: MINOR (`feat(compat)`): the gate can refuse a catalog it accepted before. That is the point.
- Complexity: one selector comparison in a loop that already iterates both sides.
- Enhancement: 0010 D27 (rule) and 0011 D9 (gate). Create `enhancement.yaml` declaring both.
