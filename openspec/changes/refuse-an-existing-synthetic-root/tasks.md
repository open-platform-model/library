## 1. sourcetree, renderstage: one existing-root check

Library tests run with an absolute private `TMPDIR`
(`export TMPDIR=$(mktemp -d -p <session scratchpad>)`), and the worktree's `.cue-cache` is a copy
of the main checkout's, never a symlink. Every commit task stages the files it names with
`git add <file>`.

- [ ] 1.1 `opm/internal/sourcetree/sourcetree.go`: add `CheckRootAbsent(role, root string) error`
  (design D1): nil when `os.Lstat(root)` reports `fs.ErrNotExist`; when it finds anything,
  `<role> <root> exists on disk; it is served from memory and the build would read what is there, so remove it`;
  any other `Lstat` error wrapped as `checking that the <role> <root> is absent: <cause>`. Its doc
  says why: cue/load merges a real directory beneath an overlay root into the load.
- [ ] 1.2 Same file: fix the `VolumeRoot` doc. The result is `/<name>` on Unix and `C:\<name>` on
  Windows, where `C:` stands for the current volume; drop the `\\<name>` spelling, which reads as a
  UNC path.
- [ ] 1.3 `opm/internal/renderstage/stage.go`: delete `checkRootAbsent` and call
  `sourcetree.CheckRootAbsent("render root", root)` from `Stage`. Drop imports that become unused.
  Keep the `RenderRoot` doc's sentence about the refusal.
- [ ] 1.4 `opm/internal/sourcetree/sourcetree_test.go`: add `TestCheckRootAbsent` with four cases
  under `t.TempDir()`: an absent path passes; a directory holding a `.cue` file, a plain file and a
  dangling symlink each refuse with the role and the path in the error. Skip the symlink case when
  `os.Symlink` fails (Windows without the privilege).
- [ ] 1.5 `go test -count=1 ./opm/internal/sourcetree/ ./opm/internal/renderstage/` green
  (`TestStage_RefusesAnExistingRenderRoot` unchanged), then `task check` green, then commit
  `refactor(renderstage): move the existing-root check into sourcetree`.

## 2. sourcetree, loader: refuse an existing synthetic root

- [ ] 2.1 `opm/internal/sourcetree/sourcetree.go`: rename `syntheticBase` to `SyntheticBase`
  (design D3). Its doc says it is the directory every `SyntheticRoot` lies under, that it is a
  variable only so a test can point it at an existing directory, and that nothing else assigns it.
  Extend the `SyntheticRoot` doc: nothing exists at the root on disk, and `FetchArtifact` refuses
  when something does.
- [ ] 2.2 `opm/internal/loader/registry.go`: in `FetchArtifact`, right after `synthRoot` is
  computed and before `OverlayFromFS` (design D2), call
  `sourcetree.CheckRootAbsent("synthetic root", synthRoot)` and return
  `staging <label> <module@version>: <cause>` on error. Update the staging comment and the
  `FetchArtifact` doc to say the acquire refuses when anything exists at the synthetic root, and
  why.
- [ ] 2.3 `opm/internal/loader/registry_test.go`: add `TestFetchModule_RefusesAnExistingSyntheticRoot`,
  not parallel. It publishes a module fixture as the bare-version test does, points
  `sourcetree.SyntheticBase` at `t.TempDir()` (restored in `t.Cleanup`), and creates
  `sourcetree.SyntheticRoot(modPath+"@v0", "v0.0.2")` as a directory holding `injected.cue`
  (`package hello` plus a field the fixture does not declare). Assert that `FetchModule` returns an
  error containing `synthetic root <path> exists on disk`, a nil source and a zero value. Then
  replace the directory with a plain file at the same path and assert the same refusal. Then remove
  it and assert that the same fetch succeeds, so the refusal came from the root and nothing else.
- [ ] 2.4 `go test -race -count=1 ./opm/internal/loader/ ./opm/internal/sourcetree/ ./opm/internal/renderstage/ ./opm/kernel/`
  green, then `task check` green, `task api:diff` reports no incompatible change, and the consumer
  build (`GOTOOLCHAIN=local bash .tasks/consumer-build.sh <consumer-checkout> . <work-dir>`) passes
  against cli and opm-operator `main`. Commit
  `fix(loader): refuse an acquire whose synthetic root exists on disk`.

## 3. openspec: wording of the render-root refusal

- [ ] 3.1 Confirm the change's `single-build-render` delta splits the sentence as written (the
  refusal is its own sentence, the colon introduces only the served files) and that the
  `registry-module-loading` delta names the refusal and its scenario. Run
  `openspec validate refuse-an-existing-synthetic-root --strict` green.
- [ ] 3.2 Tick every task in this file and commit
  `chore(openspec): mark refuse-an-existing-synthetic-root implemented`.
