## Context

Design source: the swarm design for the enhancement store, sections 6 and 9. Rules: an id beside a reason in words; no decision number; the two blocked ids lose the token; archived text follows the same rule; the repo's `config.yaml` carries the rules.

## Decisions

- **Tool first, hand second.** `enh cite-rewrite --write` does the mechanical forms. Its worklist rows are rewritten by hand with the rule. At least 40 tool edits are read before commit.
- **Commits by place.** Code and tests, live specs, archived changes, and Markdown and docs are separate commits. Types: `docs`, `test`, `chore`; `docs(openspec)` for archived text.
- **Live specs are wording only.** The edit changes no requirement and no scenario name, so there is no delta (`skip_specs: true`).
- **Test assertions change with producers.** `opm/kernel/parity_compare_unit_test.go` and the strings it checks move in one `test:` commit.
- **Excluded.** The active change `publish-go-api-bundle`. Citations of the frozen `legacy:NNN` library entries stay: they are not enhancement ids.
- **Dead tooling removed.** `.tasks/enhancements.yaml` points at an `enhancements/` folder that does not exist in this repo.
- **Lint task and CI call are out of scope.** They pin merged commits and follow in a later change.

## Risks

- A wrong rewrite changes the meaning of a comment. Mitigation: read a sample, review by an independent agent.
- The embedded template `render.cue.tmpl` feeds every render build. Mitigation: run the render tests.
