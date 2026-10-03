## Why

The `opm/kernel` package doc (`opm/kernel/doc.go`) is the render contract and the main public description of the one-Kernel API. AGENTS.md points readers to `go doc ./opm/kernel` instead of restating it. Commit 7729b92 (`chore(comments): make every enhancement reference resolvable`, 2026-09-19) rewrapped the whole file while it rewrote enhancement references. The rewrap flattened every indented block into prose:

- the `# Surface` list of verbs now runs together as one paragraph joined by literal `; -`;
- the three code examples (the `renderAll` one-Kernel-per-process example, the diagnostics wording example, the replacements wording example) are now prose paragraphs, so `go doc` and pkg.go.dev render them as unreadable run-on text.

Five later commits edited the text of the flattened file (fc1aafd, a0c6ed2, 3a53aed, f1d9908, 19a8553). Their content is correct and must be kept. Some of their lines are not wrapped (lines of 100+ columns in `# Every operation shares nothing` and `# Rendering`). No other file in library, cli or opm-operator lost a list or code block this way, so the damage is confined to this file.

Nothing stops it from happening again: `go vet`, `gofmt` and lint all accept a flattened doc comment.

## What Changes

- **Restore the layout of `opm/kernel/doc.go`** from `7729b92^`: the `# Surface` list as a Go doc list (`  - ` items with indented continuation lines), and the three examples as tab-indented code blocks. The text is HEAD's. Every edit since 7729b92, including 19a8553, is kept word for word. Paragraphs are rewrapped to the file's usual width, with no overlong lines.
- **Add a go/doc guard test** in `opm/kernel` that parses the package doc with `go/doc` and `go/doc/comment` and fails if the block after the `Surface` heading is not a list holding the kernel's verbs, or if the three code examples are not code blocks.

## Not in this change

- Rewording any part of the package doc, including the long Render refusal paragraph that the research suggested splitting into a list. That was not in the pre-sweep layout and is a content change, not a restoration.
- A general comment-formatting linter for other packages. A grep found no other flattened comments.
- Any change to exported symbols, signatures or behaviour.

## Classification

**No release.** The doc comment is restored and a test is added. No exported symbol, signature or behaviour changes. The section commits are `docs(kernel)` and `test(kernel)`, which release-please hides. Complexity (Principle VII): one test file in `opm/kernel` that uses only the standard library (`go/ast`, `go/doc`, `go/doc/comment`, `go/parser`, `go/token`). It is justified because no existing gate notices a flattened doc comment, and the package doc is the contract AGENTS.md points to.

## Downstream consumers

None. `cli` and `opm-operator` see a readable `go doc` and nothing else changes.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `kernel-runtime`: adds one requirement. The `opm/kernel` package doc renders its verb list as a list and its examples as code blocks, and a test enforces it.

## Impact

- `opm/kernel/doc.go` (comment only).
- `opm/kernel/doc_test.go` (new).
