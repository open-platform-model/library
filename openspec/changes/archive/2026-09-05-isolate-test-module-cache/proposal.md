## Why

`go test ./...` fails intermittently (one of three full runs on 2026-09-05) in `TestGenerate_BuildsThroughTheKernel` with `import failed: open cue.mod/module.cue: no such file or directory`. The cause is the test harness, not the test: `registrytest.NewRegistryFromDir` deletes each committed fixture coordinate from the shared workspace CUE module cache (`library/.cue-cache`) before serving it, while another test package's process is reading that very extracted directory. CUE's module cache is lock-safe for concurrent extraction, but its readers hold no lock; it assumes an extracted directory is immutable and never removed while any process can read it. Under `go test ./...`, locally and in CI, every package is its own process, and `opm/kernel` evicts on each of its roughly twenty-four render tests, so the window is open for most of the suite. The eviction exists for a real reason (the cache is keyed by coordinate, so an edited committed fixture under a fixed version would be shadowed by last run's bytes); the fix keeps that guarantee while restoring CUE's invariant.

## What Changes

- `opm/internal/registrytest` gives every in-process registry a private CUE module cache under the test's temp directory: `CUE_CACHE_DIR` points at it for the test scope, the served fixture prefixes (`test.example`, `testing.opmodel.dev/...`) extract into it fresh, and the `opmodel.dev` subtrees of the cache (`mod/extract/opmodel.dev`, `mod/download/opmodel.dev`) are symlinks into the shared workspace cache, so `opmodel.dev/core@v2` and the GHCR catalog stay warm across tests and processes through CUE's own lock-safe fetch protocol.
- The eviction code (`evictFromCache`, `removeReadOnlyTree`) is deleted. Nothing in the test tree deletes from a cache another process may read. A committed fixture edited under a fixed coordinate is still always served fresh, because its coordinate never exists in a private cache before the test that serves it runs.
- The private cache applies to all three constructors (`NewCatalogRegistry`, `NewModuleRegistry`, `NewRegistryFromDir`), not only the evicting one. Generated fixtures then stop sharing cache entries across packages as well: `UniquePath` keys coordinates by `t.Name()`, which two packages can collide on with different bytes and no eviction. `UniquePath` stays as a readability aid.
- `schematest.SetEnv` / `NewCache` and the tests that use them directly (`opm/schema`, `opm/helper/loader/file`, the `opm/helper/synth` unit tests, the kernel flow and live tests) keep the shared cache. They resolve only `opmodel.dev` coordinates, which nothing deletes any more.
- Docs: `CLAUDE.md` (Repository Layout entry for `registrytest`, Environment Notes) and the `registrytest` / `schematest` package docs describe the two-tier cache.

Non-goals: no change to `opm/` library code, to the fixtures under `testdata/`, or to how CI invokes the suite. `skip_specs` was considered and rejected: the harness has a contract that four test packages rely on (`cue-regression-canary` and `render-parity` already spec test harnesses in this repo), and the contract is what changes.

## Capabilities

### New Capabilities

- `test-fixture-registry`: the contract of the in-process fixture registry harness shared by the kernel, loader, synth and platform-module tests: per-test private module cache, a shared `opmodel.dev` tier, no deletion from shared state, committed fixtures always served fresh, resolution of the shared tier through CUE's own cache protocol.

### Modified Capabilities

None.

## Impact

- Code: `opm/internal/registrytest/registrytest.go` (cache setup in the shared constructor path, eviction removal), `opm/internal/schematest/schematest.go` (the shared-cache path helper stays; a private-cache helper joins it or lives in `registrytest`), `CLAUDE.md`. Test-only: `opm/` exports do not change, so there is no SemVer effect and the change lands as `test:` (no release).
- Downstream consumers (`cli`, `opm-operator`): none. They do not import `opm/internal/...`.
- Risk: the symlinked `opmodel.dev` tier relies on CUE's `modcache` tolerating a symlinked directory for `os.Stat`, extraction, lock files and temp-file renames. A spike task verifies it before the harness changes; the fallback is copying the two `opmodel.dev` subtrees (about 1.7 MB today) per test instead of linking, at the cost that a cold shared cache is then never warmed by these tests.
- Principle VII: no new abstraction; one helper that creates a directory and two symlinks replaces two helpers that delete. Principle VIII: one package, one concern, verifiable by running the suite repeatedly.
