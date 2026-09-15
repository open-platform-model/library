## MODIFIED Requirements

### Requirement: Metadata decoders are free functions

The library SHALL decode each artifact's metadata through one unexported free function per artifact, living in the package of its single caller (`opm/module` for the module, `opm/platform` for the platform, `opm/catalog` for the catalog, `opm/kernel` for the instance); `opm/schema` SHALL NOT export a decoder. Each decoder MUST accept a raw `cue.Value` at the artifact root, read it through the `opm/schema` metadata path, and return the canonical decoded metadata struct (`ModuleMetadata`, `InstanceMetadata`, `PlatformMetadata`, `CatalogMetadata`, which stay exported from `opm/schema`) or a non-nil error. Consumers reach decoded metadata only through the artifact constructors and the kernel's acquisition paths (`Module.Metadata`, `Instance.Metadata`, `Platform.Metadata`, `Catalog.Metadata`).

#### Scenario: Decoding a module artifact

- **WHEN** `module.NewModuleFromValue(v)` is called with the root of a valid `#Module` value
- **THEN** `Module.Metadata` is a `*schema.ModuleMetadata` with `Name`, `ModulePath`, `Version`, `FQN`, `UUID`, `Labels`, `Annotations` populated

#### Scenario: Decoding a catalog artifact

- **WHEN** `catalog.NewCatalogFromValue(v)` is called with the root of a valid `#Catalog` value
- **THEN** `Catalog.Metadata` is a `*schema.CatalogMetadata` with `ModulePath`, `Version`, `FQN`, `Description`, `Labels`, `Annotations` populated
- **AND** it carries no `Name`, because a catalog's identity is its module path and the version stamped on every member it ships

#### Scenario: Missing metadata is fatal for module/instance/platform

- **WHEN** a module or platform constructor, or the kernel's instance processing, is given a value whose `metadata` field is absent
- **THEN** it returns an error stating "metadata field is required" and no partial artifact

#### Scenario: Missing metadata is fatal for a catalog

- **WHEN** `catalog.NewCatalogFromValue(v)` is given a value whose `metadata` field is absent
- **THEN** it returns an error stating "metadata field is required" and a nil catalog, on the same terms as the other three

#### Scenario: Platform metadata hoists top-level type

- **WHEN** `platform.NewPlatformFromValue(v)` is called on a `#Platform` whose root has `type: "kubernetes"` alongside its `metadata` block
- **THEN** the returned `Platform.Metadata.Type` is `"kubernetes"`

#### Scenario: Decoders are not exported

- **WHEN** a developer inspects the exported identifiers of `opm/schema`
- **THEN** none of `DecodeModuleMetadata`, `DecodeInstanceMetadata`, `DecodePlatformMetadata`, `DecodeCatalogMetadata` exists

#### Scenario: Provider metadata falls back to caller-supplied name

- **WHEN** a developer inspects the exported identifiers of `opm/schema`
- **THEN** neither `DecodeProviderMetadata` nor `ProviderMetadata` exists; the provider artifact was retired with the platform construct and is not among the kinds the kernel accepts
