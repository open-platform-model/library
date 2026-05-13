## MODIFIED Requirements

### Requirement: Fixture file shape

Each fixture file SHALL expose at least one of two top-level fields: `input:` (the construction under test, typed against a v1alpha2 schema definition) and `expect:` (a concrete equality target unified with `input` when the harness asserts positive equality). Fixtures MAY define additional helper bindings, but the harness SHALL only consult the value-under-test field and its paired equality target.

A fixture MAY bundle multiple independent cases. When the harness drives a case via the `inputPath` override (default `"input"`), the value-under-test lives at the override path, and the paired positive-equality target lives at `"<inputPath>_expect"` (or stays at `"expect"` when `inputPath == "input"`). Bundled negative cases pair `inputPath` with `expectError` and need no equality target.

#### Scenario: Positive case unifies input with expect

- **WHEN** a fixture declares both `input: core.#Platform & {...}` and `expect: {...}` and the harness drives a positive case with default `inputPath`
- **THEN** the harness evaluates `input & expect` under `Validate(cue.Concrete(true))` and the case passes if and only if the result is concrete and free of bottoms

#### Scenario: Negative case ignores expect

- **WHEN** a fixture declares only `input:` and the harness drives a negative case with a non-empty error regex
- **THEN** the harness evaluates `input.Validate(cue.Concrete(true))`, requires the returned `error` to be non-nil, and matches the regex against `error.Error()`

#### Scenario: Bundled fixture exposes multiple inputs

- **WHEN** a fixture declares fields `bad_one: ...` and `bad_two: ...` and the harness drives two cases with `inputPath: "bad_one"` and `inputPath: "bad_two"` respectively
- **THEN** each case is asserted against the value at its own `inputPath`, independently, with its own `expectError` regex; failures on one case do not mask failures on the other

#### Scenario: Bundled positive case pairs `<inputPath>_expect`

- **WHEN** a fixture declares `case_a: ...` and `case_a_expect: ...` and the harness drives a case with `inputPath: "case_a"` and no `expectError`
- **THEN** the harness evaluates `case_a & case_a_expect` under `Validate(cue.Concrete(true))` and applies the same concrete-equality contract as the default `input & expect` pairing

### Requirement: Table-driven Go harness in `opm/api/v1alpha2/`

The library SHALL provide a single test file `opm/api/v1alpha2/schema_fixture_test.go` containing a function `TestSchemaFixtures` that table-drives a slice of cases over the fixtures in `apis/core/v1alpha2/testdata/`. Each case SHALL specify: a fixture filename, an optional `inputPath` overriding the default `"input"` lookup, an optional CUE path plus Go decode target for positive value-equality, and an optional regex for negative error matching. The harness SHALL load fixtures via `cuelang.org/go/cue/load` with `Config.Tags: []string{"test"}` and `Config.Dir` resolved to the on-disk `apis/core` module root.

#### Scenario: Positive case asserts decoded value

- **WHEN** a case declares a non-empty `assertField` and `assertValue`
- **THEN** the harness looks up `assertField` on the compiled value, decodes it into a Go target shaped like `assertValue`, and asserts equality

#### Scenario: Negative case asserts error regex

- **WHEN** a case declares a non-empty `expectError`
- **THEN** the harness asserts that `cue.Value.Validate(cue.Concrete(true))` returns a non-nil error whose `Error()` matches the regex; build-time errors are accepted in lieu of validate-time errors when CUE evaluates the failing constraint eagerly

#### Scenario: Missing fixture file fails the case

- **WHEN** a case names a fixture filename that does not exist under `apis/core/v1alpha2/testdata/`
- **THEN** the harness fails the subtest with a clear message and SHALL NOT silently skip

#### Scenario: Build error in fixture surfaces with fixture path

- **WHEN** a fixture has a CUE syntax error or unresolved import
- **THEN** the harness reports the load / build error on the fixture's subtest (not as a global test failure) and includes the fixture filename in the failure message

#### Scenario: `inputPath` defaults to `"input"`

- **WHEN** a case omits `inputPath`
- **THEN** the harness looks up the value-under-test at the literal CUE path `"input"` and the paired equality target at `"expect"` — preserving the seed-fixture contract without requiring any change to existing rows

### Requirement: Convention documented for contributors

The library SHALL include `apis/core/v1alpha2/testdata/README.md` documenting: the `@if(test)` requirement, the `package fixtures` convention, the meaning of `input:` and `expect:`, the location and shape of the Go harness table, the `inputPath` override and when to use it, the `close()`-on-open-maps caveat with a worked length-check example, and the steps to add a new fixture + case.

#### Scenario: README explains add-fixture workflow

- **WHEN** a contributor reads `apis/core/v1alpha2/testdata/README.md`
- **THEN** they can add a new fixture file and table row without reading the harness implementation, knowing only the location of the table in `opm/api/v1alpha2/schema_fixture_test.go`

#### Scenario: README documents bundled-fixture usage

- **WHEN** a contributor reads the README and wants to bundle multiple type-regex assertions in one fixture
- **THEN** the README's "Bundling multiple cases per fixture" section directs them to the `inputPath` override with a worked example (`type_regex_fixture.cue`)

#### Scenario: README warns about close()-on-open-maps

- **WHEN** a contributor reads the README and wants to assert "this map has exactly these keys"
- **THEN** the README's "Caveat" section explains that wrapping `expect:` in `close({...})` does NOT reject extra keys when the input field is open, and shows the length-check pattern (`_count: len([for k, _ in input.field {k}]) & N`) as the supported alternative

## ADDED Requirements

### Requirement: Predicate-distinctness positive coverage

The library SHALL include a positive fixture proving that two `#ComponentTransformer`s requiring the same FQN with DIFFERENT `requiredLabels` produce distinct `_predicateSignature` values and therefore stay out of `#PlatformBase.#matchers._invalid`.

#### Scenario: Distinct labels keep `_invalid` empty

- **WHEN** a `#Platform` registers a module containing two transformers, both requiring the same resource FQN, with `requiredLabels: {tier: "prod"}` and `requiredLabels: {tier: "dev"}` respectively
- **THEN** evaluating the platform under `cue.Concrete(true)` with `Tags: []string{"test"}` succeeds, `#matchers.resources[<FQN>]` contains both transformers as candidates, `#matchers._invalid.resources` is empty, and `#matchers._noMultiFulfiller` is `0`

### Requirement: Disabled-registration suppression coverage

The library SHALL include a positive fixture proving that `#ModuleRegistration.enabled: false` suppresses every projection (`#knownResources`, `#knownTraits`, `#composedTransformers`, `#matchers`) of the disabled module, even when that module's primitives would otherwise trigger `_noMultiFulfiller`.

#### Scenario: Disabled module's multi-fulfiller payload does not surface

- **WHEN** a `#Platform` registers two modules — one with `enabled: false` carrying transformers that would conflict on `_predicateSignature`, and one enabled module carrying a single benign transformer
- **THEN** evaluating the platform succeeds, `#matchers.resources` reflects only the enabled module's primitives, the disabled module's FQNs are absent from `#composedTransformers` / `#knownResources` / `#knownTraits`, and `#matchers._noMultiFulfiller` is `0`

### Requirement: `#TransformerContext` label/annotation merge coverage

The library SHALL include a positive fixture exercising `#TransformerContext`'s merge of module-scope, component-scope, and controller-scope labels and annotations, including the `transformer.opmodel.dev/`-prefix filter and the `app.kubernetes.io/managed-by` stamp from `#runtimeName`.

#### Scenario: Merge precedence and prefix filter both fire

- **WHEN** a fixture constructs `core.#TransformerContext` with concrete `#moduleReleaseMetadata` (carrying labels and annotations), `#componentMetadata` (carrying labels and annotations including a `transformer.opmodel.dev/internal` key in each), and `#runtimeName: "opm-cli"`
- **THEN** the computed `labels` map contains the module-scope key, the controller-scope `app.kubernetes.io/managed-by: "opm-cli"`, and the component-scope tier label, but does NOT contain any `transformer.opmodel.dev/*` key; the same prefix filter applies to `annotations`

#### Scenario: Length-check rejects filter regression

- **WHEN** the fixture's `expect` block declares both the positive label/annotation map AND a hidden `_labelKeyCount: len([for k, _ in input.labels {k}]) & N` field with `N` set to the expected key count
- **THEN** a regression that lets a filtered key through bumps the key count and fails unification with `conflicting values N+1 and N`

### Requirement: `#Module` UUID/FQN derivation coverage

The library SHALL include a positive fixture pinning a known input (name, modulePath, version) and asserting both the derived `metadata.fqn` (format-string check, `module.cue:17`) and the derived `metadata.uuid` (UUIDv5(OPMNamespace, fqn) computed at `module.cue:20`). The pinned UUID acts as a drift sentinel for `OPMNamespace` (`types.cue:50`).

#### Scenario: FQN matches format string

- **WHEN** the harness drives the `module_uuid_fixture.cue` case asserting `assertField: "input.metadata.fqn"`
- **THEN** the decoded value equals the expected `<modulePath>/<name>:<version>` string

#### Scenario: UUID matches pinned UUIDv5 hash

- **WHEN** the harness drives the case asserting `assertField: "input.metadata.uuid"`
- **THEN** the decoded value equals the pinned UUID; if `OPMNamespace` changes, this case fails loudly and the operator must regenerate the pin via `cue eval -t test --expression input.metadata.uuid` and update the harness row

### Requirement: Trait-side symmetric coverage

The library SHALL include positive and negative fixtures exercising the `requiredTraits` / `#matchers.traits` / `_invalid.traits` code path, mirroring the resource-side seed fixtures.

#### Scenario: Positive trait matchers projection passes

- **WHEN** a `#Platform` registers a module containing one `#Trait` and one `#ComponentTransformer` with `requiredTraits` keyed to that trait's FQN
- **THEN** `#matchers.traits[<FQN>]` contains one candidate, `#matchers.resources` is empty, `#matchers._invalid` is `{resources: [], traits: []}`, and `_noMultiFulfiller` is `0`

#### Scenario: Trait-side multi-fulfiller fails as expected

- **WHEN** a `#Platform` registers a module containing two transformers with the same `requiredTraits` FQN and identical predicate signatures
- **THEN** evaluation fails with an error whose message references `_noMultiFulfiller`, matching the harness regex

### Requirement: Type-regex gate coverage

The library SHALL include a bundled negative fixture (`type_regex_fixture.cue`) that exercises the regex gates on `#NameType`, `#FQNType`, `#ModuleFQNType`, and `#MajorVersionType`. Each gate SHALL be asserted via at least one `bad_<reason>` field rejected by CUE under `Validate(cue.Concrete(true))`.

#### Scenario: `#NameType` rejects uppercase identifier

- **WHEN** the harness drives the case `inputPath: "bad_name_uppercase"` against `type_regex_fixture.cue`
- **THEN** evaluation fails with an error matching the regex `invalid value`

#### Scenario: `#NameType` rejects leading-hyphen identifier

- **WHEN** the harness drives the case `inputPath: "bad_name_leading_hyphen"`
- **THEN** evaluation fails with an error matching `invalid value`

#### Scenario: `#FQNType` rejects FQN without `@vN`

- **WHEN** the harness drives the case `inputPath: "bad_fqn_no_version"`
- **THEN** evaluation fails with an error matching `invalid value`

#### Scenario: `#ModuleFQNType` rejects module FQN without semver

- **WHEN** the harness drives the case `inputPath: "bad_module_fqn_no_semver"`
- **THEN** evaluation fails with an error matching `invalid value`

#### Scenario: `#MajorVersionType` rejects version without `v` prefix

- **WHEN** the harness drives the case `inputPath: "bad_version_no_v_prefix"`
- **THEN** evaluation fails with an error matching `invalid value`
