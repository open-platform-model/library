## Why

`.github` main is now `6938f8e0247e019cb0c2db13fff5b7b558a6b67d`. It carries `.github` PR 14 (a background `git gc` no longer fails the resolver) and PR 15 (`verify-wiring-copy`): under `--pin-on-main` the wiring check now also compares the running copy byte for byte with `.github/scripts/cascade/wiring-check.sh` at the pinned SHA, restricts the CI workflow's and job's `env` and every step before the wiring step, and holds every checkout of `.github` to the pin (`.github` README at `6938f8e`, "Pinning and bumps", "The wiring check" and "Keeping the copy in sync"). The library still pins `7b9ad1b`, so its cascade runs the old resolver and its CI runs the old check, which never compares the copy.

## What Changes

- Every `open-platform-model/.github` reference (the `cascade-notify`, `cascade-receive.yml`, `cascade-publish` and `cascade-gates.yml` `uses:` lines and the resolver `ref:` in `cascade-task.yml`) moves to `6938f8e0247e019cb0c2db13fff5b7b558a6b67d # .github main`.
- `.tasks/cascade/wiring-check.sh` becomes the canonical file at that SHA, byte for byte.
- Nothing else. `.tasks/cascade/wiring-check.yaml` needs no new key (`extra-references` is optional and only opm-operator sets it), and `test.yml` already has the shape the new check requires: workflow `env` only `CUE_REGISTRY`, no job `env`, `container` or `services`, and only SHA-pinned actions (checkout, setup-go, setup-task) with `name`, `uses` and `with` before "Verify the cascade wiring".

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `cascade-wiring`: the CI wiring step also compares the copy with the file at the pin, and nothing before it may set its environment.

## Impact

**SemVer: no release.** Nothing under `opm/` changes; every commit is `ci` or `chore`. Affected files: `.github/workflows/{release,deps-cascade,cascade-gates,cascade-task}.yml`, `.tasks/cascade/wiring-check.sh`. Neither `.github` change touches `cascade-publish` or `cascade-notify`, so no canary live run is needed (README step 2); `CASCADE_DRY_RUN` stays as it is. No downstream consumer acts.
