## Why

The library has two consumers, cli and opm-operator, and today nothing in library CI tells a PR author that a change breaks them. A break surfaces only after a release, when the deps cascade bumps the pin in a consumer and its build fails there. Owner decision j4 (kernel plan walkthrough, 2026-10-03): "Consumer builds (cli + operator against the library PR head, replace in a throwaway workspace) run in library CI as a non-required job, after the prepare-release-cascade branches merge." That gate is met: the join-release-cascade PRs have merged in all five repos (library#180 among them), and the workflow hardening of library#181, whose rules every new workflow must follow, has merged too.

## What Changes

- New workflow `.github/workflows/consumer-build.yml`, separate from `test.yml` so the required `Go tests` job and the G1 step are untouched. On a same-repo `pull_request` that touches `opm/**`, `go.mod`, `go.sum`, the workflow itself or its script, it runs one job per consumer (cli, opm-operator): check out the library at the PR's merge commit (the checkout default for `pull_request`, as in `test.yml`) and the consumer at `main`, install Go from the consumer's `go.mod`, and run `go build ./...` and `go vet ./...` in the consumer against that library tree.
- The library tree reaches the consumer through a `go.work` written in a throwaway directory under `$RUNNER_TEMP` (`go work init <consumer> <library>`, selected with `GOWORK`). No `replace` and no `go.work` lands in either checkout's tracked files; the script snapshots both trees' `git status --porcelain` before the build and fails if either changed, on every path.
- New script `.tasks/consumer-build.sh` (the research's optional script, an addition to the plan entry's file list; the PR body names it) holds the build logic, so it runs the same way locally. On a failure it writes the consumer, the consumer commit and the compiler's error lines (which name the broken symbol, such as `undefined: kernel.Foo`) to the job summary, plus a warning annotation.
- The job is non-required: it never blocks a merge. It goes red when a consumer breaks. Under SD1 (deprecate, then remove) a red run means the PR changes API that a consumer's `main` still uses: deprecate it instead, or hold the removal until both consumers have migrated.
- `AGENTS.md` describes the job in a paragraph after "Workflow security", and that paragraph's `contents: read` list names `consumer-build.yml`.

Not in this change: running the consumers' test suites (cli's need a kind cluster), the API-diff check (its own change, add-api-diff-check), the Dependabot `cuelang.org/go` ignores in cli and opm-operator, and the `RELEASING.md` rule for `cuelang.org/go`, all of which the same decision j4 assigns elsewhere. No Taskfile task is added.

## Capabilities

### New Capabilities

- `consumer-build`: library PRs build and vet cli and opm-operator against the PR's merge commit, non-blocking, without touching any tracked `go.mod`.

### Modified Capabilities

None. The workflow follows the existing `workflow-hardening` requirements (read-only grants, `persist-credentials: false`, no PR job with a write grant) without changing them.

## Impact

- No `opm/` package changes; no public surface or SemVer effect. Commits are `ci` and `docs`, which release-please hides, so the change cuts no release.
- Library CI cost: two extra jobs on PRs that touch Go code, each a checkout, a Go install and a build plus vet of one consumer (locally about 7 to 8 seconds each with a warm module cache).
- cli and opm-operator are read, never written: both repos are public and are cloned with the job's read-only token.
- `.tasks/consumer-build.sh` and the workflow fall under the existing CODEOWNERS lines for `/.tasks/` and `/.github/`.
