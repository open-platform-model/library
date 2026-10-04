## Context

`Kernel.Render` builds one generated module per render. Its glue, `render.cue.tmpl`, emits two
fields the kernel reads: `diagnostics` (the verdicts) and `rendered` (each matched pair's output).
Two parts of the glue cost more than they need to:

1. **`diagnostics.failedPairs`** (`render.cue.tmpl:550-561`) unifies every matched pair's
   `#transform` with its three inputs a second time, apart from `rendered` (`:513-525`), and keeps
   the pairs whose output is `_|_`. `decodeRenderDiagnostics` forces all of `diagnostics` with
   `dv.Validate(cue.Concrete(true))` (`render_decode.go:65`), so the second evaluation runs on every
   render, refusals included. The kernel copies the list onto `RenderDiagnostics.FailedPairs`
   (`render_decode.go:86`) and uses it in `decodeRendered` only as a fallback beside `out.Err()`
   (`:155-158`, `:170-173`).
2. **`_unify` and `_pred`** (`:176-220`) evaluate the always-unify and predicate rungs for every
   transformer on the platform, per component. Every reader looks them up by a candidate key:
   `matched` (`:233-237`) and the unmatched candidate rows (`:390-393`) iterate `_candidates`; the
   demand outcomes (`:243-263`, `:291-311`) iterate bucket members, which are candidates by
   construction; and `unifyFailures` (`:435-440`) filters on `_candidates`.

Today the kernel returns `FailedPairs` on two `RenderError` paths. On the gate refusal
(`render.go:398-399`), it returns before `decodeRendered` runs, and the list comes from the glue. On
the pair failure (`render.go:401-404`), it comes from the same glue list. The only consumer is the
cli. It prints one `component %q: transformer %s failed` line per entry on any `RenderError`
(`cli/internal/workflow/render/validation.go:186-188`).
`docs/site/diagnostics/transform-failed.md` says that line appears only for a pair whose output was
an error, not for a non-concrete output.

## Goals / Non-Goals

**Goals:**

- Each matched pair's transform is unified once per render.
- `FailedPairs` lists the same pairs in the same order as today, on every path that fills it
  today: the gate refusal and the pair failure. It stays empty on success.
- The two candidate rungs are evaluated only for candidates. Every verdict, and the order of every
  verdict list, stays as it is.
- The memory and CPU effect is measured before and after with the same harness and recorded here.
- The operator memory measurement the walkthrough asked for ("before and after j2, nil-out, shared
  limit and g4") is recorded here too, as memprobe's operator columns against the wave-1 baseline
  (see "Operator memory package").

**Non-Goals:**

- No change to the public `opm/` surface.
- No operator-side code change (shared render semaphore, `GOMEMLIMIT`). Those belong to the
  operator memory package, which is sized against the numbers recorded here.
- No other edit to `decodeRendered` or `render.go`. The changes queued behind this one rebase.

## Decisions

### D1. The kernel names failed pairs from `rendered`

The glue's `failedPairs` field and its comment are deleted. The kernel gets one helper that walks
`diag.Pairs` in order, looks up `rendered."<component> :: <transformer>".output`, and returns the
pairs whose output `Err()` is non-nil:

```go
// failedPairs names the matched pairs whose rendered output is an error, in
// pair order. A missing or incomplete output is not listed: decodeRendered
// refuses those with their own causes.
func failedPairs(rendered cue.Value, pairs []RenderPair) []RenderPair {
	failed := []RenderPair{}
	for _, p := range pairs {
		if pairOutput(rendered, p).Err() != nil {
			failed = append(failed, p)
		}
	}
	return failed
}
```

`pairOutput` is the lookup `decodeRendered` already does at `render_decode.go:163-164`, pulled out
so the two share it. A missing output is not an error value: `LookupPath` returns a non-existent
value whose `Err()` names the missing path, so the helper MUST check `Exists()` before `Err()`. A
missing output is not listed, because today's glue lists only pairs whose output is `_|_`.
`decodeRendered` keeps its own "rendered output missing" cause for it.

`decodeRendered` collects the same list as it walks the pairs, so it needs no second pass. The
pair-failure branch of `Render` fills `diag.FailedPairs` from that list. The gate-refusal branch
calls the helper before it builds the `RenderError`. It reads `rendered` through the same
`LookupPath(pathRendered)` lookup `decodeRendered` uses. If `rendered` is absent there, the helper
leaves `FailedPairs` empty: the gate error already refuses the render, and an absent `rendered` is
a build defect that `decodeRendered` reports on the paths where it runs. `decodeRendered` drops the `failed` map and
the fallback cause `"transformer output is an error"`, which only that map could reach. The
`glueDiagnostics.FailedPairs` field and the `FailedPairs: pairsOf(g.FailedPairs)` line go too.
`decodeRenderDiagnostics` leaves `FailedPairs` as a non-nil empty slice, as `pairsOf` does today.
A successful render therefore reports an empty list without a change on the success path.

**Rationale:** the gate-refusal fill is what keeps the cli's refusal message unchanged. Skipping it
there would save one evaluation per pair on a refused render, but the cli prints `FailedPairs` on
refusals (`validation.go:42-45` runs the formatter for every `RenderError`). A refused render
evaluates each pair's output once after this change. Today it evaluates each one twice.

### D2. `Err()` is the same test as the glue's `== _|_`

The glue listed a pair when its output was bottom; the helper lists it when `Err()` is non-nil.
They agree on the three output shapes the fixtures exercise, and a fourth needs a note:

- **Error at the top of the output.** Both report it.
- **Error nested inside the output.** core declares `#transform: output: {...} | [...{...}]`
  (core `transformer.cue:170`). A conflict anywhere inside a struct output eliminates the struct
  arm, and the list arm never matches a struct, so the whole output is an empty disjunction. That
  is bottom to the glue and an error to `Err()`. Checked during planning: the `failing` scenario's
  conflict sits at `output.metadata.namespace`, and the `TransformError` cause `decodeRendered`
  returns for it is the CUE error
  `rendered."crash :: …/broken-transformer@0.1.0".output: 2 errors in empty disjunction`, not the
  fallback string. So `out.Err()` already catches that pair at base, without the `failed` map. A list
  output behaves the same way through its element type.
- **Incomplete output.** Neither reports it. `TestRender_IncompletePairRefusesNamingPair` already
  asserts an empty `FailedPairs` and a `not concrete` cause from the concreteness check
  (`render_test.go:444-447`). That cause is only reachable when `Err()` is nil, so the test also
  pins the `Err()` side. This is an output whose root is a struct and whose defect is only
  non-concrete fields inside it.
- **Output whose root is incomplete.** The output itself cannot be resolved to a struct or a list,
  for example `[if x {{a: 1}}][0]` with `x: bool`. Probed with cuelang.org/go v0.17.1 against
  `#T: output: {...} | [...{...}]`: `Err()` returns `incomplete bool: bool`, so the helper lists the
  pair. A top-level `== _|_` guard in the same probe also read true, so the two agree there. That
  probe is not the glue: in the same probe `== _|_` also read true for a nested-incomplete struct,
  which the real glue does not list (the `incomplete` scenario). So `== _|_` on an incomplete
  value depends on how the evaluator reaches it, and no fixture produces a root-incomplete output.
  The spec therefore defines the list by `Err()` and names this shape as listed. The
  `TransformError` cause for it is unchanged either way, because `decodeRendered` already checked
  `Err()` first at base. Only the `FailedPairs` entry for this unexercised shape rests on the
  `Err()` definition.

Section 1 pins the nested case in a test before the glue changes. The existing failed-pair test
also asserts that the cause is the pair's own CUE error naming the output path. If a later core
drops the disjunction, that assertion fails, rather than `FailedPairs` silently losing a nested
failure.

### D3. The candidate rungs iterate `_candidates`; `unifyFailures` keeps its order

`_unify` and `_pred` become:

```cue
_unify: {
	for tfqn, _ in _candidates
	let tf = #transformers[tfqn] {
		(tfqn): { /* body unchanged */ }
	}
}
```

The same goes for `_pred`. Every reader looks the rungs up by a candidate key (see Context), so no
lookup can miss. The comments at `:173-175` and `:194-196` say the rungs cover candidates.

`unifyFailures` emits its rows in the order it iterates. Today it iterates `v._unify`, which is in
`#transformers` order. Iterating the candidate-keyed `_unify` would hand the order to however CUE
orders comprehension-built fields, and no test has two unify rows that would catch a reorder. The
row list is a verdict the kernel decodes without re-sorting, so it MUST keep today's order:

```cue
unifyFailures: [
	for cid, v in verdicts
	for tfqn, _ in #transformers
	if v._candidates[tfqn] != _|_
	if !v._unify[tfqn].ok {{component: cid, transformer: tfqn, conflicts: v._unify[tfqn].conflicts}},
]
```

`unifyFailures` sits inside `#Match` (`:120-444`) beside the rungs, so `#transformers` is in scope.
This iterates `#transformers` only for a presence test.
It does no unification, so it costs nothing next to the rung bodies it no longer evaluates. The
candidate filter keeps its meaning: it now picks the rows to emit and fixes their order, where it
used to filter out rung entries for non-candidates.

### D4. Verdicts are checked identical outside the parity harness too

The parity harness (`parity_harness_test.go`, oracle `testdata/parity`) compares pair sets and
rendered output. It does not cover rung verdicts or `FailedPairs`. Those are guarded by
`render_test.go`, `render_skip_test.go`, `render_gate_internal_test.go`,
`render_collision_test.go`, `render_definedby_test.go` and `render_inventory_parity_test.go`. The
whole non-short suite therefore runs with `OPM_FLOW_TEST_FORCE=1`, so that `TestParity_*` and the
flow tests fail rather than skip. As a further check that is not committed, a throwaway test
dumps the decoded `RenderDiagnostics` (every field) and `Compiled` provenance as JSON for every
render scenario and platform pairing the render tests use. It runs at base and again after
sections 2 and 3, and the dumps must be byte-identical. The dumps and the test source live in the
scratchpad, next to the memprobe copy. "Verification" records the dump's scenario list and its
sha256 at base, after section 2 and after section 3, so the claim can be checked after the test is
deleted.

The two evaluation rules in the spec (each pair is unified once, and the rungs cover candidates
only) leave every decoded verdict unchanged, so the dump cannot catch a regression in them. A
committed test on the built value guards them instead (`opm/kernel/render_glue_shape_test.go`,
through `RenderForTest`): `diagnostics` carries no `failedPairs` field, and for a component on a
platform with non-candidate transformers, the keys of the hidden `_unify` and `_pred` fields equal
the keys of `_candidates` (looked up with `cue.Hid` in the render package).

## Research & Decisions

### Where the cost is

**Context**: The owner asked for measurement first.

**Explored**: The memprobe harness (`claude-stuff/kernel-plan-beta1/memprobe`, outside every repo).
It drives the library kernel the way opm-operator does: acquire the generated two-catalog platform
(`opmodel.dev/catalogs/k8s@v1` + `opmodel.dev/catalogs/opm@v4`), acquire the module, synthesize, render, convert to
unstructured, then simulate apply. A scratchpad copy adds a `MODULE` and a `CASES` knob and reports
process user and system CPU from `getrusage`. Its `replace` points at this worktree. The research
spike measured the two template edits separately on cert_manager, under heavier host load and with
an older memprobe build, so its CPU numbers are not comparable with the table below. The
`failedPairs` removal alone took peak heap from about 1.8 GiB to about 0.3 GiB and user CPU from
about 18 s to about 3.6 s.
Keying the rungs on candidates then saved about 0.3-0.4 s more CPU and changed nothing in memory.

**Decision**: Do both, in separate sections, so each lands and measures on its own.

**Rationale**: The first edit is the memory win. The second is a smaller CPU win, but it is the
owner's decision and costs one comprehension rewrite.

## Measurements

These numbers come from a library-side simulation, not from a running operator. memprobe mirrors
the operator's render path in one process. It does not exercise the operator's `GOMEMLIMIT`
(`config/manager/manager.yaml`) or its controller slots. The `r2` cases model the worst case of two
controllers rendering at once with no shared limit. The pins are memprobe's: the platform carries
k8s 1.0.0-beta.1 and opm 4.4.4, and core resolves to v2.0.0-beta.2. The cert_manager fixture is a
snapshot of `modules/cert_manager`. k8up and web_app were measured before from the `modules/`
checkout at commit `09cff2b`, where both directories last changed in `d4d2e09`. Copies of both were
taken into the scratchpad memprobe's `fixtures/` before any code edit, and the after run uses
those copies, so a later move of the `modules/` checkout cannot shift the comparison.

Each cell is the median of 5 runs, one process per run, with `GOGC` and `GOMEMLIMIT` unset. Heap
and RSS are in MiB, CPU in seconds. CPU and heap are quoted rather than wall time, because the host
is shared and its load average varied during the runs.

- `render_peak_heap`: the largest sampled heap during the render(s).
- `render_peak_live`: the largest live heap any GC marked during the render(s).
- `vmhwm`: process peak RSS up to the end of the simulated apply.
- `user_s`: whole-process user CPU, startup and platform acquisition included (about 2 s on
  web_app, which is a floor).

Cases: `r1-nil` is one render with the result dropped after conversion (the operator's nil-out,
merged). `r2-hold` is two concurrent renders with the results held through apply. `r2-nil` is two
concurrent renders with the results dropped.

### Before (base `58f8151`, the library code of `93a892f`)

Run 2026-10-04. The raw logs and summaries are in the scratchpad copy's `results/before-*`. The
header line reads `+dirty` for k8up and web_app because this change's untracked OpenSpec files sat
in the tree. No code was edited.

| Module (objects) | Case | `render_peak_heap` | `render_peak_live` | `vmhwm` | `user_s` |
| --- | --- | --- | --- | --- | --- |
| cert_manager (42) | r1-nil | 1839 | 1192 | 1993 | 9.84 |
| cert_manager (42) | r2-hold | 3437 | 2957 | 3766 | 46.72 |
| cert_manager (42) | r2-nil | 3454 | 2992 | 3771 | 24.06 |
| k8up (20) | r1-nil | 1039 | 828 | 1188 | 4.97 |
| k8up (20) | r2-hold | 2018 | 1177 | 2286 | 19.38 |
| k8up (20) | r2-nil | 2015 | 1214 | 2285 | 13.32 |
| web_app (2) | r1-nil | 180 | 144 | 221 | 1.87 |
| web_app (2) | r2-hold | 257 | 206 | 301 | 3.41 |
| web_app (2) | r2-nil | 255 | 210 | 300 | 2.51 |

Two cert-manager renders at once peak at about 3.7 GiB RSS, close to the operator's 4 Gi limit.
`user_s` in the `r2` cases spreads widely between runs: cert_manager `r2-hold` ran 44-48 s and
`r2-nil` ran 24-32 s, in sequence on a shared host. Read the `r2` CPU as approximate. The heap and
RSS columns varied by less than 5%.

### After

Filled in by tasks.md 4.1, at the change's head, with the same harness, cases and knobs.

### Operator memory package

The walkthrough's operator measurement ("before and after j2, nil-out, shared limit and g4"), as
memprobe's operator columns on cert_manager, all seven cases, 5 runs each, medians in MiB. memprobe
models the nil-out (`*-nil` cases against `*-hold`) and the shared limit (`r1` against `r2`: one
render slot against two) itself, so those two are read across rows rather than across columns.

- **Wave-1 baseline**: `memprobe/results/baseline-20261003T140923`, library `507379a` (before j2;
  the probe then pinned core v2.0.0-beta.1).
- **Before g4**: this change's base, j2 merged (core's pins moved out of the schema module, which is
  what `core_retained` measures). Run 2026-10-04 at `eb1e1fa` (the planning commit, no code edit),
  `results/before-full-cert_manager-20261004T231756`.
- **After g4**: this change's head (tasks.md 4.1).

| Metric | Case | Wave-1 baseline (`507379a`) | Before g4 | After g4 |
| --- | --- | --- | --- | --- |
| `core_retained` | core | 71.8 | 2.5 | |
| `idle_retained` | r1-nil | 165.2 | 95.9 | |
| `apply_retained_over_idle` | r1-hold | 1570.6 | 1570.6 | |
| `apply_retained_over_idle` | r1-nil | 1.6 | 1.6 | |
| `apply_retained_over_idle` | r2-hold | 3141.1 | 3141.1 | |
| `render_peak_heap` | r1-hold | 1896.1 | 1834.9 | |
| `render_peak_heap` | r1-nil | 1898.8 | 1820.4 | |
| `render_peak_heap` | r2-hold | 3708.4 | 3471.1 | |
| `render_peak_heap` | r2-nil | 3711.3 | 3418.1 | |
| `vmhwm` | core | 105.0 | 25.0 | |
| `vmhwm` | r1-hold | 2054.8 | 1992.0 | |
| `vmhwm` | r1-nil | 2055.4 | 1974.3 | |
| `vmhwm` | r2-hold | 3999.3 | 3790.3 | |
| `vmhwm` | r2-nil | 3995.8 | 3767.8 | |
| `stagger_vmhwm` | stag-hold | 3750.2 | 3657.6 | |
| `stagger_vmhwm` | stag-nil | 2021.8 | 1951.2 | |

j2 cut the idle footprint (`core_retained` 71.8 to 2.5, `idle_retained` 165.2 to 95.9) and left the
render peak almost where it was. The nil-out already drops what a held result retains through apply
(`apply_retained_over_idle` 1570.6 held against 1.6 dropped). The render peak itself is what g4
targets. This run's `user_s` is not quoted: the host's load average was above 50 on 16 CPUs while it
ran, and its `r1-nil` `user_s` read 18.2 s against 9.8 s in the planning run of the same code. The
heap and RSS columns agree with the planning run within 2%.

## Risks / Trade-offs

- [A future core drops the output disjunction, so a nested bottom stops making the output bottom]
  → The glue's `== _|_` would have missed it as well. The failed-pair test asserts the CUE cause
  (D2), so the shift fails a test rather than going unnoticed.
- [A refused render still evaluates every pair's output once] → It evaluated them twice before. The
  cli needs the list on refusals (D1).
- [`unifyFailures` row order] → D3 keeps `#transformers` order explicitly. The diagnostics dump in
  D4 compares the decoded rows byte for byte.
- [memprobe is a simulation, and its pins lag the current platform (opm 4.6.0)] → Before and after
  use the same pins, so the comparison holds. The absolute numbers are labelled as simulation
  output. If the owner wants a reading from a real operator, it is one kind-cluster RSS measurement
  of the operator before and after its library bump. That belongs to the operator memory package.
