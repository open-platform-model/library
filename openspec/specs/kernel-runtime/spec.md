# kernel-runtime Specification

## Purpose
The `Kernel` struct is the public anchor type for the OPM kernel runtime. It owns the `*cue.Context` and the schema cache used by every kernel operation, so downstream consumers (CLI, operator, Crossplane function) attach to a single mental anchor instead of importing the loader / module / render / validate packages individually. All future kernel-facing slices modify this capability.
## Requirements

### Requirement: Kernel Type and Construction

The library SHALL expose a `Kernel` struct in `opm/kernel/` that serves as the single public anchor type for the OPM kernel runtime. The struct SHALL be constructible only via the `kernel.New(opts ...Option)` function.

#### Scenario: Default construction

- **WHEN** a caller invokes `kernel.New()` with no options
- **THEN** a non-nil `*Kernel` is returned with a private `*cue.Context` constructed via `cuecontext.New()` and a schema cache backed by the default OCI loader
- **AND** subsequent calls to `k.CueContext()` return the same `*cue.Context` instance for the lifetime of the Kernel

#### Scenario: Construction with options

- **WHEN** a caller invokes `kernel.New(WithSchemaLoader(myLoader), WithRegistry(mapping))`
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

A single `Kernel` SHALL NOT be used concurrently across its own method calls — the owned `*cue.Context` is driven single-threaded, and sharing one `Kernel` between goroutines can race inside CUE evaluation. Callers needing concurrent operations SHALL construct one `Kernel` per goroutine; the package documentation SHALL state this and provide a one-Kernel-per-goroutine example.

A `*MaterializedPlatform` SHALL NOT be rendered against concurrently from more than one goroutine, whether through one Kernel or several. `Kernel.Compile` fills each transformer's `#transform` from the platform's `Transformers` value, and a fill is a write to that value's evaluation state, not a read: rendering the real catalog concurrently against one shared platform was measured racing (enhancement 0019 experiment 06: 2321 race-detector reports, 1540 with the platform pre-evaluated). The earlier shared-read-only contract (ADR-002) is retracted. The package documentation SHALL state this, SHALL name the two supported shapes (serialize every use of a materialized platform behind one mutex, or one Kernel plus one `Materialize` per goroutine), and SHALL NOT present concurrent rendering against one shared platform as supported until the shares-nothing render model (enhancement 0019 D8) lands.

#### Scenario: Documentation states the contract

- **WHEN** a developer reads the godoc for the `Kernel` type
- **THEN** the documentation explicitly states that a single `Kernel` is not safe for concurrent use across its own method calls
- **AND** the documentation provides an example showing one-Kernel-per-goroutine usage in a multi-worker scenario

#### Scenario: Documentation retracts the shared-platform model

- **WHEN** a developer reads the godoc for the `Kernel` type
- **THEN** the documentation states that a `*MaterializedPlatform` is not safe to render against concurrently, cites the measurement, and shows the serialized form as the stopgap

#### Scenario: Race detector runs on the render packages

- **WHEN** the repository test task runs
- **THEN** `opm/kernel` and `opm/compile` are additionally run under `go test -race`

### Requirement: Backward-Compatible Method Wrappers

For every existing exported function in `opm/helper/loader/file/` and `opm/helper/platform/`, and the `*FromValue` constructors in `opm/module/` and `opm/platform/` that takes a `*cue.Context` (directly or via a `CueContextOwner` interface), the Kernel SHALL provide a method wrapper that sources `*cue.Context` from itself. The Kernel SHALL NOT wrap functions whose canonical implementation now lives on the Kernel itself (validation, layered values, module-instance processing, and the values-file source loader); those are direct kernel methods, not wrappers. The Kernel SHALL NOT expose a `ValidateAndUnify` wrapper — the canonical replacement is `Kernel.ValidateConfigDetailed`.

#### Scenario: Loader method wrapper for module packages

- **WHEN** a caller invokes `k.LoadModulePackage(ctx, "./module", loaderfile.LoadOptions{Registry: "..."})`
- **THEN** the result is identical to calling `helper/loader/file.LoadModulePackage(k.CueContext(), "./module", loaderfile.LoadOptions{Registry: "..."})`
- **AND** any error returned is the same instance the underlying free function would return

#### Scenario: Loader method wrapper for instance packages

- **WHEN** a caller invokes `k.LoadInstancePackage(ctx, "./instance", loaderfile.LoadOptions{Registry: "..."})`
- **THEN** the result is identical to calling `helper/loader/file.LoadInstancePackage(k.CueContext(), "./instance", loaderfile.LoadOptions{Registry: "..."})`
- **AND** any error returned is the same instance the underlying free function would return

#### Scenario: Helper-shaped functions remain callable

- **WHEN** existing downstream code calls `helper/loader/file.LoadModulePackage(cueCtx, dir, opts)` or `helper/loader/file.LoadInstancePackage(cueCtx, dir, opts)` directly
- **THEN** the call succeeds with the documented behavior
- **AND** the helper signatures continue to accept `*cue.Context` so non-kernel consumers can use them without importing `opm/kernel`

#### Scenario: Validation methods are not wrappers

- **WHEN** a developer reads `opm/kernel/validate.go`
- **THEN** the file contains the canonical implementation of `ValidateConfig`, `ValidateConfigPartial`, and `ValidateConfigDetailed` directly, with no `//nolint:staticcheck // SA1019:` exemptions for delegating to deleted helper packages

#### Scenario: ValidateAndUnify wrapper is gone

- **WHEN** a developer searches `opm/kernel/wrappers.go` (or the entire `opm/kernel/`) for `ValidateAndUnify`
- **THEN** no exported method or function with that name exists
- **AND** callers MUST use `k.ValidateConfigDetailed`

### Requirement: Phase-Explicit Methods on Kernel

The `Kernel` SHALL expose two phase-explicit methods, each accepting a phase-specific input struct and returning a phase-appropriate result: `Match` (pairing without execution) and `Compile` (the full pipeline). The Kernel SHALL NOT expose a plan-only verb that runs the full pipeline and discards its output, nor a values-validation phase verb; values are validated where they are applied, by `ProcessModuleInstance`.

#### Scenario: Match phase method

- **WHEN** a caller invokes `k.Match(ctx, MatchInput{ModuleInstance, Platform})`
- **THEN** the kernel produces a `*MatchPlan` describing matched and non-matched component/transformer pairs, unresolved demands and unify failures
- **AND** does not execute any transformer

#### Scenario: Compile phase method

- **WHEN** a caller invokes `k.Compile(ctx, CompileInput{ModuleInstance, Platform, RuntimeName})`
- **THEN** the kernel runs Match then Execute against the instance's already-processed value and returns a `*CompileResult` containing `Compiled []*core.Compiled`, the `MatchPlan`, component summaries, and warnings
- **AND** the `CompileResult` carries no top-level unmatched-components field: an unmatched component fails `Compile` through the typed gate, and the plan inside the result still records `MatchPlan.Unmatched`
- **AND** performs no values validation of its own; the instance is rendered as processed

#### Scenario: Validate phase method

- **WHEN** a consumer inspects the exported methods of `Kernel` and the exported identifiers of `opm/kernel`
- **THEN** neither `Validate` nor `ValidateInput` exists
- **AND** values are validated by `ProcessModuleInstance` at the point they are filled

#### Scenario: Plan phase method

- **WHEN** a consumer inspects the exported methods of `Kernel` and the exported identifiers of `opm/kernel`
- **THEN** none of `Plan`, `PlanInput`, `PlanResult` exists
- **AND** a caller wanting a dry run calls `Match` for the pairing diagnosis or `Compile` and discards `Compiled`

### Requirement: Phase Input Structs

Each phase method SHALL accept a phase-specific input struct rather than positional arguments. Input structs SHALL be additive: new fields SHALL be addable without breaking existing call sites. Phases that operate on a constructed `*module.Instance` SHALL NOT carry a parallel `*module.Module` field; the source module is reachable via the instance's `Package` at `schema.Module`. The matcher-facing input structs (`MatchInput`, `CompileInput`) SHALL carry the platform as a `*materialize.MaterializedPlatform` (the realized form), not a raw `*platform.Platform`; callers MUST `Materialize` before invoking these phases. Neither struct SHALL carry a user-values field: values enter through `ProcessModuleInstance`, and a field the pipeline validates but does not apply SHALL NOT exist.

#### Scenario: ValidateInput shape

- **WHEN** a developer searches `opm/kernel` for `ValidateInput`
- **THEN** no such struct exists; the values-validation phase verb it fed was removed

#### Scenario: MatchInput shape

- **WHEN** a developer reads the `MatchInput` struct
- **THEN** the struct has exactly `ModuleInstance *module.Instance` and `Platform *materialize.MaterializedPlatform` as required artifact fields
- **AND** the struct has no `Module` field

#### Scenario: PlanInput shape

- **WHEN** a developer searches `opm/kernel` for `PlanInput`
- **THEN** no such struct exists; the plan verb it fed was removed

#### Scenario: CompileInput shape

- **WHEN** a developer reads the `CompileInput` struct
- **THEN** the struct has `ModuleInstance *module.Instance`, `Platform *materialize.MaterializedPlatform`, and `RuntimeName string`
- **AND** the struct has no `Module`, `Provider` or `Values` field

#### Scenario: Compile sources its #config schema from the instance

- **WHEN** `kernel.Compile` runs on a `CompileInput`
- **THEN** it performs no `#config` validation; the instance's `#config` schema was applied by `ProcessModuleInstance`, reachable via `in.ModuleInstance.ConfigSchema()`
- **AND** no `*module.Module` is required on the `CompileInput`

#### Scenario: Match does not require module metadata

- **WHEN** `kernel.Match` is invoked with a `MatchInput`
- **THEN** matching consumes `in.ModuleInstance.MatchComponents()`, the instance name for diagnostics, and `in.Platform` (a `*materialize.MaterializedPlatform`) only
- **AND** the operation completes without reading any `*module.Module` field

### Requirement: Single Pre-Unified Values Input

The kernel SHALL accept a single, pre-unified `cue.Value` for the values argument on every public method that takes user values. The kernel SHALL NOT accept `[]cue.Value` as a values argument on any public method, with the sole exception of `ValidateConfigDetailed` which accepts `[]Source` for layered input.

#### Scenario: ValidateConfig takes a single value

- **WHEN** a caller invokes `k.ValidateConfig(schema, values)` with `values` as a `cue.Value`
- **THEN** the method validates the supplied `values` against `schema` and returns the validated `cue.Value` and a CUE-native `error`
- **AND** there is no internal merge loop; the method consumes `values` as-is

#### Scenario: ProcessModuleInstance takes a single value

- **WHEN** a caller invokes `k.ProcessModuleInstance(ctx, spec, mod, values)` with `values` as a single `cue.Value`
- **THEN** the method validates `values` via the kernel's own `ValidateConfig` implementation, fills the validated value into `spec`, and returns a `*module.Instance`
- **AND** the method does not accept a slice form

#### Scenario: ValidateConfigDetailed takes a slice of Source

- **WHEN** a caller invokes `k.ValidateConfigDetailed(schema, sources, opts...)` with `sources` as `[]Source`
- **THEN** the method unifies the sources in order then validates the merged value against `schema`
- **AND** this is the only public method that accepts a multi-value input

#### Scenario: Empty values is the zero value

- **WHEN** a caller passes a zero-value `cue.Value{}` to `k.ValidateConfig` / `k.ValidateConfigPartial` / `k.ProcessModuleInstance`, or an empty `[]Source` to `k.ValidateConfigDetailed`
- **THEN** the call succeeds (no validation errors, no fill operation)
- **AND** the behavior is documented as "no values supplied"

### Requirement: Tier-2 Validation Always Runs

When values are non-empty, the kernel SHALL validate them against the Module's `#config` schema at the point they are filled into the instance, regardless of whether a Tier-1 helper validated them upstream. The Tier-2 entry point is `Kernel.ValidateConfig`, invoked by `Kernel.ProcessModuleInstance`. `Kernel.Compile` SHALL NOT perform a second validation pass: it renders the instance `ProcessModuleInstance` produced, which is already concrete.

#### Scenario: Kernel re-validates after Detailed

- **WHEN** a frontend that uses `k.ValidateConfigDetailed` supplies the resulting unified value to `k.ProcessModuleInstance`
- **THEN** the kernel performs full schema validation on the unified value via `ValidateConfig` before filling it
- **AND** any schema violation produces a CUE-native error walkable via `cueerrors.Errors`, wrapped with the instance name

#### Scenario: Kernel validates without Detailed

- **WHEN** a frontend skips `ValidateConfigDetailed` and feeds raw unified values to `ProcessModuleInstance` directly
- **THEN** the kernel still produces correct schema-validation errors
- **AND** the only loss is per-source attribution in error positions (`pos.Filename()` is empty unless the caller compiled with `cue.Filename(...)` themselves)

#### Scenario: Compile does not re-validate

- **WHEN** a caller invokes `k.Compile` with an instance returned by `ProcessModuleInstance`
- **THEN** no `#config` validation runs inside `Compile`; the pipeline is Match then Execute

### Requirement: Compile Rename

The compile pipeline's terminal verb SHALL be `Compile`. The canonical entry point is `(*Kernel).Compile`. The free function `compile.CompileModuleInstance` SHALL NOT exist. The earlier `opm/render/process_module.go` / `render.ProcessModuleInstance` names SHALL NOT reappear inside `opm/compile/`. The result type is `compile.CompileResult` (aliased as `kernel.CompileResult`); no other alias of it exists.

Note: `(*Kernel).ProcessModuleInstance` names a different operation, module-instance validation, value-filling, and metadata decoding, distinct from the compile pipeline. The two names occupy different concepts and do not conflict.

#### Scenario: Canonical compile entry

- **WHEN** a caller invokes `k.Compile(ctx, in)` on a `*Kernel`
- **THEN** the call performs the full compile pipeline against `in.Platform` and returns a `*CompileResult`

#### Scenario: compile.CompileModuleInstance symbol gone

- **WHEN** a developer searches for `CompileModuleInstance` in `opm/compile/`
- **THEN** the symbol does not exist
- **AND** callers MUST use `(*Kernel).Compile`

#### Scenario: ModuleResult aliased

- **WHEN** a developer searches `opm/compile` for `ModuleResult`
- **THEN** the identifier does not exist; the result type is referenced as `CompileResult`

### Requirement: Canonical Implementations Live on Kernel

The canonical Go implementation of values validation (full, partial, and detailed) and module-instance processing SHALL live on the `*Kernel` receiver in `opm/kernel/`. No standalone `validate.Config` / `validate.ConfigPartial` / `module.ParseModuleInstance` free functions SHALL remain in the library; the `opm/validate/` and `opm/helper/values/` packages SHALL NOT exist.

#### Scenario: ValidateConfig is a kernel method

- **WHEN** a caller invokes `k.ValidateConfig(schema, values)`
- **THEN** the method runs the full Tier-2 schema validation directly and returns the validated `cue.Value` on success or a CUE-native error on failure
- **AND** no `opm/validate/` import is required by callers

#### Scenario: ValidateConfigPartial is a kernel method

- **WHEN** a caller invokes `k.ValidateConfigPartial(schema, values)`
- **THEN** the method runs the partial-validation entry point (catches type errors, disallowed fields, and pattern violations on fields that ARE set; does not flag missing fields) and returns the value on success or a CUE-native error on failure

#### Scenario: ValidateConfigDetailed is a kernel method

- **WHEN** a caller invokes `k.ValidateConfigDetailed(schema, sources, opts...)`
- **THEN** the method unifies the sources in order, then validates the merged value (full or partial depending on `Partial()` option) and returns the merged `cue.Value` plus a CUE-native error
- **AND** no `opm/helper/values/` import is required by callers

#### Scenario: ProcessModuleInstance is a kernel method

- **WHEN** a caller invokes `k.ProcessModuleInstance(ctx, spec, mod, values)`
- **THEN** the method validates `values` via the kernel's own `ValidateConfig`, fills the validated value into `spec`, asserts concreteness via `spec.Validate(cue.Concrete(true))` (CUE stdlib), decodes instance metadata, and returns a `*module.Instance`
- **AND** the method does not delegate to any deprecated free function

#### Scenario: opm/validate package is gone

- **WHEN** a developer runs `ls opm/validate/`
- **THEN** the directory does not exist

#### Scenario: opm/helper/values package is gone

- **WHEN** a developer runs `ls opm/helper/values/`
- **THEN** the directory does not exist

#### Scenario: module.ParseModuleInstance free function is gone

- **WHEN** a developer searches `opm/` for `ParseModuleInstance`
- **THEN** no free function and no method with that name exists; `(*Kernel).ProcessModuleInstance` is the only spelling

#### Scenario: compile.CompileModuleInstance free function is gone

- **WHEN** a developer searches `opm/compile/` for `CompileModuleInstance`
- **THEN** no free function with that name exists
- **AND** the canonical compile entry point is `(*Kernel).Compile`

### Requirement: No Utility Methods on Kernel

The Kernel SHALL expose only the pipeline it runs (acquire, load, process, validate, match, plan, compile) and SHALL NOT expose a finalization, constraint-stripping or other value-utility method.

#### Scenario: No finalization method on the Kernel

- **WHEN** a consumer inspects the exported methods of `Kernel` and the exported identifiers of `opm/compile`
- **THEN** neither `Finalize` nor `FinalizeValue` exists, and `compile.Module.Execute` accepts one components value

### Requirement: Kernel.LoadSourceFromFile auto-unwraps the values field

The `*Kernel.LoadSourceFromFile(path string)` method SHALL load the file at `path` as a CUE instance via `load.Instances`, evaluate it against the kernel's `*cue.Context`, and:

- If the evaluated value contains a top-level `values:` field whose `Exists()` is true and `Err()` is nil, the returned `Source.Value` SHALL be that field.
- Otherwise the returned `Source.Value` SHALL be the whole evaluated value.

The method SHALL set `Source.Origin` to the absolute path of the loaded file and `Source.Name` to its basename. The method SHALL NOT depend on `loaderfile.LoadValuesFile` (which is removed).

#### Scenario: Values file is auto-unwrapped

- **WHEN** a caller invokes `k.LoadSourceFromFile("./values.cue")` against a file containing `values: { foo: "bar" }`
- **THEN** the returned `Source.Value` is the inner `{ foo: "bar" }` value
- **AND** `Source.Origin` is the absolute path of `values.cue`
- **AND** `Source.Name` is `values.cue`

#### Scenario: File without values field passes through

- **WHEN** a caller invokes `k.LoadSourceFromFile("./flat.cue")` against a file with no top-level `values:` field
- **THEN** the returned `Source.Value` is the whole evaluated file value
- **AND** `Source.Origin` and `Source.Name` are populated as above

### Requirement: Registry Configuration Option

The `Kernel` SHALL accept a `WithRegistry(string)` option that sets the OCI registry mapping used for catalog (and schema) resolution during `Materialize`. Absent the option, the kernel SHALL inherit `CUE_REGISTRY` from the process environment and SHALL NOT auto-apply a built-in default registry. The option MUST NOT mutate process environment state; the registry mapping is plumbed into the load configuration for the operation.

#### Scenario: Registry option used for resolution

- **WHEN** `kernel.New(WithRegistry("opmodel.dev=ghcr.io/open-platform-model"))` is called
- **THEN** catalog resolution during `Materialize` uses that mapping
- **AND** the process environment is not mutated

#### Scenario: No default applied

- **WHEN** `kernel.New()` is called with no registry option
- **THEN** the kernel inherits the process `CUE_REGISTRY`
- **AND** applies no built-in default mapping

### Requirement: Materialize Method on Kernel

The `Kernel` SHALL expose `(k *Kernel) Materialize(ctx context.Context, p *platform.Platform) (*MaterializedPlatform, error)` delegating to `opm/materialize`, using the kernel's configured registry and owned `*cue.Context`. Adding this method SHALL NOT change the signatures of existing phase methods in this slice.

#### Scenario: Delegates to materialize package

- **WHEN** a caller invokes `k.Materialize(ctx, plat)`
- **THEN** it returns the `*MaterializedPlatform` produced by `opm/materialize.Materialize` using the kernel's registry and context

#### Scenario: Existing phase signatures unchanged

- **WHEN** a developer reads `Match`, `Plan`, and `Compile` after this slice
- **THEN** their signatures are unchanged and still take `*platform.Platform`

### Requirement: Compile sources its cue.Context from the caller Kernel

The compile pipeline (Match → Execute, driven by `Kernel.Compile`) SHALL build every value it constructs — the per-pair transformer `#context.*` view and the rendered output — using the **caller Kernel's** owned `*cue.Context` (the instance returned by `k.CueContext()`). It SHALL NOT derive the build context from the materialized platform (`mp.Package.Context()` / `platform.Package.Context()`). The materialized platform's `Package` is read as input (the `FillPath` argument and cross-read source), not as the owner of the build context. The pipeline does not finalize components for execution; `#component` is filled from the instance's evaluated components value directly.

#### Scenario: Compiled output builds in the Kernel's cue.Context

- **WHEN** a developer calls `k.Compile(ctx, in)` and inspects the `*cue.Context` underlying a rendered value in the returned `CompileResult.Compiled`
- **THEN** that context is the same instance returned by `k.CueContext()`

#### Scenario: Pipeline does not call Value.Context on the platform

- **WHEN** the compile pipeline builds the transformer context and executes transformers
- **THEN** it obtains its `*cue.Context` from the caller Kernel
- **AND** it does not call `Value.Context()` on the materialized platform's `Package`

#### Scenario: Behavior preserved for single-Kernel callers

- **WHEN** a single Kernel materializes a platform and then compiles an instance against it (the platform's `Package` was built in that same Kernel's `*cue.Context`)
- **THEN** the rendered output is identical to the prior platform-context-sourced behavior, because the caller context and the platform context are the same instance

### Requirement: Kernel.SynthesizeInstance method

The `*Kernel` type SHALL expose a method `SynthesizeInstance(ctx context.Context, in synth.InstanceInput) (*module.Instance, error)` that combines `synth.Instance` and `Kernel.ProcessModuleInstance` into a single call: it builds the instance spec by unifying inputs against the embedded schema, then validates the supplied values against the module's `#config`, fills the values into the spec, enforces concreteness, decodes instance metadata, and returns the constructed `*module.Instance`.

The method SHALL use the Kernel's owned `*cue.Context` when calling `synth.Instance`. The method SHALL NOT consult any additional values source — `in.Values` is passed through to `Kernel.ProcessModuleInstance` unchanged.

#### Scenario: SynthesizeInstance produces an end-to-end instance

- **WHEN** `k.SynthesizeInstance(ctx, synth.InstanceInput{Module: mod, Name: "demo", Namespace: "default", Values: concreteValues})` is called against a registered v1alpha2 module
- **THEN** the returned `*module.Instance` is non-nil
- **AND** `Instance.APIVersion` equals the module's API version
- **AND** `Instance.Metadata.Name` equals `"demo"`, `Instance.Metadata.Namespace` equals `"default"`
- **AND** `Instance.Metadata.UUID` equals `uuid.SHA1(OPMNamespace, "<module.uuid>:demo:default")`

#### Scenario: SynthesizeInstance rejects unconcrete result

- **WHEN** `k.SynthesizeInstance(ctx, in)` is called with `in.Values == cue.Value{}` against a module whose `#config` has required fields with no defaults
- **THEN** the returned error is non-nil and wraps the `Kernel.ProcessModuleInstance` concreteness diagnostic

#### Scenario: SynthesizeInstance surfaces synth errors before validation

- **WHEN** `k.SynthesizeInstance(ctx, synth.InstanceInput{Module: nil, Name: "x", Namespace: "y"})` is called
- **THEN** the returned error is non-nil and originates from `synth.Instance` (not from `Kernel.ProcessModuleInstance`)

#### Scenario: SynthesizeInstance uses the Kernel's cue.Context

- **WHEN** `k.SynthesizeInstance(ctx, in)` is called
- **AND** a developer inspects the cue.Context underlying the returned `Instance.Package`
- **THEN** that context is the same instance returned by `k.CueContext()`

### Requirement: SynthesizeInstance is documented as the recommended in-memory entry point

The package documentation and the `Kernel.SynthesizeInstance` godoc SHALL state that `SynthesizeInstance` is the recommended entry point for building an instance from typed inputs, mirroring how `Kernel.LoadInstancePackage` is the recommended entry point for building an instance from a directory-based CUE package. Callers that explicitly want the helper-level primitive MAY call `synth.Instance` followed by `Kernel.ProcessModuleInstance` directly.

#### Scenario: Documentation directs callers to the kernel method

- **WHEN** a developer reads the godoc on `opm/helper/synth/`
- **THEN** the documentation states that `Kernel.SynthesizeInstance` is the recommended entry point
- **AND** notes that direct use of `synth.Instance` is appropriate when the caller does not hold a `*Kernel`

#### Scenario: SynthesizeInstance godoc points to LoadInstancePackage

- **WHEN** a developer reads the `Kernel.SynthesizeInstance` godoc
- **THEN** the file-driven mirror it names is `Kernel.LoadInstancePackage`
- **AND** no reference to the removed `Kernel.LoadInstanceFile` remains
