## ADDED Requirements

### Requirement: A pinned schema release is known without a load

`OCILoader` SHALL report, without any I/O, the exact core release its module identifier names: `PinnedVersion()` returns that release and `true` when `Module` (or `DefaultSchemaModule` when `Module` is empty) names a full release, and `("", false)` when it names a bare major. A kernel whose configured loader pins an exact release SHALL derive the core release its synthesized instances import from that pin and SHALL NOT load the schema to do so; a kernel whose loader names a bare major SHALL resolve the release through its schema cache as before.

#### Scenario: Default loader is pinned

- **WHEN** `(schema.OCILoader{}).PinnedVersion()` is called
- **THEN** it returns `DefaultSchemaVersion()` and `true`, with no registry or cache access

#### Scenario: Bare major is not pinned

- **WHEN** `(schema.OCILoader{Module: "opmodel.dev/core@v2"}).PinnedVersion()` is called
- **THEN** it returns `""` and `false`

#### Scenario: Synthesis on a pinned kernel loads no schema

- **WHEN** a kernel constructed with the default loader synthesizes an instance
- **THEN** the synthesized package imports core at the pinned release's major, and `k.SchemaCache().ResolvedVersion()` is still `""` afterwards because no load ran

#### Scenario: Synthesis on a bare-major kernel resolves through the cache

- **WHEN** a kernel constructed with `WithSchemaLoader(schema.OCILoader{Module: "opmodel.dev/core@v2"})` synthesizes an instance
- **THEN** the schema cache is loaded once and the synthesized package imports core at the resolved release's major
