## Why

`.github` main is now `0f9c6ac2c9b752a79f4874f637ef9955bcf00c13`. Since the library's pin `6938f8e` it carries `.github` PR 16 (a single code owner), PR 17 (owner decision 37: which repos move when, written into the README's rollout steps) and PR 19: `cascade-publish` syncs the cli publish mirror, refuses a push or recreate when a receiver's mirrored `.tasks/cascade/` files (`mirror_sources` in `wiring/lib.sh`, sha256 of the library's `pins.sh`, `lib.sh`, `classes` and `cascade.sh` read at `ca7c56b`) differ on its `origin/main`, adds the daily `cascade-mirror-drift.yml`, and the resolver's `newest` checks opm-operator's `opm_operator-v*` module tags are on its `main`. The library still pins `6938f8e`, so its cascade runs neither the mirror-drift refusal nor the updated resolver.

## What Changes

- Every `open-platform-model/.github` reference (the `cascade-notify`, `cascade-receive.yml`, `cascade-publish` and `cascade-gates.yml` `uses:` lines and the resolver `ref:` in `cascade-task.yml`) moves to `0f9c6ac2c9b752a79f4874f637ef9955bcf00c13 # .github main`.
- `.tasks/cascade/wiring-check.sh` becomes the canonical file at that SHA, byte for byte. That file is unchanged between `6938f8e` and `0f9c6ac`, so the copy is already identical; the check still compares it against the new pin.
- Nothing else. The README at `0f9c6ac` changes no caller shape or input and no library value in its config table, so `.tasks/cascade/wiring-check.yaml` and `test.yml` stay as they are. The library's four mirrored files hash to the values in `mirror_sources` on `main` (`ca7c56b`), so publish will not refuse the library for mirror drift.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `cascade-wiring`: the reference requirement cites the `.github` README at the new pin; its behavior is unchanged.

## Impact

**SemVer: no release.** Nothing under `opm/` changes; every commit is `ci` or `chore`. Affected files: `.github/workflows/{release,deps-cascade,cascade-gates,cascade-task}.yml`, `.tasks/cascade/wiring-check.sh` (byte-identical replacement). No downstream consumer acts.

**Rollout: the library is the canary and moves alone.** `.github` PR 19 changes `wiring/lib.sh`, which both `cascade-notify` and `cascade-publish` run, so under the README's "Which repos move when" rule (owner decision 37, "Both actions") one upstream moves first. The library is that canary: after this PR merges, the README step-2 dry-run checks run in the library (`gh workflow run deps-cascade.yml` succeeds with `Compute` showing `scripts from open-platform-model/.github 0f9c6ac2c9b752a79f4874f637ef9955bcf00c13`, `Publish` skipped, a summary matching a local `task -x deps:cascade`, and a passing `cascade-task.yml` dispatch). core, catalog_opm, opm-operator and cli stay on `6938f8e` until the library's first live `cascade-notify` run on the new pin, on its next release, has succeeded. While every receiver is dry-run, every receiver but the Phase 4 canary also stays dry until that canary's first live publish on the new pin has succeeded. `CASCADE_DRY_RUN` stays as it is.
