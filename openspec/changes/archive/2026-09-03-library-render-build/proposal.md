## Why

Enhancement 0019 Phase B collapses the render to one CUE build per render (D9): the kernel stages the instance and the platform into a generated render module, evaluates it once in a context that dies with the render (D8), and reads `rendered` and `diagnostics` off the built value. Nothing crosses a build boundary, so parity with pure CUE stops being a property the harness polices and becomes one the kernel cannot violate. This change builds that path; it does not switch anything over to it.

This change is `library-render-build`, the second library slice of 0019 Phase B. It consumes `library-platform-source` (the `Source` fields on instance and platform) and the released D5/D17 core schema (`core-registry-import`: registry entries carry the catalog by import, `#matchers` gone). It implements D9 (the single build), D13 (the promoted `cue.mod` and the refusal invariant), D7/D18 (skew detection and policy), D10 (matching in the build, verdicts as data), and the per-render context lifetime rule of D8.

## What Changes

Everything is additive; the existing `Materialize`/`Match`/`Compile` pipeline is untouched and stays the public default until `library-render-cutover`.

- A new kernel entry point, `Kernel.Render`, taking a source-carrying instance, a source-carrying platform, a runtime name, and a skew policy. It refuses inputs without `Source` (the render build imports packages, not values).
- Render-module staging (internal): a per-render temp directory holding a generated `cue.mod/module.cue` (dependency list derived by promotion, D13), `cue.mod/local-module.cue` (directory replacements for the instance and platform trees; an overlay-mode instance source is materialized into the temp directory first), and a generated `render.cue` importing both packages plus the embedded glue.
- Promotion (D13): the platform module's tidied dependency list whole, the instance module's unioned in for paths only it carries, the platform winning every shared path. No tidy-equivalent runs at render time. After writing the file the kernel verifies no OPM-namespace path from either input would resolve from the module graph, and refuses the render otherwise (a kernel defect, never caller-configurable).
- Skew (D7/D18): the comparison reads the staged instance module's `cue.mod` against the platform module's tidied list per OPM-namespace path. Module-newer-than-platform triggers the configured response: warn-and-render (default) or refuse. Older-than-platform and the resolved-versions comparison ride the diagnostics as plain data, no severity.
- Matching in the build (D10): the glue embeds experiment 05's measured shape: buckets built from the platform's derived `#composedTransformers`, the always-unify rung as plain `&` (the D30 provenance carve-out is not ported: one build, one set of catalog bytes), the predicate rung, verdicts as data fields, and the fail-closed gate as one unification. The kernel decodes verdicts into the existing `oerrors` types (`MissingFQN`, `UnresolvedDemand`, `UnifyError`).
- Execution: one `BuildInstance` in a fresh `cue.Context` created for the render and dropped with it (D8). `rendered` decodes to `[]*core.Compiled` with instance/component/transformer provenance; per-pair concreteness validation stays kernel-owned (an incomplete pair output is invisible to the glue's error guards, per experiment 05).

Out of scope, deliberately: rewiring `Kernel.Compile`/`Kernel.Match`, deleting `opm/materialize` and the Go matcher/executor, `synth.Platform`'s fate, the ADR-002 successor text, doc.go's concurrency rewrite, and the parity proof that gates the deletion. All of that is `library-render-cutover`.

## SemVer classification

MINOR: new kernel surface (`Render`, its input/result types, the skew policy type), no existing signature or behavior changes. The heavy lifting is internal packages.

## Affected packages and downstream consumers

- New: `opm/internal/renderstage` (staging, promotion, skew, glue generation), kernel-level `Render` entry plus input/result/diagnostics types; new render fixtures under `testdata/` (a platform module importing a catalog fixture, pinned to the D5 core prerelease).
- Touched: `opm/kernel` (new render files; `acquire.go`'s directory acquisitions now stamp `Source.Root` as the enclosing module root and `Pkg` as the package directory relative to it, so a subpackage imports correctly from the render build: MODIFIED deltas on `artifact-types` and `platform-artifact`), `opm/errors` (a skew diagnostic type if none fits; reuse first), `opm/internal/registrytest` (a helper serving a committed fixture registry tree).
- **`cli` / `opm-operator`**: no action; nothing calls `Render` until the cutover wave. The operator's platform-package generation (0019 D6) and the CLI's authored `./opm` platform become `Render`'s real inputs then.
- **Sequencing hazard (coordinated wave, stated in design.md):** the kernel's default schema loader floats on `opmodel.dev/core@v2`, so the moment core publishes the D5 alpha, `synth.Platform` (which writes subscription `version` fields the D5 schema derives instead) breaks on any cold cache, before any re-pin. Core's D5 release and the library's cutover wave must land as one train; this change's own fixtures pin the D5 prerelease explicitly and are immune.

## Complexity justification

The single build replaces measured costs, not hypothetical ones: the shared-platform model races under concurrent render (2321 detector reports), retains 348 MB per render, and serialised loses 2.5x to 5.5x to a shares-nothing worker at every module size (0019 experiments 06 to 08). Eight concluded experiments stand behind the shape; the one recurring cost, a fixed ~85 ms catalog term per render, is accepted in 0019's decision log.

## Capabilities

### New Capabilities

- `single-build-render`: the render as one CUE build: input requirements, the promoted render `cue.mod` and its refusal invariant, skew detection and policy, in-build matching verdicts as data, execution and decode, and the per-render context lifetime.

### Modified Capabilities

None. `platform-materialization`, `platform-matching` and `transform-input-fill` describe the old path, which this change does not touch; their rewrite happens at cutover.

## Impact

- Code: new `opm/internal/renderstage/*`, new `opm/kernel/render.go` (+ types + tests), embedded `render.cue` glue template, `testdata/` render fixtures (D5-pinned platform + catalog served via `opm/internal/registrytest`).
- No changes to `opm/materialize`, `opm/compile`, `opm/helper`, existing fixtures, or any published pin used by the old path.
