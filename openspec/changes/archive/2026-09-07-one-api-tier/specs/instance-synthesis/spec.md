## ADDED Requirements

### Requirement: Instance synthesis input

Instance synthesis SHALL be reached only through `Kernel.SynthesizeInstance(ctx, kernel.InstanceInput)`. `InstanceInput` SHALL carry `Module *module.Module` (required, source-carrying), `Name string` (required), `Namespace string` (required), `Values []Source` (optional; empty means "no values supplied"), `Labels map[string]string` (optional) and `Annotations map[string]string` (optional). It SHALL carry no schema cache and no `cue.Context`: the kernel owns both. A missing required field SHALL fail with an error wrapping the matching `opm/errors` sentinel (`ErrMissingModule`, `ErrMissingName`, `ErrMissingNamespace`) before any build runs. No package under `opm/` SHALL export a second synthesis entry point or input type.

#### Scenario: Required inputs validated

- **WHEN** `SynthesizeInstance` is called with `Module == nil`, or `Name == ""`, or `Namespace == ""`
- **THEN** it returns a nil instance and an error wrapping the sentinel that names the missing field

#### Scenario: One synthesis entry point

- **WHEN** a consumer inspects the exported identifiers of every package under `opm/`
- **THEN** `Kernel.SynthesizeInstance` and `kernel.InstanceInput` are the only exported synthesis symbols; no `synth` package exists

#### Scenario: Returned value is schema-unified

- **WHEN** `SynthesizeInstance` is called with valid inputs
- **THEN** the returned instance's `Package` carries the `#ModuleInstance` shape at its root, unified with the schema's `#ModuleInstance` definition resolved through the module's own `cue.mod/module.cue`

## MODIFIED Requirements

### Requirement: Values field is caller-supplied with no implicit fallback

`SynthesizeInstance` SHALL NOT consult `Module.debugValues` or any other implicit source when `InstanceInput.Values` is empty. When `Values` holds one or more sources, the kernel SHALL unify them in stack order, render the result into the synthesized package's values source (via `format.Node` on the value's syntax, never string-interpolating raw input) so it participates in the single build, and after the build check each source against the module's `#config` at the source's own positions. When `Values` is empty, the kernel SHALL omit the values source and enforce concreteness on the built spec as usual.

#### Scenario: Caller-supplied values participate in the build

- **WHEN** `SynthesizeInstance` is called with `Values` holding a concrete source satisfying the module's `#config`
- **THEN** the returned instance carries those values at the schema's values path
- **AND** the values entered the build as a rendered source file, not a post-build cross-build unification

#### Scenario: Layered values unify in order

- **WHEN** `SynthesizeInstance` is called with `Values == []Source{a, b}` where `b` sets a field `a` leaves open
- **THEN** the returned instance's values carry `a` unified with `b`

#### Scenario: Zero Values is not replaced by debugValues

- **WHEN** `SynthesizeInstance` is called with empty `Values` against a Module that defines `debugValues`
- **THEN** the build's values path is unfilled (does not equal `debugValues`) and the call fails on concreteness unless every `#config` field has a default

### Requirement: Instance construction shares one evaluate-and-shape-gate with the file loader

The kernel SHALL evaluate a synthesized instance package (in-memory overlay inside the module's staged tree) and a directory-acquired instance package (on-disk source) through one build-and-shape-gate routine. The two paths SHALL differ only in how the package source is supplied; the evaluation, shape gating, sentinel wrapping and values attribution SHALL be identical.

#### Scenario: Overlay and on-disk instances evaluate identically

- **WHEN** the same instance content is supplied once through `SynthesizeInstance` and once through `AcquireInstanceFromDir`
- **THEN** both produce a value of the same shape passing the same instance shape gate
- **AND** a malformed instance fails the shape gate identically in both paths, wrapping the same `opm/errors` sentinel

### Requirement: synth.Instance requires the module's staged source

`SynthesizeInstance` SHALL construct the instance inside the acquired module's staged source tree (the overlay and root produced by `AcquireModuleFromRegistry` or `AcquireModuleFromDir`). When `InstanceInput.Module` carries no staged source (it was constructed from a bare value), the call SHALL fail with an error wrapping `oerrors.ErrMissingSource` naming the two acquire verbs. When the module's `Source.Pkg` is non-empty (the module was acquired from a subdirectory of its CUE module), the call SHALL fail with an error stating that a synthesizable module is its module's root package, because the synthesized package imports the module by its module path. The kernel SHALL NOT perform a registry fetch or a directory walk of its own to obtain the module source.

#### Scenario: Module without staged source is rejected

- **WHEN** `SynthesizeInstance` is called with a `Module` that carries no staged source
- **THEN** it returns a nil instance and an error wrapping `oerrors.ErrMissingSource`
- **AND** no registry fetch and no directory read occurs

#### Scenario: Module acquired with source synthesizes

- **WHEN** `SynthesizeInstance` is called with a `Module` from `AcquireModuleFromRegistry`
- **THEN** the instance is staged inside the module's source tree and synthesis proceeds

#### Scenario: Module acquired from a directory synthesizes

- **WHEN** `SynthesizeInstance` is called with a `Module` from `AcquireModuleFromDir` on a module root
- **THEN** the instance is staged inside the module's overlay, the module import resolves locally, and the module's own `cue.mod/module.cue` drives transitive resolution

#### Scenario: Subpackage module is refused

- **WHEN** `SynthesizeInstance` is called with a `Module` acquired from a subdirectory of its CUE module
- **THEN** it returns an error naming the root-package requirement and runs no build

### Requirement: Synthesis surfaces its staged tree

`SynthesizeInstance` SHALL stamp on the returned instance the staged source tree the single build evaluated: `Root` = the acquired module's staged root, `Pkg` = the reserved instance package subdirectory, `Overlay` = the module's cloned overlay augmented with the synthesized instance files (`instance.cue`, and `values.cue` when values were supplied), every entry as bytes. The overlay SHALL be a clone, so repeated synthesis from one module remains safe and the caller may retain the tree without aliasing the module's own overlay.

#### Scenario: The staged tree matches the build

- **WHEN** `SynthesizeInstance` succeeds
- **THEN** the returned instance's `Source.Root` equals the module's staged root, its `Pkg` names the reserved instance subdirectory, and its `Overlay` contains every file of the module's overlay plus the synthesized instance file (plus the values file when values were supplied)
- **AND** mutating the instance's overlay does not change the module's overlay

#### Scenario: Failure returns no tree

- **WHEN** `SynthesizeInstance` fails (missing inputs, missing source, build error, concreteness)
- **THEN** no instance and no staged tree is returned alongside the error

## REMOVED Requirements

### Requirement: Synth Helper Package Location

**Reason**: Synthesis is kernel work with one caller, `Kernel.SynthesizeInstance`; keeping it under `opm/helper/` made the kernel depend on its own "opt-in" tier.

**Migration**: Call `Kernel.SynthesizeInstance(ctx, kernel.InstanceInput{...})`. There is no direct synthesis function.

### Requirement: synth.Instance function signature

**Reason**: Replaced by "Instance synthesis input": the input type is kernel-owned, carries no schema cache, and takes values as `[]Source`.

**Migration**: `synth.InstanceInput{Module, Name, Namespace, Values: v, ...}` becomes `kernel.InstanceInput{Module, Name, Namespace, Values: []kernel.Source{{Value: v, Origin: origin}}, ...}`; a caller holding bytes uses `k.LoadSourceFromBytes(origin, b)`. Drop the `SchemaCache` field.

### Requirement: Schema obtained through caller-supplied Cache

**Reason**: The kernel owns the one schema cache; a caller-supplied cache on the input existed only for direct helper callers, which no longer exist.

**Migration**: Remove the `SchemaCache` field; the kernel's cache is used. `ErrMissingSchemaCache` no longer exists.

### Requirement: synth.Instance does not validate or enforce concreteness

**Reason**: Described the split between a public helper and the kernel method; the helper is no longer public. The observable behaviour (concreteness enforced and metadata decoded by `SynthesizeInstance`, unification errors surfaced through the returned error) is specified by `kernel-runtime` "Kernel.SynthesizeInstance method" and "Tier-2 validation runs where values are applied".

**Migration**: None for callers of `Kernel.SynthesizeInstance`.
