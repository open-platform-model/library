# values-validation Specification

## Purpose

The `pkg/helper/values/` package provides tier-1 source-positioned validation and ordered unification of values layers. Frontends (CLI, controllers, Crossplane composition functions) compose values from multiple sources (defaults, release files, runtime overrides) and need to validate each source against a schema before unifying — so that diagnostics correlate to the originating source rather than to a post-unification blob. This capability defines the `Layer`/`Stack` types, the `ValidateAndUnify` entrypoint, the `MultiSourceError` aggregation type, and the kernel convenience method that delegates to it. It supersedes the temporary `validate.UnifyAndValidate` helper introduced in an earlier slice.

## Requirements

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

The temporary helper `validate.UnifyAndValidate` was removed in slice 05 (`introduce-tiered-validation`). With this change the entire `pkg/validate/` package SHALL be removed; no symbol previously exported from that package SHALL remain in the library.

#### Scenario: pkg/validate package no longer exists

- **WHEN** a developer searches the repository for `pkg/validate/`
- **THEN** the directory does not exist

#### Scenario: CHANGELOG documents the migration

- **WHEN** a downstream consumer reads the CHANGELOG entry for this change
- **THEN** the entry shows the migration table from `validate.Config` / `validate.ConfigPartial` to `(*Kernel).ValidateConfig` / `(*Kernel).ValidateConfigPartial`

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
