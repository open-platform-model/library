## Context

Slice 06 (`add-phase-methods-and-rename-compile`) introduced `MatchInput`, `PlanInput`, and `CompileInput` to expose phase-explicit methods on `*kernel.Kernel`. Each input struct currently carries a `Module *module.Module` field alongside `ModuleRelease *module.Release`. The original rationale was that schema lookups for `#config` validation and binding resolution went through the standalone `Module`. Slice 02 (`unify-artifact-shape`) and slice 09's spec work on `artifact-types` then pinned the invariant that `Release.Package` always carries an embedded `#module` reachable through the binding's `Paths().Module`. Once that invariant exists, the separate `Module` field on the three downstream input structs is redundant: every read currently performed against `in.Module` is reachable through `in.ModuleRelease.Package`.

Today's call graph reveals the redundancy:

- `Match` does not read `in.Module` at all.
- `Plan` and `Compile` only forward `in.Module` to the embedded `Validate` call so it can reach the `#config` schema and the APIVersion-keyed binding.
- `Validate` itself is the lone genuine reader, and its `ValidateInput.Module` is left untouched in this slice.

Carrying the redundant field invites two real failure modes:

1. **Drift.** A caller can construct a `*module.Release` from one `#module` and pass an unrelated `*module.Module` in the same input struct. The kernel mixes them silently — `Validate` reads schema from the supplied `Module`, while `Compile` reads components and module metadata from the release.
2. **Fixture cheating.** `pkg/kernel/phase_test.go` builds release packages without an embedded `#module` and passes `mod` separately. Tests stay green only because the kernel currently reaches for `in.Module` instead of the release's embedded reference. Any future caller that constructs a release the production way (via `module.ParseModuleRelease`) carries an embedded module and never needs the parallel `Module`.

This slice removes the redundancy and centralizes the `#config` schema lookup behind a small `(*Release).ConfigSchema()` accessor on `pkg/module`. `ValidateInput` is intentionally untouched — it owns the post-slice-04 contract, and tightening that surface here would conflate two design moves.

## Goals / Non-Goals

**Goals:**

- Remove `Module *module.Module` from `kernel.MatchInput`, `kernel.PlanInput`, and `kernel.CompileInput`.
- Add `(*Release).ConfigSchema() cue.Value` accessor on `pkg/module/release.go`, returning the embedded module's `#config` schema via the binding's `Paths().Module` + `Paths().Config`.
- Migrate `kernel.Compile`'s internal call to `Validate` so the embedded `Module` it currently forwards is sourced from `in.ModuleRelease.Package` rather than from a now-absent input field.
- Update `pkg/kernel/phase_test.go` and `pkg/kernel/kernel_test.go` fixtures to embed `#module` (with `apiVersion`, `metadata`, `#config`) into the release literal, removing the parallel `mod` artifact previously passed alongside.

**Non-Goals:**

- Modifying `ValidateInput`. Its `Module` field is the one place schema lookup happens and removing it conflates with slice 04's signature change. A follow-up slice can retire `ValidateInput` once both 04 and this slice have landed.
- Modifying `validate.Config`'s signature. Slice 04 owns that move.
- Introducing a kernel-side `ValidateConfigDetailed` or any Tier-1 validation method. Slice 05's `pkg/helper/values` is the home for source-positioned diagnostics; the kernel keeps only its Tier-2 safety net.
- Retiring `module.NewModuleFromValue`. The standalone `*module.Module` artifact survives for load-time / inspection workflows; this slice only narrows three kernel input structs.
- Adding deprecation aliases. Input structs are direct value-types; carrying a deprecated parallel `Module` field for one slice doubles the surface for no realized migration value (no in-repo external consumer today).

## Decisions

**`(*Release).ConfigSchema()` lives on `*module.Release`, not on `*kernel.Kernel`.** Reason: path knowledge (`Paths().Module` + `Paths().Config`) is a property of the release artifact and the binding registry, both already imported by `pkg/module`. Hosting the accessor on the kernel would force `pkg/kernel` to re-implement what the binding already exposes and would split the artifact's own surface across two packages. The accessor is a one-line follow-on to the existing `MatchComponents()` accessor pattern in `pkg/module/release.go`.

**Zero-value return on lookup failure, not error.** Reason: `MatchComponents()` (the existing peer accessor) returns the zero `cue.Value` when the binding is unregistered or the path is missing. The internal Tier-2 caller already short-circuits on `!schema.Exists()` returning nil. An error return would force every call site to handle a case that callers cannot reasonably recover from at runtime — the binding is registered or the artifact is malformed, both load-time concerns.

**`Compile` synthesizes the `Module` it forwards into `Validate`.** Reason: `kernel.Validate` accepts a `ValidateInput{Module, ModuleRelease, Values}` and reads `Module.Package` for `#config`. With `CompileInput.Module` removed, `Compile` must either (a) build a transient `*module.Module` from `in.ModuleRelease.Package.LookupPath(b.Paths().Module)` and pass it forward, or (b) bypass `kernel.Validate` and call `validate.Config` directly with the schema obtained via `Release.ConfigSchema()`. Option (b) is cleaner but couples this slice to slice 04's signature timing. Option (a) keeps the slice independent: a tiny private helper `moduleFromRelease(rel)` in `pkg/kernel/phases.go` constructs the transient view by re-using `module.NewModuleFromValue` against the embedded `#module` value. When slices 04 and 05 land, a follow-up can collapse the indirection.

**Test fixtures embed `#module` literally rather than going through `module.ParseModuleRelease`.** Reason: kernel tests are unit-scoped. Threading them through the loader/parser brings the parser's invariants under test alongside the kernel's, slowing diagnostics when one breaks. Embedding `#module` as a CUE literal in `relPkg` keeps the test laser-focused on the kernel surface and forces the production-shape invariant ("releases carry an embedded `#module`") to be visible in fixtures.

**No deprecation alias for the removed `Module` field.** Reason: input structs are constructed inline at call sites. Today no in-repo consumer constructs `MatchInput`, `PlanInput`, or `CompileInput` outside of `pkg/kernel/` itself. Keeping a deprecated parallel field for one slice is dead weight. Downstream consumers (CLI, operator, Crossplane fn) do not embed the kernel yet; their migration is a single line removal once they do.

## Risks / Trade-offs

**Risk — `Compile`'s synthesized `Module` introduces a transient allocation per call.** → Mitigation: the cost is one `module.NewModuleFromValue` per `Compile`, which is a `cue.Value` lookup plus a small struct allocation. Negligible against transformer evaluation. When slice 04 lands, the synthesis can collapse into a direct schema lookup via `Release.ConfigSchema()`.

**Risk — fixtures that embed `#module` drift from production reality if `module.ParseModuleRelease`'s output shape changes.** → Mitigation: the binding's `Paths().Module` is the contract, not the literal layout. A fixture that targets that path matches whatever `ParseModuleRelease` produces. If `ParseModuleRelease` ever stops embedding `#module`, both fixtures and `Release.MatchComponents()` / `Release.ConfigSchema()` would fail together — a single coherent failure, not silent divergence.

**Risk — `ValidateInput.Module` becomes the lone outlier carrying the field.** → Mitigation: this is intentional and documented in the proposal. Slice 04 reshapes `validate.Config`'s signature, after which `ValidateInput` itself becomes a candidate for retirement. Surfacing this as a follow-up keeps each slice's blast radius small.

**Trade-off — breaking change for a not-yet-realized consumer base.** → No `cli` or `opm-operator` embeds the kernel today, so the breaking surface affects no live code. Bumping the kernel module version is the SemVer-correct move and is cheap right now. Postponing the slim until consumers exist would force a synchronized migration later.

## Migration Plan

The breaking surface is contained: three input struct literals. Migration steps for any future consumer:

1. Remove the `Module:` field from `kernel.MatchInput{...}`, `kernel.PlanInput{...}`, and `kernel.CompileInput{...}` literal expressions.
2. If the consumer previously held a `*module.Module` purely to populate those fields, drop the load step entirely; only `*module.Release` is needed for those phases.
3. If the consumer needs the `#config` schema (e.g. for a UI rendering values forms), call `release.ConfigSchema()` directly.

No rollback path — the field is removed cleanly.

## Open Questions

None at the slice level. Remaining surface decisions (whether to retire `ValidateInput` post slice 04, whether to remove `module.NewModuleFromValue` once helper-side loading covers all paths) are explicit follow-up territory and deliberately out of scope here.
