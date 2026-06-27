## Context

This is slice **L1** of cross-cutting enhancement [`0002`](../../../enhancements/0002/) — renaming OPM's deployable-artifact family from `Release` to `Instance` vocabulary. The `core` slice (C1) already landed: `#ModuleRelease` → `#ModuleInstance`, published as `opmodel.dev/core@v1` `v1.0.0-alpha.1`. The library is the second hard gate in the wave (`C1 → L1 → {opm-operator ‖ cli}`); both `opm-operator` and `cli` pin the published library tag, so nothing downstream advances until this lands and is tagged.

The library is the OPM kernel — a single cohesive Go module embedded by every front-end. Its `opm/` surface is contractual (Principle VI). This change is a hard, breaking rename of that surface plus the wire strings that couple it to `core@v1`. No behavior changes.

## Goals / Non-Goals

**Goals:**

- Rename the Go `Release` public surface to `Instance` across `opm/` (types, methods, functions, fields) with a `// Was: <OldName>` breadcrumb at every renamed identifier (enhancement D11/D12).
- Pin `opmodel.dev/core@v1` `v1.0.0-alpha.1` and migrate every `@v0` reference (Go const + ~30 fixture imports + doc comments) to `@v1`.
- Update the wire-coupled strings that MUST match `core@v1`: kind literal `"ModuleRelease"` → `"ModuleInstance"`, transformer-context path `#moduleReleaseMetadata` → `#moduleInstanceMetadata`, label-domain assertions.
- Bump the CUE `language: version:` field to `v0.17.0-alpha.1` in non-frozen fixtures and the two generators.
- `git mv` every `release`-named file/dir to its instance equivalent (enhancement D10).
- Publish the `v1.0.0-alpha.N` library tag that unblocks O*/X*.

**Non-Goals:**

- Any behavioral, evaluation-semantic, or field-shape change. This is a rename only.
- Touching software-release machinery (`CHANGELOG.md`, `release-please-*`, historical `MIGRATIONS.md` prose, `TestPublicRegistry_Value`).
- The frozen `enhancements/004` and `006` experiments (kept at `language v0.16.0`, never edited).
- Splitting into separately-mergeable PRs (impossible — see Decisions).

## Decisions

### D-L1.1 — One atomic PR despite exceeding small-batch size

A pure rename cannot be split into separately *mergeable* PRs: any intermediate state where some identifiers are renamed and others are not does not compile, and the wire strings must flip in lockstep with `core@v1`. The library is one Go module with no internal concern boundary to slice on. **Alternative considered:** a compat-alias/deprecation window (type aliases `Release = Instance`). **Rejected** by enhancement D8 — hard rename, no aliases, because the whole stack moves together on the prerelease line and aliases would leak the retired vocabulary into the `v1` surface. The six capability deltas partition *spec authoring*, not the implementation.

### D-L1.2 — Three axes fold into one change

Axis 1 (Go API rename), Axis 2 (`core@v0`→`@v1` pin), Axis 3 (wire strings) are conceptually independent but mechanically inseparable: the library cannot compile *or* evaluate against `core@v1` unless all three move at once. Folding Axis 2 in is a deliberate scope decision (not creep) — called out explicitly so review doesn't mistake the ~30 fixture-import edits for unrelated churn.

### D-L1.3 — Spec deltas use REMOVE+ADD for renamed items, MODIFIED for body-only

OpenSpec's `MODIFIED` binds a delta to an existing requirement by header text, so a header rename cannot be expressed as `MODIFIED`. This repo's established idiom (the archived `api-version-dispatch` → `schema-dispatch` rename) is **REMOVE the old + ADD the new**. Applied here:

- `release-synthesis` capability → `instance-synthesis`: old spec `REMOVED` (all requirements), new spec `ADDED` (full renamed content). Directory `git mv`'d.
- Surviving capabilities (`kernel-runtime`, `artifact-types`, `helper-packages`, `schema-dispatch`, `config-validation`): a requirement whose **header** names a renamed symbol → REMOVE+ADD; a requirement whose header is clean but body references a renamed symbol → `MODIFIED` (header verbatim). **Alternative:** a `RENAMED Requirements` op. **Rejected** — no precedent in this repo and ambiguous archive-time binding.

### D-L1.4 — CUE language version `v0.17.0-alpha.1` (literal)

The `language: version:` field in non-frozen library fixtures and the two generators (`registrytest.go`, `synth/render.go`) standardizes on `v0.17.0-alpha.1` — matching the already-pinned `cuelang.org/go v0.17.0-alpha.1` toolchain and keeping the library on the prerelease line consistent with D13. (Decided with the user during exploration.)

## Risks / Trade-offs

- **[core@v1 language floor vs alpha toolchain]** → `core@v1` declares `language: version: "v0.17.0"` (final), but the toolchain is `v0.17.0-alpha.1`, which is semver-*older* than final. If CUE enforces a dependency's language floor against the running toolchain, every `core@v1`-importing fixture fails to evaluate. **Mitigation:** the FIRST implementation task is a `task cue:vet` of a minimal `core@v1`-importing fixture, before any rename work. If it fails, that is a signal back to the core slice (re-cut `core@v1` with an alpha language level, or move the library toolchain to final `v0.17.0`) — L1 cannot resolve it unilaterally and must stop there.

- **[Axis 3 silently skipped]** → renaming Go identifiers but missing `shape.ExpectedKind` or the `#moduleReleaseMetadata` FillPath leaves the library compiling green but failing at transformer execution against `core@v1`. **Mitigation:** the flow/integration tests (`task cue:test:flow`, `release_integration_test.go`) exercise the wire path end-to-end; they must be green before tag. A grep gate for residual `ModuleRelease`/`moduleRelease`/`module-release` literals (excluding the software-release allowlist) runs before merge.

- **[Downstream breakage cost — MAJOR]** → every consumer naming the old identifiers breaks. **Mitigation:** this is intended and sequenced — O*/X* pin the new tag and adapt in lockstep; `MIGRATIONS.md` records the rename recipe.

- **[Verbatim spec fidelity]** → REMOVE+ADD reproduction risks paraphrasing the original requirement text. **Mitigation:** `openspec validate` after authoring; deltas copy source verbatim with only token substitution.

## Migration Plan

1. **Gate check (blocking):** confirm a `core@v1`-importing fixture evaluates under the current toolchain (the language-floor risk above). Stop if it fails.
2. Pin `core@v1` (Axis 2): `DefaultSchemaModule`, fixture imports, doc comments; bump `language: version:` to `v0.17.0-alpha.1`.
3. Rename Go surface (Axis 1) package-by-package; `git mv` the four `release.go`/`_test.go` sets; add `// Was:` breadcrumbs.
4. Flip wire strings (Axis 3); update label-domain test assertions.
5. `task check` + `task cue:test:flow` green; residual-literal grep gate clean.
6. Append `MIGRATIONS.md` entry; bulk-archive the six spec deltas (`openspec-bulk-archive-change`).
7. Publish the `v1.0.0-alpha.N` library tag; record the slice as a `history` event in enhancement `0002/config.yaml`.

**Rollback:** revert the branch; the tag is a prerelease, not yet pinned by any published downstream until O*/X* advance.

## Deviations from Design

Recorded during implementation (all resolved; `task check` green):

- **Gate passed.** The core@v1 language-floor risk (Risks §1) did **not** materialize — a `core@v1` fixture declaring `language: version: "v0.17.0-alpha.1"` evaluates cleanly under the `v0.17.0-alpha.1` toolchain. `#ModuleInstance` exists, `#ModuleRelease` is gone (hard rename, no alias).
- **`renderInstanceFile` core-import bug.** The synth package fabricated `import core "@v0"` from a split string literal (`corePath + "@v0"`) the path-based sed missed; fixed by threading the resolved `coreVersion` and deriving `@major` (consistent with the dep pin). Caught by the wire/flow tests, as Risks §2 predicted.
- **Negative control retired.** `TestRelease_ImportedModule_NegativeControlV040` (`opm/helper/synth`) was removed, not renamed. It pinned `core@v0.4.0` to prove the positive import test was non-vacuous, but synth now emits `core.#ModuleInstance` (undefined in v0.4.0), so it can no longer exercise the self-cycle admission path. Supporting a pre-rename core is out of scope. Documented in `MIGRATIONS.md`. *(If a v1-era negative control is wanted later, make `registrytest` major-aware and assert the new failure mode.)*
- **One test intentionally keeps old vocabulary.** `opm/materialize/composed_open_test.go` still uses `#moduleReleaseMetadata` because it pins the real pre-rename catalog `opmodel.dev/catalogs/opm@v0.5.2`; its context must match that transformer's contract, not core@v1's.
- **Catalog scope gap (downstream).** The published `opmodel.dev/catalogs/opm` catalog (NOT in enhancement `0002` `affects`) still uses `module-release.opmodel.dev` / `#moduleReleaseMetadata` vocabulary. Not an L1 blocker (the flow tests pass against catalog v0.6.0), but the catalog will need its own Release→Instance pass before it is re-cut against core@v1. Flagged for enhancement 0002 scope.
- **Internal-naming sweep.** Beyond the literal `Release` surface, the abbreviations `rel`→`inst` (locals/params/test-fixture fields, excluding `registry/module.go` where `rel` = relative path), `mrm`→`mim`, plural `releases`→`instances` in comments, and stale filename references were updated for D12 consistency.

## Open Questions

- Exact `v1.0.0-alpha.N` number for the library tag — assigned by release-please at publish time, not fixed here.
