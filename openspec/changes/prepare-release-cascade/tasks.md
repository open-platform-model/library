# Tasks: prepare-release-cascade

> CI and release configuration only; no `opm/` code. Three sections, each a hidden `ci`
> commit, plus the archive commit, shipped together in one PR. Workflow files are linted with
> `go run github.com/rhysd/actionlint/cmd/actionlint@latest` (or a local `actionlint`).

## 1. G1 release-pin gate

- [ ] 1.1 Add `deps:release-check` to `Taskfile.yml` (new "Release" group after the CUE tasks,
      `desc` + `summary`, preconditions on `jq` and `git`) with the four rules of design D-b;
      verify `task deps:release-check` exits 0 on the clean tree and prints a pass line.
- [ ] 1.2 Add `task deps:release-check   # G1 release-pin gate; CI runs it on release-please-- branches`
      under Core commands in `AGENTS.md` § Build And Dev Commands.
- [ ] 1.3 Prove each rule fails, on throwaway edits reverted with `git checkout -- <file>`
      before moving on: `go mod edit -replace example.com/x=../x`; `go mod edit -require
      github.com/open-platform-model/core@v0.0.0-20261001120000-abcdef123456`; a
      `-0.dev.1` version in `testdata/modules/web_app/cue.mod/module.cue`; a `-0.dev.1` suffix on
      `DefaultSchemaModule` in `opm/schema/loader.go`; a `-0.dev.1` suffix on
      `DefaultCoreVersion` in `opm/internal/registrytest/registrytest.go:113` (a non-test file
      whose name ends in `test.go`); an empty file
      `testdata/parity/cue.mod/local-module.cue` staged with `git add -N` (undone with
      `git reset -q <file> && rm <file>`). Verify each run exits
      non-zero and names the offending file, line or module, and that `git status` is clean
      afterwards.
- [ ] 1.4 Add the "Release-pin gate (G1)" step to `.github/workflows/test.yml` job `test`, after
      "Install Task" and before "Running Tests", guarded by
      `if: startsWith(github.head_ref || github.ref_name, 'release-please--')` with a comment
      citing workspace RELEASING.md, section "Gates"; verify actionlint reports nothing for
      `test.yml`.
- [ ] 1.5 `task check` green, then commit `ci(release): gate release PRs on dev and replaced pins`

## 2. Release outputs

- [ ] 2.1 In `.github/workflows/release.yml`, give the "Run release-please" step `id: release`
      and add job `outputs` `releases_created` and `tag_name` read from
      `steps.release.outputs`, matching `opm-operator/.github/workflows/release.yml:33-35`;
      verify actionlint reports nothing for `release.yml` and `grep -n "steps.release.outputs"`
      shows both outputs.
- [ ] 2.2 `task check` green, then commit `ci(release): expose release-please outputs for later jobs`

## 3. Docs-only commits stop releasing

> Depends on: opmodel.dev change `build-docs-from-branch-head` merged before this section's
> commit merges. Until then a docs-only fix in this repo reaches opmodel.dev only with the next
> release.

- [ ] 3.1 Set `"hidden": true` on the `docs` changelog section in `release-please-config.json`,
      leaving `refactor` at `"hidden": false`; verify with
      `jq '.packages["."]["changelog-sections"][] | select(.type=="docs" or .type=="refactor")' release-please-config.json`.
- [ ] 3.2 Correct `AGENTS.md` § Commit style (the release-please sentence at line 347): `docs`
      joins `chore`, `test`, `ci` and `build` as never releasing; `feat`, `fix`, `perf`,
      `revert`, `deps` and `refactor` release. Verify `grep -n "never release" AGENTS.md` shows the new list.
- [ ] 3.3 `task check` green and `openspec validate prepare-release-cascade --strict` passes, then
      commit `ci(release): stop docs-only commits from cutting a release`

## 4. Archive

- [ ] 4.1 Archive the change on this branch (`openspec archive prepare-release-cascade`), so the
      archive rides the implementing PR; never push to main (owner decision 2026-10-01,
      RELEASING.md "Owner settings").
- [ ] 4.2 `openspec validate --specs --strict` passes for `release-pipeline`, then commit
      `chore(openspec): archive prepare-release-cascade`
