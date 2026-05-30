# catalog-packaging (REMOVED)

The catalog is no longer packaged in the library. Its source moved to the
`catalog_opm` repo (`opmodel.dev/catalogs/opm@v0`). Every requirement in this
capability is removed; the library now only *consumes* the published catalog
(see the GHCR-resolving fixtures + `cue:catalog:drift`, which are not governed by
this spec).

## REMOVED Requirements

### Requirement: Catalog module identity targets core@v0

**Reason**: The catalog source no longer lives at `library/modules/opm/`; module
identity and its `core@v0` dependency are now owned by the `catalog_opm` repo.

### Requirement: Single identity source for path and version

**Reason**: The `identity/` subpackage moved to `catalog_opm` along with the
catalog source.

### Requirement: Catalog manifest embeds #Catalog

**Reason**: `catalog.cue` and the `#transformers` map are authored in
`catalog_opm`.

### Requirement: Transformer metadata is schema-stamped in lockstep

**Reason**: Transformer authoring and stamping moved to `catalog_opm`.

### Requirement: Same-module imports omit the major-version qualifier

**Reason**: There is no in-library catalog module left to govern self-import
qualifiers.

### Requirement: Catalog is consumable by a #Subscription platform

**Reason**: Consumption of the published catalog by the `opm_platform` fixture is
exercised by the flow integration test and the `cue:catalog:drift` guard, not by
a library-owned packaging spec. The behavior survives; the requirement does not
belong to a removed packaging capability.
