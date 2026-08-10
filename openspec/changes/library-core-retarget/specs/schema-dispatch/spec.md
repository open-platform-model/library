# schema-dispatch — Delta

## MODIFIED Requirements

### Requirement: Single OPM schema, externally resolved, with no apiVersion field

The library SHALL consume exactly one OPM CUE schema package: `opmodel.dev/core@v2` (or a caller-pinned exact version, in any major, via `OCILoader.Module`), resolved through CUE's module system against `CUE_REGISTRY`. The library MUST NOT vendor or embed the schema source under `library/apis/core/` or any other in-tree location. The schema package MUST NOT define a top-level `#ApiVersion` constant. Artifact roots (`#Module`, `#ModuleInstance`, `#Component`, `#ComponentTransformer`, `#Platform`, `#Resource`, `#Trait`) MUST NOT carry an `apiVersion` field.

#### Scenario: No in-tree schema source

- **WHEN** the library tree is inspected after this change
- **THEN** no directory `library/apis/` exists
- **AND** no Go file embeds `opmodel.dev/core` source via `//go:embed`

#### Scenario: Schema resolved via module identifier

- **WHEN** the kernel's `*schema.Cache` is populated for the first time
- **THEN** the underlying load goes through `cue/load.Instances` against the configured module identifier (default `"opmodel.dev/core@v2"`)
- **AND** the resolved value's `LookupPath(cue.ParsePath("#ModuleInstance"))` exists

#### Scenario: Evaluated module has no apiVersion field

- **WHEN** an artifact authored against the library schema is loaded and evaluated
- **THEN** `apiVersion` on the artifact root does not exist

#### Scenario: Default resolves within the v2 major

- **WHEN** `(schema.OCILoader{}).Load(ctx)` runs with no `Module` override
- **THEN** the bare major `"opmodel.dev/core@v2"` is expanded through the loader's existing bare-major mechanism and resolves to the highest published version within the v2 major
- **AND** SemVer prerelease ordering applies, so a `v2.0.0-0.dev.*` snapshot tag never outranks a `v2.0.0-alpha.N` tag

#### Scenario: Caller-pinned earlier major still loads

- **WHEN** `(schema.OCILoader{Module: "opmodel.dev/core@v1.0.0-alpha.1"}).Load(ctx)` is called
- **THEN** the loader resolves exactly that version and returns its schema value
- **AND** no code path upgrades or rewrites the caller's pin

### Requirement: DefaultSchemaModule constant

`schema.DefaultSchemaModule` SHALL be `"opmodel.dev/core@v2"`. `OCILoader.Load` with an empty `Module` field SHALL resolve this identifier. Doc comments citing the default module identifier (`opm/kernel`, `opm/schema`, `opm/materialize`) SHALL cite the v2 identifier.

#### Scenario: Empty Module resolves the v2 default

- **WHEN** `(schema.OCILoader{Registry: "opmodel.dev=ghcr.io/open-platform-model"}).Load(ctx)` is called with `Module` unset
- **THEN** the loader resolves `Module` to `"opmodel.dev/core@v2"`, threads the env into `load.Config.Env`, and returns a non-zero `cue.Value` containing `#ModuleInstance`

#### Scenario: ResolvedVersion reports the v2 resolution

- **WHEN** `cache.Get(ctx)` succeeds against the default `opmodel.dev/core@v2` resolving to `v2.0.0-alpha.4`
- **THEN** `cache.ResolvedVersion()` returns `"v2.0.0-alpha.4"`
