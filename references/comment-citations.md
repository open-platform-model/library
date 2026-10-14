# Enhancement references in comments

Covers when and how a comment cites an enhancement.

Default is none: a comment says what the code does and why, in its own words.

- In all text (proposal, design, specs, docs and comments), when a rationale genuinely lives in an enhancement, cite it **once at the symbol** (in a comment): the enhancement id beside a reason in words, such as `(enhancement 0011)` after the sentence it explains. Several enhancements: `(enhancements 0010 and 0011)`. The id is never alone, and a comment that is only an id is wrong. Read an entry with `openspec show <id>-<slug> --store opm-enhancements`.
- Never a decision number. The store has no decision numbers; a citation such as an id with a colon and a decision number, or a bare decision number, names nothing. `enh cite-lint` rejects it in comments.
- Never a section, slice, phase, task or design-doc-local number (a section sign with a number, `slice C2`, `task 4.2`). They are not stable identifiers.
- Never in scaffold templates, generated files, fixtures a user copies, or CLI output strings. Those reach people who have no access to the enhancements repo.
- No `Was:` rename history. `git log` owns it.
- In CUE files the reference goes in a `// WHY` block separated from the doc comment by one blank line, never in the doc comment itself: `cue lsp` hover and `cue def` replay a doc comment verbatim.
