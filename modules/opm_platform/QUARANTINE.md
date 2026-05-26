# Quarantined — pending enhancement 0001 library slice

`platform.cue` has been renamed to `_platform.cue.quarantined` and is **not** built by `cue` tooling (CUE skips files with non-`.cue` extensions).

## Why

The re-synced `apis/core/` schema is post-enhancement-0001:

- `#Platform.#registry` is path-keyed `[Path=#ModulePathType]: #Subscription`.
- `#FQNType` uses SemVer.

This file pre-dates that reshape on both fronts:

- `#registry: {opm: {#module: …, enabled: true}}` — Module-valued (old shape).
- Imports `opmodel.dev/core/v1alpha2@v1` — old module identifier.

A partial rewrite (path-only) does not produce a unifiable value; a full rewrite belongs to enhancement 0001's library slice, which rebuilds this fixture against `#Subscription` and the repackaged `opmodel.dev/catalogs/opm@0.1.0` catalog.

See [openspec/changes/remove-api-binding-dispatch/design.md](../../openspec/changes/remove-api-binding-dispatch/design.md) §D10.

## Restoration

Enhancement 0001's library slice replaces the quarantined file with one written against `#Subscription` and the repackaged catalog. At that point, restore the `.cue` extension and delete this notice.
