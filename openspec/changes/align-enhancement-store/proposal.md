## Why

The enhancements now live in an OpenSpec store (`opm-enhancements`). Its entries have no numbered decisions, so a citation of an id and a decision number points at nothing. The new rule: an enhancement id stands beside a reason in words, and never alone, and never as a decision number. The two blocked ids lose the token and keep the reason text. Library must follow this rule in its comments, specs, archived text and docs, and its OpenSpec config must carry the rules, so that no OpenSpec skill file needs a local patch for them.

## What Changes

- Rewrite every enhancement citation in tracked files (Go and CUE comments, the embedded render template, `testdata/` comments, test messages, live specs, archived changes, Markdown) with `enh cite-rewrite`, then clear its worklist by hand.
- `parity_compare_unit_test.go` asserts on citation text: the producer strings and the assertions change together.
- `openspec/config.yaml`: add `references:` for the store `opm-enhancements`; add the `Delivers` line to `rules.proposal`; add a `context` line on how to read an entry; replace the `NNNN:Dn` comment bullets by the new comment rule; remove every `enhancement.yaml` and `task enhancements:delivery:*` rule.
- `AGENTS.md`: same rules; remove the `enhancement.yaml` text and the description of the `enhancements/` folder that no longer exists.
- Remove the dead tooling `.tasks/enhancements.yaml` and its include in `Taskfile.yml`.
- The active change `publish-go-api-bundle` is not edited.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. Live specs change in wording only (citations), so the change has no spec delta and `.openspec.yaml` sets `skip_specs: true`.

## Impact

**SemVer: no release.** Only comments, strings in tests, text and tooling files change. Every commit is `docs`, `test` or `chore`. No behaviour changes; `task check` must pass before and after.

**Overlap.** `publish-go-api-bundle` edits Go doc comments in the same packages. Where the two touch one line, the later merge resolves it by hand.

**Repo-local patches.** Edits under `.claude/` are outside this change. The owner removes the `REPO-LOCAL PATCH` blocks with `openspec update`; the report lists them.

## Enhancement

None.
