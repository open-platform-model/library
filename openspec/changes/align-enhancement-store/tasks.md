## 1. Rewrite

- [ ] 1.1 Run `enh cite-rewrite --write` once; commit its result by place (code and tests, live specs, archived changes, docs).
- [ ] 1.2 Rewrite the worklist rows by hand (Go and CUE, testdata, the render template, tests); commit.
- [ ] 1.3 Change the parity producer strings and their assertions together; run the render and kernel tests; commit as `test:`.
- [ ] 1.4 Rewrite worklist rows in live specs, archived changes and Markdown; commit.
- [ ] 1.5 `enh cite-lint` reports 0 findings and `cite-rewrite --dry-run` 0 edits; `git grep` finds no blocked-id token; `task check` passes.

## 2. Rules and tooling

- [ ] 2.1 Update `openspec/config.yaml`: `references:`, the `Delivers` rule, the `context` line, the comment rule; remove `enhancement.yaml` and `delivery` rules.
- [ ] 2.2 Update `AGENTS.md` to match; remove `.tasks/enhancements.yaml` and its `Taskfile.yml` include.
- [ ] 2.3 `openspec validate align-enhancement-store --strict` passes; `task check` passes; commit.
