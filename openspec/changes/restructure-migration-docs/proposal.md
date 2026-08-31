## Why

`MIGRATIONS.md` grew to a 1244-line monolith in which every entry still sat under the two
`## Unreleased` buckets: graduation never happened, entries drifted and duplicated, and every
breaking PR paid an editing tax on one contended file. During alpha the recipes also have no
audience: both consumers (`cli`, `opm-operator`) live in this workspace and migrate in the
same PR wave, and `CHANGELOG.md` plus `openspec/changes/archive/` already record every break.

This change reworks its earlier iteration (`add-migration-guard`), which designed CI gates
around the monolith. The correlation-key and plumbing decisions survive (the
`Migration: <slug>` trailer, squash/`PR_BODY`, PR-time-primary/release-time-backstop); the
file layout they targeted does not.

## What Changes

- **Delete every `MIGRATIONS.md` entry** (all alpha-to-alpha); the file becomes a short
  tombstone pointing at `migrations/README.md`. Git history retains the old text.
- **New `migrations/` directory** holding the written policy (`README.md`): one fragment per
  breaking change at `migrations/unreleased/<slug>.md`, graduated by release automation to
  `migrations/vX.Y.Z/<slug>.md` with a regenerated index, correlated by the
  `Migration: <slug>` trailer. **Dormant**: fragments and CI enforcement arm at the first GA
  release, not before.
- **ADR-004** records the layout, the deferred-enforcement decision, and the guard design
  carried forward from `add-migration-guard` retargeted at per-file fragments.
- **Rescue the two non-migration docs** that lived in `MIGRATIONS.md`: the warm-cache /
  air-gap deployment pattern (now self-contained in `docs/getting-started.md`) and the
  `ProcessModuleInstance` closedness wrinkle (already inline in the `verify` skill; its
  pointer now targets the OpenSpec archive entry).
- **Update every `MIGRATIONS.md` reference**: `CLAUDE.md` (entrypoint list, layout, working
  style), `README.md` (schema section, further reading), `docs/getting-started.md`,
  `.claude/skills/verify/SKILL.md`.

Out of scope: implementing the gates, the graduation job, and the `migration` skill (a
GA-time change built against ADR-004); rolling the convention out to `cli` / `opm-operator` /
`core`.

## Capabilities

### New Capabilities

- `migration-docs`: the migration-documentation policy — where fragments live, how they
  graduate at release, how they correlate to breaking commits, and when enforcement arms.

### Modified Capabilities

<!-- none — repo docs + process, no change to the kernel's runtime behavior -->

## Impact

- **Deleted content:** all `MIGRATIONS.md` entries (file kept as tombstone).
- **New files:** `migrations/README.md`, `adr/004-migration-docs-structure.md`.
- **Edited files:** `CLAUDE.md`, `README.md`, `docs/getting-started.md`,
  `.claude/skills/verify/SKILL.md`.
- **No runtime/library-code impact**; `task check` unaffected. **SemVer:** none.
- **Deferred (GA-time change):** guard workflows/scripts, repo squash/branch-protection
  settings, graduation job, `migration` skill, PR template.
