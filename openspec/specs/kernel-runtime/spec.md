# kernel-runtime Specification

## Purpose
The `Kernel` struct is the public anchor type for the OPM kernel runtime. It owns the `*cue.Context` and the cross-cutting dependencies (logger, tracer, clock) used by every kernel operation, so downstream consumers (CLI, operator, Crossplane function) attach to a single mental anchor instead of importing the loader / module / render / validate packages individually. All future kernel-facing slices modify this capability.

## Requirements

### Requirement: Kernel Type and Construction

The library SHALL expose a `Kernel` struct in `pkg/kernel/` that serves as the single public anchor type for the OPM kernel runtime. The struct SHALL be constructible only via the `kernel.New(opts ...Option)` function.

#### Scenario: Default construction

- **WHEN** a caller invokes `kernel.New()` with no options
- **THEN** a non-nil `*Kernel` is returned with a private `*cue.Context` constructed via `cuecontext.New()`, a no-op logger, and a real-time clock
- **AND** subsequent calls to `k.CueContext()` return the same `*cue.Context` instance for the lifetime of the Kernel

#### Scenario: Construction with options

- **WHEN** a caller invokes `kernel.New(WithLogger(myLogger), WithClock(myClock))`
- **THEN** the returned Kernel uses `myLogger` for all internal logging and `myClock` for any time-dependent operations

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

### Requirement: Functional Options Pattern

The Kernel SHALL accept dependency-injection configuration through functional options of type `Option`. The slice SHALL provide at minimum `WithLogger`, `WithTracer`, and `WithClock` options.

#### Scenario: WithLogger replaces the default logger

- **WHEN** `kernel.New(WithLogger(custom))` is called
- **THEN** all kernel-internal logging routes through `custom`
- **AND** the no-op default is not used

#### Scenario: Adding new options preserves backward compatibility

- **WHEN** a future slice adds a new option (e.g. `WithSchemaRegistry`)
- **THEN** existing callers of `kernel.New(...)` continue to compile and run unchanged

### Requirement: Goroutine Safety Contract

The Kernel SHALL be documented as not goroutine-safe across method calls. The package documentation SHALL state that callers needing concurrent operations construct one Kernel per goroutine.

#### Scenario: Documentation states the contract

- **WHEN** a developer reads the godoc for the `Kernel` type
- **THEN** the documentation explicitly states that the type is not safe for concurrent use across method calls
- **AND** the documentation provides an example showing one-Kernel-per-goroutine usage in a multi-worker scenario

### Requirement: Backward-Compatible Method Wrappers

For every existing exported function in `pkg/helper/loader/file/` and `pkg/helper/platform/`, and the `*FromValue` constructors in `pkg/module/` and `pkg/platform/` that takes a `*cue.Context` (directly or via a `CueContextOwner` interface), the Kernel SHALL provide a method wrapper that sources `*cue.Context` from itself. The Kernel SHALL NOT wrap functions whose canonical implementation now lives on the Kernel itself (validation, layered values, module-release processing, and the values-file source loader); those are direct kernel methods, not wrappers. The Kernel SHALL NOT expose a `ValidateAndUnify` wrapper — the canonical replacement is `Kernel.ValidateConfigDetailed`.

#### Scenario: Loader method wrapper for module packages

- **WHEN** a caller invokes `k.LoadModulePackage(ctx, "./module", loaderfile.LoadOptions{Registry: "..."})`
- **THEN** the result is identical to calling `helper/loader/file.LoadModulePackage(k.CueContext(), "./module", loaderfile.LoadOptions{Registry: "..."})`
- **AND** any error returned is the same instance the underlying free function would return

#### Scenario: Loader method wrapper for release packages

- **WHEN** a caller invokes `k.LoadReleasePackage(ctx, "./release", loaderfile.LoadOptions{Registry: "..."})`
- **THEN** the result is identical to calling `helper/loader/file.LoadReleasePackage(k.CueContext(), "./release", loaderfile.LoadOptions{Registry: "..."})`
- **AND** any error returned is the same instance the underlying free function would return

#### Scenario: Helper-shaped functions remain callable

- **WHEN** existing downstream code calls `helper/loader/file.LoadModulePackage(cueCtx, dir, opts)` or `helper/loader/file.LoadReleasePackage(cueCtx, dir, opts)` directly
- **THEN** the call succeeds with the documented behavior
- **AND** the helper signatures continue to accept `*cue.Context` so non-kernel consumers can use them without importing `pkg/kernel`

#### Scenario: Validation methods are not wrappers

- **WHEN** a developer reads `pkg/kernel/validate.go`
- **THEN** the file contains the canonical implementation of `ValidateConfig`, `ValidateConfigPartial`, and `ValidateConfigDetailed` directly, with no `//nolint:staticcheck // SA1019:` exemptions for delegating to deleted helper packages

#### Scenario: ValidateAndUnify wrapper is gone

- **WHEN** a developer searches `pkg/kernel/wrappers.go` (or the entire `pkg/kernel/`) for `ValidateAndUnify`
- **THEN** no exported method or function with that name exists
- **AND** callers MUST use `k.ValidateConfigDetailed`

### Requirement: Phase-Explicit Methods on Kernel

The `Kernel` SHALL expose four phase-explicit methods, each accepting a phase-specific input struct and returning a phase-appropriate result.

#### Scenario: Validate phase method

- **WHEN** a caller invokes `k.Validate(ctx, ValidateInput{Module, ModuleRelease, Values})`
- **THEN** the kernel performs Tier-2 schema validation of `Values` against `Module.Package`'s `#config` by calling `k.ValidateConfig` internally
- **AND** returns nil on success or a CUE-native error wrapped with `fmt.Errorf("module %q: %w", name, err)` on failure
- **AND** does not perform matching, execution, or finalization

#### Scenario: Match phase method

- **WHEN** a caller invokes `k.Match(ctx, MatchInput{Module, ModuleRelease, Platform})`
- **THEN** the kernel produces a `*MatchPlan` describing matched and non-matched component/transformer pairs
- **AND** does not execute any transformer

#### Scenario: Plan phase method

- **WHEN** a caller invokes `k.Plan(ctx, PlanInput{Module, ModuleRelease, Values, Platform, RuntimeName})`
- **THEN** the kernel runs the full Compile pipeline (Validate + Match + Execute + Finalize) and returns a `*PlanResult` containing component summaries, unmatched FQNs, ambiguous FQNs, and warnings
- **AND** does not return rendered values

#### Scenario: Compile phase method

- **WHEN** a caller invokes `k.Compile(ctx, CompileInput{Module, ModuleRelease, Values, Platform, RuntimeName})`
- **THEN** the kernel runs the full pipeline (Validate + Match + Execute + Finalize) and returns a `*CompileResult` containing `Compiled []*core.Compiled`, component summaries, unmatched FQNs, ambiguous FQNs, and warnings

### Requirement: Phase Input Structs

Each phase method SHALL accept a phase-specific input struct rather than positional arguments. Input structs SHALL be additive — new fields SHALL be addable without breaking existing call sites. Phases that operate on a constructed `*module.Release` SHALL NOT carry a parallel `*module.Module` field; the source module is reachable via the release's `Package` at the binding's `Paths().Module`.

#### Scenario: ValidateInput shape

- **WHEN** a developer reads the `ValidateInput` struct
- **THEN** the struct has at minimum `Module *module.Module`, `ModuleRelease *module.Release`, and `Values cue.Value`
- **AND** each field has godoc explaining its role

#### Scenario: MatchInput shape

- **WHEN** a developer reads the `MatchInput` struct
- **THEN** the struct has exactly `ModuleRelease *module.Release` and `Platform *platform.Platform` as required artifact fields
- **AND** the struct has no `Module` field

#### Scenario: PlanInput shape

- **WHEN** a developer reads the `PlanInput` struct
- **THEN** the struct has `ModuleRelease *module.Release`, `Values cue.Value`, `Platform *platform.Platform`, and `RuntimeName string`
- **AND** the struct has no `Module` field

#### Scenario: CompileInput shape

- **WHEN** a developer reads the `CompileInput` struct
- **THEN** the struct has `ModuleRelease *module.Release`, `Values cue.Value`, `Platform *platform.Platform`, and `RuntimeName string`
- **AND** the struct has no `Module` field
- **AND** the struct has no `Provider` field

#### Scenario: Compile sources its #config schema from the release

- **WHEN** `kernel.Compile` runs its embedded Tier-2 validation step on a `CompileInput`
- **THEN** the `#config` schema is obtained from `in.ModuleRelease.Package` via the binding's `Paths().Module` + `Paths().Config`
- **AND** no `*module.Module` is required on the `CompileInput`

#### Scenario: Match does not require module metadata

- **WHEN** `kernel.Match` is invoked with a `MatchInput`
- **THEN** matching consumes `in.ModuleRelease.MatchComponents()` and `in.Platform` only
- **AND** the operation completes without reading any `*module.Module` field

### Requirement: Single Pre-Unified Values Input

The kernel SHALL accept a single, pre-unified `cue.Value` for the values argument on every public method that takes user values. The kernel SHALL NOT accept `[]cue.Value` as a values argument on any public method, with the sole exception of `ValidateConfigDetailed` which accepts `[]Source` for layered input.

#### Scenario: ValidateConfig takes a single value

- **WHEN** a caller invokes `k.ValidateConfig(schema, values)` with `values` as a `cue.Value`
- **THEN** the method validates the supplied `values` against `schema` and returns the validated `cue.Value` and a CUE-native `error`
- **AND** there is no internal merge loop; the method consumes `values` as-is

#### Scenario: ProcessModuleRelease takes a single value

- **WHEN** a caller invokes `k.ProcessModuleRelease(ctx, spec, mod, values)` with `values` as a single `cue.Value`
- **THEN** the method validates `values` via the kernel's own `ValidateConfig` implementation, fills the validated value into `spec`, and returns a `*module.Release`
- **AND** the method does not accept a slice form

#### Scenario: ValidateConfigDetailed takes a slice of Source

- **WHEN** a caller invokes `k.ValidateConfigDetailed(schema, sources, opts...)` with `sources` as `[]Source`
- **THEN** the method unifies the sources in order then validates the merged value against `schema`
- **AND** this is the only public method that accepts a multi-value input

#### Scenario: Empty values is the zero value

- **WHEN** a caller passes a zero-value `cue.Value{}` to `k.ValidateConfig` / `k.ValidateConfigPartial` / `k.ProcessModuleRelease`, or an empty `[]Source` to `k.ValidateConfigDetailed`
- **THEN** the call succeeds (no validation errors, no fill operation)
- **AND** the behavior is documented as "no values supplied"

### Requirement: Tier-2 Validation Always Runs

When values are non-empty, the kernel SHALL validate them against the Module's `#config` schema regardless of whether a Tier-1 helper validated them upstream. The Tier-2 entry point is `Kernel.ValidateConfig`.

#### Scenario: Kernel re-validates after Detailed

- **WHEN** a frontend that uses `k.ValidateConfigDetailed` supplies the resulting unified value to `k.ValidateConfig`
- **THEN** the kernel performs full schema validation on the unified value
- **AND** any schema violation produces a CUE-native error walkable via `cueerrors.Errors`

#### Scenario: Kernel validates without Detailed

- **WHEN** a frontend skips `ValidateConfigDetailed` and feeds raw unified values directly
- **THEN** the kernel still produces correct schema-validation errors
- **AND** the only loss is per-source attribution in error positions (`pos.Filename()` is empty unless the caller compiled with `cue.Filename(...)` themselves)

### Requirement: Compile Rename

The compile pipeline's terminal verb SHALL be `Compile`. The canonical entry point is `(*Kernel).Compile`. The free function `compile.CompileModuleRelease` SHALL NOT exist after this change. The earlier `pkg/render/process_module.go` / `render.ProcessModuleRelease` names SHALL NOT reappear inside `pkg/compile/`.

Note: `(*Kernel).ProcessModuleRelease` (added by this change) names a different operation — module-release validation, value-filling, and metadata decoding — distinct from the compile pipeline. The two names occupy different concepts and do not conflict.

#### Scenario: Canonical compile entry

- **WHEN** a caller invokes `k.Compile(ctx, in)` on a `*Kernel`
- **THEN** the call performs the full compile pipeline against `in.Platform` and returns a `*CompileResult`

#### Scenario: compile.CompileModuleRelease symbol gone

- **WHEN** a developer searches for `CompileModuleRelease` in `pkg/compile/`
- **THEN** the symbol does not exist
- **AND** callers MUST use `(*Kernel).Compile`

#### Scenario: ModuleResult aliased

- **WHEN** a caller references `*render.ModuleResult`
- **THEN** the type resolves to `*render.CompileResult` via a Go type alias

### Requirement: Canonical Implementations Live on Kernel

The canonical Go implementation of values validation (full, partial, and detailed) and module-release processing SHALL live on the `*Kernel` receiver in `pkg/kernel/`. No standalone `validate.Config` / `validate.ConfigPartial` / `module.ParseModuleRelease` free functions SHALL remain in the library; the `pkg/validate/` and `pkg/helper/values/` packages SHALL NOT exist after this change.

#### Scenario: ValidateConfig is a kernel method

- **WHEN** a caller invokes `k.ValidateConfig(schema, values)`
- **THEN** the method runs the full Tier-2 schema validation directly and returns the validated `cue.Value` on success or a CUE-native error on failure
- **AND** no `pkg/validate/` import is required by callers

#### Scenario: ValidateConfigPartial is a kernel method

- **WHEN** a caller invokes `k.ValidateConfigPartial(schema, values)`
- **THEN** the method runs the partial-validation entry point (catches type errors, disallowed fields, and pattern violations on fields that ARE set; does not flag missing fields) and returns the value on success or a CUE-native error on failure

#### Scenario: ValidateConfigDetailed is a kernel method

- **WHEN** a caller invokes `k.ValidateConfigDetailed(schema, sources, opts...)`
- **THEN** the method unifies the sources in order, then validates the merged value (full or partial depending on `Partial()` option) and returns the merged `cue.Value` plus a CUE-native error
- **AND** no `pkg/helper/values/` import is required by callers

#### Scenario: ProcessModuleRelease is a kernel method

- **WHEN** a caller invokes `k.ProcessModuleRelease(ctx, spec, mod, values)`
- **THEN** the method validates `values` via the kernel's own `ValidateConfig`, fills the validated value into `spec`, asserts concreteness via `spec.Validate(cue.Concrete(true))` (CUE stdlib), decodes release metadata via the binding, and returns a `*module.Release`
- **AND** the method does not delegate to any deprecated free function

#### Scenario: pkg/validate package is gone

- **WHEN** a developer runs `ls pkg/validate/` after this change ships
- **THEN** the directory does not exist

#### Scenario: pkg/helper/values package is gone

- **WHEN** a developer runs `ls pkg/helper/values/` after this change ships
- **THEN** the directory does not exist

#### Scenario: module.ParseModuleRelease free function is gone

- **WHEN** a developer searches `pkg/module/` for `ParseModuleRelease`
- **THEN** no free function with that name exists
- **AND** the only `ParseModuleRelease` symbol in the library is the deprecated method on `*Kernel` (see the deprecation requirement below)

#### Scenario: compile.CompileModuleRelease free function is gone

- **WHEN** a developer searches `pkg/compile/` for `CompileModuleRelease`
- **THEN** no free function with that name exists
- **AND** the canonical compile entry point is `(*Kernel).Compile`

### Requirement: ParseModuleRelease Deprecated Alias

`*Kernel` SHALL expose a deprecated `ParseModuleRelease` method that delegates to `ProcessModuleRelease` for one cycle to soften the rename for downstream callers.

#### Scenario: Alias delegates to canonical method

- **WHEN** a caller invokes `k.ParseModuleRelease(ctx, spec, mod, values)`
- **THEN** the result is identical to invoking `k.ProcessModuleRelease(ctx, spec, mod, values)`

#### Scenario: Alias carries deprecation marker

- **WHEN** a developer reads the godoc for `(*Kernel).ParseModuleRelease`
- **THEN** the comment begins with `// Deprecated:` and points to `(*Kernel).ProcessModuleRelease`

### Requirement: Utility Methods on Kernel

The Kernel SHALL expose `DetectAPIVersion(v cue.Value) (apiversion.Version, error)` and `Finalize(v cue.Value) (cue.Value, error)` as methods.

#### Scenario: DetectAPIVersion delegates to apiversion package

- **WHEN** a caller invokes `k.DetectAPIVersion(v)`
- **THEN** the result is identical to calling `apiversion.Detect(v)` directly
- **AND** the method exists for discovery purposes (callers find the operation through the Kernel anchor)

#### Scenario: Finalize uses kernel cue.Context

- **WHEN** a caller invokes `k.Finalize(v)`
- **THEN** the function performs schema-constraint stripping (existing `render.FinalizeValue` behavior) using the Kernel's `cue.Context`

### Requirement: Kernel.SynthesizeRelease method

The `*Kernel` type SHALL expose a method `SynthesizeRelease(ctx context.Context, in synth.ReleaseInput) (*module.Release, error)` that combines `synth.Release` and `Kernel.ProcessModuleRelease` into a single call: it builds the release spec by unifying inputs against the embedded schema, then validates the supplied values against the module's `#config`, fills the values into the spec, enforces concreteness, decodes release metadata, and returns the constructed `*module.Release`.

The method SHALL use the Kernel's owned `*cue.Context` when calling `synth.Release`. The method SHALL NOT consult any additional values source — `in.Values` is passed through to `Kernel.ProcessModuleRelease` unchanged.

#### Scenario: SynthesizeRelease produces an end-to-end release

- **WHEN** `k.SynthesizeRelease(ctx, synth.ReleaseInput{Module: mod, Name: "demo", Namespace: "default", Values: concreteValues})` is called against a registered v1alpha2 module
- **THEN** the returned `*module.Release` is non-nil
- **AND** `Release.APIVersion` equals the module's API version
- **AND** `Release.Metadata.Name` equals `"demo"`, `Release.Metadata.Namespace` equals `"default"`
- **AND** `Release.Metadata.UUID` equals `uuid.SHA1(OPMNamespace, "<module.uuid>:demo:default")`

#### Scenario: SynthesizeRelease rejects unconcrete result

- **WHEN** `k.SynthesizeRelease(ctx, in)` is called with `in.Values == cue.Value{}` against a module whose `#config` has required fields with no defaults
- **THEN** the returned error is non-nil and wraps the `Kernel.ProcessModuleRelease` concreteness diagnostic

#### Scenario: SynthesizeRelease surfaces synth errors before validation

- **WHEN** `k.SynthesizeRelease(ctx, synth.ReleaseInput{Module: nil, Name: "x", Namespace: "y"})` is called
- **THEN** the returned error is non-nil and originates from `synth.Release` (not from `Kernel.ProcessModuleRelease`)

#### Scenario: SynthesizeRelease uses the Kernel's cue.Context

- **WHEN** `k.SynthesizeRelease(ctx, in)` is called
- **AND** a developer inspects the cue.Context underlying the returned `Release.Package`
- **THEN** that context is the same instance returned by `k.CueContext()`

### Requirement: SynthesizeRelease is documented as the recommended in-memory entry point

The package documentation and the `Kernel.SynthesizeRelease` godoc SHALL state that `SynthesizeRelease` is the recommended entry point for building a release from typed inputs, mirroring how `Kernel.LoadReleasePackage` is the recommended entry point for building a release from a directory-based CUE package. Callers that explicitly want the helper-level primitive MAY call `synth.Release` followed by `Kernel.ProcessModuleRelease` directly.

#### Scenario: Documentation directs callers to the kernel method

- **WHEN** a developer reads the godoc on `pkg/helper/synth/`
- **THEN** the documentation states that `Kernel.SynthesizeRelease` is the recommended entry point
- **AND** notes that direct use of `synth.Release` is appropriate when the caller does not hold a `*Kernel`

#### Scenario: SynthesizeRelease godoc points to LoadReleasePackage

- **WHEN** a developer reads the `Kernel.SynthesizeRelease` godoc
- **THEN** the file-driven mirror it names is `Kernel.LoadReleasePackage`
- **AND** no reference to the removed `Kernel.LoadReleaseFile` remains

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
