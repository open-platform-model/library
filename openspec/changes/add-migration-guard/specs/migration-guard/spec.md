## ADDED Requirements

### Requirement: Breaking-change detection from Conventional Commits

The guard SHALL classify a change as **breaking** using the same rule release-please uses: a Conventional Commit whose subject matches `^<type>(\(.+\))?!:` OR whose body contains a `BREAKING CHANGE:` / `BREAKING-CHANGE:` footer. Detection SHALL use a real Conventional-Commits parser, not a substring match, and SHALL read the full commit body (multiline footers).

#### Scenario: Subject bang is breaking

- **WHEN** a commit subject is `feat(module)!: rename Release artifact family to Instance`
- **THEN** the guard classifies the change as breaking

#### Scenario: Body footer is breaking

- **WHEN** a commit subject is `refactor(materialize): federate surfaces` and its body contains a `BREAKING CHANGE: MaterializedPlatform.Composed removed` footer
- **THEN** the guard classifies the change as breaking

#### Scenario: Non-breaking change is not flagged

- **WHEN** a commit is `feat(schema): add optional field` with no `!` and no `BREAKING CHANGE:` footer
- **THEN** the guard does not require a migration entry

### Requirement: Migration trailer convention

A breaking change SHALL carry a `Migration: <slug>` trailer, where `<slug>` is a kebab-case identifier shared with the OpenSpec change directory and the `MIGRATIONS.md` entry. A breaking change that needs no consumer action SHALL instead carry `Migration: none — <reason>`. A change MAY carry multiple `Migration:` trailers (or a comma-separated list) when it bundles several breaking changes.

#### Scenario: Slug trailer present

- **WHEN** a breaking PR's body contains `Migration: rename-release-to-instance`
- **THEN** the guard extracts the slug `rename-release-to-instance` for correlation

#### Scenario: Escape hatch

- **WHEN** a breaking PR's body contains `Migration: none — drops an unused exported symbol with no consumers`
- **THEN** the guard treats the breaking change as documented and does not require a `MIGRATIONS.md` entry

#### Scenario: Trailer missing on a breaking change

- **WHEN** a breaking PR has neither a `Migration: <slug>` nor a `Migration: none` trailer
- **THEN** the guard fails with a message naming the missing trailer and how to add it

### Requirement: PR-time gate blocks undocumented breaking changes

A required check SHALL run on every pull request. When the PR introduces a breaking change, the check SHALL require BOTH a `Migration: <slug>` trailer AND that the PR diff adds a `### <Type> — \`<slug>\`` entry under `## Unreleased — Breaking` in `MIGRATIONS.md` (unless the trailer is `Migration: none`). On any mismatch the check SHALL fail with an actionable message; otherwise it SHALL pass. A non-breaking PR SHALL pass without requiring a migration entry.

#### Scenario: Breaking PR with matching entry passes

- **WHEN** a breaking PR carries `Migration: foo-bar` and its diff adds `### Changed — \`foo-bar\`` under `## Unreleased — Breaking`
- **THEN** the PR-time check passes

#### Scenario: Breaking PR with trailer but no entry fails

- **WHEN** a breaking PR carries `Migration: foo-bar` but `MIGRATIONS.md` has no `\`foo-bar\`` entry under `## Unreleased — Breaking`
- **THEN** the PR-time check fails and names the missing entry

#### Scenario: Non-breaking PR is unaffected

- **WHEN** a PR contains only `fix:` / `feat:` / `chore:` changes
- **THEN** the PR-time check passes without a migration entry

### Requirement: Release-time gate is the backstop

A check SHALL run on the release-please release PR. It SHALL re-derive the breaking-change set for the release range (`<last released tag>..HEAD`), excluding commits authored by the release-please bot (and matching `^chore\(main\): release`). For every breaking commit it SHALL require a `Migration: <slug>` trailer whose slug appears as a `### … — \`<slug>\`` entry under `## Unreleased — Breaking`; `Migration: none` entries are exempt. Any uncovered breaking change SHALL block the release.

#### Scenario: Covered release passes

- **WHEN** every breaking commit since the last tag has a `Migration: <slug>` trailer with a live `Breaking`-bucket entry
- **THEN** the release-time check passes

#### Scenario: Uncovered breaking change blocks the release

- **WHEN** a breaking commit in the release range has a `Migration: <slug>` trailer but no matching `Breaking`-bucket entry
- **THEN** the release-time check fails and names the slug and the offending commit

#### Scenario: release-please's own commit is ignored

- **WHEN** the range includes a `chore(main): release 1.0.0` commit authored by the release-please bot
- **THEN** the guard does not treat it as a breaking change requiring an entry

### Requirement: Squash-merge plumbing preserves the trailer

The repository SHALL enforce squash-only merges and SHALL configure the squash commit message from the PR body (`squash_merge_commit_message=PR_BODY`, `squash_merge_commit_title=PR_TITLE`) so a `Migration:` trailer placed in the PR description reaches the squashed `main` commit. Branch protection SHALL require linear history and the `migration-guard` check, and SHALL block direct pushes to the default branch.

#### Scenario: Trailer survives squash to main

- **WHEN** a breaking PR with `Migration: foo-bar` in its description is squash-merged
- **THEN** the resulting `main` commit body contains `Migration: foo-bar`, readable by `git log`

#### Scenario: Non-squash merge methods are disabled

- **WHEN** a contributor opens the merge dropdown on a PR
- **THEN** only "Squash and merge" is available (merge commits and rebase are disabled)

### Requirement: MIGRATIONS structure is impact-based and version-agnostic

`MIGRATIONS.md` SHALL stage entries under two impact-based buckets, `## Unreleased — Breaking` and `## Unreleased — Additive`, never under version-numbered `Unreleased` headers. Each entry SHALL be `### <Type> — \`<slug>\``. A header note SHALL document the maintenance rules (bucket by impact, the `Migration:` trailer, graduate-at-release).

#### Scenario: Buckets are present and labelled by impact

- **WHEN** a contributor opens `MIGRATIONS.md`
- **THEN** the unreleased entries sit under `## Unreleased — Breaking` and `## Unreleased — Additive`, with no version number in those headers

#### Scenario: The guard reads the Breaking bucket for slugs

- **WHEN** the guard collects documented slugs
- **THEN** it parses the `### … — \`<slug>\`` headers under `## Unreleased — Breaking`
