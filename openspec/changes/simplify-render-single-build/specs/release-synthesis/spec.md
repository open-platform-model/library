## MODIFIED Requirements

### Requirement: synth.Release constructs the release by single-build CUE evaluation

`synth.Release` SHALL construct the `#ModuleRelease` value by synthesizing an in-memory CUE package and evaluating it in a **single build** (per ADR-003), through the same loader build path used by `LoadReleasePackage`. The synthesized package SHALL consist of a fabricated `cue.mod/module.cue` declaring the dependencies (the resolved `opmodel.dev/core` version and the module's `path@version`), a release source that **imports** the module and writes `#module: <import>` plus caller-supplied `metadata`, and a values source rendered from `ReleaseInput.Values`.

`synth.Release` SHALL NOT inject the module via `cue.Scope` / a `userModule` field, and SHALL NOT pre-merge `Values` into the module's `#config` in Go. The values merge SHALL be performed by the schema in CUE (`#ModuleRelease`'s `unifiedModule = #module & {#config: values}`). The Go code SHALL fill only caller-supplied inputs and let CUE derive `metadata.uuid`, `components`, `opm-secrets`, and stamped labels.

The public Go signature `Release(ctx *cue.Context, in ReleaseInput) (cue.Value, error)` and the `ReleaseInput` field set SHALL be unchanged. For every input that succeeds today, the returned value SHALL be observably equivalent (same `metadata.uuid`, same `components`, same labels).

#### Scenario: No scope or Go pre-merge in the construction path

- **WHEN** `synth.Release` is called with valid inputs
- **THEN** the construction performs no `cue.Scope`-based compile and no `FillPath` of `Values` into the module's `#config`
- **AND** the returned value carries the `#ModuleRelease` shape with `components` fanned by the schema comprehension

#### Scenario: Values merged in-build by the schema

- **WHEN** `synth.Release` is called with `Values` satisfying the module's `#config`
- **THEN** the returned value's `components` reflect the unified `#config = values` configuration
- **AND** the merge is produced by CUE evaluation of the synthesized package, not by a Go-side `#config` fill

#### Scenario: Imported-module construction succeeds

- **WHEN** `synth.Release` is called with a module whose identity is author-supplied (`metadata.modulePath` / `metadata.version` concrete) against a resolved `core` schema exposing author-supplied `#Module` identity
- **THEN** the release constructs without a `field not allowed` admission error
- **AND** the value unifies cleanly against `#ModuleRelease`

#### Scenario: Single-build parity with an authored package

- **WHEN** `synth.Release` builds a release for module M with values V, and an authored `release.cue` package imports the same M and sets the same V
- **THEN** both, passed through `Kernel.Compile`, produce the same set of compiled resources

### Requirement: Values field is caller-supplied with no implicit fallback

`synth.Release` SHALL NOT consult `Module.debugValues` or any other implicit source when `ReleaseInput.Values` is the zero `cue.Value`. When `Values.Exists()` is true, the helper SHALL render it into the synthesized package's values source (via `format.Node` on the value's syntax, never string-interpolating raw input) so it participates in the single build. When `Values.Exists()` is false, the helper SHALL omit the values source and return the unified value as-is; concreteness enforcement is deferred to `Kernel.ProcessModuleRelease`.

#### Scenario: Caller-supplied values participate in the build

- **WHEN** `synth.Release` is called with `Values` set to a concrete CUE value satisfying the module's `#config`
- **THEN** the returned value carries those values at the schema's values path
- **AND** the values entered the build as a rendered source file, not a post-build cross-build unification

#### Scenario: Zero Values is not replaced by debugValues

- **WHEN** `synth.Release` is called with `Values == cue.Value{}` against a Module that defines `debugValues`
- **THEN** the returned value's values path is unfilled (does not equal `debugValues`)

## ADDED Requirements

### Requirement: Release construction shares one evaluate-and-shape-gate with the file loader

The library SHALL provide a single build-and-validate routine that both `synth.Release` (in-memory overlay source) and `LoadReleasePackage` (on-disk source) invoke to evaluate a release package and run the shape gate. The two entry points SHALL differ only in how the package source is supplied (overlay vs. filesystem directory); the evaluation, shape-gating, and error wrapping SHALL be identical.

#### Scenario: Overlay and on-disk releases evaluate identically

- **WHEN** the same release content is supplied once as an in-memory overlay (via `synth.Release`) and once as an on-disk package (via `LoadReleasePackage`)
- **THEN** both produce a value of the same shape passing the same release shape gate
- **AND** a malformed release fails the shape gate identically in both paths

### Requirement: Imported-module render coverage exists

The library SHALL include a test that renders a release whose module is referenced by import (not inlined) end-to-end through construction and `Kernel.Compile`, producing concrete resources. This coverage SHALL exist for both the synth path and an authored-package path so that a regression in either surfaces.

#### Scenario: Real imported module renders to resources

- **WHEN** a release referencing a published module by import is rendered through the kernel
- **THEN** the compile output contains the module's expected resources
- **AND** the test fails if import-based construction regresses to a `field not allowed` admission error
