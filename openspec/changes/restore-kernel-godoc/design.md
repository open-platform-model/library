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

The restored file MUST contain the same sequence of words as HEAD's `doc.go`, after removing the comment markers (`//`) and whitespace. This includes the `-` list markers, which HEAD still carries as the literal `; -` joiners. Layout follows `7729b92^`:

- `# Surface`: one `  - ` item per verb group, continuation lines indented four spaces, as in `7729b92^` lines 17-34. HEAD has the same seven items, with the catalog item's wording as written by 291f78f.
- `# One-Kernel-per-process example`: the `renderAll` function as a tab-indented code block in gofmt layout, as in `7729b92^` lines 88-117.
- The diagnostics example: a tab-indented code block with three loops (`UnhandledTraits`, `ResolvedVersions`, `Skipped`). The `Skipped` loop was added by fc1aafd after the sweep, so it is reformatted into the block in the same style as the first two.
- The replacements example: a tab-indented code block, as in `7729b92^` lines 194-196.
- Prose paragraphs are rewrapped to the file's width (about 76 columns, as in `7729b92^`). This includes the lines that later commits left unwrapped.

The word-sequence check is the proof that the restoration changed only the layout. It is run as a shell one-liner during the task and is not committed:

```sh
words() { sed 's#^//##' "$1" | tr -s ' \t\n' '\n' ; }
diff <(git show HEAD:opm/kernel/doc.go | words /dev/stdin) <(words opm/kernel/doc.go)
```

The diff MUST be empty.

### The guard parses the package doc the way go doc does

`opm/kernel/doc_test.go` (package `kernel_test`, standard library only) MUST:

1. Parse `doc.go` with `go/parser.ParseFile(fset, "doc.go", nil, parser.ParseComments|parser.PackageClauseOnly)`. The test's working directory is the package directory.
2. Build the package doc with `doc.NewFromFiles` and parse it with `pkg.Parser().Parse(pkg.Doc)`. This is the same `go/doc/comment` model that `go doc` and pkg.go.dev render from.
3. Find the `*comment.Heading` whose text is `Surface`. Assert that a `*comment.List` follows it after at most one intervening `*comment.Paragraph` (the "One tier" lead-in), with at least seven items, each starting with a `[Kernel.` doc link. Today the items name `AcquireModuleFromRegistry`, `AcquireCatalogFromRegistry`, `AcquirePlatformFromDir`, `AcquireInstanceFromDir`, `SynthesizeInstance`, `ValidateConfigDetailed` and `Render`.
4. Assert that the doc has exactly three `*comment.Code` blocks. Their texts contain `func renderAll(`, `result.Diagnostics.UnhandledTraits` and `result.Diagnostics.Replacements` respectively.

Every failure names the block it expected and what it found (the block's Go type and the start of its text). A reader of the failure can then see it was a flattening and not a content change.

**Rationale**: The owner's 2026-10-03 decision asks for a go/doc guard on the Surface list and the code examples. Asserting on the parsed block types catches the exact failure 7729b92 caused. It does not pin the prose, so ordinary wording edits stay free. Exactly three code blocks is deliberate: adding a fourth example then means updating the test once, which costs less than a guard that misses a lost block.

## Research & Decisions

### Where the flattening came from

**Context**: The fix should not be undone by the same tool.
**Explored**: `git show --stat 7729b92` touches dozens of files. `opm/kernel/doc.go` is the only one rewrapped wholesale (315 changed lines); the others carry reference rewrites such as `0019 D5` to `0019:D5`. The commit adds no formatter to `Taskfile.yml` or `.golangci.yml`, and `task fmt` is `go fmt` plus `goimports`. Neither rewraps comments. A grep over library, cli and opm-operator finds no other flattened list or code comment.
**Decision**: Treat the rewrap as a one-off manual edit. The guard test is the only prevention added.
**Rationale**: There is no tool to disable. The test catches a repeat by any means.

### Restore from HEAD's text or replay HEAD's diffs onto 7729b92^

**Context**: The owner decision says to restore the lists and code blocks from `7729b92^` and re-apply the later edits, including 19a8553.
**Explored**: Replaying five content diffs onto the pre-sweep file, compared with reformatting HEAD using the pre-sweep layout. Both produce the same file.
**Decision**: Reformat HEAD's text into `7729b92^`'s layout, and prove word equality with HEAD using the check above.
**Rationale**: The word check is a mechanical proof that every later edit is present. A manual replay of five diffs has no such check.

## Risks / Trade-offs

- A future example added to the doc makes the exact-count assertion fail → the failure message says to add the new block's anchor to the test, which is a one-line edit.
- `go/doc/comment` could reclassify a block in a future Go release → the test would fail in CI, which is the right signal for a doc that `go doc` would also render differently.
