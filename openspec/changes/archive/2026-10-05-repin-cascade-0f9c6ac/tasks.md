## 1. Move the pin and replace the wiring check copy

- [x] 1.1 Replace `6938f8e0247e019cb0c2db13fff5b7b558a6b67d` with `0f9c6ac2c9b752a79f4874f637ef9955bcf00c13` on every `.github` reference in `.github/workflows` (four `uses:` and the `cascade-task.yml` `ref:`), keeping ` # .github main`. Verify: `grep -rn -A1 'open-platform-model/.github' .github/workflows` shows only the new SHA.
- [x] 1.2 Copy `.github/scripts/cascade/wiring-check.sh` at the new SHA to `.tasks/cascade/wiring-check.sh`. Verify: `gh api -H 'Accept: application/vnd.github.raw' "repos/open-platform-model/.github/contents/.github/scripts/cascade/wiring-check.sh?ref=<SHA>" | cmp - .tasks/cascade/wiring-check.sh`.
- [x] 1.3 Confirm `.tasks/cascade/wiring-check.yaml` and `test.yml` need no change against the README at the new SHA (config table row for the library, CI workflow and job `env`, steps before the wiring step), and that the library's `.tasks/cascade/{pins.sh,lib.sh,classes,cascade.sh}` hash to `mirror_sources` in `wiring/lib.sh` at the new SHA.
- [x] 1.4 Verify: `bash .tasks/cascade/wiring-check.sh --pin-on-main` prints `cascade wiring: ok, .github 0f9c6ac… (.github main)`; `CASCADE_TEST_SET=offline task -x deps:cascade:test` passes; `actionlint` clean; `task check` passes. Commit `ci(deps): pin the cascade to .github 0f9c6ac`.

## 2. Verify and archive

- [x] 2.1 `openspec validate repin-cascade-0f9c6ac --strict` passes and the verify skill reports no CRITICAL finding.
- [x] 2.2 `openspec archive repin-cascade-0f9c6ac -y`; `openspec validate --all --strict` passes. Commit `chore(openspec): archive repin-cascade-0f9c6ac`.
