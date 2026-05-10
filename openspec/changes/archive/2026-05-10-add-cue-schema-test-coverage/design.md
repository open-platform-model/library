## Context

`add-cue-schema-test-harness` shipped the harness; this change scales its application across the v1alpha2 schemas' load-bearing branches. The Explore survey conducted during planning confirmed three constraints that shape the design:

- `#TransformerContext` is constructable directly from a fixture (its three required inputs are unhidden), so the fixture can drive concrete `#moduleReleaseMetadata` / `#componentMetadata` / `#runtimeName` and read computed `labels`/`annotations` without going through `#transform`.
- `enabled: false` fully suppresses projection at `platform.cue:62, 73, 84`. A fixture combining a disabled multi-fulfiller registration with another enabled module surfaces a clean platform value, so the suppression can be asserted positively.
- `helpers_autosecrets.cue` is a component constructor (a function-shape that builds a `#Component` from input data), not a schema definition. The harness pattern (instantiate-validate-compare) does not fit; testing it needs unit tests in Go around the constructor's inputs/outputs. Out of scope here.

A side finding worth recording: `_predicateSignature` (`platform.cue:142-160`) does not escape `,` or `;` in label values when joining `k=v` pairs. A label value containing those characters would corrupt the signature and produce false-positive collision diagnostics. This is a real bug in the production schema, not a test gap. It is filed for separate follow-up (ADR or issue) — we explicitly do NOT add a fixture that asserts the bug.

User decisions captured in the planning round:

- **Scope: Tier 1 + Tier 2 (7 fixtures).** The four load-bearing claims (predicate-distinctness, `enabled:false`, `#TransformerContext`, UUID/FQN pin) plus the trait-side symmetric paths and type-regex gates. Tier 3 items (empty registry, module-without-`#defines`, `requiredTraits`-set distinctness) deferred.
- **`#TransformerContext` assertion style: CUE `expect:` block** (`input.Unify(expect)` under `Validate(Concrete)`). No Go decode for these cases.

## Goals / Non-Goals

**Goals:**

- Every load-bearing design claim in `platform.cue`, `transformer.cue`, and `module.cue` has at least one fixture asserting either its positive shape or its negative failure.
- The harness extension is a single optional field that does not require any existing seed fixture or harness row to change.
- Bundle the type-regex gates into one fixture file so each individual gate has a one-line declaration plus a one-line table row, not a full file each.
- Document the discovered `close()`-on-open-maps subtlety in the README so future fixture authors don't reach for it expecting absence-rejection.

**Non-Goals:**

- Coverage of `helpers_autosecrets.cue` (different shape — separate change).
- Asserting the `_predicateSignature` label-delimiter escape bug (file separately as ADR/issue; the test would lock in the bug).
- Refactoring or expanding the harness API beyond one minimal field addition.
- 100% coverage. Tier 3 items deferred to a future change.
- A Go-decode-based assertion path for `#TransformerContext` (user picked CUE `expect:` blocks).

## Decisions

### Decision 1: `inputPath` over multi-fixture splits for the type-regex bundle

**Choice:** Add an optional `inputPath string` field to `schemaCase` (default `"input"`). The paired positive-equality target is `"<inputPath>_expect"` (or stays `"expect"` when `inputPath == "input"`). One fixture file (`type_regex_fixture.cue`) carries five `bad_<reason>:` fields; five table rows distinguish them.

**Rationale:** Each type-regex assertion is one line of CUE. Splitting them into five fixture files would multiply boilerplate (`@if(test)`, `package fixtures`, `import …`, comment header) by 5x for content that fits on one line each. Bundling via `inputPath` keeps the test inventory grep-able from the harness table while keeping the fixture file scannable. The override is opt-in: every existing seed case continues to use the default `"input"` / `"expect"` paths without modification.

**Alternatives considered:**

- **One fixture per regex case.** Pure of intent (each fixture exercises one schema branch) but pays full file overhead for one-line content. Rejected for type-regex; would still be the right shape for any case that needs more than ~3 lines of CUE setup.
- **Reflection-based discovery: walk `bad_*` fields and run each automatically.** Adds harness magic for one fixture's convenience. Rejected as gratuitous abstraction; the explicit table row keeps "what's currently tested" obvious.

### Decision 2: Length-check + positive-equality, not `close()`, for `#TransformerContext` label assertions

**Choice:** The `transformer_context_labels_fixture.cue`'s `expect:` block enumerates the expected labels/annotations as a positive map AND adds hidden `_labelKeyCount: len([for k, _ in input.labels {k}]) & N` fields to assert size.

**Rationale:** `#TransformerContext.labels` ends with `...` (`transformer.cue:165-177`). Wrapping the `expect.labels` in `close({...})` does NOT cause unification to fail when the merged map carries an extra key, because the input field is open. The naive `close()` pattern silently passes a regression that lets `transformer.opmodel.dev/*` keys through the prefix filter — exactly the failure we are testing for. A length check converts the "no extras" assertion into a concrete-equality check (`len & 6`), which CUE rejects with `conflicting values N and 6` if the filter regresses.

This caveat is generalizable beyond `#TransformerContext` and is documented in the testdata README so future fixture authors don't reach for `close()` expecting absence-rejection.

**Alternatives considered:**

- **Decode to `map[string]string` + `assert.Equal` in Go.** The harness already supports `assertField`/`assertValue`. This would give the cleanest absence-rejection diagnostics. Rejected because the user explicitly picked the CUE `expect:` style for `#TransformerContext`; the length-check pattern preserves the unification-as-equality contract while still catching the failure mode that motivates the test.
- **Per-key explicit absence assertions.** CUE has no clean "field must not exist" syntax. Workarounds (e.g. `_no_internal: input.labels."transformer.opmodel.dev/internal" == _|_`) don't compose cleanly because `_|_` comparisons are not first-class. Rejected.

### Decision 3: Pin the UUIDv5 hash as a Go-side constant in the harness table, not in CUE

**Choice:** `module_uuid_fixture.cue` declares only the inputs (name, modulePath, version). The expected UUID lives in the `assertValue` of the harness row. Comment in the row documents the regeneration command.

**Rationale:** A pinned UUID is a drift sentinel for `OPMNamespace`. If `OPMNamespace` changes, every Module UUID changes, and we want the test to fail loudly until the operator decides whether the change is intentional. Pinning the UUID in CUE (e.g. `expect: metadata: uuid: "<value>"`) would couple the assertion to fixture parsing, making it harder to update in a hotfix scenario where the UUID needs regenerating but the fixture-shape stays the same. Pinning in Go keeps the regeneration story trivial: one line edit in the table, no CUE re-parse to reason about.

The fixture comment + the harness-row comment both document the regeneration command (`cue eval -t test --expression input.metadata.uuid`).

**Alternatives considered:**

- **`expect:` block with the pinned UUID inside CUE.** Symmetrical with the other positive cases. Rejected for the regeneration ergonomics noted above.
- **Compute the UUID in Go via a `cue_uuid.SHA1`-equivalent and compare.** Defeats the purpose — we would not catch a regression in the CUE-side computation, only a drift in `OPMNamespace`.

### Decision 4: Trait-side fixtures duplicate, not parameterize, the resource-side seeds

**Choice:** `trait_matchers_fixture.cue` and `multi_fulfiller_traits_fixture.cue` are written from scratch as twins of their resource-side equivalents.

**Rationale:** The schemas have separate `requiredResources` and `requiredTraits` code paths in `_predicateSignature`, `_resourceCandidates` / `_traitCandidates`, and `_invalid.{resources,traits}`. Parameterizing across the two would either require macro-style CUE templating (CUE doesn't do that) or a Go-side test helper that builds either side. Both add complexity for small savings. Two ~80-line fixture files are honest and direct.

**Alternatives considered:**

- **One fixture file with both `traits_*` and `resources_*` versions of each case.** Saves no lines (each case still needs full setup) and obscures which case is exercising which side.
- **Helper CUE library that builds either side from a parameter.** Rejected; the schemas are simple enough that direct construction is more readable than parameterized construction.

### Decision 5: Defer `helpers_autosecrets.cue` testing entirely

**Choice:** Out of scope. Document in proposal + design as a known gap.

**Rationale:** Per the Explore survey, `helpers_autosecrets.cue` is a component constructor — `#OpmSecretsComponent` builds a `#Component` from a `#secrets` input map. The harness pattern (instantiate definition + assert validation outcome) does not fit because the interesting behavior (constructed component shape, label conflicts under specific input shapes) is data-driven, not closure-driven. Testing it needs unit tests in Go around the constructor's inputs and outputs, or a different fixture style that supplies a concrete `#secrets` map and asserts the constructed `out` value. Either approach is a separate change with its own design questions.

## Risks / Trade-offs

- **`inputPath` field on `schemaCase` is mutable shape.** A future change might want a third axis (e.g. `assertPath` distinct from `inputPath`). Mitigation: the field is optional with a sensible default; adding more optional fields later is non-breaking. Not a concern.
- **Pinned UUID drifts on `OPMNamespace` change.** Failure mode IS the test value. Mitigation: regeneration command documented in two places (fixture comment, harness row comment).
- **Length-check pattern proliferation.** If many future `#TransformerContext`-shaped fixtures need this pattern, the README guidance becomes load-bearing. Mitigation: README has a worked example; if it becomes more than 2-3 fixtures, extract a CUE helper definition (`#KeyCount: { _: int & len([for k, _ in source {k}]) }`) — but premature today.
- **Trait-side fixtures duplicate logic that could regress in tandem with resource-side.** This is the WHOLE POINT of the symmetric tests; if resource-side regression also breaks trait-side, both fixtures fail and the operator sees both failure paths immediately.

## Migration Plan

Not applicable — purely additive test infrastructure. No production code is modified, no schema requirement changes, no public Go API changes.

## Open Questions

- Should the type-regex bundle's `expectError: "invalid value"` regex be tightened to match the specific regex pattern (e.g. `out of bound =~"^v\\[0-9\\]\\+\\$"`) so a regression that changes the regex without removing the gate would still fail loudly? Today the bundle would silently pass if the gate were relaxed in a way that still rejected the test value. Likely yes, but the regex strings have backslash-escape gymnastics that hurt readability — defer to a follow-up if it becomes a real concern.
- The `_count` length-check pattern may become a documented CUE fixture idiom across future fixtures. If so, lift to a shared `#KeyCount` definition in a `testdata/_helpers.cue` file (with `@if(test)` gate). Defer until we have a third use site.
