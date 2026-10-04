## Context

`Kernel.Render` stages each render in a fresh directory made by
`os.MkdirTemp("", "opm-render-")` and removes it on return (`opm/kernel/render.go:355-359`). The
`single-build-render` spec requires that removal ("The staging directory SHALL be removed when the
render completes"), and three tests in `opm/kernel/render_test.go` check it through one helper:

```go
// render_test.go:537-543 today
func stagingDirs(t *testing.T) []string {
	t.Helper()
	dirs, err := filepath.Glob(filepath.Join(os.TempDir(), "opm-render-*"))
	require.NoError(t, err)
	sort.Strings(dirs)
	return dirs
}
```

| Test | Lines | Private `TMPDIR` |
| --- | --- | --- |
| `TestRender_RepeatedRendersShareNothing` | 545-560 | yes, since #168 (`:551-553`) |
| `TestRender_LocalReplacementRefusedUnlessEnabled` | 961-977 | no |
| `TestRender_VersionlessDependencyWithoutReplacementRefused` | 1069-1090 | no |

`go test ./...` runs each package as its own process, and every process inherits one `TMPDIR`. A
listing of that shared directory sees the staging directories of every concurrent renderer.

`render_core_floor_test.go:32-33,50` already uses the safe pattern: set `TMPDIR` to `t.TempDir()`,
keep the path, and glob that path rather than `os.TempDir()`.
Once the helper exists, that test calls it too, so the pattern lives in one place.

## Goals / Non-Goals

**Goals:**

- No `opm/kernel` test result depends on what other processes put in the shared temp dir.
- Every test keeps its intent: a render or a refusal leaves no staging directory behind.
- The helper's signature makes the unsafe form impossible to write again.

**Non-Goals:**

- No change to `render.go` or to any exported symbol.
- No fix for the `opm/helper/platformmodule` module-cache race (see proposal, Out of scope).
- No rewording of the `single-build-render` scenarios that say a refusal writes no staging
  directory; the code creates one and removes it (proposal, Modified Capabilities follow-up).

The requirement lands in `single-build-render`, which already owns the staging-removal claim the
three tests check, not in `test-fixture-registry`, whose purpose is the in-process fixture
registry harness. This is a deliberate exception to the spec rule in `openspec/config.yaml`
("Focus: WHAT behavior", "Describe observable behavior"): the requirement states how the tests
that guard the staging-removal claim must observe it, because that claim is only checkable by
such a test. Its scenarios state outcomes (no foreign failure, a leak still fails); none states
the shape of the helper.

## Decisions

### D1. List an explicit, test-private root

`stagingDirs` MUST take the directory to list. A new helper `privateStagingRoot(t)` MUST set
`TMPDIR` to a fresh `t.TempDir()` with `t.Setenv` and return it. Each of the three tests MUST call
`privateStagingRoot` before its first `Render` and then assert on `stagingDirs(t, root)`.

```go
// privateStagingRoot points the process temp dir at a directory owned by this
// test, so Render stages there and no other process can add to or remove from it.
func privateStagingRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("TMPDIR", root)
	return root
}

func stagingDirs(t *testing.T, root string) []string {
	t.Helper()
	dirs, err := filepath.Glob(filepath.Join(root, "opm-render-*"))
	require.NoError(t, err)
	sort.Strings(dirs)
	return dirs
}
```

The root is fresh, so the `before` snapshot is always empty and is dropped. Each test asserts
`assert.Empty(t, stagingDirs(t, root), ...)`, keeping its existing message.

### D2. Order of `privateStagingRoot` in each test

In `TestRender_RepeatedRendersShareNothing`, `privateStagingRoot` replaces the existing
`t.Setenv` line at the same place, after the kernel, platform and instance are built. Acquisition
and synthesis do not stage, so their position does not matter. Moving them would make the diff
larger for no gain.

In the two refusal tests, the call goes where `before := stagingDirs(t)` is today, right before
`Render`.

### D3. `t.Setenv` and parallel tests

`t.Setenv` changes the environment of the whole process, and it panics in a test marked
`t.Parallel()`. No test in `opm/kernel` calls `t.Parallel()`, which `grep -n "t.Parallel"
opm/kernel/*_test.go` confirms today, so no other test in the process renders while the
variable is redirected.

If a parallel render test is ever added, the panic makes the conflict loud rather than flaky.
That is the right failure mode, and it is the point at which D4's fallback becomes worth its cost.

## Research & Decisions

### Root cause and reproduction

**Context**: The research sweep named `TestRender_RepeatedRendersShareNothing` as the flaky test,
but #168 had already given that test a private `TMPDIR`. The cause had to be confirmed before
choosing a fix.

**Explored**: At the base of this change (library `553b855`), I compiled the package once with
`go test -c ./opm/kernel` and ran three processes of the binary at the same time. All three shared
`TMPDIR` and each ran
`-test.run '^TestRender_(RepeatedRendersShareNothing|LocalReplacementRefusedUnlessEnabled|VersionlessDependencyWithoutReplacementRefused)$' -test.count=40`.

Results:

- Process 1: 0 failures.
- Process 2: 3 failures (1 `LocalReplacementRefusedUnlessEnabled`, 2
  `VersionlessDependencyWithoutReplacementRefused`).
- Process 3: 1 failure (`VersionlessDependencyWithoutReplacementRefused`).
- Every failure was a foreign `opm-render-*` entry in the listing that should have been empty
  (`render_test.go:1089`).
- `RepeatedRendersShareNothing` passed in all 120 runs.

**Decision**: The fix covers the helper and all three callers, not only the named test.

**Rationale**: The flake is in the shared helper. Patching one call site, as #168 did, left the
other two exposed.

### D4. Alternatives

**Context**: The brief offered two other shapes: point the kernel's temp root at a per-test
directory through the API, or assert only on the directories of the test's own render.

**Explored**:

- `kernel.New` options (`opm/kernel`): none names a staging parent. `Render` calls `os.MkdirTemp`
  with `""`.
- `RenderInput` and the render result: neither exposes the staging path, so a test cannot learn
  which directory was its own.

**Decision**: Fix it in the test helper (D1). Add no `Kernel` option and no exposed staging path.

**Rationale**:

- A `WithStagingDir` option or a diagnostics field adds public surface for a test-only need
  (Principle VII), and every public symbol is a SemVer commitment (Principle VI).
- The test-side fix is enough, because the reproduction shows the test that already has a
  private `TMPDIR` never failed.
- Reading `TMPDIR` through `os.MkdirTemp("")` is arguably a hidden environment lookup under
  Principle I, but changing that is a kernel API decision for the owner, not part of a test fix.
  The option stays the fallback if the concurrent check in tasks.md 1.4 still fails after D1.

## Risks / Trade-offs

- [`t.Setenv` is process-global] → No test in `opm/kernel` runs in parallel (D3), and a future
  parallel render test fails loudly rather than flaking.
- [Dropping the `before` snapshot loses a check] → Nothing is lost. The snapshot only guarded
  against pre-existing foreign directories, which a fresh private root cannot contain.
- [The reproduction is probabilistic] → The before run must show at least one failure.
- [After the fix, the three tests no longer write `opm-render-*` into the shared `TMPDIR`, so a
  re-run of the same three processes has nothing to collide on and cannot fail] → The after run
  adds a noise process that keeps creating and removing `opm-render-*` directories in the shared
  root while the new binary runs with the same or a higher count. tasks.md 1.1 and 1.4 record
  both runs in `design.md` under "Verification", including that the noise was running.

## Verification

Run on 2026-10-04 with the tasks.md section 1 script: three processes of one compiled
`opm/kernel` test binary, one shared absolute `TMPDIR`, each running the three tests `-test.count`
times. "Noise" is a fourth process looping
`d=$(mktemp -d "$T/opm-render-XXXXXX"); rmdir "$d"` in that `TMPDIR` for the whole run.

| Binary | Count | Noise | p1 | p2 | p3 |
| --- | --- | --- | --- | --- | --- |
| `B0` (base `553b855`, built at docs-only `42f094b`, unchanged tests) | 40 | no | 1 | 2 | 3 |
| `B0` | 40 | yes | 56 | 58 | 58 |
| `B1` (this change) | 80 | yes | 0 | 0 | 0 |
| `B0` (control, same session as `B1`) | 80 | yes | 103 | 113 | 112 |

- Before (1.1): 6 failures without noise, all in `LocalReplacementRefusedUnlessEnabled`
  (`render_test.go:976`) and `VersionlessDependencyWithoutReplacementRefused`
  (`render_test.go:1089`), each a foreign `opm-render-*` entry or a listing that lost one.
  `RepeatedRendersShareNothing` passed every run, as in planning.
- After (1.4): 0 failures in 240 runs per test while the noise process ran. The control run of
  `B0` under the same noise right after failed 328 times, so the noise does reach a test that
  lists the shared root.
- Leak detection kept: with the `os.RemoveAll` defer at `render.go:359` disabled for one local
  run, all three tests failed; the line was restored before committing.
