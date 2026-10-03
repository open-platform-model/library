## Context

`opm/kernel/doc.go` is the package doc and the render contract. Commit 7729b92 rewrapped it as plain prose, which removed the Go doc list and the three code blocks. After that, five commits changed its wording: fc1aafd (skip unprovided demands, which also added the `Skipped` loop to the diagnostics example), a0c6ed2 (provider count from core), 3a53aed (contract collisions), f1d9908 (acquire validates the package's own values) and 19a8553 (retire the materialize wording). The pre-sweep file (`git show 7729b92^:opm/kernel/doc.go`) has the right layout and outdated wording. HEAD has the right wording and the broken layout.

## Goals / Non-Goals

**Goals:**

- `go doc ./opm/kernel` renders the `# Surface` verbs as a list and the three examples as code blocks.
- The wording is HEAD's, word for word, so no later edit is lost.
- A test fails if the list or the code blocks are flattened again.

**Non-Goals:**

- Rewording or restructuring the doc, including turning other paragraphs into lists.
- Checking the layout of comments in other packages.

## Decisions

### HEAD supplies the words, 7729b92^ supplies the layout

The restored file MUST contain the same sequence of words as the base's `doc.go` (the branch's merge base with `origin/main`, not `HEAD`), after removing the comment markers (`//`) and whitespace. This includes the `-` list markers, which HEAD still carries as the literal `; -` joiners. Layout follows `7729b92^`:

- `# Surface`: one `  - ` item per verb group, continuation lines indented four spaces, as in `7729b92^` lines 17-34. HEAD has the same seven items, with the catalog item's wording as written by 291f78f.
- All three code blocks are tab-indented (`//` plus one tab) and use gofmt layout, with tabs for nesting. `7729b92^` supplies only each block's boundaries and statement order; it does not supply the indentation, because its `renderAll` body nests with four spaces after the tab (lines 89-116) while its other two examples nest with tabs.
- `# One-Kernel-per-process example`: the `renderAll` function, as in `7729b92^` lines 88-117.
- The diagnostics example: three loops (`UnhandledTraits`, `ResolvedVersions`, `Skipped`). The `Skipped` loop was added by fc1aafd after the sweep, so it is reformatted into the block in the same style as the first two.
- The replacements example, as in `7729b92^` lines 194-196.
- Prose paragraphs are rewrapped to the file's width (about 76 columns, as in `7729b92^`). This includes the lines that later commits left unwrapped.

The word-sequence check is the proof that the restoration changed only the layout. It is run as a shell one-liner during the task and is not committed:

```sh
words() { sed 's#^//##' "$1" | tr -s ' \t\n' '\n' ; }
diff <(git show "$(git merge-base HEAD origin/main)":opm/kernel/doc.go | words /dev/stdin) <(words opm/kernel/doc.go)
```

The diff MUST be empty. The baseline is pinned to the merge base: once section 1 is committed, `HEAD` holds the restored file, and a diff against it would be empty and prove nothing.

### The guard parses the package doc the way go doc does

`opm/kernel/doc_test.go` (package `kernel_test`, standard library only) MUST:

1. Parse every non-test `.go` file in the package directory with `go/parser.ParseFile(..., parser.ParseComments)`, as `go doc` does. A doc.go-only parse is not enough: without the package's declarations, `pkg.Parser()` cannot resolve `[Kernel.X]` links, and the list items start with plain text instead of doc links. The test's working directory is the package directory.
2. Build the package doc with `doc.NewFromFiles` and parse it with `pkg.Parser().Parse(pkg.Doc)`. With the whole package parsed, this is the same `go/doc/comment` model that `go doc` renders from.
3. Find the `*comment.Heading` whose text is `Surface`. Assert that a `*comment.List` follows it after at most one intervening `*comment.Paragraph` (the "One tier" lead-in), with at least seven items, each starting with a `*comment.DocLink` whose receiver is `Kernel`. Today the items name `AcquireModuleFromRegistry`, `AcquireCatalogFromRegistry`, `AcquirePlatformFromDir`, `AcquireInstanceFromDir`, `SynthesizeInstance`, `ValidateConfigDetailed` and `Render`.
4. Assert that each code anchor is found inside a `*comment.Code` block. There is one anchor per loop and per example boundary (`func renderAll(`, `for _, dir := range instanceDirs`, `close(errs)`, `for err := range errs`, and `result.Diagnostics.` followed by `UnhandledTraits`, `ResolvedVersions`, `Skipped` and `Replacements`), so a partly flattened example fails as well as a wholly flattened one. The number of code blocks is not pinned, so a new example does not break the test.
5. Assert that no block other than a `*comment.Code` contains a Go code fragment (` := `, `result.Diagnostics.`, `log.Printf(`, `if err != nil`). The prose never carries these, so this catches a flattened loop that no anchor names yet.

The parse and the assertions sit in helpers: one parses the package with the `doc.go` bytes supplied by the caller in place of the file on disk, and one returns the layout problems of a parsed doc. The committed test feeds them the file on disk. The proof in task 2.2 feeds them other `doc.go` bytes from a throwaway test file, with no copy of the package.

Every failure names the block it expected and what it found (the block's Go type and the start of its text). A reader of the failure can then see it was a flattening and not a content change.

**Rationale**: The owner's 2026-10-03 decision asks for a go/doc guard on the Surface list and the code examples. Asserting on the parsed block types catches the exact failure 7729b92 caused. It does not pin the prose or the number of examples, so ordinary wording edits and new examples stay free. The per-loop anchors and the code-in-prose check together catch a lost block or a partly lost one.

## Research & Decisions

### Where the flattening came from

**Context**: The fix should not be undone by the same tool.
**Explored**: `git show --stat 7729b92` touches dozens of files. `opm/kernel/doc.go` is the only one rewrapped wholesale (315 changed lines); the others carry reference rewrites to the `NNNN:Dn` form (for example `0019:D5`). The commit adds no formatter to `Taskfile.yml` or `.golangci.yml`, and `task fmt` is `go fmt` plus `goimports`. Neither rewraps comments. A grep over library, cli and opm-operator finds no other flattened list or code comment.
**Decision**: Treat the rewrap as a one-off manual edit. The guard test is the only prevention added.
**Rationale**: There is no tool to disable. The test catches a repeat by any means.

### Restore from HEAD's text or replay HEAD's diffs onto 7729b92^

**Context**: The owner decision says to restore the lists and code blocks from `7729b92^` and re-apply the later edits, including 19a8553.
**Explored**: Replaying five content diffs onto the pre-sweep file, compared with reformatting HEAD using the pre-sweep layout. Both produce the same file.
**Decision**: Reformat HEAD's text into `7729b92^`'s layout, and prove word equality with the merge base's file using the check above.
**Rationale**: The word check is a mechanical proof that every later edit is present. A manual replay of five diffs has no such check.

## Risks / Trade-offs

- [The sibling change `record-walkthrough-decisions` (its task 3.2) rewrites a sentence in the Goroutine safety paragraph of the same file] → the two changes touch disjoint hunks and merge cleanly in either order (checked with `git merge-tree`). If a rebase is ever needed, the word-sequence check runs against the new merge base, so the sibling's sentence is kept verbatim and is never reverted to make the diff empty.
- [The word-sequence check is no longer empty: the `Kernel.SynthesizeInstance` Surface item was corrected after review] → the diff is exactly that one sentence; every other word is HEAD's.
- `go/doc/comment` could reclassify a block in a future Go release → the test would fail in CI, which is the right signal for a doc that `go doc` would also render differently.
