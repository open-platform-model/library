## ADDED Requirements

### Requirement: KernelOwner Interface Surfaces Tier-1 Validation Method

The `KernelOwner` interface in `pkg/helper/values/` SHALL expose a `ValidateConfigPartial` method matching the Kernel's signature, so the helper can perform per-layer Tier-1 validation by calling back through the interface without importing `pkg/kernel` (which would create the cycle `helper/values → kernel → helper/values`).

#### Scenario: Interface includes ValidateConfigPartial

- **WHEN** a developer reads the `KernelOwner` interface in `pkg/helper/values/values.go`
- **THEN** the interface lists at minimum two methods: `CueContext() *cue.Context` and `ValidateConfigPartial(schema cue.Value, values cue.Value, contextLabel, name string) (cue.Value, *oerrors.ConfigError)`

#### Scenario: *kernel.Kernel satisfies the interface

- **WHEN** a frontend passes a `*kernel.Kernel` where a `helper/values.KernelOwner` is expected
- **THEN** the Go compiler accepts the value because `*kernel.Kernel` exposes both required methods

#### Scenario: Helper does not import pkg/kernel

- **WHEN** a developer reads the imports of `pkg/helper/values/values.go`
- **THEN** `github.com/open-platform-model/library/pkg/kernel` is not present
- **AND** `github.com/open-platform-model/library/pkg/validate` is not present (the package no longer exists)

## MODIFIED Requirements

### Requirement: ValidateAndUnify Tier-1 Validation

`pkg/helper/values/` SHALL expose `ValidateAndUnify(owner KernelOwner, schema cue.Value, layers Stack) (cue.Value, *MultiSourceError)` that performs Tier-1 source-positioned validation per layer by invoking `owner.ValidateConfigPartial` on each layer, then unifies in order on success.

#### Scenario: All layers valid

- **WHEN** every layer in the Stack passes schema validation via `owner.ValidateConfigPartial`
- **THEN** `ValidateAndUnify` returns the unified `cue.Value` and a nil `*MultiSourceError`

#### Scenario: One layer has schema violations

- **WHEN** any layer fails validation via `owner.ValidateConfigPartial`
- **THEN** `ValidateAndUnify` returns `cue.Value{}` (zero value) and a non-nil `*MultiSourceError`
- **AND** the error contains the per-layer `*oerrors.ConfigError` instances tagged with the layer's `Name` and `Source`

#### Scenario: Multiple layers have violations

- **WHEN** two or more layers fail validation
- **THEN** the returned `*MultiSourceError` aggregates errors from every failing layer
- **AND** unification is not attempted (the unified value is the zero value)

#### Scenario: Validation routes through the interface

- **WHEN** the helper validates a layer
- **THEN** the call is `owner.ValidateConfigPartial(schema, l.Value, "values", l.Name)`
- **AND** the helper does not call any function in `pkg/validate/` (the package no longer exists)

### Requirement: Removal of validate.UnifyAndValidate

The temporary helper `validate.UnifyAndValidate` was removed in slice 05 (`introduce-tiered-validation`). With this change the entire `pkg/validate/` package SHALL be removed; no symbol previously exported from that package SHALL remain in the library.

#### Scenario: pkg/validate package no longer exists

- **WHEN** a developer searches the repository for `pkg/validate/`
- **THEN** the directory does not exist

#### Scenario: CHANGELOG documents the migration

- **WHEN** a downstream consumer reads the CHANGELOG entry for this change
- **THEN** the entry shows the migration table from `validate.Config` / `validate.ConfigPartial` to `(*Kernel).ValidateConfig` / `(*Kernel).ValidateConfigPartial`
