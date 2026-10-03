## Why

Three kernel render tests assert that `Kernel.Render` leaves no staging directory behind. They
compare two listings of `opm-render-*` taken through the helper `stagingDirs`
(`opm/kernel/render_test.go:534-543`). That helper globs `os.TempDir()`, which is the shared
`TMPDIR` of every test process on the machine. `Render` creates its staging directory with
`os.MkdirTemp("", "opm-render-")` (`opm/kernel/render.go:355`), so any other process rendering at
the same time (a second `go test`, another package's binary, a cascade run) can add or remove a
directory between the two listings and fail the assertion.

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

The library's `Go tests` job will be a required check (RELEASING.md "Owner settings", "Rulesets on
main", line 478). A cascade PR whose CI is red needs a human to add `deps-cascade:hold` (RELEASING.md
"Runbook", "Handling a cascade PR", line 406), so a cross-process flake in that job costs a human
step on a bot PR that should need none.

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

- `test-fixture-registry`: adds a requirement that a test asserting on render staging
  directories lists only a temp root private to that test, so concurrent test processes
  sharing `TMPDIR` cannot change its result.

## Impact

- Packages: `opm/kernel` tests only (`render_test.go`). No `opm/` public API change, no
  behaviour change.
- Downstream consumers (cli, opm-operator): none.
- SemVer: none. The commit is `test(kernel)`, a type release-please hides, so it cuts no release.
- Depends on: nothing. This is not a `deps:cascade` task change, so it does not wait on the
  `.github` `add-cascade-resolver` change.
- Gates: `task check` (fmt, vet, lint, test, docs:bundle:check), plus the concurrent repro in
  tasks.md section 1, run before and after the fix.
