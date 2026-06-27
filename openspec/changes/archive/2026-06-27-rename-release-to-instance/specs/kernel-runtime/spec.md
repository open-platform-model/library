# kernel-runtime Specification

## ADDED Requirements

### Requirement: ParseModuleInstance Deprecated Alias

`*Kernel` SHALL expose a deprecated `ParseModuleInstance` method that delegates to `ProcessModuleInstance` for one cycle to soften the rename for downstream callers.

#### Scenario: Alias delegates to canonical method

- **WHEN** a caller invokes `k.ParseModuleInstance(ctx, spec, mod, values)`
- **THEN** the result is identical to invoking `k.ProcessModuleInstance(ctx, spec, mod, values)`

#### Scenario: Alias carries deprecation marker

- **WHEN** a developer reads the godoc for `(*Kernel).ParseModuleInstance`
- **THEN** the comment begins with `// Deprecated:` and points to `(*Kernel).ProcessModuleInstance`

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

## MODIFIED Requirements

### Requirement: Goroutine Safety Contract

A single `Kernel` SHALL NOT be used concurrently across its own method calls — the owned `*cue.Context` is driven single-threaded, and sharing one `Kernel` between goroutines can race inside CUE evaluation. Callers needing concurrent operations SHALL construct one `Kernel` per goroutine; the package documentation SHALL state this and provide a one-Kernel-per-goroutine example.

Under the v0.17 CUE toolchain, a `*MaterializedPlatform` produced by one `Kernel` SHALL be safe to share **read-only** across goroutines and other Kernels: many per-goroutine Kernels MAY render distinct `ModuleInstance`s concurrently against a single platform that was materialized once, with no mutex and no re-materialization. This is sound because the compile pipeline builds every value it constructs in the **caller** Kernel's `*cue.Context` and only cross-*reads* the shared platform (see the "Compile sources its cue.Context from the caller Kernel" requirement). The package documentation SHALL describe this concurrent-render model and provide an example of rendering against a shared platform.

#### Scenario: Documentation states the contract

- **WHEN** a developer reads the godoc for the `Kernel` type
- **THEN** the documentation explicitly states that a single `Kernel` is not safe for concurrent use across its own method calls
- **AND** the documentation provides an example showing one-Kernel-per-goroutine usage in a multi-worker scenario

#### Scenario: Documentation states the shared-platform concurrency model

- **WHEN** a developer reads the godoc for the `Kernel` type
- **THEN** the documentation states that, under the v0.17 toolchain, a `*MaterializedPlatform` materialized once is safe to be read concurrently by many per-goroutine Kernels' `Compile` calls without a mutex or re-materialization
- **AND** the documentation provides an example of per-goroutine Kernels rendering against one shared materialized platform

#### Scenario: Concurrent rendering against a shared platform is race-clean and correct

- **WHEN** one Kernel materializes a platform once, and N other goroutines each construct their own Kernel and concurrently `Compile` a distinct `ModuleInstance` against that single shared `*MaterializedPlatform`, executed under the race detector
- **THEN** no data race is reported
- **AND** each goroutine's `CompileResult` contains the output expected for its own instance, with no cross-contamination between concurrent renders

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

The `Kernel` SHALL expose four phase-explicit methods, each accepting a phase-specific input struct and returning a phase-appropriate result.

#### Scenario: Validate phase method

- **WHEN** a caller invokes `k.Validate(ctx, ValidateInput{Module, ModuleInstance, Values})`
- **THEN** the kernel performs Tier-2 schema validation of `Values` against `Module.Package`'s `#config` by calling `k.ValidateConfig` internally
- **AND** returns nil on success or a CUE-native error wrapped with `fmt.Errorf("module %q: %w", name, err)` on failure
- **AND** does not perform matching, execution, or finalization

#### Scenario: Match phase method

- **WHEN** a caller invokes `k.Match(ctx, MatchInput{Module, ModuleInstance, Platform})`
- **THEN** the kernel produces a `*MatchPlan` describing matched and non-matched component/transformer pairs
- **AND** does not execute any transformer

#### Scenario: Plan phase method

- **WHEN** a caller invokes `k.Plan(ctx, PlanInput{Module, ModuleInstance, Values, Platform, RuntimeName})`
- **THEN** the kernel runs the full Compile pipeline (Validate + Match + Execute + Finalize) and returns a `*PlanResult` containing component summaries, unmatched FQNs, ambiguous FQNs, and warnings
- **AND** does not return rendered values

#### Scenario: Compile phase method

- **WHEN** a caller invokes `k.Compile(ctx, CompileInput{Module, ModuleInstance, Values, Platform, RuntimeName})`
- **THEN** the kernel runs the full pipeline (Validate + Match + Execute + Finalize) and returns a `*CompileResult` containing `Compiled []*core.Compiled`, component summaries, unmatched FQNs, ambiguous FQNs, and warnings

### Requirement: Phase Input Structs

Each phase method SHALL accept a phase-specific input struct rather than positional arguments. Input structs SHALL be additive — new fields SHALL be addable without breaking existing call sites. Phases that operate on a constructed `*module.Instance` SHALL NOT carry a parallel `*module.Module` field; the source module is reachable via the instance's `Package` at the binding's `Paths().Module`. The matcher-facing input structs (`MatchInput`, `PlanInput`, `CompileInput`) SHALL carry the platform as a `*materialize.MaterializedPlatform` (the realized form), not a raw `*platform.Platform`; callers MUST `Materialize` before invoking these phases.

#### Scenario: ValidateInput shape

- **WHEN** a developer reads the `ValidateInput` struct
- **THEN** the struct has at minimum `Module *module.Module`, `ModuleInstance *module.Instance`, and `Values cue.Value`
- **AND** each field has godoc explaining its role

#### Scenario: MatchInput shape

- **WHEN** a developer reads the `MatchInput` struct
- **THEN** the struct has exactly `ModuleInstance *module.Instance` and `Platform *materialize.MaterializedPlatform` as required artifact fields
- **AND** the struct has no `Module` field

#### Scenario: PlanInput shape

- **WHEN** a developer reads the `PlanInput` struct
- **THEN** the struct has `ModuleInstance *module.Instance`, `Values cue.Value`, `Platform *materialize.MaterializedPlatform`, and `RuntimeName string`
- **AND** the struct has no `Module` field

#### Scenario: CompileInput shape

- **WHEN** a developer reads the `CompileInput` struct
- **THEN** the struct has `ModuleInstance *module.Instance`, `Values cue.Value`, `Platform *materialize.MaterializedPlatform`, and `RuntimeName string`
- **AND** the struct has no `Module` field
- **AND** the struct has no `Provider` field

#### Scenario: Compile sources its #config schema from the instance

- **WHEN** `kernel.Compile` runs its embedded Tier-2 validation step on a `CompileInput`
- **THEN** the `#config` schema is obtained from `in.ModuleInstance.Package` via the binding's `Paths().Module` + `Paths().Config`
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

### Requirement: Compile Rename

The compile pipeline's terminal verb SHALL be `Compile`. The canonical entry point is `(*Kernel).Compile`. The free function `compile.CompileModuleInstance` SHALL NOT exist after this change. The earlier `opm/render/process_module.go` / `render.ProcessModuleInstance` names SHALL NOT reappear inside `opm/compile/`.

Note: `(*Kernel).ProcessModuleInstance` (added by this change) names a different operation — module-instance validation, value-filling, and metadata decoding — distinct from the compile pipeline. The two names occupy different concepts and do not conflict.

#### Scenario: Canonical compile entry

- **WHEN** a caller invokes `k.Compile(ctx, in)` on a `*Kernel`
- **THEN** the call performs the full compile pipeline against `in.Platform` and returns a `*CompileResult`

#### Scenario: compile.CompileModuleInstance symbol gone

- **WHEN** a developer searches for `CompileModuleInstance` in `opm/compile/`
- **THEN** the symbol does not exist
- **AND** callers MUST use `(*Kernel).Compile`

#### Scenario: ModuleResult aliased

- **WHEN** a caller references `*render.ModuleResult`
- **THEN** the type resolves to `*render.CompileResult` via a Go type alias

### Requirement: Canonical Implementations Live on Kernel

The canonical Go implementation of values validation (full, partial, and detailed) and module-instance processing SHALL live on the `*Kernel` receiver in `opm/kernel/`. No standalone `validate.Config` / `validate.ConfigPartial` / `module.ParseModuleInstance` free functions SHALL remain in the library; the `opm/validate/` and `opm/helper/values/` packages SHALL NOT exist after this change.

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
- **THEN** the method validates `values` via the kernel's own `ValidateConfig`, fills the validated value into `spec`, asserts concreteness via `spec.Validate(cue.Concrete(true))` (CUE stdlib), decodes instance metadata via the binding, and returns a `*module.Instance`
- **AND** the method does not delegate to any deprecated free function

#### Scenario: opm/validate package is gone

- **WHEN** a developer runs `ls opm/validate/` after this change ships
- **THEN** the directory does not exist

#### Scenario: opm/helper/values package is gone

- **WHEN** a developer runs `ls opm/helper/values/` after this change ships
- **THEN** the directory does not exist

#### Scenario: module.ParseModuleInstance free function is gone

- **WHEN** a developer searches `opm/module/` for `ParseModuleInstance`
- **THEN** no free function with that name exists
- **AND** the only `ParseModuleInstance` symbol in the library is the deprecated method on `*Kernel` (see the deprecation requirement below)

#### Scenario: compile.CompileModuleInstance free function is gone

- **WHEN** a developer searches `opm/compile/` for `CompileModuleInstance`
- **THEN** no free function with that name exists
- **AND** the canonical compile entry point is `(*Kernel).Compile`

### Requirement: Compile sources its cue.Context from the caller Kernel

The compile pipeline (Finalize → Match → Execute, driven by `Kernel.Compile`) SHALL build every value it constructs — the finalized data components, the per-pair transformer `#context.*` view, and the rendered output — using the **caller Kernel's** owned `*cue.Context` (the instance returned by `k.CueContext()`). It SHALL NOT derive the build context from the materialized platform (`mp.Package.Context()` / `platform.Package.Context()`). The materialized platform's `Package` is read as input (the `FillPath` argument and cross-read source), not as the owner of the build context.

#### Scenario: Compiled output builds in the Kernel's cue.Context

- **WHEN** a developer calls `k.Compile(ctx, in)` and inspects the `*cue.Context` underlying a rendered value in the returned `CompileResult.Compiled`
- **THEN** that context is the same instance returned by `k.CueContext()`

#### Scenario: Pipeline does not call Value.Context on the platform

- **WHEN** the compile pipeline finalizes components and executes transformers
- **THEN** it obtains its `*cue.Context` from the caller Kernel
- **AND** it does not call `Value.Context()` on the materialized platform's `Package`

#### Scenario: Behavior preserved for single-Kernel callers

- **WHEN** a single Kernel materializes a platform and then compiles an instance against it (the platform's `Package` was built in that same Kernel's `*cue.Context`)
- **THEN** the rendered output is identical to the prior platform-context-sourced behavior, because the caller context and the platform context are the same instance

## REMOVED Requirements

### Requirement: ParseModuleRelease Deprecated Alias

**Reason**: Renamed for Release→Instance vocabulary (enhancement 0002 D11/D12).
**Migration**: See the ADDED requirement of the new name.

### Requirement: Kernel.SynthesizeRelease method

**Reason**: Renamed for Release→Instance vocabulary (enhancement 0002 D11/D12).
**Migration**: See the ADDED requirement of the new name.

### Requirement: SynthesizeRelease is documented as the recommended in-memory entry point

**Reason**: Renamed for Release→Instance vocabulary (enhancement 0002 D11/D12).
**Migration**: See the ADDED requirement of the new name.
