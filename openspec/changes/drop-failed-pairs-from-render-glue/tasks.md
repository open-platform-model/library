## 1. Measure the base and pin today's failed-pair verdicts

memprobe is a scratchpad copy of `claude-stuff/kernel-plan-beta1/memprobe` whose `go.mod` `replace`
names this worktree. It has a `MODULE` knob (passed as `-module`), a `CASES` knob (a space-separated
list of case names), and `user_s` and `sys_s` columns from `getrusage`. None of it is committed to
the library. Every memprobe run uses
`CASES="r1-nil r2-hold r2-nil" RUNS=5 MODULE=<m> ./run.sh <label>` for each of
`fixtures/cert_manager`, `modules/k8up` and `modules/web_app`, and additionally
`CASES="core r1-hold r1-nil r2-hold r2-nil stag-hold stag-nil" RUNS=5 MODULE=fixtures/cert_manager`
for the operator memory package table. k8up and web_app are measured from the `modules/` checkout at
commit `09cff2b` (both directories last changed in `d4d2e09`); copies taken before any code edit
sit in the scratchpad memprobe's `fixtures/`, and the after run uses those copies. Library tests
run with a private `TMPDIR` (`TMPDIR=$(mktemp -d)`).

Every commit task stages the files it names with `git add <file>` (never `git add -A`), and checks
that `git status` then shows only the throwaway dump test as untracked before committing.

- [x] 1.1 Run memprobe at the base, before any code edit, and record the medians of `render_peak_heap`, `render_peak_live`, `vmhwm` and `user_s` per module and case in design.md under "Measurements / Before". (Done while planning; the numbers are in design.md.) Then run the full-case cert_manager set and record the operator columns in design.md "Operator memory package", against the wave-1 baseline (`memprobe/results/baseline-20261003T140923`, library `507379a`).
- [x] 1.2 Add a throwaway test, not committed, that renders every scenario and platform pairing the `opm/kernel` render tests use and writes the decoded `RenderDiagnostics` (every field) plus each `Compiled`'s provenance as JSON into the scratchpad. Run it at the base and keep the dump (design D4). Keep the test source in the scratchpad next to the memprobe copy, and record the scenario list and the dump's sha256 in design.md "Verification".
- [x] 1.3 Add scenario `testdata/render/scenarios/failing_beside_refused`: component `crash` with the broken resource (as in `failing`) beside a component whose resource demand no transformer serves (as in `missing`), on `platform`. Add its row to `testdata/render/scenarios/README.md`.
- [x] 1.4 In `opm/kernel/render_test.go`, add `TestRender_FailedPairsReportedOnGateRefusal`. Render the new scenario and assert the following: the error is a `RenderError` whose cause is the gate's `UnresolvedDemandsError`; no `TransformError` is joined in; and `Diagnostics.FailedPairs` equals exactly the `crash` pair. It passes at base, where the glue fills the list.
- [x] 1.5 In `TestRender_FailingPairIsDataBesideHealthy`, assert that the `TransformError` cause is the pair's own CUE error: it names `rendered."crash :: …".output` and is not the string `transformer output is an error`. This pins design D2 (an error nested inside the output surfaces through `Err()`). It passes at base.
- [x] 1.6 `task fmt`, `task vet`, `task lint` and `task test` green, then commit `test(kernel): pin failed-pair reporting on gate refusal`.

## 2. render: drop failedPairs from the glue and name failed pairs in Go

- [x] 2.1 `opm/internal/renderstage/render.cue.tmpl`: delete `diagnostics.failedPairs` and its comment (`:550-561`).
- [x] 2.2 `opm/kernel/render_decode.go`: remove `FailedPairs` from `glueDiagnostics` and the `pairsOf(g.FailedPairs)` line. `decodeRenderDiagnostics` sets `FailedPairs` to a non-nil empty slice. Pull out the per-pair output lookup (`pairOutput`) and add the `failedPairs` helper (design D1: `Exists()` checked before `Err()`, pair order kept).
- [x] 2.3 `decodeRendered` drops the `failed` map and the `"transformer output is an error"` fallback. It returns the pairs whose output `Err()` is non-nil next to its compiled output and error, so the pair-failure path needs no second walk.
- [x] 2.4 `opm/kernel/render.go`: on the gate-refusal branch, fill `diag.FailedPairs` from the helper before building the `RenderError`. On the decode-error branch, fill it from what `decodeRendered` returned. Reword the `RenderDiagnostics.FailedPairs` doc comment: it names the matched pairs whose output is an error, it is filled on every `RenderError` raised after the build (a gate refusal included), and it is empty on success.
- [x] 2.5 `docs/site/diagnostics/transform-failed.md`: drop `"transformer output is an error"` from the maintainer comment's list of causes (`:12`) and from the "Recognise it by" comment (`:26`). The reader-visible text stays.
- [x] 2.6 All `FailedPairs` assertions (`render_test.go:135`, `:444`, `:469`, `:623`) and the section 1 tests pass unchanged. Re-run the 1.2 dump: it is byte-identical to the base dump. Record its sha256 in design.md "Verification".
- [x] 2.7 Add `opm/kernel/render_glue_shape_test.go`: through `RenderForTest`, the built value's `diagnostics` has no `failedPairs` field (design D4).
- [x] 2.8 `task fmt`, `task vet`, `task lint` and `task test` green, then commit `perf(render): name failed pairs in Go instead of re-applying every pair`.

## 3. render: evaluate the matching rungs for candidates only

- [x] 3.1 `render.cue.tmpl`: build `_unify` and `_pred` with `for tfqn, _ in _candidates let tf = #transformers[tfqn]`. Update their comments (`:173-175`, `:194-196`) to say they cover the candidates only.
- [x] 3.2 Rewrite `unifyFailures` to iterate `#transformers`, filtered on `v._candidates[tfqn]` and on `!v._unify[tfqn].ok`. That keeps today's row order (design D3). Update its comment.
- [x] 3.3 Re-run the 1.2 dump: it is byte-identical to the base dump. Record its sha256 in design.md "Verification".
- [x] 3.4 In `render_glue_shape_test.go`, assert that for a component on a platform with non-candidate transformers, the keys of the hidden `_unify` and `_pred` fields under `match.verdicts.<component>` equal the keys of `_candidates`, and that a non-candidate transformer exists on that platform (design D4).
- [x] 3.5 `task fmt`, `task vet`, `task lint` and `task test` green, then commit `perf(render): evaluate the matching rungs for candidates only`.

## 4. Measure after and run the full suite

- [x] 4.1 Re-run memprobe at the change's head with the same knobs (k8up and web_app from the scratchpad fixture copies), including the full-case cert_manager set. Fill design.md "Measurements / After" in the same table shape and the "after g4" column of "Operator memory package", add a before/after ratio per module for `render_peak_heap` and `user_s`, and update the proposal's numbers if they moved. If cert_manager's `r1-nil` `render_peak_heap` has not dropped to at most half of its base value, stop and report it to the supervisor.
- [x] 4.2 Run the full non-short suite with the network tests forced: `OPM_FLOW_TEST_FORCE=1 go test -race ./opm/kernel/... ./opm/internal/renderstage/...` and then `OPM_FLOW_TEST_FORCE=1 go test ./...`. `TestParity_*`, `TestRender_InventoryParity` and the flow tests must run, not skip. Record the result in design.md under "Verification".
- [x] 4.3 Run `task check`. Then delete the throwaway dump test from 1.2 and confirm that `git status` shows no edit outside this change.
- [x] 4.4 Commit `chore(openspec): record drop-failed-pairs-from-render-glue measurements`.

## 4b. Code-review round

- [x] 4b.1 Add scenario `testdata/render/scenarios/unify_order` and `TestRender_UnifyRowsFollowTransformerOrder`: one component with three always-unify refusals whose candidate walk order differs from `#transformers` order; assert `Diagnostics.Unify` follows `#transformers` order. Verify: rewriting `unifyFailures` to iterate `v._candidates` fails the test.
- [x] 4b.2 Add `opm/kernel/render_decode_internal_test.go` with a table test for `outputFailed` against `#T: output: {...} | [...{...}]`: healthy struct and list (not listed), nested non-concrete struct and list (not listed), nested conflict in a struct and in a list item (listed), root-incomplete (listed), missing lookup (not listed). Verify: dropping `out.Exists() &&` fails the test.
- [x] 4b.3 Add `pairKey` and use it in `pairOutput` and in the missing-output refusal. Reword the proposal on the root-incomplete shape and the design Risks bullet on row order.

## 5. Verify and archive

- [x] 5.1 Run `openspec verify` for `drop-failed-pairs-from-render-glue` (the repo's openspec-verify-change skill). Verify: no CRITICAL finding.
- [ ] 5.2 At PR time, not in the implement stage: run `openspec archive drop-failed-pairs-from-render-glue --yes`. Verify: the main `single-build-render` spec carries the modified requirement with the new scenario, the change sits under `openspec/changes/archive/`, and `openspec validate --specs --strict` passes.
- [ ] 5.3 Run the gates green, then commit `chore(openspec): archive drop-failed-pairs-from-render-glue` (the archive rides the implementing PR).
