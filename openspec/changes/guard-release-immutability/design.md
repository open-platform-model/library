## Context

See proposal.md for the motivation. State on `main` at planning time (2026-10-01, `02344e5`):

- `.github/workflows/release.yml` has one job, `release-please`: mint the release App token,
  then run `googleapis/release-please-action` v5.0.0 (pinned by SHA `45996ed`). It has no
  checkout and no step after release-please; the release step has no `id`.
- `release-please-config.json` has one package, `"."` (release-type `go`, beta prerelease),
  so the action sets the root outputs without a path prefix: `release_created`, `tag_name`,
  `sha` and so on. Read from the action source at the pinned SHA (`src/index.ts`,
  `setPathOutput` / `outputReleases`): for path `.` every field of the created release is
  written as an unprefixed output, `tagName` renamed to `tag_name`; `sha` is the commit the
  release was created for.
- release-please creates the tag through the Releases API (`target_commitish` = the release
  commit), so the tags it makes are lightweight. Checked on `v1.0.0-beta.1`: the ref object
  type is `commit`, `git ls-remote` lists only `refs/tags/v1.0.0-beta.1` (no `^{}` line) at
  `02344e5913458e4411a19668f0f4424b8141d2f4`, which equals the release's
  `targetCommitish`.
- Critic C5 (workspace research): when the tag already exists, release creation attaches to
  it regardless of `target_commitish`, so a stale tag at the wrong commit is adopted
  silently. That is the only path by which the library could publish a release whose tag
  disagrees with its release commit while the org ruleset forbids moving tags.
- Release types: `docs` and `refactor` release in this repo; `ci`, `chore`, `test`, `build`
  are hidden.

## Goals / Non-Goals

**Goals:**

- A release run whose tag does not name the release commit ends red, with a message that
  tells the human what to do next.
- Agents and humans reading `AGENTS.md` find the rule and the Go-specific recovery.

**Non-Goals:**

- Preventing tag mutation. The org `tags-immutable` ruleset and immutable releases do that;
  this repo has no say over them and the change does not reference their configuration.
- Draft-first releasing. The library uploads no assets, so its release is complete when it
  is created; draft-first is a cli and opm-operator concern.
- Auto-repair. The step never retags, deletes the release or opens a follow-up release.

## Decisions

### D1: Inline `run` step after release-please, no checkout

The assertion is a single shell step guarded by
`if: ${{ steps.release.outputs.release_created }}`, reading `tag_name` and `sha` through
`env:` (never interpolated into the script body), and querying
`https://github.com/${GITHUB_REPOSITORY}.git` with `git ls-remote`. The repo is public, so
no token is needed and the job keeps its current permissions.

Alternatives: a script file under `.github/scripts/` (needs a checkout step only to read
it, for about ten lines of shell); `gh api .../git/ref/tags/$TAG` (needs a second call to
dereference an annotated tag object, and a token). Rejected for size.

### D2: Peel before comparing

The step asks `ls-remote` for both `refs/tags/$TAG` and `refs/tags/$TAG^{}` and compares the
peeled line when present, else the plain line. Today every tag is lightweight, so the plain
line is what matches; peeling keeps the check correct if a tag is ever annotated, where the
plain line names the tag object rather than the commit. Matching is exact on the ref name
(`awk '$2 == ref'`), because `ls-remote` patterns tail-match and must never let
`v1.0.0-beta.1` pick up `v1.0.0-beta.10`.

```bash
remote="https://github.com/${GITHUB_REPOSITORY}.git"
out="$(git ls-remote "$remote" "refs/tags/${TAG}" "refs/tags/${TAG}^{}")"
got="$(awk -v r="refs/tags/${TAG}^{}" '$2 == r { print $1 }' <<<"$out")"
[ -n "$got" ] || got="$(awk -v r="refs/tags/${TAG}" '$2 == r { print $1 }' <<<"$out")"
if [ -z "$got" ]; then
  echo "::error::release tag ${TAG} is missing on the remote"; exit 1
fi
if [ "$got" != "$WANT" ]; then
  echo "::error::release tag ${TAG} names ${got}, release-please released ${WANT}. Tags are immutable: do not move or delete it. Release the next version with a retract directive for ${TAG} in go.mod."
  exit 1
fi
echo "release tag ${TAG} names ${WANT}"
```

### D3: Fail red, after the fact

The release already exists when the step runs; the step cannot un-publish it and must not
try. Its job is to make the condition loud: a red release run on `main` is the signal a
human acts on, by rolling forward. Failing before release creation would need release-please
to stop creating releases itself (`skip-github-release`) and a hand-written creation step,
which re-implements the tool for a case the org ruleset already makes rare.

### D4: Hidden commit types, one PR

`ci(release)` for the workflow and `chore(agents)` for the `AGENTS.md` line: `docs` releases
in this repo, and neither change alters what a consumer gets. PR title
`ci(release): assert the release tag names the release commit`.

## Risks / Trade-offs

- [The step never runs for real until the next release] → section 1 runs the step's exact
  script locally against `v1.0.0-beta.1` (expect pass), a wrong SHA (expect fail), a missing
  tag (expect fail) and an annotated tag in a public repo (expect the peeled commit), so the
  first real run is not the first execution.
- [A transient network error on `ls-remote` fails a release run that was fine] → the run is
  re-runnable; the step reads only, so a re-run is safe. Accepted rather than adding retries.
- [The `sha` output is release-please behaviour, not a documented contract] → read from the
  pinned action source; a dependabot bump of the action is the moment it could change, and a
  missing `sha` makes the comparison fail loudly (empty `WANT` never equals a SHA), not
  pass silently.

## Migration Plan

None. Rollback is reverting the commit; the step has no state.
