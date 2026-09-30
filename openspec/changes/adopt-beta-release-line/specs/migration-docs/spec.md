## MODIFIED Requirements

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
