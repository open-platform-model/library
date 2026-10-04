## 1. The api:diff task

- [ ] 1.1 `.tasks/apidiff/`: a tools module (`go.mod`, committed `go.sum`, a `tool` directive or a blank-import `tools.go` behind a build tag) pinning `golang.org/x/exp/cmd/apidiff` at the newest pseudo-version whose `go` directive does not exceed the library's `go.mod` directive; record the chosen version and why in a comment in its `go.mod`
- [ ] 1.2 `.tasks/apidiff/allow.txt`: one commented extended regular expression for the `schema.DefaultSchemaModule` value change
- [ ] 1.3 `.tasks/api-diff.sh` (`set -euo pipefail`): build the tool with `go build -C .tasks/apidiff` into a temporary directory; resolve the base tag (`BASE`, else `git describe --tags --abbrev=0 --match 'v*'` on `API_DIFF_BASE_REF` or `HEAD`), failing with a "fetch the tags" hint when none is reachable; export both APIs (a `git archive` copy for the base, the work tree for the head); run `apidiff -m -incompatible`; drop allow-listed lines; derive `warn` or `block` from the tag's prerelease suffix; print, annotate and write `$GITHUB_STEP_SUMMARY` (only under `GITHUB_ACTIONS`) with the base tag, mode, entries and the `feat!` / `BREAKING CHANGE:` remedy; exit 0 in `warn`, 1 in `block` with entries; remove its temporary directory on exit (`chmod -R u+w` first, the module cache extracts read-only)
- [ ] 1.4 `Taskfile.yml`: `api:diff` with a `desc`, passing `BASE` through
- [ ] 1.5 Run locally and record the outputs in the commit body: `task api:diff` (nearest tag `v1.0.0-beta.4`, expect compatible), `task api:diff BASE=v1.0.0-alpha.33` (expect nothing after the allow list; re-run with the allow list emptied to see the warn path), `task api:diff BASE=v0.7.0` (expect `block` and exit 1); confirm `go build ./...`, `go vet ./...` and `task lint` at the root ignore the nested module
- [ ] 1.6 `task fmt`, `task vet`, `task lint`, `task test` (private `TMPDIR`) and `task cascade:wiring:check` green, then commit ci: add the api:diff task

## 2. The API diff workflow

- [ ] 2.1 `.github/workflows/api-diff.yml`: `on: pull_request` with the `paths:` filter from design.md; workflow `permissions: contents: read`; job `API diff` on `ubuntu-latest`, `timeout-minutes: 10`; checkout, `setup-go` (`go-version-file: go.mod`) and `setup-task` at the SHAs `lint.yml` pins, checkout with `fetch-depth: 0` and `persist-credentials: false`; one step running `task api:diff` with `API_DIFF_BASE_REF: ${{ github.event.pull_request.base.sha }}` in its `env:`; no `continue-on-error`
- [ ] 2.2 actionlint on the new workflow and `task cascade:wiring:check` green, then commit ci: warn on breaking api changes in pull requests

## 3. Documentation

- [ ] 3.1 `AGENTS.md`: a `task api:diff` line under "Core commands", and one paragraph after "Workflow security" naming the workflow, the base-tag rule (nearest reachable, so `v0.7.0` never decides), the warn/block derivation, the tools module and its hand-moved pin, the allow list, and that the job is not required (rebase on `add-consumer-build-job` if it merged first)
- [ ] 3.2 `openspec validate add-api-diff-check --strict` and `task check` green, then commit docs: describe the api diff check
