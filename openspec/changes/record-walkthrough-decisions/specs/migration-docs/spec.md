## MODIFIED Requirements

### Requirement: Enforcement design is recorded and armed at GA

The GA-time enforcement design SHALL be recorded in ADR-004 and summarized in
`migrations/README.md` before GA: a `Migration: <slug>` trailer on breaking PRs (escape
hatch `Migration: none — <reason>`), a PR-time gate requiring
`migrations/unreleased/<slug>.md` in the diff of a breaking PR, a release-time backstop that
re-derives the breaking set and performs the graduation move, and a `migration` skill that
authors fragments. These mechanisms SHALL NOT run pre-GA. ADR-004, `README.md` and
`migrations/README.md` SHALL name 0021:D8 as the GA exit criteria that carry the arming
condition, and SHALL NOT refer to a GA release checklist.

#### Scenario: Arming condition is written down

- **WHEN** the GA release is prepared
- **THEN** ADR-004 and `migrations/README.md` name the gates, the trailer convention, and
  the graduation job that must be implemented and enabled as part of GA

#### Scenario: No gate runs pre-GA

- **WHEN** a PR is opened while the library is pre-GA
- **THEN** no migration-guard check is required on it

#### Scenario: The GA exit criteria are linked

- **WHEN** a developer reads ADR-004, `README.md` § SemVer or `migrations/README.md` § Status
- **THEN** each names 0021:D8 as the GA exit criteria, and `README.md` names the library's
  own criteria `0021:D8:R11` to `0021:D8:R14`
- **AND** none refers to a GA release checklist
