## MODIFIED Requirements

### Requirement: synth.Instance constructs the instance by single-build CUE evaluation

`synth.Instance` SHALL construct the `#ModuleInstance` value by synthesizing an in-memory CUE package and evaluating it in a **single build** (per ADR-003), through the same loader build path used by `LoadInstancePackage`. The instance source SHALL be overlaid **into the acquired module's own staged source tree** (under a reserved synthetic subdirectory of the module's staged root), so the module's published, already-tidied `cue.mod/module.cue` is the build's main-module module file and drives all transitive dependency resolution. `synth.Instance` SHALL NOT fabricate a `cue.mod/module.cue` and SHALL NOT declare a dependency set; it reuses the module's tidied deps verbatim.

The instance source SHALL **import** the module's own package by its module path and write `#module: <import>` plus caller-supplied `metadata`, and SHALL import `core` (resolved from the module's own deps) to embed `#ModuleInstance`. Because the instance is built inside the module's main module, the module import resolves **locally** (no fabricated dependency, no registry round-trip for the module itself). A values source rendered from `InstanceInput.Values` SHALL be overlaid alongside the instance source when `Values.Exists()`.

`synth.Instance` SHALL NOT inject the module via `cue.Scope` / a `userModule` field, and SHALL NOT pre-merge `Values` into the module's `#config` in Go. The values merge SHALL be performed by the schema in CUE (`#ModuleInstance`'s `unifiedModule = #module & {#config: values}`). The Go code SHALL fill only caller-supplied inputs and let CUE derive `metadata.uuid`, `components`, `opm-secrets`, and stamped labels.

The public Go signature `Instance(ctx *cue.Context, in InstanceInput) (cue.Value, error)` and the `InstanceInput` field set SHALL be unchanged. For every input that succeeds today, the returned value SHALL be observably equivalent (same `metadata.uuid`, same `components`, same labels).

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
- **THEN** both, passed through `Kernel.Compile`, produce the same set of compiled resources

### Requirement: Imported-module render coverage exists

The library SHALL include a test that renders an instance whose module is referenced by import (not inlined) end-to-end through construction and `Kernel.Compile`, producing concrete resources. This coverage SHALL exist for both the synth path and an authored-package path so that a regression in either surfaces. The synth-path coverage SHALL include a module that imports a **catalog subpackage** (e.g. a workload blueprint under `opmodel.dev/catalogs/opm/...`), so that a regression to a dependency-incomplete synthesis surfaces as a failing test.

#### Scenario: Real imported module renders to resources

- **WHEN** an instance referencing a published module by import is rendered through the kernel
- **THEN** the compile output contains the module's expected resources
- **AND** the test fails if import-based construction regresses to a `field not allowed` admission error

#### Scenario: Module importing a catalog subpackage synthesizes and renders

- **WHEN** `synth.Instance` is called with a published module whose source imports a transitive catalog subpackage (the library#31 shape), against a registry serving that catalog
- **THEN** synthesis succeeds without `cannot find module providing package opmodel.dev/catalogs/opm/...`
- **AND** the instance renders to the module's expected resources through `Kernel.Compile`

## ADDED Requirements

### Requirement: synth.Instance requires the module's staged source

`synth.Instance` SHALL construct the instance from the acquired module's staged source (the overlay + synthetic root produced when the module is fetched from the registry). When the supplied `InstanceInput.Module` does not carry staged source (it was constructed from a bare value rather than acquired with source), `synth.Instance` SHALL return the zero `cue.Value` and a non-nil error explaining that the module must be acquired with its source (e.g. via the source-carrying registry acquire entrypoint). `synth.Instance` SHALL NOT itself perform a registry fetch to obtain the module source.

#### Scenario: Module without staged source is rejected

- **WHEN** `synth.Instance` is called with a `Module` that carries no staged source
- **THEN** it returns the zero `cue.Value` and a non-nil error naming the missing source precondition
- **AND** `synth.Instance` performs no registry fetch of its own

#### Scenario: Module acquired with source synthesizes

- **WHEN** `synth.Instance` is called with a `Module` acquired through the source-carrying registry path
- **THEN** the instance is staged inside the module's source tree and synthesis proceeds

### Requirement: Transitive module dependencies resolve via the module's own module file

Because the instance is built inside the module's staged main module, `synth.Instance` SHALL resolve the module's transitive dependencies (core, catalog packages, and any further indirect deps) through the module's own `cue.mod/module.cue`, identically to how `registry-module-loading` resolves them when loading the module standalone. The caller SHALL NOT be required to declare, derive, or tidy the module's transitive closure for synthesis to succeed.

#### Scenario: Indirect catalog dependency resolves without caller declaration

- **WHEN** a module's source imports a package whose own dependency is an indirect catalog module not directly imported by the module
- **THEN** `synth.Instance` resolves that indirect dependency via the module's tidied `cue.mod/module.cue`
- **AND** synthesis succeeds without the caller supplying the transitive closure
