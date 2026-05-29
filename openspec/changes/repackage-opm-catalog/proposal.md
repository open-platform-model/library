## Why

The OPM core catalog at `library/modules/opm/` is broken and half-migrated: it still imports `opmodel.dev/core/v1alpha2@v1` and pins `opmodel.dev/core@v1.0.6` (which no longer resolves), so `cue vet ./...` fails outright. Enhancement 0001's core slice already shipped `opmodel.dev/core@v0.3.0` with a SemVer-only `#FQNType`, a top-level `#Catalog` definition (D19/D25), and a `#Subscription`-shaped `#Platform.#registry` — but nothing downstream consumes that new shape. This change re-targets the catalog onto `core@v0`, restructures it to the `#Catalog` contract, publishes it under its new identity, and restores the one in-repo consumer that depends on it. It is the `library`/`modules` slice of enhancement 0001 (D22 sequencing: Part B has shipped, so this can land).

## What Changes

- **BREAKING (catalog OCI module):** the catalog republishes under a new identifier `opmodel.dev/catalogs/opm@v0`, restarting at `v0.1.0` (D23 hard switch — legacy `opmodel.dev/modules/opm@v1.x` is not republished or aliased).
- Re-target every primitive from `opmodel.dev/core/v1alpha2@v1` to `opmodel.dev/core@v0`; update the `cue.mod/module.cue` dep via `task update-deps`.
- Add a sibling `identity/` subpackage (`ModulePath` + `Version`) and a root `catalog.cue` embedding bare `c.#Catalog` (modules pattern, `M=metadata` label-alias per D25). Every primitive sources `metadata.modulePath`/`version` from `identity/`; the `#Catalog.#transformers` pattern schema-stamps transformer metadata (D19).
- Replace every `version: "v1"` literal with `id.Version` (forced — the new `#FQNType` requires SemVer, so `@v1` no longer matches) and every hardcoded `modulePath:` literal with an `id.ModulePath`-derived value.
- Sweep all same-module self-imports `opmodel.dev/modules/opm/*` → `opmodel.dev/catalogs/opm/*`, dropping the now-stale `@vN` qualifier on same-module subpackage imports (proven to break under an `@v0` module).
- Add a dedicated `cue:publish:catalog` task (rsync → `.build/` + generated `identity/version_override.cue` stamp + `cue vet` + `cue mod publish vX.Y.Z` + reject `0.0.0-dev`, per D8/D9/D19). Standalone registry-presence (version-gated) publish — **not** release-please; the Go-module release-please path is untouched.
- Replace `publish-cue.yml` with a catalog-only `publish-catalog.yml`; retarget/clean `cue-versions.yml` (drop dead `apis/core`, flip `modules/opm` → `catalogs/opm` at `v0.1.0`).
- Restore the quarantined `modules/opm_platform/_platform.cue.quarantined` against `#Subscription` + the republished catalog; un-quarantine it; **stop publishing it** (it is purely the flow-test fixture).

## Capabilities

### New Capabilities

- `catalog-packaging`: the structural contract for the OPM core catalog as a CUE module — `opmodel.dev/catalogs/opm@v0` identity, `identity/` subpackage as the single SemVer/path source, root `catalog.cue` embedding `c.#Catalog`, primitive metadata sourcing, transformer schema-stamping, and the `core@v0` dependency. Includes the `#Subscription`-shaped consumer fixture that must resolve against the published catalog.
- `catalog-publishing`: the release contract for the catalog — the `cue:publish:catalog` stamping task, the `0.0.0-dev` publish guard, the registry-presence (version-gated) `publish-catalog.yml` workflow, version sourcing from `cue-versions.yml` (manual bump), and the first `v0.1.0` OCI tag.

### Modified Capabilities
<!-- None. The Go-kernel specs (platform-materialization, platform-matching, etc.) are unchanged — materialize/match already landed and consume #composedTransformers/#matchers off the materialized platform regardless of catalog identity. -->

## Impact

- **CUE modules:** `library/modules/opm/` (every `.cue` file — imports, metadata, new `identity/` + `catalog.cue`), `library/modules/opm_platform/` (registry reshape, un-quarantine), their `cue.mod/module.cue` deps.
- **Tooling/CI:** `library/Taskfile.yml` (new `cue:publish:catalog`), `library/.github/workflows/` (`publish-cue.yml` → `publish-catalog.yml`), `library/cue-versions.yml`.
- **Go code:** none — no `opm/` public API change. From the Go-library SemVer view this is PATCH-equivalent (Principle VI); the break is confined to the catalog's own OCI module versioning.
- **Registry:** publishes `opmodel.dev/catalogs/opm@v0.1.0` to the local registry (and GHCR via CI). Legacy `opmodel.dev/modules/opm@v1.x` and `opmodel.dev/modules/opm-platform@v1` cease to be published.
- **Downstream:** workspace `modules/*` (jellyfin, garage, …) that import the old catalog path rewire as a separate non-blocking wave once `v0.1.0` exists (D23) — out of scope here.
- **Scope note (Principle VIII):** wide but mechanically cohesive and naturally phased; `tasks.md` sequences it into three independently verifiable commit groups (repackage → publish → fixture restore), with phase 3 gated on the phase-2 tag existing.
