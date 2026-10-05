## MODIFIED Requirements

### Requirement: synth.Instance requires the module's staged source

`SynthesizeInstance` SHALL construct the instance inside the acquired module's staged source tree (the overlay and root produced by `AcquireModuleFromRegistry` or `AcquireModuleFromDir`). When `InstanceInput.Module` carries no staged source (it was constructed from a bare value), the call SHALL fail with an error wrapping `oerrors.ErrMissingSource` naming the two acquire verbs. When the module's `Source.Pkg` is non-empty (the module was acquired from a subdirectory of its CUE module), the call SHALL fail with an error stating that a synthesizable module is its module's root package, because the synthesized package imports the module by its module path. The kernel SHALL NOT perform a registry fetch or a directory walk of its own to obtain the module source. When the module was acquired from a registry, its staged root is a synthetic root that does not exist on disk; because cue/load reads a real directory beneath an overlay root, `synth.Instance` SHALL check that root before every build and SHALL fail, with an error naming the path, returning no tree and building nothing, when anything (a directory, a file or a symlink) exists there. A module acquired from a directory has a real root on disk and is not checked.

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

#### Scenario: An existing synthetic root refuses a synthesis

- **WHEN** a module is acquired from the registry, and afterwards a directory holding a `.cue` file exists at its synthetic root, and `synth.Instance` is called with that module
- **THEN** it returns an error naming the synthetic root and a nil tree, and runs no build
- **AND** once nothing exists at the root, the same call succeeds
