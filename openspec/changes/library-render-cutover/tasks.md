# Tasks: library-render-cutover

Three PRs in order (design.md § PR train). Task 0 gates PR 1's guard tasks (1.3 to 1.5); everything in PR 2 gates on PR 1 green.

## 0. Prerequisite (outside this repo)

- [ ] 0.1 Enhancement 0019 records the single-provider guard's post-federation home (glue row + gate assumed; see design.md). If the decision differs, replace tasks 1.3 to 1.5 with "delete without port" and drop the guard requirement from `specs/single-build-render/spec.md` before starting.

## 1. PR 1: proof and guard (additive, `alpha.6` pin)

- [ ] 1.1 `testdata/parity/opm_platform_next/`: the parity platform in `#CatalogEntry` form (`cue.mod` pinning `opmodel.dev/core@v2` at `v2.0.0-alpha.7` and `opmodel.dev/catalogs/opm@v4` at the build `opm_platform` subscribes to; `platform.cue` importing the catalog with one `#catalog:` entry).
- [ ] 1.2 `opm/kernel/parity_cutover_test.go`: for every `parityCase`, render through `Compile` (existing path, `opm_platform`) and `Render` (`opm_platform_next`), assert `compareRendered` equality order-sensitively; record ordering-only cases per the harness's existing per-case mechanism. Must pass before PR 2 opens.
- [ ] 1.3 `opm/internal/renderstage/render.cue.tmpl`: `overSubscribed` rows folded over the enabled `platform.#registry` entries (provider-fulfilled required keys mapped to the set of registry keys whose `#transformers` require them; rows where the set has more than one key), exported under `diagnostics`, and the gate extended with `len(overSubscribed) == 0`.
- [ ] 1.4 `opm/errors`: `OverSubscribedContractError{Key, Catalogs}`; `opm/kernel/render_decode.go`: decode rows into `RenderDiagnostics.OverSubscribed` and the gate error.
- [ ] 1.5 `testdata/render/scenarios`: a two-catalog over-subscription scenario (second `registrytest`-served catalog supplying a transformer requiring the same provider-fulfilled key); `render_test.go` cases: refused naming key and both paths; catalog-fulfilled plurality still renders.
- [ ] 1.6 `task check` green; old suite untouched.

## 2. PR 2: the cutover (`feat(kernel)!:`)

- [ ] 2.1 `opm/schema/loader.go`: `DefaultSchemaModule` to `opmodel.dev/core@v2.0.0-alpha.7`; update its test and the doc comment; run the workspace `task deps:update` for `library` only (the deferred bump) and commit its `cue.mod` diffs.
- [ ] 2.2 Fixtures: `modules/opm_platform` and `testdata/parity/opm_platform` rewritten to the `#CatalogEntry` form (delete `opm_platform_next`, it becomes `opm_platform`); `testdata/modules/web_app`, `testdata/parity`, `testdata/render/*` and every other fixture `cue.mod` on `alpha.7`; `opm/internal/registrytest` inline `cue.mod` literals likewise (`defaultCoreVersion`); the inline `alpha.6` literals in `opm/kernel/parity_probe_test.go` and `opm/internal/renderstage/modfile_test.go` (its skew table row keeps `alpha.6` only as the deliberate mismatched side). Acceptance: `grep -rn 'alpha\.6'` over `*.go`, `*.cue` and `*.md` outside `CHANGELOG.md`, `openspec/changes/archive/` and the compat regression case matches nothing but that skew row.
- [ ] 2.3 Relocate `UnmatchedComponentsError` and `MatchResult` to `opm/errors`; update `render_decode.go` and `RenderError`'s doc.
- [ ] 2.4 Delete `opm/materialize`, `opm/compile`, `opm/helper/synth/platform.go` (+ tests), `opm/schema/context.go` (+ test), `opm/kernel/{compile,phases,inputs,results,materialize}.go`, `synth_platform_test.go`, `materialize_test.go`, `phase_test.go`; prune `opm/schema/paths.go` to the inventory the `schema-dispatch` delta lists (`Metadata`, `Components`, `Values`, `Config`, `Module`, `ModuleMetadataPath`, `DebugValues`); drop `registrytest.CtxOwner`.
- [ ] 2.5 Port or retire the old-path tests: `component_fill_test.go`, `instance_fill_test.go`, `flow_integration_test.go`, `flow_synth_*_test.go`, `integration_*_test.go`, `owngraph_integration_test.go`, `kernel_test.go` cases that drive `Compile`/`Materialize`. Each becomes a `Render` test against the D5-shaped fixtures or is listed in the PR description against the `render_test.go` scenario that covers it; none is dropped without a counterpart.
- [ ] 2.6 Parity harness onto `Render` only: `parity_harness_test.go` kernel side uses `Render`; every case `structural`; the pair-set exemption code path deleted; `parity_cutover_test.go` deleted (its record is the PR).
- [ ] 2.7 `opm/kernel/doc.go` and `opm/platform/doc.go`: remove every reference to deleted identifiers (minimal correctness; the narrative rewrite is PR 3); `Taskfile.yml` race targets to `opm/kernel` + `opm/internal/renderstage`.
- [ ] 2.8 `opm/helper/loader/internal/shape`: `PlatformSpec` validates `#registry` under `cue.Concrete(true)` and wraps the failure in `ErrMissingRequiredField` (an entry with no embedded catalog fails as `required field missing: version`); `platform_test.go` cases for the refused subscription shape and the passing embedded shape.
- [ ] 2.9 `task check` green under `-race`; `go vet ./...` shows no dangling references; grep `cli/` and `opm-operator/` for the removed identifiers and list every hit in the PR description for the switch changes.

## 3. PR 3: supersession and docs (D8)

- [ ] 3.1 `adr/005-shares-nothing-renders.md` per design.md (two rules, rejected alternatives with measurements, pool sizing by memory); ADR-002 status "Superseded by ADR-005"; ADR-003 status paragraph retiring the federation premise.
- [ ] 3.2 `opm/kernel/doc.go`: goroutine section rewritten to the shares-nothing contract (one Kernel per goroutine for the context-owning methods; `Render` shares nothing; no mutex or shared-platform shape shown; pool sized by memory); phase section describes `Render` and the dry-run idiom.
- [ ] 3.3 `README.md`, `docs/getting-started.md` (materialize section becomes acquire-platform-module + `Render`), `CLAUDE.md` (layout table, "Materialize lifetime & registry contract" replaced by a `Render` contract paragraph, the localhost-mapping test list), `docs/design/compile-pipeline-known-gaps.md` (materialize-era notes retired).
- [ ] 3.4 `task check` green; `task docs`-equivalent lint if present.
