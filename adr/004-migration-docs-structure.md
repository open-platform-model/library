# ADR-004: Per-change migration fragments, dormant until GA

## Status

Accepted

## Context

`MIGRATIONS.md` was the single hand-written home for every breaking-change migration recipe.
By 2026-08-31 it had grown to 1244 lines in which every entry still sat in the two
`## Unreleased` buckets: graduation into versioned sections never happened, entries drifted
and duplicated, and each breaking PR paid an editing tax on one contended file. During the
alpha line the recipes also have no audience: both consumers (`cli`, `opm-operator`) live in
this workspace and migrate in the same coordinated PR wave, and every break is already
recorded by release-please in `CHANGELOG.md` and, with design context, in
`openspec/changes/archive/`.

An earlier design (the `add-migration-guard` OpenSpec change) kept the monolith and added CI
gates around it: a `Migration: <slug>` trailer on breaking PRs, a PR-time check that the
trailer has a matching entry, and a release-time backstop. Its correlation-key and plumbing
decisions remain sound; its file layout does not.

Two constraints shape any layout:

- The author of a breaking change cannot know the next version number. release-please
  computes it from the accumulated Conventional Commits at release time, so version-named
  files or headers cannot be authored in the breaking PR.
- Sequence-number prefixes (`NNNN-<slug>.md`) collide: two parallel PRs claim the same
  number and one rebases.

## Decision

Migration documentation moves from the `MIGRATIONS.md` monolith to per-change fragments
under `migrations/`, and enforcement is deferred until the first GA release.

- **Layout:** `migrations/unreleased/<slug>.md` is authored in the breaking PR, where
  `<slug>` is the OpenSpec change slug. At release, automation graduates fragments with
  `git mv migrations/unreleased/* migrations/vX.Y.Z/` and regenerates the index in
  `migrations/README.md`. Version information therefore lives in the directory name,
  stamped by the one actor that knows the version. Per-file moves make graduation a
  trivial scripted step, unlike rewriting sections of a monolith, which is why the
  earlier design had to defer it.
- **Correlation (carried forward from `add-migration-guard`):** one slug flows through the
  OpenSpec change directory, the breaking commit's `Migration: <slug>` trailer (placed in
  the PR body; squash-only merges with `squash_merge_commit_message=PR_BODY` land it on
  `main`), and the fragment filename. `Migration: none — <reason>` stays the escape hatch.
  Breaking detection uses a real Conventional Commits parser, the same signal
  release-please uses.
- **Enforcement (armed at GA, not before):** a required PR-time check (breaking commit
  implies `migrations/unreleased/<slug>.md` present in the diff; a file-existence check,
  simpler than the earlier design's monolith-diff parsing) plus a release-time backstop on
  the release-please PR that re-derives the breaking set, verifies coverage, and performs
  the graduation move. A `migration` skill authors fragments alongside the change.
- **Pre-GA policy:** no fragments are written. The alpha-era `MIGRATIONS.md` entries were
  deleted (git history retains them); the file remains as a short tombstone pointing at
  `migrations/README.md`.

Alternatives rejected: keeping the gated monolith (the tedium and contention this ADR
removes); fragments inside each OpenSpec change directory (archived out of user sight with
the change); sequence-numbered filenames (parallel-PR collisions); version-named files at
authoring time (the author cannot know the version).

## Consequences

**Positive:** breaking PRs touch one new file instead of editing a contended monolith;
graduation becomes a cheap scripted move; the gate check simplifies to file existence;
alpha development carries zero migration-doc overhead while `CHANGELOG.md` and the OpenSpec
archive keep the record.

**Negative:** until GA there is no single consumer-facing migration document at all; an
external early adopter must read `CHANGELOG.md` and the OpenSpec archive. Accepted: the
library is explicitly pre-GA.

**Trade-off:** deferring enforcement means the discipline must be re-established at GA
rather than maintained continuously. The arming condition is recorded here and in
`migrations/README.md` so the GA release checklist picks it up; the GA-time change
implements the gates, the graduation job, and the skill against this ADR.
