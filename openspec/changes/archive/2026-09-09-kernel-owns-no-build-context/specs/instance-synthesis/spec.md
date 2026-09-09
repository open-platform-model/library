## MODIFIED Requirements

### Requirement: Instance synthesis input

Instance synthesis SHALL be reached only through `Kernel.SynthesizeInstance(ctx, kernel.InstanceInput)`. `InstanceInput` SHALL carry `Module *module.Module` (required, source-carrying), `Name string` (required), `Namespace string` (required), `Values []Source` (optional; empty means "no values supplied"), `Labels map[string]string` (optional) and `Annotations map[string]string` (optional). It SHALL carry no schema cache and no `cue.Context`: the kernel owns the schema cache, and synthesis builds in a context it creates for the call. A missing required field SHALL fail with an error wrapping the matching `opm/errors` sentinel (`ErrMissingModule`, `ErrMissingName`, `ErrMissingNamespace`) before any build runs. No package under `opm/` SHALL export a second synthesis entry point or input type.

#### Scenario: Required inputs validated

- **WHEN** `SynthesizeInstance` is called with `Module == nil`, or `Name == ""`, or `Namespace == ""`
- **THEN** it returns a nil instance and an error wrapping the sentinel that names the missing field

#### Scenario: One synthesis entry point

- **WHEN** a consumer inspects the exported identifiers of every package under `opm/`
- **THEN** `Kernel.SynthesizeInstance` and `kernel.InstanceInput` are the only exported synthesis symbols; no `synth` package exists

#### Scenario: Returned value is schema-unified

- **WHEN** `SynthesizeInstance` is called with valid inputs
- **THEN** the returned instance's `Package` carries the `#ModuleInstance` shape at its root, unified with the schema's `#ModuleInstance` definition resolved through the module's own `cue.mod/module.cue`
