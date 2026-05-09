## ADDED Requirements

### Requirement: Three Kernel Validation Primitives

The library SHALL expose three validation methods on `*Kernel` in `pkg/kernel/`: `ValidateConfig`, `ValidateConfigPartial`, and `ValidateConfigDetailed`. All three SHALL return CUE-native errors (`cuelang.org/go/cue/errors.Error` or a tree of them, accessed via `cuelang.org/go/cue/errors.Errors`); the library SHALL NOT define a Go-typed projection over those errors.

#### Scenario: ValidateConfig signature and behavior

- **WHEN** a caller invokes `k.ValidateConfig(schema, values cue.Value)`
- **THEN** the method returns `(cue.Value, error)` where the first value is `schema.Unify(values)` on success and the zero `cue.Value` on failure
- **AND** the method asserts concreteness via `cue.Concrete(true)` on the unified value
- **AND** disallowed fields under closed schemas are reported with source positions via the internal `walkDisallowed` mechanism
- **AND** the error (if any) implements `cuelang.org/go/cue/errors.Error` and is walkable via `cueerrors.Errors(err)`

#### Scenario: ValidateConfigPartial signature and behavior

- **WHEN** a caller invokes `k.ValidateConfigPartial(schema, values cue.Value)`
- **THEN** the method returns `(cue.Value, error)` with the unified value on success
- **AND** concreteness is NOT asserted (missing required fields are not flagged)
- **AND** type errors, constraint violations, and disallowed fields on fields that ARE set are still reported with source positions
- **AND** the error (if any) is CUE-native, walkable via `cueerrors.Errors(err)`

#### Scenario: ValidateConfigDetailed signature and behavior

- **WHEN** a caller invokes `k.ValidateConfigDetailed(schema cue.Value, sources []Source, opts ...Option)`
- **THEN** the method unifies the sources in stack order (`sources[0].Value.Unify(sources[1].Value)…`), then validates the merged value against `schema`
- **AND** without options the method behaves as `ValidateConfig` on the merged value (concrete check enforced)
- **AND** with `Partial()` in `opts` the method behaves as `ValidateConfigPartial` on the merged value (no concrete check)
- **AND** returns `(cue.Value, error)` with the merged value on success and the zero value on failure

#### Scenario: Empty inputs short-circuit to success

- **WHEN** any of the three methods receives a zero `cue.Value` (or an empty `[]Source` for Detailed)
- **THEN** the method returns `(cue.Value{}, nil)` without performing validation
- **AND** the behavior is documented as "no values supplied"

#### Scenario: Errors carry source positions when filename was set at compile time

- **WHEN** a Source's `Value` was compiled with `cue.Filename(Origin)` (directly or via a library loader)
- **AND** validation produces an error
- **THEN** every `cueerrors.Error` returned exposes a non-empty `Position().Filename()` matching the originating Source's `Origin`
- **AND** `cueerrors.Positions(ce)` returns primary plus contributing positions, each with a populated filename

### Requirement: Source Type and Layered Input

The library SHALL expose a `Source` struct in `pkg/kernel/` describing one labeled values input, plus a `ValidateOption` type with a `Partial()` constructor for `ValidateConfigDetailed`. The option type is named `ValidateOption` (not `Option`) to avoid colliding with the existing kernel-construction `Option` already exported from `pkg/kernel/kernel.go`.

#### Scenario: Source struct shape

- **WHEN** a frontend constructs a `kernel.Source`
- **THEN** the struct exposes three fields: `Value cue.Value` (the values payload), `Name string` (human-friendly label), and `Origin string` (stable identifier)
- **AND** the godoc on `Source.Value` states that the value MUST have been compiled with `cue.Filename(Origin)` for per-source attribution to flow through into errors

#### Scenario: Partial option

- **WHEN** a caller invokes `k.ValidateConfigDetailed(schema, sources, Partial())`
- **THEN** the merged value is validated without `cue.Concrete(true)`
- **AND** `walkDisallowed` still runs (disallowed-field reporting is independent of concreteness)

#### Scenario: Stack ordering for layered inputs

- **WHEN** a frontend constructs `[]Source{a, b, c}` and passes it to `ValidateConfigDetailed`
- **THEN** unification proceeds `a → a∪b → a∪b∪c`
- **AND** field conflicts resolve to the layer that wrote them last

### Requirement: Module and Release Typed Convenience Methods

`*Module` SHALL expose a `ConfigSchema()` accessor (`*Release` already exposes one). The Kernel SHALL expose typed convenience methods `ValidateModuleValues`, `ValidateModuleValuesPartial`, `ValidateModuleValuesDetailed`, `ValidateReleaseValues`, `ValidateReleaseValuesPartial`, `ValidateReleaseValuesDetailed` — each a 1-line schema-lookup wrapper that delegates to the corresponding primitive.

The convenience methods live on `*Kernel` rather than on `*Module`/`*Release` because `pkg/kernel` already imports `pkg/module`; placing methods that take a `*kernel.Kernel` on `*Module`/`*Release` would close the import cycle.

#### Scenario: Module.ConfigSchema accessor

- **WHEN** a caller invokes `m.ConfigSchema()`
- **THEN** the result is the `cue.Value` at `b.Paths().Config` inside `m.Package`, where `b` is the binding for `m.APIVersion`
- **AND** the accessor returns a zero value if the module has no `#config` field

#### Scenario: Kernel.ValidateModuleValues delegates without name wrapping

- **WHEN** a caller invokes `k.ValidateModuleValues(m, values)`
- **THEN** the result is identical to `k.ValidateConfig(m.ConfigSchema(), values)`
- **AND** the method does NOT wrap the error with the module name (caller wraps if needed; the phase method `Kernel.Validate` is the wrapping entry point)

#### Scenario: Kernel.ValidateModuleValuesPartial delegates

- **WHEN** a caller invokes `k.ValidateModuleValuesPartial(m, values)`
- **THEN** the result is identical to `k.ValidateConfigPartial(m.ConfigSchema(), values)`

#### Scenario: Kernel.ValidateModuleValuesDetailed delegates

- **WHEN** a caller invokes `k.ValidateModuleValuesDetailed(m, sources, opts...)`
- **THEN** the result is identical to `k.ValidateConfigDetailed(m.ConfigSchema(), sources, opts...)`

#### Scenario: Release equivalents

- **WHEN** a caller invokes any of `k.ValidateReleaseValues(r, values)`, `k.ValidateReleaseValuesPartial(r, values)`, or `k.ValidateReleaseValuesDetailed(r, sources, opts...)`
- **THEN** the behavior mirrors the Module equivalents, sourcing the schema from `r.ConfigSchema()` (which resolves the embedded `#module` reference at `b.Paths().Module` then `b.Paths().Config`)

### Requirement: Source Loader Helpers

The library SHALL expose three loader helpers on `*Kernel` that produce `Source` values with `cue.Filename` baked in: `LoadSourceFromFile`, `LoadSourceFromBytes`, `LoadSourceFromString`.

#### Scenario: LoadSourceFromFile

- **WHEN** a caller invokes `k.LoadSourceFromFile(path string)`
- **THEN** the returned `Source` has `Origin = path` and `Value` is the compiled `cue.Value` carrying `cue.Filename(path)` (or an equivalent populated by `cue/load.Instances`)
- **AND** `Name` defaults to the basename of `path`

#### Scenario: LoadSourceFromBytes

- **WHEN** a caller invokes `k.LoadSourceFromBytes(origin, name string, b []byte)`
- **THEN** the returned `Source` has `Origin = origin`, `Name = name`, and `Value` is `k.CueContext().CompileBytes(b, cue.Filename(origin))`
- **AND** validation errors on `Value` report `pos.Filename() == origin`

#### Scenario: LoadSourceFromString

- **WHEN** a caller invokes `k.LoadSourceFromString(origin, name, s string)`
- **THEN** the returned `Source` has `Origin = origin`, `Name = name`, and `Value` is `k.CueContext().CompileString(s, cue.Filename(origin))`

### Requirement: No Library-Defined Display Helper

The library SHALL NOT expose a print helper, formatter, or any other presentation-layer function for validation errors. Constitution principles I (Kernel Neutrality) and IV (Composability via Stable Contracts) place output formatting and presentation outside the library's contract; frontends own their own display.

#### Scenario: No PrintErrors symbol in pkg/kernel

- **WHEN** a developer searches `pkg/kernel/` for `PrintErrors`, `FormatErrors`, or any similar formatter
- **THEN** no exported symbol with that purpose exists
- **AND** the library does not import a presentation-only sink (no `io.Writer`-taking validation helper)

#### Scenario: Frontends use cueerrors.Print or roll their own

- **WHEN** a frontend wants to render validation errors
- **THEN** it calls `cuelang.org/go/cue/errors.Print` directly for raw CUE-formatted output
- **OR** it walks `cueerrors.Errors(err)` plus `cueerrors.Positions(ce)` and renders in whatever shape its consumer needs (CLI prose, K8s status conditions, XR composition status, IDE diagnostics)
- **AND** schema-internal path prefixes (`#module.#config.`, `#config.`) are stripped at the frontend if user-facing display requires it

### Requirement: Phase Method Wraps With Module Name

`Kernel.Validate(ctx, ValidateInput)` SHALL retain its public signature and SHALL internally call `Kernel.ValidateConfig` then wrap any returned error with `fmt.Errorf("module %q: %w", name, err)` where `name` is derived from `ValidateInput.ModuleRelease.Metadata.Name` (or a "<unknown>" fallback).

#### Scenario: Validate phase signature unchanged

- **WHEN** a caller invokes `k.Validate(ctx, ValidateInput{Module, ModuleRelease, Values})`
- **THEN** the method returns nil on success or a wrapped `error` on failure
- **AND** the wrapped error is walkable via `errors.As` and `cueerrors.Errors` to reach the underlying CUE diagnostics
- **AND** the textual prefix on `Error()` is `module "<name>": ` followed by the CUE error message

#### Scenario: ProcessModuleRelease uses ValidateConfig and wraps with release name

- **WHEN** `k.ProcessModuleRelease(ctx, spec, mod, values)` performs its values validation step
- **THEN** the call routes through `k.ValidateConfig(schema, values)` (no per-call options)
- **AND** any returned error is wrapped with `fmt.Errorf("release %q: %w", releaseName, err)`
- **AND** the subsequent `spec.Validate(cue.Concrete(true))` call (CUE stdlib) is unchanged

### Requirement: Internal Closed-Schema Workaround

The library SHALL retain `walkDisallowed` and `fieldNotAllowedError` as private internals of `pkg/kernel/validate.go`. The error type SHALL implement `cuelang.org/go/cue/errors.Error` so that disallowed-field diagnostics flow alongside CUE-native errors transparently.

#### Scenario: Disallowed field in closed schema produces positioned error

- **WHEN** validation runs against a closed schema and encounters a field the schema does not declare
- **THEN** the resulting error includes a `cueerrors.Error` with `Position()` pointing to the offending field in the user's source (not the schema's closure declaration)
- **AND** the error's `Path()` returns the dotted path of the disallowed field

#### Scenario: Internal types not exported

- **WHEN** a developer searches `pkg/kernel/` for `WalkDisallowed` or `FieldNotAllowedError`
- **THEN** no exported symbol with that name exists
- **AND** the unexported helpers are documented in the package's internal godoc only

### Requirement: No Custom Validation Error Types

The library SHALL NOT define custom Go-typed wrappers around CUE validation errors. The names `ConfigError`, `ValidationError`, `FieldError`, `ErrorLocation`, `GroupedError`, `MultiSourceError`, `LayerError`, and `DetailedError` SHALL NOT exist as exported symbols anywhere in the library.

#### Scenario: pkg/errors carries no validation projections

- **WHEN** a developer reads `pkg/errors/`
- **THEN** the package contains `TransformError` and unrelated sentinels only
- **AND** no `ConfigError`, `ValidationError`, `FieldError`, `ErrorLocation`, or `GroupedError` types are present

#### Scenario: Frontends rely on cuelang.org/go/cue/errors

- **WHEN** a frontend wants per-position iteration over validation errors
- **THEN** it imports `cuelang.org/go/cue/errors` and uses `errors.Errors(err)` plus `errors.Positions(ce)` to walk the tree
- **AND** the library does not provide a parallel walking API
