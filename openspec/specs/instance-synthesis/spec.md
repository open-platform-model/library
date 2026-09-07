# instance-synthesis Specification

## Purpose
TBD - created by archiving change rename-release-to-instance. Update Purpose after archive.
## Requirements

### Requirement: Derived fields come from schema unification

`synth.Instance` SHALL NOT compute `metadata.uuid`, `components`, or schema-stamped labels in Go. These fields SHALL flow from CUE evaluation as a consequence of unifying the inputs with `#ModuleInstance`. The Go code SHALL fill only the caller-supplied fields (name, namespace, `#module`, optional values/labels/annotations) and let CUE derive the rest.

#### Scenario: UUID is computed by CUE

- **WHEN** `synth.Instance` is called twice with the same `(Module, Name, Namespace)`
- **THEN** the returned CUE values carry identical `metadata.uuid` strings
- **AND** the UUID equals `uuid.SHA1(OPMNamespace, "<module.uuid>:<name>:<namespace>")` per core's `#ModuleInstance` schema (`core/src/module_instance.cue`)

#### Scenario: UUID diverges with namespace

- **WHEN** `synth.Instance` is called with identical `(Module, Name)` but two different `Namespace` values
- **THEN** the returned CUE values carry different `metadata.uuid` strings

#### Scenario: Components are fanned by schema comprehension

- **WHEN** `synth.Instance` is called with a Module declaring N concrete components in `#components`
- **THEN** the returned CUE value's `components` field contains exactly those N entries with the schema-applied projections
- **AND** the synth helper itself does not enumerate `#components` in Go

#### Scenario: Auto-secrets component included when module has #Secret instances

- **WHEN** `synth.Instance` is called with a Module whose `#config` (after `Values` is filled) contains at least one `#Secret` instance
- **THEN** the returned CUE value's `components` field contains an `opm-secrets` entry
- **AND** when the module contains no `#Secret` instances, no `opm-secrets` entry appears

#### Scenario: Standard instance labels are stamped by schema

- **WHEN** `synth.Instance` is called with valid inputs
- **THEN** the returned CUE value's `metadata.labels` contains keys `module-instance.opmodel.dev/name` and `module-instance.opmodel.dev/uuid`
- **AND** their values equal the instance name and the derived UUID respectively

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

### Requirement: Optional labels and annotations are filled into instance metadata

When `InstanceInput.Labels` is non-empty, `synth.Instance` SHALL fill `metadata.labels` with those entries. When `InstanceInput.Annotations` is non-empty, `synth.Instance` SHALL fill `metadata.annotations` with those entries. The schema's label stamping (Requirement: Derived fields come from schema unification) unifies with caller-supplied labels — caller labels MUST NOT be allowed to remove schema-stamped labels.

#### Scenario: Caller labels merged with schema-stamped labels

- **WHEN** `synth.Instance` is called with `Labels == {"env": "prod"}`
- **THEN** the returned CUE value's `metadata.labels` contains the `env: prod` entry
- **AND** still contains `module-instance.opmodel.dev/name` and `module-instance.opmodel.dev/uuid`

#### Scenario: Annotations are passed through unchanged

- **WHEN** `synth.Instance` is called with `Annotations == {"opmodel.dev/owner": "team-x"}`
- **THEN** the returned CUE value's `metadata.annotations` contains that entry

### Requirement: Instance construction shares one evaluate-and-shape-gate with the file loader

The kernel SHALL evaluate a synthesized instance package (in-memory overlay inside the module's staged tree) and a directory-acquired instance package (on-disk source) through one build-and-shape-gate routine. The two paths SHALL differ only in how the package source is supplied; the evaluation, shape gating, sentinel wrapping and values attribution SHALL be identical.

#### Scenario: Overlay and on-disk instances evaluate identically

- **WHEN** the same instance content is supplied once through `SynthesizeInstance` and once through `AcquireInstanceFromDir`
- **THEN** both produce a value of the same shape passing the same instance shape gate
- **AND** a malformed instance fails the shape gate identically in both paths, wrapping the same `opm/errors` sentinel

### Requirement: Imported-module render coverage exists

The library SHALL include a test that renders an instance whose module is referenced by import (not inlined) end-to-end through construction and `Kernel.Render`, against a D5-shaped platform (a platform module importing its catalog), producing concrete resources. This coverage SHALL exist for both the synth path and an authored-package path so that a regression in either surfaces. The synth-path coverage SHALL include a module that imports a **catalog subpackage** (e.g. a workload blueprint under `opmodel.dev/catalogs/opm/...`), so that a regression to a dependency-incomplete synthesis surfaces as a failing test.

#### Scenario: Real imported module renders to resources

- **WHEN** an instance referencing a published module by import is rendered through `Kernel.Render`
- **THEN** the rendered output contains the module's expected resources
- **AND** the test fails if import-based construction regresses to a `field not allowed` admission error

#### Scenario: Module importing a catalog subpackage synthesizes and renders

- **WHEN** `synth.Instance` is called with a published module whose source imports a transitive catalog subpackage (the library#31 shape), against a registry serving that catalog
- **THEN** synthesis succeeds without `cannot find module providing package opmodel.dev/catalogs/opm/...`
- **AND** the instance renders to the module's expected resources through `Kernel.Render`

#### Scenario: Single-build parity with an authored package

- **WHEN** `synth.Instance` builds an instance for module M with values V, and an authored `instance.cue` package imports the same M and sets the same V
- **THEN** both, passed through `Kernel.Render` against the same platform, produce the same rendered objects

### Requirement: synth.Instance constructs the instance by single-build CUE evaluation

`synth.Instance` SHALL construct the `#ModuleInstance` value by synthesizing an in-memory CUE package and evaluating it in a **single build** (per ADR-003), through the same loader build path used by `LoadInstancePackage`. The instance source SHALL be overlaid **into the acquired module's own staged source tree** (under a reserved synthetic subdirectory of the module's staged root), so the module's published, already-tidied `cue.mod/module.cue` is the build's main-module module file and drives all transitive dependency resolution. `synth.Instance` SHALL NOT fabricate a `cue.mod/module.cue` and SHALL NOT declare a dependency set; it reuses the module's tidied deps verbatim.

The instance source SHALL **import** the module's own package by its module path and write `#module: <import>` plus caller-supplied `metadata`, and SHALL import `core` (resolved from the module's own deps) to embed `#ModuleInstance`. Because the instance is built inside the module's main module, the module import resolves **locally** (no fabricated dependency, no registry round-trip for the module itself). A values source rendered from `InstanceInput.Values` SHALL be overlaid alongside the instance source when `Values.Exists()`.

`synth.Instance` SHALL NOT inject the module via `cue.Scope` / a `userModule` field, and SHALL NOT pre-merge `Values` into the module's `#config` in Go. The values merge SHALL be performed by the schema in CUE (`#ModuleInstance`'s `unifiedModule = #module & {#config: values}`). The Go code SHALL fill only caller-supplied inputs and let CUE derive `metadata.uuid`, `components`, `opm-secrets`, and stamped labels.

The public Go signature is `Instance(ctx *cue.Context, in InstanceInput) (cue.Value, *module.Source, error)` and the `InstanceInput` field set SHALL be unchanged. For every input that succeeds today, the returned value SHALL be observably equivalent (same `metadata.uuid`, same `components`, same labels).

#### Scenario: No scope or Go pre-merge in the construction path

- **WHEN** `synth.Instance` is called with valid inputs
- **THEN** the construction performs no `cue.Scope`-based compile and no `FillPath` of `Values` into the module's `#config`
- **AND** the returned value carries the `#ModuleInstance` shape with `components` fanned by the schema comprehension

#### Scenario: No fabricated module file

- **WHEN** `synth.Instance` constructs an instance
- **THEN** the build's main-module `cue.mod/module.cue` is the module's own published module file (carrying its full tidied dependency closure)
- **AND** `synth.Instance` does not generate a `cue.mod/module.cue` or a `deps:` block of its own

#### Scenario: Values merged in-build by the schema

- **WHEN** `synth.Instance` is called with `Values` satisfying the module's `#config`
- **THEN** the returned value's `components` reflect the unified `#config = values` configuration
- **AND** the merge is produced by CUE evaluation of the synthesized package, not by a Go-side `#config` fill

#### Scenario: Imported-module construction succeeds

- **WHEN** `synth.Instance` is called with a module whose identity is author-supplied (`metadata.modulePath` / `metadata.version` concrete)
- **THEN** the instance constructs without a `field not allowed` admission error
- **AND** the value unifies cleanly against `#ModuleInstance`

#### Scenario: Single-build parity with an authored package

- **WHEN** `synth.Instance` builds an instance for module M with values V, and an authored `instance.cue` package imports the same M and sets the same V
- **THEN** both, passed through `Kernel.Render`, produce the same set of rendered resources

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

### Requirement: Transitive module dependencies resolve via the module's own module file

Because the instance is built inside the module's staged main module, `synth.Instance` SHALL resolve the module's transitive dependencies (core, catalog packages, and any further indirect deps) through the module's own `cue.mod/module.cue`, identically to how `registry-module-loading` resolves them when loading the module standalone. The caller SHALL NOT be required to declare, derive, or tidy the module's transitive closure for synthesis to succeed.

#### Scenario: Indirect catalog dependency resolves without caller declaration

- **WHEN** a module's source imports a package whose own dependency is an indirect catalog module not directly imported by the module
- **THEN** `synth.Instance` resolves that indirect dependency via the module's tidied `cue.mod/module.cue`
- **AND** synthesis succeeds without the caller supplying the transitive closure

### Requirement: Synthesis surfaces its staged tree

`SynthesizeInstance` SHALL stamp on the returned instance the staged source tree the single build evaluated: `Root` = the acquired module's staged root, `Pkg` = the reserved instance package subdirectory, `Overlay` = the module's cloned overlay augmented with the synthesized instance files (`instance.cue`, and `values.cue` when values were supplied), every entry as bytes. The overlay SHALL be a clone, so repeated synthesis from one module remains safe and the caller may retain the tree without aliasing the module's own overlay.

#### Scenario: The staged tree matches the build

- **WHEN** `SynthesizeInstance` succeeds
- **THEN** the returned instance's `Source.Root` equals the module's staged root, its `Pkg` names the reserved instance subdirectory, and its `Overlay` contains every file of the module's overlay plus the synthesized instance file (plus the values file when values were supplied)
- **AND** mutating the instance's overlay does not change the module's overlay

#### Scenario: Failure returns no tree

- **WHEN** `SynthesizeInstance` fails (missing inputs, missing source, build error, concreteness)
- **THEN** no instance and no staged tree is returned alongside the error

### Requirement: Synthesized instances carry their source

`Kernel.SynthesizeInstance` SHALL stamp the staged tree returned by `synth.Instance` onto the returned `*module.Instance.Source`. Its signature and validation behavior SHALL be unchanged: it still chains the validated entry point, and a caller that ignores `Source` observes exactly the prior behavior.

#### Scenario: Synthesized instance carries the tree

- **WHEN** a caller invokes `Kernel.SynthesizeInstance` successfully
- **THEN** the returned `*Instance` has a non-nil `Source` whose `Overlay` holds the synthesized instance package inside the module's staged root

#### Scenario: Behavior otherwise unchanged

- **WHEN** an existing caller uses the returned instance without reading `Source`
- **THEN** metadata, package value, validation outcomes and error surfaces are identical to the behavior before this change

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
