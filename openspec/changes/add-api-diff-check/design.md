## Context

Owner decision j4 (beta.1 walkthrough, 2026-10-03): the library API-diff check warns until GA and blocks after. The library is on the beta line (`v1.0.0-beta.4` is the newest tag at origin/main), where a breaking change to `opm/` is allowed only as a `feat!` commit whose `BREAKING CHANGE:` footer is the migration note (ADR-010). From GA a breaking change is MAJOR (CONSTITUTION VI) and needs a migration fragment under `migrations/unreleased/` per `migrations/README.md` (ADR-004, AGENTS.md "Working Style for Agents"). The repository also carries an older release tag, `v0.7.0` (AGENTS.md, "Release tags are immutable"), which a "newest release tag" rule would pick and so flip the check to blocking before GA.

library#181 (merged as `58f8151`) set the rules every workflow here follows: workflow-level `permissions:` with read-only grants, `persist-credentials: false` on every checkout, actions pinned by full SHA with a version comment, and third-party tools pinned to a checksum committed in this repository.

## Goals / Non-Goals

**Goals:** every pull request that touches Go code shows the incompatible changes it makes to the exported `opm/` API, measured against the last release tag; changes already on the base branch since that tag are shown as inherited and never charged to the pull request; the outcome is a warning on a prerelease base and a failure on a release base, derived from the tag; the job holds a read-only token and builds a tool whose checksum is committed; a maintainer can run the same check locally.

**Non-Goals:** a required status check (owner, ruleset), consumer builds (`add-consumer-build-job`), suggesting the next version number, judging compatible additions, an allow list of accepted incompatible changes, any `opm/` change.

## Decisions

- The tool SHALL be `golang.org/x/exp/cmd/apidiff`, built from a tools module at `.tasks/apidiff/` (`go.mod` with a `tool` directive plus a committed `go.sum`, no Go source) with `go build -C .tasks/apidiff -o <tmp>/apidiff golang.org/x/exp/cmd/apidiff` under `GOWORK=off GOFLAGS=-mod=readonly` (exported once for every Go command the script runs). `-mod=readonly` refuses any module whose hash is not in the committed `go.sum`; `GOWORK=off` and the explicit flag keep an ancestor `go.work` or a maintainer's `GOFLAGS=-mod=mod` from changing what gets built. The pinned pseudo-version SHALL be the newest whose `go` directive does not exceed the library's own `go.mod` directive (today `go 1.25.0`), so CI's `setup-go` from the library's `go.mod` builds it without a toolchain switch. That is `v0.0.0-20260820122028-d6e0b57b1a69`, the last `x/exp` commit before `ca53665` ("upgrade go directive to at least 1.26.0"), with `x/tools v0.49.0`. A bump moves the pin by hand when a Go upgrade needs it.
- The base tag SHALL be `git describe --tags --abbrev=0 --match 'v[0-9]*' <base>`, where `<base>` is `github.event.pull_request.base.sha` in CI (passed through `env:` as `API_DIFF_BASE_REF`, never inlined into the script) and, locally, the merge base of `HEAD` with `origin/main` (or `HEAD` when there is no `origin/main`). `BASE=<tag>` overrides the resolution for a local run only: the script ignores it when `GITHUB_ACTIONS` is set, so nothing in CI changes the mode. Nearest-reachable, not newest-by-SemVer, keeps `v0.7.0` out. No reachable tag fails the script with a hint to fetch the tags.
- Three API exports, each `apidiff -m -w <file> github.com/open-platform-model/library` run with `GOWORK=off GOFLAGS=-mod=readonly`: the tag, from a `git archive <tag>` copy in a temporary directory; the base commit, the same way (skipped when the base commit is the tag's commit); the head, from the work tree. Two comparisons with `apidiff -m -incompatible`: tag→head (everything incompatible since the release) and tag→base (what main already carries). Internal packages are skipped by `apidiff` itself (it prints "Ignoring internal package" to stderr), which matches "public surface = `opm/` only".
- An entry of tag→head that also appears, line for line, in tag→base is **inherited**: it is listed under "already on the base branch since `<tag>`" and never warns, fails or names a remedy. Only the remaining entries are **new** in this pull request and decide the outcome. A pull request that edits an inherited entry again (a constant moved twice) changes the line and so shows it as new.
- Mode: `warn` when the base tag has a prerelease suffix (`-` after the patch number), `block` otherwise. With no new entries the script says the API is compatible with the tag (listing any inherited entries) and exits 0. With new entries, `warn` prints each as a `::warning::` annotation, writes the job summary and exits 0; `block` prints them as `::error::` and exits 1. The summary names the base tag, the mode, the new and inherited entries and the remedy. In `warn` mode: "a breaking change needs a `feat!` commit with a `BREAKING CHANGE:` footer that is its migration note (ADR-010)". In `block` mode: "a breaking change is MAJOR (CONSTITUTION VI) and needs a migration fragment under `migrations/unreleased/` per `migrations/README.md` (ADR-004)". Annotations and the summary are written only when `GITHUB_ACTIONS` is set; locally the same text goes to stdout.
- `.github/workflows/api-diff.yml`: `on: pull_request` with `paths:` for `**.go`, `go.mod`, `go.sum`, `Taskfile.yml`, `.tasks/api-diff.sh`, `.tasks/apidiff/**` and the workflow itself; `permissions: contents: read`; `concurrency` per pull request with `cancel-in-progress: true`; one job `API diff` on `ubuntu-latest` with `timeout-minutes: 10`; checkout at the SHA `lint.yml` pins, with `fetch-depth: 0` (tags and the base commit) and `persist-credentials: false`; `setup-go` and `setup-task` at the SHAs the other workflows pin; one step `task api:diff`. No `continue-on-error`: in `warn` mode the script itself exits 0, so a failure of the script (a build error, a missing tag) still shows red instead of hiding behind a green check. This replaces the plan entry's "continue-on-error plus a job summary" wording; the outcome the owner asked for (warn until GA) is the same.

## Research & Decisions

### apidiff or gorelease
**Context**: The plan names both and asks for a prototype against a `-beta.N` base before choosing.
**Explored**: `gorelease -base=<version>` resolves the base through the module proxy and adds a version suggestion; `apidiff -m` compares two export-data files the script produces from git. Prototyped `apidiff` on 2026-10-04 in a scratch directory: `v1.0.0-beta.3` against origin/main reported nothing; `v1.0.0-alpha.33` against origin/main reported `./opm/schema.DefaultSchemaModule: value changed from "opmodel.dev/core@v2.0.0-alpha.10" to "opmodel.dev/core@v2.0.0-beta.2"` under `-incompatible`; the exit code was 0 in both cases.
**Decision**: `apidiff`.
**Rationale**: Reading the base from a git tag needs no proxy round trip and treats `-beta.N` like any other tag; the version suggestion `gorelease` adds is release-please's job here. Because `apidiff` exits 0 on incompatible changes, the script decides from the output.

### Where the tool's checksum lives
**Context**: `workflow-hardening` requires a pinned, checksum-verified tool. `apidiff` ships no release binaries.
**Explored**: (a) `go install golang.org/x/exp/cmd/apidiff@<version>`, verified only against the public checksum database; (b) a `tool` line in the library's `go.mod`; (c) a separate tools module with its own committed `go.sum`.
**Decision**: (c).
**Rationale**: (a) has no trust root in this repository. (b) puts `golang.org/x/exp` and `golang.org/x/tools` into the module graph every embedder resolves through MVS. (c) keeps the library's `go.mod` untouched, and its `go.sum` is the committed checksum `go build -mod=readonly` enforces. A nested module is invisible to `go build ./...`, `go vet ./...` and golangci-lint at the root.

### Inherited entries
**Context**: Plan review: tag→head accumulates. A break merged after the tag would be reported again on every later pull request, telling an unrelated one it needs a `feat!` commit, and after GA failing it until the next release.
**Decision**: Export the API at the base commit too and charge a pull request only with entries tag→base does not already list.
**Rationale**: The pull request's own contribution is what its author can act on. The tag stays the reference for the mode and for what "incompatible" means, so a break made and then reverted inside one release cycle never shows.

### No allow list
**Context**: The release cascade rewrites the `schema.DefaultSchemaModule` string constant on every core move (`.tasks/cascade/cascade.sh`, phase C1), and `apidiff` counts a constant's value change as incompatible. The planner proposed a committed allow list for that entry; plan review found it is not part of owner decision j4 or the supervisor decisions, and no ratification was logged.
**Decision**: No allow list in this change.
**Rationale**: It narrows what the post-GA block covers, which is the owner's call. With inherited entries, the cost is bounded: the `deps/cascade` pull request that moves core shows one new entry (a warning before GA, a red non-required check after), and every later pull request lists it as inherited only. A core move already carries `need-human-review`, so the extra annotation reaches a reviewer who is reading it anyway. If the owner wants the entry silenced, a follow-up adds a fixed-prefix allow line for that one symbol.

### Rehearsal (task 1.4)
Results of the local runs, recorded here rather than in a commit body (squash-body hazard, AGENTS.md):

Run on 2026-10-04 with the pinned `apidiff` (`d6e0b57`), Go 1.26.5 locally, from this branch. Scratch copies were `git clone`s of the branch with the uncommitted script and tools module copied in, and `API_DIFF_BASE_REF=HEAD`.

| Run | Expected | Seen |
| --- | --- | --- |
| `task api:diff` | nearest tag `v1.0.0-beta.4`, compatible | `mode: warn`, "The API is compatible with v1.0.0-beta.4", exit 0, about 15 s |
| `task api:diff BASE=v1.0.0-alpha.33` | the `DefaultSchemaModule` entry, inherited | `mode: warn`, compatible, one inherited entry (`./opm/schema.DefaultSchemaModule: value changed from "opmodel.dev/core@v2.0.0-alpha.10" to "opmodel.dev/core@v2.0.0-beta.2"`), exit 0 |
| `task api:diff BASE=v0.7.0` | mode line `block` | `mode: block (v0.7.0 is a release: ...)`; 110 entries, all inherited (main already carries them), so exit 0 |
| `API_DIFF_BASE_REF=v0.7.0` (base commit = tag) | block with new entries | 110 new entries, 110 `::error` lines, the MAJOR / migration-fragment remedy, exit 1 |
| Scratch copy: `Roots` in `opm/helper/platformmodule` unexported, with `GITHUB_ACTIONS=true`, `GITHUB_STEP_SUMMARY=<file>` and `BASE=v0.7.0` | listed as new, warn, `BASE` ignored | "BASE is ignored in CI", base tag `v1.0.0-beta.4`, `./opm/helper/platformmodule.Roots: removed` as new, one `::warning title=Incompatible API change::` line, the same text in the summary file with the `feat!` / `BREAKING CHANGE:` remedy, exit 0 |
| Scratch copy: `modversion.Canonical` renamed in `opm/internal/` and its callers | nothing new | compatible with `v1.0.0-beta.4`, exit 0 |
| Scratch copy: one `x/exp` hash in `.tasks/apidiff/go.sum` corrupted | build refused before any export | `verifying golang.org/x/exp@...: checksum mismatch ... SECURITY ERROR`, then "building apidiff from .tasks/apidiff failed", exit 2 |
| `git clone --depth 1` with `GITHUB_ACTIONS=true` | fails with the fetch-the-tags hint | `::error title=API diff::no v[0-9]* tag is reachable from <sha>; fetch the tags ...`, exit 2 |

At the root, `go list ./...` lists no package of the tools module, and `go build ./...`, `go vet ./...` and `task lint` (0 issues) pass with it present.

## Risks / Trade-offs

- [A future Go release changes export data the pinned `x/tools` cannot read] → the script fails red, not silently green; the fix is a pin bump in `.tasks/apidiff/`.
- [The library's `go` directive moves to 1.26.0 (wave 2, `lib-e2e5`)] → the pin may then move to a newer `x/exp`; not required by this change.
- [A merge-base without tags (a shallow clone)] → `fetch-depth: 0` in CI; locally `git describe` fails loudly and the script says to fetch tags.
- [The check is not required] → by owner decision only a ruleset can make the GA block binding; until then a red check is advisory, as are the cascade statuses in `warn` mode. A break merged over a red check is inherited by later pull requests, not charged to them.
- [Each core cascade PR shows the `DefaultSchemaModule` entry] → see "No allow list"; after GA that PR's check is red until the owner decides otherwise.
- [Rebase with `add-consumer-build-job`] → both add one AGENTS.md paragraph and touch the "Workflow security" permissions list in the same section, plus their own workflow file; the second to merge rebases.
