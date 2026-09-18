## MODIFIED Requirements

### Requirement: DefaultSchemaModule constant

`schema.DefaultSchemaModule` SHALL name an exact core release, not the floating `opmodel.dev/core@v2` major: the release the kernel's render glue, fixtures and parity oracle were verified against. At this change that is `opmodel.dev/core@v2.0.0-alpha.10`, the first release carrying the comparable-predicate report on the derived `#Platform.#contracts` inventory (`comparable` and `discriminated`; enhancement 0015 D5 and OQ9) on top of the D1, D2 and D18 inventory, the D5 registry shape and the D12 context projection. `OCILoader.Load` with an empty `Module` field SHALL resolve this identifier. The constant advances only by a deliberate change that re-verifies the glue and fixtures against the new release; a default that floats ahead of the glue breaks every synthesized artifact on a cold cache. Doc comments citing the default module identifier (`opm/kernel`, `opm/schema`) SHALL cite the pinned identifier, the pin assertion in `opm/schema/loader_test.go` SHALL move with the constant, and the core version every served test fixture declares SHALL be the same release, so a fixture never pins a core the default kernel does not render against.

#### Scenario: Empty Module resolves the v2 default

- **WHEN** `(schema.OCILoader{Registry: "opmodel.dev=ghcr.io/open-platform-model"}).Load(ctx)` is called with `Module` unset
- **THEN** the loader resolves `Module` to `"opmodel.dev/core@v2.0.0-alpha.10"`, threads the env into `load.Config.Env`, and returns a non-zero `cue.Value` containing `#ModuleInstance`

#### Scenario: ResolvedVersion reports the v2 resolution

- **WHEN** `cache.Get()` succeeds against the default
- **THEN** `cache.ResolvedVersion()` returns `"v2.0.0-alpha.10"`

#### Scenario: No doc comment cites a deleted package or the floating major

- **WHEN** a developer searches `opm/` for the default module identifier
- **THEN** every citation names the pinned release and none lives in `opm/materialize`, which no longer exists

#### Scenario: Served fixtures pin the default release

- **WHEN** a test authors a module beside the served fixtures with the fixture harness's declared core version
- **THEN** that version equals the release `DefaultSchemaModule` pins, and the render fixtures under `testdata/render` declare the same release in every `cue.mod`
