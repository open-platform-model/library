## ADDED Requirements

### Requirement: Library PRs build the consumers against the PR's merge commit

A pull request from a branch in this repository that changes `opm/**`, `go.mod`, `go.sum`, the consumer-build workflow or its script SHALL run one job per consumer, cli and opm-operator. Each job SHALL check out the consumer's `main`, install the Go version the consumer's `go.mod` names, and run `go build ./...` and `go vet ./...` in the consumer with the library resolved to the PR's merge commit, the tree `actions/checkout` checks out by default for a pull request. Source: ADR-013, decision j4.

#### Scenario: A PR changes library code

- **WHEN** a same-repo pull request changes a file under `opm/`
- **THEN** `Consumer build (cli)` and `Consumer build (opm-operator)` run, each building and vetting that consumer's `main` against the PR's merge commit

#### Scenario: A PR changes only docs

- **WHEN** a pull request changes no file under `opm/`, neither `go.mod` nor `go.sum`, and neither the workflow nor its script
- **THEN** no consumer-build job runs

#### Scenario: A PR comes from a fork

- **WHEN** a pull request's head repository is not this repository
- **THEN** the consumer-build jobs are skipped, and no workflow runs them under `pull_request_target`

### Requirement: The library reaches the consumer only through a throwaway workspace

The job SHALL point the consumer at the library through a `go.work` created outside both checkouts and selected with `GOWORK`. No `replace` directive, `go.work` or `go.work.sum` SHALL be written into a tracked tree of either checkout, and the job SHALL fail if either checkout's `git status --porcelain` after the build differs from its status before the build, whether the build passed or failed. Each consumer SHALL get its own workspace. Source: ADR-013, decision j4.

#### Scenario: The build finishes

- **WHEN** a consumer build completes, whether it passed or failed
- **THEN** the consumer's `go.mod` and `go.sum` and the library's are unchanged, and the only `go.work` is in the runner's temporary directory

### Requirement: A consumer break is reported and never blocks a merge

The consumer-build jobs SHALL NOT be required checks. When a consumer fails to build or vet, its job SHALL fail, and the job summary SHALL name the consumer, the consumer commit, the failing step and the compiler's error lines, which name the broken symbol; the job SHALL also emit a warning annotation naming the consumer. One consumer's failure SHALL NOT cancel the other consumer's job. Source: ADR-013, decision j4.

#### Scenario: A PR removes an exported function a consumer calls

- **WHEN** a PR removes an exported `opm/` function that cli calls
- **THEN** `Consumer build (cli)` fails with a summary that shows the `undefined:` error naming that function and the cli commit it built, `Consumer build (opm-operator)` still runs to its own result, and the PR stays mergeable

#### Scenario: Both consumers build

- **WHEN** both consumers build and vet cleanly against the PR's merge commit
- **THEN** both jobs pass and write no failure summary

### Requirement: The consumer-build workflow holds only a read token

The consumer-build workflow SHALL follow the `workflow-hardening` requirements, SHALL pin every action by full commit SHA, and SHALL clone the public consumer repositories with no added secret. Source: ADR-013, decision j4.

#### Scenario: A same-repo PR runs the consumer build

- **WHEN** the consumer-build job runs on a same-repo pull request
- **THEN** it holds a `contents: read` token, reads no repository secret, and neither checkout's `.git/config` carries a credential
