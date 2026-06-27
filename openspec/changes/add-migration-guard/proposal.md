## Why

Breaking changes to the library's public API ship without a guaranteed migration recipe. `MIGRATIONS.md` is hand-maintained, so it drifts: entries get forgotten, duplicated (two `## Unreleased` sections recently), or left stale after a later change supersedes them. There is no automated link between "a breaking change landed" and "its migration recipe exists." release-please already detects breaking changes (from Conventional Commits) to compute the version bump — we can reuse that exact signal to **block a release (and a PR) whose breaking changes are not documented in `MIGRATIONS.md`**.

The authoring agent/human on a working branch cannot know the next version number (release-please derives it later), so the mechanism must key off **change impact**, not version.

## What Changes

- **MIGRATIONS bucket scheme → impact-based, version-agnostic** (already landed as `docs(migrations)` groundwork): `## Unreleased — Breaking` / `## Unreleased — Additive`, plus a "How this file is maintained" note. Entries are `### <Type> — \`<slug>\``.
- **`Migration: <slug>` commit trailer** — the correlation key shared end-to-end across the OpenSpec change dir, the breaking commit, and the `MIGRATIONS.md` entry. `Migration: none — <reason>` is the escape hatch for breaking-in-name changes that need no consumer action.
- **`migration-guard` (PR-time check, primary)** — on each PR, if it introduces a breaking change (`type!:` / `BREAKING CHANGE:`), require a `Migration: <slug>` trailer **and** a matching `### … — \`<slug>\`` entry under `## Unreleased — Breaking` in the diff. Block with an actionable message otherwise.
- **`migration-release-guard` (release-time check, backstop)** — on the release-please PR, re-derive the breaking set for the release range, exclude release-please's own commits, and assert every breaking commit's `Migration: <slug>` has a live `Breaking`-bucket entry.
- **Squash-merge enforced** — repo settings disable merge/rebase commits and set `squash_merge_commit_message=PR_BODY` so the `Migration:` trailer (placed in the PR description) lands on the squashed `main` commit, where the release-time check reads it. Branch protection requires linear history + the `migration-guard` check.
- **PR template** scaffolding the `Migration:` line so authors/agents don't forget it.
- **ADR** capturing the decision and the correlation-key rationale.

Out of scope (this iteration): auto-**graduation** of `Unreleased` entries into a versioned section at release (deferred — the gate is blocking, so a human/agent fixes the branch); applying the same gate to `cli` / `opm-operator` / `core` (prove it in `library` first, template later); non-breaking (`feat:`) coverage (breaking-only).

## Capabilities

### New Capabilities
- `migration-guard`: the CI contract that ties breaking changes to `MIGRATIONS.md` entries via the `Migration: <slug>` trailer — breaking-change detection, the PR-time gate, the release-time backstop, the squash/trailer plumbing it depends on, and the impact-based MIGRATIONS structure it reads.

### Modified Capabilities
<!-- none — this is repo tooling + process, not a change to the kernel's runtime behavior -->

## Impact

- **New files:** `.github/workflows/migration-guard.yml` (PR-time), `.github/workflows/migration-release-guard.yml` (release-time) or a job added to the existing release-please workflow, a guard script (`.github/scripts/` — Go or a small Node/bash using a real Conventional-Commits parser), `.github/pull_request_template.md`, `adr/NNN-migration-guard.md`.
- **Repo settings (outward-facing, applied by a maintainer):** squash-only merges, `squash_merge_commit_message=PR_BODY`, branch protection (linear history + required `migration-guard` check, no direct pushes to `main`).
- **`MIGRATIONS.md`:** structure already updated (53e529a); the maintenance note is finalized to match this design.
- **Process:** every breaking PR now carries a `Migration: <slug>` trailer and a MIGRATIONS entry. Mostly AI-authored, so the convention is easy to script into agents.
- **No runtime/library-code impact** — this is CI + docs only. `task check` is unaffected.
- **SemVer:** none (no `opm/` API change).
- **Soft dependency:** the impact-based bucket names land with the `rename-release-to-instance` branch's MIGRATIONS work; this change builds on that state.
