# Tasks: library-render-cutover

Three PRs in order (design.md § PR train). Task 0 gates PR 1's guard tasks (1.3 to 1.5); everything in PR 2 gates on PR 1 green.

## 0. Prerequisite (outside this repo)

- [x] 0.1 Enhancement 0019 records the single-provider guard's post-federation home (glue row + gate assumed; see design.md). If the decision differs, replace tasks 1.3 to 1.5 with "delete without port" and drop the guard requirement from `specs/single-build-render/spec.md` before starting.

## 1. PR 1: proof and guard (additive, `alpha.6` pin)

- [x] 1.1 `testdata/parity/opm_platform_next/`: the parity platform in `#CatalogEntry` form (`cue.mod` pinning `opmodel.dev/core@v2` at `v2.0.0-alpha.7` and `opmodel.dev/catalogs/opm@v4` at the build `opm_platform` subscribes to; `platform.cue` importing the catalog with one `#catalog:` entry).
- [x] 1.2 `opm/kernel/parity_cutover_test.go`: for every `parityCase`, render through `Compile` (existing path, `opm_platform`) and `Render` (`opm_platform_next`), assert `compareRendered` equality order-sensitively; record ordering-only cases per the harness's existing per-case mechanism. Must pass before PR 2 opens.
- [x] 1.3 `opm/internal/renderstage/render.cue.tmpl`: `overSubscribed` rows folded over the enabled `platform.#registry` entries (provider-fulfilled required keys mapped to the set of registry keys whose `#transformers` require them; rows where the set has more than one key), exported under `diagnostics`, and the gate extended with `len(overSubscribed) == 0`.
- [x] 1.4 `opm/errors`: `OverSubscribedContractError{Key, Catalogs}`; `opm/kernel/render_decode.go`: decode rows into `RenderDiagnostics.OverSubscribed` and the gate error.
- [x] 1.5 `testdata/render/scenarios`: a two-catalog over-subscription scenario (second `registrytest`-served catalog supplying a transformer requiring the same provider-fulfilled key); `render_test.go` cases: refused naming key and both paths; catalog-fulfilled plurality still renders.
- [x] 1.6 `task check` green; old suite untouched.

## 2. PR 2: the cutover (`feat(kernel)!:`)

- [x] 2.1 `opm/schema/loader.go`: `DefaultSchemaModule` to `opmodel.dev/core@v2.0.0-alpha.7`; update its test and the doc comment; run the library's `task cue:deps:update` (the deferred bump) so every fixture `cue.mod` re-pins.
- [x] 2.2 Fixtures: `modules/opm_platform` and `testdata/parity/opm_platform` rewritten to the `#CatalogEntry` form (delete `opm_platform_next`, it becomes `opm_platform`); `testdata/modules/web_app`, `testdata/parity`, `testdata/render/*` and every other fixture `cue.mod` on `alpha.7`; `opm/internal/registrytest` inline `cue.mod` literals likewise (`defaultCoreVersion`); the inline `alpha.6` literals in `opm/kernel/parity_probe_test.go` and `opm/internal/renderstage/modfile_test.go` (its skew table row keeps `alpha.6` only as the deliberate mismatched side). Acceptance: `grep -rn 'alpha\.6'` over `*.go`, `*.cue` and `*.md` outside `CHANGELOG.md`, `openspec/changes/archive/` and the compat regression case matches nothing but that skew row.
- [x] 2.3 Relocate `UnmatchedComponentsError` and `MatchResult` to `opm/errors`; update `render_decode.go` and `RenderError`'s doc.
- [x] 2.4 Delete `opm/materialize`, `opm/compile`, `opm/helper/synth/platform.go` (+ tests), `opm/schema/context.go` (+ test), `opm/kernel/{compile,phases,inputs,results,materialize}.go`, `synth_platform_test.go`, `materialize_test.go`, `phase_test.go`; prune `opm/schema/paths.go` to the inventory the `schema-dispatch` delta lists (`Metadata`, `Components`, `Values`, `Config`, `Module`, `ModuleMetadataPath`, `DebugValues`); drop `registrytest.CtxOwner`.
- [x] 2.5 Port or retire the old-path tests: `component_fill_test.go`, `instance_fill_test.go`, `flow_integration_test.go`, `flow_synth_*_test.go`, `integration_*_test.go`, `owngraph_integration_test.go`, `kernel_test.go` cases that drive `Compile`/`Materialize`. Each becomes a `Render` test against the D5-shaped fixtures or is recorded (in the change's design or the merged PR) against the `render_test.go` scenario that covers it; none is dropped without a counterpart.
- [x] 2.6 Parity harness onto `Render` only: `parity_harness_test.go` kernel side uses `Render`; every case `structural`; the pair-set exemption code path deleted; `parity_cutover_test.go` deleted (its record is the PR).
- [x] 2.7 `opm/kernel/doc.go` and `opm/platform/doc.go`: remove every reference to deleted identifiers (minimal correctness; the narrative rewrite is PR 3); `Taskfile.yml` race targets to `opm/kernel` + `opm/internal/renderstage`.
- [x] 2.8 `opm/helper/loader/internal/shape`: `PlatformSpec` validates `#registry` under `cue.Concrete(true)` and wraps the failure in `ErrMissingRequiredField` (an entry with no embedded catalog fails as `required field missing: version`); `platform_test.go` cases for the refused subscription shape and the passing embedded shape.
- [x] 2.9 `task check` green under `-race`; `go vet ./...` shows no dangling references; grep `cli/` and `opm-operator/` for the removed identifiers and record every hit for the switch changes (`cli-render-switch`, `operator-render-switch`).

## 3. PR 3: supersession and docs (D8)

- [x] 3.1 `adr/005-shares-nothing-renders.md` per design.md (two rules, rejected alternatives with measurements, pool sizing by memory); ADR-002 status "Superseded by ADR-005"; `adr/006-single-build-artifact-construction.md` (D9: one build per artifact, no cross-build fill); ADR-003 status "Superseded by ADR-006".
  - [x] 3.1.1 ADR-002 line 9 ("Until that lands, `opm/kernel`'s package documentation names the two supported shapes ...") retired: 0019 D8 landed, `opm/kernel/doc.go` now states the shares-nothing contract.
  - [x] 3.1.2 ADR-003 superseded by ADR-006: the status header names the deleted federation application (`indexCatalogs`, `MaterializedPlatform.Transformers` / `.Matchers`, "read the materialized index off ...") and the premise 0010 D14 and 0019 D5 removed; pointers in `opm/platform/doc.go`, `opm/helper/synth` and `README.md` redirected to ADR-006.
- [x] 3.2 `opm/kernel/doc.go`: goroutine section rewritten to the shares-nothing contract (one Kernel per goroutine for the context-owning methods; `Render` shares nothing; no mutex or shared-platform shape shown; pool sized by memory); phase section describes `Render` and the dry-run idiom.
- [x] 3.3 `README.md`, `docs/getting-started.md` (materialize section becomes acquire-platform-module + `Render`), `CLAUDE.md` (layout table, "Materialize lifetime & registry contract" replaced by a `Render` contract paragraph, the localhost-mapping test list), `docs/design/compile-pipeline-known-gaps.md` (materialize-era notes retired).
  - [x] 3.3.1 `README.md`: line 10 (`#matchers` index), 29 (Platform row: `#composedTransformers` is read by the glue in CUE), 45-51 (layout rows `compile/`, `materialize/`, `synth/ ... Platform`), 63-80 ("Compile pipeline" section and the two-phase-methods sentence become the `Render` description), 86 (quick start wording); also the pre-existing stale 114-115 (`LoadPlatformFile`, `opm/helper/platform`).
  - [x] 3.3.2 `CLAUDE.md`: 86 (flow-notes pointer), 98 (`compile/` layout row; add `opm/internal/registrytest` and the relocated `opm/errors/unmatched.go`), 171-201 (the whole materialize contract section), 246 (`cue:test:flow` comment), 255-276 (kernel API surface and "Compile pipeline" block: `Render` is the sole render path, consumed by the switch changes), 290 (`opm/compile` adapts), 303 (commit scopes drop `compile`, `materialize`, gain `render`, `renderstage`).
  - [x] 3.3.3 `CONSTITUTION.md`: 63 (entry-point verbs), 66 and 68 (`opm/compile/`, `opm/materialize/` rows), 76 (flow diagram `compile` step).
  - [x] 3.3.4 `docs/getting-started.md`: 3, 41, 116-135 (materialize section), 137-155 (`Compile` section), 163-166 (phase table, dry-run sentence), 174-175 (migration rows).
  - [x] 3.3.5 `docs/design/`: retire `compile-pipeline-known-gaps.md` (the change names it); mark `per-version-build-isolation-c2.md` moot (its premise, `Materialize` selecting same-major versions, no longer exists); add a "the code discussed no longer exists" note to `transformer-output-hidden-field-scope-bug.md` and to `cue-closedness-regression-alpha2.md:316`.
- [x] 3.4 `task check` green; `task docs`-equivalent lint if present.
