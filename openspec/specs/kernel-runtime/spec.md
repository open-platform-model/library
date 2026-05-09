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

For every existing exported function in `pkg/loader/`, `pkg/module/`, `pkg/render/`, and `pkg/validate/` that takes a `*cue.Context`, the Kernel SHALL provide a method wrapper that sources `*cue.Context` from itself.

#### Scenario: Loader method wrapper

- **WHEN** a caller invokes `k.LoadModulePackage(ctx, "./module")`
- **THEN** the result is identical to calling `loader.LoadModulePackage(k.CueContext(), "./module")`
- **AND** any error returned is the same instance the underlying free function would return

#### Scenario: Existing free functions remain callable

- **WHEN** existing downstream code calls `loader.LoadModulePackage(cueCtx, dir)` directly
- **THEN** the call succeeds with the same behavior as before this slice
- **AND** the function carries a `// Deprecated:` doc comment pointing to the corresponding Kernel method

### Requirement: Phase-Explicit Methods on Kernel

The `Kernel` SHALL expose four phase-explicit methods, each accepting a phase-specific input struct and returning a phase-appropriate result.

#### Scenario: Validate phase method

- **WHEN** a caller invokes `k.Validate(ctx, ValidateInput{Module, ModuleRelease, Values})`
- **THEN** the kernel performs Tier-2 schema validation of `Values` against `Module.Package`'s `#config`
- **AND** returns nil on success or a `*oerrors.ConfigError` on failure
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

The kernel SHALL accept a single, pre-unified `cue.Value` for the values argument on every public method that takes user values. The kernel SHALL NOT accept `[]cue.Value` as a values argument on any public method.

#### Scenario: validate.Config takes a single value

- **WHEN** a caller invokes `validate.Config(schema, values, contextLabel, name)` with `values` as a `cue.Value`
- **THEN** the function validates the supplied `values` against `schema` and returns the validated value or a `*ConfigError`
- **AND** there is no internal merge loop; the function consumes `values` as-is

#### Scenario: ParseModuleRelease takes a single value

- **WHEN** a caller invokes `module.ParseModuleRelease(ctx, spec, mod, values)` with `values` as a single `cue.Value`
- **THEN** the function validates `values` via `validate.Config`, fills the validated value into `spec`, and returns a `*Release`
- **AND** the function does not accept a slice form

#### Scenario: Empty values is the zero value

- **WHEN** a caller passes a zero-value `cue.Value{}` to `validate.Config` or `module.ParseModuleRelease`
- **THEN** the call succeeds (no validation errors, no fill operation)
- **AND** the behavior matches the previous slice's "no values supplied" path

### Requirement: Tier-2 Validation Always Runs

When values are non-empty, the kernel SHALL validate them against the Module's `#config` schema regardless of whether a Tier-1 helper validated them upstream.

#### Scenario: Kernel re-validates after Tier-1

- **WHEN** a frontend that uses `pkg/helper/values` (slice 05) supplies a unified value to `validate.Config`
- **THEN** the kernel performs full schema validation on the unified value
- **AND** any schema violation produces a `*ConfigError`

#### Scenario: Kernel validates without Tier-1

- **WHEN** a frontend skips Tier-1 helper validation and feeds raw unified values directly
- **THEN** the kernel still produces correct schema-validation errors via `*ConfigError`
- **AND** the only loss is per-source attribution in error messages

### Requirement: Temporary Migration Helper

The library SHALL provide `validate.UnifyAndValidate(vs []cue.Value) cue.Value` (or equivalent name) as a temporary helper that performs the previous slice-merge behavior, returning a single `cue.Value` callers can pass to the new signature. This helper SHALL be marked `// Deprecated:` from introduction and SHALL be removed when `pkg/helper/values` (slice 05) lands.

#### Scenario: Migration helper exists

- **WHEN** a caller invokes `validate.UnifyAndValidate(vs)` with the same slice they previously passed to `Config`
- **THEN** the helper returns a single unified `cue.Value` ready to pass to the new `Config` signature
- **AND** the helper carries a `// Deprecated:` doc comment pointing to `pkg/helper/values`

#### Scenario: Migration helper retired in slice 05

- **WHEN** slice 05 (`introduce-tiered-validation`) merges
- **THEN** the next change cycle removes `validate.UnifyAndValidate`
- **AND** consumers migrate to `pkg/helper/values` for layering

### Requirement: Compile Rename

The compile pipeline's terminal verb SHALL be `Compile`. `pkg/compile/compile_module.go` carries `compile.CompileModuleRelease`. The earlier `pkg/render/process_module.go` / `render.ProcessModuleRelease` names SHALL NOT reappear.

#### Scenario: New name available

- **WHEN** a caller invokes `compile.CompileModuleRelease(ctx, rel, plat, runtimeName)`
- **THEN** the call performs the full compile pipeline against the supplied `*platform.Platform` and returns a `*CompileResult`

#### Scenario: ProcessModuleRelease alias removed

- **WHEN** a developer searches for `ProcessModuleRelease` in `pkg/compile/` or `pkg/kernel/`
- **THEN** the symbol does not exist
- **AND** callers MUST use `CompileModuleRelease` (free function) or `(*Kernel).Compile` (method)

#### Scenario: ModuleResult aliased

- **WHEN** a caller references `*render.ModuleResult`
- **THEN** the type resolves to `*render.CompileResult` via a Go type alias

### Requirement: Utility Methods on Kernel

The Kernel SHALL expose `DetectAPIVersion(v cue.Value) (apiversion.Version, error)` and `Finalize(v cue.Value) (cue.Value, error)` as methods.

#### Scenario: DetectAPIVersion delegates to apiversion package

- **WHEN** a caller invokes `k.DetectAPIVersion(v)`
- **THEN** the result is identical to calling `apiversion.Detect(v)` directly
- **AND** the method exists for discovery purposes (callers find the operation through the Kernel anchor)

#### Scenario: Finalize uses kernel cue.Context

- **WHEN** a caller invokes `k.Finalize(v)`
- **THEN** the function performs schema-constraint stripping (existing `render.FinalizeValue` behavior) using the Kernel's `cue.Context`
