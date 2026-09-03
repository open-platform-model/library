## MODIFIED Requirements

### Requirement: Instance acquisition from a directory

The kernel SHALL expose `AcquireInstanceFromDir`, which loads a `#ModuleInstance` CUE package from a directory through the existing shape-gated loader path, processes it through the validated entry point (concreteness enforced, metadata decoded, no extra values supplied), and returns a `*module.Instance` whose `Source` describes the directory in on-disk mode: `Overlay` nil, `Root` = the absolute path of the enclosing module root (the nearest ancestor holding `cue.mod/module.cue`, the directory itself when it is the root), and `Pkg` = the package directory relative to `Root` (empty for the root package). A directory with no enclosing module is its own root with an empty `Pkg`. This is what lets a package in a subdirectory of its module be imported correctly by a follow-on build such as `Kernel.Render`.

#### Scenario: Acquired instance carries its source

- **WHEN** a caller invokes `AcquireInstanceFromDir` on a module root directory holding a valid, concrete instance package
- **THEN** the returned `*Instance` has decoded `Metadata`, a `Package` identical to what `LoadInstancePackage` returns for that directory, `Source.Root` equal to the directory's absolute path, an empty `Source.Pkg`, and a nil `Overlay`

#### Scenario: Acquired subpackage names its module root

- **WHEN** a caller invokes `AcquireInstanceFromDir` on a subdirectory of a module (its `cue.mod/module.cue` lives in an ancestor)
- **THEN** the returned `*Instance` has `Source.Root` equal to the module root's absolute path and `Source.Pkg` equal to the subdirectory's slash-separated path relative to it

#### Scenario: Validation failures propagate

- **WHEN** the directory's instance package is not fully concrete
- **THEN** `AcquireInstanceFromDir` returns the same validation error the validated entry point produces, and no partial `*Instance`

#### Scenario: Loader failures propagate

- **WHEN** the directory does not exist, holds no CUE package, or fails the instance shape gate
- **THEN** the error wraps the same sentinel the file loader reports today (`ErrInvalidPackage`, `ErrWrongKind`, or `ErrMissingRequiredField`)
