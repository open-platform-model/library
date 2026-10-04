## 1. Consumer build script and workflow

- [ ] 1.1 `.tasks/consumer-build.sh <consumer-dir> <library-dir> <work-dir>`: `go work init` in the work directory, `go build ./...` then `go vet ./...` in the consumer with `GOWORK` set, output logged; on failure a job-summary section (consumer, consumer commit, failing step, first 50 compiler error lines) when `GITHUB_STEP_SUMMARY` is set, a `::warning` naming the consumer, non-zero exit; finally `git status --porcelain` empty in both checkouts
- [ ] 1.2 Rehearse the script locally against fresh exports of cli and opm-operator `origin/main` in the scratchpad: both pass, and both trees stay clean
- [ ] 1.3 Rehearse a break: in a scratch copy of the library, remove one exported `opm/` function cli calls, run the script with `GITHUB_STEP_SUMMARY` pointed at a scratch file, and check the summary names the function and the cli commit; record the result in design.md
- [ ] 1.4 `.github/workflows/consumer-build.yml`: `on: pull_request` with the `paths:` filter from design.md, workflow `permissions: contents: read`, per-PR `concurrency` with `cancel-in-progress`, same-repo `if:`, matrix `consumer: [cli, opm-operator]` with `fail-fast: false`, job name `Consumer build (<consumer>)`, library checkout at `library/` and consumer checkout of `main` at `consumer/` (both `persist-credentials: false`), `actions/setup-go` at `test.yml`'s SHA with `go-version-file: consumer/go.mod` and `cache-dependency-path: consumer/go.sum`, the script run with `GOTOOLCHAIN=local` and a work directory under `$RUNNER_TEMP`
- [ ] 1.5 actionlint on the new workflow, `task cascade:wiring:check` and `task check` (tests with a private `TMPDIR`) green, then commit ci: build cli and opm-operator against the library PR head

## 2. Documentation

- [ ] 2.1 `AGENTS.md`: a CI paragraph naming the non-required `Consumer build` jobs, what triggers them, the throwaway `go.work`, how to read a red run (a `feat!` break is expected), and how to run `.tasks/consumer-build.sh` locally
- [ ] 2.2 `task check` and `openspec validate add-consumer-build-job --strict` green, then commit docs: describe the consumer build job in AGENTS.md
