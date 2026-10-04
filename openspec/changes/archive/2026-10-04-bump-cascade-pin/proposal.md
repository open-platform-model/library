## Why

`.github` `bound-cascade-publish` (`.github` PR 12, squash `7b9ad1bea132f7a3f053a5db61ac3933b59ee226`) bounds what the cascade `publish` step accepts, pins compute's tools and adds a required `gates-only` input to `cascade-publish`. It also makes the wiring check one canonical script: every product repo runs a byte-identical copy at `.tasks/cascade/wiring-check.sh` with its own values in `.tasks/cascade/wiring-check.yaml`, and its required CI step passes `--pin-on-main`, so a SHA that exists only in a fork of `.github` is refused (`.github` README, "Pinning and bumps" and "The wiring check").

The library still pins `.github` `2376ffa` and runs its own older check offline. Until it moves, none of the publish bounds apply to the library's cascade, and the publish job has no gates-only clause. The new `cascade-publish` refuses a missing `gates-only` input, so the pin and the caller edit must move together.

## What Changes

- Every `open-platform-model/.github` reference (the `cascade-notify`, `cascade-receive.yml`, `cascade-publish` and `cascade-gates.yml` `uses:` lines and the resolver `ref:` in `cascade-task.yml`) moves to `7b9ad1bea132f7a3f053a5db61ac3933b59ee226 # .github main`.
- `.tasks/cascade/wiring-check.sh` becomes the canonical file at that SHA, byte for byte. A new `.tasks/cascade/wiring-check.yaml` holds the library's row of the README table: receiver, empty `env-allow`, `publish-workflows: [release.yml, docs.yml]`, CI `test.yml` job `test`, notify `needs: [release-please]` with `release.yml`'s own `if:` and `tag`, `labels-managed: false`.
- `deps-cascade.yml` `publish`: the `if:` gains `&& inputs.gates_only != true`, and the `Publish` step passes `gates-only: ${{ inputs.gates_only == true }}`.
- `test.yml` step "Verify the cascade wiring" runs `bash .tasks/cascade/wiring-check.sh --pin-on-main` with `GH_TOKEN: ${{ github.token }}` (a read of a public repo; the job stays `contents: read`). `task cascade:wiring:check` stays the offline local entry.
- `.github/dependabot.yml`: the `github-actions` entry gets `cooldown: {default-days: 7}`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `cascade-wiring`: the publish job's inputs and `if:` carry the gates-only switch; the wiring check is the canonical copy with a config and runs with `--pin-on-main` in CI.

## Impact

**SemVer: no release.** Nothing under `opm/` changes; every commit is `ci` or `chore`, which release-please hides. Affected files: `.github/workflows/{release,deps-cascade,cascade-gates,cascade-task,test}.yml`, `.github/dependabot.yml`, `.tasks/cascade/wiring-check.{sh,yaml}`, `Taskfile.yml` (description only). No downstream consumer acts.

Depends on `.github` PR 12 merged (done) and library `harden-release-workflows` (PR 181, merged). `CASCADE_DRY_RUN` stays as it is: the library's receiver is a dry run, so `Publish` is skipped and the new publish bounds first meet real GitHub on the canary's first live run (supervisor's canary rule, all five receivers move together while dry).
