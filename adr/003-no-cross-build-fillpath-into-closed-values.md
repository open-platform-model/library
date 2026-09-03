# ADR-003: Construct artifacts without cross-build FillPath into closed values

## Status

Superseded by ADR-006 (artifacts are constructed in one CUE build), 2026-09-03.

This ADR replaced the deleted `003-single-build-cue-evaluation-invariant.md`, which framed single-build as the universal invariant. It reframed the invariant as the no-cross-build-fill principle with two tactics, single-build and federation, and rejected the single-build-only framing on one premise: that a platform must compose several versions of one catalog `path@major`, which one build cannot hold.

That premise is gone. Enhancement 0010 D14 made each subscription name exactly one build, and 0019 D5 made the platform a CUE module that imports its catalogs, so minimum version selection admits one version per `path@major` per build by construction. The federation application described under Decision (`indexCatalogs`, `MaterializedPlatform.Transformers` / `.Matchers`, "read the materialized index off `Transformers` / `Matchers`") was deleted with `opm/materialize` by `library-render-cutover`. The framing this ADR rejected is now the rule: ADR-006 restates the principle as "every artifact the kernel constructs is one CUE build that imports its inputs" and records why. The `composed_open_test.go` guard named under Open went with the surface it guarded; the closedness regression canary is `opm/internal/cueregression/closedness_test.go`. ADR-005 records the lifetime and concurrency rules of that build.

The text below is preserved as the record of the decision as written; the materialize path it describes no longer exists.

## Context

OPM builds composite CUE artifacts by combining independently-built `cue.Value`s — a release's components filled into a platform's transformers, a module imported into a synthesized release, a catalog's `#transformers` indexed onto a platform. The natural Go-API tactic is `FillPath` / `Unify` across builds.

Filling a value into a **closed**, independently-built value corrupts lazy in-expression resolution of output-local hidden fields: a transformer's `#transform` reading a hidden field declared *inside* `output` (e.g. `_convertedSidecars`) evaluates to `non-concrete value _`. This was traced to a CUE Go-API closedness bug — not a CUE-language or schema defect (`docs/design/transformer-output-hidden-field-scope-bug.md` §11–§12). `cue vet` cannot catch a regression here; the corruption surfaces only at marshal time, and only on the closed-fill path.

Two artifact-construction paths hit this:

- **Render / synth** (`synth.Release`): builds a `#ModuleRelease` that imports a `#Module`.
- **Materialize** (`Materialize`): indexes the selected catalogs' `#transformers` into a composed transformer map plus a `#matchers` reverse index for a `#Platform`.

A version constraint distinguishes them. A platform may subscribe to multiple versions of the same catalog `path@major` at once (Module-A authored against `catalog@0.5.0`, Module-B against `@0.5.1`, both live on one platform). CUE Minimal Version Selection admits exactly one version per `path@major` per build.

## Decision

**Never `FillPath` a value into a closed, independently-built value to construct an artifact.** Satisfy the invariant with whichever tactic the path's version constraints allow:

- **Single-build** where one build suffices (render / synth): construct the artifact by evaluating one synthesized CUE package that *imports* its inputs, so there is no cross-build fill. Applied by `simplify-render-single-build` (`synth.Release` builds the release via single-build evaluation of an importing package; the `userModule`-scope workaround and the Go `#config` pre-merge were deleted).

- **Federation** where single-build is impossible (materialize): MVS forbids holding multiple same-major catalog versions in one build, so the composed-transformer map and the `#matchers` index are built natively in the owner `*cue.Context` (`indexCatalogs`) and exposed as first-class `MaterializedPlatform.Transformers` / `Matchers` fields — and are **not** filled onto the closed `c.#Platform`. Distinct version-bearing FQNs are distinct keys in one merged native map, so multi-version-per-major composition is preserved. Applied by `federate-materialize-transformers`.

Rejected — framing this ADR as a *single-build-only* invariant. Single-build cannot hold OPM's required multi-version-per-major catalog composition, so it is the correct tactic for synth but the wrong universal rule. The invariant is the no-cross-build-fill principle; single-build and federation are its two applications.

Rejected (materialize path) — keeping the closed twin alive and routing reads to a separate open `Composed` map guarded by a "never read `#transform` off `Package`" comment (the interim fix, bug doc §13). Correctness then depends on every future reader honoring a comment `cue vet` cannot enforce — a live tripwire. Federation removes the twin structurally: there is no surface from which a corrupt read is possible.

## Consequences

**Positive:** The closedness corruption is eliminated by construction, not worked around — no surface exists from which reading a `#transform` is corrupt. Both construction paths converge on one stated principle. Multi-version-per-major composition is preserved (federation), and the synth / render path sheds its `userModule`-scope and Go `#config` pre-merge workarounds (single-build).

**Negative / Trade-off:** The two paths use different mechanisms, so "how OPM constructs artifacts" is one principle with two implementations rather than one code path. Federation keeps the composed surfaces as read-only `cue.Value` fields on `MaterializedPlatform` rather than on the platform value, so downstream consumers must read the materialized index off `Transformers` / `Matchers`, not off `Package` (a MAJOR Go-API break, recorded in `MIGRATIONS.md`).

**Open:** The underlying CUE Go-API closedness bug is unfixed upstream. If CUE later fixes it, the federation and single-build tactics become belt-and-braces rather than load-bearing — the regression guards (e.g. `composed_open_test.go`, which asserts the closed surface still corrupts) would start passing against the closed value and should be revisited at that point.
