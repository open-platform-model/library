# Design: isolate-test-module-cache

## Context

See `proposal.md` § Why for the failure and its cause. The facts the approach rests on, read from `cuelang.org/go@v0.17.1/mod/modcache`:

- The module cache lives under `$CUE_CACHE_DIR/mod` in two subtrees: `extract/<path>@<ver>/` (the extracted tree, directories made read-only with `makeDirsReadOnly`) and `download/<path>/@v/<ver>.{zip,mod,partial,lock}` (download artifacts, the `.partial` marker written before extraction and removed after, and the `.lock` file that guards download plus extraction).
- `Fetch` returns the extract directory as soon as it exists with no `.partial` marker, without taking the lock. Readers never lock. Concurrent extraction of the same version is serialised by `.lock`, and a waiter re-checks the directory after acquiring it.
- `modcache.RemoveAll` is exported and removes a tree whose directories are read-only (it restores write permission first). `t.TempDir()`'s own cleanup is a plain `os.RemoveAll`, which fails on a 0555 directory and fails the test.

Today `schematest.SetEnv` points `CUE_CACHE_DIR` at the shared `library/.cue-cache` (gitignored, about 1.7 MB, holding `opmodel.dev`, `test.example` and `testing.opmodel.dev` trees); every `registrytest` constructor calls it, then `NewRegistryFromDir` evicts the coordinates it serves. Tests inside a package run sequentially (no `t.Parallel()` in the affected packages); `TestRender_ConcurrentKernelsShareNothing` runs goroutines inside one test. Development and CI run on Linux, where symlinks are available.

```
  today                                   after

  every test process                      every test
  -> shared .cue-cache/mod                -> <t.TempDir>/mod
       extract/opmodel.dev   (shared)          extract/opmodel.dev  --> symlink to shared
       extract/test.example  (shared!)         extract/test.example (private, empty)
       extract/testing...    (shared,          extract/testing...   (private, empty)
                              evicted)         download/opmodel.dev --> symlink to shared
       download/...                            download/<prefix>    (private)
```

## Goals / Non-Goals

**Goals:**

- No code in the test tree deletes from a cache another process can read.
- Served coordinates are isolated per test; the `opmodel.dev` namespace stays shared and warm.
- Call sites in the four consuming test packages do not change; the change is inside the two `opm/internal` helper packages.

**Non-Goals:**

- A per-process cache (see Decisions).
- Changing fixtures under `testdata/`, the library's `opm/` code, or how CI invokes the suite.
- Windows support for the harness.

## Decisions

### Private cache per test, not per process

**Context**: the cache must outlive every build the test runs and be removed afterwards.
**Options**: (1) `t.TempDir()` per test; (2) one `os.MkdirTemp` per process behind a `sync.Once`; (3) one per package through a `TestMain` in each consuming package.
**Decision**: option 1. Its lifetime matches the registry's own `t.Cleanup(reg.Close)`, needs no `TestMain` in four packages, and the per-test cost is a directory, two symlinks and the re-extraction of a handful of tiny fixtures, which the eviction path already forced on every kernel render test. Option 2 leaves a temp directory nobody owns at process exit.

### The opmodel.dev tier is a symlink into the shared cache, not a copy

**Context**: core must not be fetched from GHCR per test, and a cold shared cache must still warm.
**Options**: (1) symlink `mod/extract/opmodel.dev` and `mod/download/opmodel.dev` of the private cache to the shared cache's directories; (2) copy the two subtrees per test (`os.CopyFS`, about 1.7 MB); (3) no shared tier.
**Decision**: option 1. Every read and write for the namespace then lands in the shared cache under CUE's own lock and `.partial` protocol, so a cold cache warms once and a newly needed version is fetched once for all packages. Option 2 costs little per test but a cold shared cache is never warmed by these tests and a version not yet in it is refetched by every test that needs it. Option 3 is a network round-trip per test. The link is at the namespace directory (`opmodel.dev`), which contains `core` and `catalogs/...`; the served prefixes (`test.example`, `testing.opmodel.dev`) are sibling top-level directories in both subtrees, so they never touch the link. The shared directories are created (empty) before linking when the shared cache is cold.

### One helper in schematest owns the private cache

**Context**: `schematest` already owns cache-directory knowledge (`WorkspaceCacheDir`, `SetEnv`); `registrytest` owns registries.
**Decision**: `schematest` gains `PrivateCacheDir(t) string`: creates `<t.TempDir()>/mod/{extract,download}`, ensures the shared `mod/extract/opmodel.dev` and `mod/download/opmodel.dev` exist, symlinks them, registers a `t.Cleanup` that runs `modcache.RemoveAll` on the private root (registered after `t.TempDir()`, so it runs before `TempDir`'s cleanup, which then finds nothing to do), and returns the root. `registrytest.buildRegistry` and `NewRegistryFromDir` call `schematest.SetEnv(t)` as today (registry default) and then `t.Setenv("CUE_CACHE_DIR", schematest.PrivateCacheDir(t))`. `SetEnv` and `NewCache` are unchanged for the tests that use the shared cache directly.

### Eviction is deleted, not narrowed

**Context**: `evictFromCache` and `removeReadOnlyTree` exist only to make committed fixtures fresh; `NewRegistryFromDir` parses each fixture's module file solely to compute the coordinates to evict.
**Decision**: delete both helpers and the coordinate collection (the `modfile` import goes with it if nothing else uses it). Freshness now follows from the private cache being empty for served coordinates. Narrowing eviction (once per process, or hash-gated) would only shrink the window.

### The fix is verified by contention, not by a unit test

**Context**: the failure is a cross-process race; a deterministic unit test would have to simulate two module caches racing, which tests the simulation.
**Decision**: acceptance is (a) a spike that proves the symlinked tier works warm and cold, (b) a stress run of the two packages that shared the coordinates, `go test ./opm/kernel/ ./opm/helper/platformmodule/ -count=5`, which runs both packages as parallel processes five times over, and (c) three clean full-suite runs. The scenario "Harness holds no eviction path" is a grep. This design is recorded so a later reader knows why no `TestRace...` exists.

## Risks / Trade-offs

- [CUE's `modcache` refuses or mishandles a symlinked directory in `extract` or `download` (stat, lock-file creation, temp-file rename into the linked directory)] → the spike task runs a render test warm and cold through the linked tier before the harness changes; the fallback is copying the two subtrees per test (`os.CopyFS`), recorded here if taken, with the cold-cache cost noted above.
- [Cold shared cache and several parallel packages needing core at once] → CUE's `.lock` serialises extraction; the waiter re-checks the directory after locking (`fetch.go`, the "populated while we were waiting" branch).
- [Read-only extracted directories under `t.TempDir()`] → the `modcache.RemoveAll` cleanup runs before `TempDir`'s cleanup; `modcache` is marked experimental, but the library already depends on `mod/modregistrytest`, `mod/modconfig` and `mod/modfile` from the same tree.
- [A test that publishes a fixture and reads it through the kernel's `WithRegistry` mapping in a second Kernel] → both read the same process environment, so `CUE_CACHE_DIR` reaches every build in the test; unchanged from today.
- [More temp-directory churn in CI] → each private cache holds a few KB of fixtures; the shared tier is linked, not copied.

## Migration Plan

Test-only; lands as `test:` with no release. Rollback is a revert.

## Open Questions

- `UniquePath` (coordinates keyed by `t.Name()`) becomes cosmetic once caches are private; it can be dropped in the tests-and-docs slice of the simplification plan without changing this design.
