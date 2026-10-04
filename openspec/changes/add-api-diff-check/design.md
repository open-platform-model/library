## Context

Owner decision j4 (beta.1 walkthrough, 2026-10-03): the library API-diff check warns until GA and blocks after. The library is on the beta line (`v1.0.0-beta.4` is the newest tag at origin/main `58f8151`), where a breaking change to `opm/` is allowed only as a `feat!` commit whose `BREAKING CHANGE:` footer is the migration note (ADR-010). The repository also carries an older release tag, `v0.7.0` (AGENTS.md, "Release tags are immutable"), which a "newest release tag" rule would pick and so flip the check to blocking before GA.

library#181 (merged as `58f8151`) set the rules every workflow here follows: workflow-level `permissions:` with read-only grants, `persist-credentials: false` on every checkout, actions pinned by full SHA with a version comment, and third-party tools pinned to a checksum committed in this repository.

## Goals / Non-Goals

**Goals:** every pull request that touches Go code shows the incompatible changes to the exported `opm/` API since the last release tag; the outcome is a warning on a prerelease base and a failure on a release base, derived from the tag; the job holds a read-only token and builds a tool whose checksum is committed; a maintainer can run the same check locally.

**Non-Goals:** a required status check (owner, ruleset), consumer builds (`add-consumer-build-job`), suggesting the next version number, judging compatible additions, any `opm/` change.

## Decisions

- The tool SHALL be `golang.org/x/exp/cmd/apidiff`, built from a tools module at `.tasks/apidiff/` (`go.mod` plus committed `go.sum`, one `tools.go`-style blank import or a `tool` directive) with `go build -C .tasks/apidiff -o <tmp>/apidiff golang.org/x/exp/cmd/apidiff`. `-mod=readonly` is the default, so the build refuses any module whose hash is not in the committed `go.sum`. The pinned pseudo-version SHALL be the newest whose `go` directive does not exceed the library's own `go.mod` directive (today `go 1.25.0`; `x/exp` at `v0.0.0-20260908205506-85c1c2202aba` already declares `go 1.26.0`), so CI's `setup-go` from the library's `go.mod` builds it without a toolchain switch. A bump moves the pin by hand when a Go upgrade needs it.
- The base tag SHALL be `git describe --tags --abbrev=0 --match 'v*' <base>`, where `<base>` is `github.event.pull_request.base.sha` in CI (passed through `env:`, never inlined into the script) and `HEAD` locally; `BASE=<tag>` overrides the resolution for a local run. Nearest-reachable, not newest-by-SemVer, keeps `v0.7.0` out.
- The base API SHALL be exported from a `git archive <tag>` copy in a temporary directory (`apidiff -m -w old.export github.com/open-platform-model/library`, run in that copy), the head API from the work tree the same way, then `apidiff -m -incompatible old.export new.export`. Internal packages are skipped by `apidiff` itself (it prints "Ignoring internal package" to stderr), which matches "public surface = `opm/` only".
- Entries matching a line of the committed allow list `.tasks/apidiff/allow.txt` (extended regular expressions, one per line, comments with `#`) SHALL be dropped before the decision. It starts with one line, for the value change of `schema.DefaultSchemaModule`, which every release-cascade core move makes; a string constant's value is not a compile-time break for any consumer that does not use it as a constant expression of a fixed type, and treating it as one would fail every cascade PR after GA.
- Mode: `warn` when the base tag has a prerelease suffix (`-` after the patch number), `block` otherwise. With no remaining entries the script prints that the API is compatible and exits 0. With entries, `warn` prints each as a `::warning::` annotation, writes the job summary and exits 0; `block` prints them as `::error::` and exits 1. The summary names the base tag, the mode, the entries and the remedy: "breaking change: needs a `feat!` commit with a `BREAKING CHANGE:` footer (ADR-010)", and in `block` mode adds that a break after GA needs a new major. Annotations and the summary are written only when `GITHUB_ACTIONS` is set; locally the same text goes to stdout.
- `.github/workflows/api-diff.yml`: `on: pull_request` with `paths:` for `**.go`, `go.mod`, `go.sum`, `.tasks/api-diff.sh`, `.tasks/apidiff/**` and the workflow itself; `permissions: contents: read`; one job `API diff` on `ubuntu-latest` with `timeout-minutes: 10`; checkout at the SHA `lint.yml` pins, with `fetch-depth: 0` (tags) and `persist-credentials: false`; `setup-go` and `setup-task` at the SHAs the other workflows pin; one step `task api:diff`. No `continue-on-error`: in `warn` mode the script itself exits 0, so a failure of the script (a build error, a missing tag) still shows red instead of hiding behind a green check.

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

### The DefaultSchemaModule allow entry
**Context**: In the two prototype runs, the only incompatible entry was the core pin constant, which the release cascade moves on every core release.
**Decision**: Allow-list that one entry by a regular expression naming the symbol and "value changed".
**Rationale**: Without it the check warns on every cascade PR before GA (noise that teaches readers to ignore it) and fails every one after. The allow list is committed and under code-owner review (`/.tasks/`), so widening it is visible.

## Risks / Trade-offs

- [A future Go release changes export data the pinned `x/tools` cannot read] → the script fails red, not silently green; the fix is a pin bump in `.tasks/apidiff/`.
- [The library's `go` directive moves to 1.26.0 (wave 2, `lib-e2e5`)] → the pin may then move to a newer `x/exp`; not required by this change.
- [A merge-base without tags (a shallow clone)] → `fetch-depth: 0` in CI; locally `git describe` fails loudly and the script says to fetch tags.
- [The check is not required] → by owner decision only a ruleset can make the GA block binding; until then a red check is advisory, as are the cascade statuses in `warn` mode.
- [Rebase with `add-consumer-build-job`] → both add one AGENTS.md paragraph in the same section and their own workflow file; the second to merge rebases.
