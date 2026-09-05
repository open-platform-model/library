## ADDED Requirements

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

## MODIFIED Requirements

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

## REMOVED Requirements

### Requirement: Three Kernel Validation Primitives

**Reason**: `ValidateConfig` and `ValidateConfigPartial` had no caller outside the library once values stopped entering through `ProcessModuleInstance`; a single value is a one-element `[]Source`, and partial mode is an internal attribution pass, not a public contract. One primitive is the whole surface (Principle VII).

**Migration**: `k.ValidateConfig(schema, v)` becomes `k.ValidateConfigDetailed(schema, []kernel.Source{{Value: v, Origin: "<origin>"}})`. A caller that needs partial validation of a draft (the cli's `module vet`) keeps its own pass until a later change designs a kernel spelling for per-source partial validation.

### Requirement: ProcessModuleInstance Wraps With Instance Name

**Reason**: Values no longer enter through `ProcessModuleInstance`: both instance acquirers unify values inside the CUE build (`AcquireInstanceFromDir` with extra values renders them into the package overlay; `SynthesizeInstance` renders them into the synthesized package), so the method's values branch was unreachable and the method is now kernel-internal.

**Migration**: Build instances through `Kernel.AcquireInstanceFromDir` (optionally with `WithValues(sources...)`) or `Kernel.SynthesizeInstance`. Validation errors from the extra-values path are still framed `instance "<name>": …`, with per-source positions from the sources' origins.
