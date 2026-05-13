## ADDED Requirements

### Requirement: Binding exposes its loaded schema as a cue.Value

The `opm/api.Binding` interface SHALL include a method `SchemaValue(ctx *cue.Context) (cue.Value, error)` that returns the binding's schema package as a fully built `cue.Value`. Implementations SHALL load their `EmbeddedSchema() fs.FS` via `cuelang.org/go/cue/load.Instances` with the embedded filesystem as overlay, build the resulting instance with the supplied `*cue.Context`, and return the package root value.

Implementations SHALL cache the loaded `cue.Value` so repeated calls amortise the load cost. Caching SHALL be safe for the documented "one Kernel (one `*cue.Context`) per process" usage pattern; concurrent calls from multiple goroutines SHALL be safe even on the first invocation. Implementations MAY assume that all calls during the binding's lifetime use the same `*cue.Context` — passing a different context after the first call has implementation-defined behavior.

Schema-load failures (malformed embed, missing package file) SHALL be returned as wrapped errors. Callers MAY treat such errors as fatal because the embed is fixed at build time.

#### Scenario: Repeated calls return the same value

- **WHEN** `binding.SchemaValue(ctx)` is called twice with the same `*cue.Context`
- **THEN** the two returned `cue.Value`s identify the same underlying instance
- **AND** the second call does not re-execute `load.Instances`

#### Scenario: Returned value exposes #ModuleRelease

- **WHEN** `binding.SchemaValue(ctx)` is called on the v1alpha2 binding
- **AND** the returned value is queried with `LookupPath(cue.ParsePath("#ModuleRelease"))`
- **THEN** the result exists and represents the `#ModuleRelease` definition declared in `apis/core/v1alpha2/module_release.cue`

#### Scenario: No registry round-trip required

- **WHEN** `binding.SchemaValue(ctx)` is called in an environment with `CUE_REGISTRY` unset and no network access
- **THEN** the call succeeds and returns a non-zero `cue.Value`

#### Scenario: Concurrent first calls are safe

- **WHEN** two goroutines call `binding.SchemaValue(ctx)` simultaneously on a binding that has not yet loaded its schema
- **THEN** exactly one schema load runs to completion
- **AND** both goroutines receive the same `cue.Value`

#### Scenario: Schema-load failure wraps the underlying error

- **WHEN** a binding's embedded filesystem cannot be built into a `cue.Value` (test seam: inject a broken embed)
- **THEN** `binding.SchemaValue(ctx)` returns the zero `cue.Value` and a non-nil error
- **AND** subsequent calls return the same cached error rather than retrying the load
