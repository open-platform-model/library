## Why

The owner made release tags immutable across the open-platform-model repos that release
(rule of 2026-10-01, enhancement 0021 D10): no tag under `refs/tags/` is ever moved, deleted
or re-created, and a wrong release is fixed by releasing the next version. For the library
the tag is the whole release: it ships no binary and no image, and `go get` plus the Go
module proxy resolve and cache the tag itself.

The platform carries the guarantee, not this repo: org rulesets refuse tag update and
deletion with no bypass, and allow tag creation only to the `opm-release-please` App, so a
stale or hand-made tag cannot exist and a per-repo tag check would guard nothing. What the
repo still lacks is a way to release a fix for a minor that `main` has moved past. The owner
chose lazy maintenance branches named `release/vX.Y` (for the library, the first is
`release/v1.0`), cut from the newest `vX.Y.*` tag by one automated action, never deleted,
and changed only by PR. Today the release workflow runs only on `main` and release-please
always targets the default branch, so a release branch would never release.

No release branch is cut by this change. During beta every fix lands on `main` and rolls
forward; the first branch is cut at GA or when `main` starts work a released minor must not
get.

## What Changes

- The release workflow (`.github/workflows/release.yml`) also runs on pushes to
  `release/**` and passes the pushed branch to release-please as its target branch, so a
  release branch gets its own release PR and its own patch tags. Pushes to `main` behave
  exactly as today.
- A new `workflow_dispatch` workflow, `Cut release branch`
  (`.github/workflows/cut-release-branch.yml`), takes one input, the minor `X.Y`, and calls
  the org's reusable workflow `open-platform-model/.github` `cut-release-branch.yml` with tag
  prefix `v` and package path `.`. That workflow creates `release/vX.Y` from the highest
  `vX.Y.*` tag and opens a PR into the new branch with the branch-local release-please
  settings (`versioning: always-bump-patch`, `prerelease: false`). The library keeps no
  copy of that logic.
- `AGENTS.md` gains one line under Repository Rules pointing at the workspace rule on
  immutable release tags, the library's roll-forward recovery (the next `-beta.N`, from GA
  the next patch, with a `retract` in `go.mod` for the bad version), and the release-branch
  model (cut only by the dispatch, backports by PR).
- No Go code, no `opm/` surface and no change to `main`'s release configuration. The change
  lands as hidden commit types (`ci`, `chore`), so it cuts no release. SemVer: none.

## Capabilities

### New Capabilities

- `release-branches`: the release automation's support for `release/vX.Y` maintenance
  branches (release-please runs against the pushed branch; a branch is cut only by the
  shared dispatch) and its guarantee that it never mutates a tag.

### Modified Capabilities

None.

## Impact

- Files: `.github/workflows/release.yml`, `.github/workflows/cut-release-branch.yml` (new),
  `AGENTS.md`.
- Downstream (cli, opm-operator): none; they consume tags exactly as before.
- Depends on `open-platform-model/.github` shipping `cut-release-branch.yml` (section 2
  waits for it) and on the workspace root rule "Release Tags Are Immutable" (section 3
  cites it).
- Sibling changes in core, catalog_opm, cli and opm-operator add the same trigger and
  dispatch with their own tag prefixes.
