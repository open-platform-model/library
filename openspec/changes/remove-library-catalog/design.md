# Design

## Research & Decisions

### Where does the catalog live now, and what stays in the library?

**Context**: The catalog was migrated to the `catalog_opm` repo and published as
`opmodel.dev/catalogs/opm@v0.4.0` (verified: GHCR's only published tag is
`v0.4.0`). We must decide what the library deletes versus keeps.

**Explored**: Traced every reference to `modules/opm` / `catalogs/opm` across Go,
CUE, Taskfile, workflows, and specs.

**Decision**: Split references into two clean halves.

- *Producer side (delete)* — everything that authored or published the catalog
  from the library: `modules/opm/`, `out.cue`, `publish-catalog.yml`,
  `cue:publish:catalog`, the `cue-versions.yml` `catalogs/opm` entry, and the
  `catalog-packaging` / `catalog-publishing` specs.
- *Consumer side (keep)* — everything that pulls the published catalog from GHCR:
  `testdata/modules/web_app` (pins `@v0.4.0`), `modules/opm_platform`
  (subscribes by registry path, materialized against the latest tag), the flow
  integration test, the `cue:catalog:drift` guard, and the `CATALOG_GHCR_REPO` /
  `CATALOG_FIXTURE_MODULES` / `CUE_GHCR_REGISTRY` Taskfile vars.

**Rationale**: Two repos publishing the same OCI module is a split-brain hazard;
the producer side MUST be single-homed in `catalog_opm`. The consumer side is
how the kernel's Materialize/Match/Compile flow is exercised end-to-end and is
independent of where the catalog source lives — it already resolves from GHCR.

### Does deleting `modules/opm/` break `cue:vet` or the flow test?

**Context**: `CUE_MODULE_GLOBS` includes `modules/*`, so `cue:discover` /
`cue:vet` currently sweep `modules/opm` and `modules/opm_platform`.

**Decision**: Safe. After deletion the glob resolves only `modules/opm_platform`
(plus `testdata/modules/web_app`). Verified no Go code loads `modules/opm` from
disk; the flow test loads only `modules/opm_platform` + `testdata/modules/web_app`,
both of which import the catalog from GHCR (`web_app`) or subscribe by path
(`opm_platform`) — neither references the on-disk `modules/opm` tree.

**Rationale**: The on-disk catalog was never a build input for the consuming
fixtures once they were repointed to GHCR; it was only a publish source.

### `modules/opm_platform` has no catalog dep in `cue.mod` — is that correct?

**Context**: One might expect the platform fixture to pin `catalogs/opm@v0.4.0`
like `web_app` does.

**Decision**: Leave `opm_platform/cue.mod/module.cue` as-is (core@v0 only). It
subscribes to the catalog by registry *path* (`"opmodel.dev/catalogs/opm"`),
which the kernel resolves at Materialize time by enumerating published versions
and pulling the latest — there is no CUE-level import to pin. Only `web_app`
(which imports catalog packages directly) carries the dep pin.

**Rationale**: Adding a phantom dep would not be exercised and would drift.

### Spec removal mechanics

**Context**: The user approved deleting both specs outright. OpenSpec normally
syncs change deltas into `openspec/specs/` on archive.

**Decision**: Author full `## REMOVED Requirements` deltas for every requirement
in both capabilities (the auditable record of *what* left and *why*), and delete
the live `openspec/specs/catalog-packaging/` and `openspec/specs/catalog-publishing/`
directories as the final implementation step — the end state an archive sync
would produce when every requirement is removed.

**Rationale**: Keeps the change self-describing while reaching the approved end
state (both specs gone) in one pass.
