## 1. Split the long block of publish-go-api-bundle

- [x] 1.1 Copy `openspec/changes/publish-go-api-bundle/specs/docs-bundle/spec.md` to a scratch folder. In the original, replace the ADDED block "The library publishes one docs bundle" with the two blocks of design.md: sentence 1 under the old header with the scenario "Internal packages stay out"; sentences 2 and 3 under "The bundle carries both kinds of page and its tasks build it" with the scenario "The bundle holds both kinds of page". Touch no other block and no other file of the change.
- [x] 1.2 Compare the scratch copy and the new file: the sentences and both scenarios are byte equal, and only the new header and the block break differ. Both bodies are 500 characters or less and hold SHALL.
- [x] 1.3 `openspec validate --all --strict --no-interactive` under 1.14.1 exits 0, and `openspec validate publish-go-api-bundle --strict --no-interactive` exits 0.
- [x] 1.4 Commit `docs(openspec): split the long docs-bundle requirement of publish-go-api-bundle`

## 2. Add the pinned gate

- [ ] 2.1 `Taskfile.yml`: add `OPENSPEC_VERSION: 1.14.1` to `vars`, the tasks `openspec:install` and `openspec:check` as in design.md, and `openspec:check` at the end of the `check` task.
- [ ] 2.2 `.github/workflows/lint.yml`: add the job `openspec` as in design.md, with the action pins already used in the file and the `setup-node` pin of `core`. `permissions` stay `contents: read`.
- [ ] 2.3 `task openspec:install` and `task check` pass; `task openspec:check` exits 0. A scratch copy with a requirement body over 500 characters makes `task openspec:check` exit 1. `actionlint` and `zizmor` are clean on the changed workflow; the `yq` timeout query of the workflow guide is answered for the new job.
- [ ] 2.4 Commit `ci(openspec): validate every spec and change in strict mode`

## 3. Verify and archive

- [ ] 3.1 `openspec validate add-openspec-strict-gate --strict --no-interactive` passes.
- [ ] 3.2 Archive the change with `skip_specs: true` (no spec sync) and commit `chore(openspec): archive add-openspec-strict-gate`.
