## Why

`.github` main is now `6df4000f6cabbf460a5cad6431b6bc9044fa3027`. Since the library's pin `0f9c6ac` it carries `.github` PR 20, which accepts `opm-controller` beside `opm-operator` in every map of `wiring/lib.sh` (`notify_targets`, `receiver_sources`, `publish_paths`, `receiver_classes`, `mirror_sources` and the others), in the resolver's tag sources and in the pin parsers. The GitHub repo `opm-operator` is being renamed `opm-controller`. A run derives the repo name from `GITHUB_REPOSITORY`, so the name flips the moment GitHub renames the repo, and a cascade pinned before PR 20 fails closed on the new name. The library still pins `0f9c6ac`, so its `cascade-notify`, `cascade-receive.yml`, `cascade-publish` and `cascade-gates.yml` do not know the new name.

## What Changes

- Every `open-platform-model/.github` reference (the `cascade-notify`, `cascade-receive.yml`, `cascade-publish` and `cascade-gates.yml` `uses:` lines and the resolver `ref:` in `cascade-task.yml`) moves to `6df4000f6cabbf460a5cad6431b6bc9044fa3027 # .github main`.
- `.tasks/cascade/wiring-check.sh` stays as it is. `.github` PR 20 does not change `.github/scripts/cascade/wiring-check.sh`: the copy hashes to `1c46bd25…` at `0f9c6ac` and at `6df4000`, and `cmp` reports no difference.
- Nothing else. The README at `6df4000` adds only the "opm-operator rename" paragraph and the `opm-controller` names in two table rows. It changes no caller shape or input and no library value in its config table, so `.tasks/cascade/wiring-check.yaml` and `test.yml` stay as they are. The library's four mirrored files (`pins.sh`, `lib.sh`, `classes`, `cascade.sh`) still hash to the library row of `mirror_sources` at `6df4000`, so publish will not refuse the library for mirror drift.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `cascade-wiring`: the reference requirement cites the `.github` README at the new pin; its behavior is unchanged.

## Impact

**SemVer: no release.** Nothing under `opm/` changes; every commit is `ci` or `docs`/`chore`. Affected files: `.github/workflows/{release,deps-cascade,cascade-gates,cascade-task}.yml`. No downstream consumer acts.

**Rollout: the library moves first.** The owner merges this pull request before the same pin pull requests in opm-operator and cli (pin round a). The `.github` README rule for a change to `wiring/lib.sh` (owner decision 37, "Both actions") asks the other repos to wait for the first live `cascade-notify` run of one canary on the new pin. The owner waived that canary once, on 2026-10-09, for this `.github` change (PR 20). core and catalog_opm get no pin in this round. `CASCADE_DRY_RUN` stays as it is.

**Behavior note.** The library is not a receiver of the renamed repo, and its own cascade behavior stays the same with one exception. At `6df4000`, `changelog_repos` for the library also lists `opm-controller`, so a library compute run that is not skipped tries to read the releases of that repo. The read fails with a "cannot read the releases" note until the rename, and changes nothing else, because the library pins no controller version.
