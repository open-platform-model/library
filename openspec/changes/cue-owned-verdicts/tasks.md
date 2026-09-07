# Tasks: cue-owned-verdicts

## 1. `Compiled` moves into the kernel (`opm/kernel`, `opm/core`)

- [ ] 1.1 `opm/kernel/render.go`: declare `Compiled` beside `RenderResult` with the four fields and the doc `core.Compiled` carries today (`Value`, `Instance`, `Component`, `Transformer`); `RenderResult.Compiled` becomes `[]*Compiled`; `render_decode.go`: `decodeRendered` and `splitOutput` construct `*Compiled`; delete `opm/core`; migrate the three `core.Compiled` references in `render_test.go` and `flow_synth_imported_test.go`; verify `go build ./...` is green and `grep -rn 'opm/core\|core\.Compiled' opm` is empty.
- [ ] 1.2 Docs for the move: `CLAUDE.md` (Repository Layout: drop the `core/` line; Kernel API surface bullet; the `*core.Compiled` paragraph; the pipeline diagram's `decodeRendered` line), `README.md` (pipeline diagram and both prose mentions), `CONSTITUTION.md` III (drop the `opm/core/` line), `docs/getting-started.md` (intro sentence, the provenance paragraph, the verb table), `.claude/skills/verify/SKILL.md` (field inspection line); verify `grep -rn 'opm/core\|core\.Compiled' CLAUDE.md README.md CONSTITUTION.md docs .claude` is empty.

## 2. Glue emits the verdict rows (`opm/internal/renderstage`)

- [ ] 2.1 `render.cue.tmpl`: add `#apiVersionLess` (regexp over `v(\d+)(alpha|beta)?(\d+)?`; level, major, minor, then key string) and make `unresolvedResources` / `unresolvedTraits` rows carry `alternatives` (same-base keys of the demand's own bucket universe, `list.Sort` on the comparator) and `disqualified: [{transformer, conflicts: _unify[tfqn].conflicts}]`; verify with a `render_test.go` fixture demanding a base implemented at `v1alpha1`, `v1beta1` and `v1` that the decoded alternatives arrive in that order and that a disqualified candidate's row lists its conflicting FQNs.
- [ ] 2.2 `render.cue.tmpl`: replace `unmatchedComponents` and `candidates` with `unmatched: [{component, candidates: [{transformer, matched, missingLabels}]}]` sorted by component then transformer; drop `bucketKeys`, `missing` and `resolved` from `diagnostics` (keep `match.resolved` for `gate`); verify the existing predicate-refusal test reads the missing label off `Diagnostics.Unmatched[0].Candidates[0]`.

## 3. Decoder reads rows flat (`opm/kernel`)

- [ ] 3.1 `render_decode.go`: `glueDiagnostics` mirrors the new export exactly; delete `byCandidate`, `matchMatrix`, `pairsOf`'s sibling helpers that only served the join, and the regroup in `gateErrors`; `gateErrors` builds the three aggregates directly from the decoded slices; delete `opm/internal/renderstage/alternatives.go` and `TestAlternatives`; verify `go build ./...` shows no `opm/compat` import under `opm/internal` (`go list -deps ./opm/internal/renderstage | grep compat` is empty) and `go test ./opm/kernel/... ./opm/internal/renderstage/...` is green.
- [ ] 3.2 `render_test.go`: add `assertGateAgrees(t, built, refused bool)` reading `gate` off the built value (expose the built value to tests through the existing internal test seam or a `renderForTest` helper) and call it from every refusal and success test in the file; verify each test passes and that flipping the helper's expectation fails it.

## 4. Rows and aggregates (`opm/errors`)

- [ ] 4.1 Declare the rows: `UnifyRefusal{Component, Transformer, Conflicts}` replaces `UnifyError`; `CandidateVerdict{Transformer, Matched, MissingLabels}` replaces `MatchResult`; `UnmatchedComponent{Component, Candidates}`; `OverSubscribedContract{Key, Catalogs}` replaces `OverSubscribedContractError`; `UnresolvedDemand.Disqualified []UnifyRefusal` and no `Error` method on any row; `TransformError{Component, Transformer, Cause}`; update `RenderDiagnostics` field types and `decodeRendered`'s `TransformError` sites; verify `go vet ./...` is green and `grep -rn 'UnifyError\|MatchResult\|ComponentName\|TransformerFQN' opm` is empty.
- [ ] 4.2 Declare the aggregates: `*UnmatchedComponentsError{Components []UnmatchedComponent}` with `Error()` listing each component and its unmatched candidates and no `Unwrap`; `*OverSubscribedContractsError{Contracts}` joined into the gate once; `*UnresolvedDemandsError` without `Unwrap`; rewrite `match_test.go`, `unmatched_test.go`, `oversubscribed_test.go` as message-shape and `errors.AsType` tests over the new types; verify every test in `opm/errors` passes and that `errors.AsType[*oerrors.TransformError]` on an unmatched-components refusal returns false.

## 5. Warnings leave the result (`opm/kernel`)

- [ ] 5.1 `render.go`, `render_decode.go`: delete `RenderResult.Warnings`, the skew sentence and `unhandledTraitWarnings`; reword the `SkewWarn` and `RenderDiagnostics` docs to name `ResolvedVersions[].Newer` and `UnhandledTraits` as the advisory sources; update the `Warnings` assertions in `render_test.go` (optional-trait, skew-warn, determinism and empty-warnings tests), `integration_test.go`, `flow_integration_test.go` and `flow_synth_catalog_import_test.go` to read the rows; verify `go test ./opm/kernel/...` is green and `grep -rn 'Warnings' opm` is empty.

## 6. Docs

- [ ] 6.1 `CLAUDE.md` § Render contract (verdicts list, "warned by default" sentence), `opm/kernel/doc.go` (result shape, example loop), `opm/errors` package doc (rows versus causes), `docs/getting-started.md:213-235` (the `result.Warnings` loop becomes a loop over the two diagnostics fields); verify `grep -rn 'result.Warnings\|RenderResult.Warnings\|UnifyError\|MatchResult' CLAUDE.md README.md docs opm/kernel/doc.go` is empty.

## 7. Validation gates

- [ ] 7.1 `task fmt`, `task vet`, `task lint`, `task test` green; `go test -race ./opm/kernel/... ./opm/internal/renderstage/...` green; `task cue:test:flow` green against GHCR; verify `openspec validate --change cue-owned-verdicts` passes.
- [ ] 7.2 `go build ./...` and `go vet ./...` in `../cli` and `../opm-operator` against a temporary `replace github.com/open-platform-model/library => ../library`, applying only the consumer edits `proposal.md` § Impact lists (the operator's `pkg/core/compiled_adapter.go` reads `*kernel.Compiled`), then revert the replace; verify the edit list matches the proposal and no `Warnings`, `Matches`, `UnifyError`, `OverSubscribedContractError` or `opm/core` reference to the library remains in either consumer.
