## MODIFIED Requirements

### Requirement: Kernel.SynthesizeInstance method

The `*Kernel` type SHALL expose a method `SynthesizeInstance(ctx context.Context, in InstanceInput) (*module.Instance, error)`, where `InstanceInput` is declared in `opm/kernel` and carries `Module *module.Module` (required, source-carrying), `Name string` (required), `Namespace string` (required), `Values []Source` (optional; empty means "no values supplied"), `Labels map[string]string` and `Annotations map[string]string` (optional). It SHALL build the instance spec by single-build CUE evaluation inside the module's staged source with the unified values rendered into the package, check the values sources against the module's `#config` at their own positions after the build, assert concreteness on the built spec, decode instance metadata, and return the constructed `*module.Instance` with its `Source` populated. The synthesized package imports `core` at the major of the kernel's schema release (read from the loader's pin with no schema load when it names an exact release, resolved through the schema cache otherwise); the release that import resolves to comes from the module's own `cue.mod/module.cue`. The build SHALL run in a context the method creates and releases; the method SHALL NOT consult any additional values source.

When the build fails and the call's merged values exist, the method SHALL attribute the failure to the values sources the way `AcquireInstanceFromDir` does: it SHALL build the synthesized package again in the same context without the rendered values file, take `#config` from that build (never from the module's `Package`), and validate the values it already compiled against it with concreteness enforced. A values error SHALL be returned framed `Kernel.SynthesizeInstance: instance "<name>": …`, with positions that name each violating source's `Origin`. In every other case (no values, the values-free build also fails, or the values are clean) the method SHALL return the original build error unchanged.

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

#### Scenario: SynthesizeInstance attributes a values conflict that fails the build

- **WHEN** `k.SynthesizeInstance(ctx, in)` is called with a source (Origin `/values/bad.cue`) whose value violates the module's `#config` at a path a component of the module consumes, so the synthesized build fails
- **THEN** the returned error reads `Kernel.SynthesizeInstance: instance "<name>": …`, names the violating path, and at least one of its positions reports `/values/bad.cue`
- **AND** no instance is returned

#### Scenario: A build failure with clean values is returned unchanged

- **WHEN** `k.SynthesizeInstance(ctx, in)` is called with values that satisfy the module's `#config` against a module whose own component fails to evaluate
- **THEN** the returned error is the build error framed `Kernel.SynthesizeInstance: …`, not framed `instance "<name>": …`, and no instance is returned

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

### Requirement: Tier-2 validation runs where values are applied

When values are non-empty, the kernel SHALL validate them against the Module's `#config` schema at the point they are applied to the instance, regardless of whether a Tier-1 helper validated them upstream. Values are applied inside a CUE build: `AcquireInstanceFromDir` renders the unified sources into the package overlay, and `SynthesizeInstance` renders them into the synthesized package; both then check the sources against `#config` so a violation is reported at the sources' own positions, and both assert concreteness on the whole built spec. Each verb SHALL compile each values source once per call: the merge that feeds the build, the post-build check against `#config` and the failure-path attribution SHALL all use those compiled values, which live in the call's own context with the spec they are checked against. `ValidateConfigDetailed` compiles its sources once per call as well. `Kernel.Render` SHALL NOT perform a second validation pass: the render build imports the instance as processed, which is already concrete.

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

