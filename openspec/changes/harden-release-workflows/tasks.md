## 1. Release key in the release Environment

- [x] 1.1 `release.yml`: workflow `permissions: {}`; the `release-please` job declares `environment: release` and `permissions: {}`; `publish-docs` and `notify-downstream` keep their grants
- [x] 1.2 `task cascade:wiring:check` and actionlint green, then commit ci(release): read the release app key only in the release environment

## 2. Read-only tokens for the CI workflows

- [x] 2.1 `cue.yml`, `lint.yml`, `test.yml`: workflow `permissions: contents: read`
- [x] 2.2 `persist-credentials: false` on every checkout in `cue.yml`, `lint.yml`, `test.yml` and `cascade-task.yml`
- [x] 2.3 Re-check every workflow: no `pull_request` job holds a write grant, no publishing workflow uses an Actions cache, `dependabot.yml` covers `github-actions` and ignores `open-platform-model/.github*`
- [x] 2.4 actionlint green, then commit ci: give the ci workflows read-only tokens

## 3. Checksum-verified golangci-lint

- [x] 3.1 `lint.yml`: replace the `HEAD` install script with the v2.8.0 release archive, fetched over HTTPS and checked with `sha256sum -c` against the committed digest
- [x] 3.2 Run the install step's shell locally (download, verify, extract, `golangci-lint version`) and `task lint`, then commit ci(lint): install golangci-lint from a checksum-verified release

## 4. Code owners and documentation

- [x] 4.1 `.github/CODEOWNERS` with the six owner lines
- [x] 4.2 `AGENTS.md`: the release Environment, read-only workflow tokens, the lint pin and the code-owner paths
- [x] 4.3 `task check` and `openspec validate harden-release-workflows --strict` green, then commit ci: require code-owner review of release and cascade files
