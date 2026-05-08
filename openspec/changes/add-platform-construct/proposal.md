## Why

Catalog enhancement [014-platform-construct](../../../../catalog/enhancements/014-platform-construct/) retires `#Provider` and replaces it with `#Platform` — a richer artifact that holds a `#registry` of registered Modules and computes `#composedTransformers`, `#matchers.{resources,traits}`, `#knownResources`, `#knownTraits` views over the registry. The kernel needs a Go type that mirrors this construct so the match phase (slice 09) can consume it.

This is slice 08 of the kernel-redesign umbrella ([001-kernel-redesign-around-platform](../../../enhancements/001-kernel-redesign-around-platform/README.md)). It introduces the `Platform` Go type with the uniform `(APIVersion, Metadata, Package)` shape established by slice 02. It does NOT yet rewrite the matcher (slice 09) and does NOT remove `Provider` (it lives in parallel until slice 09 retires it). It also adds a Tier-1 helper for loading platform CUE artifacts via the existing loader machinery.

## What Changes

- Introduce `pkg/platform/` package with:
  - `Platform` struct: `{ APIVersion apiversion.Version; Metadata *PlatformMetadata; Package cue.Value }`.
  - `PlatformMetadata` struct: name, type, description, labels, annotations (mirroring catalog 014's `metadata` block plus the `type` field).
  - `NewPlatformFromValue(k *kernel.Kernel, v cue.Value) (*Platform, error)` constructor analogous to `NewModuleFromValue` from slice 02.
- Extend the version binding interface (from `add-multi-apiversion-support`) to expose paths within a Platform package: `Registry`, `KnownResources`, `KnownTraits`, `ComposedTransformers`, `Matchers`. Each binding (`pkg/api/v1alpha2/`) implements these.
- Add `pkg/helper/loader/file/platform.go` with `LoadPlatformFile(ctx, path, opts)` mirroring `LoadReleaseFile` for platform.cue artifacts.
- Add an `(k *Kernel) LoadPlatformFile(...)` wrapper.
- Extend phase input structs (from slice 06) with an optional `Platform *Platform` field. Slice 09 will make it required and remove `Provider`.
- This is a MINOR change. Provider remains; Platform is additive. The `Platform` field on inputs is initially optional and ignored if `Provider` is set.

## Capabilities

### New Capabilities

- `platform-artifact`: The `Platform` Go type, its construction from CUE, its loading helper, and the binding-side path constants needed to read its computed views.

### Modified Capabilities

- `artifact-types`: Adds `Platform` to the set of artifact types accepted by the kernel; consistent with the uniform `(APIVersion, Metadata, Package)` shape.

## Impact

- **`pkg/platform/` (new)** — `Platform` type, `PlatformMetadata`, `NewPlatformFromValue`.
- **`pkg/api/v1alpha2/`** — adds `Paths().Registry`, `Paths().KnownResources`, `Paths().KnownTraits`, `Paths().ComposedTransformers`, `Paths().Matchers`. Adds `DecodePlatformMetadata`.
- **`pkg/helper/loader/file/`** — adds `LoadPlatformFile`. Mirrors `LoadReleaseFile` shape and behavior.
- **`pkg/kernel/`** — adds `LoadPlatformFile` wrapper; extends `MatchInput`, `PlanInput`, `CompileInput` with optional `Platform *Platform` field; documents that the field becomes required after slice 09.
- **`pkg/provider/`** — unchanged. Provider remains the matcher's input until slice 09.
- **Downstream consumers** — `cli` and `opm-operator` MAY start constructing `Platform` artifacts; the kernel does not yet require it.
- **Constitution Principle II (Type Safety)** — uniform shape upheld; new artifact follows the contract.
- **Constitution Principle V (CUE-Native Module Resolution)** — Platform is composed via CUE registration; binding paths point at CUE-computed views.
