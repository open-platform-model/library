## Why

OpenSpec 1.14.1 fails `openspec validate --all --strict` on any warning, and "Requirement text is very long (>500 characters)" is a warning. Since PR 242 every main spec of the library passes. One item still fails: the open change `publish-go-api-bundle`, whose ADDED requirement "The library publishes one docs bundle" in `docs-bundle/spec.md` has a body of 554 characters. The library has no OpenSpec gate, so a long requirement can come back unseen.

## What Changes

- The `specs/` delta of `publish-go-api-bundle` splits that one ADDED block into two ADDED blocks that each state one behavior. Every sentence stays word for word and every scenario stays byte for byte. The intent, design, tasks and code of `publish-go-api-bundle` do not change. design.md records the edit and the sentence and scenario ledger.
- `Taskfile.yml` pins the CLI once as `OPENSPEC_VERSION: 1.14.1` and gets two tasks: `openspec:install` (`npm install -g --ignore-scripts @fission-ai/openspec@{{.OPENSPEC_VERSION}}`) and `openspec:check` (`openspec validate --all --strict --no-interactive`, which refuses to run under another version). `task check` runs `openspec:check`. This is the form that `core` uses.
- `.github/workflows/lint.yml` gets a job `openspec` that sets up Node, runs `task openspec:install` and then `task openspec:check`. The workflow runs on pull requests and on pushes to `main`, has `contents: read` only, and the job has no path filter.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. This change has no spec delta of its own, and `.openspec.yaml` sets `skip_specs: true`, as the archived CI-only change `wire-cite-lint` does. The edit of the `docs-bundle` delta belongs to `publish-go-api-bundle`; it merges with this change, and the archive of that change keeps the two blocks in the main spec.

## Impact

**SemVer: no release.** No `opm/` code changes. Commits are `docs(openspec)` and `ci(openspec)`, which release-please does not count as a release. No change is breaking, so no `Migration:` trailer is needed; if a trailer is wanted, the form is `Migration: none, spec text and CI only`.

**Risk:** the edit of the `docs-bundle` delta drops or changes a rule. The ledger in design.md and a byte comparison of the old and new blocks cover it. A second risk: the new job runs npm code, with no lockfile, on pull requests. The job token is read only, the install runs no script, and no write job calls `openspec:check` or `task check`.

**Order:** the gate is added after the delta is split, so every commit on the branch passes it. If the owner merges `publish-go-api-bundle` first, this branch needs a merge of `main` and the edit moves to the main spec `docs-bundle` (see design.md, Order).

## Enhancement

None.
