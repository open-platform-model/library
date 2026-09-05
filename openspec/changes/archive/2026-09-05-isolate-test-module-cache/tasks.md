# Tasks: isolate-test-module-cache

## 1. Spike: symlinked opmodel.dev tier

- [x] 1.1 In a scratch copy of `TestRender_HappyPath` (or the smallest render test), point `CUE_CACHE_DIR` at a `t.TempDir()` cache whose `mod/extract/opmodel.dev` and `mod/download/opmodel.dev` are symlinks into `.cue-cache`; verify the test passes warm (core already in `.cue-cache`) and that `.cue-cache` gains no new entries.
- [x] 1.2 Repeat with `.cue-cache/mod/extract/opmodel.dev` and `.cue-cache/mod/download/opmodel.dev` moved aside (cold): verify the test fetches core once into `.cue-cache` through the links, a second run is warm, and no `.partial` or `.lock` file is left behind. Restore the moved directories. If either run fails on the symlink, record the copy fallback in `design.md` before continuing.

## 2. opm/internal/schematest

- [x] 2.1 Add `PrivateCacheDir(t testing.TB) string` per the design (temp root, `mod/{extract,download}`, shared `opmodel.dev` directories created if absent and symlinked, `t.Cleanup` running `modcache.RemoveAll` on the root); verify with a unit test in `schematest` that the two links resolve to `WorkspaceCacheDir` subpaths and that the test's own cleanup leaves no directory behind after a read-only subdirectory is created under the root.
- [x] 2.2 Package doc: describe the two tiers (shared workspace cache for `opmodel.dev`, private per-test cache for served fixtures) and which helper each test kind uses; verify `go vet ./opm/internal/...` and the doc reads correctly in `go doc ./opm/internal/schematest`.

## 3. opm/internal/registrytest

- [x] 3.1 `buildRegistry` and `NewRegistryFromDir` set `CUE_CACHE_DIR` to `schematest.PrivateCacheDir(t)` after `schematest.SetEnv(t)`; verify `go test ./opm/helper/loader/registry/ ./opm/helper/synth/` pass and that a served fixture's extract directory appears under the temp root, not under `.cue-cache`.
- [x] 3.2 Delete `evictFromCache`, `removeReadOnlyTree`, and the coordinate collection in `NewRegistryFromDir` (drop the `modfile` import if unused); update the `NewRegistryFromDir` and package docs (no eviction, fresh private cache); verify `grep -rn 'RemoveAll\|evict' opm/internal/` reports nothing outside CUE-owned calls and `go test ./opm/kernel/ ./opm/helper/platformmodule/` pass.
- [x] 3.3 Committed-fixture freshness: edit one byte of a `testdata/render/registry` fixture under its existing version, verify `TestGenerate_BuildsThroughTheKernel` sees the edit (assertion on the changed field fails or passes accordingly), then revert the edit.

## 4. Docs

- [x] 4.1 `CLAUDE.md`: Repository Layout entry for `opm/internal/registrytest` and `opm/internal/schematest`, and Environment Notes (`.cue-cache` is the shared `opmodel.dev` tier; served fixtures live in per-test caches; nothing evicts); verify `grep -n 'evict' CLAUDE.md README.md docs/` is empty.

## 5. Verification

- [x] 5.1 Stress the former race: `go test ./opm/kernel/ ./opm/helper/platformmodule/ -count=5` (both packages as parallel processes, five rounds) passes with no `no such file or directory` failure.
- [x] 5.2 Three consecutive `go test ./... -count=1` runs pass; `.cue-cache` contains no `test.example` or `testing.opmodel.dev` entries created by them (pre-existing ones may be deleted by hand once).
- [x] 5.3 `task fmt`, `task vet`, `task lint`, `task test` green; `go test -race ./opm/kernel/... ./opm/internal/renderstage/...` green.
