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
