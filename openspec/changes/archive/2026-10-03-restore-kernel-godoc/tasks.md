# Tasks: restore-kernel-godoc

No exported symbol and no behaviour changes. Each section's gate is `task check` (fmt, vet, lint, test) and `openspec validate restore-kernel-godoc --strict`. If the known flaky `TestGenerate_BuildsThroughTheKernel` fails in a full run, re-run it alone; it is a cross-process cache race and not caused by this change.

## 1. Restore the package doc layout (kernel)

- [x] 1.1 Rewrite the comment layout of `opm/kernel/doc.go` following `git show 7729b92^:opm/kernel/doc.go`. The `# Surface` verbs become a Go doc list (`//   - ` items with four-space continuation lines). The `renderAll` example, the diagnostics example (with fc1aafd's `Skipped` loop, formatted like the other two loops) and the replacements example become tab-indented code blocks in gofmt layout (tab nesting in all three; `7729b92^` supplies only block boundaries and statement order). Prose paragraphs are rewrapped to about 76 columns, which also fixes the overlong lines later commits left in `# Every operation shares nothing` and `# Rendering`. The words stay HEAD's.
- [x] 1.2 Verify the words did not change. The word-sequence diff in design.md (the merge base's `doc.go`, `git show $(git merge-base HEAD origin/main):opm/kernel/doc.go`, compared with the worktree's, after removing comment markers and whitespace) is empty, and `git diff opm/kernel/doc.go` touches comment lines only.
- [x] 1.3 Verify the rendering. `go doc ./opm/kernel` prints the seven Surface items as a list and the three examples as indented code. `gofmt -l opm/kernel` is empty. No line in `doc.go` is longer than 100 columns, apart from code-block lines kept as long as `7729b92^` had them.
- [x] 1.4 `task check` green, then commit `docs(kernel): restore the package doc's verb list and code examples`.

## 2. Guard the layout with a go/doc test (kernel)

- [x] 2.1 Add `opm/kernel/doc_test.go` (package `kernel_test`, standard library only) as specified in design.md. It parses every non-test `.go` file of the package with `go/parser` (`ParseComments`) and `doc.NewFromFiles`, parses `pkg.Doc` with `pkg.Parser()`, and asserts that a `*comment.List` of at least seven items, each led by a `*comment.DocLink` with receiver `Kernel`, follows the `Surface` heading, and that each of `func renderAll(`, `result.Diagnostics.UnhandledTraits` and `result.Diagnostics.Replacements` is found inside a `*comment.Code` block. The parse takes the `doc.go` bytes as an argument and the assertions take the parsed doc, so the proof in 2.2 needs no package copy. Failure messages name the expected block and the Go type and opening text of what was found.
- [x] 2.2 Show that the test catches both failures from a throwaway test file, never committed (`opm/kernel/zz_proof_test.go`, removed afterwards). It feeds the helpers the merge base's flattened `doc.go` (`git show $(git merge-base HEAD origin/main):opm/kernel/doc.go`, saved in the scratchpad) and gets problems for both the Surface list and the code blocks; it feeds them section 1's file with one code block indented as prose and gets a problem naming that block. Afterwards `git status` shows only `doc_test.go` as new.
- [x] 2.3 `task check` green, then commit `test(kernel): fail when the package doc loses its list or code blocks`.

## 3. Apply the code review (kernel)

- [x] 3.1 Anchor every loop and example boundary in `doc_test.go` (`for _, dir := range instanceDirs`, `close(errs)`, `for err := range errs`, `result.Diagnostics.ResolvedVersions`, `result.Diagnostics.Skipped` beside the first three), and fail when any block other than a `*comment.Code` contains a Go code fragment. Drop the commit hash from the test's header comment.
- [x] 3.2 Re-run the 2.2 proof with the diagnostics example's `Skipped` loop alone rewrapped into prose; the test reports that loop.
- [x] 3.3 Correct the `Kernel.SynthesizeInstance` Surface item in `doc.go`: the import names the major of the kernel's schema release, and the release it resolves to is the one the module's own `cue.mod/module.cue` pins, as the method's godoc says.
- [x] 3.4 Align proposal.md (overlong prose lines only) and the design.md risk on `record-walkthrough-decisions` (disjoint hunks) with what shipped.
- [x] 3.5 `task check` green, then commit `docs(kernel): anchor every doc example loop and correct the synthesis import sentence`.
