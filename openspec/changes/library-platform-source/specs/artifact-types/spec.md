## ADDED Requirements

### Requirement: Artifacts carry their staged source

`module.Source` SHALL describe the staged source tree an artifact was loaded or synthesized from, in two modes: an in-memory tree (`Overlay` non-empty, keyed under the synthetic absolute `Root`) or an on-disk tree (`Overlay` nil, `Root` a real directory). `module.Source` SHALL carry a `Pkg` field naming the package directory relative to `Root`; empty means the root package (`.`). `module.Instance` SHALL expose `Source *module.Source`, nil when the instance was constructed from a bare `cue.Value`. The existing `Module.HasSource()` gate semantics SHALL be unchanged.

#### Scenario: Value-constructed instance has no source

- **WHEN** a caller builds an instance via `NewInstanceFromValue`
- **THEN** the returned `*Instance` has `Source == nil`

#### Scenario: On-disk source mode

- **WHEN** an artifact's `Source` has `Overlay == nil` and a non-empty `Root`
- **THEN** consumers treat `Root` as a real filesystem directory holding the tree, and `Pkg` as the package directory within it

#### Scenario: Synth gate unchanged

- **WHEN** `synth.Instance` is invoked with a module whose `Source` is nil or overlay-empty
- **THEN** it still fails with `ErrMissingSource`, exactly as before this change

### Requirement: Instance acquisition from a directory

The kernel SHALL expose `AcquireInstanceFromDir`, which loads a `#ModuleInstance` CUE package from a directory through the existing shape-gated loader path, processes it through the validated entry point (concreteness enforced, metadata decoded, no extra values supplied), and returns a `*module.Instance` whose `Source` names the directory (`Root` = absolute directory, `Overlay` nil, root package).

#### Scenario: Acquired instance carries its source

- **WHEN** a caller invokes `AcquireInstanceFromDir` on a directory holding a valid, concrete instance package
- **THEN** the returned `*Instance` has decoded `Metadata`, a `Package` identical to what `LoadInstancePackage` returns for that directory, and `Source.Root` equal to the directory's absolute path with a nil `Overlay`

#### Scenario: Validation failures propagate

- **WHEN** the directory's instance package is not fully concrete
- **THEN** `AcquireInstanceFromDir` returns the same validation error the validated entry point produces, and no partial `*Instance`

#### Scenario: Loader failures propagate

- **WHEN** the directory does not exist, holds no CUE package, or fails the instance shape gate
- **THEN** the error wraps the same sentinel the file loader reports today (`ErrInvalidPackage`, `ErrWrongKind`, or `ErrMissingRequiredField`)
