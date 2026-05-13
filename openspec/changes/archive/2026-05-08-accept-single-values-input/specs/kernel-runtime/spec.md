## MODIFIED Requirements

### Requirement: Single Pre-Unified Values Input

The kernel SHALL accept a single, pre-unified `cue.Value` for the values argument on every public method that takes user values. The kernel SHALL NOT accept `[]cue.Value` as a values argument on any public method.

#### Scenario: validate.Config takes a single value

- **WHEN** a caller invokes `validate.Config(schema, values, contextLabel, name)` with `values` as a `cue.Value`
- **THEN** the function validates the supplied `values` against `schema` and returns the validated value or a `*ConfigError`
- **AND** there is no internal merge loop; the function consumes `values` as-is

#### Scenario: ParseModuleRelease takes a single value

- **WHEN** a caller invokes `module.ParseModuleRelease(ctx, spec, mod, values)` with `values` as a single `cue.Value`
- **THEN** the function validates `values` via `validate.Config`, fills the validated value into `spec`, and returns a `*Release`
- **AND** the function does not accept a slice form

#### Scenario: Empty values is the zero value

- **WHEN** a caller passes a zero-value `cue.Value{}` to `validate.Config` or `module.ParseModuleRelease`
- **THEN** the call succeeds (no validation errors, no fill operation)
- **AND** the behavior matches the previous slice's "no values supplied" path

### Requirement: Tier-2 Validation Always Runs

When values are non-empty, the kernel SHALL validate them against the Module's `#config` schema regardless of whether a Tier-1 helper validated them upstream.

#### Scenario: Kernel re-validates after Tier-1

- **WHEN** a frontend that uses `opm/helper/values` (slice 05) supplies a unified value to `validate.Config`
- **THEN** the kernel performs full schema validation on the unified value
- **AND** any schema violation produces a `*ConfigError`

#### Scenario: Kernel validates without Tier-1

- **WHEN** a frontend skips Tier-1 helper validation and feeds raw unified values directly
- **THEN** the kernel still produces correct schema-validation errors via `*ConfigError`
- **AND** the only loss is per-source attribution in error messages

### Requirement: Temporary Migration Helper

The library SHALL provide `validate.UnifyAndValidate(vs []cue.Value) cue.Value` (or equivalent name) as a temporary helper that performs the previous slice-merge behavior, returning a single `cue.Value` callers can pass to the new signature. This helper SHALL be marked `// Deprecated:` from introduction and SHALL be removed when `opm/helper/values` (slice 05) lands.

#### Scenario: Migration helper exists

- **WHEN** a caller invokes `validate.UnifyAndValidate(vs)` with the same slice they previously passed to `Config`
- **THEN** the helper returns a single unified `cue.Value` ready to pass to the new `Config` signature
- **AND** the helper carries a `// Deprecated:` doc comment pointing to `opm/helper/values`

#### Scenario: Migration helper retired in slice 05

- **WHEN** slice 05 (`introduce-tiered-validation`) merges
- **THEN** the next change cycle removes `validate.UnifyAndValidate`
- **AND** consumers migrate to `opm/helper/values` for layering
