# catalog-publishing (REMOVED)

The library no longer publishes the catalog. Publishing is owned by the
`catalog_opm` repo. Every requirement in this capability is removed.

## REMOVED Requirements

### Requirement: Dedicated catalog publish task with version stamping

**Reason**: `cue:publish:catalog` (rsync-to-`.build/` + `version_override.cue`
stamping) is deleted from the library; the equivalent publish flow lives in
`catalog_opm`.

### Requirement: Publish guard rejects the dev version

**Reason**: The `0.0.0-dev` publish guard moved with the publish task to
`catalog_opm`.

### Requirement: Version layering separates OCI tag from FQN

**Reason**: The OCI-tag-vs-FQN version layering is enforced where the catalog is
now published (`catalog_opm`), not in the library.

### Requirement: Registry-presence publish workflow

**Reason**: `library/.github/workflows/publish-catalog.yml` is deleted; the
`catalog_opm` repo runs the catalog's release workflow.

### Requirement: Legacy publish paths retired

**Reason**: Subsumed — the library now publishes no catalog at all. The
`catalogs/opm` entry leaves `cue-versions.yml`; the remaining entries are
in-repo fixtures that are verified (never published).
