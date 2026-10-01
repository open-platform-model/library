## Why

The owner made release tags immutable across the open-platform-model repos that release
(rule of 2026-10-01): no tag under `refs/tags/` is ever moved, deleted or re-created, and a
wrong release is fixed by releasing the next version. The rule exists because the docs
system pins refs per site version, and for the library the tag is also the whole release: it
ships no binary and no image, and `go get` plus the Go module proxy resolve and cache the tag
itself. The platform side (the org `tags-immutable` ruleset and immutable releases) is set by
the owner in the browser. One repo-local gap stays open: release-please creates the tag by
creating the GitHub Release, and when a tag of that name already exists GitHub attaches the
release to the existing tag whatever commit it points at. The release workflow then reports
success for a release whose tag names a commit release-please never released. Nothing in the
library's workflow notices.

## What Changes

- The release workflow (`.github/workflows/release.yml`) gains a step that runs only when
  release-please reports a created release: it resolves the remote tag to the commit it
  names (peeling an annotated tag) and fails the run, with an error naming the tag, both
  SHAs and the roll-forward recovery, unless that commit equals the release commit
  release-please reports. The step reads only; it never creates, moves or deletes a tag.
- `AGENTS.md` gains one line under Repository Rules pointing at the workspace rule on
  immutable release tags and stating the library's recovery: a wrong or broken release is
  fixed by the next `-beta.N` (from GA, the next patch) with a `retract` directive in
  `go.mod` for the bad version, never by touching the tag.
- No Go code, no `opm/` surface and no release configuration change. The change lands as
  hidden commit types (`ci`, `chore`), so it cuts no release. SemVer: none.

## Capabilities

### New Capabilities

- `release-tag-integrity`: the release workflow's guarantee that a published library
  release tag names the commit release-please released, that the workflow never mutates a
  tag, and that a mismatch is surfaced as a failed run with roll-forward recovery.

### Modified Capabilities

None.

## Impact

- Files: `.github/workflows/release.yml`, `AGENTS.md`.
- Downstream (cli, opm-operator): none; they consume tags exactly as before.
- Runtime: one `git ls-remote` against the public repo per release run that created a
  release; non-release pushes skip the step.
- Sibling changes in core, catalog_opm, cli and opm-operator carry the same assertion; the
  workspace root carries the rule text this repo's pointer names.
