## MODIFIED Requirements

### Requirement: Values field is caller-supplied with no implicit fallback

`synth.Instance` SHALL NOT consult `Module.debugValues` or any other implicit source when `InstanceInput.Values` is the zero `cue.Value`. When `Values.Exists()` is true, the helper SHALL render it into the synthesized package's values source (via `format.Node` on the value's syntax, never string-interpolating raw input) so it participates in the single build. When `Values.Exists()` is false, the helper SHALL omit the values source and return the unified value as-is; concreteness enforcement is deferred to the kernel's instance processing behind `Kernel.SynthesizeInstance`.

#### Scenario: Caller-supplied values participate in the build

- **WHEN** `synth.Instance` is called with `Values` set to a concrete CUE value satisfying the module's `#config`
- **THEN** the returned value carries those values at the schema's values path
- **AND** the values entered the build as a rendered source file, not a post-build cross-build unification

#### Scenario: Zero Values is not replaced by debugValues

- **WHEN** `synth.Instance` is called with `Values == cue.Value{}` against a Module that defines `debugValues`
- **THEN** the returned CUE value's values path is unfilled (does not equal `debugValues`)

### Requirement: synth.Instance does not validate or enforce concreteness

`synth.Instance` SHALL return the unified CUE value without invoking `cue.Concrete` validation. Concreteness enforcement on the final spec and metadata decoding are downstream responsibilities of the kernel's instance processing, reached through `Kernel.SynthesizeInstance`; the values merge against `#config` happens inside the build. Errors from CUE during unification (e.g. type mismatch between caller-supplied labels and the schema's label-map type) SHALL be returned as the result of `cue.Value.Err()` on the returned value, surfaced to the caller through the returned `error`.

#### Scenario: Unification error returned

- **WHEN** `synth.Instance` is called with inputs that conflict with the schema (e.g. `Name` containing characters disallowed by `#NameType`)
- **THEN** the returned error is non-nil and the returned `cue.Value` is the zero value or carries the unification error

#### Scenario: No concreteness check at synth time

- **WHEN** `synth.Instance` is called with `Values == cue.Value{}` against a `#config` that has no defaults
- **THEN** the call succeeds (returns a non-zero `cue.Value` and a nil error)
- **AND** the returned value's values path is unfilled rather than concrete
