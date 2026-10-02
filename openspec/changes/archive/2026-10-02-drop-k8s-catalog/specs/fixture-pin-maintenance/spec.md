## MODIFIED Requirements

### Requirement: Intentionally old version literals are declared

The repository root SHALL carry a `.cascade-frozen` file listing every test file that keeps a
version literal of an OPM-owned module path (`opmodel.dev/core`, `opmodel.dev/catalogs/opm`) on purpose. Each entry SHALL name
a repo-relative `path`, the `pins` (module paths) frozen at that location, and a one-sentence
`reason`. A file qualifies when its literal names an older release a floor, skew or
pre-collision test needs, or when the literal is synthetic: a default move does not break it,
because it is parsed or compiled in memory, or coupled to another synthetic literal, and
independent of the current pins. An expectation of the current
core or of the pinned catalog build, and a module file a test authors beside the served fixtures,
SHALL NOT be a literal: it reads the derived version, so no
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
