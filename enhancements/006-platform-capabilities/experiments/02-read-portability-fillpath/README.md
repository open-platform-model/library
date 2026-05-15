# 02-read-portability-fillpath

## Hypothesis

Given the matcher works in isolation (validated by [01-matcher-mechanics](../01-matcher-mechanics/README.md)), the wider end-to-end claims hold:

1. **D7 — read surface**: a component body referencing `#consumes.required["…/route@v1"].spec.domain` inside a string interpolation resolves to a concrete value after the resolved `#consumes` is unified into `#module`.
2. **Portability**: a `#ModuleRelease` with no `#platform` and no `#consumes` writeback fails `cue vet -c` with an actionable diagnostic pointing at the interpolation site whose dependency is non-concrete.
3. **D13 — `#`-prefix exclusion**: `cue export` on a bound release does not emit `#platform`, `#module`, or `#consumes` fields. `#Platform` values do not leak into rendered output.
4. **D13 — `FillPath` kernel writeback**: a Go program loading the release CUE and invoking `cue.Value.FillPath` against `#-`prefixed paths (`#platform`, and `#module/#consumes/required/<fqn>`) produces the same component-body value as the inline-bound CUE fixture.

## Setup

Schemas under `schemas/` are copies of the equivalent files from [01-matcher-mechanics/schemas/](../01-matcher-mechanics/schemas/), plus one new file: `module_release.cue` — see "Finding F1" below for why the shape diverges from [006/03-schema.md §`#ModuleRelease integration`](../../03-schema.md#modulerelease-integration).

Fixtures under `cases/jellyfin/`:

- `module.cue` — `JellyfinModule` (consumes `route@v1`; component body `JELLYFIN_AppHost.value` interpolates `#consumes.required[fqn].spec.domain`).
- `platform-prod.cue` — `ProdPlatform` (provides `route@v1` with `domain: "apps.example.com"`).
- `release-bound.cue` — `ReleaseBound` (simulates the kernel's writeback by inline-unifying provider spec into `#module.#consumes` and supplying `#platform: ProdPlatform`).
- `release-unbound.cue` — `ReleaseUnbound` (no `#platform`, no `#consumes` writeback; the artifact end-users would ship before the runtime binds it).

Go harness under `cmd/fillpath/`:

- `main.go` — loads the unbound release, walks the module's declared `#consumes.required` keys, FillPaths each platform-provided value into `#module.#consumes.required[fqn]`, FillPaths `#platform`, then reads the resolved component-body value back out. Exits non-zero on mismatch.

## Run

```bash
cd enhancements/006-platform-capabilities/experiments/02-read-portability-fillpath
./run.sh
```

`run.sh` executes the four claim checks and prints the result of each.

## Outcome (2026-05-15)

All four claims pass with their expected results. Captured output (abbreviated):

```text
=== Claim 1 — read surface concretizes (bound release) ===
app: {
    env: {
        AppHost: {
            name:  "JELLYFIN_AppHost"
            value: "jellyfin.apps.example.com"
        }
    }
}
[exit=0]

=== Claim 2 — unbound release diagnostic ===
ReleaseUnbound._withConfig.#components.app.env.AppHost.value: invalid interpolation:
  non-concrete value =~"^[a-z0-9.-]+\\.[a-z]+$" (type string)
[exit=1]

=== Claim 3 — #-prefix exclusion (cue export) ===
apiVersion: opmodel.dev/v1alpha2
kind: ModuleRelease
metadata: { name: jellyfin-prod, namespace: media }
values: {}
components:
  app:
    env:
      AppHost:
        name: JELLYFIN_AppHost
        value: jellyfin.apps.example.com
[result] OK — no definition fields in export

=== Claim 4 — Go harness FillPath kernel writeback ===
[kernel] opmodel.dev/exp/caps/routing/route@v1: filling matched provider value
components.app.env.AppHost.value = "jellyfin.apps.example.com"
OK — FillPath kernel-style writeback resolved the read-surface interpolation.
[exit=0]
```

## Findings

### F1 — `#ContextBuilder` cannot be invoked inside `#ModuleRelease` (revises D5)

The headline finding. The design specified in [006/03-schema.md §`#ModuleRelease integration`](../../03-schema.md#modulerelease-integration) has the release invoke `#ContextBuilder` inline and unify `out.consumes` back into `#module.#consumes`:

```cue
let _builderOut = (#ContextBuilder & {
    #platform: #platform
    #consumes: _withConfig.#consumes
}).out
let unifiedModule = _withConfig & {
    #consumes: _builderOut.consumes
}
```

This shape creates a self-referential evaluation cycle. `_builderOut` is a function of `_withConfig.#consumes`; `unifiedModule.#consumes` is set to `_builderOut.consumes`; reads against `unifiedModule.#consumes` (through component-body interpolation) re-enter the cycle. CUE 0.16.1 (both the default evaluator and `CUE_EXPERIMENT=evalv3=1`) does **not** solve this fixed point — it freezes `#consumes.required[fqn].spec.domain` at the type-level constraint (`string` with the route regex), even though the provider has supplied a concrete value at the entry point.

Probe sequence (run during this experiment; transcripts available on request, summarized here):

- Direct top-level unification (`mod & {#consumes: resolved}` at package scope) — **works**: interpolation resolves.
- Same chain through a `#Wrap`/`#Caller` that takes `#module: #Module` as input and unifies inside the body — **works** when the unified value (`resolved`) is supplied directly as a parameter, not derived.
- The full design (CB call inside release with `_builderOut = CB(#module.#consumes)` and `#module.#consumes = _builderOut.consumes`) — **fails**: the matcher's output is consumed by the same field that feeds the matcher's input. Result: `builderOut.consumes.required[fqn].spec.domain = string` (the matcher never sees a concrete provider in the comprehension's `if` guard because the fixed-point hasn't converged).

The mechanism that *does* work is the one this experiment ships:

1. The release does **no** in-line CB call. `_withConfig` exists for 004 D34 reasons (config-first ordering) but the `#consumes` writeback is **not** performed inside the release.
2. The kernel/CLI/operator performs the match externally (a top-level `#ContextBuilder` invocation against `#module.#consumes` and the chosen `#Platform.#provides` — a one-shot call, no cycle since the inputs are top-level concrete values).
3. The kernel `FillPath`s every matched entry into `#module.#consumes.required[fqn]` AND fills `#platform` onto the release.
4. The release evaluates with the writeback already done.

This is what `cmd/fillpath/main.go` demonstrates end-to-end. Direct in-CUE unification (the `release-bound.cue` fixture, which inlines the same writeback statically) produces the identical result.

**Impact on the design (D5 and D7):**

- **D5 — "Matching lives in `#ContextBuilder`, CUE-side"** — still valid for the *matching itself* (the comprehension and conditional-struct logic work — verified in 01). What changes: the CB call **must be a top-level invocation orchestrated by the kernel**, not an inline step inside `#ModuleRelease`. The "no Go pipeline change" claim in D5 needs revision — there *is* a small Go-side orchestration (load, match, FillPath, evaluate) that the runtime performs.
- **D7 — "`#consumes` is both declaration and resolved read surface"** — fully valid as a *read pattern* (claim 1 confirmed). The mechanism that puts the resolved value into `#consumes` is FillPath from outside, not in-CUE unify-back from inside.
- The `#TransformerContext.#runtimeName!` precedent cited in D13 is the right precedent — but the analogous pattern for `#platform` and `#consumes` is *runtime-fills-before-evaluation*, not *in-CUE-derives-from-self*.

### F2 — D13 `#`-prefix exclusion works as specified

`cue export` on the bound release emits only `apiVersion`, `kind`, `metadata`, `values`, and `components`. No `#platform`, no `#module`, no `#consumes` field leaks. The `#`-prefix exclusion is reliable for production export.

### F3 — `FillPath` against `#`-prefixed paths works

`cue.MakePath(cue.Def("#platform"))` and `cue.MakePath(cue.Def("#module"), cue.Def("#consumes"), cue.Str("required"), cue.Str(fqn))` both compose paths the Go `cue.Value.FillPath` accepts and that produce the expected concrete result post-fill. D13's mechanism is real and codable in a few lines (see `cmd/fillpath/main.go`).

### F4 — Unbound-release diagnostic is actionable but suboptimal

`cue vet -c` on `ReleaseUnbound` correctly fails with `invalid interpolation: non-concrete value` pointing at `module.cue:28:11` (the interpolation site) and the route regex constraint. It is **actionable** — a developer can read the diagnostic and figure out that the route capability needs a provider. It is **suboptimal** compared to what D6 hoped for, which was a diagnostic naming `#consumes.required[fqn].spec` directly. With the kernel-driven writeback model, a better release-time diagnostic would come from the kernel checking unfilled `#consumes.required` keys *before* CUE evaluation and emitting a clean "platform X does not satisfy module Y's required capability Z" message.

## Notes for 006 docs

- **04-decisions.md D5** — add a "Revised by exp 02 F1" note: matching algorithm is CUE-side; CB invocation is kernel-orchestrated at top level, not inline in `#ModuleRelease`.
- **03-schema.md §`#ModuleRelease integration`** — the 3-step inline-CB-call-then-unify-back pattern needs revision. Suggest: the release schema carries no CB invocation; the kernel performs match + FillPath; the release evaluates with `#consumes` already resolved.
- **02-design.md §"Matching — `#ContextBuilder` gains a step"** — the "`#ModuleRelease` invokes the builder inline (one call)" sentence is the load-bearing claim that F1 refutes. Suggest rewriting to describe the kernel-orchestrated flow.
- **04-decisions.md D6** — the "incomplete-`spec` diagnostic at the FQN path" works in isolation (01's 02-required-missing case). End-to-end (this experiment's unbound release), the diagnostic surfaces at the component-body interpolation site instead, because the lack of writeback means the `#consumes` entry stays at the *declared* type-level constraint and the interpolation hits it. Kernel-side pre-check ("module X requires capability Y; platform Z does not provide it") would give a more direct release-time error.
- **04-decisions.md D13** — confirmed. Add: the kernel's responsibility expands modestly from "fill `#platform`" to "match + FillPath matched `#consumes` entries + fill `#platform`". Still a small surface; F3 shows the FillPath plumbing is a few lines.

## Status

Concluded — 2026-05-15. Findings F1 (revises D5/D7's inline-CB claim) and F4 (suboptimal unbound diagnostic) are material to the 006 design and should be folded back into 04-decisions.md before implementation begins.
