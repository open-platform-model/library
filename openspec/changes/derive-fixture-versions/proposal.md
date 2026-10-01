## Why

Every catalog bump breaks `TestParity_ShippedCatalog`, because the shipped case table hardcodes
the catalog build in seven transformer ids (`opm/kernel/parity_harness_test.go:150-196`, each
`shippedCatalogPrefix + "<name>@4.4.2"`). The failure text blames the fixture ("a failing oracle
is a broken fixture"), so it reads like a kernel divergence. It is only a version literal the bump
tooling cannot reach. Library commits `588a638`, `ae9e247`, `fad87fd` and `7f0dd5b` each
re-edited the same seven lines. Core bumps have the same shape: `6081afd` (core `v2.0.0-beta.1`)
hand-edited literals across seven test files on top of the cue.mod pins.

The release cascade (workspace RELEASING.md, sections "The cascade" and "Cascade files") lets a bot
open library's `deps/cascade` PR. A bot can rewrite `cue.mod/module.cue` files and the default
constant. It should not have to find and rewrite Go string literals. Workspace RELEASING.md,
section "Pin classes", splits the library's pins into shipped, test and frozen. This change makes
that split real in the code: the library derives its current-version test expectations from the
pins, and the old literals that are frozen on purpose stay as they are and are declared. After
it, a catalog bump touches only cue.mod files, and a core bump touches only
`schema.DefaultSchemaModule` and cue.mod files.

## What Changes

- **Parity harness reads the catalog build from the parity module's cue.mod.** The shipped case
  table names transformers without a version. The harness reads the `opmodel.dev/catalogs/opm@v4`
  pin from `testdata/parity/cue.mod/module.cue`, which is the module the oracle builds in (the
  harness loads `./shipped` with `Dir: testdata/parity`). It then builds each transformer id from
  that pin. It also checks that the catalog build the oracle reports equals that pin, with a
  neutral message naming both. CUE v0.17.1 builds against the version the main module lists, so
  this is a consistency check on the harness, not a guard against resolution.
- **Current-core test literals derive from the default.** `registrytest.DefaultCoreVersion` stops
  being a second hand-kept literal and derives from `schema.DefaultSchemaVersion()`. A
  default-move dry run on the untouched tree (recorded in design.md, DF6) found the test
  literals a core bump breaks: the six `ResolvedVersions` core rows in `opm/kernel/render_test.go`
  and the two pin assertions in `opm/schema/loader_test.go`. The rows read `DefaultCoreVersion`.
  The loader assertions become structural (an exact v2 release, never the floating major). Three
  module files a test authors beside the served fixtures (`render_test.go:906`,
  `stage_test.go:345,569`) also read it, because their pin means "the current core" even though a
  downward dry run cannot break them. The core release no longer appears in the fixture platform
  header comments, in the `AGENTS.md` layout line, or in Go doc comments that present it as the
  default (`opm/schema/loader.go`, `opm/schema/cache.go`).
- **Frozen literals are declared, not touched.** A new root `.cascade-frozen` lists every test
  file that keeps an OPM-owned version literal (core, `catalogs/opm`, `catalogs/k8s`) on purpose,
  with a one-sentence reason each. There are two kinds: floor and skew tests that need an old core
  (`render_collision_test.go`, `render_core_floor_test.go`, `opm/platform/contracts_test.go`,
  `opm/errors/coretooold_test.go`), and synthetic literals that a default move does not break (the
  `platformmodule` golden, explicit-pin and closure tests, the `renderstage` module-file parse
  and local-file tests, `catalog/requires_test.go`, the `objectset` registration id, and the
  parse tables in `registrytest_test.go` and `loader_test.go`).
- **`task cue:deps:update` runs in one pass and fails loudly.** It runs one `cue mod get` per
  module over every dependency at once, like the root `deps:update`, then `cue mod tidy`. It holds
  `opmodel.dev/core@v2` at the release `schema.DefaultSchemaModule` pins (overridable with
  `DEFAULT_CORE`). It exits non-zero when a get or tidy fails. When the held get fails because a
  dependency needs a newer core, it reruns the get without core in a throwaway copy and names
  each dependency whose own core requirement is newer than the default. Today the task gets each
  dependency separately and appends `|| true`, so a stale sibling turns every get into a silent
  no-op (`Taskfile.yml:315-321`). The stale "tracks the latest" claims in the `cue:catalog:drift`
  CI step comment (`.github/workflows/cue.yml:40-41`) and in `cue-versions.yml:9-12` are
  corrected in the same section.
- **Proof:** the last section bumps the test catalog pins from `4.4.2` to `4.4.4`, the newest
  `catalogs/opm` build and itself built on core `v2.0.0-beta.1`. It lands as `test(fixtures)`, and
  the diff touches only `cue.mod/module.cue` files.
- Not **BREAKING**. No `opm/` type, signature or behavior changes. Every edit is in test code,
  test fixtures, comments, `Taskfile.yml` and two CI/registry comments.

SemVer class: none (no release). Every section lands as `test`, `build` or `chore`, which
release-please hides. The PR title is `test(fixtures): derive test versions from the pins`.

### Depends on / gates

- Depends on: nothing. This change is independent of the other phase-1 changes (catalog_opm
  `prepare-release-cascade`, library `prepare-release-cascade`, opm-operator
  `prepare-release-cascade`, cli `prepare-release-cascade`, `bump-stale-testdata-pins`,
  `add-embedded-operator-e2e-job`) and of the workspace `docs/release-cascade` branch. It cites
  the workspace RELEASING.md only for vocabulary.
- File overlap: library `prepare-release-cascade` also edits `Taskfile.yml` and `AGENTS.md`, in
  different places (this change: the `cue:deps:update` task and the `testdata/` layout line). The
  second of the two to merge rebases over the first; no hunk is shared.
- Gates: library `add-deps-cascade-task` (phase B) builds on this change. Its `deps:cascade`
  advances `DefaultSchemaModule` first (labelled `need-human-review`), then calls the single-pass
  `task cue:deps:update` (passing `DEFAULT_CORE` where it needs to), reads `.cascade-frozen`, and
  relies on a catalog or core bump not touching Go test files. `join-release-cascade` (phase C)
  for library must not start until this change is merged.
- Workspace RELEASING.md follow-up (for the `docs/release-cascade` branch author): in section
  "Pin classes", the library shipped row should say `DefaultCoreVersion` is derived from
  `DefaultSchemaModule` (not mirrored), and the library frozen row should point at library
  `.cascade-frozen` instead of listing files; in section "The cascade", "What each repo's task
  moves" should list only `DefaultSchemaModule` for the library.
- Not in this change: a guard that fails when a new version literal appears in a `*_test.go`
  outside `.cascade-frozen` (belongs in `add-deps-cascade-task`, which first gives the file a
  reader). Also out: the `.cascade-hold` file, any cascade workflow, a text sync of the core line
  in `testdata/cue.mod` and `testdata/render/**` (the cascade's job, not `cue:deps:update`'s), and
  the docs examples (`docs/getting-started.md`, `AGENTS.md:329`) that show an explicit
  `OCILoader{Module: ...}` pin as user-facing illustration.

## Capabilities

### New Capabilities

- `fixture-pin-maintenance`: how the library's test cue.mod pins are updated (one resolution
  pass, core held at the default, loud failure that names the dependency forcing a newer core),
  and how test files that keep a version literal on purpose are declared.

### Modified Capabilities

- `render-parity`: the shipped case table takes the catalog build from the parity module's pin
  instead of a literal.
- `schema-dispatch`: in "DefaultSchemaModule constant", the fixture harness's declared core
  version is derived from the constant, the pin assertion checks shape instead of repeating the
  release, no test spells the default release where it means the current core, and doc comments
  cite the default by the constant's name.

## Impact

- Packages (test code and doc comments only): `opm/kernel` (parity harness, render tests),
  `opm/internal/registrytest` (`DefaultCoreVersion` becomes a derived package-level value),
  `opm/internal/renderstage` (`stage_test.go`), `opm/schema` (`loader_test.go`, and doc comments
  in `loader.go` and `cache.go`).
- Fixtures: header comments in 12 `platform.cue` files (11 under `testdata/render/platform*/`
  plus `modules/opm_platform`); catalog pins in `modules/opm_platform`,
  `testdata/modules/web_app`, `testdata/parity` and `testdata/parity/opm_platform` (section 5).
- Tooling: `Taskfile.yml` `cue:deps:update`; `.github/workflows/cue.yml` comment;
  `cue-versions.yml` header comment; new root `.cascade-frozen`.
- Downstream consumers (cli, opm-operator): none. They do not import `opm/internal/registrytest`.
- No `enhancement.yaml`: the release cascade is not an enhancement. Its design lives in the
  workspace RELEASING.md.
