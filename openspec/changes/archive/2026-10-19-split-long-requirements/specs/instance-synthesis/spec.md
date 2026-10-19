## MODIFIED Requirements

### Requirement: Instance synthesis input
Instance synthesis SHALL be reached only through `Kernel.SynthesizeInstance(ctx, kernel.InstanceInput)`. `InstanceInput` SHALL carry `Module *module.Module` (required, source-carrying), `Name string` (required), `Namespace string` (required), `Values []Source` (optional; empty means "no values supplied"), `Labels map[string]string` (optional) and `Annotations map[string]string` (optional).

#### Scenario: Required inputs validated

- **WHEN** `SynthesizeInstance` is called with `Module == nil`, or `Name == ""`, or `Namespace == ""`
- **THEN** it returns a nil instance and an error wrapping the sentinel that names the missing field

#### Scenario: One synthesis entry point

- **WHEN** a consumer inspects the exported identifiers of every package under `opm/` outside `opm/internal/`
- **THEN** `Kernel.SynthesizeInstance` and `kernel.InstanceInput` are the only exported synthesis symbols; no public `synth` package exists, and `opm/internal/synth` is importable only from inside the library

#### Scenario: Returned value is schema-unified

- **WHEN** `SynthesizeInstance` is called with valid inputs
- **THEN** the returned instance's `Package` carries the `#ModuleInstance` shape at its root, unified with the schema's `#ModuleInstance` definition resolved through the module's own `cue.mod/module.cue`

## REMOVED Requirements

### Requirement: Values field is caller-supplied with no implicit fallback
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "No implicit source replaces an empty Values" and "Caller-supplied Values are unified in stack order, rendered into the build and checked against #config".

### Requirement: Optional labels and annotations are filled into instance metadata
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Caller labels and annotations fill instance metadata", "Caller labels cannot remove schema-stamped labels" and "The staged instance source is deterministic and emits entries in ascending key order".

### Requirement: Imported-module render coverage exists
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "A test renders an instance whose module is referenced by import, on the synth path and an authored-package path" and "The synth-path coverage includes a module that imports a catalog subpackage".

### Requirement: synth.Instance constructs the instance by single-build CUE evaluation
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "synth.Instance builds the instance in a single build through the shape gate", "The instance source is overlaid into the module's staged source tree and no module file is fabricated", "The instance source imports the module's package and core, and the module import resolves locally", "A values file rendered from the input's Values is overlaid beside the instance source", "The schema merges the values and the Go code fills only caller-supplied inputs", "synth.Instance lives in opm/internal/synth with a fixed Go signature", "The cueCtx and coreVersion parameters of synth.Instance are the call context and the core release" and "The Input parameter of synth.Instance is the package-internal input".

### Requirement: synth.Instance requires the module's staged source
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Synthesis constructs the instance inside the acquired module's staged source tree", "A module without staged source is refused and the kernel fetches nothing itself", "A module acquired from a subdirectory of its CUE module is refused", "An existing synthetic root refuses a synthesis of a registry-acquired module" and "A module acquired from a directory is not checked for a synthetic root".

### Requirement: Synthesis surfaces its staged tree
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Synthesis stamps the staged source tree on the returned instance" and "The stamped overlay is a clone".

## ADDED Requirements

### Requirement: No implicit source replaces an empty Values
`SynthesizeInstance` SHALL NOT consult `Module.debugValues` or any other implicit source when `InstanceInput.Values` is empty. When `Values` is empty, the kernel SHALL omit the values source and enforce concreteness on the built spec as usual.

#### Scenario: Zero Values is not replaced by debugValues

- **WHEN** `SynthesizeInstance` is called with empty `Values` against a Module that defines `debugValues`
- **THEN** the build's values path is unfilled (does not equal `debugValues`) and the call fails on concreteness

### Requirement: Caller-supplied Values are unified in stack order, rendered into the build and checked against #config
When `Values` holds one or more sources, the kernel SHALL unify them in stack order, render the result into the synthesized package's values source (via `format.Node` on the value's syntax, never string-interpolating raw input) so it participates in the single build, and after the build check each source against the module's `#config` at the source's own positions.

#### Scenario: Caller-supplied values participate in the build

- **WHEN** `SynthesizeInstance` is called with `Values` holding a concrete source satisfying the module's `#config`
- **THEN** the returned instance carries those values at the schema's values path
- **AND** the values entered the build as a rendered source file, not a post-build cross-build unification

#### Scenario: Layered values unify in order

- **WHEN** `SynthesizeInstance` is called with `Values == []Source{a, b}` where `b` sets a field `a` leaves open
- **THEN** the returned instance's values carry `a` unified with `b`

### Requirement: Caller labels and annotations fill instance metadata
When `InstanceInput.Labels` is non-empty, `synth.Instance` SHALL fill `metadata.labels` with those entries. When `InstanceInput.Annotations` is non-empty, `synth.Instance` SHALL fill `metadata.annotations` with those entries.

#### Scenario: Annotations are passed through unchanged

- **WHEN** `synth.Instance` is called with `Annotations == {"opmodel.dev/owner": "team-x"}`
- **THEN** the returned CUE value's `metadata.annotations` contains that entry

### Requirement: Caller labels cannot remove schema-stamped labels
The schema's label stamping (Requirement: Derived fields come from schema unification) unifies with caller-supplied labels: caller labels MUST NOT be allowed to remove schema-stamped labels.

#### Scenario: Caller labels merged with schema-stamped labels

- **WHEN** `synth.Instance` is called with `Labels == {"env": "prod"}`
- **THEN** the returned CUE value's `metadata.labels` contains the `env: prod` entry
- **AND** still contains `module-instance.opmodel.dev/name` and `module-instance.opmodel.dev/uuid`

### Requirement: The staged instance source is deterministic and emits entries in ascending key order
The staged instance source SHALL be deterministic: for identical inputs, the bytes synthesis emits for `metadata.labels` and `metadata.annotations` SHALL be identical across calls and across processes, independent of the order the caller's maps iterate in. Entries SHALL be emitted in ascending key order.

#### Scenario: Label and annotation order is stable

- **WHEN** `synth.Instance` is called twice with `Labels` and `Annotations` each holding the same four or more entries, inserted into the maps in different orders
- **THEN** the `instance.cue` bytes on the two returned staged trees are byte-identical
- **AND** the entries appear in ascending key order

### Requirement: A test renders an instance whose module is referenced by import, on the synth path and an authored-package path
The library SHALL include a test that renders an instance whose module is referenced by import (not inlined) end-to-end through construction and `Kernel.Render`, against a catalog-importing platform (a platform module importing its catalog, enhancement 0019), producing concrete resources. This coverage SHALL exist for both the synth path and an authored-package path so that a regression in either surfaces.

#### Scenario: Real imported module renders to resources

- **WHEN** an instance referencing a published module by import is rendered through `Kernel.Render`
- **THEN** the rendered output contains the module's expected resources
- **AND** the test fails if import-based construction regresses to a `field not allowed` admission error

#### Scenario: Single-build parity with an authored package

- **WHEN** `synth.Instance` builds an instance for module M with values V, and an authored `instance.cue` package imports the same M and sets the same V
- **THEN** both, passed through `Kernel.Render` against the same platform, produce the same rendered objects

### Requirement: The synth-path coverage includes a module that imports a catalog subpackage
The synth-path coverage SHALL include a module that imports a **catalog subpackage** (e.g. a workload blueprint under `opmodel.dev/catalogs/opm/...`), so that a regression to a dependency-incomplete synthesis surfaces as a failing test.

#### Scenario: Module importing a catalog subpackage synthesizes and renders

- **WHEN** `synth.Instance` is called with a published module whose source imports a transitive catalog subpackage (the library#31 shape), against a registry serving that catalog
- **THEN** synthesis succeeds without `cannot find module providing package opmodel.dev/catalogs/opm/...`
- **AND** the instance renders to the module's expected resources through `Kernel.Render`

### Requirement: synth.Instance builds the instance in a single build through the shape gate
`synth.Instance` SHALL construct the `#ModuleInstance` value by synthesizing an in-memory CUE package and evaluating it in a **single build** (per ADR-006), through the same build-and-shape-gate step a directory-acquired instance package runs (`loader.LoadDir` with the instance shape gate, the step behind `Kernel.AcquireInstanceFromDir`).

#### Scenario: Single-build parity with an authored package

- **WHEN** `synth.Instance` builds an instance for module M with values V, and an authored `instance.cue` package imports the same M and sets the same V
- **THEN** both, passed through `Kernel.Render`, produce the same set of rendered resources

### Requirement: The instance source is overlaid into the module's staged source tree and no module file is fabricated
The instance source SHALL be overlaid **into the acquired module's own staged source tree** (under a reserved synthetic subdirectory of the module's staged root), so the module's published, already-tidied `cue.mod/module.cue` is the build's main-module module file and drives all transitive dependency resolution. `synth.Instance` SHALL NOT fabricate a `cue.mod/module.cue` and SHALL NOT declare a dependency set; it reuses the module's tidied deps verbatim.

#### Scenario: No fabricated module file

- **WHEN** `synth.Instance` constructs an instance
- **THEN** the build's main-module `cue.mod/module.cue` is the module's own published module file (carrying its full tidied dependency closure)
- **AND** `synth.Instance` does not generate a `cue.mod/module.cue` or a `deps:` block of its own

### Requirement: The instance source imports the module's package and core, and the module import resolves locally
The instance source SHALL **import** the module's own package by its module path and write `#module: <import>` plus caller-supplied `metadata`, and SHALL import `core` (resolved from the module's own deps) to embed `#ModuleInstance`. Because the instance is built inside the module's main module, the module import SHALL resolve **locally** (no fabricated dependency, no registry round-trip for the module itself).

#### Scenario: Imported-module construction succeeds

- **WHEN** `synth.Instance` is called with a module whose identity is author-supplied (`metadata.modulePath` / `metadata.version` concrete)
- **THEN** the instance constructs without a `field not allowed` admission error
- **AND** the value unifies cleanly against `#ModuleInstance`

### Requirement: A values file rendered from the input's Values is overlaid beside the instance source
A values file rendered from the input's `Values` (the unified value of the caller's `InstanceInput.Values` sources, which `Kernel.SynthesizeInstance` compiles and unifies before the call) SHALL be overlaid alongside the instance source when that value exists.

#### Scenario: The values file is overlaid beside the instance source

- **WHEN** `Kernel.SynthesizeInstance` has compiled and unified the `InstanceInput.Values` sources before the call and that value exists
- **THEN** a values file rendered from the unified value is overlaid alongside the instance source

### Requirement: The schema merges the values and the Go code fills only caller-supplied inputs
`synth.Instance` SHALL NOT inject the module via `cue.Scope` / a `userModule` field, and SHALL NOT pre-merge `Values` into the module's `#config` in Go. The values merge SHALL be performed by the schema in CUE (`#ModuleInstance`'s `unifiedModule = #module & {#config: values}`). The Go code SHALL fill only caller-supplied inputs and let CUE derive `metadata.uuid`, `components`, `opm-secrets`, and stamped labels.

#### Scenario: No scope or Go pre-merge in the construction path

- **WHEN** `synth.Instance` is called with valid inputs
- **THEN** the construction performs no `cue.Scope`-based compile and no `FillPath` of `Values` into the module's `#config`
- **AND** the returned value carries the `#ModuleInstance` shape with `components` fanned by the schema comprehension

#### Scenario: Values merged in-build by the schema

- **WHEN** `synth.Instance` is called with `Values` satisfying the module's `#config`
- **THEN** the returned value's `components` reflect the unified `#config = values` configuration
- **AND** the merge is produced by CUE evaluation of the synthesized package, not by a Go-side `#config` fill

### Requirement: synth.Instance lives in opm/internal/synth with a fixed Go signature
`synth.Instance` SHALL live in `opm/internal/synth`, off the public surface; its Go signature is `Instance(cueCtx *cue.Context, coreVersion string, in Input) (cue.Value, *module.Source, error)`.

#### Scenario: synth.Instance lives off the public surface

- **WHEN** a consumer looks for `synth.Instance`
- **THEN** it lives in `opm/internal/synth`, off the public surface, with the Go signature `Instance(cueCtx *cue.Context, coreVersion string, in Input) (cue.Value, *module.Source, error)`

### Requirement: The cueCtx and coreVersion parameters of synth.Instance are the call context and the core release
`synth.Instance`: `cueCtx` SHALL be the context `Kernel.SynthesizeInstance` created for the call, `coreVersion` is the kernel's core release, from which the synthesized `core` import takes its major (the release the import resolves to comes from the module's own `cue.mod/module.cue`).

#### Scenario: The cueCtx and coreVersion of synth.Instance

- **WHEN** `Kernel.SynthesizeInstance` calls `synth.Instance`
- **THEN** `cueCtx` is the context created for the call, and `coreVersion` is the kernel's core release, from which the synthesized `core` import takes its major

### Requirement: The Input parameter of synth.Instance is the package-internal input
`synth.Instance`: `Input` SHALL be the package-internal input carrying `Module`, `Name`, `Namespace`, the unified `Values`, `Labels`, `Annotations` and the kernel's registry environment.

#### Scenario: The Input of synth.Instance

- **WHEN** `Kernel.SynthesizeInstance` calls `synth.Instance`
- **THEN** `Input` is the package-internal input

### Requirement: Synthesis constructs the instance inside the acquired module's staged source tree
`SynthesizeInstance` SHALL construct the instance inside the acquired module's staged source tree (the overlay and root produced by `AcquireModuleFromRegistry` or `AcquireModuleFromDir`).

#### Scenario: Module acquired with source synthesizes

- **WHEN** `SynthesizeInstance` is called with a `Module` from `AcquireModuleFromRegistry`
- **THEN** the instance is staged inside the module's source tree and synthesis proceeds

#### Scenario: Module acquired from a directory synthesizes

- **WHEN** `SynthesizeInstance` is called with a `Module` from `AcquireModuleFromDir` on a module root
- **THEN** the instance is staged inside the module's overlay, the module import resolves locally, and the module's own `cue.mod/module.cue` drives transitive resolution

### Requirement: A module without staged source is refused and the kernel fetches nothing itself
`SynthesizeInstance` SHALL construct the instance inside the acquired module's staged source tree (the overlay and root produced by `AcquireModuleFromRegistry` or `AcquireModuleFromDir`). When `InstanceInput.Module` carries no staged source (it was constructed from a bare value), the call SHALL fail with an error wrapping `oerrors.ErrMissingSource` naming the two acquire verbs. The kernel SHALL NOT perform a registry fetch or a directory walk of its own to obtain the module source.

#### Scenario: Module without staged source is rejected

- **WHEN** `SynthesizeInstance` is called with a `Module` that carries no staged source
- **THEN** it returns a nil instance and an error wrapping `oerrors.ErrMissingSource`
- **AND** no registry fetch and no directory read occurs

### Requirement: A module acquired from a subdirectory of its CUE module is refused
`SynthesizeInstance` SHALL construct the instance inside the acquired module's staged source tree (the overlay and root produced by `AcquireModuleFromRegistry` or `AcquireModuleFromDir`). When the module's `Source.Pkg` is non-empty (the module was acquired from a subdirectory of its CUE module), the call SHALL fail with an error stating that a synthesizable module is its module's root package, because the synthesized package imports the module by its module path.

#### Scenario: Subpackage module is refused

- **WHEN** `SynthesizeInstance` is called with a `Module` acquired from a subdirectory of its CUE module
- **THEN** it returns an error naming the root-package requirement and runs no build

### Requirement: An existing synthetic root refuses a synthesis of a registry-acquired module
When the module was acquired from a registry, its staged root is a synthetic root that does not exist on disk; because cue/load reads a real directory beneath an overlay root, `synth.Instance` SHALL check that root before every build and SHALL fail, with an error naming the path, returning no tree and building nothing, when anything (a directory, a file or a symlink) exists there.

#### Scenario: An existing synthetic root refuses a synthesis

- **WHEN** a module is acquired from the registry, and afterwards a directory holding a `.cue` file exists at its synthetic root, and `synth.Instance` is called with that module
- **THEN** it returns an error naming the synthetic root and a nil tree, and runs no build
- **AND** once nothing exists at the root, the same call succeeds

### Requirement: A module acquired from a directory is not checked for a synthetic root
A module acquired from a directory has a real root on disk and SHALL NOT be checked. The check is that of requirement "An existing synthetic root refuses a synthesis of a registry-acquired module".

#### Scenario: A module acquired from a directory is not checked

- **WHEN** a module acquired from a directory is synthesized
- **THEN** its real root on disk is not checked

### Requirement: Synthesis stamps the staged source tree on the returned instance
`SynthesizeInstance` SHALL stamp on the returned instance the staged source tree the single build evaluated: `Root` = the acquired module's staged root, `Pkg` = the reserved instance package subdirectory, `Overlay` = the module's cloned overlay augmented with the synthesized instance files (`instance.cue`, and `values.cue` when values were supplied), every entry as bytes.

#### Scenario: The staged tree matches the build

- **WHEN** `SynthesizeInstance` succeeds
- **THEN** the returned instance's `Source.Root` equals the module's staged root, its `Pkg` names the reserved instance subdirectory, and its `Overlay` contains every file of the module's overlay plus the synthesized instance file (plus the values file when values were supplied)
- **AND** mutating the instance's overlay does not change the module's overlay

#### Scenario: Failure returns no tree

- **WHEN** `SynthesizeInstance` fails (missing inputs, missing source, build error, concreteness)
- **THEN** no instance and no staged tree is returned alongside the error

### Requirement: The stamped overlay is a clone
The overlay SHALL be a clone, so repeated synthesis from one module remains safe and the caller may retain the tree without aliasing the module's own overlay. The overlay is that of requirement "Synthesis stamps the staged source tree on the returned instance".

#### Scenario: The overlay is a clone

- **WHEN** the same module is synthesized repeatedly
- **THEN** the overlay is a clone, so repeated synthesis from one module remains safe and the caller may retain the tree without aliasing the module's own overlay

### Requirement: InstanceInput carries no schema cache and no cue.Context
`InstanceInput` SHALL carry no schema cache and no `cue.Context`: the kernel owns the schema cache, and synthesis builds in a context it creates for the call.

#### Scenario: InstanceInput holds no schema cache and no context

- **WHEN** a consumer inspects `InstanceInput`
- **THEN** it carries no schema cache and no `cue.Context`
- **AND** the kernel owns the schema cache, and synthesis builds in a context it creates for the call

### Requirement: A missing required field fails before any build runs
A missing required field SHALL fail with an error wrapping the matching `opm/errors` sentinel (`ErrMissingModule`, `ErrMissingName`, `ErrMissingNamespace`) before any build runs. The fields are those of requirement "Instance synthesis input".

#### Scenario: A missing required field fails before any build

- **WHEN** a required field of `InstanceInput` is missing
- **THEN** the call fails with an error wrapping the matching `opm/errors` sentinel (`ErrMissingModule`, `ErrMissingName`, `ErrMissingNamespace`) before any build runs

### Requirement: No public package exports a second synthesis entry point or input type
No public package under `opm/` SHALL export a second synthesis entry point or input type; the builder behind `SynthesizeInstance` lives in `opm/internal/synth`, which no consumer outside the library can import.

#### Scenario: No second synthesis entry point or input type

- **WHEN** a public package under `opm/` is inspected
- **THEN** it exports no second synthesis entry point or input type; the builder behind `SynthesizeInstance` lives in `opm/internal/synth`, which no consumer outside the library can import
