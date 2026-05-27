## ADDED Requirements

### Requirement: Tests acquire the schema via pre-seeded CUE module cache

Tests that need a built schema `cue.Value` SHALL exercise the same `OCILoader` code path used in production, pointed at a pre-seeded CUE module cache directory rooted at `library/testdata/cue-cache/`. Tests SHALL NOT depend on network access, MUST NOT hit `localhost:5000` or any external registry, and MUST NOT use a test-only `Loader` implementation.

The test helper that constructs a `*schema.Cache` for test use SHALL:

- Resolve the absolute path to `library/testdata/cue-cache/` relative to the test file location (e.g., via `runtime.Caller`).
- Set `CUE_CACHE_DIR` to that path (via `OCILoader.CacheDir` or `t.Setenv`).
- Set `CUE_REGISTRY` to a value that will not be consulted (cache hit prevents registry lookup), or set `OCILoader.Registry` to a non-network value.
- Construct a fresh `*schema.Cache` per test (or per test-package) so memoization scope is explicit and one test's state does not leak into another.

#### Scenario: Tests pass without network access

- **WHEN** the test suite is run in an environment with no outbound network access and `testdata/cue-cache/` populated
- **THEN** every test that constructs a schema `Cache` via the test helper succeeds without registry contact

#### Scenario: Tests fail loudly when cache is missing the pinned module

- **WHEN** `testdata/cue-cache/` exists but does not contain the pinned schema module version
- **THEN** the test helper either fails the test with a message naming the pinned module identifier or surfaces CUE's "module not found" error wrapped with that identifier
- **AND** the failure mode is not a silent skip

#### Scenario: Test helper does not pollute process state

- **WHEN** a test constructs a schema `Cache` via the helper and the test completes
- **THEN** any environment variable mutation (e.g., `CUE_CACHE_DIR`) is reverted at test scope (`t.Cleanup` or `t.Setenv` semantics)

### Requirement: Schema test cache is populated by a task

The `library/Taskfile.yml` (or equivalent automation) SHALL provide a task that fetches the pinned schema module version from the configured registry and writes the result into `library/testdata/cue-cache/` in the layout CUE expects under `$CUE_CACHE_DIR`. The task SHALL be idempotent (re-running on a populated directory either no-ops or refreshes safely). The task SHALL be documented as the canonical mechanism to refresh test fixtures when either the pinned schema version or the CUE SDK version bumps.

#### Scenario: Seed task populates the cache

- **WHEN** an operator runs the seed task with `library/testdata/cue-cache/` empty
- **THEN** the directory is populated with the pinned schema module in CUE's cache layout
- **AND** subsequent test runs succeed without further network access

#### Scenario: Seed task is rerun after CUE bump

- **WHEN** the CUE SDK version pinned in `library/go.mod` changes
- **THEN** the operator reruns the seed task as part of the upgrade, and `MIGRATIONS.md` documents this as a required step

### Requirement: Synth test fixture relocates outside apis/

The `synthtest/fixture.cue` file previously located at `apis/core/synthtest/fixture.cue` SHALL be relocated to `library/testdata/synth/fixture.cue`. Tests under `opm/helper/synth/` and `opm/kernel/` that previously resolved that fixture via `apisCoreDir()` or `kernelSynthApisCoreDir()` SHALL resolve the new path relative to the test file. The fixture's CUE content SHALL be preserved unchanged across the move; only its location and the test-side path-resolution helpers change.

#### Scenario: Fixture file exists at new location

- **WHEN** the change is applied
- **THEN** `library/testdata/synth/fixture.cue` exists with the same content the file previously had at `library/apis/core/synthtest/fixture.cue`
- **AND** no path resolving to `apis/core/synthtest/` remains in any test file

#### Scenario: Synth tests pass after fixture move

- **WHEN** `task test` runs against the relocated fixture
- **THEN** `opm/helper/synth/release_test.go` and `opm/kernel/synth_test.go` pass without modification beyond the path-resolution helper change

### Requirement: No test-only Loader in the public API

The library MUST NOT export any `schema.Loader` implementation intended for test use (no `FSLoader`, no `MemoryLoader`, no `StaticLoader`). The public `opm/schema` package contains exactly one Loader: `OCILoader`. Any test-side construction of `*schema.Cache` SHALL use `OCILoader` against the pre-seeded cache directory.

The library MAY introduce internal-only test helpers under `internal/` for fixture path resolution and `*schema.Cache` construction, but these MUST NOT be exported.

#### Scenario: Public symbols enumerated

- **WHEN** a consumer enumerates `opm/schema` exported types
- **THEN** the only type satisfying `Loader` is `OCILoader`

#### Scenario: Tests do not import an FSLoader

- **WHEN** the library's own tests are inspected
- **THEN** no test file constructs a `Loader` other than `OCILoader`, nor calls any unexported test-only `Cache` constructor that bypasses `Loader.Load`
