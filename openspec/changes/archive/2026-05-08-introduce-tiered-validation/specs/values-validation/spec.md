## ADDED Requirements

### Requirement: Layer and Stack Types

The library SHALL expose `Layer` and `Stack` types in `pkg/helper/values/` describing an ordered sequence of values sources.

#### Scenario: Layer carries name, source, value

- **WHEN** a frontend constructs a `Layer`
- **THEN** the struct exposes three fields: `Name string` (human-friendly), `Source string` (stable identifier for machine correlation), and `Value cue.Value` (raw values)

#### Scenario: Stack is ordered, later overrides earlier

- **WHEN** a frontend constructs `Stack{a, b, c}`
- **THEN** unification proceeds `a → a∪b → a∪b∪c`
- **AND** field conflicts resolve to the last layer that wrote them

### Requirement: ValidateAndUnify Tier-1 Validation

`pkg/helper/values/` SHALL expose `ValidateAndUnify(k *kernel.Kernel, schema cue.Value, layers Stack) (cue.Value, *MultiSourceError)` that performs Tier-1 source-positioned validation per layer, then unifies in order.

#### Scenario: All layers valid

- **WHEN** every layer in the Stack passes schema validation
- **THEN** `ValidateAndUnify` returns the unified `cue.Value` and a nil `*MultiSourceError`

#### Scenario: One layer has schema violations

- **WHEN** any layer fails schema validation
- **THEN** `ValidateAndUnify` returns `cue.Value{}` (zero value) and a non-nil `*MultiSourceError`
- **AND** the error contains the per-layer `*oerrors.ConfigError` instances tagged with the layer's `Name` and `Source`

#### Scenario: Multiple layers have violations

- **WHEN** two or more layers fail validation
- **THEN** the returned `*MultiSourceError` aggregates errors from every failing layer
- **AND** unification is not attempted (the unified value is the zero value)

### Requirement: MultiSourceError Aggregation

`MultiSourceError` SHALL expose structured access to per-layer errors so frontends can format diagnostics in their preferred shape (CLI prose, K8s status conditions, XR composition status).

#### Scenario: Errors() returns per-layer slice

- **WHEN** a frontend invokes `err.Errors()` on a `*MultiSourceError`
- **THEN** the result is a slice of structured per-layer error records, each containing the layer name, source, and underlying `*ConfigError`

#### Scenario: Error() implements stdlib `error`

- **WHEN** the `*MultiSourceError` is treated as a Go `error`
- **THEN** `Error()` returns a human-readable summary of all per-layer errors

### Requirement: Kernel Convenience Method

The `Kernel` struct SHALL expose `(k *Kernel) ValidateAndUnify(schema cue.Value, layers Stack) (cue.Value, *MultiSourceError)` delegating to `pkg/helper/values.ValidateAndUnify`.

#### Scenario: Kernel method matches helper

- **WHEN** a caller invokes `k.ValidateAndUnify(schema, layers)`
- **THEN** the result is identical to calling `helper/values.ValidateAndUnify(k, schema, layers)` directly

### Requirement: Removal of validate.UnifyAndValidate

The temporary helper `validate.UnifyAndValidate` introduced by slice 04 SHALL be removed in this slice. Frontends migrate to `pkg/helper/values.ValidateAndUnify`.

#### Scenario: validate.UnifyAndValidate no longer exists

- **WHEN** a developer searches `pkg/validate/` for `UnifyAndValidate`
- **THEN** no symbol with that name is exported

#### Scenario: CHANGELOG documents the migration

- **WHEN** a downstream consumer reads the CHANGELOG for this release
- **THEN** the entry shows the before/after for migrating from `validate.UnifyAndValidate` to `pkg/helper/values.ValidateAndUnify`
