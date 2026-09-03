# Design: library-render-cutover

## Context

See `proposal.md` § Why. `Kernel.Render` (landed, `library-render-build`) and the old pipeline coexist. The old pipeline is bound to the `alpha.6` core pin: `opm/materialize` reads `#Subscription.version!` and fills `#matchers`, both gone in `alpha.7`; `opm/schema/context.go` hand-builds the context core now projects (D12); `synth.Platform` writes subscription scalars. `schema.DefaultSchemaModule` is `opmodel.dev/core@v2.0.0-alpha.6`; `testdata/render/*` already pins `alpha.7` explicitly and is the only D5-shaped fixture set.

Frontend call sites today: `cli/internal/platform/materialize.go` (`SynthesizePlatform` + `Materialize`), `cli/internal/workflow/render/{kernel,render,types}.go` (`Compile`, `compile.ComponentSummary`), `opm-operator/internal/controller/platform_controller.go` (`SynthesizePlatform` + `Materialize`), `opm-operator/internal/platform/store.go` (`*MaterializedPlatform`), `opm-operator/internal/render/kernel_*_renderer.go` (`Compile`), `opm-operator/internal/reconcile/resolution.go` (`compile.UnmatchedComponentsError`). Zero direct `synth.Platform` callers outside the kernel wrapper.

## Goals / Non-Goals

**Goals**

- One render path. Every kernel test that rendered through `Compile` renders through `Render`; the parity harness targets `Render`.
- Prove before deleting: byte-level `Compile`-versus-`Render` agreement on the parity cases is the gate for PR 2.
- The single-provider guard survives federation's death, inside the build.
- D8 written down: ADR-005, the supersession headers, the doc.go contract.

**Non-Goals**

- No frontend edits (`cli-render-switch`, `operator-render-switch` own them; this change names the call sites only).
- No component summary on `RenderResult` (`compile.ComponentSummary` was presentation data; the CLI derives it from `RenderDiagnostics.Pairs` plus the instance, or adds a helper on its side).
- No render memoization, pooling or scheduling (D8; pool sizing is the consumer's, by memory).
- No 0015 work (the guard's eventual relocation to platform-package generation with the D1 inventory).

## Research & Decisions

### PR train inside one change, ordered proof → cutover → docs

**Context**: Principle VIII's gate names "redesigning the compile pipeline in one go" as oversized; the re-pin and the deletion cannot be separated (an `alpha.7` default schema breaks `Materialize` on every cold cache).
**Explored**: three OpenSpec changes (proof, cutover, docs): the proof change would exist only to be deleted by the next, and the docs change would leave `doc.go` describing deleted types for one release; a single PR (rejected outright by the gate).
**Decision**: one change, three PRs, each green on its own: (1) additive proof + guard port on the `alpha.6` pin; (2) re-pin + surface removal + deletion + tests onto `Render`; (3) ADR-005, supersession headers, doc rewrite. The same shape `library-render-build` used.
**Rationale**: PR 1 is the evidence PR 2 needs and is worthless once PR 2 lands, so it is a step, not a change; PR 3 is separable because `doc.go`'s minimal correctness (no references to deleted identifiers) is PR 2's, and the narrative rewrite is PR 3's.

### Delete `Compile`/`Match` rather than rewire them onto `Render`

**Context**: `library-render-build` left this open ("whether Render replaces Compile or Compile is rewired onto it").
**Explored**: keeping `Compile` as a thin wrapper over `Render` (its `CompileInput` takes `*MaterializedPlatform`, a type being deleted, so the wrapper would change signature anyway and carry two names for one verb); keeping `Match` as `Render`-minus-decode (no cheaper: CUE evaluates every pair inside the build, 3.0 ms of 1831 ms is decode, 0019 experiment 08).
**Decision**: remove `Compile`, `Match`, `Materialize`, `SynthesizePlatform` and their input/result types. `Render` is the verb; a dry run reads `RenderDiagnostics` and discards `Compiled`.
**Rationale**: two names for one path is the "two answers to one question" shape 0019 removes everywhere; the break is MAJOR either way and pre-GA.

### `opm/compile` goes; `UnmatchedComponentsError` moves to `opm/errors`

**Context**: `RenderError` and the operator both depend on `compile.UnmatchedComponentsError`; everything else in `opm/compile` is the Go matcher/executor.
**Explored**: keeping `opm/compile` as a one-type package (a package named for a deleted pipeline holding one error); re-declaring the type under `opm/kernel` (kernel already imports `opm/errors` for every other typed diagnostic).
**Decision**: move the type to `opm/errors` unchanged in shape (`Components`, `Matches`), delete the package. `MatchResult` (the per-component verdict map value) moves with it as the error's field type.
**Rationale**: all other typed gate causes (`UnresolvedDemandsError`, `TransformError`) already live there; one import for consumers.

### The single-provider guard as an in-build row and gate

**Context**: `providerGuard` dies with `opm/materialize`; 0019's decision on its home is this change's stated prerequisite, assumed here to be the glue.
**Explored**: keeping a Go check over the built platform value (a second matcher-adjacent Go walk beside a glue that already computes buckets from the same map; the shape D1 names as wrong direction); dropping enforcement until 0015 (a regression of 0010 D37 with no tripwire); the glue row.
**Decision**: `render.cue.tmpl` gains `overSubscribed`: for each enabled entry in `platform.#registry`, for each transformer in `entry.#transformers`, for each key in `requiredResources ∪ requiredTraits` whose value declares `fulfilment: "provider"`, collect the entry's registry key into a per-key set; rows are keys with more than one registry key. The registry key is the structural provenance materialize used (its subscription key): core binds it to `#catalog.metadata.modulePath`, so it is the `path@major` string, where a transformer's own `metadata.modulePath` is `<registryPath>/transformers` with no major and cannot tell two majors of one catalog apart. `diagnostics.overSubscribed` carries them; `gate` becomes `match.resolved & (len(overSubscribed) == 0) & true`. The kernel decodes rows into a new `oerrors.OverSubscribedContractError{Key, Catalogs}` carried on `RenderError`. The divergent-fulfilment arm is not ported: one build resolves every embedded copy to the same catalog bytes (0019 experiment 05 asserted them equal).
**Rationale**: verdicts-as-data is D10's shape, the row costs one comprehension, and the refusal keeps naming the key and both registry keys. Fixture: `testdata/render/scenarios` gains a two-catalog over-subscription scenario served by `registrytest`.

### A subscription-shaped platform is refused by the shape gate, not by core

**Context**: with the `alpha.7` pin the kernel must refuse the pre-D5 platform shape (a `version` scalar, no embedded catalog). Measured 2026-09-03 against `alpha.7`: `cue vet -c` passes it, `AcquirePlatformFromDir` accepts it with an empty `#composedTransformers`, and only `#registry.Validate(cue.Concrete(true))` reports `required field missing: version` (the D5 readout `version: #catalog.metadata.version` with no catalog behind it).
**Explored**: a core change making `#catalog!` required (defeated by core's own key binding, which supplies a partial `#catalog` on every entry; measured on a local core copy, identical outcome); letting the render gate catch it (every component unmatched, a misleading diagnosis for a wrong platform shape); validating the whole platform value (root `Validate` skips definitions, so it never reaches `#registry`).
**Decision**: the platform shape gate (`loader/internal/shape`, `PlatformSpec`) validates `#registry` under `cue.Concrete(true)` and wraps the result in `ErrMissingRequiredField`. No core change.
**Rationale**: the gate already owns platform identity checks, the incompleteness is core's D5 tripwire, and the refusal lands at acquisition where the CLI and operator already branch on the loader sentinels.

### The old-versus-new proof reuses the parity table

**Context**: `render-parity`'s oracle compares the kernel against pure CUE; what deletion needs is `Compile` against `Render` on the same cases.
**Explored**: a fresh comparison suite (duplicates the table, the encoder and the diff reporter); extending `parity_harness_test.go` with a second kernel-side renderer.
**Decision**: PR 1 adds `testdata/parity/opm_platform_next/` (the parity platform in `#CatalogEntry` form, `cue.mod` pinning `alpha.7` and `catalogs/opm@v4` at the same build the subscription platform names) and a test that renders each case through `Compile` (old platform) and `Render` (new platform) and asserts `compareRendered` equality, order-sensitively, using the existing encoder. PR 2 deletes the old side and points the oracle comparison at `Render`; `opm_platform_next` becomes `opm_platform`.
**Rationale**: the D14 ordering question was already settled by the harness's per-case records; the same machinery answers "did the collapse change a byte".

### Core pin flip and the floating-schema hazard

**Context**: `schema.OCILoader` floats on `opmodel.dev/core@v2` when `DefaultSchemaModule` is unpinned; the old path breaks on any cold cache the moment the default resolves to `alpha.7`.
**Decision**: `DefaultSchemaModule` becomes `opmodel.dev/core@v2.0.0-alpha.7` in PR 2, the same PR that deletes the old path; `opm/schema/loader_test.go`'s pin assertion moves with it. The workspace `task deps:update` bump for `library` is unblocked by PR 2 and performed as part of it (the deferred run `core-registry-import` named).
**Rationale**: the flip and the deletion are one atomic fact; splitting them leaves a release that cannot render at all.

### ADR-005 carries D8, ADR-006 carries D9; ADR-002 and ADR-003 get supersession headers, not rewrites

**Context**: ADR-002 is already marked retracted in place (library PR 93); ADR-003's no-cross-build-fill principle stands while its federation rationale is stale.
**Decision**: ADR-005 "Shares-nothing renders" records the two rules (own build, own context; context does not outlive the render), the rejected alternatives with their measurements (mutex: 2.5x to 5.5x throughput and 348 MB retained; per-worker platform copies: 41.9 MB to 581.8 MB per render), and the pool-sizing consequence. ADR-002's status line becomes "Superseded by ADR-005". ADR-006 "Artifacts are constructed in one CUE build" records D9 (every artifact the kernel constructs is one build that imports its inputs; no cross-build fill) with its measurements and the rejected federation and caller-context-fill tactics; ADR-003's status line becomes "Superseded by ADR-006". A first draft gave ADR-003 a confirming paragraph instead, on the reading that only its rationale was stale; that reading was wrong, because ADR-003's decision explicitly rejected the single-build-only framing that is now the rule, so the decision changed, not its context.
**Rationale**: 0019 D8 asks for exactly this shape ("a superseded-by header on ADR-002, a new ADR carrying both rules"), and D9 is a separate decision with the same claim to a record; rewriting the old ADRs would erase the record the headers point at.

## Public surface changes (`opm/`)

Removed: `Kernel.{Compile,Match,Materialize,SynthesizePlatform}`, `kernel.{CompileInput,MatchInput,CompileResult,MatchPlan}`, package `opm/materialize` (all identifiers), package `opm/compile` (all identifiers), `synth.{Platform,PlatformInput,SubscriptionSpec}` and their sentinels, `schema.{BuildTransformerContext,ModuleInstanceContextData,ComponentContextData}`, `schema.Matchers` and any path constant left unread. Added: `oerrors.UnmatchedComponentsError` (relocated, same shape), `oerrors.MatchResult` (relocated), `oerrors.OverSubscribedContractError`, `RenderDiagnostics.OverSubscribed`. Unchanged: `Render` and its types, every acquire/synthesize-instance/validate method.

## Risks / Trade-offs

- [0019 decides the guard belongs elsewhere (0011 publish gate)] → PR 1's guard tasks are replaced by "delete without port" and the `single-build-render` guard requirement is dropped; nothing in PR 2/3 changes. The change is not started until the decision is recorded.
- [Byte differences surface in the proof (ordering beyond D14's env case, or a value)] → an ordering-only difference is recorded per case as the harness already does; a value difference is a defect in `Render` or the glue and blocks PR 2 until fixed. Pair-set agreement was already proven by `library-render-build` task 9.
- [Frontends cannot build for one release] → `cli` and `opm-operator` re-pin only with their switch changes, as with every earlier wave; the workspace `task deps:update` is run per repo, not globally, until the wave closes.
- [`compile.ComponentSummary` consumers] → CLI-side derivation from `RenderDiagnostics.Pairs` and the instance's components; named in the proposal's consumer table for `cli-render-switch`.
- [Deleting `flow_*`/`integration_*` tests loses coverage the new tests do not carry] → each deleted test is either ported onto `Render` against the D5-shaped fixtures or listed in the PR 2 description with the `render_test.go` scenario that covers it; a test with no counterpart is ported, not dropped.

## Migration Plan

Three PRs on `main`, in order, each green: (1) `test(kernel): prove Compile and Render render the parity table byte-equal` + `feat(render): single-provider guard in the render build`; (2) `feat(kernel)!: Render is the sole render path; drop materialize, compile and platform synthesis` (with the `alpha.7` pin flip and fixture rewrites in the same PR); (3) `docs(adr): ADR-005 shares-nothing renders; supersede ADR-002`. No migration fragment (pre-GA, ADR-004). Rollback of PR 2 before the wave closes is a revert; after `cli`/`opm-operator` re-pin, rollback is the wave's, not this repo's alone.

## Open Questions

None that change the specs or the task breakdown. The one deferrable is the exact `oerrors.OverSubscribedContractError` field set beyond `Key` and `Catalogs` (settle at implementation against what the CLI's diagnostics printer wants).
