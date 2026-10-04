## 1. Move the pin and adopt the canonical wiring check

- [x] 1.1 Replace `2376ffae4bfc665f327d51581350dea694c01504` with `7b9ad1bea132f7a3f053a5db61ac3933b59ee226` on every `.github` reference in `.github/workflows` (four `uses:` and the `cascade-task.yml` `ref:`), keeping ` # .github main`. Verify: `grep -rn -A1 'open-platform-model/.github' .github/workflows` shows only the new SHA.
- [x] 1.2 Copy `.github/scripts/cascade/wiring-check.sh` at the new SHA to `.tasks/cascade/wiring-check.sh`. Verify: `git -C ../.github show <SHA>:.github/scripts/cascade/wiring-check.sh | cmp - .tasks/cascade/wiring-check.sh`.
- [x] 1.3 Write `.tasks/cascade/wiring-check.yaml` from the README table, with `notify.if` and `notify.tag` read from `release.yml`.
- [x] 1.4 `deps-cascade.yml` `publish`: add `&& inputs.gates_only != true` to the `if:` and `gates-only: ${{ inputs.gates_only == true }}` to the `Publish` step.
- [x] 1.5 `test.yml`: the step "Verify the cascade wiring" runs `bash .tasks/cascade/wiring-check.sh --pin-on-main` with `GH_TOKEN: ${{ github.token }}`; update the comments that describe it (`test.yml`, `deps-cascade.yml`, `Taskfile.yml` description).
- [x] 1.6 `.github/dependabot.yml`: `cooldown: {default-days: 7}` on the `github-actions` entry.
- [x] 1.7 Verify: `bash .tasks/cascade/wiring-check.sh --pin-on-main` prints `cascade wiring: ok, .github 7b9ad1b… (.github main)`; `task cascade:wiring:check` passes; `CASCADE_TEST_SET=offline task -x deps:cascade:test` passes; `actionlint` clean; `gh api repos/open-platform-model/.github/compare/<SHA>...main --jq .status` prints `identical` or `ahead`. Commit `ci(deps): pin the cascade to .github 7b9ad1b`.

## 2. Verify and archive

- [x] 2.1 `openspec validate bump-cascade-pin --strict` passes and the verify skill reports no CRITICAL finding.
- [x] 2.2 `openspec archive bump-cascade-pin -y`; `openspec validate --all --strict` passes. Commit `chore(openspec): archive bump-cascade-pin`.
