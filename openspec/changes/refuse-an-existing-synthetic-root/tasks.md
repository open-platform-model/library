## 1. sourcetree, renderstage: one existing-root check

Library tests run with an absolute private `TMPDIR`
(`export TMPDIR=$(mktemp -d -p <session scratchpad>)`), and the worktree's `.cue-cache` is a copy
of the main checkout's, never a symlink. Every commit task stages the files it names with
`git add <file>`.

- [ ] 1.1 `opm/internal/sourcetree/sourcetree.go`: add `CheckRootAbsent(role, served, root string) error`
  (design D1): nil when `os.Lstat(root)` reports `fs.ErrNotExist`; when it finds anything,
  `<role> <root> exists on disk; the <served> is served from memory under it and the build would read what is there, so remove it`;
  any other `Lstat` error wrapped as `checking that the <role> <root> is absent: <cause>`. Its doc
  says why: cue/load merges a real directory beneath an overlay root into the load.
- [ ] 1.2 Same file: fix the `VolumeRoot` doc. The result is `/<name>` on Unix and `C:\<name>` on
  Windows, where `C:` stands for the current volume; drop the `\\<name>` spelling, which reads as a
  UNC path.
- [ ] 1.3 `opm/internal/renderstage/stage.go`: delete `checkRootAbsent` and call
  `sourcetree.CheckRootAbsent("render root", "render module", root)` from `Stage`, so the message
  stays byte-identical. Drop imports that become unused. Keep the `RenderRoot` doc's sentence about
  the refusal.
- [ ] 1.4 `opm/internal/sourcetree/sourcetree_test.go`: add `TestCheckRootAbsent` with four cases
  under `t.TempDir()`: an absent path passes; a directory holding a `.cue` file, a plain file and a
  dangling symlink each refuse with the role and the path in the error. Skip the symlink case when
  `os.Symlink` fails (Windows without the privilege).
- [ ] 1.5 `go test -count=1 ./opm/internal/sourcetree/ ./opm/internal/renderstage/` green
  (`TestStage_RefusesAnExistingRenderRoot` unchanged), then `task check` green, then commit
  `refactor(renderstage): move the existing-root check into sourcetree`.

## 2. sourcetree, loader: refuse an existing synthetic root at acquire

- [ ] 2.1 `opm/internal/sourcetree/sourcetree.go`: rename `syntheticBase` to `SyntheticBase`
  (design D4). Its doc says it is the directory every `SyntheticRoot` lies under, that it is a
  variable only so a test can point it at an existing directory, and that nothing else assigns it.
  Add `IsSynthetic(root string) bool`, true when root lies directly under `SyntheticBase` (design
  D3). Extend the `SyntheticRoot` doc: nothing exists at the root on disk, and the acquire and
  synthesis refuse when something does.
- [ ] 2.2 `opm/internal/loader/registry.go`: in `FetchArtifact`, right after `synthRoot` is
  computed and before `OverlayFromFS` (design D2), call
  `sourcetree.CheckRootAbsent("synthetic root", spec.Label, synthRoot)` and return
  `staging <label> <module@version> in overlay: <cause>` on error. Update the staging comment and
  the `FetchArtifact` doc to say the acquire refuses when anything exists at the synthetic root,
  and why.
- [ ] 2.3 `opm/internal/loader/registry_test.go`: add `TestFetchModule_RefusesAnExistingSyntheticRoot`,
  not parallel. It publishes a module fixture as the bare-version test does, points
  `sourcetree.SyntheticBase` at `t.TempDir()` (restored in `t.Cleanup`), and creates
  `sourcetree.SyntheticRoot(modPath+"@v0", "v0.0.2")` as a directory holding `injected.cue`
  (`package hello` plus a field the fixture does not declare). Assert that `FetchModule` returns an
  error containing `synthetic root <path> exists on disk`, a nil source and a zero value. Then
  replace the directory with a plain file at the same path and assert the same refusal. Then remove
  it and assert that the same fetch succeeds, so the refusal came from the root and nothing else.
  Add `TestFetchArtifact_CatalogRefusesAnExistingSyntheticRoot` next to the catalog bare-version
  test: the same refusal and recovery for `FetchArtifact` with `loader.CatalogSpec`.
- [ ] 2.4 `opm/internal/sourcetree/sourcetree_test.go`: `TestIsSynthetic` covers a
  `SyntheticRoot`, the base itself, a path beneath a synthetic root and an unrelated absolute path.
- [ ] 2.5 `go test -race -count=1 ./opm/internal/loader/ ./opm/internal/sourcetree/ ./opm/internal/renderstage/ ./opm/kernel/`
  green, then `task check` green. Commit
  `fix(loader): refuse an acquire whose synthetic root exists on disk`.

## 3. synth: refuse an existing synthetic root at synthesis

- [ ] 3.1 `opm/internal/synth/instance.go`: in `Instance`, after `buildOverlay` and before
  `LoadDir` (design D3), when `sourcetree.IsSynthetic(moduleRoot)` call
  `sourcetree.CheckRootAbsent("synthetic root", "instance package", moduleRoot)` and return
  `instance synthesis: <cause>` with a nil tree on error. Document it in the `Instance` doc.
- [ ] 3.2 `opm/internal/synth/instance_test.go`: add `TestInstance_RefusesAnExistingSyntheticRoot`,
  not parallel. It points `sourcetree.SyntheticBase` at `t.TempDir()`, publishes a core-v2 module
  fixture to an in-memory registry, fetches it with `loader.FetchModule` (the root is absent, so the
  acquire succeeds), then creates the synthetic root holding an injected `.cue` file. Assert that
  `synth.Instance` returns an error containing `synthetic root <path> exists on disk` and a nil
  tree. Remove the root and assert that the same call succeeds. A module acquired from a directory
  keeps synthesizing: the existing directory-acquired synthesis tests stay green.
- [ ] 3.3 `go test -race -count=1 ./opm/internal/synth/ ./opm/kernel/` green, then `task check`
  green, `task api:diff` reports no incompatible change, and the consumer build
  (`GOTOOLCHAIN=local bash .tasks/consumer-build.sh <consumer-checkout> . <work-dir>`) passes
  against cli and opm-operator `main`. Commit
  `fix(synth): refuse a synthesis whose synthetic root exists on disk`.

## 4. openspec: deltas and wording

- [ ] 4.1 Confirm the change's `single-build-render` delta splits the sentence as written (the
  refusal is its own sentence, the colon introduces only the served files), that the
  `registry-module-loading` delta names the refusal with a module and a catalog scenario, and that
  the `instance-synthesis` delta names the synthesis refusal and keeps every main-spec scenario of
  the requirement. Run `openspec validate refuse-an-existing-synthetic-root --strict` green.
- [ ] 4.2 Tick every task in this file and commit
  `chore(openspec): mark refuse-an-existing-synthetic-root implemented`.
