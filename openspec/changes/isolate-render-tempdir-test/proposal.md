## Why

Three kernel render tests assert that `Kernel.Render` leaves no staging directory behind. They
compare two listings of `opm-render-*` taken through the helper `stagingDirs`
(`opm/kernel/render_test.go:534-543`). That helper globs `os.TempDir()`, which is the shared
`TMPDIR` of every test process on the machine. `Render` creates its staging directory with
`os.MkdirTemp("", "opm-render-")` (`opm/kernel/render.go:355`), so any other process rendering at
the same time (a second `go test` of the same package, for example from a parallel worktree) can
add or remove a directory between the two listings and fail the assertion.

`TestRender_RepeatedRendersShareNothing` (`render_test.go:545`) was already patched in #168 with
`t.Setenv("TMPDIR", t.TempDir())` (`render_test.go:551-553`), so its own listing is private. The
helper still reads the ambient temp dir, and its two other callers never got the patch:

- `TestRender_LocalReplacementRefusedUnlessEnabled` (`render_test.go:967`, `:976`)
- `TestRender_VersionlessDependencyWithoutReplacementRefused` (`render_test.go:1080`, `:1089`)

A run on 2026-10-04, from the base of this change, reproduced the flake. Three processes of one
compiled `opm/kernel` test binary shared one `TMPDIR`, and each ran the three tests 40 times.
Two of the three processes failed: four failures in all, every one in the two unpatched tests,
for example `render_test.go:1089: expected: []string(nil) actual: []string{".../opm-render-..."}`.
The test that already sets a private `TMPDIR` passed all 120 runs.

The flake is local. `render.go:355` is the only place that creates `opm-render-*`, and only
`opm/kernel` tests call `Render`. The library's `task test` runs `opm/kernel` once, in its `-race`
pass (`Taskfile.yml:133`), and no `opm/kernel` test calls `t.Parallel`, so a single CI run cannot
collide with itself. The collisions come from several suites sharing one machine's `TMPDIR`: the
swarm's parallel library worktrees, or a developer running the suite twice at once. A future CI
layout that ran two `opm/kernel` processes on one runner would hit it too, but none does today.

## What Changes

- `stagingDirs` takes the directory to list instead of reading `os.TempDir()`. A test can then
  only list a temp root it chose, and no future caller can glob the shared temp dir.
- A test helper gives a test a private staging root: it sets `TMPDIR` to `t.TempDir()` for the
  test and returns that directory, the pattern `render_core_floor_test.go:32-33,50` already uses.
- The three callers use the helper. Because the root is fresh, each asserts that the root holds
  no `opm-render-*` directory after its renders or refusals, which is the same claim as before
  (nothing is left behind), without the `before` snapshot.
- Production code (`opm/kernel/render.go`) does not change, and the change adds no exported symbol.

Out of scope:

- An option on `Kernel` naming the staging parent directory. It would add public surface for a
  test-only need (Principle VII). The design records it as the fallback if the test-side fix
  does not hold.
- `TestGenerate_BuildsThroughTheKernel` and the cross-process `.cue-cache` race recorded for
  `opm/helper/platformmodule`. That is a different cause (the module cache, not the temp dir)
  and needs its own change if it still reproduces after `2026-09-05-isolate-test-module-cache`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `single-build-render`: adds a requirement that a test asserting on render staging
  directories lists only a temp root private to that test, so concurrent test processes
  sharing `TMPDIR` cannot change its result. `single-build-render` already owns the claim
  those tests check ("The staging directory SHALL be removed when the render completes").

Follow-up, not in this change: two `single-build-render` scenarios say a refusal happens "before
staging" and that "no staging directory is written" (main spec, the version-less dependency
scenario and "Replacement without opt-in refused"). The code creates the staging directory
(`render.go:355`) before `renderstage.Stage` refuses, then removes it. The wording needs its own
change.

## Impact

- Packages: `opm/kernel` tests only (`render_test.go`). No `opm/` public API change, no
  behaviour change.
- Downstream consumers (cli, opm-operator): none.
- SemVer: none. The commit is `test(kernel)`, a type release-please hides, so it cuts no release.
- Depends on: nothing. This is not a `deps:cascade` task change, so it does not wait on the
  `.github` `add-cascade-resolver` change.
- Gates: `task check` (fmt, vet, lint, test, docs:bundle:check), plus the concurrent repro in
  tasks.md section 1, run before the fix and after it with a noise process creating and removing
  `opm-render-*` directories in the shared `TMPDIR`.
