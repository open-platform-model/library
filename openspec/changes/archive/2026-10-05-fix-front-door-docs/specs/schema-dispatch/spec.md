## ADDED Requirements

### Requirement: Front-door docs describe the default core as an exact pin

Where `README.md` or `AGENTS.md` states which core the default schema loader uses, it SHALL state an exact pin (`schema.DefaultSchemaModule`) with the bare major (`opmodel.dev/core@v2`) as opt-in, or link to this spec. Neither SHALL present floating-major resolution as the mechanism that absorbs additive schema changes. Where either says that an additive schema change within a major needs no Go API change, it SHALL say how the change reaches a render: artifacts re-pin core in their own `cue.mod`, and `DefaultSchemaModule` moves separately, in the release cascade's `fix(deps)` PR, which re-verifies the render glue.

#### Scenario: The API stability section says pinned by default

- **WHEN** a developer reads `README.md` § API stability
- **THEN** where it says which core the default loader uses, it states an exact core release with a bare major opt-in, or links to this spec
- **AND** it does not say that additive shape changes are absorbed by floating-major resolution

#### Scenario: The schema versioning section says pinned by default

- **WHEN** a developer reads `AGENTS.md` § OPM schema versioning
- **THEN** where it says which core the default loader uses, it names `schema.DefaultSchemaModule` as an exact-release default and the bare major as opt-in, or links to this spec
- **AND** neither `README.md` nor `AGENTS.md` uses the word "floating"
