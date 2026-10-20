## Context

State on `origin/main` (`cd8d7d7d`, OpenSpec 1.14.1): `openspec validate --all --strict --no-interactive` exits 1 with one WARNING, in `change/publish-go-api-bundle`. All 26 main specs pass. The measured body of the block (T2.1 rule: trimmed lines joined by newlines) is 554 characters.

`core` already has the gate: `OPENSPEC_VERSION` in its Taskfile, tasks `openspec:install` and `openspec:check`, and CI steps in `ci.yml`. The library has `lint.yml` and `test.yml` for pull requests, and no `ci.yml`. The library has no earlier change with a spec-less CI scope except `wire-cite-lint` (archived 2026-10-12): it has `proposal.md` and `tasks.md`, no `specs/` folder and `skip_specs: true` in `.openspec.yaml`. This change does the same.

## Goals / Non-Goals

**Goals**

- `openspec validate --all --strict --no-interactive` under 1.14.1 exits 0 on the branch.
- A pinned gate runs it in `task check` and in CI on every pull request.

**Non-Goals**

- Any change of meaning of the requirement or its scenarios; any change to the intent, design, tasks or code of `publish-go-api-bundle`; any Go or CUE code.
- A new spec. The gate is a CI rule; the repo records CI wiring in `lint.yml` and the Taskfile, as `cite-lint` is.

## Research & Decisions

### Split of the long ADDED block

**Context**: The block holds three sentences. Lengths: 301, 125 and 126 characters, (the sum with the joining spaces is 554).
**Explored**: Form b and c2 of the T2.1 guide do not apply: the block is ADDED in an open change, so no old scenario is live in a main spec. The block can be split freely into ADDED blocks, each with at least one scenario, as long as each sentence stays word for word and each scenario stays byte for byte.
**Decision**: Two ADDED blocks, in place of the old one, in the same position:

1. Keep the header "The library publishes one docs bundle". Body: sentence 1 only (301 characters). Scenario: "Internal packages stay out", because sentence 1 names the `go-api` source over `./opm/...`.
2. New header "The bundle carries both kinds of page and its tasks build it". Body: sentences 2 and 3 (about 252 characters), carrying the docs-kit DESIGN citation as in the old text. Scenario: "The bundle holds both kinds of page", because it runs `task docs:bundle` and reads pages of both kinds.

Three blocks (one per sentence) were rejected: the third sentence would have no scenario of the old text, and a new scenario may restate only a case the old text names. The cost is that block 2 holds two close rules (what the bundle carries; the tasks that build it). Reversibility: high; the delta is text in an open change.
**Rationale**: Each block is under 500, has SHALL, and has a scenario with a body. No word is added or dropped. The header of block 2 is new words; the header is a name, not a rule. The old header stays on the block that keeps the first rule.

The tasks do the edit by cutting the old body at the sentence boundary after "(docs-kit DESIGN decision 20)." and putting each sentence in its own block. They add no connecting words.

### Ledger

| Old scenario name | New place | Changed text? |
| --- | --- | --- |
| The bundle holds both kinds of page | The bundle carries both kinds of page and its tasks build it | No |
| Internal packages stay out | The library publishes one docs bundle | No |

| Old clause | New place |
| --- | --- |
| Sentence 1: declare the project `library` in `docs-kit.cue` with its placement, ownership, tag prefix and two sources | The library publishes one docs bundle, body |
| Sentence 2: the bundle carries the authored pages and the Go API reference (docs-kit DESIGN decision 20) | The bundle carries both kinds of page and its tasks build it, body |
| Sentence 3: `task docs:bundle` builds into `out/library/`; `task docs:bundle:check` builds and lints without publishing | Same block, body |

Edit of an open change (owner ruling of 2026-10-20): only `openspec/changes/publish-go-api-bundle/specs/docs-bundle/spec.md` changes. The apply phase confirms that `git diff` shows no other file of that change touched, and that a word-by-word comparison of the old block and the two new blocks differs only in the new header and the paragraph break.

### The gate: where it lives

**Context**: The CLI is npm code with no lockfile. A job that holds a write token must not run it.
**Decision**: Follow `core`: one version pin in the Taskfile (`OPENSPEC_VERSION`), `task openspec:install` and `task openspec:check`, each used by CI as a step, so the pin has one home. `task openspec:install` runs `npm install -g --ignore-scripts @fission-ai/openspec@{{.OPENSPEC_VERSION}}`. `task openspec:check` sets `OPENSPEC_TELEMETRY=0`, has a precondition that `openspec --version` equals the pin, and runs `openspec validate --all --strict --no-interactive`.
**Rationale**: The same pin and the same task serve a developer and CI.

### The gate: which workflow

**Context**: `release.yml` and `docs.yml` hold write permissions. `lint.yml` and `test.yml` run on pull requests and pushes to `main` with `permissions: contents: read`. `lint.yml` already carries a second job (`cite-lint`), so it hosts non-Go rules.
**Decision**: A new job `openspec` in `lint.yml`, with steps: checkout (`persist-credentials: false`, same pin as the file), Install Task (same pin), `actions/setup-node` at the SHA that `core` pins (`820762786026740c76f36085b0efc47a31fe5020`, v7.0.0, `node-version: '24'`), `task openspec:install`, `task openspec:check`. No `if:`, no path filter, no cache. `task check` also runs `openspec:check`; no job with a write token calls `task check` today (checked with grep over `.github/workflows`).
**Rationale**: A job, not a step in the Go lint job, so a failure names the gate. The job name is `OpenSpec`. Whether the repository ruleset must list it as a required check is a setting the owner changes; see the open question.

**Open question for the owner**: after the job exists, add its check name to the required checks of the ruleset for `main`. The change cannot do that.

### Order

The delta split is committed first. Only then are the Taskfile and the workflow added, so every commit passes the gate it adds. If `publish-go-api-bundle` merges first, its spec `docs-bundle` is on `main` as one long requirement and `--all --strict` fails there; the owner then chooses between landing this change first or a MODIFIED split of the main spec. This change assumes the owner lands it with or before `publish-go-api-bundle`.

### Reporting

`openspec validate add-openspec-strict-gate --no-interactive` is the proposal check. The gate itself is proved in apply by running `openspec validate --all --strict --no-interactive` on the branch (exit 0), and by a negative run: a scratch copy where a requirement body is over 500 characters makes `task openspec:check` exit 1.

## Risks / Trade-offs

- npm without a lockfile on pull requests: mitigated by `--ignore-scripts`, a read-only token, a pin of the version, and no write job. A bump of the pin is a manual edit of one line.
- A future OpenSpec version may add warnings: the pin makes the bump a deliberate change.
