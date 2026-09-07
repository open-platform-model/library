## ADDED Requirements

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

## MODIFIED Requirements

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

## REMOVED Requirements

### Requirement: Single Pre-Unified Values Input

**Reason**: Superseded by "Values enter as sources": `SynthesizeInstance` no longer takes a single pre-unified `cue.Value`; all three values-taking entries take `Source` values, so the requirement's name and its single-value scenarios no longer describe the surface.

**Migration**: A caller holding one pre-unified `cue.Value` wraps it as a one-element `Source` list (`kernel.Source{Value: v, Origin: origin}`), or builds it with `LoadSourceFromFile` / `LoadSourceFromBytes`.

### Requirement: Backward-Compatible Method Wrappers

**Reason**: The wrappers existed to forward to helper loaders and constructors the kernel no longer exposes. With the helpers folded into the kernel and the acquire verbs the only way to obtain a source-carrying artifact, a second value-only tier is surface without a consumer (the operator used none of it; the cli's five sites are acquire calls in disguise).

**Migration**: `k.LoadModulePackage(ctx, dir, opts)` + `k.NewModuleFromValue(v)` becomes `k.AcquireModuleFromDir(ctx, dir)`; `k.LoadPlatformPackage` + `k.NewPlatformFromValue` becomes `k.AcquirePlatformFromDir(ctx, dir)`; `k.LoadInstancePackage` becomes `k.AcquireInstanceFromDir(ctx, dir)`. The raw value is the artifact's `Package` field. A frontend holding a value it built itself calls `module.NewModuleFromValue(v)` or `platform.NewPlatformFromValue(v)` directly.
