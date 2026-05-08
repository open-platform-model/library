# Changelog

All notable changes to this library are documented here. The library follows [SemVer 2.0.0](https://semver.org/spec/v2.0.0.html); this changelog distinguishes Go-module SemVer from OPM-schema versioning per the README.

## Unreleased — next MINOR

### Added

- `pkg/apiversion` — `Version` type, `V1alpha2` constant, `ErrUnknownAPIVersion` sentinel, and `Detect(cue.Value) (Version, error)` helper.
- `pkg/api` — `Binding` interface, `Paths` inventory, `ModuleMetadata`/`ReleaseMetadata`/`ProviderMetadata` LCD structs, `ReleaseView` interface, registry (`Register`, `Lookup`, `For`), and `EmbeddedSchema(version) (fs.FS, error)`.
- `pkg/api/v1alpha2` — first concrete binding. Registers itself in `init()` and exposes the `apis/core/v1alpha2/` schema via `go:embed`.
- `apis/core/v1alpha2/embed.go` — embedded CUE source filesystem for offline schema validation.
- `module.Module.APIVersion`, `module.Release.APIVersion`, `provider.Provider.APIVersion` — populated by the loader at load time.
- `*module.Release` accessor methods (`ReleaseName`, `Namespace`, `ReleaseUUID`, `ModuleFQN`, `ModuleVersion`, `Labels`, `Annotations`) — make `*Release` satisfy `api.ReleaseView` for binding-driven context injection.
- `pkg/api/v1alpha2.AnnotationDefaultNamespace` constant (`"module.opmodel.dev/defaultNamespace"`) — discoverable key for the v1alpha2 default-namespace annotation. See ADR-001.

### Changed

- **BREAKING (`pkg/render`)** — `render.Match(components, p) → render.Match(components, p, b api.Binding)`. The renderer no longer reads hardcoded CUE path strings; every lookup goes through the binding. Downstream code today calls `render.ProcessModuleRelease`, which keeps its signature, so the practical migration cost is zero.
- **BREAKING (`pkg/loader`)** — `LoadReleaseFile` now returns `(cue.Value, string, apiversion.Version, error)` and `LoadModulePackage` now returns `(cue.Value, apiversion.Version, error)`. Both reject artifacts whose `apiVersion` is missing or unrecognised with an error that wraps `apiversion.ErrUnknownAPIVersion`.
- `apis/core/v1alpha2/types.cue` — `#ApiVersion` is now the literal `"opmodel.dev/v1alpha2"` (was a self-reference). User-authored artifacts evaluate to the literal automatically once they re-resolve against the new schema.
- `render.ProcessModuleRelease` — now resolves the binding via `api.Lookup(rel.APIVersion)` and rejects release/provider apiVersion mismatches before invoking any transformer.
- **BREAKING (`apis/core/v1alpha2`)** — `#Transformer` renamed to `#ComponentTransformer`; `kind: "Transformer"` → `kind: "ComponentTransformer"`. Aligns the schema file with the canonical naming used in `apis/core/v1alpha2/docs/adapters.md` and catalog enhancement 014. `#TransformerMap` and `#TransformerContext` keep their names (they describe the union and the context, not the transformer type).
- **BREAKING (`pkg/api`, `pkg/module`, `pkg/provider`)** — Go field renamed `ApiVersion` → `APIVersion` on `module.Module`, `module.Release`, `provider.Provider`. The `apiversion` *package* and `apiversion.Version` *type* keep lowercase casing. Mechanical migration for downstream consumers.

### Unified Artifact Shape

- **BREAKING (`pkg/module`)** — `module.Module` and `module.Release` collapse to the unified artifact shape `{ APIVersion, Metadata, Package cue.Value }`. The `Package` field carries the loaded CUE value and is the source of truth for every kernel-internal read; `Metadata` is a decoded ergonomic cache.
- **BREAKING (`pkg/module`)** — Removed `Module.Config`, `Module.Raw`, `Module.ModulePath`. Read these via `mod.Package.LookupPath(binding.Paths().Config)` (and equivalents) using the binding from `api.Lookup(mod.APIVersion)`.
- **BREAKING (`pkg/module`)** — Removed `Release.Module`, `Release.Spec`, `Release.Values`. Read these via `rel.Package.LookupPath(binding.Paths().Module)`, `binding.Paths().Components`, `binding.Paths().Values`. The release's `Package` already embeds the source `#module` reference.
- Added `module.NewModuleFromValue(k, v)` and `module.NewReleaseFromValue(k, v)` constructor helpers. Each detects `apiVersion` via `apiversion.Detect`, looks up the binding, decodes typed metadata, and stamps `APIVersion` on the returned struct. `Package` is set unmodified from the input.
- Added `(k *Kernel) NewModuleFromValue` / `NewReleaseFromValue` thin wrappers on `pkg/kernel/` for consumer ergonomics.
- Added `Paths.Module` (`"#module"`) and `Paths.ModuleMetadata` (`"#moduleMetadata"`) to `pkg/api` for release-side lookup of the embedded source module.

#### Migration recipe

| Before | After |
| --- | --- |
| `mod.Config` | `mod.Package.LookupPath(b.Paths().Config)` (with `b, _ := api.Lookup(mod.APIVersion)`) |
| `mod.Raw` | `mod.Package` |
| `rel.Spec` | `rel.Package` |
| `rel.Spec.LookupPath(...)` | `rel.Package.LookupPath(...)` |
| `rel.Values` | `rel.Package.LookupPath(b.Paths().Values)` |
| `rel.Module` | `rel.Package.LookupPath(b.Paths().Module)` (raw CUE) |
| `rel.Module.Metadata.FQN` | `rel.ModuleFQN()` (now reads from `Package` via the binding) |
| `rel.MatchComponents()` | unchanged — now reads through the binding internally |
| `module.Module{Spec: ..., Config: ...}` literals | `module.NewModuleFromValue(k, v)` or `module.Module{APIVersion: ..., Metadata: ..., Package: v}` |
| `&module.Release{Spec: spec, Values: merged, Module: mod}` literals | `module.NewReleaseFromValue(k, v)` or `&module.Release{APIVersion: ..., Metadata: ..., Package: spec}` |

`provider.Provider` is unchanged in this slice — it is being retired in a later slice in favour of the upcoming `Platform` type.

### Removed

- The unexported `moduleReleaseContextData` and `componentContextData` structs in `pkg/render/execute.go`. Equivalent exported types now live in `pkg/api/v1alpha2/context.go`.
- The unexported `injectContext` helper in `pkg/render/execute.go`. Replaced by a one-line delegation to `binding.BuildTransformerContext`.
- **BREAKING (`pkg/api`, `pkg/module`)** — `ModuleMetadata.DefaultNamespace` field. The value lives only as the optional `module.opmodel.dev/defaultNamespace` annotation per `apis/core/v1alpha2/module.cue:35`; `Decode()` could never populate the typed field. Implements ADR-001 (now Accepted). Read via `Module.Metadata.Annotations[v1alpha2.AnnotationDefaultNamespace]`.
- **BREAKING (`pkg/api`, `pkg/api/v1alpha2`)** — `Paths.ComponentBlueprints`. Dead code — Blueprints unify into a Component's `spec` at CUE-evaluation time per `apis/core/v1alpha2/component.cue:_allFields`; the renderer never walks them. The path was unread.
