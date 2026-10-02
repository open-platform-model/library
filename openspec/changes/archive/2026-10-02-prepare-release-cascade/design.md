## Context

See proposal.md, "Why", for motivation and specs/release-pipeline/spec.md for the behaviour.
The facts that shape the approach:

- Release PRs come from release-please on a branch starting `release-please--` (today
  `release-please--branches--main--components--library`, PR #155). The release
  job acts as the release App (`.github/workflows/release.yml:17-25`), so the release PR's
  `pull_request` CI starts on its own, with `github.head_ref` set. core and catalog_opm instead
  dispatch release-PR CI with `workflow_dispatch`, where `head_ref` is empty; the shared G1
  shape in workspace RELEASING.md, section "Gates", keys on `head_ref || ref_name` to cover both.
- `.github/workflows/test.yml:15-39` has one job, `Go tests`, which runs `go mod tidy` then
  `task test`. `lint.yml` and `cue.yml` are separate workflows. The library has no required
  checks today (RELEASING.md, "Owner settings"), so any gate is advisory until the owner's
  ruleset names `Go tests`.
- The library's only shipped OPM pin is a Go string, `DefaultSchemaModule =
  "opmodel.dev/core@v2.0.0-beta.1"` (`opm/schema/loader.go:44`), mirrored by
  `registrytest.DefaultCoreVersion` (`opm/internal/registrytest/registrytest.go:113`). No
  `cue.mod` is consumed outside the repo: `modules/opm_platform` and `testdata/**` are test
  inputs (`Taskfile.yml:22-28`).
- `go.mod:5-9` has no `github.com/open-platform-model/*` requirement, and two indirect
  third-party requirements are legitimately at pseudo-versions (`cuelabs.dev/go/oci/ociregistry`,
  `github.com/protocolbuffers/txtpbfmt`).
- `cue.mod/local-module.cue` is CUE's local-replacement file, read by the library itself
  (`opm/internal/renderstage/modfile.go:22`). None is tracked today; tests that need one write
  it at run time.
- release-please opens no release PR when the generated changelog is empty
  (`googleapis/release-please` `src/strategies/base.ts`, the `changelogEmpty` check, "No user
  facing commits found"), so a hidden type cuts no release.
- Today (checked on `f1d9908`): no replace directive, no OPM Go pin, no `-0.dev.` literal in a
  tracked cue.mod or non-test Go file, no tracked `local-module.cue`. The gate is green on
  arrival.

## Goals / Non-Goals

**Goals:**
- One Taskfile entry point, `task deps:release-check`, that CI and a human run identically, and
  that needs no network, no `cue` binary and no Go module download.
- A gate that cannot read as "skipped = passed" on a release PR.
- Release outputs in the shape the other repos already use, so the later shared notify workflow
  treats every repo the same.

**Non-Goals:**
- Checking that a pinned version exists upstream (G1's "not an existing tag" clause). With no
  OPM Go dependency there is nothing to check; `go mod tidy` in the same job already fails on an
  unresolvable Go version, and `cue:catalog:drift` already checks catalog existence.
- Freshness (G2) and upstream-settled (G3) statuses; they come with the shared cascade
  workflows.
- Reading `.cascade-frozen` or `.cascade-hold`. A frozen pin is old, not dev; a hold caps a bump.
  Neither can make a dev or replaced pin legitimate in a release, so G1 ignores both files.

## Decisions

### D-a. G1 is a Taskfile task plus one guarded CI step

The check lives in `Taskfile.yml` as `deps:release-check` (a new "Release" group after the CUE
tasks). `test.yml` gains one step before "Running Tests":

```yaml
      - name: Release-pin gate (G1)
        # Release PRs only: a release never ships a dev, pseudo-version or
        # replaced pin. head_ref || ref_name is the shared G1 shape; here
        # release-PR CI runs on pull_request, so head_ref decides.
        if: startsWith(github.head_ref || github.ref_name, 'release-please--')
        run: task deps:release-check
```

It runs before `go mod tidy`, so it judges the committed `go.mod`, not a tidied copy.

*Alternatives:* a separate job (rejected by RELEASING.md "Gates": a skipped job shows as a
neutral pass and a ruleset cannot tell it from green); a step in `lint.yml` (works, but the
plan names the test job and `Go tests` is the check most likely to be required first); a
standalone script under `.github/` (rejected: the repo's entry point is the Taskfile, and a
human must be able to run the same check before cutting a release).

### D-b. The task body

The body as shipped in `Taskfile.yml` (preconditions on `go`, `jq` and `git`):

```bash
#!/usr/bin/env bash
set -euo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; BOLD='\033[1m'; RESET='\033[0m'
fail=0
report() { echo -e "==> ${RED}$1${RESET}"; echo "$2" | sed 's/^/    /'; fail=1; }

modjson=$(go mod edit -json)

# 1. Go replace directives (any module).
hits=$(echo "$modjson" | jq -r '.Replace[]? | "\(.Old.Path)\(if .Old.Version then "@"+.Old.Version else "" end) => \(.New.Path)\(if .New.Version then "@"+.New.Version else "" end)"')
[ -z "$hits" ] || report "go.mod carries replace directives:" "$hits"

# 2. OPM Go requirements at a pseudo-version (vX.0.0-ts-hash,
# vX.Y.Z-pre.0.ts-hash, vX.Y.Z-0.ts-hash).
hits=$(echo "$modjson" \
  | jq -r '.Require[]? | select(.Path | startswith("github.com/open-platform-model/")) | "\(.Path) \(.Version)"' \
  | grep -E '[-.][0-9]{14}-[0-9a-f]{12}$' || true)
[ -z "$hits" ] || report "OPM Go modules at a pseudo-version:" "$hits"

# 3. CUE dev pins in every tracked cue.mod, and dev literals in shipped
# (non-test) Go under opm/. Capture and test the output, never grep's
# or xargs' exit status; -H keeps the filename for a single file.
# Non-test files are selected by name: a ':!:*_test.go' pathspec also
# drops registrytest.go and schematest.go.
hits=$(git ls-files -z -- '*cue.mod/module.cue' | xargs -0 -r grep -HnE 'v: *"[^"]*-0\.dev\.' || true)
[ -z "$hits" ] || report "dev CUE pins in cue.mod/module.cue:" "$hits"
hits=$(git ls-files -z -- 'opm/*.go' | grep -zv '_test\.go$' | xargs -0 -r grep -HnE '"[^"]*-0\.dev\.[^"]*"' || true)
[ -z "$hits" ] || report "dev version literals in shipped Go:" "$hits"

# 4. Tracked CUE local replacements.
hits=$(git ls-files -- '*cue.mod/local-module.cue')
[ -z "$hits" ] || report "tracked cue.mod/local-module.cue:" "$hits"

if [ "$fail" -ne 0 ]; then
  echo -e "${RED}${BOLD}Release-pin gate failed.${RESET} A release never ships a dev, pseudo-version or replaced pin." >&2
  exit 1
fi
echo -e "==> ${GREEN}Release-pin gate passed:${RESET} no replace, OPM pseudo-version, dev pin or tracked local-module.cue."
```

`go mod edit -json` parses `go.mod` alone, so the task is offline. The pseudo-version regex
`[-.][0-9]{14}-[0-9a-f]{12}$` matches all three Go pseudo-version shapes (`vX.0.0-ts-hash`,
`vX.Y.Z-pre.0.ts-hash`, `vX.Y.Z-0.ts-hash`); it was checked against the two third-party
pseudo-versions in `go.mod` today, which it matches and which the path filter then excludes.
Rule 3 captures the matches and fails on a non-empty capture rather than on grep's exit
status: `xargs -r` exits 0 on an empty list, xargs exits 123 when only some of several grep
batches match, and `grep -n` drops the filename when it gets one file. Capturing with `-H`
avoids all three. Non-test files are selected by name (`grep -zv '_test\.go$'`), not by a
`:!:*_test.go` exclude pathspec: that pathspec also drops non-test files whose names end in
`test.go` (`opm/internal/registrytest/registrytest.go`, which holds `DefaultCoreVersion`, and
`opm/internal/schematest/schematest.go`), and its matching is not something to rely on across
git versions. Rule 1 lists replace directives as `old[@version] => new[@version]` and fails
on a non-empty listing, the same capture-and-test shape as rules 2 to 4.

### D-c. "Shipped cue.mods" means every tracked cue.mod in the library

RELEASING.md "Gates" scopes the CUE dev-pin rule to shipped cue.mods. The library ships no
cue.mod: its shipped CUE pin is the Go constant in `opm/schema/loader.go:44`, which rule 3's
second grep covers. The library's fixtures, though, are the evidence behind the release claim
"verified against core X and catalog Y"; a `-0.dev.` fixture pin means that evidence ran against
an unreleased upstream. So rule 3 scans every tracked `cue.mod/module.cue`. It costs nothing
today (zero hits) and frozen old-core fixtures are unaffected, because old is not dev.
This scope is wider than RELEASING.md "Gates" ("a shipped `cue.mod`") and than the peers; it
was settled on 2026-10-02 to keep the wide scan.

*Alternative:* scan only `modules/*` (the `CUE_MODULE_GLOBS` non-test glob). Rejected: that glob
holds a test-only platform fixture too, so it is no closer to "shipped", and it would miss
`testdata/render/**`.

### D-d. Release outputs mirror opm-operator

`release.yml` gives the release-please step `id: release` and the job:

```yaml
    outputs:
      releases_created: ${{ steps.release.outputs.releases_created }}
      tag_name: ${{ steps.release.outputs.tag_name }}
```

The library is a single root package (`release-please-config.json:7-8`, `"."`), so the
unprefixed `tag_name` output is the release tag. This is the exact shape of
`opm-operator/.github/workflows/release.yml:33-35`. No job reads the outputs yet.

### D-e. Hide `docs`, keep `refactor` (RELEASING.md "Pin classes")

`release-please-config.json:21` flips `docs` to `"hidden": true`; `refactor` (`:22`) stays
`false`. `pr-title.yml:33-44` keeps `docs` as an allowed title type: hiding changes what
releases, not what is valid. `AGENTS.md:347` moves `docs` from the releasing list to the
never-releasing list. A docs commit already in an open release PR's range stays in it only if a
releasing commit keeps the PR open; otherwise release-please closes nothing and simply stops
proposing one.

## Research & Decisions

### Where the G1 step goes

**Context**: RELEASING.md "Gates" requires G1 inside an existing required job.
**Explored**: `.github/workflows/test.yml`, `lint.yml`, `cue.yml`; the plan's change breakdown
names "the existing test job" for the library.
**Decision**: a step in `test.yml` job `test` (`Go tests`), before `go mod tidy`.
**Rationale**: the job runs on every PR including release PRs, it already installs Go and Task,
and checking before tidy judges the committed `go.mod`.

### release-please behaviour on hidden types

**Context**: RELEASING.md "Pin classes" assumes hiding `docs` stops docs-only releases.
**Explored**: release-please `src/strategies/base.ts` (`changelogEmpty`, line 331 on `main`).
**Decision**: rely on it; no `Release-As` or extra config.
**Rationale**: the library already depends on this for `test`, `ci`, `build` and `chore`.

## Risks / Trade-offs

- [Advisory until the ruleset exists] → the proposal states the dependency on the owner setting;
  the gate still shows red on the release PR, which a human merging it sees.
- [A future fixture legitimately needs a tracked `local-module.cue`] → write it at test time as
  today's tests do; if that becomes impossible, narrow rule 4 in its own change rather than
  exempting by path here.
- [The Go literal grep could hit a comment or error string quoting `-0.dev.`] → it scans
  double-quoted strings anywhere in non-test Go under `opm/`, comments included, and comments
  here may double-quote versions (`opm/catalog/requires.go:18`). None quotes a dev version
  today; one that does fails the gate on a release PR and is reworded, never exempted by
  weakening the gate.
- [A test needs a dev-pinned fixture] → rule 3 scans every tracked `cue.mod/module.cue`
  (D-c), so such a test writes its fixture at run time, as tests needing a `local-module.cue`
  already do.
- [Release PR branch name changes] → release-please's branch prefix has been
  `release-please--` across every major; the same condition is used in all four repos, so a
  change would surface everywhere at once.
- [Fewer releases, and library docs lag on opmodel.dev] → a docs-only fix no longer cuts a
  release. Today opmodel.dev builds library (and opm-operator) docs at exactly the version the
  newest cli tag pins (`opmodel.dev/site/versions.conf`, line mode); only core and catalog_opm
  docs come from a branch head. Owner decision 2026-10-02 (RELEASING.md, "Rollout and changes"):
  the opmodel.dev change `build-docs-from-branch-head` builds library, opm-operator and cli docs
  from the branch head too, and it must merge before section 3's commit merges. Until then a
  docs-only fix in this repo reaches opmodel.dev only with the next library release (and the cli
  release that pins it). The same gate applies to the opm-operator and cli edits. Because the
  change merges as one PR, this gate holds the whole PR, and with it G1 and the release outputs
  that `add-deps-cascade-task` and `join-release-cascade` need.

## Migration Plan

Merge as one PR (three `ci` sections, squash title `ci(release): ...`, hidden, no release).
The PR merges only after the opmodel.dev change `build-docs-from-branch-head` has merged. If that
change lags while G1 or the release outputs are needed, section 3's commit can be dropped from
this branch and landed as its own follow-up PR; it never merges before the opmodel.dev change.
The archive commit (`chore(openspec): archive prepare-release-cascade`, syncing
`release-pipeline` into `openspec/specs/`) rides this PR; nothing is pushed to `main`
afterwards (RELEASING.md "Owner settings").
Rollback is a revert of the squash commit. Release PR #155 (`chore(main): release 1.0.0-beta.2`, branch
`release-please--branches--main--components--library`) is open today; its next CI run after
this merges evaluates the merge ref, so it picks up G1. It is green on arrival (see Context).
