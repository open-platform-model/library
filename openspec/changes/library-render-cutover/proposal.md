## Why

`Kernel.Render` exists beside the old pipeline: `Materialize` (one OCI pull per subscription, Go-indexed transformers on a `MaterializedPlatform` twin), `Match` (the Go matcher reading `#matchers`), `Compile` (`FillPath` of three inputs into transformers reached through a shared platform value). The old path reads `#Subscription.version!` and `#Platform.#matchers`, both removed by core `v2.0.0-alpha.7` (0019 D5/D17), so the kernel is pinned to `alpha.6` and cannot consume the platform shape every other 0019 slice now writes (the CLI's platform module, the operator's generated module). The old path is also the measured-expensive one: it races under concurrent render, retains 348 MB per render, and serialised loses 2.5x to 5.5x to a shares-nothing worker (0019 experiments 06 to 08).

This change is `library-render-cutover`, the last library slice of 0019 Phase B. It makes `Render` the kernel's only render path, re-pins core to `alpha.7`, deletes the old pipeline and its twin, and supersedes ADR-002 with the shares-nothing rule (D8). It implements 0019 D8 and completes D9/D10/D13; it declares them in `enhancement.yaml`.

**Scope statement (Principle VIII).** The re-pin forces the deletion: once the default schema is `alpha.7`, `Materialize` cannot evaluate a platform, so the old path cannot survive the re-pin by even one release. The change is therefore one OpenSpec change delivered as an ordered PR train of three (proof, cutover, docs) rather than three changes, the same shape `library-render-build` used. Each PR is independently green.

**Prerequisite, not owned here:** 0019 records where the single-provider guard (0010 D32/D37, today `providerGuard` in `opm/materialize`) lives after federation dies. This proposal assumes the answer is an in-build diagnostics row plus fail-closed gate in the render glue (the D10 shape; 0015 later relocates it to platform-package generation with the D1 inventory). If 0019 decides otherwise, PR 1's guard tasks change and nothing else does.

## What Changes

**PR 1, additive on the current pin (proof and guard):**

- The old-versus-new proof: the parity harness renders every parity case through `Compile` (against the `alpha.6` subscription platform) and through `Render` (against a D5-shaped twin of the parity platform pinned to `alpha.7`, importing the same published catalog bytes) and asserts the rendered values byte-equal per case, order-sensitively, before anything is deleted. Rendered-object ordering: `Render` emits CUE's natural order (0019 D14); the proof records any ordering-only difference per case exactly as the existing harness does.
- The single-provider guard ported into the render glue: a diagnostics row (`overSubscribed`, naming the contract key and every registry key whose transformers require it) folded over the enabled `#registry` entries' required demands, and the D28 gate extended so an over-subscribed provider-fulfilled key refuses the render with a decoded typed error. Divergent-fulfilment copies cannot occur in one build (one set of catalog bytes) and are not ported.

**PR 2, the cutover (BREAKING, `feat(kernel)!:`):**

- `schema.DefaultSchemaModule` moves from `alpha.6` to `alpha.7`; `modules/opm_platform`, `testdata/parity`, `testdata/modules/web_app` and every fixture `cue.mod` re-pin; the two platform fixtures rewrite from `version!` subscriptions to `#CatalogEntry` imports (a `cue.mod` dependency on `opmodel.dev/catalogs/opm@v4`). The platform shape gate validates every `#registry` entry for completeness, so the old subscription shape is refused at acquisition (design.md § shape gate).
- **BREAKING** `Kernel.Compile`, `Kernel.Match`, `Kernel.Materialize`, `Kernel.SynthesizePlatform` and their types (`CompileInput`, `MatchInput`, `CompileResult`, `MatchPlan`, `synth.PlatformInput`, `synth.SubscriptionSpec`) are removed. `Render` is the sole render entry; a dry run is `Render` with `Compiled` discarded (CUE evaluates every pair inside the build regardless, 0019 experiment 08).
- **BREAKING** `opm/materialize` is deleted (`MaterializedPlatform`, `Materialize`, `CueContextOwner`, `MaterializeError` consumers re-home). `opm/compile` is deleted except `UnmatchedComponentsError`, which moves to `opm/errors` (the operator and `RenderError` already depend on it). `opm/schema/context.go` (the Go context mirror, made redundant by core D12 in `alpha.7`) and `opm/helper/synth/platform.go` are deleted; unread `schema` path constants go with them.
- `opm/kernel/compile.go`, `phases.go`, `inputs.go`, `results.go`, `materialize.go` and the old-path tests (`component_fill_test.go`, `instance_fill_test.go`, `materialize_test.go`, `phase_test.go`, `synth_platform_test.go`, the `flow_*` and `integration_*` tests that drive `Compile`) are deleted or rewritten onto `Render`; the parity harness targets `Render` only and its D30 pair-set exemption is retired (plain `&` in the glue, D10).
- `MIGRATIONS.md`/`migrations/`: no fragment, per ADR-004 (pre-GA, dormant); the break is recorded by release-please and this archive, and both consumers migrate in the same wave.

**PR 3, supersession and documentation (D8):**

- ADR-005 states the two rules: each render is its own CUE build in its own `cue.Context`, and that context does not outlive the render; nothing built is shared between renders; concurrency is across renders, never within one; a render pool is sized by memory (61 MB + 7.75 MB per component, 0019 `06-operational.md`). ADR-002's status becomes "Superseded by ADR-005"; ADR-003's federation rationale is retired in place (the no-cross-build-fill principle stands; its multi-version-composition premise was deleted by 0010 D14).
- `opm/kernel/doc.go` drops the materialize-era goroutine text and the mutex stopgap, documents `Render` as the phase surface, and states the shares-nothing contract; `README.md`, `docs/getting-started.md` and `CLAUDE.md` (the Materialize lifetime section, the layout table) follow.

## SemVer classification

MAJOR (Principle VI): public methods, types and two packages leave `opm/`. Pre-GA, so no migration fragment (ADR-004); the library's alpha line absorbs it, which 0019 `06-operational.md` names as the sequencing constraint (land before `library` declares `v1.0.0`).

## Affected packages and downstream consumers

- Deleted: `opm/materialize`, `opm/compile` (bar the relocated error), `opm/helper/synth/platform.go`, `opm/schema/context.go`. Touched: `opm/kernel` (surface removal, harness and tests onto `Render`), `opm/errors` (relocated error, over-subscription error), `opm/internal/renderstage` (guard row + gate), `opm/schema` (pin flip, path prune), `opm/platform` (doc.go), `modules/opm_platform`, `testdata/*`.
- **`cli`** (`internal/platform/materialize.go`, `internal/workflow/render/{kernel,render,types}.go`): `SynthesizePlatform` + `Materialize` + `Compile` become `AcquirePlatformFromDir` + `Render`; `compile.ComponentSummary` disappears from `CompileResult` (the frontend derives its summary from `RenderDiagnostics.Pairs` and the instance). That is `cli-render-switch`, which cannot land before this change's PR 2 and cannot lag it: the wave re-pins together.
- **`opm-operator`** (`internal/controller/platform_controller.go`, `internal/platform/store.go`, `internal/render/kernel_*_renderer.go`, `internal/reconcile/resolution.go`): the held `*MaterializedPlatform` slot and the `AcquireKernel` serialisation stopgap (PR 110) retire; renders go through `Render` against the generated platform module (`operator-platform-module-generation`); `compile.UnmatchedComponentsError` becomes `oerrors.UnmatchedComponentsError`. That is `operator-render-switch`, same wave.
- **`catalog_opm` / `modules`**: no artifact impact.

## Complexity justification

Net deletion: two packages, four kernel files, one helper, one mirror, and the materialize-era contract text, against one relocated error type and one glue row. The single build that replaces them is already landed and measured (`library-render-build`); this change removes the second implementation of the same pipeline.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `single-build-render`: `Render` becomes the sole render path (the "existing pipeline is unaffected" requirement is removed); the in-build single-provider guard is added; the behavioral matching requirements the Go matcher carried (unresolved-demand diagnosis with alternatives, trait posture, label predicate) are restated against the build.
- `render-parity`: the oracle compares against `Render`; the pair-set exemption is retired; the equality requirement becomes structural-only; the old-versus-new proof is recorded as the requirement that gated deletion.
- `transform-input-fill`: the three inputs reach a transformer by unification inside the build, not by `FillPath`; the preserved-field-class and whole-instance requirements survive with their mechanism restated.
- `kernel-runtime`: the goroutine-safety contract becomes shares-nothing; the phase-method, phase-input, Materialize-method and compile-context requirements are removed; the registry option, Tier-2 validation and no-utility-methods requirements are restated against `Render`.
- `platform-artifact`: the phase-input platform and binding-path-constant requirements are removed; the staged-source requirement states that `Render` reads `Source`.
- `helper-packages`: `synth.Platform` is removed from the helper set; the loader shape gate validates every platform `#registry` entry for completeness.
- `schema-dispatch`: `DefaultSchemaModule` is specified as the pinned `alpha.7` release; the path inventory shrinks to the paths that keep a reader; the transformer-context builder and the `compile.Match` signature requirements are removed.
- `instance-synthesis`: the imported-module render coverage renders through `Render`.
- `artifact-types`: the internal call-site requirement is restated against the `opm/schema` path vars and the by-import render build.
- `platform-materialization`: removed entirely.
- `platform-matching`: removed entirely (behavior carried by `single-build-render`).
- `platform-synthesis`: removed entirely.

## Impact

- Code as listed above; `Taskfile.yml` race targets (`opm/kernel`, `opm/compile`) drop the deleted package.
- Docs: ADR-005 (new), ADR-002/003 status headers, `opm/kernel/doc.go`, `opm/platform/doc.go`, `README.md`, `docs/getting-started.md`, `CLAUDE.md`, `docs/design/compile-pipeline-known-gaps.md` (materialize-era notes retired).
- `enhancement.yaml` declares 0019 D8, D9, D10, D13 and D26 (the in-build single-provider guard).
