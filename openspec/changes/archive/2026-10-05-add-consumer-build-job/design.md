## Context

cli (`go 1.26.0`) and opm-operator (`go 1.26.2`) both require `github.com/open-platform-model/library v1.0.0-beta.4` at their `main`; the library's own `go.mod` says `go 1.25.0`. Neither consumer has a `replace` or a `go.work`. Both repos are public. The library's required check is `Go tests` only, so any new job is non-required unless someone adds it to the ruleset. library#181 set the workflow rules: workflow-level `permissions:`, no write grant on a `pull_request` job, `persist-credentials: false` on every checkout, actions pinned by full SHA with a version comment, and downloaded tools pinned to a committed sha256.

## Goals / Non-Goals

**Goals:** a library PR author sees, on the PR, whether cli and opm-operator still compile and vet against the PR's merge commit, and which symbol broke if not; no `replace` reaches a tracked file; the required job and G1 are untouched; the workflow meets the library#181 rules.

**Non-Goals:** consumer tests, e2e or lint; the consumers' `//go:build ignore` programs; fork PRs; running on `main` pushes; making the job required; the rest of decision j4 (API diff, Dependabot ignores, `RELEASING.md`).

## Decisions

- The workflow MUST be a new file, `consumer-build.yml`, with `on: pull_request` and a `paths:` filter: `opm/**`, `go.mod`, `go.sum`, `.github/workflows/consumer-build.yml`, `.tasks/consumer-build.sh`. A path filter is safe here only because the check is not required (a required check that a filter skips stalls the PR).
- The job MUST run only for same-repo PRs: `if: github.event.pull_request.head.repo.full_name == github.repository`. It MUST NOT use `pull_request_target`.
- Workflow `permissions: contents: read`; no job grants more. Every checkout sets `persist-credentials: false`. Every action is pinned by full commit SHA with a version comment, at the SHAs `test.yml` pins.
- Concurrency is declared at the workflow level, `concurrency: {group: consumer-build-${{ github.event.pull_request.number }}, cancel-in-progress: true}`, so a new push stops the old run. At the workflow level the group covers the whole run, both matrix legs together; a job-level group without `${{ matrix.consumer }}` would make the cli and opm-operator legs cancel each other.
- One job, a matrix over `consumer: [cli, opm-operator]` with `fail-fast: false`, so one consumer's break does not hide the other's result. Job name `Consumer build (<consumer>)`, `timeout-minutes: 15`, so a stuck module download does not run for the 360-minute default.
- Steps: check out the library at `library/` with the checkout default for `pull_request`, which is the PR's merge commit (the PR head merged onto its base, as in `test.yml`): the code that would land; check out `open-platform-model/<consumer>` at `ref: main` into `consumer/`; `actions/setup-go` with `go-version-file: consumer/go.mod` and `cache: false` (see Risks); then `bash library/.tasks/consumer-build.sh <consumer-dir> <library-dir> <work-dir>` with `GOTOOLCHAIN=local`, so the Go that runs is the one the consumer's `go.mod` names, never a silent toolchain download.
- `.tasks/consumer-build.sh` runs under `set -euo pipefail`, so a failing `go` command in a `| tee` pipeline fails the pipeline, and SHALL:
  1. snapshot `git status --porcelain` of both checkouts before anything else, checking git's own exit status (a git failure is an error, never an empty snapshot);
  2. create the work directory and run `go work init <consumer-dir> <library-dir>` there (with `GOTOOLCHAIN=local` the `go.work` `go` line is the installed toolchain's, which is at least the consumer's);
  3. run `go build ./...` and then `go vet ./...` in the consumer with `GOWORK=<work-dir>/go.work`, teeing each step's output to a log in the work directory; the first failing step (init, build or vet) is recorded and the later steps are skipped;
  4. on failure, and only when `GITHUB_STEP_SUMMARY` is set, append a summary section naming the consumer, its `git rev-parse HEAD`, the step that failed, and the compiler's `file:line:col: message` lines (first 50), or, when no such line matches (a toolchain or `go work` error such as `go: module ... requires go >= 1.26.3`), the log's last 30 lines; and print a `::warning` line naming the consumer;
  5. on every path, passed or failed, take the snapshot again and compare it with step 1's, so neither a `go.work`, a `go.work.sum` nor an edited `go.mod` was left in a tracked tree; a difference is an error. Comparing against a snapshot, not requiring an empty status, lets the script run locally against a library worktree with uncommitted edits;
  6. exit non-zero when a step failed or a tree changed.
- One `go.work` per consumer, never one for both. A shared workspace would run MVS across cli and opm-operator together and build each with versions it does not use.
- `go vet ./...` compiles the consumers' `_test.go` files too, so an API used only in a consumer's tests is covered without running them. Files behind a `//go:build ignore` constraint are not: cli's `tests/integration/platform-build/main.go` and `tests/integration/render-parity/main.go` import the library but neither command compiles them (a Non-Goal).

```bash
# .tasks/consumer-build.sh <consumer-dir> <library-dir> <work-dir> (sketch)
set -euo pipefail
before_c=$(git -C "$consumer" status --porcelain)   # set -e fails on a git error
before_l=$(git -C "$library" status --porcelain)
failed=""
mkdir -p "$work"
(cd "$work" && go work init "$consumer" "$library") >"$work/init.log" 2>&1 || failed=init
export GOWORK="$work/go.work"
for step in build vet; do
  [ -z "$failed" ] || break
  go -C "$consumer" "$step" ./... 2>&1 | tee "$work/$step.log" || failed=$step
done
[ -z "$failed" ] || report "$failed"
[ "$(git -C "$consumer" status --porcelain)" = "$before_c" ] || tree_changed consumer
[ "$(git -C "$library" status --porcelain)" = "$before_l" ] || tree_changed library
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
**Rationale**: The approach works without extra setup steps. Still unverified: that the checkout of another public repo with the read-only job token behaves on Actions as locally (proved by the PR's own run).

### Script rehearsal (tasks 1.2 and 1.3)
**Context**: Whether `.tasks/consumer-build.sh` reports what the spec asks, run against real git checkouts.
**Explored**: On 2026-10-04, shallow `git clone`s of cli `main` (`abc0093`) and opm-operator `main` (`8dc24b3`) in the scratchpad, Go 1.26.5, `GOTOOLCHAIN=local`.
**Decision**: The script behaves as designed:
- Against this worktree (with the uncommitted script itself in it): both consumers build and vet, exit 0, both trees unchanged; the snapshot comparison accepts the worktree's own pending edits.
- A `go` wrapper that drops `stray.txt` into the consumer during `go vet`: exit 1, the tree check names `?? stray.txt`.
- A consumer directory that is not a git repository: exit 1, `git status failed`, before any build.
- A shallow clone of library `main` with `func WithRegistry(` renamed in `opm/kernel`: exit 1; the summary names cli commit `abc0093c…`, the step `go build ./...` and `internal/config/kernel.go:13:27: undefined: kernel.WithRegistry`, and the `::warning` names cli; the tree check still ran (both trees unchanged).
- The same clone with its `go` line raised to `1.27.0`: `go work init` fails, exit 1, and with no `file:line:col` line the summary falls back to the log tail: `go: ../library/go.mod requires go >= 1.27.0 (running go 1.26.5; GOTOOLCHAIN=local)`.
**Rationale**: Every failure path of the script ran on a git checkout, so the tree check is exercised, and the failing pipeline status reaches the script under `pipefail`.

### Green with a warning, or red and non-required
**Context**: The plan entry says "keep it warn-only"; the owner decision says "a non-required job".
**Explored**: (a) `continue-on-error: true`, so the check stays green and only the annotation and summary show the break; (b) the job fails, and is not in the required checks.
**Decision**: (b).
**Rationale**: Non-required already means the job never blocks a merge, which is what "warn-only" asks for. A green check on a broken consumer is easy to miss, while a red non-required check is visible in the PR list and still mergeable. The summary and warning carry the detail either way. The supervisor confirmed this reading in the plan review. It differs on purpose from the API-diff check (add-api-diff-check), which in warn mode before GA exits 0 with a warning; AGENTS.md states that this job goes red without blocking a merge; the add-api-diff-check paragraph, which lands next to it, states its own green-with-warning behaviour before GA.

### Consumer ref
**Context**: Which consumer commit to build.
**Decision**: `main` of each consumer, resolved when the job runs; the summary records the commit.
**Rationale**: `main` is what the next cascade bump lands on. A failure can also come from a library break already on library `main` that a consumer has not adapted to; the summary's consumer commit and the PR diff tell the two apart, and SD1 (deprecate, then remove) keeps that case rare.

## Risks / Trade-offs

- [A PR goes red because it changes API a consumer's `main` still uses] → per SD1 (deprecate, then remove) the library does not delete or rename such API; it deprecates it, and the removal waits until both consumers have migrated. A red run is therefore a signal to rework the PR that way, not an expected outcome to merge through. The job stays non-required, so the owner can still merge deliberately.
- [A consumer's own `main` is broken] → the job goes red for a reason outside the PR; the summary shows the consumer commit, and the job is non-required.
- [CI minutes] → two jobs per Go-touching PR, cancelled on a new push. The setup-go cache is off (code review): this workflow runs only on `pull_request`, so it would never write a `main`-scoped entry, every PR's first run would start cold anyway, and each PR would save two more entries against the repo's 10 GB cache quota, which can evict `test.yml`'s `main`-scoped Go cache. The cost is a cold module download per run, mostly opm-operator's k8s graph.
- [A later edit drops a guarantee] → `.tasks/consumer-build-test.sh` (supervisor triage SD18) runs offline in `task check` and in `test.yml`'s `Go tests` job against synthetic git repos (`GOPROXY=off GOTOOLCHAIN=local`): a pass, a renamed library symbol (exit 1, `undefined:` in the summary), a file dropped into the consumer during vet and one dropped into the library (exit 1 each), a non-git consumer (exit 1), and three mutants of the script, `set -eu` without pipefail, the consumer tree check deleted and the library tree check deleted, each of which must fail its case.
- [A consumer starts needing a generated file or a private module] → the build fails with a clear Go error; the job would then need a setup step, decided then.

### Review fixes to the script

`GOWORK` is exported before `go work init`, because init writes to `GOWORK` when the caller's environment already sets it, which would put the file outside the work directory. The build step is `go build -o /dev/null ./...`, so a consumer whose `./...` is one main package gets no binary written into its checkout (the synthetic test consumer is one).
