## MODIFIED Requirements

### Requirement: Phase Input Structs

Each phase method SHALL accept a phase-specific input struct rather than positional arguments. Input structs SHALL be additive — new fields SHALL be addable without breaking existing call sites. Phases that operate on a constructed `*module.Release` SHALL NOT carry a parallel `*module.Module` field; the source module is reachable via the release's `Package` at the binding's `Paths().Module`. The matcher-facing input structs (`MatchInput`, `PlanInput`, `CompileInput`) SHALL carry the platform as a `*materialize.MaterializedPlatform` (the realized form), not a raw `*platform.Platform`; callers MUST `Materialize` before invoking these phases.

#### Scenario: ValidateInput shape

- **WHEN** a developer reads the `ValidateInput` struct
- **THEN** the struct has at minimum `Module *module.Module`, `ModuleRelease *module.Release`, and `Values cue.Value`
- **AND** each field has godoc explaining its role

#### Scenario: MatchInput shape

- **WHEN** a developer reads the `MatchInput` struct
- **THEN** the struct has exactly `ModuleRelease *module.Release` and `Platform *materialize.MaterializedPlatform` as required artifact fields
- **AND** the struct has no `Module` field

#### Scenario: PlanInput shape

- **WHEN** a developer reads the `PlanInput` struct
- **THEN** the struct has `ModuleRelease *module.Release`, `Values cue.Value`, `Platform *materialize.MaterializedPlatform`, and `RuntimeName string`
- **AND** the struct has no `Module` field

#### Scenario: CompileInput shape

- **WHEN** a developer reads the `CompileInput` struct
- **THEN** the struct has `ModuleRelease *module.Release`, `Values cue.Value`, `Platform *materialize.MaterializedPlatform`, and `RuntimeName string`
- **AND** the struct has no `Module` field
- **AND** the struct has no `Provider` field

#### Scenario: Compile sources its #config schema from the release

- **WHEN** `kernel.Compile` runs its embedded Tier-2 validation step on a `CompileInput`
- **THEN** the `#config` schema is obtained from `in.ModuleRelease.Package` via the binding's `Paths().Module` + `Paths().Config`
- **AND** no `*module.Module` is required on the `CompileInput`

#### Scenario: Match does not require module metadata

- **WHEN** `kernel.Match` is invoked with a `MatchInput`
- **THEN** matching consumes `in.ModuleRelease.MatchComponents()`, the release name for diagnostics, and `in.Platform` (a `*materialize.MaterializedPlatform`) only
- **AND** the operation completes without reading any `*module.Module` field
