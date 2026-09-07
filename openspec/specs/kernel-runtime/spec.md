# kernel-runtime Specification

## Purpose
The `Kernel` struct is the public anchor type for the OPM kernel runtime. It owns the `*cue.Context` and the schema cache used by every kernel operation, so downstream consumers (CLI, operator, Crossplane function) attach to a single mental anchor instead of importing the loader / module / render / validate packages individually. All future kernel-facing slices modify this capability.
## Requirements

### Requirement: Kernel Type and Construction

The library SHALL expose a `Kernel` struct in `opm/kernel/` that serves as the single public anchor type for the OPM kernel runtime. The struct SHALL be constructible only via the `kernel.New(opts ...Option)` function. Absent `WithSchemaLoader`, the schema cache SHALL be backed by an OCI loader whose registry mapping is the kernel's `WithRegistry` value (empty when the option is absent), so the schema and every other kernel operation resolve through one mapping. An explicit `WithSchemaLoader` SHALL take precedence regardless of option order.

#### Scenario: Default construction

- **WHEN** a caller invokes `kernel.New()` with no options
- **THEN** a non-nil `*Kernel` is returned with a private `*cue.Context` constructed via `cuecontext.New()` and a schema cache backed by the default OCI loader reading the process environment
- **AND** subsequent calls to `k.CueContext()` return the same `*cue.Context` instance for the lifetime of the Kernel

#### Scenario: Registry option seeds the schema loader

- **WHEN** a caller invokes `kernel.New(WithRegistry(mapping))` with no `WithSchemaLoader`
- **THEN** the first schema-touching call resolves the core schema through `mapping`
- **AND** the process environment is not mutated

#### Scenario: Construction with options

- **WHEN** a caller invokes `kernel.New(WithSchemaLoader(myLoader), WithRegistry(mapping))` in either order
- **THEN** the returned Kernel resolves the core schema through `myLoader` and uses `mapping` for catalog and module resolution

### Requirement: cue.Context Encapsulation

The Kernel SHALL own its `*cue.Context` for the kernel's entire lifetime. The `*cue.Context` MUST NOT appear in the parameter list of any public method on `Kernel`.

#### Scenario: No leaked cue.Context in method signatures

- **WHEN** any public method is added to `Kernel` in this slice or in subsequent slices
- **THEN** the method signature does not include `*cue.Context` as a parameter
- **AND** internal operations source the context from the Kernel's private field

#### Scenario: Advanced accessor for programmatic CUE construction

- **WHEN** a caller invokes `k.CueContext()`
- **THEN** the same `*cue.Context` owned by the Kernel is returned
- **AND** the doc comment marks the accessor as advanced and documents that values built with this context are safe to pass back into Kernel methods

### Requirement: Configuration Options

The Kernel SHALL accept configuration through functional options of type `Option`. The provided options SHALL be `WithSchemaLoader` and `WithRegistry`; the Kernel SHALL NOT expose an injection slot no kernel operation reads.

#### Scenario: Adding new options preserves backward compatibility

- **WHEN** a future slice adds a new option (e.g. `WithSchemaRegistry`)
- **THEN** existing callers of `kernel.New(...)` continue to compile and run unchanged

#### Scenario: No observability slots ahead of a reader

- **WHEN** a developer inspects the `Kernel` struct and its options after this change
- **THEN** no logger, tracer or clock field or option exists
- **AND** the injection surface for the execution half is introduced by enhancement 0009 together with its first reader (revised 0009 D9)

### Requirement: Goroutine Safety Contract

A single `Kernel` SHALL NOT be used concurrently across its own method calls: the owned `*cue.Context` (used by acquisition, synthesis and validation) is driven single-threaded, and sharing one `Kernel` between goroutines can race inside CUE evaluation. Callers needing concurrent operations SHALL construct one `Kernel` per goroutine; the package documentation SHALL state this and provide a one-Kernel-per-goroutine example.

`Kernel.Render` SHALL share nothing between renders: each render is its own CUE build in a fresh `cue.Context` that is released when `Render` returns, no built value is retained by the kernel, and the Kernel's own context is not used. Concurrency is across renders, never within one; a consumer rendering from several goroutines gives each goroutine its own Kernel and calls `Render`, with no shared platform value and no mutex. The package documentation SHALL state this, SHALL NOT present any shared built value as a supported shape, and SHALL state that a render pool is sized by memory (about 61 MB plus 7.75 MB per component per concurrent render, 0019 experiment 08) rather than by core count. The retracted shared-materialized-platform model and its mutex stopgap SHALL NOT appear as supported shapes.

#### Scenario: Documentation states the contract

- **WHEN** a developer reads the godoc for the `Kernel` type
- **THEN** it states that a single `Kernel` is not safe for concurrent use across its own method calls, shows one-Kernel-per-goroutine usage, and states that `Render` shares nothing between renders

#### Scenario: Documentation retracts the shared-platform model

- **WHEN** a developer reads the godoc for the `Kernel` type
- **THEN** no shared materialized platform, held built value or mutex-serialised render appears as a supported shape; the retraction is recorded in ADR-002's supersession header and the shares-nothing rule in ADR-005

#### Scenario: Repeated renders retain nothing

- **WHEN** `Render` is invoked repeatedly on one Kernel
- **THEN** each invocation builds in a fresh context, the results are byte-identical for identical inputs, and the kernel holds no built value between calls

#### Scenario: Race detector runs on the render packages

- **WHEN** the repository test task runs
- **THEN** `opm/kernel` and `opm/internal/renderstage` are additionally run under `go test -race`

### Requirement: Canonical Implementations Live on Kernel

The canonical Go implementation of layered values validation and of instance processing (concreteness assertion and metadata decoding on a built instance spec) SHALL live on the `*Kernel` receiver in `opm/kernel/`. Instance processing SHALL be kernel-internal: it is reached only through `AcquireInstanceFromDir` and `SynthesizeInstance` and is not an exported method. No standalone `validate.Config` / `validate.ConfigPartial` / `module.ParseModuleInstance` free functions SHALL remain in the library; the `opm/validate/` and `opm/helper/values/` packages SHALL NOT exist.

#### Scenario: ValidateConfigDetailed is a kernel method

- **WHEN** a caller invokes `k.ValidateConfigDetailed(schema, sources)`
- **THEN** the method unifies the sources in order, then validates the merged value with concreteness enforced and returns the merged `cue.Value` plus a CUE-native error
- **AND** no `opm/validate/` or `opm/helper/values/` import is required by callers

#### Scenario: ValidateConfig is a kernel method

- **WHEN** a consumer inspects the exported methods of `Kernel`
- **THEN** `ValidateConfig` does not exist; `ValidateConfigDetailed` is the one validation method and runs the full Tier-2 schema validation directly

#### Scenario: ValidateConfigPartial is a kernel method

- **WHEN** a consumer inspects the exported methods of `Kernel`
- **THEN** `ValidateConfigPartial` does not exist
- **AND** the partial-validation pass (type errors, disallowed fields and pattern violations on fields that are set, missing fields not flagged) is an unexported kernel internal used by `AcquireInstanceFromDir` to attribute extra-values errors to their sources

#### Scenario: ProcessModuleInstance is a kernel method

- **WHEN** a consumer inspects the exported methods of `Kernel`
- **THEN** `ProcessModuleInstance` does not exist; instance processing is an unexported kernel internal
- **AND** both `AcquireInstanceFromDir` and `SynthesizeInstance` assert concreteness on the built spec via `spec.Validate(cue.Concrete(true))` (CUE stdlib), decode instance metadata, and return a `*module.Instance`

#### Scenario: opm/validate package is gone

- **WHEN** a developer runs `ls opm/validate/`
- **THEN** the directory does not exist

#### Scenario: opm/helper/values package is gone

- **WHEN** a developer runs `ls opm/helper/values/`
- **THEN** the directory does not exist

#### Scenario: module.ParseModuleInstance free function is gone

- **WHEN** a developer searches `opm/` for `ParseModuleInstance`
- **THEN** no free function and no method with that name exists

### Requirement: No Utility Methods on Kernel

The Kernel SHALL expose only the pipeline it runs (acquire, validate, synthesize, render) and SHALL NOT expose a finalization, constraint-stripping, raw-load or other value-utility method.

#### Scenario: No finalization method on the Kernel

- **WHEN** a consumer inspects the exported methods of `Kernel` and the exported identifiers of `opm/kernel`
- **THEN** neither `Finalize` nor `FinalizeValue` exists
- **AND** no method accepts a second, narrowed components value: the render build reads the imported instance's `components` once, for matching and execution alike

#### Scenario: No raw-load methods on the Kernel

- **WHEN** a consumer inspects the exported methods of `Kernel`
- **THEN** none of `LoadModulePackage`, `LoadPlatformPackage`, `LoadInstancePackage`, `NewModuleFromValue`, `NewPlatformFromValue` exists
- **AND** a caller that wants the raw value of an acquired artifact reads its `Package` field

### Requirement: Kernel.LoadSourceFromFile auto-unwraps the values field

The `*Kernel.LoadSourceFromFile(path string)` method SHALL load the file at `path` as a CUE instance via `load.Instances`, evaluate it against the kernel's `*cue.Context`, and:

- If the evaluated value contains a top-level `values:` field whose `Exists()` is true and `Err()` is nil, the returned `Source.Value` SHALL be that field.
- Otherwise the returned `Source.Value` SHALL be the whole evaluated value.

The method SHALL set `Source.Origin` to the absolute path of the loaded file; `Source` carries no other label. The method SHALL NOT depend on `loaderfile.LoadValuesFile` (which is removed).

#### Scenario: Values file is auto-unwrapped

- **WHEN** a caller invokes `k.LoadSourceFromFile("./values.cue")` against a file containing `values: { foo: "bar" }`
- **THEN** the returned `Source.Value` is the inner `{ foo: "bar" }` value
- **AND** `Source.Origin` is the absolute path of `values.cue`

#### Scenario: File without values field passes through

- **WHEN** a caller invokes `k.LoadSourceFromFile("./flat.cue")` against a file with no top-level `values:` field
- **THEN** the returned `Source.Value` is the whole evaluated file value
- **AND** `Source.Origin` is populated as above

### Requirement: Registry Configuration Option

The `Kernel` SHALL accept a `WithRegistry(string)` option that sets the one OCI registry mapping every kernel operation uses for catalog, module and schema resolution: the render build's catalog imports (`Render`), registry module acquisition (`AcquireModuleFromRegistry`), directory acquisition (`AcquireModuleFromDir`, `AcquirePlatformFromDir`, `AcquireInstanceFromDir`), instance synthesis (`SynthesizeInstance`) and the default schema cache. No acquire verb SHALL take a per-call registry override. Absent the option, the kernel SHALL inherit `CUE_REGISTRY` from the process environment and SHALL NOT auto-apply a built-in default registry. The option MUST NOT mutate process environment state; the mapping is plumbed into each operation's load configuration.

#### Scenario: Registry option used for resolution

- **WHEN** `kernel.New(WithRegistry("opmodel.dev=ghcr.io/open-platform-model"))` is called and `Render` runs against a platform whose `cue.mod` names a catalog under `opmodel.dev`
- **THEN** the catalog import resolves through that mapping
- **AND** the process environment is not mutated

#### Scenario: Directory acquisition uses the kernel mapping

- **WHEN** a kernel constructed with `WithRegistry(mapping)` acquires a platform or instance from a directory whose imports resolve from `opmodel.dev`
- **THEN** those imports resolve through `mapping` with no per-call argument

#### Scenario: No per-call registry parameter

- **WHEN** a consumer inspects the signatures of the acquire verbs and `SynthesizeInstance`
- **THEN** none takes a load-options or registry argument, and no `LoadOptions` type is exported from `opm/`

#### Scenario: No default applied

- **WHEN** `kernel.New()` is called with no registry option
- **THEN** the kernel inherits the process `CUE_REGISTRY`
- **AND** applies no built-in default mapping

### Requirement: Kernel.SynthesizeInstance method

The `*Kernel` type SHALL expose a method `SynthesizeInstance(ctx context.Context, in InstanceInput) (*module.Instance, error)`, where `InstanceInput` is declared in `opm/kernel` and carries `Module *module.Module` (required, source-carrying), `Name string` (required), `Namespace string` (required), `Values []Source` (optional; empty means "no values supplied"), `Labels map[string]string` and `Annotations map[string]string` (optional). It SHALL build the instance spec by single-build CUE evaluation inside the module's staged source with the unified values rendered into the package, check the values sources against the module's `#config` at their own positions after the build, assert concreteness on the built spec, decode instance metadata, and return the constructed `*module.Instance` with its `Source` populated. The schema used is the kernel's own cache; the method SHALL NOT consult any additional values source.

A missing required input SHALL fail with an error wrapping the corresponding `opm/errors` sentinel (`ErrMissingModule`, `ErrMissingName`, `ErrMissingNamespace`); a module without staged source SHALL fail wrapping `ErrMissingSource`; a module whose `Source.Pkg` is non-empty SHALL fail with an error stating that a synthesizable module is its module's root package.

#### Scenario: SynthesizeInstance produces an end-to-end instance

- **WHEN** `k.SynthesizeInstance(ctx, kernel.InstanceInput{Module: mod, Name: "demo", Namespace: "default", Values: []kernel.Source{concrete}})` is called against an acquired module
- **THEN** the returned `*module.Instance` is non-nil
- **AND** `Instance.Metadata.Name` equals `"demo"`, `Instance.Metadata.Namespace` equals `"default"`
- **AND** `Instance.Metadata.UUID` equals `uuid.SHA1(OPMNamespace, "<module.uuid>:demo:default")`
- **AND** `Instance.Source` is non-nil

#### Scenario: SynthesizeInstance rejects unconcrete result

- **WHEN** `k.SynthesizeInstance(ctx, in)` is called with empty `in.Values` against a module whose `#config` has required fields with no defaults
- **THEN** the returned error is non-nil, is framed `instance "<name>": …`, and wraps the concreteness diagnostic

#### Scenario: SynthesizeInstance attributes a values violation to its source

- **WHEN** `k.SynthesizeInstance(ctx, in)` is called with a source whose value violates the module's `#config`
- **THEN** the returned error names the violating path and its positions report the source's `Origin`

#### Scenario: SynthesizeInstance surfaces synth errors before validation

- **WHEN** `k.SynthesizeInstance(ctx, kernel.InstanceInput{Module: nil, Name: "x", Namespace: "y"})` is called
- **THEN** the returned error wraps `oerrors.ErrMissingModule` and no build runs

#### Scenario: SynthesizeInstance refuses a subpackage module

- **WHEN** `k.SynthesizeInstance` is called with a module acquired from a subdirectory of its CUE module (`Source.Pkg` non-empty)
- **THEN** the returned error states that the module must be its module's root package and no build runs

#### Scenario: SynthesizeInstance uses the Kernel's cue.Context

- **WHEN** `k.SynthesizeInstance(ctx, in)` is called
- **AND** a developer inspects the cue.Context underlying the returned `Instance.Package`
- **THEN** that context is the same instance returned by `k.CueContext()`

#### Scenario: Operator-shaped caller needs no cue.Context

- **WHEN** a frontend holds raw values bytes and their origin
- **THEN** it builds the input with `k.LoadSourceFromBytes(origin, raw)` and passes the result in `InstanceInput.Values`
- **AND** it does not call `k.CueContext()`

### Requirement: SynthesizeInstance is documented as the recommended in-memory entry point

The package documentation and the `Kernel.SynthesizeInstance` godoc SHALL state that `SynthesizeInstance` is the entry point for building an instance from typed inputs, mirroring `Kernel.AcquireInstanceFromDir` for a directory-based CUE package, and that the module it takes comes from `AcquireModuleFromRegistry` or `AcquireModuleFromDir`. The documentation SHALL NOT present a helper-level composition that reaches synthesis or instance processing directly, since neither is exported.

#### Scenario: Documentation directs callers to the kernel method

- **WHEN** a developer reads the godoc on `opm/kernel`
- **THEN** the documentation states that `Kernel.SynthesizeInstance` is the entry point for typed-input synthesis and names the two acquire verbs that produce its module
- **AND** no reference to `synth.Instance`, `opm/helper/synth` or `Kernel.LoadInstancePackage` remains

#### Scenario: SynthesizeInstance godoc points to LoadInstancePackage

- **WHEN** a developer reads the `Kernel.SynthesizeInstance` godoc
- **THEN** the directory-driven mirror it names is `Kernel.AcquireInstanceFromDir`, and the module sources it names are `AcquireModuleFromRegistry` and `AcquireModuleFromDir`
- **AND** no reference to `Kernel.LoadInstancePackage`, `Kernel.ProcessModuleInstance` or `synth.Instance` remains

### Requirement: Tier-2 validation runs where values are applied

When values are non-empty, the kernel SHALL validate them against the Module's `#config` schema at the point they are applied to the instance, regardless of whether a Tier-1 helper validated them upstream. Values are applied inside a CUE build: `AcquireInstanceFromDir` renders the unified sources into the package overlay, and `SynthesizeInstance` renders them into the synthesized package; both then check the sources against `#config` so a violation is reported at the sources' own positions, and both assert concreteness on the whole built spec. `Kernel.Render` SHALL NOT perform a second validation pass: the render build imports the instance as processed, which is already concrete.

#### Scenario: Kernel re-validates after Detailed

- **WHEN** a frontend validates sources with `k.ValidateConfigDetailed` and then supplies the same sources to `k.AcquireInstanceFromDir` or in `InstanceInput.Values`
- **THEN** the kernel validates them again where they are applied: the build unifies them with the package's `values`, and the sources are checked against `#config` at their own positions
- **AND** any schema violation produces a CUE-native error walkable via `cueerrors.Errors`, wrapped with the instance name, whose positions name the originating source's `Origin` rather than the rendered values file

#### Scenario: Kernel validates without Detailed

- **WHEN** a frontend skips `ValidateConfigDetailed` and supplies raw sources to `AcquireInstanceFromDir` or `SynthesizeInstance` directly
- **THEN** the kernel still produces correct schema-validation errors from the build, attributed to the sources' `Origin`

#### Scenario: Render does not re-validate

- **WHEN** a caller invokes `k.Render` with an instance returned by `AcquireInstanceFromDir` or `SynthesizeInstance`
- **THEN** no `#config` validation runs inside `Render`; the staged instance package is imported by the build as it was processed

### Requirement: Values enter as sources

User values SHALL reach the kernel through exactly three public inputs, each carrying values as `Source` values: `ValidateConfigDetailed(schema, sources []Source)` for layered validation; `AcquireInstanceFromDir(ctx, dir, values ...Source)`, which unifies the sources inside the package build; and `SynthesizeInstance(ctx, in)` with `in.Values []Source`, which unifies the sources and renders the result into the synthesized package. The kernel SHALL NOT accept `[]cue.Value` or a bare `cue.Value` as a values argument on any public method, and SHALL NOT expose a method that fills values into an already-built instance from Go.

#### Scenario: A single pre-unified value is a one-element source list

- **WHEN** a caller holds a single pre-unified `cue.Value` to validate or apply
- **THEN** it passes it as a one-element `[]Source` (or one variadic `Source`) to the relevant entry
- **AND** there is no internal merge loop for a single source; the value is consumed as-is

#### Scenario: ProcessModuleInstance takes a single value

- **WHEN** a consumer inspects the exported methods of `Kernel`
- **THEN** `ProcessModuleInstance` does not exist
- **AND** values reach an instance only through the variadic sources of `AcquireInstanceFromDir` or `InstanceInput.Values` on `SynthesizeInstance`

#### Scenario: ValidateConfigDetailed takes a slice of Source

- **WHEN** a caller invokes `k.ValidateConfigDetailed(schema, sources)` with `sources` as `[]Source`
- **THEN** the method unifies the sources in order then validates the merged value against `schema`

#### Scenario: Extra values enter the instance build

- **WHEN** a caller invokes `k.AcquireInstanceFromDir(ctx, dir, a, b)`
- **THEN** the sources are unified and rendered into the package overlay so the schema's own values unification performs the merge in CUE
- **AND** no Go code fills the merged value into the evaluated instance

#### Scenario: Synthesized values are sources

- **WHEN** a caller invokes `k.SynthesizeInstance(ctx, in)` with `in.Values` holding one or more `Source` values
- **THEN** the sources are unified in order, the result is rendered into the synthesized package's values source and participates in the single build
- **AND** the method does not accept a bare `cue.Value`

#### Scenario: Empty values is the zero value

- **WHEN** a caller passes an empty `[]Source` to `k.ValidateConfigDetailed`, no trailing sources to `k.AcquireInstanceFromDir`, or an empty `in.Values` to `k.SynthesizeInstance`
- **THEN** the call succeeds with no values applied (the package or synthesized spec must already be concrete on every required field)
- **AND** the behavior is documented as "no values supplied"
