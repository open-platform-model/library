## 1. ci: build opm-controller as a consumer

- [x] 1.1 `.github/workflows/consumer-build.yml`: matrix value, step name, `if:` and checkout `repository:` name `opm-controller`; the header comment follows. Keep the literal checkout form.
- [x] 1.2 `.tasks/consumer-build.sh` comments name `opm-controller`. Leave `.tasks/cascade/wiring-check.sh` (the `.github` copy) untouched.
- [x] 1.3 `bash .tasks/consumer-build-test.sh` and `task cascade:wiring:check` pass; `task check` green, then commit `ci(consumer-build): build opm-controller in place of opm-operator`.

## 2. Godoc, lint messages and tests name the controller

- [x] 2.1 Comments and godoc under `opm/` (`helper/objectset`, `helper/platformmodule`, `internal/loader`, `k8s/inventory`, `k8s/labels`, `k8s/object`, `k8s/ownership`, `kernel`, `platform`) say "controller" for the product and `opm-controller` for the repository; `opm/schema/loader.go`'s "Operators in restricted environments" stays.
- [x] 2.2 Test messages (`kernel/parity_harness_test.go`, `kernel/render_runtime_name_test.go`), the hold-message banned word in `k8s/lifecycle/hold_test.go` (`controller`), and the comment in `testdata/render/scenarios/bad_traits/instance.cue`.
- [x] 2.3 `.golangci.yml`: the header comment and the depguard `desc` messages say "the controller".
- [x] 2.4 `task check` green, then commit `docs: name the controller in godoc, lint messages and tests`.

## 3. Prose and spec Purposes name the controller

- [ ] 3.1 `README.md`, `AGENTS.md`, `CONSTITUTION.md` (the pre-GA clause), `openspec/config.yaml` (the same clause), `MIGRATIONS.md`, `migrations/README.md`, `.claude/skills/{verify,security-audit}/SKILL.md`, and `.github/workflows/release.yml`'s comment.
- [ ] 3.2 ADR bodies 002, 004, 005, 007, 008, 010 and 011; `docs/design/*` (product wording and the relative links to `../../../opm-controller/...`).
- [ ] 3.3 `docs/site/diagnostics/*` (`Check against:` pointers, "Controller conditions entry", product wording) and `docs/site/embedding/embed-the-kernel.md`.
- [ ] 3.4 The active change `publish-go-api-bundle` (proposal, design, tasks) names `opm-controller`.
- [ ] 3.5 Hand-edit the Purpose of `consumer-build`, `kernel-runtime` and `registry-module-loading` (a delta Purpose is ignored for an existing spec).
- [ ] 3.6 Residue check: `git grep -nIiE 'opm[-_ ]?operator|operator'` outside `CHANGELOG.md`, `openspec/changes/archive/**` and this change's own spec deltas leaves only the human-role hits, ADR-001 and `.tasks/cascade/wiring-check.sh`; `openspec validate --all --strict` green; `task check` green, then commit `docs: name the controller in the prose`.
