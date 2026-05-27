## ADDED Requirements

### Requirement: Tests acquire the schema via the real OCILoader against the configured registry

Tests that need a built schema `cue.Value` SHALL exercise the same `OCILoader` code path used in production, configured to use a workspace-local CUE module cache at `library/.cue-cache/`. Tests MUST NOT use a test-only `Loader` implementation. Tests MAY depend on network access for the cold-cache path; the workspace-local cache directory is gitignored and populated on first test run by `OCILoader.Load` fetching from the configured `CUE_REGISTRY`.

The test helper that constructs a `*schema.Cache` for test use SHALL:

- Resolve the absolute path to `library/.cue-cache/` relative to the test file location (e.g., via `runtime.Caller`).
- Set `OCILoader.CacheDir` (or `CUE_CACHE_DIR` via `t.Setenv`) to the workspace-local path so test-driven fetches do not pollute the user's `~/.cache/cuelang/mod/`.
- Set `CUE_REGISTRY` to `schema.PublicRegistry` (via `OCILoader.Registry` or `t.Setenv`) so the test loader resolves `opmodel.dev/core` against the same registry mapping production callers configure.
- Construct a fresh `*schema.Cache` per test (or per test-package) so memoization scope is explicit and one test's state does not leak into another.

#### Scenario: Tests pass on a warm workspace cache

- **WHEN** the test suite is run after a previous test run has populated `library/.cue-cache/`
- **THEN** every test that constructs a schema `Cache` via the test helper succeeds without contacting the registry

#### Scenario: Tests fetch on cold workspace cache

- **WHEN** the test suite is run with `library/.cue-cache/` absent or empty
- **THEN** the test helper triggers a fetch from `CUE_REGISTRY` (default `schema.PublicRegistry` → GHCR), populating the workspace cache
- **AND** the test suite proceeds without further intervention

#### Scenario: Test helper does not pollute process state

- **WHEN** a test constructs a schema `Cache` via the helper and the test completes
- **THEN** any environment variable mutation (e.g., `CUE_CACHE_DIR`, `CUE_REGISTRY`) is reverted at test scope (`t.Cleanup` or `t.Setenv` semantics)

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
