## Why

OpenSpec 1.14.1 makes the over-500-character requirement check a WARNING. `--strict` fails on any warning, and no flag turns the check off. The repo moves to 1.14.1 now so its generated skills match the tool that every other repo uses. A later change rewrites the long requirements of this repo and adds a 1.14.1 strict gate.

## What Changes

- Regenerate the `openspec-*` skills in `.claude/skills/` with 1.14.1 (`generatedBy: "1.14.1"`).
- Keep the one local edit of `openspec-verify-change` (Cleanliness dimension, Task Hygiene, Namingscheme), merged with a 3-way `git merge-file` against pristine 1.12.0.
- Leave `openspec-cleanup-change` (hand-written) and `openspec/config.yaml` untouched.
- This repo has no OpenSpec CLI pin and no CLI gate. This change adds none.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. The change has no spec delta (`skip_specs: true`).

## Impact

- Files: `.claude/skills/openspec-*/SKILL.md` only.
- No code, no schema, no spec text. Agent instruction files change, so the diff is reviewed as code and scanned for hidden Unicode.

## Enhancement

None.
