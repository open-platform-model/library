## RENAMED Requirements

- FROM: `### Requirement: No Custom Validation Error Types`
- TO: `### Requirement: One validation marker and no projection of CUE errors`

## MODIFIED Requirements

### Requirement: One validation marker and no projection of CUE errors

The library SHALL define exactly one Go type for a configuration validation failure: `ConfigValidationError` in `opm/errors`, a pointer-receiver marker that wraps the CUE error tree unchanged. It SHALL carry the tree in its `Err` field, return it from `Unwrap`, and return the tree's own text from `Error`. It SHALL NOT project, group, re-order or reword the CUE errors, and the library SHALL NOT provide a walking or formatting API beside `cuelang.org/go/cue/errors`. The names `ConfigError`, `ValidationError`, `FieldError`, `ErrorLocation`, `GroupedError`, `MultiSourceError`, `LayerError`, and `DetailedError` SHALL NOT exist as exported symbols anywhere in the library.

Every error type in `opm/errors` that has an `Error` method SHALL declare it on the pointer receiver, so a caller matches each of them with `errors.As` and a pointer target. `IdentityError` is the one exception, for a transition: the library returns `*IdentityError`, and the type keeps a value receiver and a deprecated value target while a consumer at `main` still uses the value forms (the deprecate, then remove rule of AGENTS.md, Consumer build paragraph). During the transition both `errors.As` targets, `*IdentityError` and `IdentityError`, SHALL match the returned error. A later change gives the type a pointer receiver and removes the value forms.

#### Scenario: opm/errors carries no validation projections

- **WHEN** a developer reads `opm/errors/`
- **THEN** its exported identifiers are the acquisition and synthesis sentinels, the acquisition identity error (`IdentityError`), `TransformError`, the fetch and resolution classification, the render verdict rows and refusal causes (skew, routing, contract, demand, unmatched-component and core-floor), and the one validation marker `ConfigValidationError`, which wraps a CUE validation error and projects nothing
- **AND** no `ConfigError`, `ValidationError`, `FieldError`, `ErrorLocation`, or `GroupedError` types are present

#### Scenario: Frontends rely on cuelang.org/go/cue/errors

- **WHEN** a frontend wants per-position iteration over validation errors
- **THEN** it imports `cuelang.org/go/cue/errors` and uses `errors.Errors(err)` plus `errors.Positions(ce)` to walk the tree
- **AND** the library does not provide a parallel walking API

#### Scenario: A validation failure is matched by type

- **WHEN** `ValidateConfigDetailed` refuses values that do not satisfy the schema, and the caller wraps the error with `%w` any number of times
- **THEN** `errors.As` with a `*ConfigValidationError` target succeeds
- **AND** `cueerrors.Errors` on the same error returns the same CUE errors, in the same order, as on the `Err` field

#### Scenario: Every typed error matches through a pointer target

- **WHEN** a module acquired from a registry declares another path or version than the coordinate it was fetched by
- **THEN** `errors.As` with a `*IdentityError` target succeeds on the returned error
- **AND** every other type in `opm/errors` that implements `error` does so on its pointer receiver only

#### Scenario: The deprecated value target still matches

- **WHEN** the same error is matched with `errors.As` and an `IdentityError` value target, or with `errors.AsType[IdentityError]`, through a `%w` wrap
- **THEN** the match succeeds and yields the same field values as the pointer target
- **AND** the documentation of `IdentityError` marks the value forms Deprecated and names the pointer forms

### Requirement: Single Kernel Validation Primitive

The library SHALL expose exactly one validation method on `*Kernel` in `opm/kernel/`: `ValidateConfigDetailed(schema cue.Value, sources []Source) (cue.Value, error)`. It SHALL compile each source in the schema's own context so that every position names the source's `Origin` (a bytes source with `cue.Filename(Origin)`, a file-backed source through `cue/load` at its file, "File-Backed Sources Resolve Imports Through the Kernel Mapping"), unify the compiled sources in stack order, run the closed-schema disallowed-field walk, and assert concreteness on the merged value. A failure of the validation step (the disallowed-field walk, the unification with the schema, or concreteness) SHALL be returned as a `*ConfigValidationError` (`opm/errors`) that wraps the CUE-native error tree (`cuelang.org/go/cue/errors.Error` or a tree of them, accessed via `cuelang.org/go/cue/errors.Errors`) unchanged; the library SHALL NOT define a Go-typed projection over those errors. A source that fails to compile or to load is returned as before, without the marker: such a failure can be a registry or a file failure, which is not a statement about the values. The kernel SHALL NOT expose a single-value or a partial-mode variant: a single value is a one-element `[]Source`, and partial validation is an internal mode the kernel uses for per-source attribution under `AcquireInstanceFromDir` with extra values and under `SynthesizeInstance`, not a public entry.

#### Scenario: ValidateConfigDetailed signature and behavior

- **WHEN** a caller invokes `k.ValidateConfigDetailed(schema, sources)`
- **THEN** the method compiles each source in the schema's context with its `Origin` as the position filename, unifies the compiled values in stack order (`sources[0]`, then `sources[1]`, …), then validates the merged value against `schema` with concreteness enforced
- **AND** disallowed fields under closed schemas are reported with source positions via the internal `walkDisallowed` mechanism
- **AND** returns `(cue.Value, error)` with the merged value on success and the zero value on failure
- **AND** the error of a failed validation step is a `*ConfigValidationError` whose `Err` implements `cuelang.org/go/cue/errors.Error`, and the returned error is walkable via `cueerrors.Errors(err)`

#### Scenario: No single-value or partial variant is exported

- **WHEN** a consumer inspects the exported methods of `Kernel` and the exported identifiers of `opm/kernel`
- **THEN** none of `ValidateConfig`, `ValidateConfigPartial`, `ValidateOption` or `Partial` exists
- **AND** `ValidateConfigDetailed` accepts no option arguments

#### Scenario: Empty inputs short-circuit to success

- **WHEN** `ValidateConfigDetailed` receives an empty `[]Source`, a zero schema, or sources whose merged value does not exist
- **THEN** the method returns `(cue.Value{}, nil)` without performing validation
- **AND** the behavior is documented as "no values supplied"

#### Scenario: Errors carry source positions when filename was set at compile time

- **WHEN** validation of a `Source` produces an error
- **THEN** for every error at a field a Source sets (a type, constraint or disallowed-field violation), `cueerrors.Positions(ce)` includes a position whose `Filename()` equals that Source's `Origin`, because the kernel compiled that source under its `Origin`
- **AND** for a disallowed-field violation that position is the primary `Position()`
- **AND** a concreteness error for a field no source sets may carry no source position
- **AND** a disjunction's summary error (`N errors in empty disjunction`) may carry no position; each of its per-branch errors includes the Source's `Origin`
- **AND** positions contributed by the schema carry the schema's own filenames

### Requirement: Callers compose ConfigSchema with the one validation primitive

`*Module` and `*Instance` SHALL each expose a `ConfigSchema()` accessor returning the `#config` schema reachable on the artifact's `Package` (for an instance, through its embedded `#module`), or the zero `cue.Value` when absent. The Kernel SHALL NOT expose per-artifact wrappers over the validation primitive: a caller composes `ConfigSchema()` with `ValidateConfigDetailed` directly.

#### Scenario: Module.ConfigSchema accessor

- **WHEN** a caller invokes `m.ConfigSchema()`
- **THEN** the result is the `cue.Value` at `corepath.Config` inside `m.Package`
- **AND** the accessor returns a zero value if the module has no `#config` field

#### Scenario: Instance.ConfigSchema accessor

- **WHEN** a caller invokes `r.ConfigSchema()`
- **THEN** the result is the `cue.Value` at `corepath.Config` inside the instance's embedded module at `schema.Module`
- **AND** the accessor returns a zero value if the instance has no embedded module or the module has no `#config`

#### Scenario: No per-artifact validation wrapper exists

- **WHEN** a consumer inspects the exported methods of `Kernel`
- **THEN** none of `ValidateModuleValues`, `ValidateInstanceValues`, `ValidateModuleValuesDetailed`, `ValidateInstanceValuesDetailed` or `ValidateConfig` exists
- **AND** `k.ValidateConfigDetailed(m.ConfigSchema(), sources)` is the spelling for a concrete, layered check against a module, with no name wrapping

#### Scenario: No partial validation wrapper exists

- **WHEN** a consumer inspects the exported methods of `Kernel`
- **THEN** none of `ValidateModuleValuesPartial`, `ValidateInstanceValuesPartial` or `ValidateConfigPartial` exists
- **AND** no public spelling for a partial check exists; partial mode is a kernel-internal attribution pass

#### Scenario: An instance composes the same primitive

- **WHEN** a caller holds a `*module.Instance` rather than a `*module.Module`
- **THEN** it composes `r.ConfigSchema()` with `ValidateConfigDetailed`; no instance-typed wrapper exists on the Kernel
