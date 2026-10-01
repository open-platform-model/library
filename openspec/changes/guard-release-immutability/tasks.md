# Tasks: guard-release-immutability

Worktree `library/.claude/worktrees/tags-guard`, branch `tags/guard-release-immutability`
(from `origin/main` at `02344e5`). The branch already carries this plan as
`chore(openspec): plan guard-release-immutability`. Hidden commit types only (`ci`,
`chore`): `docs` releases in this repo. One PR, title
`ci(release): assert the release tag names the release commit`. No task creates, moves or
deletes a tag or a release.

## 1. Tag-SHA assertion in the release workflow (.github/workflows/release.yml)

- [ ] 1.1 Give the release-please step `id: release`. Verify: `grep -n 'id: release'
      .github/workflows/release.yml` hits the release-please step.
- [ ] 1.2 Add the step `Assert the release tag names the release commit` after
      release-please, guarded by `if: ${{ steps.release.outputs.release_created }}`, with
      `env: TAG: ${{ steps.release.outputs.tag_name }}` and
      `WANT: ${{ steps.release.outputs.sha }}`, running the design.md D2 script under
      `shell: bash` with `set -euo pipefail` (no `${{ }}` inside `run:`). A short comment
      above it says why (an existing tag is adopted by release creation whatever commit it
      names) and that the step only reads. Verify: the workflow parses
      (`python3 -c 'import yaml,sys; yaml.safe_load(open(".github/workflows/release.yml"))'`,
      plus `actionlint` when installed).
- [ ] 1.3 Exercise the exact `run:` body locally (copy it into a scratchpad script, set
      `GITHUB_REPOSITORY=open-platform-model/library`). Verify all four:
      `TAG=v1.0.0-beta.1 WANT=02344e5913458e4411a19668f0f4424b8141d2f4` exits 0;
      the same tag with `WANT=` any other SHA exits 1 naming both SHAs;
      `TAG=v0.0.0-does-not-exist` exits 1 with the missing-tag error; empty `WANT` exits 1.
      Then, with `GITHUB_REPOSITORY` pointed at a public repo whose tags are annotated
      (`git/git` `v2.46.0`: tag object `75a062f`, peeled `39bf06a`, checked 2026-10-01),
      `WANT=` the peeled SHA exits 0 and `WANT=` the tag-object SHA exits 1. Read-only
      throughout; nothing is pushed.
- [ ] 1.4 `task check` green, then commit
      `ci(release): assert the release tag names the release commit`.

## 2. AGENTS.md pointer (AGENTS.md)

- [ ] 2.1 Add one bullet under `## Repository Rules`: release tags are immutable per the
      workspace root `AGENTS.md` rule (cite its heading exactly as it reads on the root at
      apply time); the library's recovery is the next `-beta.N` (from GA, the next patch)
      with a `retract` for the bad version in `go.mod`; the release workflow fails red when
      a tag does not name the release commit. Verify: one bullet added, no other line
      changed (`git diff --stat` shows `AGENTS.md | 1 +` or 2 with wrap), no bare at-sign
      and no em-dash in it.
- [ ] 2.2 `task check` green, then commit
      `chore(agents): point at the immutable release tag rule`.

## 3. Archive

- [ ] 3.1 `openspec validate guard-release-immutability --strict` passes, then
      `openspec archive guard-release-immutability --yes`. Verify:
      `openspec/specs/release-tag-integrity/spec.md` exists with the Purpose from the delta
      (no `TBD` placeholder), and `openspec validate --specs --strict` passes.
- [ ] 3.2 `task check` green, then commit
      `chore(openspec): archive guard-release-immutability`.
