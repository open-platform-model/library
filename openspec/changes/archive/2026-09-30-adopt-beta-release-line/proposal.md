## Why

OPM leaves alpha: core publishes `opmodel.dev/core@v2.0.0-beta.1` (gate G1), and every
prerelease line (core, catalogs/k8s, library, cli, opm-operator) moves to a beta line whose
promise is stated once and kept. The library's default schema pin, its served fixtures and
its release configuration still name the alpha line (`opmodel.dev/core@v2.0.0-alpha.13`,
`prerelease-type: alpha`), and its README still describes a pre-beta stability story. Until
the library pins the beta core and cuts `v1.0.0-beta.1`, neither cli nor opm-operator can
adopt the beta line (gate G2).

## What Changes

- `schema.DefaultSchemaModule` moves to `opmodel.dev/core@v2.0.0-beta.1`, and with it
  `schema.DefaultSchemaVersion()`, the core a generated platform module pins by default
  (`opm/helper/platformmodule`). The doc comment keeps its history true: `2.0.0-alpha.13`
  stays named as the first release reporting contract collisions, and beta.1 is stated as
  the release carrying that content.
- `registrytest.DefaultCoreVersion` moves with it; every `cue.mod` pinning core
  `v2.0.0-alpha.13` under `testdata/**` and `modules/**` is re-pinned to `v2.0.0-beta.1`
  (fixture header comments naming the core move too). Only the core pin moves: every other
  dependency pin (`opmodel.dev/catalogs/opm@v4` at `v4.4.2`, `cue.dev/x/k8s.io@v0` at
  `v0.12.0`) stays, and a catalog move is a later, separate `test(fixtures)` PR. Deliberately
  older floor and skew fixtures (`alpha.4`, `alpha.6`, `alpha.7`, `alpha.9` to `alpha.12`, the
  synthetic closure graph) stay as they are.
- Tests carrying the default as a literal move with it; `render_core_floor_test.go` reads
  `registrytest.DefaultCoreVersion` instead of a literal so the next move needs no edit there.
- `release-please-config.json`: `prerelease-type` `alpha` to `beta`, so beta.2 and later follow
  the beta counter. The first crossing to `1.0.0-beta.1` comes only from a one-shot
  `Release-As: 1.0.0-beta.1` footer in the squash commit on main; `release-as` never enters the
  config and the manifest is never hand-edited.
- README "API stability" states the beta promise and replaces "a breaking change here is a
  major bump" with its pre-GA and GA forms; it also tells consumers to pin an explicit
  `v1.0.0-beta.N`, because Go's `@latest` resolves the retired `v0.7.0`. Principle VI in
  `CONSTITUTION.md` and the `openspec/config.yaml` context gains the matching pre-GA clause,
  carrying the full promise (stable lines keep normal SemVer, the GA carrier rule) in the same
  wording as the cli and opm-operator constitutions.
  `migrations/README.md`, `AGENTS.md` and a new ADR-010 record that during beta a breaking
  change's migration note is the `BREAKING CHANGE:` footer of its `feat!` commit as the
  CHANGELOG shows it, and that `migrations/` stays dormant until GA (ADR-004 unchanged: GA arms
  it). `AGENTS.md` layout line, `docs/getting-started.md` pin example and the
  `docs/site/embedding/embed-the-kernel.md` page brief move to the beta line.
- Not **BREAKING** for the Go API: no `opm/` type, signature or behavior changes except the
  default core release a zero-value `OCILoader` and a generated platform module resolve.

SemVer class: PATCH (`fix(deps)`), released as `1.0.0-beta.1` by the carrier footer.

Release deliverable: this change's deliverable is itself a release crossing (the library's
first beta), so under the tasks-rule exception in `openspec/config.yaml` ("a change whose
stated deliverable is itself a release or publishing operation") the PR and the carrier footer
are part of the implementation: tasks.md names them in its unnumbered release block. One PR
(the default delivery mode), merged by the supervisor.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `schema-dispatch`: the "DefaultSchemaModule constant" requirement names
  `opmodel.dev/core@v2.0.0-beta.1` as the verified default and keeps `2.0.0-alpha.13` only as
  the release that first carried the collision report.
- `platform-artifact`: the "Platform Type Shape" collision scenario describes the served
  colliding platform as pinning the default core release instead of the `alpha.13` literal.
- `migration-docs`: "Pre-GA dormancy" covers the beta period and names the `BREAKING CHANGE:`
  footer of a `feat!` commit as the beta migration note; the guidance file is `AGENTS.md`.

## Impact

- Code: `opm/schema/loader.go` (constant + doc), `opm/internal/registrytest/registrytest.go`,
  doc-comment examples in `opm/schema`, `opm/catalog`, `opm/internal/renderstage` that cite the
  default. No other `opm/` source changes.
- Fixtures: 35 `cue.mod/module.cue` files under `testdata/` and `modules/` (core pin only),
  12 platform headers.
- Docs and governance: `README.md`, `CONSTITUTION.md` and `openspec/config.yaml` (Principle VI
  pre-GA clause), `AGENTS.md`, `migrations/README.md`, new `adr/010-*`,
  `docs/getting-started.md`, `docs/site/embedding/embed-the-kernel.md`.
- Tests: `opm/schema/loader_test.go`, `opm/kernel/render_core_floor_test.go`,
  `opm/kernel/render_test.go`, `opm/internal/renderstage/modfile_test.go`, `stage_test.go`,
  `opm/helper/platformmodule/generate_test.go`, `opm/catalog/requires_test.go`,
  `opm/internal/registrytest/registrytest_test.go`.
- Release: `release-please-config.json`; open release PR #151 (`release 1.0.0-alpha.37`) is
  retitled by the carrier to `chore(main): release 1.0.0-beta.1` and must never be merged as
  alpha; it is held (hold comment posted) and only the supervisor merges it, at G2.
- Downstream: cli (C3) and opm-operator (C4) bump to `v1.0.0-beta.1` after G2; from then their
  generated platform modules pin core `v2.0.0-beta.1`. Go `@latest` still resolves the stale
  `v0.7.0`; the README now says so and consumers pin explicitly, as they already do. Retiring
  `v0.7.0` stays a GA item.
- Precondition: gate G1 (core `v2.0.0-beta.1` on GHCR). Nothing in sections 1 to 4 can be
  verified before it. If G1 lands on another `v2.0.0-beta.N`, that tag replaces `beta.1` for
  core everywhere in this change (see the tasks.md Gates block); the library's own target
  stays `1.0.0-beta.1`.
