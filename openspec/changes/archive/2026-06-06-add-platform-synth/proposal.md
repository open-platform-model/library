## Why

The kernel can build a `#ModuleRelease` from typed in-memory inputs (`synth.Release` + `Kernel.SynthesizeRelease`), but the parallel path for `#Platform` does not exist. Today a `#Platform` can only enter the kernel by loading authored CUE (`LoadPlatformPackage` → `NewPlatformFromValue`). Frontends that hold platform configuration as typed data — an operator reconciling a Platform CRD spec, a CLI assembling subscriptions from flags — have no first-class way to construct a `#Platform` without emitting CUE source by hand. This closes that gap and restores symmetry across the three artifact types.

## What Changes

- Add `synth.Platform(ctx, PlatformInput) (cue.Value, error)` in `opm/helper/synth` — a peer of `synth.Release`. It unifies typed identity + subscription inputs against the `#Platform` definition resolved through the caller-supplied `*schema.Cache` and returns the spec `cue.Value`.
- Add `synth.PlatformInput` (typed fields: `Name`, `Type`, `SchemaCache` required; `Description`, `Labels`, `Annotations`, and a typed `Subscriptions` map optional) plus platform-specific sentinel errors mirroring the `Release` set.
- Add `Kernel.SynthesizePlatform(ctx, PlatformInput) (*platform.Platform, error)` in `opm/kernel/synth.go` — the recommended entry point. It chains `synth.Platform` into `NewPlatformFromValue`, returning a typed pre-materialize `*platform.Platform`. It does **not** call `Materialize` (registry I/O stays an explicit, separate, caller-driven step per Principle I / D14).
- Update `opm/helper/synth/doc.go`, which currently scopes the package to ModuleRelease only.
- No schema change in `core/` — the `#Platform` definition is already present in the core package returned by `*schema.Cache`.

## Capabilities

### New Capabilities

- `platform-synthesis`: Building a validated `#Platform` artifact (`*platform.Platform`) from typed in-memory inputs, as a peer to release synthesis — covering the helper primitive (`synth.Platform`), the kernel entry point (`Kernel.SynthesizePlatform`), required/optional input handling, subscription-and-filter mapping, and the boundary that synthesis stops before `Materialize`.

### Modified Capabilities

None — no existing spec-level behavior changes.

## Impact

- **New public Go surface** (additive, no break): `synth.Platform`, `synth.PlatformInput`, platform sentinel errors in `opm/helper/synth`; `Kernel.SynthesizePlatform` in `opm/kernel`.
- **Files**: new `opm/helper/synth/platform.go` (+ test), edits to `opm/kernel/synth.go`, `opm/helper/synth/doc.go`; new fixtures/tests.
- **No downstream break**: purely additive — `cli/` and `opm-operator/` keep compiling unchanged. The operator is the most likely first consumer once a Platform CRD lands, but this change ships no operator code.
- **MIGRATIONS.md**: an additive surface entry, not a break.
