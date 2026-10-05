## Why

The library's public Go surface (`opm/`, `opm/internal/` excluded) is a contract with two frontends and with every embedder outside the workspace (Principle VI). Nothing checks it today: a breaking change is caught only if its author remembers the `feat!` commit and the `BREAKING CHANGE:` footer that carry the migration note on the beta line (ADR-010). The owner decided this as ADR-013, decision j4: "Library API-diff check warns until GA and blocks after." The supervisor added (wave 2 notes) that it is a non-required job, that the release cascade's `DefaultSchemaModule` value change goes on a fixed allow line (SD17), that it follows the workflow-hardening rules library#181 merged, and that the AGENTS.md hunk is coordinated with `add-consumer-build-job`.

## What Changes

- A pinned API-diff tool: `golang.org/x/exp/cmd/apidiff` at one exact pseudo-version, built from a small tools module under `.tasks/apidiff/` whose committed `go.sum` is the checksum the Go toolchain verifies before it builds anything. The library's own `go.mod` does not change.
- `.tasks/api-diff.sh` and a `task api:diff` entry: resolve the base as the nearest release tag reachable from the base commit (`git describe --tags --abbrev=0 --match 'v[0-9]*'`; `BASE=<tag>` overrides it for a local run, never in CI), export the module's API at that tag, at the base commit (both from `git archive` copies) and at the work tree, and list the incompatible changes `apidiff -m -incompatible` reports for the exported `opm/` packages. An entry the base commit already carries since the tag is listed as inherited, and the release cascade's `DefaultSchemaModule` value change is matched by one fixed allow line (SD17); neither is charged to the pull request.
- `.tasks/api-diff-test.sh` and `task api:diff:test`: offline tests of the script's split, mode and exit codes against fixture diffs, run by `task check` and the `Go tests` job.
- The mode is derived from the base tag, never from a variable: a prerelease tag (one with a `-` suffix, today `v1.0.0-beta.4`) warns: a warning annotation per new entry, a job summary naming the changes and the fix ("a breaking change needs a `feat!` commit with a `BREAKING CHANGE:` footer", ADR-010), and exit 0. A release tag (no suffix, from the first GA on) fails the job and names the post-GA rule: a breaking change is MAJOR (CONSTITUTION VI) and needs a migration fragment per `migrations/README.md` (ADR-004). A failure of the check itself (tool build, no reachable tag) fails the job in both modes.
- `.github/workflows/api-diff.yml`: a `pull_request` workflow (Go and module paths only) with `permissions: contents: read`, `persist-credentials: false`, a full-history checkout so the tags are present, per-PR concurrency, and actions pinned by full SHA like the other workflows. It is not a required status check.
- `AGENTS.md`: the `task api:diff` command line, `api-diff.yml` in the workflow-security permissions list, and one paragraph after it. `add-consumer-build-job` edits the same section; whichever of the two merges second rebases.

Not in this change: any allow entry beyond the cascade's core pin, the post-GA route for a deliberate break and making the check a required status (open points for GA, design.md), the consumer build job (`add-consumer-build-job`), the Dependabot and `RELEASING.md` parts of j4 (cli, opm-operator and workspace changes), and any Dependabot entry for the tools module (it is bumped by hand like golangci-lint).

## Capabilities

### New Capabilities

- `api-diff-check`: the pull-request check that compares the exported `opm/` API against the last release tag, warns on a prerelease base and fails on a release base.

### Modified Capabilities

None. The new workflow meets `workflow-hardening` as written: read-only grants, no persisted credential, and a tool whose trust root (the tools module `go.sum`) is committed here.

## Impact

- No `opm/` package changes and no public surface or SemVer effect. Commits are `ci` and `docs`, which release-please hides, so the change cuts no release.
- New files sit under `.github/`, `.tasks/` and `Taskfile.yml`, all already under code-owner review.
- cli and opm-operator are unaffected. Until GA the check only annotates; after GA a red result on a deliberate break is expected, and the route (migration fragment, new major, required status) is the owner's, decided then.
- Prototype (2026-10-04, `apidiff` at `v0.0.0-20260908205506-85c1c2202aba`; the pin is the older `v0.0.0-20260820122028-d6e0b57b1a69`, the newest on `go 1.25.0`): a `-beta.N` base tag works because the base is read from git, not resolved through the module proxy; `v1.0.0-beta.3` against `main` reports nothing; `v1.0.0-alpha.33` against `main` reports one incompatible entry, the `DefaultSchemaModule` value change; `apidiff` exits 0 either way, so the script decides from its output.
