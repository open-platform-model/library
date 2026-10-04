## 1. kernel: isolate the render staging-directory tests

The concurrent check used in 1.1 and 1.4, run from the repo root (`T` is the absolute path of any
fresh directory every process shares as `TMPDIR`; `B` is an absolute path for the compiled binary;
both must be absolute because the script changes directory):

```sh
export TMPDIR="$T"
go test -c -o "$B" ./opm/kernel
cd opm/kernel
R='^TestRender_(RepeatedRendersShareNothing|LocalReplacementRefusedUnlessEnabled|VersionlessDependencyWithoutReplacementRefused)$'
for i in 1 2 3; do "$B" -test.run "$R" -test.count=40 > "$T/p$i.log" 2>&1 & done; wait
grep -c '^--- FAIL' "$T"/p*.log
```

- [x] 1.1 Run the concurrent check on the unchanged tree, and keep its binary as `B0` for the control in 1.4. Record the failure count per process and the failing test names in design.md under "Verification". At least one failure is expected (the planning run found 4 in 120 runs per test). If none appears, raise `-test.count` until one does before going on.
- [x] 1.2 In `opm/kernel/render_test.go`, add `privateStagingRoot(t)`. Change `stagingDirs` to take the root it lists (design D1), and update its doc comment to say what it lists.
- [x] 1.3 Move the three callers to the helper. In `TestRender_RepeatedRendersShareNothing`, the helper replaces the `t.Setenv("TMPDIR", t.TempDir())` line and its comment (`:551-553`). In `TestRender_LocalReplacementRefusedUnlessEnabled` (`:967`, `:976`) and `TestRender_VersionlessDependencyWithoutReplacementRefused` (`:1080`, `:1089`), it replaces `before := stagingDirs(t)` (design D2). Each test then asserts `assert.Empty` on `stagingDirs(t, root)`, keeping its message. Check that `grep -n 'os.TempDir()' opm/kernel/*_test.go` finds nothing.
- [x] 1.4 Run the concurrent check again with the same or a higher `-test.count`, and keep a noise process running in the shared root for the whole run: `while :; do d=$(mktemp -d "$T/opm-render-XXXXXX"); rmdir "$d"; done &` (kill it after `wait`). Without the noise the re-run cannot fail, because after the fix nothing writes `opm-render-*` into `$T`. It must show 0 failures in every process. As a control, the same noise against `B0` must still fail. Record the result in design.md under "Verification", stating that the noise was running. If any test fails on a foreign `opm-render-*` directory, stop and report it to the supervisor: design D4's fallback (a `Kernel` staging-dir option) is an API decision for the owner.
- [x] 1.5 `task check` green, then commit `test(kernel): list render staging dirs under a test-private temp root`.
- [x] 1.6 `TestRender_OlderCorePlatformRefusedBeforeStaging` calls `privateStagingRoot` and `stagingDirs` in place of its inline `TMPDIR` and glob (77e5658).

## 2. Verify and archive

- [x] 2.1 Run `openspec verify` for `isolate-render-tempdir-test` (the repo's openspec-verify-change skill). Verify: no CRITICAL finding.
- [x] 2.2 Run `openspec archive isolate-render-tempdir-test --yes`. Verify: the main `single-build-render` spec carries the new requirement, the change sits under `openspec/changes/archive/`, and `openspec validate --specs --strict` passes.
- [x] 2.3 Gates green, then commit `chore(openspec): archive isolate-render-tempdir-test` (the archive rides the implementing PR).
