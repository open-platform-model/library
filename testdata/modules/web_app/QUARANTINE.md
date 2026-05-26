# Quarantined — pending enhancement 0001 library slice

`module.cue` and `components.cue` have been renamed to `_module.cue.quarantined` and `_components.cue.quarantined`. CUE tooling skips them (non-`.cue` extension).

## Why

The re-synced `apis/core/` schema is post-enhancement-0001:

- `#FQNType` uses SemVer (`example.com/foo@0.1.0`, not `@v1`).
- `#Module` carries `#ctx` instead of `#defines`.

This fixture pre-dates that reshape on every front:

- Imports `opmodel.dev/core/v1alpha2@v1` (old module identifier).
- References the `opmodel.dev/modules/opm` catalog at MAJOR-only `@v1` FQNs (pre-SemVer).

A partial rewrite (paths only) does not produce a unifiable value; a full rewrite requires authoring against `#Subscription`, `#ctx`, SemVer FQNs, and the repackaged `opmodel.dev/catalogs/opm@0.1.0` catalog — all of which land with enhancement 0001's library slice.

See [openspec/changes/remove-api-binding-dispatch/design.md](../../../openspec/changes/remove-api-binding-dispatch/design.md) §D10.

## Restoration

Enhancement 0001's library slice replaces the quarantined files with ones written against `#Subscription` / `#ctx` / SemVer FQNs. At that point, restore the `.cue` extensions and delete this notice.
