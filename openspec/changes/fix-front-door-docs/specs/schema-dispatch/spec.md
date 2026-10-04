## ADDED Requirements

### Requirement: Front-door docs describe the default core as an exact pin

`README.md` and `AGENTS.md` SHALL describe the kernel's default schema loader as pinning an exact core release (`schema.DefaultSchemaModule`) and a bare major (`opmodel.dev/core@v2`) as opt-in. They SHALL NOT present floating-major resolution as the mechanism that absorbs additive schema changes. Where they say that an additive schema change within a major needs no Go-side bump, they SHALL say how it reaches a render: artifacts re-pin core in their own `cue.mod`, and the default moves only through a change that re-verifies the render glue.

#### Scenario: The API stability section says pinned by default

- **WHEN** a developer reads `README.md` § API stability
- **THEN** it states that the default loader pins an exact core release and that a bare major is opt-in
- **AND** it does not say that additive shape changes are absorbed by floating-major resolution

#### Scenario: The schema versioning section says pinned by default

- **WHEN** a developer reads `AGENTS.md` § OPM schema versioning
- **THEN** it names `schema.DefaultSchemaModule` as an exact-release default and the bare major as opt-in
- **AND** neither `README.md` nor `AGENTS.md` uses the word "floating"
