# Tasks — restructure-migration-docs

> Docs + process only; no `opm/` code. Reworked from `add-migration-guard` (its bucket
> restructure, task 1.1 / 53e529a, is superseded by the deletion below; its guard design
> carries forward via ADR-004).

## 1. New structure

- [x] 1.1 `migrations/README.md` — policy: pre-GA dormancy, `unreleased/` → `vX.Y.Z/`
      layout, fragment format, GA enforcement summary, generated-index placeholder.
- [x] 1.2 `adr/004-migration-docs-structure.md` — layout + deferred-enforcement decision,
      guard design carried forward from `add-migration-guard`.

## 2. Delete the monolith

- [x] 2.1 `MIGRATIONS.md` → tombstone pointing at `migrations/README.md` (all alpha entries
      deleted; git history retains them).
- [x] 2.2 Rescue non-migration content: warm-cache/air-gap pattern self-contained in
      `docs/getting-started.md` (dangling pointer dropped); closedness wrinkle pointer in
      `.claude/skills/verify/SKILL.md` retargeted at the
      `library-phase-and-values-prune` archive entry.

## 3. Reference updates

- [x] 3.1 `CLAUDE.md`: entrypoint list, repository layout, working-style bullet.
- [x] 3.2 `README.md`: schema-resolution section, further-reading list.

## 4. Verify

- [x] 4.1 `openspec validate restructure-migration-docs --strict`.
- [x] 4.2 `grep -rn "MIGRATIONS" --include="*.md"` over repo docs/skills: remaining hits are
      the tombstone itself, `CHANGELOG.md`, ADR-003/ADR-004, and historical OpenSpec archive
      entries only; Go comment pointers in `opm/helper/doc.go`, `opm/schema/loader_test.go`,
      `opm/helper/synth/instance_integration_test.go` retargeted or dropped.

## 5. Deferred to the GA-time change (against ADR-004 — not tasks here)

- Guard script + `migration-guard.yml` (PR-time, file-existence check) and the release-time
  backstop with the graduation `git mv` + index regeneration.
- Repo settings (squash-only, `PR_BODY`, branch protection) and PR template.
- `migration` skill that authors `migrations/unreleased/<slug>.md`.
