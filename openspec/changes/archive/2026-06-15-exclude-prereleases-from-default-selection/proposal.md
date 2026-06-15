## Why

An unfiltered subscription (`enable: true`, no `filter`) resolves to the *highest published SemVer* — including pre-release tags. When a `v0.6.0-dev.*` pre-release was published to the catalog's GHCR repo, every unfiltered platform silently materialized that in-progress dev build instead of the latest release (`v0.5.1`). Because transformer match is exact-FQN and FQNs embed the version, no stable module fixture could match the `@0.6.0-dev.*` index, and the `library` flow integration tests (`TestFlow_WebApp_OnOpmPlatform`, `..._SynthPath_...`) failed with empty match maps.

The catalog drift CI check already treats pre-releases as non-authoritative (it compares fixtures against the highest *stable* GHCR tag). Materialize's default selection contradicted that intent: drift demanded fixtures at the latest stable, while materialize matched against the latest pre-release — a contradiction no fixture pin can satisfy.

This also aligns with SemVer convention: a constraint does not match a pre-release unless the constraint itself names one. The default should behave the same way.

## What Changes

This change is **behavioral, MINOR** (no signature change; a more predictable default for the no-filter path).

- **MODIFIED no-filter selection** — an enabled subscription with no `filter` now selects the highest published **stable** (non-pre-release) version. If a path has published *only* pre-releases, Materialize falls back to the highest pre-release so the path still materializes.
- **Pre-releases remain reachable, but only by explicit opt-in** — `filter.allow` naming an exact pre-release version, or a `filter.range` whose constraint contains a pre-release (standard Masterminds/semver semantics). The `range`/`allow`/`deny` algorithm is otherwise unchanged.

Out of scope: any change to enumeration, the `range`/`allow`/`deny` ordering, multi-version composition, or the opt-in cache. The fix is confined to the no-filter survivor selection.

Note (not fixed here): catalog `v0.4.0` does not parse under the current toolchain (`missing ',' in argument list`). It is harmless under the new default (never selected), but any future `filter.range`/`filter.allow` that selects it will fail Materialize. Tracked separately.

## Capabilities

### Modified Capabilities

- `platform-materialization`: the no-filter default now excludes pre-releases (highest stable, with pre-release-only fallback); pre-releases require explicit `filter.allow`/`filter.range` opt-in.

## Impact

- **`opm/materialize/filter.go`**: the `f.isEmpty()` branch of `filterVersions` selects the highest stable via a new unexported `highestStable` helper.
- **`testdata/modules/web_app/cue.mod/module.cue`**: catalog pin bumped `v0.5.0 → v0.5.1` so the fixture tracks the latest stable (also clears the existing drift-check failure).
- **Docs**: `filter.go` godoc + `MIGRATIONS.md` note; this spec delta.
- **Tests**: table-test for `highestStable` / no-filter pre-release exclusion; the flow integration tests pass once the default no longer picks `v0.6.0-dev.*`.
- **SemVer**: **MINOR**. No `opm/` signature change. Behavior of an unfiltered subscription changes from "highest published" to "highest stable"; callers relying on a pre-release being auto-selected must now name it via `filter.allow`/`filter.range`.
- **Downstream**: `cli/`, `opm-operator/` unaffected at the API level; their unfiltered platforms simply stop auto-selecting pre-releases.
