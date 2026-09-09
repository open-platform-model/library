## MODIFIED Requirements

### Requirement: The render module's dependency list is derived by promotion

The generated render module's dependency list SHALL be derived by promotion: the platform module's tidied dependency list adopted whole, the instance module's list unioned in for paths only the instance carries, and the platform's entry winning every shared path. No tidy-equivalent and no registry consultation SHALL run at render time to compute the list. The two input trees SHALL enter the build through build-local directory replacements; neither input is fetched from a registry. Each input module's own path SHALL be listed in the render module's `cue.mod/module.cue` with a placeholder version of its own major and marked as the default major (unless a promoted entry already marks another major of the same root path default), so an unqualified import of the input's own subpackage from inside it resolves through the main module's default-major table exactly as it does when the input is the main module.

When local replacements are enabled for the render, each input's own `cue.mod/local-module.cue` replacements SHALL be promoted under the same precedence as its dependencies: the platform's replacements whole, the instance's only for paths the platform's dependency list does not name. A promoted replacement SHALL be written into the render module's main-module view exactly as the input wrote it, except that a relative directory target SHALL be resolved against that input's own module root, since the render module's root is elsewhere. A replaced path the promoted dependency list does not carry SHALL be listed with a placeholder version of its major, so the coverage invariant holds for a replaced OPM-namespace path. An input dependency that carries no version and is covered by no promoted replacement SHALL be refused before staging with an error naming the path and the input.

#### Scenario: Platform wins a shared path

- **WHEN** the instance module's `cue.mod` requires catalog build `1.3.0` and the platform module's `cue.mod` carries `1.2.0` for the same path
- **THEN** the render module lists `1.2.0`, and the build evaluates the platform's catalog bytes

#### Scenario: Instance-only paths survive

- **WHEN** the instance module depends on a path the platform module does not carry
- **THEN** the render module lists the instance's entry for that path, and the instance's import resolves

#### Scenario: An input's unqualified self-import resolves

- **WHEN** the instance module's root package imports one of its own subpackages without a major qualifier (an `identity` package, the shape `opm module init` writes) and the instance enters the build through a directory replacement
- **THEN** the render module marks the instance module's major default, and the build resolves the import from the replaced directory

#### Scenario: A platform replacement of its catalog is honoured

- **WHEN** local replacements are enabled, the platform module's `cue.mod/local-module.cue` replaces its catalog path with a directory, and that directory holds a catalog whose transformer output differs from the published build
- **THEN** the render module's main-module view replaces the catalog path with that directory, the build evaluates the directory's transformer bytes, and the render's replacement rows name the path, the directory and the platform as its source

#### Scenario: An instance replacement on a platform-named path is inert

- **WHEN** local replacements are enabled and the instance module's `cue.mod/local-module.cue` replaces a path the platform module's dependency list carries
- **THEN** the render module carries no replacement for that path, the build evaluates the platform's pinned bytes, and no replacement row names the path

#### Scenario: An instance replacement on an instance-only path is honoured

- **WHEN** local replacements are enabled and the instance module's `cue.mod/local-module.cue` replaces a path the platform module does not carry with a relative directory
- **THEN** the render module's main-module view replaces that path with the directory resolved against the instance module's root, the build resolves the instance's import from it, and a replacement row names the path, the absolute directory and the instance as its source

#### Scenario: A replace-only dependency is listed with a placeholder

- **WHEN** local replacements are enabled and the instance module's `cue.mod/module.cue` lists a dependency with no version that its `cue.mod/local-module.cue` replaces with a directory
- **THEN** the render module's `cue.mod/module.cue` lists the path with a placeholder version of its major, the main-module view replaces it, and the render proceeds

#### Scenario: A version-less dependency without a replacement is refused

- **WHEN** an input's `cue.mod/module.cue` lists a dependency with no version and no promoted replacement covers the path
- **THEN** `Render` fails before staging with an error naming the path and the input, and no modfile formatting error is surfaced

## ADDED Requirements

### Requirement: Local replacements are honoured only when the caller opts in

`Kernel.Render` SHALL accept a per-render opt-in for local replacements, off by default. When it is off, an input whose `cue.mod/local-module.cue` carries at least one replacement SHALL be refused before staging with an error naming the input and the file; the file SHALL NOT be silently ignored. When it is on, promoted replacements SHALL be reported on the render diagnostics as rows carrying the replaced path, the absolute target (a directory, or a module path for a module replacement) and the input that supplied it, in path order; the kernel SHALL NOT render them as message strings. An input without the file SHALL behave identically under either setting.

#### Scenario: Replacement without opt-in refused

- **WHEN** `Render` is invoked with the opt-in off and the platform module's `cue.mod/local-module.cue` replaces its catalog path
- **THEN** it returns an error naming the platform and `cue.mod/local-module.cue`, and no staging directory is written

#### Scenario: Inputs without the file are unaffected

- **WHEN** `Render` is invoked with the opt-in on and neither input carries `cue.mod/local-module.cue`
- **THEN** the render's output and diagnostics are byte-identical to the same render with the opt-in off, and the replacement rows are empty

#### Scenario: Replacement rows are data

- **WHEN** a render with the opt-in on honours one platform replacement and one instance replacement
- **THEN** the result's diagnostics carry exactly two replacement rows, sorted by path, each naming its source input, and no field of the result holds a formatted message about them
