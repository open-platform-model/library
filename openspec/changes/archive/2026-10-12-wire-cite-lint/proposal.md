## Why

Every repo of the organization runs the same comment rule: an enhancement id stands beside a reason in words, never as a decision number, never alone. `.github` now offers the reusable workflow `cite-lint.yml` (PR 22, commit `2b66da51`) and the library has no local task or CI job for it. A later citation by decision number, or a lone id, must fail the pull request that adds it.

## What Changes

- Every `open-platform-model/.github` reference (the `cascade-notify`, `cascade-receive.yml`, `cascade-publish` and `cascade-gates.yml` `uses:` lines and the resolver `ref:` in `cascade-task.yml`) moves from `6df4000f` to `2b66da51fede4332969c324b069012632d8edea9 # .github main`.
- `.tasks/cascade/wiring-check.sh` is replaced by the canonical file at `2b66da51` (byte for byte). It adds the `cite-lint` reference kind.
- `.tasks/cascade/wiring-check.yaml` declares the `cite-lint` call in `lint.yml`.
- `Taskfile.yml` gets the task `cite-lint`. It runs the `enh` tool of `open-platform-model/enhancements` at commit `e0335b2e` (the commit `cite-lint.yml` uses) with `go run`.
- `.github/workflows/lint.yml` gets a job `cite-lint` that calls the reusable workflow at the `.github` pin. It runs on `pull_request` and on `push` to `main`.
- Excludes: none. The tool reports 0 findings on the whole tree.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. The change has no spec delta and `.openspec.yaml` sets `skip_specs: true`.

## Impact

**SemVer: no release.** Nothing under `opm/` changes; every commit is `ci`, `build`, `docs` or `chore`.

**Behavior note.** `.github` changed between `6df4000f` and `2b66da51` in the wiring check, the `cite-lint` workflow and docs only. The cascade workflows and actions the library calls are unchanged, so cascade behavior stays the same.
