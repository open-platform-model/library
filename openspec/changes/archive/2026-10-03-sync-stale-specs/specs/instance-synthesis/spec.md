## MODIFIED Requirements

### Requirement: synth.Instance constructs the instance by single-build CUE evaluation

`synth.Instance` SHALL construct the `#ModuleInstance` value by synthesizing an in-memory CUE package and evaluating it in a **single build** (per ADR-006), through the same build-and-shape-gate step a directory-acquired instance package runs (`loader.LoadDir` with the instance shape gate, the step behind `Kernel.AcquireInstanceFromDir`). The instance source SHALL be overlaid **into the acquired module's own staged source tree** (under a reserved synthetic subdirectory of the module's staged root), so the module's published, already-tidied `cue.mod/module.cue` is the build's main-module module file and drives all transitive dependency resolution. `synth.Instance` SHALL NOT fabricate a `cue.mod/module.cue` and SHALL NOT declare a dependency set; it reuses the module's tidied deps verbatim.

The instance source SHALL **import** the module's own package by its module path and write `#module: <import>` plus caller-supplied `metadata`, and SHALL import `core` (resolved from the module's own deps) to embed `#ModuleInstance`. Because the instance is built inside the module's main module, the module import resolves **locally** (no fabricated dependency, no registry round-trip for the module itself). A values file rendered from the input's `Values` (the unified value of the caller's `InstanceInput.Values` sources, which `Kernel.SynthesizeInstance` compiles and unifies before the call) SHALL be overlaid alongside the instance source when that value exists.

`synth.Instance` SHALL NOT inject the module via `cue.Scope` / a `userModule` field, and SHALL NOT pre-merge `Values` into the module's `#config` in Go. The values merge SHALL be performed by the schema in CUE (`#ModuleInstance`'s `unifiedModule = #module & {#config: values}`). The Go code SHALL fill only caller-supplied inputs and let CUE derive `metadata.uuid`, `components`, `opm-secrets`, and stamped labels.

`synth.Instance` lives in `opm/internal/synth`, off the public surface; its Go signature is `Instance(cueCtx *cue.Context, coreVersion string, in Input) (cue.Value, *module.Source, error)`. `cueCtx` is the context `Kernel.SynthesizeInstance` created for the call, `coreVersion` is the kernel's core release, from which the synthesized `core` import takes its major (the release the import resolves to comes from the module's own `cue.mod/module.cue`), and `Input` is the package-internal input carrying `Module`, `Name`, `Namespace`, the unified `Values`, `Labels`, `Annotations` and the kernel's registry environment.

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

### Requirement: Instance synthesis input

Instance synthesis SHALL be reached only through `Kernel.SynthesizeInstance(ctx, kernel.InstanceInput)`. `InstanceInput` SHALL carry `Module *module.Module` (required, source-carrying), `Name string` (required), `Namespace string` (required), `Values []Source` (optional; empty means "no values supplied"), `Labels map[string]string` (optional) and `Annotations map[string]string` (optional). It SHALL carry no schema cache and no `cue.Context`: the kernel owns the schema cache, and synthesis builds in a context it creates for the call. A missing required field SHALL fail with an error wrapping the matching `opm/errors` sentinel (`ErrMissingModule`, `ErrMissingName`, `ErrMissingNamespace`) before any build runs. No public package under `opm/` SHALL export a second synthesis entry point or input type; the builder behind `SynthesizeInstance` lives in `opm/internal/synth`, which no consumer outside the library can import.

#### Scenario: Required inputs validated

- **WHEN** `SynthesizeInstance` is called with `Module == nil`, or `Name == ""`, or `Namespace == ""`
- **THEN** it returns a nil instance and an error wrapping the sentinel that names the missing field

#### Scenario: One synthesis entry point

- **WHEN** a consumer inspects the exported identifiers of every package under `opm/` outside `opm/internal/`
- **THEN** `Kernel.SynthesizeInstance` and `kernel.InstanceInput` are the only exported synthesis symbols; no public `synth` package exists, and `opm/internal/synth` is importable only from inside the library

#### Scenario: Returned value is schema-unified

- **WHEN** `SynthesizeInstance` is called with valid inputs
- **THEN** the returned instance's `Package` carries the `#ModuleInstance` shape at its root, unified with the schema's `#ModuleInstance` definition resolved through the module's own `cue.mod/module.cue`
