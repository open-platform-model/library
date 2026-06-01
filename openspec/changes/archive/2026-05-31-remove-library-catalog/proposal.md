## Why

The OPM core catalog has been extracted from the library into its own repository
(`catalog_opm`, published as `opmodel.dev/catalogs/opm@v0`, latest `v0.4.0` on
GHCR). The library no longer owns the catalog source or its release pipeline —
keeping the source tree at `library/modules/opm/` and the `publish-catalog.yml` /
`cue:publish:catalog` machinery alive means two repos can publish the same OCI
module, with the library's `cue-versions.yml` version as a competing source of
truth. This change decommissions the catalog inside the library: it deletes the
source tree and the publish path, and retires the two specs that asserted the
catalog "SHALL remain at `library/modules/opm/`".

The library keeps exactly one relationship to the catalog: it *consumes* the
published `opmodel.dev/catalogs/opm@v0` from GHCR in its test fixtures. That
consumer path already landed (commit `2e2f88f`): `testdata/modules/web_app` pins
`@v0.4.0`, `modules/opm_platform` subscribes by registry path (resolved at
materialize time against the latest published tag), the flow integration test
pulls from GHCR, and the `cue:catalog:drift` guard fails CI if a fixture falls
behind the latest published tag. None of that is touched here.

## What Changes

- **Delete the catalog source tree** `library/modules/opm/` in full (resources,
  traits, blueprints, transformers, schemas, `catalog.cue`, `identity/`,
  `cue.mod/`). It lives in the `catalog_opm` repo now.
- **Delete `library/modules/out.cue`** — a stale generated `#Platform` artifact
  importing the long-dead `opmodel.dev/modules/opm/*` paths.
- **Delete `library/.github/workflows/publish-catalog.yml`** — the library-side
  registry-presence publisher. Publishing is the `catalog_opm` repo's job.
- **Remove the `cue:publish:catalog` task** (and its `CATALOG_DIR: modules/opm`
  / `.build/catalog` staging) from `library/Taskfile.yml`.
- **Remove the `catalogs/opm:` entry** from `library/cue-versions.yml` and the
  header comment describing the catalog's publish flow. The file keeps only the
  in-repo fixture entries it verifies (never publishes).
- **Update stale doc/comment references** to `modules/opm` as a publishable
  module in `library/CLAUDE.md` and `library/Taskfile.yml` help text.
- **Retire two specs** — `catalog-packaging` and `catalog-publishing` — entirely.
  Both describe a library-owned catalog that no longer exists here.

Explicitly **out of scope / unchanged**: the `cue:catalog:drift` guard, the
`CATALOG_GHCR_REPO` / `CATALOG_FIXTURE_MODULES` / `CUE_GHCR_REGISTRY` vars, the
GHCR-resolving fixtures (`web_app`, `opm_platform`), the flow integration test,
and the `cue.yml` CI job — these are the *consumer* side and stay.

## Capabilities

### Removed Capabilities

- `catalog-packaging`: the structural contract for the OPM core catalog as a
  CUE module living at `library/modules/opm/`. The catalog is now authored and
  versioned in the `catalog_opm` repo; the library no longer packages it.
- `catalog-publishing`: the release contract for publishing the catalog from the
  library (`cue:publish:catalog` stamping task, `0.0.0-dev` guard, the
  registry-presence `publish-catalog.yml` workflow, version sourcing from
  `cue-versions.yml`). Publishing now happens in the `catalog_opm` repo.

### Modified Capabilities
<!-- None. The Go-kernel specs (platform-materialization, platform-matching,
release-synthesis, etc.) are unchanged: they consume the catalog as a published
OCI module via Materialize regardless of where its source lives. -->

## Impact

- **CUE modules:** deletes `library/modules/opm/` (whole tree) and
  `library/modules/out.cue`. `modules/opm_platform` and
  `testdata/modules/web_app` are untouched.
- **Tooling/CI:** `library/Taskfile.yml` (drop `cue:publish:catalog`, fix help
  text), `library/.github/workflows/publish-catalog.yml` (delete),
  `library/cue-versions.yml` (drop `catalogs/opm` entry + comment). `cue.yml`
  is unchanged — it only consumes the published catalog via the drift guard +
  flow test.
- **Go code:** none. No `opm/` public API change. PATCH-equivalent from the
  Go-library SemVer view (Principle VI).
- **Registry:** the library stops being able to publish `opmodel.dev/catalogs/opm`.
  The `catalog_opm` repo is the sole publisher; GHCR already carries `v0.4.0`.
- **Scope note (Principle VIII):** a single cohesive removal — one thing (the
  in-library catalog) and its publish path leave together. Mechanically simple;
  no behavior change to the consuming kernel.
