## Why

The raw Kubernetes catalog `opmodel.dev/catalogs/k8s@v1` is retired; `opmodel.dev/catalogs/opm@v4`
is the only first-party catalog. The library still names the retired path as an OPM-owned module
(the beta-line lists in the README, the constitution and the OpenSpec config, the
`.cascade-frozen` pins, two main specs) and as the sample second catalog in the platform-module
generator tests. A reader taking those names at face value would look for a catalog that no
longer exists.

## What Changes

- Drop `opmodel.dev/catalogs/k8s@v1` from the beta-line lists in `README.md`, `CONSTITUTION.md`
  and `openspec/config.yaml`.
- The `platformmodule` tests use `example.com/catalogs/extra@v1` as their second catalog. It sorts
  before `opmodel.dev/catalogs/opm@v4` exactly as the retired path did, so the positional aliases
  (`cat0`, `cat1`) and the sorted-order goldens hold unchanged. The generator comment that said
  "both first-party catalogs" says "the first-party catalog".
- `.cascade-frozen` stops listing the retired path for those two test files.
- Spec text: `fixture-pin-maintenance` no longer names the retired path as an OPM-owned module;
  the `platform-module-generation` "Two catalogs" scenario names the neutral sample catalog.
- `adr/010-beta-migration-notes-in-changelog.md` keeps its historical mention.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `fixture-pin-maintenance`: the OPM-owned module paths no longer include the retired k8s catalog.
- `platform-module-generation`: the two-catalog scenario uses a neutral second catalog.

## Impact

Documentation, test samples and two spec texts only. No Go API, kernel behavior or published
artifact changes.
