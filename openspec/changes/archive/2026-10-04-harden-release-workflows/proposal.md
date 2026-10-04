## Why

The 2026-10-04 security pass over the release cascade (supervisor plan "Release cascade security pass", owner selections 28 to 31) found four gaps in the library's own workflows:

- `RELEASE_APP_PRIVATE_KEY` is an organization secret that any workflow on any branch can read on a push event. The key mints a token that can create tags, and the library's tags are permanent on proxy.golang.org (finding GOV-2). Owner selection 29 moves it into a `main`-only Environment `release`, which the supervisor has already created in this repo.
- `cue.yml`, `lint.yml` and `test.yml` declare no `permissions:`, and `release.yml` grants `contents: write` and `pull-requests: write` to the whole workflow. While the repo default token is write, a same-repo PR (the `deps/cascade` PR included) runs `go test`, CUE tooling and the linter with a write token persisted in `.git/config` (findings N4, GOV-3). Owner selection 30 flips the repo default to read-only once every workflow states what it needs.
- `lint.yml` pipes `install.sh` from golangci-lint's mutable `HEAD` into `sh` on every push to `main`. The script verifies its own download, so a compromised `HEAD` runs with the job's token unchecked (governance review, missed item 1).
- No `CODEOWNERS` exists, so the code-owner review the `main` ruleset will require (owner selection 28) has nothing to bind to.

## What Changes

- `release.yml`: the `release-please` job, the only reader of `RELEASE_APP_PRIVATE_KEY`, declares `environment: release`. The workflow drops its write grants for `permissions: {}`; `release-please` gets none (it acts with the App token only), and `publish-docs` and `notify-downstream` keep their own job grants.
- `cue.yml`, `lint.yml`, `test.yml`: `permissions: contents: read`, and their checkouts set `persist-credentials: false`. `cascade-task.yml` gets the same on its repo checkout.
- `lint.yml` downloads the golangci-lint v2.8.0 release archive by URL and checks it against a sha256 committed in the workflow before it runs anything from it.
- `.github/CODEOWNERS` names the two owners for `/.github/`, `/.tasks/`, `/Taskfile*.yml`, the release-please config and manifest, and `/.cascade-frozen`.
- `AGENTS.md` states the release key's Environment, the read-only workflow tokens, the lint pin and the code-owner paths.

Re-checked and left alone: no library workflow that publishes uses an Actions cache (`release.yml` has no setup step; `docs.yml` calls docs-kit's `publish.yml`), no PR-triggered workflow holds a write grant once the defaults above are explicit, and `.github/dependabot.yml` already covers `github-actions` and ignores `open-platform-model/.github*`. The `.github` pin, `.tasks/cascade/wiring-check.sh` and `deps-cascade.yml` are wave 2's; this change needs no edit to the wiring check, since it matches only the `cascade` Environment.

Not in this change: moving the secret into the Environment (owner), the ruleset and repo-settings edits (supervisor), and pinning Task's `version: 3.x` installs, which the plan assigns to `.github` and opm-operator.

## Impact

- No `opm/` package changes; no public surface or SemVer effect. Commits are `ci`, which release-please hides, so the change cuts no release.
- Until the owner stores `RELEASE_APP_PRIVATE_KEY` in `release`, the job reads the organization secret through the Environment, so the change is safe to merge now.
- The gate holds only once the organization secret no longer reaches the library. An Environment secret of the same name takes precedence only inside that Environment, so a workflow on any branch that skips the Environment reads the organization secret for as long as it lists this repository. The owner action is two steps: store the key in `release`, then remove the library from the organization secret's selected repositories (or delete the organization secret), and check that the repository's organization-secrets listing no longer shows it.
- Downstream consumers (cli, opm-operator) are unaffected.
