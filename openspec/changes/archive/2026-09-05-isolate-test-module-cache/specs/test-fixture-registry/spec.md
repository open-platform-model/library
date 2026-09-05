## Purpose

The in-process fixture registry harness (`opm/internal/registrytest`, with the cache-directory helpers in `opm/internal/schematest`) that the kernel, loader, synth and platform-module test packages share: how served fixtures are resolved and cached so that tests stay hermetic across packages running as parallel processes, while the OPM core schema stays warm and is fetched once.

## ADDED Requirements

### Requirement: Served fixtures resolve from a per-test module cache

Every fixture registry the harness stands up (inline catalog fixtures, inline module fixtures, committed fixture trees) SHALL give the test a private CUE module cache directory owned by that test and empty for every served coordinate, and SHALL point `CUE_CACHE_DIR` at it for the test scope. A coordinate the registry serves SHALL be fetched and extracted into that private cache on first use within the test; no bytes cached by an earlier test or by another process for that coordinate SHALL be visible to the build. The private cache SHALL be removed when the test ends, read-only extracted directories included.

#### Scenario: Committed fixture edited under a fixed coordinate

- **WHEN** a committed fixture tree (for example `testdata/render/registry`) is edited without changing the version its module file declares, and a test serves it
- **THEN** the test builds against the edited bytes

#### Scenario: Two packages serve the same coordinate concurrently

- **WHEN** two test packages, running as separate processes, each stand up a registry serving the same fixture coordinate
- **THEN** each resolves its own copy from its own cache, and neither observes a partially extracted or removed directory

#### Scenario: Same test name in two packages

- **WHEN** tests in two packages share a name, so the harness derives the same unique fixture path for both, and they publish different fixture bytes under it
- **THEN** each test builds against its own bytes

#### Scenario: Private cache is cleaned up

- **WHEN** a test that stood up a registry ends
- **THEN** its private cache directory is removed and the test does not fail on temp-directory cleanup

### Requirement: The shared workspace cache is never deleted from

The harness SHALL NOT remove any entry from the shared workspace module cache. No eviction mechanism SHALL exist in the test tree: freshness of served fixtures comes from the private cache, never from deleting shared state that another process may be reading.

#### Scenario: Harness holds no eviction path

- **WHEN** a developer searches `opm/internal` for code that removes entries from the shared workspace cache
- **THEN** none exists

#### Scenario: Full suite with parallel packages

- **WHEN** the full suite runs with packages in parallel, repeatedly
- **THEN** no test fails because a shared cache directory disappeared mid-read (the `import failed: open cue.mod/module.cue: no such file or directory` failure)

### Requirement: The opmodel.dev namespace resolves through the shared tier

Within a private cache, the `opmodel.dev` module namespace (`opmodel.dev/core`, `opmodel.dev/catalogs/...`) SHALL resolve through the shared workspace cache: the private cache's extract and download subtrees for that namespace SHALL be the shared cache's own, so a version fetched once is reused by every test and process, and a fetch of a not-yet-cached `opmodel.dev` version lands in the shared cache through the module cache's own lock-protected fetch path. The harness itself SHALL write nothing into the shared tier.

#### Scenario: Warm core is reused

- **WHEN** the shared workspace cache already holds the core version the fixtures declare and a test stands up a registry
- **THEN** the test resolves core from the shared cache with no registry round-trip

#### Scenario: Cold core is fetched once

- **WHEN** the shared workspace cache does not hold that core version
- **THEN** the first test to need it fetches it into the shared cache, and later tests in any package reuse it

#### Scenario: Concurrent cold fetch

- **WHEN** two test processes need the same not-yet-cached `opmodel.dev` version at the same time
- **THEN** the module cache's lock serialises the extraction and both tests build against the completed directory

### Requirement: Tests that need only opmodel.dev keep the shared cache

Tests that serve no fixture (schema cache tests, file-loader tests, synth unit tests, and the flow and live tests that pull from GHCR) SHALL keep using the shared workspace cache directly through the shared-cache environment helper.

#### Scenario: Schema cache test

- **WHEN** a test configures the environment with the shared-cache helper and loads `opmodel.dev/core@v2`
- **THEN** it resolves from the shared workspace cache and no private cache is created
