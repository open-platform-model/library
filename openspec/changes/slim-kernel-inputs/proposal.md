## Why

After slice 06 (`add-phase-methods-and-rename-compile`) introduced `MatchInput`, `PlanInput`, and `CompileInput`, those structs each carry a `Module *module.Module` field that is structurally redundant. `*module.Release` already embeds the source module via `Release.Package` at the binding's `Paths().Module`, so every read currently performed against the separate `Module` field — `#config` schema lookup, APIVersion detection, module metadata — is reachable through the release. Carrying both invites caller mistakes (passing a Module that disagrees with the release's embedded one) and forces every test fixture to construct two artifacts where one suffices.

This slice narrows the kernel input contract to a single artifact handle (`*module.Release`) for the three downstream phases (Match, Plan, Compile). `ValidateInput` is untouched in this slice — its `Module` field is the one place schema lookup happens and removing it conflates with slice 04's signature change. A small accessor on `*module.Release` centralizes the "where does `#config` live" knowledge inside `pkg/module`, so internal call sites stop re-deriving the path.

## What Changes

- **BREAKING**: Remove `Module *module.Module` field from `kernel.MatchInput`.
- **BREAKING**: Remove `Module *module.Module` field from `kernel.PlanInput`.
- **BREAKING**: Remove `Module *module.Module` field from `kernel.CompileInput`.
- Add `(r *Release) ConfigSchema() cue.Value` accessor on `*module.Release` returning the embedded module's `#config` schema via the binding's `Paths().Module` + `Paths().Config`. Returns the zero `cue.Value` when no binding is registered or the path does not exist.
- Migrate `kernel.Compile` internals so the embedded `Validate(ValidateInput{...})` call sources its `Module` from `in.ModuleRelease.Package` (via a small private helper) rather than from a now-absent `in.Module`. This preserves the current Tier-2 validation behavior without exposing the old field.
- Update `pkg/kernel/phase_test.go` fixtures to embed an `#module` block (carrying `apiVersion`, `metadata`, `#config`) into the release `relPkg` literal, removing the standalone `mod` artifact previously passed alongside the release.
- This is a MAJOR change for `pkg/kernel/`. Bump kernel module version. No deprecation alias is offered: the input structs are direct value-types and a deprecated parallel field would double the surface for one slice.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `kernel-runtime`: Tightens the input contract for `MatchInput`, `PlanInput`, and `CompileInput`. Each accepts `*module.Release` as its sole artifact handle for those phases; the redundant `*module.Module` field is removed. `ValidateInput` is unchanged in this slice.
- `artifact-types`: Adds the `Release.ConfigSchema()` accessor as an ergonomic surface on top of the existing binding-path requirement so call sites reach `#config` without re-walking the path.

## Impact

- **`pkg/kernel/inputs.go`** — three input structs lose their `Module` field. Doc comments updated.
- **`pkg/kernel/phases.go`** — `Match`, `Plan`, `Compile` stop reading `in.Module`. `Compile`'s internal call to `Validate` synthesizes a transient `*module.Module` from `in.ModuleRelease.Package` (or threads through a refactored helper that takes schema + release directly).
- **`pkg/module/release.go`** — adds `ConfigSchema()` accessor.
- **`pkg/kernel/phase_test.go` and `pkg/kernel/kernel_test.go`** — fixtures updated to embed `#module` in `relPkg`. Tests that previously asserted on `mod` directly read from `rel.Package.LookupPath(...)` or use the new accessor.
- **Downstream consumers** — none in this repo today (CLI / operator / Crossplane fn do not yet embed the kernel). When they do, the migration is single-line: drop the `Module:` field from the input struct literal.
- **Constitution Principle II (Type Safety)** — input contract sharpens; one artifact handle per phase.
- **Constitution Principle VII (Simplicity)** — removes a redundant field; reduces the SemVer surface by three positions.
- **Constitution Principle VIII (Small Batch Sizes)** — three input struct edits + one accessor + fixture rewrites; tractable in a single short session.
