# Design: cue-owned-verdicts

## Context

See `proposal.md` § Why. The verdict path on `main` today, one render:

```
  render.cue.tmpl                      render_decode.go                 opm/errors
  +---------------------------+        +----------------------------+   +------------------------+
  | #Match per component      |        | glueDiagnostics (mirror)   |   |                        |
  |   _candidates, _unify,    |        |                            |   |                        |
  |   _pred, matched          |        |                            |   |                        |
  | diagnostics:              | decode |                            |   |                        |
  |   unresolved[].disq: [tf] | -----> | byCandidate JOIN unify --> | > | UnresolvedDemand       |
  |   unifyFailures[]         | -----> | explode 1 UnifyError/FQN   |   |   .Disqualified        |
  |   candidates[] (flat)     | -----> | GROUP -> matchMatrix ----> | > | UnmatchedComponents-   |
  |   bucketKeys              | -----> | Alternatives() + compat -> | > |   Error.Matches (map)  |
  |   warnings[]              | -----> | group + English text ----> | > | RenderResult.Warnings  |
  |   missing, resolved       | -----> | decoded, never read        |   |                        |
  | gate: resolved & ...      | (never | gateErrors(): REGROUP ---> | > | errors.Join(...)       |
  +---------------------------+  read) +----------------------------+   +------------------------+
```

Constraints: matching semantics are pinned by the parity oracle (`render-parity`, matched pair sets agree) and must not move; `Render`'s stage and build steps are untouched; the kernel reads `diagnostics` through `LookupPath` so it stays decodable beside a failing `gate`; the glue cannot export a CUE error value (0019 D10 measured boundary), so every conflict is reported as the FQN it occurred at; `one-api-tier` edits `opm/errors` (sentinels) and lands first so the package history reads as one story.

## Goals / Non-Goals

**Goals:**

- Every verdict the kernel exports is a row the glue emitted, decoded once, with no join, group or sort that changes its content.
- `renderstage` imports nothing outside `opm/internal` and `opm/module`.
- `opm/errors` has one shape rule: rows are data, gate causes are pointer-receiver aggregates over rows.
- A successful or partially-refused render exposes the same candidate evidence as a refusal.
- `Render`'s output type is declared beside the verb that produces it; `opm/` carries no single-type package.

**Non-Goals:**

- Changing which demands resolve, which components match, or what the single-provider guard computes.
- Reporting the verbatim CUE conflict (not exportable from inside the build).
- Moving `opm/compat` (slice 5) or touching `IdentityError` (loader path).
- Deciding how the cli or the operator word a warning.

## Research & Decisions

### The apiVersion ladder sort runs in the glue

**Context**: `UnresolvedDemand.Alternatives` must be sorted on the `alpha < beta < GA`, then major, then minor ladder (0010 D34/D4) so the diagnostic is deterministic. Today the sort is `compat.CompareAPIVersions` inside `renderstage`.
**Explored**: (A) emit `alternatives` unsorted from CUE and sort in Go with a local comparator, deleting the `compat` import but adding a second ladder implementation in Go beside `compat`'s; (B) keep the import until slice 5, which then has to leave a copy of the comparator behind anyway; (C) sort in CUE with `list.Sort` and a comparator over `regexp.FindSubmatch`. A spike against cue v0.17.1 (`scratchpad/ladder.cue`) confirms C orders `v1alpha2 < v1beta1 < v1 < v2 < v10`, identical to `compat.CompareAPIVersions`, in about 15 lines of CUE.
**Decision**: C. The glue carries `#apiVersionLess` (level, major, minor, then the key string for a total order) and `unresolved[].alternatives` is `list.Sort` over the same-base keys of the demand's own bucket universe. Go copies the list.
**Rationale**: the only ladder in the render path is the one the build applies; Go holds no comparator, so slice 5 can move `compat` without leaving a stub. The cost is a regexp per alternative on refusal paths only.

### Disqualification rows carry their conflicts from the glue

**Context**: `_resOutcome[fqn].disqualified` and `_traitOutcome[fqn].disqualified` are comprehensions over the bucket that already test `!_unify[tfqn].ok`; `_unify[tfqn].conflicts` is in scope.
**Decision**: the comprehension yields `{transformer: tfqn, conflicts: _unify[tfqn].conflicts}`. `unifyFailures` keeps emitting every disqualified candidate (including those on demands that resolved through another candidate) with the same row shape, so `Diagnostics.Unify` and `UnresolvedDemand.Disqualified` decode from the same CUE row type.
**Rationale**: one row per candidate, not one per conflicting FQN (today's `UnifyError` explosion), matches how the glue thinks and how a frontend reports ("transformer X refused on FQNs a, b").

### Unmatched rows carry their candidates

**Context**: the glue knows an unmatched component (`len(v.matched) == 0`) and its candidate verdicts (`v._candidates`, `v._pred[tfqn].missingLabels`) in the same scope. Go reconstructs the join twice.
**Explored**: keeping `candidates` flat on the diagnostics as well (a superset: matched candidates too). No consumer reads a matched candidate's verdict; the pair list already names them.
**Decision**: `diagnostics.unmatched: [{component, candidates: [{transformer, matched, missingLabels}]}]`, sorted by component and transformer in the glue (`list.Sort` on the key strings). `unmatchedComponents`, `candidates`, `bucketKeys`, `missing` and `resolved` leave the export; `match.resolved` stays internal for `gate`.
**Rationale**: one shape for the diagnostics and the gate cause; the match matrix becomes visible on a partial dry run.

### Rows are data, gate causes are aggregates

**Context**: `opm/errors` mixes value-receiver rows that are also errors (`UnifyError`, `UnresolvedDemand`, `OverSubscribedContractError`) with pointer-receiver aggregates and wrappers. Consumers route on the aggregates only (`opm-operator/internal/reconcile/resolution.go`) and format from the rows (`cli/internal/cmdutil/output.go`).
**Explored**: making every type a pointer receiver (rows inside slices then stop being errors unless addressed); keeping `Error()` on rows and `Unwrap() []error` on aggregates (the current shape, which is what forced the fabricated `TransformError` unwrap to "look like the others").
**Decision**:

```go
// rows: data, no Error method
type UnresolvedDemand struct{ Component, FQN, Kind string; Alternatives []string; Disqualified []UnifyRefusal }
type UnifyRefusal struct{ Component, Transformer string; Conflicts []string }
type UnmatchedComponent struct{ Component string; Candidates []CandidateVerdict }
type CandidateVerdict struct{ Transformer string; Matched bool; MissingLabels []string }
type OverSubscribedContract struct{ Key string; Catalogs []string }

// gate causes: pointer receivers, Error() lists the rows, no Unwrap
type UnresolvedDemandsError struct{ Demands []UnresolvedDemand }
type UnmatchedComponentsError struct{ Components []UnmatchedComponent }
type OverSubscribedContractsError struct{ Contracts []OverSubscribedContract }

// wrappers with a real cause keep Unwrap
type TransformError struct{ Component, Transformer string; Cause error }
type SkewError struct{ Path, ModuleVersion, PlatformVersion string }
```

`RenderDiagnostics` holds `Unresolved []UnresolvedDemand`, `Unify []UnifyRefusal`, `Unmatched []UnmatchedComponent`, `OverSubscribed []OverSubscribedContract`; the three aggregates are built from those slices without copying or reordering.
**Rationale**: `errors.AsType[*X]` is uniform for every cause; a row cannot be mistaken for a refusal; the aggregate's `Error()` is the one English rendering the library keeps, which is Go convention for errors and not a presentation surface (the cli already replaces it).

### `RenderResult.Warnings` is deleted, not typed

**Context**: the data behind both warning kinds is on the diagnostics (`ResolvedVersions[].Newer`, `UnhandledTraits`). The owner chose frontend-owned wording over typed rows with a `String()`.
**Decision**: remove the field and both formatters. `SkewWarn`'s doc says "renders against the platform's build and marks the row `Newer`"; `RenderDiagnostics`'s doc names the two advisory sources so a new frontend does not have to read the kernel to find them.
**Rationale**: Principle IV; the cli and the operator want different wording and different dedup keys; a typed row with `String()` is the same smell one step removed.

### The CUE `gate` stays and its agreement is pinned

**Context**: `gate: match.resolved & (len(guard.overSubscribed) == 0) & true` makes the built value erroneous on refusal; `renderstage.Build` never checks it and `gateErrors` recomputes the predicate from the decoded rows.
**Explored**: dropping `gate` (one gate, Go authoritative; a staged render module then evaluates green under `cue eval` while the kernel refuses it); keeping both with nothing tying them.
**Decision**: keep `gate` as the module's own fail-closed statement (0010 D28 "as one unification", and the reason a staging directory is self-refusing when debugged by hand); `gateErrors` stays authoritative because it produces the typed causes; every refusal test in `render_test.go` asserts `built.LookupPath("gate").Err() != nil` and every success test asserts it is `true`, through one helper.
**Rationale**: one line of CUE buys a self-describing artifact; the helper turns "two gates that must agree" into a pinned invariant instead of a hope.

### Unread glue fields leave the export

**Decision**: `missing` and `resolved` (deferred here by `cut-dead-surface`) and `bucketKeys` (consumed only by the deleted `Alternatives`) are removed from `diagnostics`; `glueDiagnostics` mirrors the export exactly so a stray field is a compile-time question, not a silent decode. `missing` was the only reader of the per-component `missingResources` / `missingTraits` lists, so those go with it.

### The decoder asserts concreteness, because `resolved` was carrying it

**Context**: an unhandled trait whose `optional` posture the catalog never stated must refuse as a build error naming that field (0010 D28/D46). On `main` that works by accident: `diagnostics.resolved` is a scalar `bool` derived from `len(unresolved)`, so an incomplete guard anywhere upstream makes it incomplete and `dv.Decode` fails, which is what `unstatedPosture` keys off. Removing `resolved` from the export removes that mechanism — and CUE decodes a comprehension whose `if` did not evaluate as an EMPTY LIST, not as an error, so the render would silently succeed with no verdicts. Measured: the unstated fixture renders green with `Unresolved: []`, while `gate` correctly errors.
**Explored**: (A) keep `resolved` on the export purely to hold the invariant (a field with no reader, which this change exists to remove); (B) decode `traitPostures` into `glueDiagnostics` so its incomplete bool fails the decode (a second field with no reader, and it only covers the posture case); (C) assert `dv.Validate(cue.Concrete(true))` before decoding and raise `unstatedPosture` from there.
**Decision**: C.
**Rationale**: the rule is "every verdict the kernel reads is concrete", which is what was being relied on all along; stating it covers any future incomplete verdict, not just the posture one, and it stops a fail-closed guarantee from depending on a field kept alive for a side effect. The new `gate`-agreement helper is the independent check that the two refusals still coincide.

### Ordering that left Go had to arrive in the build

**Context**: the decoder used to sort three things Go no longer touches: the same-base alternatives (the apiVersion ladder), and — less visibly — the single-provider guard's rows and each row's `catalogs`, which arrived from CUE struct iteration and were sorted in `decodeRenderDiagnostics`.
**Decision**: the guard sorts in the glue too: `overSubscribed` is `list.Sort` on the key, and each row's `catalogs` is `list.Sort(..., list.Ascending)`.
**Rationale**: "decoded without derivation" is one rule, not a rule about alternatives. A row that arrives unsorted and is sorted in Go is the same defect as an alternative computed in Go, and the refusal's determinism (pinned by the over-subscription test) would otherwise depend on CUE's field order.

### `Compiled` lives beside `Render`

**Context**: `opm/core` holds one type, `Compiled`, documented as "shared domain primitives" (CONSTITUTION III). No second primitive arrived in the twenty-six alphas since; `Render` is its only producer, and the two consumers wrap it on arrival (`opm-operator/pkg/core/compiled_adapter.go`, `cli/internal/workflow/render/render.go:143`).
**Explored**: keeping the package as the landing place for future shared primitives (none is planned; Principle VII); a type alias `core.Compiled = kernel.Compiled` for one release (pre-GA, consumers re-pin in the same wave, so the alias is a second name with no reader).
**Decision**: `kernel.Compiled` with the same four fields and doc, declared in `render.go` beside `RenderResult`; `opm/core` is deleted; CONSTITUTION III's package list drops it.
**Rationale**: the type is the render verb's output and nothing else; declaring it where it is produced removes one import from every consumer and one package from the SemVer surface. It lands first, as one mechanical commit, because every later task in this change rewrites the two files that name it.

## Risks / Trade-offs

- [A comprehension that yields an empty list where Go used to join] → the existing refusal fixtures assert `Disqualified` and `Alternatives` contents (`render_test.go` missing-FQN and disqualified-candidate tests); a dropped row fails a test, not a user.
- [`list.Sort` with a custom comparator is slower than the Go sort] → runs only on refusal paths over a handful of keys; not measurable against the build.
- [Consumers lose the kernel's warning wording] → the two formatters are about ten lines each and are listed in the proposal's migration cost; the cli's copy of the skew sentence keeps its current wording if it wants to.
- [`UnresolvedDemandsError.Unwrap` removal] → no consumer routes on `UnresolvedDemand` by type (grep over both consumers); the cli's generic funnel keys on `*UnresolvedDemandsError`, which stays.

## Migration Plan

1. Land task 1 (the `Compiled` move) as one mechanical `refactor(kernel)!:` commit, then tasks 2 and 3 (glue and decoder) as one commit group: the tree is green with the old verdict shapes still in place, because the decoder adapts internally first.
2. Land task 4 (`opm/errors` reshape) and task 5 (`Warnings` removal) as the breaking group; `refactor(kernel)!:` and `refactor(errors)!:`.
3. Consumers migrate in their own PRs against the next alpha, as the sibling changes `migrate-kernel-api-and-verdicts` in `cli` and in `opm-operator`. Each carries this change's edits together with `one-api-tier`'s, because both library changes ship in the same alpha and neither consumer compiles between them. This change edits no other repo; its task 7.2 only records the breakage against a local `replace`. Rollback is a re-pin to the previous alpha; no persisted state changes shape.
