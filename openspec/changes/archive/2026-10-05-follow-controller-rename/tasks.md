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

- [x] 3.1 `README.md`, `AGENTS.md`, `CONSTITUTION.md` (the pre-GA clause), `openspec/config.yaml` (the same clause), `MIGRATIONS.md`, `migrations/README.md`, `.claude/skills/{verify,security-audit}/SKILL.md`, and `.github/workflows/release.yml`'s comment.
- [x] 3.2 ADR bodies 002, 004, 005, 007, 008, 010 and 011; `docs/design/*` (product wording and the relative links to `../../../opm-controller/...`).
- [x] 3.3 `docs/site/diagnostics/*` (`Check against:` pointers, "Controller conditions entry", product wording) and `docs/site/embedding/embed-the-kernel.md`.
- [x] 3.4 The active change `publish-go-api-bundle` (proposal, design, tasks) names `opm-controller`.
- [x] 3.5 Hand-edit the Purpose of `consumer-build`, `kernel-runtime` and `registry-module-loading` (a delta Purpose is ignored for an existing spec).
- [x] 3.6 Residue check: `git grep -nIiE 'opm[-_ ]?operator|operator'` outside `CHANGELOG.md`, `openspec/changes/archive/**` and this change's own spec deltas leaves only the human-role hits, ADR-001, `.tasks/cascade/wiring-check.sh` and the main spec bodies this change's deltas move at archive; `openspec validate --all --strict` green; `task check` green, then commit `docs: name the controller in the prose`.

## 4. Rebase over library #217, #218, #219 and #220 (2026-10-06)

- [x] 4.1 ADR-013 (added by #219): its rows and prose name `opm-controller` and the controller; PR references follow as `opm-controller#N` (GitHub redirects the old name).
- [x] 4.2 `opm/kernel/required_config_test.go` (added by #217): its four comments name the controller.
- [x] 4.3 The `consumer-build` deltas carry #219's `Source: ADR-013, decision j4.`; the `kernel-runtime` sync keeps #217's appended requirement before the re-added SynthesizeInstance requirement. No main-spec wording that names the old product arrived outside these deltas, so no follow-up change is needed.
- [x] 4.4 `task check` and `openspec validate --all --strict` green, then commit `docs: name the controller in ADR-013 and the required-config tests`.
