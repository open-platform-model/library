## REMOVED Requirements

### Requirement: Test module pins update in one resolution pass
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The update task updates each test module with one cue mod get that names every direct opmodel.dev dependency, followed by one cue mod tidy" and "A non-core OPM dependency is requested at its newest version within its major and a third-party dependency is not named in the get".

### Requirement: Core is held at the kernel default
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The update task requests core at the release the kernel default names and core moves only together with the default", "A held resolution that fails because a dependency requires a newer core exits non-zero and names the dependency" and "The update task exits non-zero when the default cannot be read from the Go source".

### Requirement: Intentionally old version literals are declared
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The repository root carries a .cascade-frozen file listing every test file that keeps a version literal on purpose", "A file qualifies for the frozen list when its literal names an older release a test needs or is synthetic", "An expectation of the current core or of the pinned catalog build is not a frozen literal" and "Release-cascade tooling does not rewrite a version literal at a listed location".

## ADDED Requirements

### Requirement: The update task updates each test module with one cue mod get that names every direct opmodel.dev dependency, followed by one cue mod tidy
The library's dependency-update task SHALL update every test CUE module it discovers (the `CUE_MODULE_GLOBS` set) with one `cue mod get` per module. That one call SHALL name every direct `opmodel.dev/*` dependency of the module, followed by one `cue mod tidy`. A module whose OPM dependencies are several releases stale SHALL therefore update in a single run.

#### Scenario: Nothing to update

- **WHEN** every test module already pins the newest release of each dependency
- **THEN** the task exits zero, leaves every `cue.mod/module.cue` byte-identical, and reports each dependency unchanged

### Requirement: A non-core OPM dependency is requested at its newest version within its major and a third-party dependency is not named in the get
Each non-core OPM dependency SHALL be requested at the newest published version within its major. A third-party dependency (such as `cue.dev/x/k8s.io`) SHALL NOT be named in the get (Requirement: The update task updates each test module with one cue mod get that names every direct opmodel.dev dependency, followed by one cue mod tidy); it moves only when tidy raises it to what an OPM dependency requires.

#### Scenario: A newer catalog raises a third-party pin only as far as it requires

- **WHEN** a test module pins an older catalog build, and the newest catalog build requires a newer `cue.dev/x/k8s.io` build than the module pins
- **THEN** one run of the task moves the catalog to its newest build, tidy raises `cue.dev/x/k8s.io` only to the version that catalog build requires, and the task prints an old-to-new line for each

#### Scenario: A third-party pin is not chased

- **WHEN** a test module's `cue.dev/x/k8s.io` pin already satisfies every OPM dependency, and a newer `cue.dev/x/k8s.io` release exists
- **THEN** the task leaves that pin unchanged

### Requirement: The update task requests core at the release the kernel default names and core moves only together with the default
The update task SHALL request `opmodel.dev/core@v2` at the release `schema.DefaultSchemaModule` names, read from the Go source unless an explicit override names another release, never at the newest published core. Core in the test modules SHALL move only together with the default, never on its own.

#### Scenario: Default advanced first

- **WHEN** `DefaultSchemaModule` has been advanced to a newer core and the task runs
- **THEN** the core pin of every module in the task's discovered set (`CUE_MODULE_GLOBS`) moves to that release and the task exits zero; the text-pinned trees outside that set are not the task's to move

### Requirement: A held resolution that fails because a dependency requires a newer core exits non-zero and names the dependency
When the held resolution (Requirement: The update task requests core at the release the kernel default names and core moves only together with the default) fails because a dependency requires a newer core, the update task SHALL exit non-zero. The failure SHALL print the module directory and CUE's own error, and SHALL name each dependency whose own `opmodel.dev/core@v2` requirement is newer than the default, with that requirement and the default.

#### Scenario: A catalog built on a newer core

- **WHEN** the newest catalog build requires a core newer than the kernel default
- **THEN** the task exits non-zero, printing the module directory and CUE's error, and names the catalog build, the core it requires, and the default

### Requirement: The update task exits non-zero when the default cannot be read from the Go source
When the default (Requirement: The update task requests core at the release the kernel default names and core moves only together with the default) cannot be read from the Go source, the task SHALL exit non-zero with a message saying so.

#### Scenario: Unreadable default

- **WHEN** the task cannot read the default release from the Go source and no override is set
- **THEN** it exits non-zero with a message naming the file it read, instead of exiting silently

### Requirement: The repository root carries a .cascade-frozen file listing every test file that keeps a version literal on purpose
The repository root SHALL carry a `.cascade-frozen` file listing every test file that keeps a version literal of an OPM-owned module path (`opmodel.dev/core`, `opmodel.dev/catalogs/opm`) on purpose. Each entry SHALL name a repo-relative `path`, the `pins` (module paths) frozen at that location, and a one-sentence `reason`.

#### Scenario: Floor test keeps its old core

- **WHEN** the default core advances
- **THEN** `opm/kernel/render_core_floor_test.go` still re-pins its fixture copy to `v2.0.0-alpha.10`, and `.cascade-frozen` names that file with the reason

### Requirement: A file qualifies for the frozen list when its literal names an older release a test needs or is synthetic
A file (Requirement: The repository root carries a .cascade-frozen file listing every test file that keeps a version literal on purpose) SHALL qualify when its literal names an older release a floor, skew or pre-collision test needs, or when the literal is synthetic: a default move does not break it, because it is parsed or compiled in memory, or coupled to another synthetic literal, and independent of the current pins.

#### Scenario: Synthetic literal is listed

- **WHEN** a test parses an in-memory module file whose core or catalog version is never resolved
- **THEN** that test file is listed in `.cascade-frozen` with a reason saying the literal is synthetic

### Requirement: An expectation of the current core or of the pinned catalog build is not a frozen literal
An expectation of the current core or of the pinned catalog build, and a module file a test authors beside the served fixtures, SHALL NOT be a literal: it reads the derived version, so no entry exists to cover it. A listed file (Requirement: The repository root carries a .cascade-frozen file listing every test file that keeps a version literal on purpose) can still read the derived version in other places, for example to find the pin it replaces with an old one.

#### Scenario: Current-core expectation is not frozen

- **WHEN** a test asserts a resolved-version row for the current core
- **THEN** it reads the derived default, and no `.cascade-frozen` entry is needed for that assertion

### Requirement: Release-cascade tooling does not rewrite a version literal at a listed location
Release-cascade tooling SHALL NOT rewrite a version literal at a listed location (Requirement: The repository root carries a .cascade-frozen file listing every test file that keeps a version literal on purpose).

#### Scenario: Release-cascade tooling leaves a listed literal alone

- **WHEN** release-cascade tooling runs over a location listed in `.cascade-frozen`
- **THEN** it does not rewrite the version literal at that location
