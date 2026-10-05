## ADDED Requirements

### Requirement: Front-door docs name the four artifact types and their schema module

Wherever `README.md` lists the artifact kinds the kernel acquires or accepts, it SHALL name all four: `Module`, `ModuleInstance`, `Platform` and `Catalog`, or link to this spec. Where `README.md` or `AGENTS.md` labels the schema of the artifact types, it SHALL name the core CUE module that carries it, `opmodel.dev/core@v2`, or link to this spec. Neither SHALL label it with an API version (`v1alpha2` or any other), since the artifact roots carry no `apiVersion` field.

#### Scenario: No front-door list names three kinds

- **WHEN** a developer reads the kernel's owns list and its "does not own" list in `README.md`
- **THEN** each list that names the kinds the kernel accepts or acquires names `Module`, `ModuleInstance`, `Platform` and `Catalog`, and none names three of them as the whole set

#### Scenario: The schema is named by its module

- **WHEN** a developer reads the artifact-type tables in `README.md` and `AGENTS.md`
- **THEN** each table that labels the schema names `opmodel.dev/core@v2` (or links to this spec), and neither carries an API-version label such as `v1alpha2`
