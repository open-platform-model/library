## Purpose

How the library keeps the dependency pins of its test CUE modules current: one resolution pass
per module, core held at the kernel's default release, a loud failure instead of a silent no-op,
and a declared list of test files whose version literals stay old on purpose.

## ADDED Requirements

### Requirement: Test module pins update in one resolution pass

The library's dependency-update task SHALL update every test CUE module it discovers (the
`CUE_MODULE_GLOBS` set) with one `cue mod get` per module. That one call SHALL name every direct
dependency of the module, followed by one `cue mod tidy`. Each non-core dependency SHALL be
requested at the newest published version within its major. A module whose dependencies are
several releases stale SHALL therefore update in a single run.

#### Scenario: Several stale dependencies move together

- **WHEN** a test module pins an older catalog build and an older `cue.dev/x/k8s.io` build, and both have newer releases on their majors
- **THEN** one run of the task moves both pins and prints an old-to-new line for each

#### Scenario: Nothing to update

- **WHEN** every test module already pins the newest release of each dependency
- **THEN** the task exits zero, leaves every `cue.mod/module.cue` byte-identical, and reports each dependency unchanged

### Requirement: Core is held at the kernel default

The update task SHALL request `opmodel.dev/core@v2` at the release `schema.DefaultSchemaModule`
names, read from the Go source, never at the newest published core. When resolution leaves any
test module pinning a core other than that release, the task SHALL exit non-zero. The failure
SHALL name the module, the core it resolved, the default, and the dependency that forced the
lift. Core in the test modules moves only together with the default, never on its own.

#### Scenario: A catalog built on a newer core

- **WHEN** the newest catalog build requires a core newer than the kernel default
- **THEN** the task exits non-zero, naming the module, the resolved core, the default, and the catalog as the cause

#### Scenario: Default advanced first

- **WHEN** `DefaultSchemaModule` has been advanced to a newer core and the task runs
- **THEN** every test module's core pin moves to that release and the task exits zero

### Requirement: Update failures are loud

The update task SHALL exit non-zero when any `cue mod get` or `cue mod tidy` fails. It SHALL print
the module directory and the command's own error output. It SHALL NOT continue to the next module
as though the failed one were unchanged, and SHALL NOT discard the command's error output.

#### Scenario: Unresolvable dependency

- **WHEN** a test module names a dependency the configured registry cannot serve
- **THEN** the task exits non-zero and prints that module's directory and CUE's error, and does not report the module as unchanged

### Requirement: Intentionally old version literals are declared

The repository root SHALL carry a `.cascade-frozen` file listing every test file that keeps an
`opmodel.dev/core` or `opmodel.dev/catalogs/opm` version literal on purpose. Each entry SHALL name
a repo-relative `path`, the `pins` (module paths) frozen at that location, and a one-sentence
`reason`. A file qualifies when its literal names an older release a floor, skew or
pre-collision test needs, or when the literal is synthetic: parsed or compiled in memory, never
resolved against a registry, and independent of the current pins. An expectation of the current
core or of the pinned catalog build SHALL NOT be a literal: it reads the derived version, so no
entry exists to cover it. A listed file can still read the derived version in other places, for
example to find the pin it replaces with an old one. Release-cascade tooling SHALL NOT rewrite a
version literal at a listed location.

#### Scenario: Floor test keeps its old core

- **WHEN** the default core advances
- **THEN** `opm/kernel/render_core_floor_test.go` still re-pins its fixture copy to `v2.0.0-alpha.10`, and `.cascade-frozen` names that file with the reason

#### Scenario: Synthetic literal is listed

- **WHEN** a test parses an in-memory module file whose core or catalog version is never resolved
- **THEN** that test file is listed in `.cascade-frozen` with a reason saying the literal is synthetic

#### Scenario: Current-core expectation is not frozen

- **WHEN** a test asserts a resolved-version row for the current core
- **THEN** it reads the derived default, and no `.cascade-frozen` entry is needed for that assertion
