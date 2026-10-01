## Why

Every catalog bump breaks `TestParity_ShippedCatalog`, because the shipped case table hardcodes
the catalog build in seven transformer ids (`opm/kernel/parity_harness_test.go:150-196`, each
`shippedCatalogPrefix + "<name>@4.4.2"`). The failure text blames the fixture ("a failing oracle
is a broken fixture"), so it reads like a kernel divergence. It is only a version literal the bump
tooling cannot reach (trap T1 in workspace RELEASING.md, section "Pin classes"; library commits
`588a638`, `ae9e247`, `fad87fd`, `7f0dd5b` each re-edited the same seven lines). Core bumps have the
same shape: `6081afd` (core `v2.0.0-beta.1`) hand-edited literals across seven test files on top of
the cue.mod pins.

The release cascade (workspace RELEASING.md, sections "The cascade" and "Cascade files") lets a bot
open library's `deps/cascade` PR. A bot can rewrite `cue.mod/module.cue` files and the two
default-version constants. It should not have to find and rewrite Go string literals. The owner
settled this as decision D8: the library derives its test versions from the pins, and the old
literals that are frozen on purpose stay as they are. This change carries out D8. After it, a
catalog bump touches only cue.mod files, and a core bump touches only
`schema.DefaultSchemaModule` and cue.mod files.

## What Changes

- **Parity harness reads the catalog build from the parity module's cue.mod.** The shipped case
  table names transformers without a version. The harness reads the `opmodel.dev/catalogs/opm@v4`
  pin from `testdata/parity/cue.mod/module.cue`, which is the module the oracle builds in (the
  harness loads `./shipped` with `Dir: testdata/parity`). It then builds each transformer id from
  that pin. The harness also asserts that the catalog build the oracle resolved equals that pin.
  If MVS lifts the pin, the test fails and names both versions, so the lift is not hidden.
- **Current-core test literals derive from the default.** `registrytest.DefaultCoreVersion` stops
  being a second hand-kept literal and derives from `schema.DefaultSchemaVersion()`. Test literals
  that mean "the current core" read it instead of spelling `v2.0.0-beta.1`: the
  `ResolvedVersions` rows and the authored instance module file in `opm/kernel/render_test.go`,
  and the authored module files in `opm/internal/renderstage/stage_test.go`. The literal pin
  assertion in `opm/schema/loader_test.go` becomes a structural assertion (an exact v2 release,
  never the floating major). The core release no longer appears in the fixture platform header
  comments under `testdata/render/platform*/platform.cue` or in
  `modules/opm_platform/platform.cue`, nor in the `AGENTS.md` layout line.
- **Frozen literals are declared, not touched.** A new root `.cascade-frozen` lists every test
  file that keeps a version literal on purpose, with a one-sentence reason each. There are two
  kinds: floor and skew tests that need an old core (`render_collision_test.go`,
  `render_core_floor_test.go`, `opm/platform/contracts_test.go`, `opm/errors/coretooold_test.go`),
  and synthetic literals that are parsed in memory and never resolved (the `platformmodule`
  closure and golden graph on `4.0.1`/`alpha.7`, the `renderstage` module-file parse tests,
  `catalog/requires_test.go`, the `objectset` registration id, and the parse tables in
  `registrytest_test.go` and `loader_test.go`). Section 1 checks the synthetic class by mutation
  before the file is written.
- **`task cue:deps:update` runs in one pass and fails loudly.** It runs one `cue mod get` per
  module over every dependency at once, like the root `deps:update`, then `cue mod tidy`. It holds
  `opmodel.dev/core@v2` at the release `schema.DefaultSchemaModule` pins. It exits non-zero when a
  get or tidy fails, and also when resolution lifts core past the default. Today the task gets
  each dependency separately and appends `|| true`, so a stale sibling turns every get into a
  silent no-op (`Taskfile.yml:315-321`). The stale comment on the `cue:catalog:drift` CI step
  (`.github/workflows/cue.yml:39-42`) is corrected in the same section.
- **Proof:** the last section bumps the parity and test catalog pins from `4.4.2` to `4.4.4`, the
  newest `catalogs/opm` build and itself built on core `v2.0.0-beta.1`. It lands as
  `test(fixtures)`, and the diff touches only `cue.mod/module.cue` files.
- Not **BREAKING**. No `opm/` type, signature or behavior changes. Every edit is in test code,
  test fixtures, comments, `Taskfile.yml` and one CI comment.

SemVer class: none (no release). Every section lands as `test`, `build` or `chore`, which
release-please hides. The PR title is `test(fixtures): derive test versions from the pins`.

### Depends on / gates

- Depends on: nothing. This change is independent of the other phase-1 changes (catalog_opm
  `prepare-release-cascade`, library `prepare-release-cascade`, opm-operator
  `prepare-release-cascade`, cli `prepare-release-cascade`, `bump-stale-testdata-pins`,
  `add-embedded-operator-e2e-job`) and of the workspace `docs/release-cascade` branch. It cites
  the workspace RELEASING.md only for vocabulary.
- Gates: library `add-deps-cascade-task` (phase B) builds on this change. Its `deps:cascade`
  calls the single-pass `task cue:deps:update`, reads `.cascade-frozen`, and relies on a catalog
  or core bump not touching Go test files. `join-release-cascade` (phase C) for library must not
  start until this change is merged.
- Not in this change: a guard that fails when a new version literal appears in a `*_test.go`
  outside `.cascade-frozen` (belongs in `add-deps-cascade-task`, which first gives the file a
  reader). Also out: the `.cascade-hold` file, any cascade workflow, and the docs examples
  (`docs/getting-started.md`, `AGENTS.md:329`) that show an explicit `OCILoader{Module: ...}` pin
  as user-facing illustration.

## Capabilities

### New Capabilities

- `fixture-pin-maintenance`: how the library's test cue.mod pins are updated (one resolution
  pass, core held at the default, loud failure), and how test files that keep a version literal
  on purpose are declared.

### Modified Capabilities

- `render-parity`: the shipped case table takes the catalog build from the parity module's pin
  instead of a literal, and the harness asserts that the pin is the build the oracle resolved.
- `schema-dispatch`: in "DefaultSchemaModule constant", the fixture harness's declared core
  version is derived from the constant, the pin assertion no longer repeats the release as a
  literal, and test expectations of the current core read the derived version.

## Impact

- Packages (test code only): `opm/kernel` (parity harness, render tests), `opm/internal/registrytest`
  (`DefaultCoreVersion` becomes a derived package-level value), `opm/internal/renderstage`
  (`stage_test.go`), `opm/schema` (`loader_test.go`).
- Fixtures: header comments in 13 `platform.cue` files; catalog pins in
  `modules/opm_platform`, `testdata/modules/web_app`, `testdata/parity` and
  `testdata/parity/opm_platform` (section 5).
- Tooling: `Taskfile.yml` `cue:deps:update`; `.github/workflows/cue.yml` comment; new root
  `.cascade-frozen`.
- Downstream consumers (cli, opm-operator): none. They do not import `opm/internal/registrytest`.
- No `enhancement.yaml`: decision D2 says the release cascade is not an enhancement. Its design
  lives in the workspace RELEASING.md.
