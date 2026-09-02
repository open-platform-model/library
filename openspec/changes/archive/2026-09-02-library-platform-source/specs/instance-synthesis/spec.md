## ADDED Requirements

### Requirement: Synthesis surfaces its staged tree

`synth.Instance` SHALL return, alongside the evaluated instance value, the staged source tree the single build evaluated: `Root` = the acquired module's staged root, `Pkg` = the reserved instance package subdirectory, `Overlay` = the module's cloned overlay augmented with the synthesized instance files (`instance.cue`, and `values.cue` when values were supplied). The returned overlay SHALL be the clone the build used, so repeated synthesis from one module remains safe and the caller may retain the tree without aliasing the module's own overlay.

#### Scenario: The staged tree matches the build

- **WHEN** `synth.Instance` succeeds
- **THEN** the returned tree's `Root` equals the module's staged source root, its `Pkg` names the reserved instance subdirectory, and its `Overlay` contains every file of the module's overlay plus the synthesized instance file (plus the values file when values were supplied)

#### Scenario: Failure returns no tree

- **WHEN** `synth.Instance` fails (missing inputs, schema unavailable, build error)
- **THEN** no staged tree is returned alongside the error

### Requirement: Synthesized instances carry their source

`Kernel.SynthesizeInstance` SHALL stamp the staged tree returned by `synth.Instance` onto the returned `*module.Instance.Source`. Its signature and validation behavior SHALL be unchanged: it still chains the validated entry point, and a caller that ignores `Source` observes exactly the prior behavior.

#### Scenario: Synthesized instance carries the tree

- **WHEN** a caller invokes `Kernel.SynthesizeInstance` successfully
- **THEN** the returned `*Instance` has a non-nil `Source` whose `Overlay` holds the synthesized instance package inside the module's staged root

#### Scenario: Behavior otherwise unchanged

- **WHEN** an existing caller uses the returned instance without reading `Source`
- **THEN** metadata, package value, validation outcomes and error surfaces are identical to the behavior before this change
