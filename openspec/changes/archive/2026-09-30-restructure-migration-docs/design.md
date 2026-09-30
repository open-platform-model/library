## Context

`MIGRATIONS.md` was the single hand-written home for migration recipes; `CHANGELOG.md`
(release-please) owns per-version summaries. The monolith never graduated its `Unreleased`
buckets, and during alpha its recipes had no audience (in-workspace consumers migrate in the
same PR wave). The earlier iteration of this change (`add-migration-guard`) settled the
enforcement mechanics: `Migration: <slug>` trailer as the correlation key, squash-only merges
with `PR_BODY` so the trailer reaches `main`, a PR-time gate as primary and a release-time
backstop, detection via a real Conventional Commits parser. Those decisions carry forward;
the monolithic file they targeted is replaced.

Two constraints bind the layout: the author of a breaking change cannot know the next
version (release-please computes it at release), and sequence-number prefixes collide across
parallel PRs.

## Goals / Non-Goals

**Goals**

- Remove the per-PR editing tax and drift of a single contended migrations file.
- Zero migration-doc overhead during alpha, with the record intact elsewhere.
- A layout whose listing reads in release order and whose version labels are stamped by the
  one actor that knows the version.
- Preserve the still-valid guard design for GA arming.

**Non-Goals**

- Building the gates, graduation job, PR template, or `migration` skill now (GA-time change
  against ADR-004).
- Covering non-breaking changes (fragments allow `impact: additive` but nothing requires
  them).
- Rolling out to `cli` / `opm-operator` / `core`.

## Decisions

### D1 — Per-change fragments, not a monolith

One file per breaking change at `migrations/<bucket>/<slug>.md`, `<slug>` = the OpenSpec
change slug. Removes merge contention, makes the future gate a file-existence check (the
earlier design had to parse a monolith diff for `### … — \`<slug>\`` headers), and makes
graduation a per-file move.

### D2 — Version-free authoring; version directories stamped at release

Fragments are authored under `migrations/unreleased/`; the release job runs
`git mv migrations/unreleased/* migrations/vX.Y.Z/` and regenerates the index in
`migrations/README.md`. The version lives in the directory name, written at the moment it
becomes knowable. Rejected: version-named files at authoring time (unknowable),
`NNNN-` sequence prefixes (parallel-PR collisions), relying on raw directory listing for
order (SemVer does not sort lexically; the generated index owns ordering).

### D3 — Dormant until GA

Pre-GA, no fragments are written and no gate runs. `CHANGELOG.md` and
`openspec/changes/archive/` carry the record; the alpha entries in `MIGRATIONS.md` are
deleted (git history retains them). Arming is part of the GA release work, recorded in
ADR-004 and `migrations/README.md` so it cannot be silently forgotten.

### D4 — Guard design carried forward, retargeted

Unchanged from the earlier iteration: breaking detection by Conventional Commits parser
(subject `!` or `BREAKING CHANGE:` footer), `Migration: <slug>` trailer in the PR body,
`Migration: none — <reason>` escape hatch, squash-only + `PR_BODY` plumbing, PR-time primary
gate + release-time backstop. Retargeted: the gate asserts
`migrations/unreleased/<slug>.md` exists in the diff; the backstop also performs the D2
graduation move (feasible now precisely because it is a `git mv`, which is why the earlier
design deferred it).

### D5 — Tombstone, not deletion, for `MIGRATIONS.md`

A short pointer file remains at the old path: external links and muscle memory survive at
the cost of one small file.

## Risks / Trade-offs

- **[No consumer-facing migration doc until GA]** → an external early adopter reads
  `CHANGELOG.md` + the OpenSpec archive. Accepted; the library is explicitly pre-GA.
- **[Re-establishing discipline at GA is a cliff]** → mitigated by recording the arming
  condition in ADR-004 and `migrations/README.md`, and by the GA-time change shipping the
  skill that authors fragments so the habit is tool-assisted from day one.
- **[Dormant structure drifts before GA]** → the policy README is small and layout-only;
  the GA-time change revalidates it when building the gates.

## Migration Plan

1. `migrations/README.md` (policy) + ADR-004.
2. `MIGRATIONS.md` → tombstone; reference updates across repo docs and the verify skill.
3. (GA-time, separate change) gates, repo settings, graduation job, PR template,
   `migration` skill.

**Rollback:** docs-only; revert the commit.

## Open Questions

- None blocking. The GA-time change re-opens the earlier iteration's script-language
  question (Go vs Node with `conventional-commits-parser`) when it builds the gates.
