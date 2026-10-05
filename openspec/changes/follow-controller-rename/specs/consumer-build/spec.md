## MODIFIED Requirements

### Requirement: Library PRs build the consumers against the PR's merge commit

A pull request from a branch in this repository that changes `opm/**`, `go.mod`, `go.sum`, the consumer-build workflow or its script SHALL run one job per consumer, cli and opm-controller. Each job SHALL check out the consumer's `main`, install the Go version the consumer's `go.mod` names, and run `go build ./...` and `go vet ./...` in the consumer with the library resolved to the PR's merge commit, the tree `actions/checkout` checks out by default for a pull request. Source: owner walkthrough decision j4 (2026-10-03).

#### Scenario: A PR changes library code

- **WHEN** a same-repo pull request changes a file under `opm/`
- **THEN** `Consumer build (cli)` and `Consumer build (opm-controller)` run, each building and vetting that consumer's `main` against the PR's merge commit

#### Scenario: A PR changes only docs

- **WHEN** a pull request changes no file under `opm/`, neither `go.mod` nor `go.sum`, and neither the workflow nor its script
- **THEN** no consumer-build job runs

#### Scenario: A PR comes from a fork

- **WHEN** a pull request's head repository is not this repository
- **THEN** the consumer-build jobs are skipped, and no workflow runs them under `pull_request_target`

### Requirement: A consumer break is reported and never blocks a merge

The consumer-build jobs SHALL NOT be required checks. When a consumer fails to build or vet, its job SHALL fail, and the job summary SHALL name the consumer, the consumer commit, the failing step and the compiler's error lines, which name the broken symbol; the job SHALL also emit a warning annotation naming the consumer. One consumer's failure SHALL NOT cancel the other consumer's job. Source: owner walkthrough decision j4 (2026-10-03).

#### Scenario: A PR removes an exported function a consumer calls

- **WHEN** a PR removes an exported `opm/` function that cli calls
- **THEN** `Consumer build (cli)` fails with a summary that shows the `undefined:` error naming that function and the cli commit it built, `Consumer build (opm-controller)` still runs to its own result, and the PR stays mergeable

#### Scenario: Both consumers build

- **WHEN** both consumers build and vet cleanly against the PR's merge commit
- **THEN** both jobs pass and write no failure summary
