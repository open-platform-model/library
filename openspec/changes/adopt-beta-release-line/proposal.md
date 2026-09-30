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
  (fixture header comments naming the core move too). Deliberately older floor and skew
  fixtures (`alpha.4`, `alpha.6`, `alpha.7`, `alpha.9` to `alpha.12`, the synthetic closure
  graph) stay as they are.
- Tests carrying the default as a literal move with it; `render_core_floor_test.go` reads
  `registrytest.DefaultCoreVersion` instead of a literal so the next move needs no edit there.
- `release-please-config.json`: `prerelease-type` `alpha` to `beta`, so beta.2 and later follow
  the beta counter. The first crossing to `1.0.0-beta.1` comes only from a one-shot
  `Release-As: 1.0.0-beta.1` footer in the squash commit on main; `release-as` never enters the
  config and the manifest is never hand-edited.
- README "API stability" states the beta promise. `migrations/README.md`, `AGENTS.md` and a new
  ADR-010 record that during beta a breaking change's migration note is its `BREAKING CHANGE:`
  footer as the CHANGELOG shows it, and that `migrations/` stays dormant until GA (ADR-004
  unchanged: GA arms it). `AGENTS.md` layout line and `docs/getting-started.md` pin example move
  to the beta core.
- Not **BREAKING** for the Go API: no `opm/` type, signature or behavior changes except the
  default core release a zero-value `OCILoader` and a generated platform module resolve.

SemVer class: PATCH (`fix(deps)`), released as `1.0.0-beta.1` by the carrier footer.

Delivery: this change's deliverable is itself a release crossing (the library's first beta),
so under the repo's tasks rule the PR and the carrier footer ARE part of the implementation:
tasks.md names them. One PR, merged by the supervisor.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `schema-dispatch`: the "DefaultSchemaModule constant" requirement names
  `opmodel.dev/core@v2.0.0-beta.1` as the verified default and keeps `2.0.0-alpha.13` only as
  the release that first carried the collision report.
- `platform-artifact`: the "Platform Type Shape" collision scenario describes the served
  colliding platform as pinning the default core release instead of the `alpha.13` literal.
- `migration-docs`: "Pre-GA dormancy" covers the beta period and names the
  `BREAKING CHANGE:` footer as the beta migration note; the guidance file is `AGENTS.md`.

## Impact

- Code: `opm/schema/loader.go` (constant + doc), `opm/internal/registrytest/registrytest.go`,
  doc-comment examples in `opm/schema`, `opm/catalog`, `opm/internal/renderstage` that cite the
  default. No other `opm/` source changes.
- Fixtures: 35 `cue.mod/module.cue` files under `testdata/` and `modules/`, 12 platform headers.
- Tests: `opm/schema/loader_test.go`, `opm/kernel/render_core_floor_test.go`,
  `opm/kernel/render_test.go`, `opm/internal/renderstage/modfile_test.go`, `stage_test.go`,
  `opm/helper/platformmodule/generate_test.go`, `opm/catalog/requires_test.go`,
  `opm/internal/registrytest/registrytest_test.go`.
- Release: `release-please-config.json`; open release PR #151 (`release 1.0.0-alpha.37`) is
  retitled by the carrier to `chore(main): release 1.0.0-beta.1` and must never be merged as
  alpha.
- Downstream: cli (C3) and opm-operator (C4) bump to `v1.0.0-beta.1` after G2; from then their
  generated platform modules pin core `v2.0.0-beta.1`. Go `@latest` still resolves the stale
  `v0.7.0`; consumers pin explicitly, as they already do.
- Precondition: gate G1 (core `v2.0.0-beta.1` on GHCR). Nothing in sections 1 to 4 can be
  verified before it.
