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

### Requirement: No Utility Methods on Kernel

The Kernel SHALL expose only the pipeline it runs (acquire, load, process, validate, synthesize, render) and SHALL NOT expose a finalization, constraint-stripping or other value-utility method.

#### Scenario: No finalization method on the Kernel

- **WHEN** a consumer inspects the exported methods of `Kernel` and the exported identifiers of `opm/kernel`
- **THEN** neither `Finalize` nor `FinalizeValue` exists
- **AND** no method accepts a second, narrowed components value: the render build reads the imported instance's `components` once, for matching and execution alike

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

The `Kernel` SHALL accept a `WithRegistry(string)` option that sets the OCI registry mapping used for catalog and schema resolution: the render build's catalog imports (`Render`), registry module acquisition (`AcquireModuleFromRegistry`) and the schema cache. Absent the option, the kernel SHALL inherit `CUE_REGISTRY` from the process environment and SHALL NOT auto-apply a built-in default registry. The option MUST NOT mutate process environment state; the mapping is plumbed into each operation's load configuration (for a render, the staged module's `load.Config.Env`).

#### Scenario: Registry option used for resolution

- **WHEN** `kernel.New(WithRegistry("opmodel.dev=ghcr.io/open-platform-model"))` is called and `Render` runs against a platform whose `cue.mod` names a catalog under `opmodel.dev`
- **THEN** the catalog import resolves through that mapping
- **AND** the process environment is not mutated

#### Scenario: No default applied

- **WHEN** `kernel.New()` is called with no registry option
- **THEN** the kernel inherits the process `CUE_REGISTRY`
- **AND** applies no built-in default mapping

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

### Requirement: Tier-2 validation runs where values are applied

When values are non-empty, the kernel SHALL validate them against the Module's `#config` schema at the point they are filled into the instance, regardless of whether a Tier-1 helper validated them upstream. The Tier-2 entry point is `Kernel.ValidateConfig`, invoked by `Kernel.ProcessModuleInstance` (and so by `Kernel.SynthesizeInstance`). `Kernel.Render` SHALL NOT perform a second validation pass: the render build imports the instance as processed, which is already concrete.

#### Scenario: Kernel re-validates after Detailed

- **WHEN** a frontend that uses `k.ValidateConfigDetailed` supplies the resulting unified value to `k.ProcessModuleInstance`
- **THEN** the kernel performs full schema validation on the unified value via `ValidateConfig` before filling it
- **AND** any schema violation produces a CUE-native error walkable via `cueerrors.Errors`, wrapped with the instance name

#### Scenario: Kernel validates without Detailed

- **WHEN** a frontend skips `ValidateConfigDetailed` and feeds raw unified values to `ProcessModuleInstance` directly
- **THEN** the kernel still produces correct schema-validation errors
- **AND** the only loss is per-source attribution in error positions (`pos.Filename()` is empty unless the caller compiled with `cue.Filename(...)` themselves)

#### Scenario: Render does not re-validate

- **WHEN** a caller invokes `k.Render` with an instance returned by `ProcessModuleInstance` or `SynthesizeInstance`
- **THEN** no `#config` validation runs inside `Render`; the staged instance package is imported by the build as it was processed
