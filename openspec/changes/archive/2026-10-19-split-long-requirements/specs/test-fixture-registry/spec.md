## MODIFIED Requirements

### Requirement: The opmodel.dev namespace resolves through the shared tier
Within a private cache, the `opmodel.dev` module namespace (`opmodel.dev/core`, `opmodel.dev/catalogs/...`) SHALL resolve through the shared workspace cache: the private cache's extract and download subtrees for that namespace SHALL be the shared cache's own, so a version fetched once is reused by every test and process, and a fetch of a not-yet-cached `opmodel.dev` version lands in the shared cache through the module cache's own lock-protected fetch path.

#### Scenario: Warm core is reused

- **WHEN** the shared workspace cache already holds the core version the fixtures declare and a test stands up a registry
- **THEN** the test resolves core from the shared cache with no registry round-trip

#### Scenario: Cold core is fetched once

- **WHEN** the shared workspace cache does not hold that core version
- **THEN** the first test to need it fetches it into the shared cache, and later tests in any package reuse it

#### Scenario: Concurrent cold fetch

- **WHEN** two test processes need the same not-yet-cached `opmodel.dev` version at the same time
- **THEN** the module cache's lock serialises the extraction and both tests build against the completed directory

## REMOVED Requirements

### Requirement: Served fixtures resolve from a per-test module cache
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Every fixture registry the harness stands up gives the test a private CUE module cache directory", "A served coordinate is fetched into the private cache on first use and no earlier bytes are visible" and "The private cache is removed when the test ends".

## ADDED Requirements

### Requirement: Every fixture registry the harness stands up gives the test a private CUE module cache directory
Every fixture registry the harness stands up (inline catalog fixtures, inline module fixtures, committed fixture trees) SHALL give the test a private CUE module cache directory owned by that test and empty for every served coordinate, and SHALL point `CUE_CACHE_DIR` at it for the test scope.

#### Scenario: Two packages serve the same coordinate concurrently

- **WHEN** two test packages, running as separate processes, each stand up a registry serving the same fixture coordinate
- **THEN** each resolves its own copy from its own cache, and neither observes a partially extracted or removed directory

#### Scenario: Same test name in two packages

- **WHEN** tests in two packages share a name, so the harness derives the same unique fixture path for both, and they publish different fixture bytes under it
- **THEN** each test builds against its own bytes

### Requirement: A served coordinate is fetched into the private cache on first use and no earlier bytes are visible
A coordinate the registry serves SHALL be fetched and extracted into that private cache (Requirement: Every fixture registry the harness stands up gives the test a private CUE module cache directory) on first use within the test; no bytes cached by an earlier test or by another process for that coordinate SHALL be visible to the build.

#### Scenario: Committed fixture edited under a fixed coordinate

- **WHEN** a committed fixture tree (for example `testdata/render/registry`) is edited without changing the version its module file declares, and a test serves it
- **THEN** the test builds against the edited bytes

### Requirement: The private cache is removed when the test ends
The private cache (Requirement: Every fixture registry the harness stands up gives the test a private CUE module cache directory) SHALL be removed when the test ends, read-only extracted directories included.

#### Scenario: Private cache is cleaned up

- **WHEN** a test that stood up a registry ends
- **THEN** its private cache directory is removed and the test does not fail on temp-directory cleanup

### Requirement: The harness itself writes nothing into the shared tier
The harness itself SHALL write nothing into the shared tier (Requirement: The opmodel.dev namespace resolves through the shared tier).

#### Scenario: The harness writes nothing into the shared tier

- **WHEN** the harness stands up a registry for a test
- **THEN** the harness itself writes nothing into the shared tier
