## MODIFIED Requirements

### Requirement: No Custom Validation Error Types

The library SHALL NOT define custom Go-typed wrappers around CUE validation errors. The names `ConfigError`, `ValidationError`, `FieldError`, `ErrorLocation`, `GroupedError`, `MultiSourceError`, `LayerError`, and `DetailedError` SHALL NOT exist as exported symbols anywhere in the library.

#### Scenario: opm/errors carries no validation projections

- **WHEN** a developer reads `opm/errors/`
- **THEN** its exported types are the acquisition and synthesis sentinels, `TransformError`, and the render verdict rows and refusal causes, none of which projects a CUE validation error
- **AND** no `ConfigError`, `ValidationError`, `FieldError`, `ErrorLocation`, or `GroupedError` types are present

#### Scenario: Frontends rely on cuelang.org/go/cue/errors

- **WHEN** a frontend wants per-position iteration over validation errors
- **THEN** it imports `cuelang.org/go/cue/errors` and uses `errors.Errors(err)` plus `errors.Positions(ce)` to walk the tree
- **AND** the library does not provide a parallel walking API

### Requirement: Module and Instance Typed Convenience Methods

`*Module` and `*Instance` SHALL each expose a `ConfigSchema()` accessor returning the `#config` schema reachable on the artifact's `Package` (for an instance, through its embedded `#module`), or the zero `cue.Value` when absent. The Kernel SHALL NOT expose per-artifact wrappers over the validation primitive: a caller composes `ConfigSchema()` with `ValidateConfigDetailed` directly.

#### Scenario: Module.ConfigSchema accessor

- **WHEN** a caller invokes `m.ConfigSchema()`
- **THEN** the result is the `cue.Value` at `schema.Config` inside `m.Package`
- **AND** the accessor returns a zero value if the module has no `#config` field

#### Scenario: Instance.ConfigSchema accessor

- **WHEN** a caller invokes `r.ConfigSchema()`
- **THEN** the result is the `cue.Value` at `schema.Config` inside the instance's embedded module at `schema.Module`
- **AND** the accessor returns a zero value if the instance has no embedded module or the module has no `#config`

#### Scenario: Kernel.ValidateModuleValues delegates without name wrapping

- **WHEN** a consumer inspects the exported methods of `Kernel`
- **THEN** neither `ValidateModuleValues` nor `ValidateInstanceValues` exists, and neither does `ValidateConfig`
- **AND** `k.ValidateConfigDetailed(m.ConfigSchema(), []Source{{Origin: o, Data: b}})` is the spelling for a concrete check against a module, with no name wrapping

#### Scenario: Kernel.ValidateModuleValuesPartial delegates

- **WHEN** a consumer inspects the exported methods of `Kernel`
- **THEN** neither `ValidateModuleValuesPartial` nor `ValidateInstanceValuesPartial` exists, and neither does `ValidateConfigPartial`
- **AND** no public spelling for a partial check exists; partial mode is a kernel-internal attribution pass

#### Scenario: Kernel.ValidateModuleValuesDetailed delegates

- **WHEN** a consumer inspects the exported methods of `Kernel`
- **THEN** neither `ValidateModuleValuesDetailed` nor `ValidateInstanceValuesDetailed` exists
- **AND** `k.ValidateConfigDetailed(m.ConfigSchema(), sources)` is the spelling for layered validation against a module

#### Scenario: Instance equivalents

- **WHEN** a caller holds a `*module.Instance` rather than a `*module.Module`
- **THEN** it composes `r.ConfigSchema()` with the same primitive; no instance-typed wrapper exists on the Kernel

### Requirement: Single Kernel Validation Primitive

The library SHALL expose exactly one validation method on `*Kernel` in `opm/kernel/`: `ValidateConfigDetailed(schema cue.Value, sources []Source) (cue.Value, error)`. It SHALL compile each source in the schema's own context so that every position names the source's `Origin` (a bytes source with `cue.Filename(Origin)`, a file-backed source through `cue/load` at its file, "File-Backed Sources Resolve Imports Through the Kernel Mapping"), unify the compiled sources in stack order, run the closed-schema disallowed-field walk, and assert concreteness on the merged value. It SHALL return CUE-native errors (`cuelang.org/go/cue/errors.Error` or a tree of them, accessed via `cuelang.org/go/cue/errors.Errors`); the library SHALL NOT define a Go-typed projection over those errors. The kernel SHALL NOT expose a single-value or a partial-mode variant: a single value is a one-element `[]Source`, and partial validation is an internal mode the kernel uses for per-source attribution under `AcquireInstanceFromDir` with extra values, not a public entry.

#### Scenario: ValidateConfigDetailed signature and behavior

- **WHEN** a caller invokes `k.ValidateConfigDetailed(schema, sources)`
- **THEN** the method compiles each source in the schema's context with its `Origin` as the position filename, unifies the compiled values in stack order (`sources[0]`, then `sources[1]`, …), then validates the merged value against `schema` with concreteness enforced
- **AND** disallowed fields under closed schemas are reported with source positions via the internal `walkDisallowed` mechanism
- **AND** returns `(cue.Value, error)` with the merged value on success and the zero value on failure
- **AND** the error (if any) implements `cuelang.org/go/cue/errors.Error` and is walkable via `cueerrors.Errors(err)`

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
- **THEN** every `cueerrors.Error` returned exposes a non-empty `Position().Filename()` matching the originating Source's `Origin`, because the kernel compiled that source under its `Origin`
- **AND** `cueerrors.Positions(ce)` returns primary plus contributing positions, each with a populated filename
