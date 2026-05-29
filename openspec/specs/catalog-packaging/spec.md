# catalog-packaging Specification

## Purpose
TBD - created by syncing change repackage-opm-catalog. Update Purpose after archive.
## Requirements
### Requirement: Catalog module identity targets core@v0

The OPM core catalog SHALL be a CUE module identified as `opmodel.dev/catalogs/opm@v0` and SHALL declare its schema dependency as `opmodel.dev/core@v0`. No file in the catalog tree SHALL import `opmodel.dev/core/v1alpha2@v1` or any other pre-`@v0` core path. The on-disk location SHALL remain `library/modules/opm/`.

#### Scenario: Catalog vets against core@v0

- **WHEN** `cue vet ./...` runs from `library/modules/opm/` with `CUE_REGISTRY` resolving `opmodel.dev/core@v0`
- **THEN** evaluation succeeds with no unresolved-import or unknown-field errors

#### Scenario: No legacy core import survives

- **WHEN** the catalog tree is searched for `opmodel.dev/core/v1alpha2`
- **THEN** zero matches are found

### Requirement: Single identity source for path and version

The catalog SHALL expose a sibling `identity/` subpackage that is the sole source of the catalog's `ModulePath` and `Version`. Every primitive (`#Resource`, `#Trait`, `#Blueprint`, `#ComponentTransformer`) SHALL derive `metadata.modulePath` and `metadata.version` from `identity/` rather than from a hardcoded literal. The committed `identity.Version` SHALL default to `"0.0.0-dev"`.

#### Scenario: Primitive metadata sources from identity

- **WHEN** a primitive's `metadata.version` is evaluated in the in-repo (unpublished) tree
- **THEN** it resolves to `identity.Version` (`"0.0.0-dev"`) and its `metadata.modulePath` resolves to an `identity.ModulePath`-derived path (e.g. `"\(id.ModulePath)/resources"`)

#### Scenario: No stray hardcoded catalog path or non-SemVer version remains

- **WHEN** the catalog tree is searched for `version: "v1"` or `modulePath:` literals containing `opmodel.dev/modules/opm`
- **THEN** zero matches are found

### Requirement: Catalog manifest embeds #Catalog

The catalog root SHALL contain a `catalog.cue` that embeds a bare `c.#Catalog` value (modules-pattern style, no `Catalog:` wrapper), sets `metadata` from `identity/`, and populates `#transformers` keyed by each transformer's own `metadata.fqn`. Resources, traits, and blueprints SHALL surface transitively through transformer required/optional maps and SHALL NOT be enumerated in the manifest.

#### Scenario: Manifest evaluates to a concrete catalog

- **WHEN** the `catalog.cue` package is evaluated
- **THEN** `metadata.fqn` is concrete and matches `core`'s `#CatalogFQNType`, and every `#transformers` entry's map key equals that transformer's `metadata.fqn`

### Requirement: Transformer metadata is schema-stamped in lockstep

The `#Catalog.#transformers` pattern constraint SHALL stamp every transformer entry's `metadata.modulePath` to `"<catalog-modulePath>/transformers"` and `metadata.version` to the catalog's version. A transformer whose authored `modulePath` or `version` diverges from the stamp SHALL be rejected by `cue vet` with a conflicting-values diagnostic.

#### Scenario: Divergent transformer subpath fails loudly

- **WHEN** a transformer authors `metadata.modulePath` with a typo (e.g. `/trasnformers`) and the catalog is vetted
- **THEN** `cue vet` exits non-zero with a "conflicting values" error citing the offending file and line

#### Scenario: Canonical transformer metadata unifies cleanly

- **WHEN** a transformer authors `metadata.modulePath`/`version` identical to the stamp (sourced from `identity/`)
- **THEN** `cue vet` exits zero

### Requirement: Same-module imports omit the major-version qualifier

Imports referencing other packages within the catalog module SHALL NOT carry a trailing `@vN` major-version qualifier; cross-module imports (`core@v0`, `k8s.io@v0`) SHALL retain theirs. The catalog tree SHALL contain no same-module import bearing a stale `@v1` qualifier.

#### Scenario: Self-imports resolve under the @v0 module

- **WHEN** `cue vet ./...` runs against the `@v0` catalog module
- **THEN** all `opmodel.dev/catalogs/opm/...` self-imports resolve and no `import failed: cannot find package ...@v1` error is raised

### Requirement: Catalog is consumable by a #Subscription platform

A `#Platform` whose `#registry` subscribes to the catalog SHALL resolve and materialize against the published catalog. The in-repo `opm_platform` fixture SHALL be expressed with a `#Subscription`-shaped `#registry`, import the republished catalog, and vet cleanly once the catalog tag exists.

#### Scenario: Restored fixture vets against the published catalog

- **WHEN** the `opm_platform` fixture is vetted after the catalog is published
- **THEN** evaluation succeeds and the fixture's subscription path resolves to the published catalog module

#### Scenario: Quarantine is fully lifted

- **WHEN** the `opm_platform` directory is inspected after restoration
- **THEN** `platform.cue` exists as a built `.cue` file and neither `_platform.cue.quarantined` nor `QUARANTINE.md` remains
