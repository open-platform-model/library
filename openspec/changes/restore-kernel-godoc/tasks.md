# Tasks: restore-kernel-godoc

No exported symbol and no behaviour changes. Each section's gate is `task check` (fmt, vet, lint, test) and `openspec validate restore-kernel-godoc --strict`. If the known flaky `TestGenerate_BuildsThroughTheKernel` fails in a full run, re-run it alone; it is a cross-process cache race and not caused by this change.

## 1. Restore the package doc layout (kernel)

- [ ] 1.1 Rewrite the comment layout of `opm/kernel/doc.go` following `git show 7729b92^:opm/kernel/doc.go`. The `# Surface` verbs become a Go doc list (`//   - ` items with four-space continuation lines). The `renderAll` example, the diagnostics example (with fc1aafd's `Skipped` loop, formatted like the other two loops) and the replacements example become tab-indented code blocks in gofmt layout. Prose paragraphs are rewrapped to about 76 columns, which also fixes the overlong lines later commits left in `# Every operation shares nothing` and `# Rendering`. The words stay HEAD's.
- [ ] 1.2 Verify the words did not change. The word-sequence diff in design.md (HEAD's `doc.go` compared with the worktree's, after removing comment markers and whitespace) is empty, and `git diff opm/kernel/doc.go` touches comment lines only.
- [ ] 1.3 Verify the rendering. `go doc ./opm/kernel` prints the seven Surface items as a list and the three examples as indented code. `gofmt -l opm/kernel` is empty. No line in `doc.go` is longer than 100 columns, apart from code-block lines kept as long as `7729b92^` had them.
- [ ] 1.4 `task check` green, then commit `docs(kernel): restore the package doc's verb list and code examples`.

## 2. Guard the layout with a go/doc test (kernel)

- [ ] 2.1 Add `opm/kernel/doc_test.go` (package `kernel_test`, standard library only) as specified in design.md. It parses `doc.go` with `go/parser` and `doc.NewFromFiles`, parses `pkg.Doc` with `pkg.Parser()`, and asserts that a `*comment.List` of at least seven `[Kernel.`-led items follows the `Surface` heading, and that exactly three `*comment.Code` blocks exist, anchored on `func renderAll(`, `result.Diagnostics.UnhandledTraits` and `result.Diagnostics.Replacements`. Failure messages name the expected block and the Go type and opening text of what was found.
- [ ] 2.2 Show that the test catches both failures in a throwaway copy, never committed. With HEAD-of-main's flattened `doc.go` restored in a scratch copy of the package, the test fails on both the Surface list and the code blocks. With one code block in section 1's file indented as prose, it fails naming that block. Afterwards `git status` shows only `doc_test.go` as new.
- [ ] 2.3 `task check` green, then commit `test(kernel): fail when the package doc loses its list or code blocks`.
