# ADR-010: Beta migration notes live in the CHANGELOG footer

## Status

Accepted (2026-09-30). Implemented by `adopt-beta-release-line`.

## Context

[ADR-004](004-migration-docs-structure.md) moved migration documentation to per-change
fragments under `migrations/` and left that directory dormant until the first GA release:
no fragment is written before GA, and the enforcement gates arm at GA. It was written for the
alpha line, where both consumers (`cli`, `opm-operator`) live in this workspace and migrate in
the same coordinated PR wave, so a standing recipe document had no audience.

The library now leaves alpha. From its first beta (`v1.0.0-beta.1`) it is on the path to GA,
together with the other prerelease lines of the cutover (`opmodel.dev/core@v2`,
`opmodel.dev/catalogs/k8s@v1`, cli, opm-operator). The beta promise still allows a breaking
change, but only as a `feat!` commit whose `BREAKING CHANGE:` footer is the migration note;
the break advances the `-beta.N` counter and never moves the module path to a new major.
That promise needs a stated home for the migration note during beta. Two homes are
available: the `migrations/` fragments ADR-004 designed, or the commit footer that
release-please already renders into `CHANGELOG.md`.

The consumers have not changed: they still live in this workspace and still migrate in the
same PR wave. The one reader a beta break adds is an embedder outside the workspace, who
reads the release notes of the version they bump to.

## Decision

During beta, the migration note for a breaking change is the `BREAKING CHANGE:` footer of its
`feat!` commit, as release-please renders it into `CHANGELOG.md` and the GitHub release. The
footer states what broke and how a consumer migrates; it is written in the breaking PR and
must survive into the squash commit on `main`.

`migrations/` stays dormant through beta. No fragment is required or written, and the
PR-time and release-time gates of ADR-004 stay unarmed. GA, not beta, arms ADR-004: from the
first GA release the fragment layout and its enforcement apply as ADR-004 records them.

ADR-004 is not amended. It stays the record of the fragment design; this ADR records only
where the migration note lives until that design is armed.

Alternatives rejected: arming `migrations/` at the first beta (the consumers still migrate
in one PR wave, and the owner decision places the gate at GA); amending ADR-004 in place
(an accepted ADR is a record, not a living document); a separate hand-written beta migration
file (the monolith ADR-004 removed, reintroduced for one release line).

## Consequences

**Positive:** A beta break carries its migration note in the one place every consumer
already reads, the CHANGELOG entry and release notes of the version that introduced it, with
no extra file, gate or authoring step.

**Negative:** The note is only as good as the footer. A squash merge that drops or rewrites
the commit body loses it, and a body line that release-please cannot parse drops the whole
commit from the CHANGELOG (the squash-body hazard in `AGENTS.md`). The reviewer of a `feat!`
PR has to check the final squash message, not only the branch commits.

**Trade-off:** Footer notes are short and unstructured compared with ADR-004's fragment
format. That is accepted for beta, where breaks are expected to be few and announced; GA
restores the structured, enforced fragments for the stable line.
