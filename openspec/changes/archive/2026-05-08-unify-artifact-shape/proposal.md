## Why

Today's artifact types in the kernel — `module.Module`, `module.Release`, `provider.Provider` — each have a different shape. `Module` carries `Metadata`, `Spec`, `Config`. `Release` carries `Metadata`, `Module`, `Spec`, `Values`. `Provider` carries `Metadata` and a transformer map field. There is no unifying contract; multi-version dispatch must reason about each shape independently. New artifact types (e.g. the upcoming `Platform` from catalog 014) would invent a fourth shape.

This is slice 02 of the kernel-redesign umbrella ([001-kernel-redesign-around-platform](../../../enhancements/001-kernel-redesign-around-platform/README.md)). It collapses every artifact type to one Go shape: `(APIVersion, Metadata, Package cue.Value)`. The `Package` field carries the whole loaded CUE package; the kernel reads sub-fields via the version binding from `add-multi-apiversion-support`. Decoded `Metadata` remains as an ergonomic projection — but the CUE package is the source of truth (umbrella D3, D10).

## What Changes

- Refactor `module.Module` to `{ APIVersion apiversion.Version; Metadata *ModuleMetadata; Package cue.Value }`. Remove `Spec` and `Config` fields — these become `Package.LookupPath(binding.Paths().Spec)` and `Package.LookupPath(binding.Paths().Config)` at internal call sites.
- Refactor `module.Release` to `{ APIVersion apiversion.Version; Metadata *ReleaseMetadata; Package cue.Value }`. Remove embedded `Module` and separate `Spec`, `Values` fields. The release's CUE package contains the module via the version binding's import path; values are filled into the package at the binding's values path.
- **BREAKING** to `pkg/module/` exported types — consumers reading `.Spec` / `.Config` / `.Values` directly must migrate to `Package.LookupPath(...)` via the binding.
- Introduce constructor helpers that build each artifact from a raw `cue.Value` by detecting `apiVersion`, decoding `Metadata`, and stamping the `APIVersion` field.
- `provider.Provider` is **not** modified in this slice. Provider is being retired in slice 09 in favor of `Platform` (slice 08); refactoring it now would be wasted work. Slice 08 introduces `Platform` already in the new shape.
- This is a MAJOR change for `pkg/module/`. Bump the kernel module version accordingly when this lands.

## Capabilities

### New Capabilities

- `artifact-types`: The unified Go shape for every OPM artifact accepted by the kernel. Defines the `(APIVersion, Metadata, Package)` contract and the constructor helpers that produce these types from raw `cue.Value`s.

### Modified Capabilities

None in main specs (this is the first slice to touch artifact types).

## Impact

- **`pkg/module/`** — `Module` and `Release` types refactored. Constructor helpers added (`NewModuleFromValue`, `NewReleaseFromValue`).
- **`pkg/loader/`** — module / release loading produces typed values via the new constructors. `loader.LoadModulePackage` continues to return raw `cue.Value`; a new helper builds `*Module` from it.
- **`pkg/render/`, `pkg/validate/`** — internal call sites that read `mod.Spec`, `mod.Config`, `rel.Values`, `rel.Module.Config` migrate to `Package.LookupPath` via the binding (from `add-multi-apiversion-support`).
- **`pkg/kernel/`** — Kernel wrapper methods (slice 01) update return types where applicable.
- **Downstream consumers** — `cli` and `opm-operator` reading the removed fields must migrate. Per-call-site impact is small; the binding interface from `add-multi-apiversion-support` already provides `Paths()` for clean access.
- **Constitution Principle II (Type Safety)** — uniform shape strengthens type discipline.
- **Constitution Principle VI (SemVer)** — MAJOR bump on next kernel release containing this slice.
