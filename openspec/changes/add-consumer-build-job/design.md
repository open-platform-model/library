## Context

cli (`go 1.26.0`) and opm-operator (`go 1.26.2`) both require `github.com/open-platform-model/library v1.0.0-beta.4` at their `main`; the library's own `go.mod` says `go 1.25.0`. Neither consumer has a `replace` or a `go.work`. Both repos are public. The library's required check is `Go tests` only, so any new job is non-required unless someone adds it to the ruleset. library#181 set the workflow rules: workflow-level `permissions:`, no write grant on a `pull_request` job, `persist-credentials: false` on every checkout, actions pinned by full SHA with a version comment, and downloaded tools pinned to a committed sha256.

## Goals / Non-Goals

**Goals:** a library PR author sees, on the PR, whether cli and opm-operator still compile and vet against the PR head, and which symbol broke if not; no `replace` reaches a tracked file; the required job and G1 are untouched; the workflow meets the library#181 rules.

**Non-Goals:** consumer tests, e2e or lint; fork PRs; running on `main` pushes; making the job required; the rest of decision j4 (API diff, Dependabot ignores, `RELEASING.md`).

## Decisions

- The workflow MUST be a new file, `consumer-build.yml`, with `on: pull_request` and a `paths:` filter: `opm/**`, `go.mod`, `go.sum`, `.github/workflows/consumer-build.yml`, `.tasks/consumer-build.sh`. A path filter is safe here only because the check is not required (a required check that a filter skips stalls the PR).
- The job MUST run only for same-repo PRs: `if: github.event.pull_request.head.repo.full_name == github.repository`. It MUST NOT use `pull_request_target`.
- Workflow `permissions: contents: read`; no job grants more. Every checkout sets `persist-credentials: false`.
- `concurrency: consumer-build-<PR number>` with `cancel-in-progress: true`, so a new push stops the old run.
- One job, a matrix over `consumer: [cli, opm-operator]` with `fail-fast: false`, so one consumer's break does not hide the other's result. Job name `Consumer build (<consumer>)`.
- Steps: check out the library (PR head, the event default) at `library/`; check out `open-platform-model/<consumer>` at `ref: main` into `consumer/`; `actions/setup-go` (the SHA `test.yml` pins) with `go-version-file: consumer/go.mod` and `cache-dependency-path: consumer/go.sum`; then `bash library/.tasks/consumer-build.sh <consumer-dir> <library-dir> <work-dir>` with `GOTOOLCHAIN=local`, so the Go that runs is the one the consumer's `go.mod` names, never a silent toolchain download.
- `.tasks/consumer-build.sh` SHALL:
  1. create the work directory and run `go work init <consumer-dir> <library-dir>` there (with `GOTOOLCHAIN=local` the `go.work` `go` line is the installed toolchain's, which is at least the consumer's);
  2. run `go build ./...` and then `go vet ./...` in the consumer with `GOWORK=<work-dir>/go.work`, teeing output to a log in the work directory;
  3. on failure, and only when `GITHUB_STEP_SUMMARY` is set, append a summary section naming the consumer, its `git rev-parse HEAD`, the step that failed, and the compiler's `file:line:col: message` lines (first 50), and print a `::warning` line naming the consumer; then exit non-zero;
  4. assert, in both checkouts, that `git status --porcelain` is empty, so neither a `go.work`, a `go.work.sum` nor an edited `go.mod` was left in a tracked tree.
- One `go.work` per consumer, never one for both. A shared workspace would run MVS across cli and opm-operator together and build each with versions it does not use.
- `go vet ./...` compiles the consumers' test files too, so an API used only in a consumer's tests is covered without running them.

```bash
# .tasks/consumer-build.sh <consumer-dir> <library-dir> <work-dir> (sketch)
mkdir -p "$work" && (cd "$work" && go work init "$consumer" "$library")
export GOWORK="$work/go.work"
for step in build vet; do
  go -C "$consumer" "$step" ./... 2>&1 | tee "$work/$step.log" || report "$step"
done
for tree in "$consumer" "$library"; do test -z "$(git -C "$tree" status --porcelain)"; done
```

## Research & Decisions

### go.work versus a replace in a copied go.mod
**Context**: The owner decision says "replace in a throwaway workspace"; the plan entry names a `go.work`.
**Explored**: (a) `go mod edit -replace` on the consumer checkout; (b) `-modfile` pointing at a copied `go.mod` with the replace; (c) a `go.work` outside both checkouts, selected with `GOWORK`.
**Decision**: (c).
**Rationale**: (a) edits a tracked file, which the decision rules out. (b) works but needs a copied `go.sum` and still leaves a writable `go.mod` next to the build. (c) is a workspace replace by definition (`use` substitutes the module for every requirement of it), touches no file in either checkout, and is removed with `$RUNNER_TEMP`. This reading keeps the decision's intent: the replace lives only in a throwaway workspace.

### Local rehearsal at origin/main
**Context**: Whether the consumers build in workspace mode against the library at all, before writing the workflow.
**Explored**: On 2026-10-04, cli `5180cad1` and opm-operator `8dc24b3` exported to the scratchpad, one `go work init <consumer> <library worktree>` each in its own directory, then `go build ./...` and `go vet ./...` with `GOWORK` set, Go 1.26.5, `GOTOOLCHAIN=local`.
**Decision**: Both pass (about 7 and 8 seconds with a warm module cache). cli's `go:embed dist/install.yaml` is committed, and opm-operator needs no generated code, so a plain clone builds.
**Rationale**: The approach works without extra setup steps. Still unverified, and checked by section 1: that the script's failure report names the broken symbol, and that the checkout of another public repo with the read-only job token behaves on Actions as locally (proved by the PR's own run).

### Green with a warning, or red and non-required
**Context**: The plan entry says "keep it warn-only"; the owner decision says "a non-required job".
**Explored**: (a) `continue-on-error: true`, so the check stays green and only the annotation and summary show the break; (b) the job fails, and is not in the required checks.
**Decision**: (b).
**Rationale**: Non-required already means the job never blocks a merge, which is what "warn-only" asks for. A green check on a broken consumer is easy to miss, while a red non-required check is visible in the PR list and still mergeable. The summary and warning carry the detail either way.

### Consumer ref
**Context**: Which consumer commit to build.
**Decision**: `main` of each consumer, resolved when the job runs; the summary records the commit.
**Rationale**: `main` is what the next cascade bump lands on. A failure can also come from a library break already on library `main` that a consumer has not adapted to; the summary's consumer commit and the PR diff tell the two apart, and SD1 (deprecate, then remove) keeps that case rare.

## Risks / Trade-offs

- [A `feat!` PR goes red here by design] → intended; the summary names the symbol, and the consumer migrates after the release through the cascade.
- [A consumer's own `main` is broken] → the job goes red for a reason outside the PR; the summary shows the consumer commit, and the job is non-required.
- [CI minutes] → two jobs per Go-touching PR, cancelled on a new push; the Go module cache is keyed on the consumer's `go.sum`. Caches written by a PR run are scoped to its ref and never reach `main` or the release workflow.
- [A consumer starts needing a generated file or a private module] → the build fails with a clear Go error; the job would then need a setup step, decided then.
