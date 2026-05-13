## Why

The just-landed `add-cue-schema-test-harness` change shipped a working fixture harness plus three seed fixtures, covering roughly 20% of v1alpha2 schema logic — only the resources side of `#PlatformBase.#matchers`, the resources side of `_noMultiFulfiller`, and a single `#knownResources` collision. Several load-bearing design claims remain unverified:

1. **Predicate-distinctness** (`platform.cue:142-160`). The whole reason `_predicateSignature` exists is to let two transformers share a resource FQN as long as their `requiredLabels` / `requiredTraits` differ. Today nothing fails if a refactor breaks this differentiation and starts flagging legitimate distinct predicates as conflicts.
2. **`enabled: false` projection guard** (`platform.cue:62, 73, 84`). D14 in the spec — disabled registrations must hide every projection. Untested.
3. **`#TransformerContext` label/annotation merge** (`transformer.cue:89-188`). Second-densest computation block in the schema after `#PlatformBase`. Filters keys with `transformer.opmodel.dev/` prefix, stamps `app.kubernetes.io/managed-by` from `#runtimeName`, merges across module/component/controller scopes. Drives every rendered Kubernetes object's labels. Zero coverage.
4. **`#Module` UUID/FQN derivation** (`module.cue:17, 20`). UUIDv5(OPMNamespace, fqn) is deterministic across the whole platform. Without a pinned-value test, a regression in `OPMNamespace` (`types.cue:50`) silently changes every Module UUID and every label that stamps it.
5. **Trait-side symmetric paths**. `#knownTraits` projection, trait-side `#matchers`, trait-side `_noMultiFulfiller` — all twins of the resource-side code. A code edit that touches resources but misses traits ships unnoticed.
6. **Type regex gates** in `types.cue` for `#NameType`, `#FQNType`, `#ModuleFQNType`, `#MajorVersionType`. A relaxation of any one regex would let invalid identifiers through with no test signal.

This change closes those gaps with seven new fixtures + twelve new harness rows behind the same `@if(test)` convention. No production schema or Go-package change. Same SemVer surface (PATCH).

## What Changes

- Add seven CUE fixtures under `apis/core/v1alpha2/testdata/`:
  - `predicate_distinct_labels_fixture.cue` (positive) — two transformers, same FQN, distinct `requiredLabels`; asserts `_invalid` empty and `_noMultiFulfiller: 0`.
  - `enabled_false_suppresses_fixture.cue` (positive) — disabled registration carries the multi-fulfiller payload; enabled registration carries one benign transformer; asserts only the enabled module's primitives appear in `#matchers` and projections.
  - `transformer_context_labels_fixture.cue` (positive) — directly constructs `#TransformerContext`; `expect:` asserts merged `labels`/`annotations` plus a hidden `_labelKeyCount` / `_annotationKeyCount` length check that rejects `transformer.opmodel.dev/*` filter regressions.
  - `module_uuid_fixture.cue` (positive, two harness rows) — pinned name/modulePath/version, asserts `metadata.fqn` (format-string check) and `metadata.uuid` (UUIDv5 drift sentinel for `OPMNamespace`).
  - `trait_matchers_fixture.cue` (positive) — symmetric to `platform_matchers_fixture.cue` but exercises `#matchers.traits`.
  - `multi_fulfiller_traits_fixture.cue` (negative) — symmetric to `multi_fulfiller_fixture.cue` but on `requiredTraits`.
  - `type_regex_fixture.cue` (negative bundle, five harness rows) — single fixture file with multiple `bad_<reason>:` fields, each violating one regex; harness drives via `inputPath` override.
- Extend the harness in `opm/api/v1alpha2/schema_fixture_test.go`:
  - Add optional `inputPath string` field to `schemaCase`. Defaults to `"input"`. The paired positive-equality field is `"<inputPath>_expect"` (or stays `"expect"` when `inputPath == "input"`).
  - Replace the literal `cue.ParsePath("input")` / `cue.ParsePath("expect")` with derived paths so bundled fixtures can carry several cases without renaming the seed contract.
  - 12 new rows in `schemaCases` (2 from `module_uuid_fixture`, 5 from `type_regex_fixture`, 1 each from the rest).
- Update `apis/core/v1alpha2/testdata/README.md`:
  - Document the `inputPath` override + when to use it.
  - Document the `close()`-on-open-map caveat with a worked length-check example pointing at `transformer_context_labels_fixture.cue`.
- Add three `cue:test:*` Taskfile entries (already landed in the previous change). Reused unchanged.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `schema-testing`: extend the harness contract with the optional `inputPath` override and add coverage requirements for the four load-bearing design claims (predicate-distinctness, `enabled:false` suppression, `#TransformerContext` merge, `#Module` UUID/FQN pin) plus the trait-side symmetric paths and type-regex gates.

## Impact

**Affected library packages**

- `apis/core/v1alpha2/testdata/`: seven new fixture files + README update.
- `opm/api/v1alpha2/schema_fixture_test.go`: one struct field added, derivation logic replaces two literal path constants, twelve table rows appended. No new test functions.

**Affected downstream consumers**

None. Test-only addition.

**SemVer**

PATCH. Test infrastructure only; no `opm/` surface change, no schema change.

**Dependencies**

No new external dependencies.

**Risk**

Low. The fixture-authoring surface is the same one shipped by the previous change; we are only widening its application. The harness extension is a single optional field with a default that preserves prior behavior — every existing seed case continues to work without modification. The pinned UUID in `module_uuid_fixture.cue` is the one place the test inventory carries a constant; it is regenerated only when `OPMNamespace` itself changes (the documented failure mode is the whole point of pinning).

The `close()`-on-open-maps subtlety is the one finding worth flagging to readers: future fixtures asserting absence-of-keys must pair `expect:` blocks with hidden `_count` fields. The README documents this with a worked example.
