## MODIFIED Requirements

### Requirement: Instance acquisition from a directory

The kernel SHALL expose `AcquireInstanceFromDir`, which loads a `#ModuleInstance` CUE package from a directory through the existing shape-gated loader path, processes it through the validated entry point (concreteness enforced, metadata decoded), and returns a `*module.Instance` whose `Source` describes the directory: `Root` = the absolute path of the enclosing module root (the nearest ancestor holding `cue.mod/module.cue`, the directory itself when it is the root), and `Pkg` = the package directory relative to `Root` (empty for the root package). A directory with no enclosing module is its own root with an empty `Pkg`. This is what lets a package in a subdirectory of its module be imported correctly by a follow-on build such as `Kernel.Render`.

With no option, no extra values are supplied and `Source.Overlay` is nil (on-disk mode). With the extra-values option, the caller supplies one or more values sources (the `Source` type `ValidateConfigDetailed` accepts); the kernel SHALL unify them, render the result as a values file (`opm-values.cue`, a reserved name so it can never shadow a file the package authored) declaring the package's own package name and the top-level `values` field, place it beside the package's on-disk files in an overlay, and build the package in one pass through the instance shape gate, so the schema's own values unification performs the merge in CUE. The returned instance's `Source` SHALL then be overlay mode: same `Root` and `Pkg`, `Overlay` carrying every on-disk `.cue` file under the module root (the files `cue/load` reads; the module's own `cue.mod/module.cue` included) plus the rendered values file, exactly as `load.Config.Overlay` expects, so `Kernel.Render` imports the layered package by source. The kernel MUST NOT write into the caller's directory and MUST NOT fill values into the evaluated value from Go.

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

#### Scenario: Extra values layer onto the package

- **WHEN** a caller invokes `AcquireInstanceFromDir` with two values sources on an instance package whose `values` leave a field unset
- **THEN** the returned instance's `values` carry the unified sources merged with the package's own values, `Source.Overlay` is non-nil and contains the on-disk `.cue` files plus the rendered values file, and the caller's directory is unchanged

#### Scenario: Layered instance renders

- **WHEN** an instance acquired with extra values is passed to `Kernel.Render`
- **THEN** the render build imports the layered package and the rendered objects reflect the extra values

#### Scenario: Conflicting extra values fail at acquisition

- **WHEN** an extra values source conflicts with the package's own values or its module's `#config` schema
- **THEN** `AcquireInstanceFromDir` returns the validation error naming the conflicting path, with source positions attributable to the values source (a conflict with the package's own values is re-validated as layered validation does; a `#config` violation is checked on the sources after the build), and no partial `*Instance`
