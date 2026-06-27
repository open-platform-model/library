## Why

OPM's deployable artifact is being renamed from the `Release` vocabulary to `Instance` across the whole stack (enhancement [`0002`](../../../enhancements/0002/), slice **L1**). "Release" is Helm's word and foregrounds a shipping event, when the construct's defining property is *multiplicity* — one `#Module` materialized as many concrete deployments. The `core` slice (C1) already landed: `#ModuleRelease` → `#ModuleInstance`, published as `opmodel.dev/core@v1` `v1.0.0-alpha.1`. The library is the kernel every front-end embeds; until its Go surface, its pinned core, and its wire-coupled strings follow, nothing downstream (`opm-operator`, `cli`) can advance.

This is a **BREAKING** change to the `opm/` public API (Principle VI MAJOR) and ships on the v1 prerelease line (`v1.0.0-alpha.N`, enhancement D13).

## What Changes

Three orthogonal axes land together in one atomic PR — they are inseparable because the library will not compile *or* evaluate against `core@v1` unless all three move at once.

- **BREAKING — Go API rename (`Release` → `Instance`).** `module.Release` → `module.Instance` (type + `ReleaseName`/`ReleaseUUID`/`NewReleaseFromValue` methods → instance forms), `schema.ReleaseMetadata`/`ReleaseView` → `Instance*`, `synth.Release`/`ReleaseInput` → instance forms, kernel `ProcessModuleRelease`/`SynthesizeRelease`/`LoadReleasePackage`/`ValidateReleaseValues*` → instance forms, `core.Resource.Release()` → `Instance()`, `core.Compiled.Release` field → `Instance`. `git mv` the four `release.go` files (+ their `_test.go`) under `opm/module`, `opm/helper/synth`, `opm/helper/loader/file` → `instance.go`.
- **core pin `@v0` → `@v1` (folded in).** `schema.DefaultSchemaModule` `opmodel.dev/core@v0` → `@v1`; every test-fixture `import core "opmodel.dev/core@v0"` (~30 sites) and doc comment moves with it. Mechanically independent of the rename but cannot land separately — the library cannot test against the renamed core without it.
- **Wire-coupled strings (runtime-breaking if missed).** `shape.ExpectedKind "ModuleRelease"` → `"ModuleInstance"`; the CUE transformer-context path `#moduleReleaseMetadata` → `#moduleInstanceMetadata` (`schema/paths.go` `ContextModuleReleaseMetadata`, `schema/context.go` `FillPath` + `ModuleReleaseContextData` type); kind literal in `cmd/flow-inspect`; the `module-release.opmodel.dev/*` label-domain assertions in `synth/release_integration_test.go` (the domain itself is stamped by the core schema, not a Go literal here). These are not cosmetic: rename the Go identifiers but miss these and the library compiles green, then fails at transformer execution against `core@v1`.
- **CUE language version → `v0.17.0-alpha.1`.** Harmonize the `language: version:` field in non-frozen fixtures and the two generators in `opm/internal/registrytest/registrytest.go` (currently `v0.16.0`) and `synth/render.go` (`synthLanguageVersion`). The `cuelang.org/go` toolchain is already pinned at `v0.17.0-alpha.1`. Frozen `enhancements/004` and `006` experiments stay at `v0.16.0` (historical, never edited).
- **Conventions (enhancement D10/D11/D12).** Rename every `release`-named file/dir on disk; carry a `// Was: <OldName>` breadcrumb at every renamed Go identifier and a "Renamed from …" note at every renamed spec/doc section.

Out of scope: any behavioral, evaluation-semantic, or field-shape change; software-release machinery (`CHANGELOG.md`, `release-please-config.json`, `.release-please-manifest.json`, historical `MIGRATIONS.md` prose, `schema/loader_test.go` `TestPublicRegistry_Value`).

## Capabilities

### New Capabilities

- `instance-synthesis`: the renamed form of the existing `release-synthesis` capability — `synth.Instance(...)`/`InstanceInput` and the synthesize-from-typed-inputs flow. The `release-synthesis/` spec dir is `git mv`'d to `instance-synthesis/` (D10).

### Modified Capabilities

- `kernel-runtime`: kernel entry points renamed — `ProcessModuleRelease`/`SynthesizeRelease`/`LoadReleasePackage`/`NewReleaseFromValue`/`ValidateReleaseValues*` → instance forms; per-instance compile-pipeline prose.
- `artifact-types`: the `module.Release` artifact type → `module.Instance` (type, receiver methods, `ReleaseView` → `InstanceView`, metadata projection).
- `helper-packages`: loader `LoadReleasePackage` and the `release.go` file loader → instance forms; `git mv` of the loader file.
- `schema-dispatch`: kind detection `"ModuleRelease"` → `"ModuleInstance"` (`shape.ExpectedKind`) and the `#moduleReleaseMetadata` → `#moduleInstanceMetadata` context-path coupling.
- `config-validation`: typed validation shortcuts `ValidateReleaseValues`/`...Partial`/`...Detailed` → instance forms.

## Impact

- **Affected packages (`opm/`):** `module`, `schema`, `helper/synth`, `helper/loader/file`, `helper/loader/internal/shape`, `kernel`, `core`, `compile`, `errors`; `cmd/flow-inspect`. ~241 production + ~237 test `Release` references across ~34 files; 8 files `git mv`'d.
- **Downstream migration cost (MAJOR):** every consumer that names the old identifiers breaks. `opm-operator` (O1–O3) and `cli` (X1–X4) both pin the published library tag and adapt in lockstep; this slice publishes the tag that unblocks them. `MIGRATIONS.md` gets a new entry recording the rename recipe.
- **Dependencies:** pins `opmodel.dev/core@v1` `v1.0.0-alpha.1`; CUE language level `v0.17.0-alpha.1` (toolchain already there).
- **Risk to verify first:** core@v1 declares `language: version: "v0.17.0"` (final) while the toolchain is `v0.17.0-alpha.1` (semver-older). The first implementation step is a `task cue:vet` of a `core@v1`-importing fixture to confirm it evaluates; if CUE enforces the dep's language floor strictly, that is a signal back to the core slice, not something L1 can resolve alone.
- **SemVer:** MAJOR (breaking `opm/` API), shipped as `v1.0.0-alpha.N` per enhancement D13.

## Scope justification (Principle VIII)

This single change exceeds the usual small-batch size (~478 references, multi-package). The exemption is a deliberate upstream decision in enhancement 0002 / `planned-changes.md` L1: a pure rename **cannot** be split into separately *mergeable* PRs because intermediate states do not compile, and the library is one cohesive Go module with no internal concern boundary to slice on. The work is mechanical and atomic-by-necessity, not unbounded feature scope. The six capability deltas below partition the *spec authoring* without fragmenting the implementation.
