## MODIFIED Requirements

### Requirement: File-Backed Sources Resolve Imports Through the Kernel Mapping
When a `Source` is file-backed (its `Origin` is an absolute path naming an existing file), the operation that compiles it SHALL load the file at its directory with the kernel's registry mapping (`WithRegistry`) applied to the load configuration, exactly as directory acquisition applies it. This SHALL hold on every path that compiles sources: `ValidateConfigDetailed`, `AcquireInstanceFromDir` with trailing values, `SynthesizeInstance`, and the kernel's internal per-source attribution pass.

#### Scenario: Values file importing a registry module

- **WHEN** a kernel constructed with `WithRegistry(mapping)` compiles a file-backed `Source` whose file imports a package served only through `mapping`
- **THEN** the import resolves and the source compiles
- **AND** the same source compiled by a kernel constructed without the option, in a process whose `CUE_REGISTRY` does not route that path, fails at the import

#### Scenario: Same mapping on every compiling path

- **WHEN** the file-backed source above is passed to `ValidateConfigDetailed`, as a trailing value to `AcquireInstanceFromDir`, and as `InstanceInput.Values` to `SynthesizeInstance`, on the kernel constructed with `WithRegistry(mapping)`
- **THEN** each path resolves the import through `mapping`
- **AND** the process `CUE_REGISTRY` is unchanged afterwards

### Requirement: Single Kernel Validation Primitive
The library SHALL expose exactly one validation method on `*Kernel` in `opm/kernel/`: `ValidateConfigDetailed(schema cue.Value, sources []Source) (cue.Value, error)`. The kernel SHALL NOT expose a single-value or a partial-mode variant: a single value is a one-element `[]Source`, and partial validation is an internal mode the kernel uses for per-source attribution under `AcquireInstanceFromDir` with extra values and under `SynthesizeInstance`, not a public entry.

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

## REMOVED Requirements

### Requirement: Source Type and Layered Input
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The Source struct is the one values input of every values-taking kernel entry", "A Source pairs CUE source bytes with its origin and carries no cue.Value and no display label" and "Each kernel entry compiles its sources with cue.Filename(Origin) in the context of the schema they meet".

### Requirement: Internal Closed-Schema Workaround
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The closed-schema walk is private to opm/kernel/validate.go and its error implements cueerrors.Error" and "The disallowed-field error Path returns one selector segment per field label and a label with dots is one segment".

### Requirement: A validation failure has one marker type and no projection
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "ConfigValidationError in opm/errors is the one Go type of a validation failure and carries the CUE error tree" and "ConfigValidationError projects nothing and the library has no parallel walking API and none of the retired error names".

### Requirement: Every typed error has a pointer receiver
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Every error type in opm/errors declares Error on its pointer receiver" and "IdentityError follows the pointer-receiver rule and a value-typed IdentityError target does not match".

## ADDED Requirements

### Requirement: The Source struct is the one values input of every values-taking kernel entry
The library SHALL expose a `Source` struct in `opm/kernel/` describing one values input for every values-taking kernel entry: `ValidateConfigDetailed`, the variadic values of `AcquireInstanceFromDir`, and `InstanceInput.Values` on `SynthesizeInstance`. No kernel entry SHALL take values in any other shape.

#### Scenario: Partial option

- **WHEN** a consumer inspects the exported identifiers of `opm/kernel`
- **THEN** neither a `ValidateOption` type nor a `Partial` constructor exists
- **AND** `ValidateConfigDetailed` takes exactly two arguments, `schema` and `sources`

#### Scenario: Stack ordering for layered inputs

- **WHEN** a frontend constructs `[]Source{a, b, c}` and passes it to `ValidateConfigDetailed`, as the trailing arguments of `AcquireInstanceFromDir`, or as `InstanceInput.Values`
- **THEN** unification proceeds `a → a∪b → a∪b∪c`
- **AND** field conflicts resolve to the layer that wrote them last

#### Scenario: One values shape everywhere

- **WHEN** a frontend has values as a file or as bytes
- **THEN** it wraps them once as `Source` values (`LoadSourceFromFile`, `LoadSourceFromBytes`, or a hand-built `Source`) and passes the same slice to validation, directory acquisition or synthesis without conversion
- **AND** it needs no `cue.Context` of its own to do so

### Requirement: A Source pairs CUE source bytes with its origin and carries no cue.Value and no display label
A `Source` pairs the values payload, as CUE source bytes, with its stable origin; it SHALL carry no `cue.Value` and no display label, since a value is bound to the context that built it, presentation is outside the kernel's contract, and CUE positions carry the origin.

#### Scenario: Source struct shape

- **WHEN** a frontend constructs a `kernel.Source`
- **THEN** the struct exposes exactly two fields: `Data []byte` (the values payload as CUE source) and `Origin string` (the stable identifier CUE positions report)
- **AND** the godoc on `Source` states that the kernel compiles `Data` with `cue.Filename(Origin)` in the context of the operation that uses it

### Requirement: Each kernel entry compiles its sources with cue.Filename(Origin) in the context of the schema they meet
Each kernel entry SHALL compile the sources it receives in the context of the schema they meet, with `cue.Filename(Origin)`, so every position in every error names the origin.

#### Scenario: Positions survive compilation at use

- **WHEN** a source's `Data` violates a module's `#config` on its third line
- **THEN** the error the kernel returns positions the violation at `Origin`, line 3

### Requirement: A file-backed source without the option reads the process CUE_REGISTRY and the process environment is not mutated
(Requirement: File-Backed Sources Resolve Imports Through the Kernel Mapping) Absent the option, the load SHALL read the process `CUE_REGISTRY` unchanged. A source compiled from bytes carries no imports and is unaffected. The process environment SHALL NOT be mutated.

#### Scenario: Absent the option the load reads CUE_REGISTRY

- **WHEN** a file-backed `Source` is compiled by a kernel constructed without the option
- **THEN** the load reads the process `CUE_REGISTRY` unchanged

#### Scenario: A source compiled from bytes is unaffected

- **WHEN** a `Source` is compiled from bytes
- **THEN** it carries no imports and is unaffected

#### Scenario: The process environment is not mutated

- **WHEN** a file-backed `Source` is compiled
- **THEN** the process environment is not mutated

### Requirement: The closed-schema walk is private to opm/kernel/validate.go and its error implements cueerrors.Error
The library SHALL retain `walkDisallowed` and `fieldNotAllowedError` as private internals of `opm/kernel/validate.go`. The error type SHALL implement `cuelang.org/go/cue/errors.Error` so that disallowed-field diagnostics flow alongside CUE-native errors transparently.

#### Scenario: Internal types not exported

- **WHEN** a developer searches `opm/kernel/` for `WalkDisallowed` or `FieldNotAllowedError`
- **THEN** no exported symbol with that name exists
- **AND** the unexported helpers are documented in the package's internal godoc only

#### Scenario: The disallowed-field error is a cueerrors.Error

- **WHEN** validation reports a disallowed field
- **THEN** the error implements `cuelang.org/go/cue/errors.Error`, so the diagnostic flows alongside CUE-native errors transparently

### Requirement: The disallowed-field error Path returns one selector segment per field label and a label with dots is one segment
`fieldNotAllowedError`: The error's `Path()` SHALL return `values` followed by one segment per field label on the way to the disallowed field, each in CUE's selector form (a definition as `#name`, a plain label bare, a label that needs quoting quoted), with a leading `#module` `#config` pair or a leading `#config` omitted. A label that contains dots SHALL be one segment, so joining the segments with `.` gives the field's dotted path and splitting is never needed to recover a label.

#### Scenario: Disallowed field in closed schema produces positioned error

- **WHEN** validation runs against a closed schema and encounters a field the schema does not declare
- **THEN** the resulting error includes a `cueerrors.Error` with `Position()` pointing to the offending field in the user's source (not the schema's closure declaration)
- **AND** the error's `Path()` returns one segment per field label of the disallowed field, which joined with `.` give its dotted path

#### Scenario: A label containing dots is one path segment

- **WHEN** validation runs against a closed schema whose `labels` struct declares no `"app.kubernetes.io/name"` field and the values set `labels: "app.kubernetes.io/name": "web"`
- **THEN** the disallowed-field error's `Path()` is `["values", "labels", "\"app.kubernetes.io/name\""]`, three segments with the dotted label as one

### Requirement: ValidateConfigDetailed compiles each source in the schema's context, unifies in stack order, walks disallowed fields and asserts concreteness
`ValidateConfigDetailed`: it SHALL compile each source in the schema's own context so that every position names the source's `Origin` (a bytes source with `cue.Filename(Origin)`, a file-backed source through `cue/load` at its file, "File-Backed Sources Resolve Imports Through the Kernel Mapping"), unify the compiled sources in stack order, run the closed-schema disallowed-field walk, and assert concreteness on the merged value.

#### Scenario: Sources are compiled in the schema's context and unified in stack order

- **WHEN** `ValidateConfigDetailed` receives sources
- **THEN** it compiles each source in the schema's own context so that every position names the source's `Origin`, unifies the compiled sources in stack order, runs the closed-schema disallowed-field walk and asserts concreteness on the merged value

### Requirement: A failure of the validation step is a ConfigValidationError that wraps the CUE error tree unchanged
`ValidateConfigDetailed`: A failure of the validation step (the disallowed-field walk, the unification with the schema, or concreteness) SHALL be returned as a `*ConfigValidationError` (`opm/errors`) that wraps the CUE-native error tree (`cuelang.org/go/cue/errors.Error` or a tree of them, accessed via `cuelang.org/go/cue/errors.Errors`) unchanged; the library SHALL NOT define a Go-typed projection over those errors.

#### Scenario: A failed validation step is a ConfigValidationError

- **WHEN** the disallowed-field walk, the unification with the schema or concreteness fails
- **THEN** `ValidateConfigDetailed` returns a `*ConfigValidationError` that wraps the CUE-native error tree unchanged

### Requirement: A source that fails to compile or to load is returned without the ConfigValidationError marker
(Requirement: A failure of the validation step is a ConfigValidationError that wraps the CUE error tree unchanged) A source that fails to compile or to load SHALL be returned as before, without the marker: such a failure can be a registry or a file failure, which is not a statement about the values.

#### Scenario: A source that fails to load is returned without the marker

- **WHEN** a source fails to compile or to load
- **THEN** the error is returned without the `*ConfigValidationError` marker

### Requirement: ConfigValidationError in opm/errors is the one Go type of a validation failure and carries the CUE error tree
The library SHALL define exactly one Go type for a configuration validation failure: `ConfigValidationError` in `opm/errors`, a pointer-receiver marker that wraps the CUE error tree unchanged. It SHALL carry the tree in its `Err` field, return it from `Unwrap`, and return the tree's own text from `Error`.

#### Scenario: A validation failure is matched by type

- **WHEN** `ValidateConfigDetailed` refuses values that do not satisfy the schema, and the caller wraps the error with `%w` any number of times
- **THEN** `errors.As` with a `*ConfigValidationError` target succeeds
- **AND** `cueerrors.Errors` on the same error returns the same CUE errors, in the same order, as on the `Err` field

### Requirement: ConfigValidationError projects nothing and the library has no parallel walking API and none of the retired error names
`ConfigValidationError`: it SHALL NOT project, group, re-order or reword the CUE errors, and the library SHALL NOT provide a walking or formatting API beside `cuelang.org/go/cue/errors`. The names `ConfigError`, `ValidationError`, `FieldError`, `ErrorLocation`, `GroupedError`, `MultiSourceError`, `LayerError`, and `DetailedError` SHALL NOT exist as exported symbols anywhere in the library.

#### Scenario: opm/errors carries no validation projections

- **WHEN** a developer reads `opm/errors/`
- **THEN** its exported identifiers are the acquisition and synthesis sentinels, the acquisition identity error (`IdentityError`), `TransformError`, the fetch and resolution classification, the render verdict rows and refusal causes (skew, routing, contract, demand, unmatched-component and core-floor), and the one validation marker `ConfigValidationError`, which wraps a CUE validation error and projects nothing
- **AND** no `ConfigError`, `ValidationError`, `FieldError`, `ErrorLocation`, or `GroupedError` types are present

#### Scenario: Frontends rely on cuelang.org/go/cue/errors

- **WHEN** a frontend wants per-position iteration over validation errors
- **THEN** it imports `cuelang.org/go/cue/errors` and uses `errors.Errors(err)` plus `errors.Positions(ce)` to walk the tree
- **AND** the library does not provide a parallel walking API

### Requirement: Every error type in opm/errors declares Error on its pointer receiver
Every error type in `opm/errors` that has an `Error` method SHALL declare it on the pointer receiver, with no exception, so a caller matches each of them with `errors.As` and a pointer target and the value type is not an error.

#### Scenario: Every typed error matches through a pointer target

- **WHEN** a module acquired from a registry declares another path or version than the coordinate it was fetched by
- **THEN** `errors.As` with a `*IdentityError` target succeeds on the returned error
- **AND** every other type in `opm/errors` that implements `error` does so on its pointer receiver only

#### Scenario: No typed error keeps a value receiver

- **WHEN** a test parses every non-test file of `opm/errors`
- **THEN** every `Error` method has a pointer receiver, and the test lists no exception
- **AND** no type of the package declares an `As` method

### Requirement: IdentityError follows the pointer-receiver rule and a value-typed IdentityError target does not match
`IdentityError` SHALL follow the rule (Requirement: Every error type in opm/errors declares Error on its pointer receiver): the library returns `*IdentityError`, the type declares `Error` on its pointer receiver, and it SHALL NOT declare an `As` method. A value-typed `IdentityError` target SHALL NOT match: `errors.AsType[IdentityError]` does not compile, because the value type does not implement `error`.

#### Scenario: The value type is not an error

- **WHEN** a test asks whether the type `IdentityError` implements `error`
- **THEN** only `*IdentityError` does

#### Scenario: IdentityError is returned as a pointer

- **WHEN** the library returns an `IdentityError`
- **THEN** it returns `*IdentityError`, the type declares `Error` on its pointer receiver, and it declares no `As` method
