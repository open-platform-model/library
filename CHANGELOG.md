# Changelog

All notable changes to this library are documented here. The library follows [SemVer 2.0.0](https://semver.org/spec/v2.0.0.html); this changelog distinguishes Go-module SemVer from OPM-schema versioning per the README.

## Unreleased — next MINOR

### Added

- `pkg/helper/` — opt-in convenience boundary. Subpackages under
  `pkg/helper/` are opinionated frontend helpers that a frontend MAY
  skip; everything outside `pkg/helper/` is part of the kernel
  contract. Boundary documented in `pkg/helper/doc.go`. See slice 07
  (`reorganize-helpers-under-helper`) of the kernel-redesign
  enhancement.
- `pkg/helper/loader/file/` — new home of the filesystem-coupled
  loader (`LoadModulePackage`, `LoadReleaseFile`, `LoadValuesFile`,
  `LoadProvider`, `LoadOptions`). Symbols, return types, and error
  semantics are unchanged from the prior `pkg/loader/` package.
- `pkg/helper/loader/bytes/` — skeleton for in-memory loading.
  Doc-only package; no functions yet. Full implementation deferred
  until a Crossplane composition fn, fuzzing harness, or in-memory
  test consumer pulls on the design.

### Changed

- **BREAKING (`pkg/loader` → `pkg/helper/loader/file`)** — the
  filesystem loader moved under the helper boundary. The old
  `pkg/loader/` import path is preserved for one SemVer cycle as a
  thin re-export shim with `// Deprecated:` notices on every symbol.
  Migration is mechanical: replace the import path and the symbols
  resolve identically. The shim is scheduled for removal in the next
  MAJOR release.

  ```diff
  - import "github.com/open-platform-model/library/pkg/loader"
  + import loader "github.com/open-platform-model/library/pkg/helper/loader/file"
  ```

  `LoadOptions` is a Go type alias (`type LoadOptions = file.LoadOptions`),
  so values constructed against either identifier are interchangeable
  during the migration window.

- `pkg/kernel` wrapper methods (`LoadModulePackage`, `LoadReleaseFile`,
  `LoadValuesFile`, `LoadProvider`) now delegate directly to
  `pkg/helper/loader/file` instead of going through the deprecated
  shim. Wrapper signatures and behaviour are unchanged.

- `pkg/kernel` phase methods — `Validate`, `Match`, `Plan`, `Compile` on `*Kernel`, each accepting a phase-specific input struct (`ValidateInput`, `MatchInput`, `PlanInput`, `CompileInput`). `Plan` returns a `*PlanResult` with component summaries, unmatched FQNs, ambiguous FQNs, and warnings — no rendered values. `Compile` returns a `*CompileResult` (re-exported from `pkg/compile`) with rendered values plus the same summary fields. The phase methods map onto frontend subcommands: vet → Validate, match → Match, plan → Plan, apply → Compile.
- `pkg/kernel.DetectAPIVersion(v cue.Value)` and `pkg/kernel.Finalize(v cue.Value)` utility methods.
- `pkg/compile.CompileResult` with `Unmatched` and `Ambiguous` fields. `pkg/compile.ModuleResult` is a `// Deprecated:` Go type alias for `CompileResult`.
- `pkg/compile.CompileModuleRelease` (file: `pkg/compile/compile_module.go`). `pkg/compile.ProcessModuleRelease` and `Kernel.ProcessModuleRelease` are `// Deprecated:` aliases for `CompileModuleRelease` and `Kernel.Compile` respectively.
- `pkg/apiversion` — `Version` type, `V1alpha2` constant, `ErrUnknownAPIVersion` sentinel, and `Detect(cue.Value) (Version, error)` helper.
- `pkg/api` — `Binding` interface, `Paths` inventory, `ModuleMetadata`/`ReleaseMetadata`/`ProviderMetadata` LCD structs, `ReleaseView` interface, registry (`Register`, `Lookup`, `For`), and `EmbeddedSchema(version) (fs.FS, error)`.
- `pkg/api/v1alpha2` — first concrete binding. Registers itself in `init()` and exposes the `apis/core/v1alpha2/` schema via `go:embed`.
- `apis/core/v1alpha2/embed.go` — embedded CUE source filesystem for offline schema validation.
- `module.Module.APIVersion`, `module.Release.APIVersion`, `provider.Provider.APIVersion` — populated by the loader at load time.
- `*module.Release` accessor methods (`ReleaseName`, `Namespace`, `ReleaseUUID`, `ModuleFQN`, `ModuleVersion`, `Labels`, `Annotations`) — make `*Release` satisfy `api.ReleaseView` for binding-driven context injection.
- `pkg/api/v1alpha2.AnnotationDefaultNamespace` constant (`"module.opmodel.dev/defaultNamespace"`) — discoverable key for the v1alpha2 default-namespace annotation. See ADR-001.

### Changed

- **BREAKING (`pkg/render` → `pkg/compile`)** — package directory renamed; import path is now `github.com/open-platform-model/library/pkg/compile`. Identifiers (`Match`, `MatchPlan`, `Module`, `NewModule`, `CompileResult`, `ModuleResult`, `CompileModuleRelease`, `ProcessModuleRelease`, `FinalizeValue`, `UnmatchedComponentsError`) are unchanged; mechanical import rewrite for downstream consumers.
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
