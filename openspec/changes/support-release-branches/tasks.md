# Tasks: support-release-branches

Worktree `library/.claude/worktrees/tags-guard`, branch `tags/guard-release-immutability`
(from `origin/main` at `02344e5`). The branch already carries this plan. Hidden commit
types only (`ci`, `chore`): `docs` releases in this repo. One PR, title
`ci(release): support release/vX.Y maintenance branches`. No task creates, moves or
deletes a tag, a release or a `release/*` branch.

Apply waits for: workspace root rule "Release Tags Are Immutable" and enhancement 0021 D10
merged (section 3 and the spec citations), and `open-platform-model/.github`
`cut-release-branch.yml` merged on its `main` (section 2).

## 1. Release workflow serves release branches (.github/workflows/release.yml)

- [ ] 1.1 Change the push trigger to `branches: [main, 'release/**']` and add
      `target-branch: ${{ github.ref_name }}` to the release-please step's `with:`. Update
      the step comment: on a `release/vX.Y` branch release-please tags patches from that
      branch; the branch-local settings come from the PR the cut workflow opens. Verify:
      `yq '.on.push.branches' .github/workflows/release.yml` prints `main` and
      `release/**`; `yq '.jobs.release-please.steps[] | select(.name == "Run release-please")
      | .with."target-branch"' .github/workflows/release.yml` prints
      `${{ github.ref_name }}`.
- [ ] 1.2 Lint: `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7
      .github/workflows/release.yml` reports nothing.
- [ ] 1.3 `task check` green, then commit
      `ci(release): run release-please on release branches`.

## 2. Cut-release-branch dispatch (.github/workflows/cut-release-branch.yml)

- [ ] 2.1 Read the merged `open-platform-model/.github` `.github/workflows/cut-release-branch.yml`
      on its `main`: record its full commit SHA, its `workflow_call` input names (tag
      prefix, minor, package path), the secrets it declares and the permissions it needs.
      Stop and report if any differs from design.md "Thin caller" in a way the caller
      cannot absorb.
- [ ] 2.2 Write the caller per design.md: name `Cut release branch`, `workflow_dispatch`
      with required string input `minor`, top-level `permissions: contents: write,
      pull-requests: write`, one job `cut` that `uses:` the shared workflow pinned by the
      SHA from 2.1 with a version or ref comment, `with:` tag prefix `v`, `minor`, package
      path `.`, and the secrets it declares (`secrets: inherit` only if it declares none
      explicitly). No `run:` step. Verify: actionlint as in 1.2 on this file reports
      nothing; `grep -c 'run:' .github/workflows/cut-release-branch.yml` prints 0.
- [ ] 2.3 `task check` green, then commit
      `ci(release): add the cut-release-branch dispatch`.

## 3. AGENTS.md pointer (AGENTS.md)

- [ ] 3.1 Add one bullet under `## Repository Rules`: release tags are immutable per the
      workspace root `AGENTS.md` section "Release Tags Are Immutable"; the library's
      recovery is the next `-beta.N` (from GA, the next patch) with a `retract` for the bad
      version in `go.mod`; maintenance lines are `release/vX.Y` branches cut only by the
      `Cut release branch` dispatch, never deleted, changed only by PR (backports
      included); during beta no branch is cut and fixes land on `main`. Verify: one bullet
      added and no other line changed (`git diff --stat` shows only `AGENTS.md`), no
      at-sign and no em-dash in it.
- [ ] 3.2 `task check` green, then commit
      `chore(agents): point at the immutable release tag rule`.

## 4. Archive

- [ ] 4.1 Re-check `enhancement.yaml` and the spec `Source:` lines against 0021 as merged
      (decision id, requirement numbers). `openspec validate support-release-branches
      --strict` passes, then `openspec archive support-release-branches --yes`. Verify:
      `openspec/specs/release-branches/spec.md` exists with the Purpose from the delta (no
      `TBD` placeholder), and `openspec validate --specs --strict` passes. From the
      workspace root, `task enhancements:delivery:log FROM=<archived-change-path>
      SUMMARY="library release workflow serves release/vX.Y branches; cut dispatch added"`
      prints `logged` for 0021.
- [ ] 4.2 `task check` green, then commit
      `chore(openspec): archive support-release-branches`.
