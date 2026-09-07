## Why

The render build already decides every matching verdict as CUE data, but the Go decoder re-derives, re-joins and re-formats those verdicts before handing them out, and the kernel writes English sentences that belong to the frontends. Five facts on `main` today:

1. **Alternatives are computed in Go from a raw universe.** The glue exports `bucketKeys` (every contract key the platform's transformers name) "for the kernel's same-base alternatives", and `opm/internal/renderstage/alternatives.go` filters and sorts them on the kube apiVersion ladder with `compat.CompareAPIVersions`. That import is the only reason a kernel-internal package depends on `opm/compat`, a publish-side package, and it blocks slice 5 (moving `compat` to the cli).
2. **The decoder joins and groups what the glue already keyed.** `unresolved[].disqualified` names transformer FQNs and `unifyFailures[]` carries their conflicts; `render_decode.go:113-141` builds a map to join them back, although `_unify[tfqn].conflicts` sits beside the `disqualified` comprehension in the glue. `candidates` arrives flat; Go groups it into a match matrix (`:162-175`) and `gateErrors` regroups it again for the unmatched components (`:201-211`). `missing` and `resolved` are decoded and never read (deferred here by `cut-dead-surface`).
3. **The error package mixes rows with errors and promises what it cannot keep.** `UnifyError.Cause` is documented as "the CUE error tree verbatim" and pinned by a test named for it, but the glue cannot export a CUE error from inside the build, so the kernel fills it with `fmt.Errorf`. `UnmatchedComponentsError.Unwrap` fabricates one `*TransformError` per unmatched component that no consumer reads. `TransformError` spells `ComponentName`/`TransformerFQN` where every sibling spells `Component`/`Transformer`. Four types have value receivers and four have pointer receivers, so the operator writes `errors.AsType[oerrors.IdentityError]` beside `errors.AsType[*oerrors.SkewError]`. The match matrix lives only on `UnmatchedComponentsError.Matches`, never on `RenderDiagnostics`, so a partial dry run cannot say why a component did not match.
4. **The kernel formats prose.** `RenderResult.Warnings []string` is English built from `Diagnostics.ResolvedVersions[].Newer` (`render.go:217-219`) and `Diagnostics.UnhandledTraits` (`render_decode.go:290-307`). Both consumers pass the strings through untouched: the cli logs each one, the operator emits one event per distinct string. Principle IV says presentation stays outside the library; the cli already overrides the kernel's unresolved-demand text with its own formatter, and the operator would rather emit structured event fields.
5. **`opm/core` is a package for one type.** `Compiled` (`opm/core/compiled.go`, 29 lines) is `Render`'s terminal output; nothing else in the library produces or consumes it, and both consumers wrap it on arrival (`opm-operator/pkg/core/compiled_adapter.go:15`, `cli/internal/workflow/render/render.go:143`). The package boundary buys no reuse and costs the operator an import; the cli reaches the fields through `RenderResult` without one. Slice 5 listed the move; it lands here, first, because every later task in this change rewrites the two files that name the type.

This is slice 4 of the eight-slice simplification plan reviewed on 2026-09-05. It preserves the semantics of enhancement 0010 D28 (fail-closed demand gate), D34/D4 (same-base alternatives on the apiVersion ladder) and 0019 D10 (verdicts as data); only the export shape of the diagnostics and the error types move. No enhancement decision is implemented or resolved, so this change carries no `enhancement.yaml`.

**Scope statement (Principle VIII).** Four packages are touched (`opm/internal/renderstage`, `opm/kernel`, `opm/errors`, and `opm/core`, which is deleted) for one concern: the shapes the build emits and the kernel exports. The tasks are ordered the `Compiled` move first (one mechanical commit), glue and decoder second, error package third, `Warnings` fourth, so a reviewer can stop after the first or the second group and still have a green tree and a shippable library. If the owner prefers two changes, the split is after the decoder group.

## What Changes

**`opm/internal/renderstage` (the glue and its decoder contract):**

- The glue's `diagnostics.unresolved[]` rows gain `alternatives` (same-base contract keys the platform implements, sorted in-build on the `alpha < beta < GA`, then major, then minor ladder) and `disqualified` becomes `[{transformer, conflicts}]` from the `_unify` verdict the comprehension already reads. `diagnostics.unifyFailures[]` keeps its shape.
- `diagnostics.unmatched[]` replaces `unmatchedComponents` and the flat `candidates`: one row per unmatched component carrying `{component, candidates: [{transformer, matched, missingLabels}]}`.
- `bucketKeys`, `missing` and `resolved` leave the `diagnostics` export. `match.resolved` stays internal and still feeds `gate`.
- `alternatives.go` is deleted with its `opm/compat` import. `renderstage` imports nothing outside `opm/internal` and `opm/module`.
- The `gate:` field stays as the module's own fail-closed statement; the kernel's `gateErrors` remains the authoritative refusal that produces typed causes, and the render tests pin that the two agree.

**`opm/kernel`:**

- **BREAKING** `core.Compiled` becomes `kernel.Compiled`, same four fields (`Value`, `Instance`, `Component`, `Transformer`), declared beside the verb that produces it; `RenderResult.Compiled` is `[]*Compiled`. `opm/core` is deleted.
- `decodeRenderDiagnostics` decodes flat: no `byCandidate` join, no `matchMatrix`, no regroup in `gateErrors`. `glueDiagnostics` mirrors the new export exactly.
- **BREAKING** `RenderDiagnostics.Unmatched` becomes `[]oerrors.UnmatchedComponent` (component plus its candidate verdicts); `RenderDiagnostics.Unify` becomes `[]oerrors.UnifyRefusal` (one row per disqualified candidate carrying its conflicts); `RenderDiagnostics.OverSubscribed` becomes `[]oerrors.OverSubscribedContract` (a row, not an error).
- **BREAKING** `RenderResult.Warnings` is removed. `SkewWarn` renders and marks the row `Newer` on `Diagnostics.ResolvedVersions`; unhandled optional traits stay on `Diagnostics.UnhandledTraits`. Frontends format both.

**`opm/errors` (BREAKING):**

- Rows are data with no `Error` method: `UnresolvedDemand` (`Disqualified []UnifyRefusal`), `UnifyRefusal{Component, Transformer, Conflicts []string}` (replaces `UnifyError`), `UnmatchedComponent{Component, Candidates []CandidateVerdict}`, `CandidateVerdict{Transformer, Matched, MissingLabels}` (replaces `MatchResult`), `OverSubscribedContract{Key, Catalogs}` (replaces `OverSubscribedContractError`).
- Gate causes are pointer-receiver aggregates over those rows: `*UnresolvedDemandsError{Demands}`, `*UnmatchedComponentsError{Components []UnmatchedComponent}` (no `Matches` map, no fabricated `Unwrap`), `*OverSubscribedContractsError{Contracts}` (new; today each row is joined into the gate bare).
- `*TransformError{Component, Transformer, Cause}`: fields renamed to the family's vocabulary.
- `UnresolvedDemandsError.Unwrap() []error` is removed with the rows' `Error` methods; `Error()` on each aggregate keeps listing its rows.

**Docs:** `CLAUDE.md` (Render contract), `opm/kernel/doc.go`, `docs/getting-started.md` (the `result.Warnings` loop becomes a loop over the diagnostics rows), `opm/errors` package doc; every `core.Compiled` mention in `CLAUDE.md`, `README.md`, `CONSTITUTION.md` III, `docs/getting-started.md` and the verify skill reads `kernel.Compiled`.

**Not in this change:** moving `opm/compat` (slice 5, unblocked by this change); `IdentityError`'s receiver (loader path, `one-api-tier` territory); the cli's and operator's own `pkg/errors.TransformError` copies (slice 7); any change to matching semantics, the parity oracle, or the single-provider guard's computation.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `single-build-render`: the in-build matching requirement states that alternatives, per-candidate disqualification and the unmatched candidate matrix are computed by the glue and decoded without derivation; the skew and trait-posture requirements report on the diagnostics rows instead of a warnings surface; the unresolved-demand requirement names the row shape; a new requirement forbids presentation strings on a render result and states the typed gate causes carry the diagnostics rows unchanged; a new requirement pins that the module's own `gate` agrees with the kernel's refusal; the rendered-output requirement names `kernel.Compiled` and pins that no other package under `opm/` declares a compiled-output type.

## Impact

**SemVer:** MAJOR on the alpha line (Principle VI): `opm/core` is deleted and its one type moves to `opm/kernel`, a `RenderResult` field is removed, three `RenderDiagnostics` fields change element type, five `opm/errors` types are renamed or reshaped and two lose methods. Pre-GA, so no migration fragment (ADR-004); both consumers migrate in the same PR wave.

**Downstream migration cost, `cli` (5 files, about 10 lines):**

- `internal/workflow/render/validation.go:58-68`: format `d.Unresolved` rows directly (no `UnresolvedDemandsError` reconstruction); iterate `d.Unmatched` rows by `.Component` and, in verbose mode, print each candidate verdict.
- `internal/cmdutil/output.go:42-78`: `FormatUnresolvedDemands` takes `[]liberrors.UnresolvedDemand`; the `errors.As` branch reads `demandsErr.Demands`.
- `internal/workflow/render/render.go:201`, `types.go:24-27,61`, `output_internal.go:16`, `cmd/instance/diff.go:87`: `Result.Warnings` is filled by a cli formatter over `out.Diagnostics.UnhandledTraits` and the `Newer` rows of `out.Diagnostics.ResolvedVersions`.

The cli imports `opm/core` nowhere: `out.Compiled` is read through `RenderResult` (`internal/workflow/render/render.go:143`), so the `Compiled` move costs it no edit.

**Downstream migration cost, `opm-operator` (4 files, about 10 lines):**

- `pkg/core/compiled_adapter.go:4,15`: the import becomes `opm/kernel` and the parameter type `*kernel.Compiled`; the field copy is unchanged.
- `internal/render/kernel_module_renderer.go:149`: `RenderResult.Warnings` is filled by an operator formatter over the same two diagnostics fields, or the tracker keys on structured rows.
- `internal/reconcile/warnings.go:77-97`: unchanged if the renderer keeps producing strings; otherwise keys on `(component, fqn)` and `(path, versions)`.
- `internal/reconcile/resolution.go:26-40`: unchanged (`*UnresolvedDemandsError` and `*UnmatchedComponentsError` keep their names and pointer receivers); the comment naming `OverSubscribedContractError` reads `*OverSubscribedContractsError`.

**Library tests:** `opm/kernel/render_test.go` and `flow_synth_imported_test.go` (three `core.Compiled` references become `Compiled`), `opm/errors/{match,unmatched,oversubscribed}_test.go` (rewritten for the row and aggregate split), `opm/kernel/render_test.go` (eleven assertions on `Warnings`, `Alternatives`, `Disqualified`, `Unify`, `Matches`), `opm/kernel/{integration,flow_integration,flow_synth_catalog_import}_test.go` (three `Warnings`/`Unify` emptiness checks), `opm/internal/renderstage/stage_test.go` (`TestAlternatives` moves to a glue-level test). The parity oracle (`testdata/parity`, `compareRendered`) compares rendered objects, not the diagnostics shape, and is untouched.
