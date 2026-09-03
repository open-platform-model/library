# Design: library-render-build

## Context

See `proposal.md` § Why. The architecture was decided and measured in `enhancements/0019` (D7 to D10, D13, D17, D18; experiments 01 to 08); this document maps it onto this repo. Prerequisites: `library-platform-source` (Source on both inputs) and a published core prerelease carrying D5/D17 (`core-registry-import`: `#CatalogEntry`, derived `#composedTransformers`, no `#matchers`).

The render module's target shape (0019 `02-design.md` § The render build):

```
<render tmp dir>/
  cue.mod/module.cue        promoted dependency list (D13)
  cue.mod/local-module.cue  replaceWith → instance dir, platform dir
  render.cue                imports both packages + embedded glue
  [instance tree]           materialized here when Source is overlay-mode
```

## Goals / Non-Goals

**Goals**

- `Kernel.Render` end to end: stage, promote, skew-check, build once in a per-render context, decode diagnostics and rendered output.
- The glue reproduces the Go matcher's exact pair set (D10's gate) and preserves D28's fail-closed semantics.

**Non-Goals**

- No cutover: `Materialize`/`Match`/`Compile` untouched; deletion, re-pins, ADR-002's successor, `doc.go` rewrite, `synth.Platform`'s fate, and the old-vs-new parity proof are `library-render-cutover`.
- No memoization or pooling of render builds (D8: shares-nothing; consumers size pools by memory).
- No single-provider guard in the glue: 0019 has not decided its home (glue diagnostic vs 0011 publish gate). The old path's `providerGuard` keeps running untouched; the cutover change MUST NOT land until 0019 records that decision.

## Research & Decisions

### One entry point named Render, beside Compile

**Context**: The new path needs a public entry while the old one still exists.
**Explored**: rewiring `Compile` behind a feature flag (rejected: two behaviors under one name); naming it `Compile2`-style (rejected).
**Decision**: `Kernel.Render(ctx, RenderInput) (*RenderResult, error)`, with `RenderInput{Instance, Platform, RuntimeName, Skew}` and `RenderResult{Compiled []*core.Compiled, Diagnostics RenderDiagnostics, Warnings []string}`. `RenderDiagnostics` carries the decoded verdict sets (reusing `oerrors.MissingFQN`, `oerrors.UnresolvedDemand`, `oerrors.UnifyError`) plus the per-path resolved-versions rows (D18's data surface). Whether `Render` replaces `Compile` or `Compile` is rewired onto it is the cutover change's decision.
**Rationale**: "render" is 0019's vocabulary throughout (one build per render, render module); the old names stay stable until deleted.

### Staging writes a real temp directory, including overlay-mode instances

**Context**: `local-module.cue` directory replacements are served from disk (`modpkgload/replace.go`, cited in 0019 `02-design.md`); a synthesized instance's Source is an in-memory overlay.
**Explored**: serving replacements through `load.Config.Overlay` (unproven against the replace path; a spike that stops paying once temp staging is accepted); requiring frontends to persist instances themselves (pushes kernel mechanics into every frontend, against the workspace rule that structure lives in the library).
**Decision**: `Render` materializes everything into one per-render temp directory: the generated `cue.mod` pair, `render.cue`, and the instance tree when its `Source.Overlay` is non-nil. An on-disk source (platform always, per the operator writing its generated platform module to disk; dir-acquired instances too) is referenced in place via its absolute path in `replaceWith`. The directory is removed on return (`defer`), success or failure.
**Rationale**: writing a handful of small files per render is noise against the measured fixed ~85 ms catalog term (experiment 07); determinism holds because content, not path, feeds the build. If overlay-through-replacement is later proven, dropping the copy is an internal optimization with no contract change.

### Promotion via mod/modfile, whole-list, platform-wins

**Context**: D13 fixes the rule; the mechanics need a parser.
**Explored**: computing the list from the module graph (rejected by D13: that is maximum-version selection, the resolver this exists to bypass); render-time tidy (rejected by D13: tidy writes resolutions, render inherits one).
**Decision**: parse both inputs' `cue.mod/module.cue` with `cuelang.org/go/mod/modfile`; emit the platform's dependency entries whole (preserving default-major markers: the default-major trap is a measured failure, 0019 `02-design.md`), union in instance-only paths, platform wins shared paths; write the render `module.cue` with a fresh module path (a reserved, never-published render module identity) and the same `language.version` floor as the inputs' maximum. The refusal invariant then re-parses the written file and asserts every OPM-namespace path (`opmodel.dev/...`) present in either input list appears in it.
**Rationale**: promotion is string-level modfile mechanics over two committed files the inputs already carry (`Source` gives both roots); no evaluation, no I/O. The invariant is cheap (one parse) and is the D13 "authority fails by omission" tripwire.

### Skew compares the two committed lists, never the promoted one

**Context**: D18 fixes source, default and the older-is-data rule.
**Decision**: per OPM-namespace path present in both lists, compare SemVer; instance newer → `SkewPolicy` decides (`SkewWarn` zero-value default, `SkewRefuse`); every compared path emits a resolved-versions row into `RenderDiagnostics` unconditionally. Implemented in `renderstage` beside promotion (same parsed inputs).
**Rationale**: reading the promoted list would compare the platform against itself (D18's stated refusal); doing it beside promotion reuses the parses.

### The glue is an embedded template; matching is experiment 05's shape

**Context**: D10 ratified the glue: buckets from `#composedTransformers` (required ∪ optional), always-unify as plain `&`, predicate rung, verdicts as data behind `== _|_` guards, the D28 gate as one unification, and two measured boundaries (unstated posture refuses as a build error naming `core/trait.cue`'s `optional`; incomplete pair output is invisible to the guards and caught by kernel-side concreteness).
**Explored**: keeping matching in Go against the built value (D10's recorded fallback). Not viable even transitionally here: the Go matcher reads `#matchers`, which D17 removed; adapting it to fold buckets in Go is throwaway work against a glue that already exists and reproduced the kernel's exact pair set (18/18 verdict rows, experiment 05).
**Decision**: `render.cue` is a `go:embed` template: a static glue body (`#Match` comprehensions, context fill of `#runtimeName`, `rendered` assembly with provenance keys, `diagnostics` struct) plus generated import lines for the instance package (its module path + `Source.Pkg`) and the platform package. The kernel fills nothing into the build except what the template text carries; `#runtimeName` enters as a rendered literal (quoted via CUE formatting, mirroring `synth`'s no-string-interpolation rule for caller values).
**Rationale**: the glue's inputs are exactly the two imports; keeping it template-static makes the generated file inspectable and the parity comparison (cutover) byte-stable. D12's context projection shipped in the D5 core prerelease (2.0.0-alpha.7, `#transform` projects `#context` from its other two inputs), so the glue supplies only `#context.#runtimeName` and derives no metadata blocks itself.

### Fixtures pin the D5 core prerelease; the in-process registry serves the catalog

**Context**: the render fixture needs a platform module that imports a catalog, resolvable in tests without GHCR writes.
**Decision**: new `testdata/render/` fixtures: a small catalog module (registrytest-published, as the materialize tests already do), a platform module importing it with `#CatalogEntry` entries, an instance module demanding its contracts; every fixture `cue.mod` pins `opmodel.dev/core` to the exact D5 prerelease. Tests thread the in-process registry env through `Render`'s load config, the same pattern as `opm/internal/registrytest` consumers.
**Rationale**: no test-only backdoors; the production resolver/loader path runs unchanged (the standing materialize-test rule in `CLAUDE.md`).

## Public surface changes (`opm/`)

Additive only: `Kernel.Render`, `RenderInput`, `RenderResult`, `RenderDiagnostics`, `SkewPolicy` (+ two values), a skew-row type. Machinery in `opm/internal/renderstage` (not public surface). A skew diagnostic error type is added to `opm/errors` only if no existing type fits refuse-mode reporting.

## Risks / Trade-offs

- [Floating schema hazard: `schema.OCILoader` defaults to `opmodel.dev/core@v2` = v2.latest, so core's D5 release flips every cold cache to the new shape and breaks `synth.Platform` (it writes subscription `version` fields D5 derives) before any re-pin] → this change's fixtures pin explicitly and are immune; the wave constraint is recorded here and in the core change: core's D5 release and `library-render-cutover` land as one coordinated train, or library CI pins the schema fixture to the prior alpha in a prep commit first.
- [Glue error quality: refuse-mode diagnostics lose the verbatim CUE cause `UnifyError` carries today without a second diagnostic evaluation] → accepted by D10 with the Go-matcher fallback recorded; the decoded conflicting-FQN data is the contract, and richer causes can be added behind the same types later.
- [Sequencing: implementation cannot start until `core-registry-import` publishes a prerelease] → tasks are ordered so the modfile/promotion/skew half (core-independent) can land first within the change; only the fixtures and the build-path tests gate on the release.
- [Temp-dir staging on hot render paths] → measured noise next to the fixed catalog term; revisit only with a profile in hand.
- [Unstated-posture refusal surfaces as a raw incomplete-value build error, not a decoded diagnostic] → measured boundary accepted by D10; the publish-side stated-posture gate belongs to the 0011 family.

## Migration Plan

Additive PR train inside one change: (1) `renderstage` staging + promotion + refusal invariant + skew with unit tests (no core dependency); (2) glue template + `Render` + decode, with the D5-pinned fixtures, once the core prerelease exists. Nothing external migrates; rollback is a revert of unconsumed surface.

## Open Questions

- Whether the render module's generated module path should be a fixed reserved literal or derived per render (content-hash suffix). Safe to answer at implementation: the path never resolves anywhere (build-local by construction) and does not affect the specs.
