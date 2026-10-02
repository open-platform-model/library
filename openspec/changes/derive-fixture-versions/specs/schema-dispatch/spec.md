## MODIFIED Requirements

### Requirement: DefaultSchemaModule constant

`schema.DefaultSchemaModule` SHALL name an exact core release, not the floating `opmodel.dev/core@v2` major. It is the release the kernel's render glue, fixtures and parity oracle were verified against. At this change that is `opmodel.dev/core@v2.0.0-beta.1`, the first release of core's beta line. It carries the schema of `2.0.0-alpha.13` unchanged. `2.0.0-alpha.13` remains the first release that reports contract collisions on the derived `#Platform.#contracts` inventory (`collisions` and `collidingEntries`, with `routable` false while any exist, and with `defined` and `definedBy` folding only single-definer keys; core change `fold-colliding-contract-keys`). It reports them on top of the per-registry-entry provider count (`providedBy`), the comparable-predicate report (`comparable` and `discriminated`; 0015:D5, 0015:OQ9), the 0015:D1/D2/D18 inventory, the 0019:D5 registry shape and the 0019:D12 context projection. The default's documentation SHALL keep naming `2.0.0-alpha.13` as that release and SHALL NOT attribute those reports to the beta release. The default is not the render floor: `Kernel.Render` and `Platform.Contracts()` accept every core from `schema.ProvidedBySince` (`2.0.0-alpha.12`) on, and a platform pinning a release between the floor and `2.0.0-alpha.13` decodes an absent collision report as no collision. `OCILoader.Load` with an empty `Module` field SHALL resolve this identifier. The constant advances only through a deliberate change that re-verifies the glue and fixtures against the new release; a default that floats ahead of the glue breaks every synthesized artifact on a cold cache. Doc comments that cite the default module identifier (`opm/kernel`, `opm/schema`) SHALL cite it by the constant's name (`DefaultSchemaModule`), never by the floating major, and SHALL NOT present a literal release as the current default; a release literal in a doc comment is a format example only, so a default move edits no doc comment.

The constant SHALL be the only Go source that the default move edits. The fixture harness's declared core version SHALL be derived from it, never kept as a second literal. No test SHALL spell the default release where it means the current core: a test that expects the current core (a resolved-version row, a module file it authors beside the served fixtures, a staged module file built beside them) SHALL read the harness's declared core version. Synthetic, floor and skew literals that happen to equal or predate the default are not "the current core"; they stay literal and are declared in `.cascade-frozen` (capability `fixture-pin-maintenance`). The schema tests SHALL assert the constant's shape: an exact release on the `v2` major whose version suffix is `DefaultSchemaVersion()`. They SHALL NOT repeat the release as a literal. Advancing the default therefore edits the constant and the fixture `cue.mod` pins, and no test file and no doc comment. The deliberate re-verification is the review of the change that advances it.

The core version every served test fixture declares SHALL be the same release, so a fixture never pins a core the default kernel does not render against. A test that exercises an older core copies a fixture and re-pins the copy. It never commits a fixture pinning another release, and it reads the fixture's current pin from the fixture harness's declared core version rather than from a literal. Test files that keep an older or synthetic core literal on purpose are declared as frozen (capability `fixture-pin-maintenance`).

#### Scenario: Empty Module resolves the v2 default

- **WHEN** `(schema.OCILoader{Registry: "opmodel.dev=ghcr.io/open-platform-model"}).Load(ctx)` is called with `Module` unset
- **THEN** the loader resolves `Module` to `schema.DefaultSchemaModule`, threads the env into `load.Config.Env`, and returns a non-zero `cue.Value` containing `#ModuleInstance`

#### Scenario: ResolvedVersion reports the v2 resolution

- **WHEN** `cache.Get()` succeeds against the default
- **THEN** `cache.ResolvedVersion()` returns `schema.DefaultSchemaVersion()`

#### Scenario: No doc comment cites a deleted package or the floating major

- **WHEN** a developer searches `opm/` for the default module identifier
- **THEN** every citation names the constant `DefaultSchemaModule` (or, on the constant's own line, the exact release it holds), none names the floating major as the default, and none lives in `opm/materialize`, which no longer exists

#### Scenario: Served fixtures pin the default release

- **WHEN** a test authors a module beside the served fixtures with the fixture harness's declared core version
- **THEN** that version equals the release `DefaultSchemaModule` pins, and the render fixtures under `testdata/render` declare the same release in every `cue.mod`

#### Scenario: An older-core test survives a default move

- **WHEN** the default core release advances and every served fixture is re-pinned with it
- **THEN** a test that copies a served fixture and re-pins the copy to an older core still finds the fixture's pin through the harness's declared core version and needs no edit of its own

#### Scenario: A default move edits no test file

- **WHEN** `DefaultSchemaModule` advances to a newer published core and the fixture `cue.mod` pins are re-pinned with it
- **THEN** the suite passes with no `*_test.go` edited, and the only Go change is the constant

#### Scenario: The pin assertion checks shape, not the release

- **WHEN** the schema tests check the default
- **THEN** they assert that the constant names an exact release on the `v2` major and that `DefaultSchemaVersion()` is its version suffix, and no assertion spells that release as a literal
