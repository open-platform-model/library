# migration-docs Specification

## Purpose
Where the library records migration guidance for breaking changes: per-change fragments under
`migrations/`, dormant until GA, with the GA-time enforcement design recorded in ADR-004.

## Requirements

### Requirement: Migration documentation lives in per-change fragments under migrations/

Migration documentation SHALL live in the `migrations/` directory as one fragment per
breaking change, named `<slug>.md` where `<slug>` is the OpenSpec change slug. Fragments
SHALL be authored under `migrations/unreleased/` and SHALL NOT carry version numbers in
their names; at release, automation graduates them to `migrations/vX.Y.Z/` and regenerates
the index in `migrations/README.md`. The repository SHALL NOT maintain a monolithic
migrations document; `MIGRATIONS.md` remains only as a tombstone pointing at
`migrations/README.md`.

#### Scenario: Policy is discoverable at the directory root

- **WHEN** a contributor opens `migrations/README.md`
- **THEN** it states the fragment layout (`unreleased/` → `vX.Y.Z/`), the fragment format,
  the release-time graduation move, and the enforcement design with its arming condition

#### Scenario: The old monolith redirects

- **WHEN** a reader opens `MIGRATIONS.md`
- **THEN** it contains no migration entries and points at `migrations/README.md`

#### Scenario: No version numbers at authoring time

- **WHEN** a fragment is authored in a breaking PR
- **THEN** it is created as `migrations/unreleased/<slug>.md`, with no version in path or
  filename

### Requirement: Pre-GA dormancy

Until the library's first GA release, through its alpha and beta lines alike, no migration fragment SHALL be required for a breaking change. The breaking-change record pre-GA SHALL be `CHANGELOG.md` (release-please) plus the OpenSpec archive. From the first beta on, a breaking change SHALL land only as a `feat!` commit whose `BREAKING CHANGE:` footer is the migration note, so the CHANGELOG entry release-please generates from it carries the migration guidance for that beta; the library SHALL record this rule in an ADR. Repo guidance (`AGENTS.md`, `README.md`, `migrations/README.md`) SHALL state this policy rather than instruct authors to update a migrations document.

#### Scenario: Breaking change during alpha

- **WHEN** a breaking change lands while the library is pre-GA
- **THEN** no `migrations/` fragment is required, and the change is recorded via its Conventional Commit (CHANGELOG) and its OpenSpec change archive

#### Scenario: Repo guidance matches the policy

- **WHEN** a contributor follows `AGENTS.md`'s working-style guidance for a kernel-surface change pre-GA
- **THEN** it directs them to check downstream consumers and, on the beta line, to write the migration note as the commit's `BREAKING CHANGE:` footer, not to write a migration entry

#### Scenario: Breaking change during beta

- **WHEN** a breaking change lands while the library is on its beta line
- **THEN** no `migrations/` fragment is required, the commit is typed `feat!` and carries a `BREAKING CHANGE:` footer stating how consumers migrate, the release it lands in advances the `-beta.N` counter, and the CHANGELOG entry for that release shows the footer as the migration note

### Requirement: Enforcement design is recorded and armed at GA

The GA-time enforcement design SHALL be recorded in ADR-004 and summarized in
`migrations/README.md` before GA: a `Migration: <slug>` trailer on breaking PRs (escape
hatch `Migration: none — <reason>`), a PR-time gate requiring
`migrations/unreleased/<slug>.md` in the diff of a breaking PR, a release-time backstop that
re-derives the breaking set and performs the graduation move, and a `migration` skill that
authors fragments. These mechanisms SHALL NOT run pre-GA.

#### Scenario: Arming condition is written down

- **WHEN** the GA release is prepared
- **THEN** ADR-004 and `migrations/README.md` name the gates, the trailer convention, and
  the graduation job that must be implemented and enabled as part of GA

#### Scenario: No gate runs pre-GA

- **WHEN** a PR is opened while the library is pre-GA
- **THEN** no migration-guard check is required on it
