## 1. Tier 1 fixtures (4)

- [x] 1.1 Create `apis/core/v1alpha2/testdata/predicate_distinct_labels_fixture.cue` — two transformers, same FQN, different `requiredLabels`; `expect:` asserts both candidates present, `_invalid: []`, `_noMultiFulfiller: 0`. Verified `cue vet -c -t test` exit 0 with `_assert: input & expect`.
- [x] 1.2 Create `apis/core/v1alpha2/testdata/enabled_false_suppresses_fixture.cue` — disabled module with multi-fulfiller payload + enabled module with single benign transformer; `expect:` asserts only the enabled primitives surface. Verified concrete-unification.
- [x] 1.3 Create `apis/core/v1alpha2/testdata/transformer_context_labels_fixture.cue` — direct `#TransformerContext` construction across three label/annotation scopes including `transformer.opmodel.dev/internal` keys that MUST filter; `expect:` asserts merged maps + hidden `_labelKeyCount`/`_annotationKeyCount` length-checks (sanity-verified by flipping count and observing `conflicting values` failure).
- [x] 1.4 Create `apis/core/v1alpha2/testdata/module_uuid_fixture.cue` — pinned name=`demo`, modulePath=`example.com/demo`, version=`0.1.0`. Computed pinned values via `cue eval -t test --expression input.metadata.{fqn,uuid}`: FQN = `example.com/demo/demo:0.1.0`, UUID = `174b36e1-e5ea-5ba4-8de2-8b30c882e669`. Documented regeneration command in fixture comment.

## 2. Tier 2 fixtures (3)

- [x] 2.1 Create `apis/core/v1alpha2/testdata/trait_matchers_fixture.cue` — symmetric to seed `platform_matchers_fixture.cue` but exercises `#matchers.traits`. Verified concrete-unification of `input & expect`.
- [x] 2.2 Create `apis/core/v1alpha2/testdata/multi_fulfiller_traits_fixture.cue` — symmetric to seed `multi_fulfiller_fixture.cue` but on `requiredTraits`. Verified `cue vet -c -t test` exit 1 with `_noMultiFulfiller` in error message.
- [x] 2.3 Create `apis/core/v1alpha2/testdata/type_regex_fixture.cue` — five `bad_<reason>` fields exercising `#NameType` (uppercase + leading-hyphen), `#FQNType` (no @vN), `#ModuleFQNType` (no semver), `#MajorVersionType` (no v prefix). Verified each via `cue eval -t test --expression bad_<reason>` produces "invalid value" error.

## 3. Harness extension

- [x] 3.1 Add optional `inputPath string` field to `schemaCase` in `pkg/api/v1alpha2/schema_fixture_test.go:30-36`
- [x] 3.2 Replace literal `cue.ParsePath("input")` and `cue.ParsePath("expect")` with derived paths: `inputPath` defaults to `"input"`; `expectPath` is `"<inputPath>_expect"` unless `inputPath == "input"` in which case it stays `"expect"` (preserves seed-fixture contract)
- [x] 3.3 Update doc comment on `schemaCase` to document the new field, the `<inputPath>_expect` derivation, and when to use the override

## 4. Harness table rows (12 new rows)

- [x] 4.1 Add Tier 1 rows under a `// ── Tier 1 ──` comment: `predicate_distinct_labels_keeps_invalid_empty`, `disabled_module_suppresses_projections`, `transformer_context_merges_three_scopes`, `module_fqn_matches_format_string` (with `assertField`/`assertValue`), `module_uuid_matches_pinned_v5_hash` (with `assertField`/`assertValue` and regeneration comment)
- [x] 4.2 Add Tier 2 rows under a `// ── Tier 2 ──` comment: `trait_matchers_projects_single_candidate`, `multi_fulfiller_traits_violates_no_multi_fulfiller` (regex `_noMultiFulfiller`), and five `type_regex_rejects_*` rows each with distinct `inputPath` and `expectError: "invalid value"`
- [x] 4.3 Run `task cue:test:verbose` and confirm 15 subtests pass (3 seed + 12 new)

## 5. README update

- [x] 5.1 Add "Bundling multiple cases per fixture (`inputPath` override)" section to `apis/core/v1alpha2/testdata/README.md` with rationale, when to use it, when not to use it, and worked example pointing at `type_regex_fixture.cue`
- [x] 5.2 Add "Caveat: `close()` does not forbid extra keys on open maps" section with the length-check pattern and a worked example pointing at `transformer_context_labels_fixture.cue`

## 6. Validation gates

- [x] 6.1 `task fmt` — exit 0
- [x] 6.2 `task vet` — exit 0
- [x] 6.3 `task lint` — `0 issues.` exit 0
- [x] 6.4 `task cue:test` — `pkg/api/v1alpha2 0.102s ok`; 15 subtests pass
- [x] 6.5 `task test` — full suite green (no regression in existing tests)
- [x] 6.6 `cue vet ./...` from `apis/core/v1alpha2/` — exit 0, no fixture file paths in output
