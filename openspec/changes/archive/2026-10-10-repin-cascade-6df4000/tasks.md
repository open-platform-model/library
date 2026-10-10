## 1. Move the pin

- [x] 1.1 Replace `0f9c6ac2c9b752a79f4874f637ef9955bcf00c13` with `6df4000f6cabbf460a5cad6431b6bc9044fa3027` on every `.github` reference in `.github/workflows` (four `uses:` and the `cascade-task.yml` `ref:`), keeping ` # .github main`. Verify: `grep -rn -A1 'open-platform-model/.github' .github/workflows` shows only the new SHA.
- [x] 1.2 Confirm `.tasks/cascade/wiring-check.sh` needs no replacement: the canonical file at `6df4000` hashes the same as the copy (`sha256sum`), and `git diff 0f9c6ac 6df4000 -- .github/scripts/cascade/wiring-check.sh` in `.github` is empty.
- [x] 1.3 Confirm `.tasks/cascade/wiring-check.yaml` and `test.yml` need no change against the README at the new SHA (config table row for the library, CI workflow and job `env`, steps before the wiring step), and that the library's `.tasks/cascade/{pins.sh,lib.sh,classes,cascade.sh}` hash to the library row of `mirror_sources` in `wiring/lib.sh` at the new SHA.
- [x] 1.4 Verify: `task cascade:wiring:check` prints `cascade wiring: ok, .github 6df4000f… (.github main)` and exits 0; `actionlint` is clean on the four workflows. Commit `ci(deps): pin the cascade to .github 6df4000`.

## 2. Verify and archive

- [x] 2.1 `openspec validate repin-cascade-6df4000 --strict` passes and the verify skill reports no CRITICAL finding.
- [x] 2.2 `openspec archive repin-cascade-6df4000 -y`; `openspec validate --all --strict` passes. Commit `chore(openspec): archive repin-cascade-6df4000`.
