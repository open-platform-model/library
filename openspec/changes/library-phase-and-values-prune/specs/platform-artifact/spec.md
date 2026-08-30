## MODIFIED Requirements

### Requirement: Optional Platform Field on Phase Inputs

The phase input structs (`MatchInput`, `CompileInput`) SHALL carry a required `Platform *materialize.MaterializedPlatform` field. The field is the realized platform; a raw `*platform.Platform` is not accepted, and a caller MUST `Materialize` before invoking either phase.

#### Scenario: Platform field present and optional

- **WHEN** a developer reads `MatchInput` or `CompileInput`
- **THEN** each struct has a `Platform *materialize.MaterializedPlatform` field documented as required
- **AND** invoking the phase with a nil `Platform` returns an error naming the field
