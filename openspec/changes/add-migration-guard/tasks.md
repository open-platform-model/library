# Tasks — add-migration-guard

> CI + docs only; no `opm/` code. Build the PR-time gate first (where reliability lives), then settings, then the release-time backstop. Branched off the `rename-release-to-instance` MIGRATIONS state.

## 1. Convention + docs

- [x] 1.1 MIGRATIONS buckets → impact-based `## Unreleased — Breaking` / `## Unreleased — Additive` + maintenance note. *(landed in `53e529a`)*
- [ ] 1.2 ADR `adr/NNN-migration-guard.md` — record D1–D6: breaking-detection signal, the `Migration: <slug>` trailer + `none` escape hatch, squash/`PR_BODY` plumbing, PR-time-primary/release-time-backstop, the parser choice. Include the exact `gh api` settings commands.
- [ ] 1.3 Finalize the MIGRATIONS "How this file is maintained" note to match the ADR wording (the `Migration:` trailer + graduate-at-release reference).
- [ ] 1.4 `.github/pull_request_template.md` scaffolding a `Migration: <slug>` (or `Migration: none — <reason>`) line with a one-line how-to.

## 2. PR-time gate (primary)

- [ ] 2.1 Guard script (`.github/scripts/migration-guard.*`) using a real Conventional-Commits parser: input = PR title + body + the `MIGRATIONS.md` diff; output = pass/fail + actionable message. Decide Go vs Node (lean Node, reuse `conventional-commits-parser`).
- [ ] 2.2 Detect breaking via subject `type(scope)!:` or body `BREAKING CHANGE:`/`BREAKING-CHANGE:` (multiline).
- [ ] 2.3 If breaking: require `Migration: <slug>` (collect multiple / comma-list) or `Migration: none — <reason>`; missing → fail.
- [ ] 2.4 For each `<slug>`: assert the PR diff adds `### … — \`<slug>\`` under `## Unreleased — Breaking`; mismatch → fail with the slug + fix.
- [ ] 2.5 `.github/workflows/migration-guard.yml` on `pull_request` (+ `pull_request_target` considerations for the body); wire the script.
- [ ] 2.6 Unit-test the script against fixtures: breaking+covered (pass), breaking+no-trailer (fail), breaking+trailer+no-entry (fail), `none` (pass), non-breaking (pass), multi-slug (pass).

## 3. Repo settings (maintainer-applied, outward-facing)

- [ ] 3.1 Squash-only + PR_BODY: `gh api -X PATCH repos/open-platform-model/library -F allow_squash_merge=true -F allow_merge_commit=false -F allow_rebase_merge=false -F squash_merge_commit_title=PR_TITLE -F squash_merge_commit_message=PR_BODY` (document; a maintainer runs it).
- [ ] 3.2 Branch protection on `main`: require linear history, require the `migration-guard` status check, block direct pushes. (Document the `gh api` / UI steps.)
- [ ] 3.3 Verify end-to-end on a throwaway breaking PR: trailer in description survives squash to `main` (`git log` shows `Migration:`).

## 4. Release-time gate (backstop)

- [ ] 4.1 Release guard script: range = `$(jq -r '."."' .release-please-manifest.json)`-tag `..HEAD`; exclude release-please bot commits + `^chore\(main\): release`.
- [ ] 4.2 `B = {slug from each breaking commit trailer}` (skip `none`); breaking commit with no trailer → fail.
- [ ] 4.3 `M = {slugs under ## Unreleased — Breaking}`; require `B ⊆ M`; uncovered → fail naming slug + commit.
- [ ] 4.4 Wire as a job on the release-please PR's checks (blocks the merge that tags) — `.github/workflows/migration-release-guard.yml` or a job in the release workflow.
- [ ] 4.5 Fixture tests: covered range (pass), uncovered (fail), release-please commit ignored.

## 5. Verify & document

- [ ] 5.1 `openspec validate add-migration-guard --strict`; `openspec-verify-change`.
- [ ] 5.2 README/CONTRIBUTING note: "breaking change → add a `Migration: <slug>` trailer + a `MIGRATIONS.md` entry"; link the ADR.
- [ ] 5.3 Open the PR; squash-merge it (dogfood the gate on itself — it touches no `opm/` API so it's non-breaking and should pass cleanly).

## Out of scope (tracked separately)

- [ ] Auto-graduation of `Unreleased` entries into a versioned `## X.Y.Z` section at release time.
- [ ] Templating the gate to `cli` / `opm-operator` / `core`.
