## ADDED Requirements

### Requirement: The Kernel owns no build context

The Kernel SHALL hold no `cue.Context`. Every kernel operation that evaluates CUE (module, platform and instance acquisition, synthesis, validation and render) SHALL create its own `cue.Context`, build in it, and let it go when the operation returns; the values an operation returns (an artifact's `Package`, a validated value) keep that operation's runtime alive for exactly as long as the caller holds them. No kernel method SHALL take a `cue.Value` from the caller as an input, and no kernel method SHALL return a value the caller is expected to unify with a value from another kernel call. The only long-lived evaluation state the Kernel owns is its schema cache, which holds a private context no accessor exposes.

#### Scenario: Repeated acquisitions retain nothing

- **WHEN** a long-lived Kernel acquires the same platform directory twenty times and the caller drops each result
- **THEN** the process heap after the twentieth acquisition is within noise of the heap after the first, and no accessor on the Kernel reaches a built value

#### Scenario: Artifacts cross Kernels

- **WHEN** a module is acquired with one Kernel, synthesized into an instance with a second, and rendered with a third against a platform acquired with a fourth
- **THEN** the render produces the same objects as the same sequence on one Kernel

#### Scenario: No context accessor

- **WHEN** a consumer inspects the exported methods of `Kernel`
- **THEN** no method returns or accepts a `*cue.Context`

## MODIFIED Requirements

### Requirement: Kernel Type and Construction

The library SHALL expose a `Kernel` struct in `opm/kernel/` that serves as the single public anchor type for the OPM kernel runtime. The struct SHALL be constructible only via the `kernel.New(opts ...Option)` function. Absent `WithSchemaLoader`, the schema cache SHALL be backed by an OCI loader whose registry mapping is the kernel's `WithRegistry` value (empty when the option is absent), so the schema and every other kernel operation resolve through one mapping. An explicit `WithSchemaLoader` SHALL take precedence regardless of option order. Construction SHALL create no `cue.Context` and perform no CUE evaluation.

#### Scenario: Default construction

- **WHEN** a caller invokes `kernel.New()` with no options
- **THEN** a non-nil `*Kernel` is returned holding a schema cache backed by the default OCI loader reading the process environment
- **AND** the Kernel holds no `cue.Context` and has evaluated nothing

#### Scenario: Registry option seeds the schema loader

- **WHEN** a caller invokes `kernel.New(WithRegistry(mapping))` with no `WithSchemaLoader`
- **THEN** the first schema-touching call resolves the core schema through `mapping`
- **AND** the process environment is not mutated

#### Scenario: Construction with options

- **WHEN** a caller invokes `kernel.New(WithSchemaLoader(myLoader), WithRegistry(mapping))` in either order
- **THEN** the returned Kernel resolves the core schema through `myLoader` and uses `mapping` for catalog and module resolution

### Requirement: Goroutine Safety Contract

A single `Kernel` SHALL be safe for concurrent use across its own method calls: no operation shares evaluation state with another, because each creates and releases its own `cue.Context`, and the schema cache is memoized under synchronization. A consumer that needs concurrent operations uses one `Kernel` per process; the package documentation SHALL state this and SHALL NOT recommend one Kernel per goroutine.

`Kernel.Render` SHALL share nothing between renders: each render is its own CUE build in a fresh `cue.Context` that is released when `Render` returns, and no built value is retained by the kernel. Concurrency is across operations, never within one; a consumer rendering from several goroutines calls `Render` on one Kernel, with no shared platform value and no mutex. The package documentation SHALL state this, SHALL NOT present any shared built value as a supported shape, and SHALL state that a render pool is sized by memory (about 61 MB plus 7.75 MB per component per concurrent render, 0019 experiment 08) rather than by core count. The retracted shared-materialized-platform model and its mutex stopgap SHALL NOT appear as supported shapes.

#### Scenario: Documentation states the contract

- **WHEN** a developer reads the godoc for the `Kernel` type
- **THEN** it states that a single `Kernel` is safe for concurrent use across its method calls, shows one Kernel shared by concurrent goroutines, and states that every operation shares nothing

#### Scenario: Documentation retracts the shared-platform model

- **WHEN** a developer reads the godoc for the `Kernel` type
- **THEN** no shared materialized platform, held built value or mutex-serialised render appears as a supported shape; the retraction is recorded in ADR-002's supersession header and the shares-nothing rule in ADR-005 and ADR-007

#### Scenario: Repeated renders retain nothing

- **WHEN** `Render` is invoked repeatedly on one Kernel
- **THEN** each invocation builds in a fresh context, the results are byte-identical for identical inputs, and the kernel holds no built value between calls

#### Scenario: Concurrent acquisitions on one Kernel

- **WHEN** several goroutines acquire artifacts and synthesize instances on one Kernel at the same time under the race detector
- **THEN** every call succeeds with the same result as the sequential run and the race detector reports nothing

#### Scenario: Race detector runs on the render packages

- **WHEN** the repository test task runs
- **THEN** `opm/kernel` and `opm/internal/renderstage` are additionally run under `go test -race`

### Requirement: Kernel.LoadSourceFromFile auto-unwraps the values field

The `*Kernel.LoadSourceFromFile(path string)` method SHALL read the file at `path`, parse it for syntax, and return a `Source` whose `Data` is the file's bytes and whose `Origin` is the file's absolute path; `Source` carries no other label. The method SHALL NOT evaluate CUE. When a kernel operation compiles that source, it SHALL load it through cue/load at the file's directory (so imports resolve as they do for any CUE file), evaluate it in that operation's own context, and:

- If the evaluated value contains a top-level `values:` field whose `Exists()` is true and `Err()` is nil, the compiled source SHALL be that field.
- Otherwise the compiled source SHALL be the whole evaluated value.

The method SHALL NOT depend on `loaderfile.LoadValuesFile` (which is removed).

#### Scenario: Values file is auto-unwrapped

- **WHEN** a caller invokes `k.LoadSourceFromFile("./values.cue")` against a file containing `values: { foo: "bar" }` and passes the source to `ValidateConfigDetailed`
- **THEN** the value validated is the inner `{ foo: "bar" }` value
- **AND** `Source.Origin` is the absolute path of `values.cue`

#### Scenario: File without values field passes through

- **WHEN** a caller invokes `k.LoadSourceFromFile("./flat.cue")` against a file with no top-level `values:` field and passes the source to `ValidateConfigDetailed`
- **THEN** the whole file value is validated
- **AND** `Source.Origin` is populated as above

#### Scenario: Syntax errors fail at load, schema errors at use

- **WHEN** a caller invokes `k.LoadSourceFromFile(path)` on a file that does not parse as CUE
- **THEN** the call returns an error positioned at the file's path and no `Source`
- **AND** a file that parses but violates a module's `#config` loads fine and fails, positioned at the file's path, in the operation that applies it

### Requirement: Kernel.SynthesizeInstance method

The `*Kernel` type SHALL expose a method `SynthesizeInstance(ctx context.Context, in InstanceInput) (*module.Instance, error)`, where `InstanceInput` is declared in `opm/kernel` and carries `Module *module.Module` (required, source-carrying), `Name string` (required), `Namespace string` (required), `Values []Source` (optional; empty means "no values supplied"), `Labels map[string]string` and `Annotations map[string]string` (optional). It SHALL build the instance spec by single-build CUE evaluation inside the module's staged source with the unified values rendered into the package, check the values sources against the module's `#config` at their own positions after the build, assert concreteness on the built spec, decode instance metadata, and return the constructed `*module.Instance` with its `Source` populated. The build SHALL run in a context the method creates and releases; the schema used is the kernel's own cache; the method SHALL NOT consult any additional values source.

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

- **WHEN** `k.SynthesizeInstance(ctx, in)` is called twice on one Kernel
- **THEN** the Kernel has no `cue.Context` to use: the two returned instances' `Package` values belong to two distinct contexts the method created, neither reachable through the Kernel

#### Scenario: Operator-shaped caller needs no cue.Context

- **WHEN** a frontend holds raw values bytes and their origin
- **THEN** it builds the input with `k.LoadSourceFromBytes(origin, raw)` and passes the result in `InstanceInput.Values`
- **AND** it needs no `cue.Context` of its own, since the kernel exposes none

## REMOVED Requirements

### Requirement: cue.Context Encapsulation

**Reason**: The Kernel no longer owns a `cue.Context`, so there is nothing to encapsulate and no accessor to mark advanced. The guarantee that no public method names a `*cue.Context` is restated by "The Kernel owns no build context".

**Migration**: callers of `k.CueContext()` compile values with `cuecontext.New()` when the value stands alone, or with `v.Context()` of the value they will unify with; the schema's context is `k.SchemaCache().Get()` followed by `.Context()`.
