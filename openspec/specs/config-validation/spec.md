# config-validation Specification

## Purpose

The OPM kernel's CUE-native validation surface. The library exposes three validation primitives on `*Kernel` (`ValidateConfig`, `ValidateConfigPartial`, `ValidateConfigDetailed`) plus typed Module/Release wrappers and Source loader helpers, all returning errors that implement `cuelang.org/go/cue/errors.Error`. The library does not project CUE errors into custom Go types and does not expose presentation-layer formatters; frontends own their own display by walking `cueerrors.Errors(err)` directly. This capability supersedes the prior `values-validation` capability (the `opm/helper/values/` package and its `Layer`/`Stack`/`MultiSourceError` types), collapsing per-layer-partial-then-unify into unify-then-validate while preserving per-source attribution through `cue.Filename` set at compile time.
## Requirements

### Requirement: Source Type and Layered Input

The library SHALL expose a `Source` struct in `opm/kernel/` describing one values input for `ValidateConfigDetailed` and for the extra-values option of `AcquireInstanceFromDir`. A `Source` pairs the values payload with its stable origin; it SHALL carry no display label, since presentation is outside the kernel's contract and CUE positions carry the origin.

#### Scenario: Source struct shape

- **WHEN** a frontend constructs a `kernel.Source`
- **THEN** the struct exposes exactly two fields: `Value cue.Value` (the values payload) and `Origin string` (the stable identifier CUE positions report)
- **AND** the godoc on `Source.Value` states that the value MUST have been compiled with `cue.Filename(Origin)` for per-source attribution to flow through into errors

#### Scenario: Partial option

- **WHEN** a consumer inspects the exported identifiers of `opm/kernel`
- **THEN** neither a `ValidateOption` type nor a `Partial` constructor exists
- **AND** `ValidateConfigDetailed` takes exactly two arguments, `schema` and `sources`

#### Scenario: Stack ordering for layered inputs

- **WHEN** a frontend constructs `[]Source{a, b, c}` and passes it to `ValidateConfigDetailed`
- **THEN** unification proceeds `a → a∪b → a∪b∪c`
- **AND** field conflicts resolve to the layer that wrote them last

### Requirement: Source Loader Helpers

The library SHALL expose two loader helpers on `*Kernel` that produce `Source` values with `cue.Filename` baked in: `LoadSourceFromFile` for a values file on disk and `LoadSourceFromBytes` for an in-memory payload. It SHALL NOT expose a string-input variant; a string is `[]byte(s)`.

#### Scenario: LoadSourceFromFile

- **WHEN** a caller invokes `k.LoadSourceFromFile(path string)`
- **THEN** the returned `Source` has `Origin` equal to the absolute path and `Value` is the compiled `cue.Value` carrying `cue.Filename` for that path (populated by `cue/load.Instances`)

#### Scenario: LoadSourceFromBytes

- **WHEN** a caller invokes `k.LoadSourceFromBytes(origin string, b []byte)`
- **THEN** the returned `Source` has `Origin = origin` and `Value` is `k.CueContext().CompileBytes(b, cue.Filename(origin))`
- **AND** validation errors on `Value` report `pos.Filename() == origin`

#### Scenario: LoadSourceFromString

- **WHEN** a consumer inspects the exported methods of `Kernel`
- **THEN** `LoadSourceFromString` does not exist
- **AND** a caller holding a string compiles it via `k.LoadSourceFromBytes(origin, []byte(s))`

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

The library SHALL retain `walkDisallowed` and `fieldNotAllowedError` as private internals of `opm/kernel/validate.go`. The error type SHALL implement `cuelang.org/go/cue/errors.Error` so that disallowed-field diagnostics flow alongside CUE-native errors transparently.

#### Scenario: Disallowed field in closed schema produces positioned error

- **WHEN** validation runs against a closed schema and encounters a field the schema does not declare
- **THEN** the resulting error includes a `cueerrors.Error` with `Position()` pointing to the offending field in the user's source (not the schema's closure declaration)
- **AND** the error's `Path()` returns the dotted path of the disallowed field

#### Scenario: Internal types not exported

- **WHEN** a developer searches `opm/kernel/` for `WalkDisallowed` or `FieldNotAllowedError`
- **THEN** no exported symbol with that name exists
- **AND** the unexported helpers are documented in the package's internal godoc only

### Requirement: No Custom Validation Error Types

The library SHALL NOT define custom Go-typed wrappers around CUE validation errors. The names `ConfigError`, `ValidationError`, `FieldError`, `ErrorLocation`, `GroupedError`, `MultiSourceError`, `LayerError`, and `DetailedError` SHALL NOT exist as exported symbols anywhere in the library.

#### Scenario: opm/errors carries no validation projections

- **WHEN** a developer reads `opm/errors/`
- **THEN** the package contains `TransformError` and unrelated sentinels only
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
- **AND** `k.ValidateConfigDetailed(m.ConfigSchema(), []Source{{Value: v, Origin: o}})` is the spelling for a concrete check against a module, with no name wrapping

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

The library SHALL expose exactly one validation method on `*Kernel` in `opm/kernel/`: `ValidateConfigDetailed(schema cue.Value, sources []Source) (cue.Value, error)`. It SHALL unify the sources in stack order, run the closed-schema disallowed-field walk, and assert concreteness on the merged value. It SHALL return CUE-native errors (`cuelang.org/go/cue/errors.Error` or a tree of them, accessed via `cuelang.org/go/cue/errors.Errors`); the library SHALL NOT define a Go-typed projection over those errors. The kernel SHALL NOT expose a single-value or a partial-mode variant: a single value is a one-element `[]Source`, and partial validation is an internal mode the kernel uses for per-source attribution under `AcquireInstanceFromDir` with extra values, not a public entry.

#### Scenario: ValidateConfigDetailed signature and behavior

- **WHEN** a caller invokes `k.ValidateConfigDetailed(schema, sources)`
- **THEN** the method unifies the sources in stack order (`sources[0].Value.Unify(sources[1].Value)…`), then validates the merged value against `schema` with concreteness enforced
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

- **WHEN** a Source's `Value` was compiled with `cue.Filename(Origin)` (directly or via a library loader)
- **AND** validation produces an error
- **THEN** every `cueerrors.Error` returned exposes a non-empty `Position().Filename()` matching the originating Source's `Origin`
- **AND** `cueerrors.Positions(ce)` returns primary plus contributing positions, each with a populated filename
