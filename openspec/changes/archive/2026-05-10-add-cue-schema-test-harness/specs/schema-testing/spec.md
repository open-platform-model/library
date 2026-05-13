## ADDED Requirements

### Requirement: Fixtures live outside the embedded schema FS

The library SHALL place all CUE schema test fixtures under `apis/core/v1alpha2/testdata/` and SHALL NOT include that directory in the `//go:embed` pattern declared in `apis/core/embed.go`. Fixtures MUST NOT ship to downstream Go consumers via the embedded `Schema` filesystem.

#### Scenario: Embed pattern excludes testdata

- **WHEN** `opm/api/v1alpha2/embed_test.go` walks the embedded `Schema` filesystem returned by `api.EmbeddedSchema(apiversion.V1alpha2)`
- **THEN** no path under `testdata/` appears in the result

#### Scenario: Disk schema set matches embed regardless of testdata contents

- **WHEN** a contributor adds a new fixture under `apis/core/v1alpha2/testdata/`
- **THEN** `TestEmbeddedSchema_FileSetMatchesDisk` continues to pass without modification, because both sides exclude `testdata/` from the comparison set

### Requirement: Fixtures opt in via `@if(test)`

Every CUE file under `apis/core/v1alpha2/testdata/` SHALL begin with the file-level attribute `@if(test)` placed before its `package` clause. Fixtures MUST NOT use the filename suffix `_test.cue`. Fixtures MUST declare `package fixtures`.

#### Scenario: Default `cue vet` does not load fixtures

- **WHEN** an operator runs `cue vet ./...` from `apis/core/v1alpha2/` without `-t test`
- **THEN** no fixture under `testdata/` is compiled, evaluated, or reported on

#### Scenario: Loader with `Tags: []string{"test"}` includes fixtures

- **WHEN** a Go caller invokes `load.Instances([]string{"v1alpha2/testdata/<fixture>.cue"}, &load.Config{Dir: apis/core, Tags: []string{"test"}})`
- **THEN** the returned instance compiles successfully and exposes the fixture's `input` and (where present) `expect` fields

#### Scenario: Loader without test tag returns no fixture instances

- **WHEN** a Go caller invokes `load.Instances` against the same path without setting `Config.Tags`
- **THEN** the returned `*build.Instance` either reports zero CUE files or omits all `@if(test)`-gated files; no fixture content is evaluated

### Requirement: Fixture file shape

Each fixture file SHALL expose at least one of two top-level fields: `input:` (the construction under test, typed against a v1alpha2 schema definition) and `expect:` (a concrete equality target unified with `input` when the harness asserts positive equality). Fixtures MAY define additional helper bindings, but the harness SHALL only consult `input:` and `expect:`.

#### Scenario: Positive case unifies input with expect

- **WHEN** a fixture declares both `input: core.#Platform & {...}` and `expect: {...}` and the harness drives a positive case
- **THEN** the harness evaluates `input & expect` under `Validate(cue.Concrete(true))` and the case passes if and only if the result is concrete and free of bottoms

#### Scenario: Negative case ignores expect

- **WHEN** a fixture declares only `input:` and the harness drives a negative case with a non-empty error regex
- **THEN** the harness evaluates `input.Validate(cue.Concrete(true))`, requires the returned `error` to be non-nil, and matches the regex against `error.Error()`

### Requirement: Table-driven Go harness in `opm/api/v1alpha2/`

The library SHALL provide a single test file `opm/api/v1alpha2/schema_fixture_test.go` containing a function `TestSchemaFixtures` that table-drives a slice of cases over the fixtures in `apis/core/v1alpha2/testdata/`. Each case SHALL specify: a fixture filename, an optional CUE path plus Go decode target for positive value-equality, and an optional regex for negative error matching. The harness SHALL load fixtures via `cuelang.org/go/cue/load` with `Config.Tags: []string{"test"}` and `Config.Dir` resolved to the on-disk `apis/core` module root.

#### Scenario: Positive case asserts decoded value

- **WHEN** a case declares a non-empty `assertField` and `assertValue`
- **THEN** the harness looks up `assertField` on the compiled value, decodes it into a Go target shaped like `assertValue`, and asserts equality

#### Scenario: Negative case asserts error regex

- **WHEN** a case declares a non-empty `expectError`
- **THEN** the harness asserts that `cue.Value.Validate(cue.Concrete(true))` returns a non-nil error whose `Error()` matches the regex

#### Scenario: Missing fixture file fails the case

- **WHEN** a case names a fixture filename that does not exist under `apis/core/v1alpha2/testdata/`
- **THEN** the harness fails the subtest with a clear message and SHALL NOT silently skip

#### Scenario: Build error in fixture surfaces with fixture path

- **WHEN** a fixture has a CUE syntax error or unresolved import
- **THEN** the harness reports the load / build error on the fixture's subtest (not as a global test failure) and includes the fixture filename in the failure message

### Requirement: Seed coverage at introduction

The first landing of this capability SHALL include three seed fixtures + cases covering at minimum: one positive `#Platform.#matchers` projection, one negative `_noMultiFulfiller` failure (two transformers with identical predicate signatures on the same FQN), and one negative FQN-collision failure (two `#Module`s defining the same `#defines.resources[FQN]` registered on one `#Platform`).

#### Scenario: Positive matchers projection passes

- **WHEN** the seed fixture for `#PlatformBase.#matchers` is evaluated under `cue.Concrete(true)` with `Tags: []string{"test"}`
- **THEN** the resulting value has a non-empty `#matchers.resources` map for at least one resource FQN, an empty `#matchers._invalid.resources` list, and unifies with the fixture's `expect` block

#### Scenario: Multi-fulfiller negative case fails as expected

- **WHEN** the multi-fulfiller fixture is evaluated under `cue.Concrete(true)`
- **THEN** the harness receives an error whose message references `_noMultiFulfiller` and the case's regex matches

#### Scenario: FQN collision negative case fails as expected

- **WHEN** the FQN-collision fixture is evaluated under `cue.Concrete(true)`
- **THEN** the harness receives an error whose message references conflicting or incompatible values for the colliding FQN, and the case's regex matches

### Requirement: Convention documented for contributors

The library SHALL include `apis/core/v1alpha2/testdata/README.md` documenting: the `@if(test)` requirement, the `package fixtures` convention, the meaning of `input:` and `expect:`, the location and shape of the Go harness table, and the steps to add a new fixture + case.

#### Scenario: README explains add-fixture workflow

- **WHEN** a contributor reads `apis/core/v1alpha2/testdata/README.md`
- **THEN** they can add a new fixture file and table row without reading the harness implementation, knowing only the location of the table in `opm/api/v1alpha2/schema_fixture_test.go`
