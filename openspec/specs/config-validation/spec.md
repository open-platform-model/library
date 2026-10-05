# config-validation Specification

## Purpose

The OPM kernel's CUE-native validation surface. The library exposes one validation primitive on `*Kernel`, `ValidateConfigDetailed`, which a caller composes with the `ConfigSchema()` accessors on `*module.Module` and `*module.Instance`, and two source loaders on `*Kernel` (`LoadSourceFromFile`, `LoadSourceFromBytes`) that produce a `Source{Origin, Data}`. Its errors implement `cuelang.org/go/cue/errors.Error`; no single-value, partial-mode or per-artifact validation wrapper is exported. The library does not project CUE errors into custom Go types and does not expose presentation-layer formatters; frontends own their own display by walking `cueerrors.Errors(err)` directly. This capability supersedes the prior `values-validation` capability (the `opm/helper/values/` package and its `Layer`/`Stack`/`MultiSourceError` types), collapsing per-layer-partial-then-unify into unify-then-validate while preserving per-source attribution through `cue.Filename` set at compile time.

## Requirements

### Requirement: Source Type and Layered Input

The library SHALL expose a `Source` struct in `opm/kernel/` describing one values input for every values-taking kernel entry: `ValidateConfigDetailed`, the variadic values of `AcquireInstanceFromDir`, and `InstanceInput.Values` on `SynthesizeInstance`. A `Source` pairs the values payload, as CUE source bytes, with its stable origin; it SHALL carry no `cue.Value` and no display label, since a value is bound to the context that built it, presentation is outside the kernel's contract, and CUE positions carry the origin. Each kernel entry SHALL compile the sources it receives in the context of the schema they meet, with `cue.Filename(Origin)`, so every position in every error names the origin. No kernel entry SHALL take values in any other shape.

#### Scenario: Source struct shape

- **WHEN** a frontend constructs a `kernel.Source`
- **THEN** the struct exposes exactly two fields: `Data []byte` (the values payload as CUE source) and `Origin string` (the stable identifier CUE positions report)
- **AND** the godoc on `Source` states that the kernel compiles `Data` with `cue.Filename(Origin)` in the context of the operation that uses it

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

#### Scenario: Positions survive compilation at use

- **WHEN** a source's `Data` violates a module's `#config` on its third line
- **THEN** the error the kernel returns positions the violation at `Origin`, line 3

### Requirement: Source Loader Helpers

The library SHALL expose two loader helpers on `*Kernel` that produce `Source` values: `LoadSourceFromFile` for a values file on disk and `LoadSourceFromBytes` for an in-memory payload. Both SHALL parse the payload for syntax and return a syntax error positioned at `Origin`; neither SHALL evaluate CUE, since evaluation happens in the context of the operation that uses the source. It SHALL NOT expose a string-input variant; a string is `[]byte(s)`.

#### Scenario: LoadSourceFromFile

- **WHEN** a caller invokes `k.LoadSourceFromFile(path string)`
- **THEN** the returned `Source` has `Origin` equal to the absolute path and `Data` equal to the file's bytes

#### Scenario: LoadSourceFromBytes

- **WHEN** a caller invokes `k.LoadSourceFromBytes(origin string, b []byte)`
- **THEN** the returned `Source` has `Origin = origin` and `Data = b`
- **AND** validation errors on the compiled source report `pos.Filename() == origin`

#### Scenario: Syntax errors fail at load

- **WHEN** either helper receives a payload that does not parse as CUE
- **THEN** it returns an error positioned at `Origin` and no `Source`

#### Scenario: LoadSourceFromString

- **WHEN** a consumer inspects the exported methods of `Kernel`
- **THEN** `LoadSourceFromString` does not exist
- **AND** a caller holding a string compiles it via `k.LoadSourceFromBytes(origin, []byte(s))`

### Requirement: File-Backed Sources Resolve Imports Through the Kernel Mapping

When a `Source` is file-backed (its `Origin` is an absolute path naming an existing file), the operation that compiles it SHALL load the file at its directory with the kernel's registry mapping (`WithRegistry`) applied to the load configuration, exactly as directory acquisition applies it. Absent the option, the load SHALL read the process `CUE_REGISTRY` unchanged. This SHALL hold on every path that compiles sources: `ValidateConfigDetailed`, `AcquireInstanceFromDir` with trailing values, `SynthesizeInstance`, and the kernel's internal per-source attribution pass. A source compiled from bytes carries no imports and is unaffected. The process environment SHALL NOT be mutated.

#### Scenario: Values file importing a registry module

- **WHEN** a kernel constructed with `WithRegistry(mapping)` compiles a file-backed `Source` whose file imports a package served only through `mapping`
- **THEN** the import resolves and the source compiles
- **AND** the same source compiled by a kernel constructed without the option, in a process whose `CUE_REGISTRY` does not route that path, fails at the import

#### Scenario: Same mapping on every compiling path

- **WHEN** the file-backed source above is passed to `ValidateConfigDetailed`, as a trailing value to `AcquireInstanceFromDir`, and as `InstanceInput.Values` to `SynthesizeInstance`, on the kernel constructed with `WithRegistry(mapping)`
- **THEN** each path resolves the import through `mapping`
- **AND** the process `CUE_REGISTRY` is unchanged afterwards

### Requirement: No Library-Defined Display Helper

The library SHALL NOT expose a print helper, formatter, or any other presentation-layer function for validation errors. Constitution principles I (Kernel Neutrality) and IV (Composability via Stable Contracts) place output formatting and presentation outside the library's contract; frontends own their own display.

#### Scenario: No PrintErrors symbol in opm/kernel

- **WHEN** a developer searches `opm/kernel/` for `PrintErrors`, `FormatErrors`, or any similar formatter
- **THEN** no exported symbol with that purpose exists
- **AND** the library does not import a presentation-only sink (no `io.Writer`-taking validation helper)

#### Scenario: Frontends use cueerrors.Print or roll their own

- **WHEN** a frontend wants to render validation errors
- **THEN** it calls `cuelang.org/go/cue/errors.Print` directly for raw CUE-formatted output
- **OR** it walks `cueerrors.Errors(err)` plus `cueerrors.Positions(ce)` and renders in whatever shape its consumer needs (CLI prose, K8s status conditions, XR composition status, IDE diagnostics)
- **AND** schema-internal path prefixes (`#module.#config.`, `#config.`) are stripped at the frontend if user-facing display requires it

### Requirement: Internal Closed-Schema Workaround

The library SHALL retain `walkDisallowed` and `fieldNotAllowedError` as private internals of `opm/kernel/validate.go`. The error type SHALL implement `cuelang.org/go/cue/errors.Error` so that disallowed-field diagnostics flow alongside CUE-native errors transparently. The error's `Path()` SHALL return `values` followed by one segment per field label on the way to the disallowed field, each in CUE's selector form (a definition as `#name`, a plain label bare, a label that needs quoting quoted), with a leading `#module` `#config` pair or a leading `#config` omitted. A label that contains dots SHALL be one segment, so joining the segments with `.` gives the field's dotted path and splitting is never needed to recover a label.

#### Scenario: Disallowed field in closed schema produces positioned error

- **WHEN** validation runs against a closed schema and encounters a field the schema does not declare
- **THEN** the resulting error includes a `cueerrors.Error` with `Position()` pointing to the offending field in the user's source (not the schema's closure declaration)
- **AND** the error's `Path()` returns one segment per field label of the disallowed field, which joined with `.` give its dotted path

#### Scenario: A label containing dots is one path segment

- **WHEN** validation runs against a closed schema whose `labels` struct declares no `"app.kubernetes.io/name"` field and the values set `labels: "app.kubernetes.io/name": "web"`
- **THEN** the disallowed-field error's `Path()` is `["values", "labels", "\"app.kubernetes.io/name\""]`, three segments with the dotted label as one

#### Scenario: Internal types not exported

- **WHEN** a developer searches `opm/kernel/` for `WalkDisallowed` or `FieldNotAllowedError`
- **THEN** no exported symbol with that name exists
- **AND** the unexported helpers are documented in the package's internal godoc only

### Requirement: No Custom Validation Error Types

The library SHALL NOT define custom Go-typed wrappers around CUE validation errors. The names `ConfigError`, `ValidationError`, `FieldError`, `ErrorLocation`, `GroupedError`, `MultiSourceError`, `LayerError`, and `DetailedError` SHALL NOT exist as exported symbols anywhere in the library.

#### Scenario: opm/errors carries no validation projections

- **WHEN** a developer reads `opm/errors/`
- **THEN** its exported identifiers are the acquisition and synthesis sentinels, the acquisition identity error (`IdentityError`), `TransformError`, and the render verdict rows and refusal causes (skew, routing, contract, demand, unmatched-component and core-floor), none of which projects a CUE validation error
- **AND** no `ConfigError`, `ValidationError`, `FieldError`, `ErrorLocation`, or `GroupedError` types are present

#### Scenario: Frontends rely on cuelang.org/go/cue/errors

- **WHEN** a frontend wants per-position iteration over validation errors
- **THEN** it imports `cuelang.org/go/cue/errors` and uses `errors.Errors(err)` plus `errors.Positions(ce)` to walk the tree
- **AND** the library does not provide a parallel walking API

### Requirement: Single Kernel Validation Primitive

The library SHALL expose exactly one validation method on `*Kernel` in `opm/kernel/`: `ValidateConfigDetailed(schema cue.Value, sources []Source) (cue.Value, error)`. It SHALL compile each source in the schema's own context so that every position names the source's `Origin` (a bytes source with `cue.Filename(Origin)`, a file-backed source through `cue/load` at its file, "File-Backed Sources Resolve Imports Through the Kernel Mapping"), unify the compiled sources in stack order, run the closed-schema disallowed-field walk, and assert concreteness on the merged value. It SHALL return CUE-native errors (`cuelang.org/go/cue/errors.Error` or a tree of them, accessed via `cuelang.org/go/cue/errors.Errors`); the library SHALL NOT define a Go-typed projection over those errors. The kernel SHALL NOT expose a single-value or a partial-mode variant: a single value is a one-element `[]Source`, and partial validation is an internal mode the kernel uses for per-source attribution under `AcquireInstanceFromDir` with extra values and under `SynthesizeInstance`, not a public entry.

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
- **THEN** for every error at a field a Source sets (a type, constraint or disallowed-field violation), `cueerrors.Positions(ce)` includes a position whose `Filename()` equals that Source's `Origin`, because the kernel compiled that source under its `Origin`
- **AND** for a disallowed-field violation that position is the primary `Position()`
- **AND** a concreteness error for a field no source sets may carry no source position
- **AND** a disjunction's summary error (`N errors in empty disjunction`) may carry no position; each of its per-branch errors includes the Source's `Origin`
- **AND** positions contributed by the schema carry the schema's own filenames

### Requirement: Callers compose ConfigSchema with the one validation primitive

`*Module` and `*Instance` SHALL each expose a `ConfigSchema()` accessor returning the `#config` schema reachable on the artifact's `Package` (for an instance, through its embedded `#module`), or the zero `cue.Value` when absent. The Kernel SHALL NOT expose per-artifact wrappers over the validation primitive: a caller composes `ConfigSchema()` with `ValidateConfigDetailed` directly.

#### Scenario: Module.ConfigSchema accessor

- **WHEN** a caller invokes `m.ConfigSchema()`
- **THEN** the result is the `cue.Value` at `schema.Config` inside `m.Package`
- **AND** the accessor returns a zero value if the module has no `#config` field

#### Scenario: Instance.ConfigSchema accessor

- **WHEN** a caller invokes `r.ConfigSchema()`
- **THEN** the result is the `cue.Value` at `schema.Config` inside the instance's embedded module at `schema.Module`
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
