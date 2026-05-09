# Changelog

All notable changes to this library are documented here. The library follows [SemVer 2.0.0](https://semver.org/spec/v2.0.0.html); this changelog distinguishes Go-module SemVer from OPM-schema versioning per the README.

## Unreleased — next MAJOR

### Added — `redesign-config-validation`

- `kernel.Source` struct, `kernel.ValidateOption`, and `kernel.Partial()` —
  the minimal types the new validation surface needs. `Source.Value` MUST
  be compiled with `cue.Filename(Origin)` so per-source attribution flows
  through CUE's native `token.Pos.Filename`.
- `(*kernel.Kernel).ValidateConfigDetailed(schema, sources, opts...)` —
  unifies an ordered `[]Source` then validates the merged value. With
  `kernel.Partial()` the concrete check is skipped (closed-schema
  disallowed-field reporting still runs).
- `(*kernel.Kernel).LoadSourceFromFile`, `LoadSourceFromBytes`,
  `LoadSourceFromString` — bake `cue.Filename(Origin)` into the Source's
  value so error positions report the originating source.
- `(*module.Module).ConfigSchema()` accessor (Release already exposed it).
- `(*kernel.Kernel).ValidateModuleValues` / `ValidateModuleValuesPartial` /
  `ValidateModuleValuesDetailed` and the three matching
  `ValidateReleaseValues*` — typed convenience shortcuts that resolve
  `#config` for the caller and delegate to the primitives.

### Changed — `redesign-config-validation` (BREAKING)

- `(*kernel.Kernel).ValidateConfig` and `ValidateConfigPartial` lose their
  `contextLabel` and `name` parameters. New signatures:

  ```diff
  - func (k *Kernel) ValidateConfig(schema, values cue.Value, ctxLabel, name string) (cue.Value, *oerrors.ConfigError)
  + func (k *Kernel) ValidateConfig(schema, values cue.Value) (cue.Value, error)
  - func (k *Kernel) ValidateConfigPartial(schema, values cue.Value, ctxLabel, name string) (cue.Value, *oerrors.ConfigError)
  + func (k *Kernel) ValidateConfigPartial(schema, values cue.Value) (cue.Value, error)
  ```

  Module-name framing moves to callers via `fmt.Errorf("module %q: %w",
  name, err)`. The kernel's own `Kernel.Validate(ctx, in)` phase method
  applies the framing automatically.
- All validation functions return CUE-native errors
  (`cuelang.org/go/cue/errors.Error`-walkable). Frontends iterate with
  `cueerrors.Errors(err)` and `cueerrors.Positions(ce)`; the library no
  longer projects diagnostics into Go-typed structs.

### Removed — `redesign-config-validation` (BREAKING)

- `pkg/helper/values/` package deleted in full: `Layer`, `Stack`,
  `MultiSourceError`, `LayerError`, `ValidateAndUnify`, `KernelOwner`
  interface, all gone. Use `[]kernel.Source` and
  `(*kernel.Kernel).ValidateConfigDetailed` instead.
- `(*kernel.Kernel).ValidateAndUnify` wrapper deleted.
  `Kernel.ValidateConfigDetailed` is the canonical replacement.
- `pkg/errors/config_error.go` deleted in full: `ConfigError`,
  `GroupedErrors`, `GroupedErrorsFromError`, `groupCUEErrors`,
  `normalizeCUEPath`.
- `pkg/errors` types removed: `ValidationError`, `FieldError`,
  `ErrorLocation`, `GroupedError`, `DetailError`, `NewValidationError`.
  `TransformError` and the sentinels (`ErrValidation`, `ErrConnectivity`,
  `ErrPermission`, `ErrNotFound`) survive.

### Migration recipes — `redesign-config-validation`

```text
OLD                                                    NEW
────────────────────────────────────────────────────────────────────────────────────
k.ValidateConfig(schema, vals, "module", name)         val, err := k.ValidateConfig(schema, vals)
                                                       if err != nil { return fmt.Errorf("module %q: %w", name, err) }

cfgErr.GroupedErrors()                                 for _, ce := range cueerrors.Errors(err) {
                                                         for _, pos := range cueerrors.Positions(ce) { … }
                                                       }

cfgErr.Error()                                         var buf bytes.Buffer
                                                       cueerrors.Print(&buf, err, nil)
                                                       // (frontend owns formatting; kernel ships no printer)

helpervalues.Stack{ Layer{Name, Source, Value}, … }    []kernel.Source{ {Value, Name, Origin}, … }
helpervalues.ValidateAndUnify(k, schema, stack)        k.ValidateConfigDetailed(schema, sources)

multiSrcErr.Errors()                                   for _, ce := range cueerrors.Errors(err) {
                                                         filename := ce.Position().Filename()
                                                         // bucket / present as desired
                                                       }

ctx.CompileBytes(b)                                    src, err := k.LoadSourceFromBytes(origin, name, b)
                                                       // src.Value carries cue.Filename(origin)
```

### Added

- `(*kernel.Kernel).ValidateConfigPartial(schema, values, contextLabel, name)` —
  promotes the previously package-private partial-validation entry point onto
  the kernel surface. Tier-1 building block used by
  `pkg/helper/values.ValidateAndUnify` (see the `KernelOwner` interface widening
  below) so each layer in a Stack validates independently without flagging
  fields that other layers will fill in.
- `(*kernel.Kernel).ProcessModuleRelease(ctx, spec, mod, values)` — canonical
  name for the values-validation + spec-fill + concreteness-check + metadata-
  decode pipeline. The body is the impl previously living in
  `module.ParseModuleRelease`; the rename better describes what the method does
  (it processes the release, not just parses it).
- `module.ReleaseMetadataFromAPI(*api.ReleaseMetadata) *module.ReleaseMetadata`
  — the previously unexported `releaseMetadataFromAPI` converter is now
  exported so callers building a `*module.Release` outside of `pkg/module` (the
  kernel's `ProcessModuleRelease` impl) can construct it consistently.

### Deprecated

- `(*kernel.Kernel).ParseModuleRelease` — thin alias delegating to
  `(*kernel.Kernel).ProcessModuleRelease`. Will be removed in a future MAJOR
  release. Migration is mechanical:

  ```diff
  - rel, err := k.ParseModuleRelease(ctx, spec, mod, values)
  + rel, err := k.ProcessModuleRelease(ctx, spec, mod, values)
  ```

### Removed (BREAKING) — pkg/validate deleted; deprecated free functions folded into Kernel

- `pkg/validate/` package removed in full. Both `validate.Config` and
  `validate.ConfigPartial` are gone; the canonical implementation now lives on
  `*kernel.Kernel`.
- `module.ParseModuleRelease` (free function) removed. The canonical entry
  point is `(*kernel.Kernel).ProcessModuleRelease`; `(*kernel.Kernel).ParseModuleRelease`
  remains as a deprecated alias for one cycle.
- `compile.CompileModuleRelease` (free function) removed. The canonical entry
  point is `(*kernel.Kernel).Compile`.

  Migration matrix:

  | Old call                               | New call                                  |
  |----------------------------------------|-------------------------------------------|
  | `validate.Config(s, v, ctx, name)`     | `k.ValidateConfig(s, v, ctx, name)`       |
  | `validate.ConfigPartial(s, v, c, n)`   | `k.ValidateConfigPartial(s, v, c, n)`     |
  | `module.ParseModuleRelease(ctx, ...)`  | `k.ProcessModuleRelease(ctx, ...)`        |
  | `compile.CompileModuleRelease(ctx,...)`| `k.Compile(ctx, kernel.CompileInput{...})`|

### Changed (BREAKING) — pkg/helper/values.KernelOwner widened

- `KernelOwner` in `pkg/helper/values/` now requires a second method,
  `ValidateConfigPartial(schema, values, contextLabel, name)`, in addition to
  `CueContext()`. This is what lets the helper call back into the kernel's
  partial-validation path without importing `pkg/kernel` (which would close a
  cycle now that `pkg/validate/` is gone). `*kernel.Kernel` automatically
  satisfies the wider interface; downstream test fakes that previously
  implemented `KernelOwner` MUST add the new method.

- `(*module.Release).ConfigSchema() cue.Value` — accessor returning the
  embedded source module's `#config` schema via the binding's `Paths().Module`
  followed by `Paths().Config`. Returns the zero `cue.Value` (not an error)
  on a nil receiver, an unregistered binding for `r.APIVersion`, a missing
  `#module` reference, or a missing `#config` definition. Mirrors the
  `MatchComponents()` accessor pattern. Centralizes the `#config` lookup
  knowledge inside `pkg/module` so kernel callers stop re-walking the path.

### Changed (BREAKING) — kernel input structs lose redundant `Module` field

- `kernel.MatchInput.Module` removed. The release artifact is the sole
  module-side handle for matching; the source module, when needed, is
  reachable via `ModuleRelease.Package.LookupPath(b.Paths().Module)`.
- `kernel.PlanInput.Module` removed. The `#config` schema is reachable via
  `ModuleRelease.ConfigSchema()`.
- `kernel.CompileInput.Module` removed. `Compile`'s embedded Tier-2 validation
  step now sources its `#config` schema from the release's embedded `#module`
  reference; no parallel `*module.Module` is required.
- `ValidateInput` is unchanged in this slice; its `Module` field stays.

  Mechanical migration:

  ```diff
  - plan, err := k.Match(ctx, kernel.MatchInput{
  -     Module:        mod,
  -     ModuleRelease: rel,
  -     Platform:      plat,
  - })
  + plan, err := k.Match(ctx, kernel.MatchInput{
  +     ModuleRelease: rel,
  +     Platform:      plat,
  + })

  - out, err := k.Compile(ctx, kernel.CompileInput{
  -     Module:        mod,
  -     ModuleRelease: rel,
  -     Platform:      plat,
  -     RuntimeName:   "opm-cli",
  -     Values:        values,
  - })
  + out, err := k.Compile(ctx, kernel.CompileInput{
  +     ModuleRelease: rel,
  +     Platform:      plat,
  +     RuntimeName:   "opm-cli",
  +     Values:        values,
  + })
  ```

  Releases produced by `module.ParseModuleRelease` already embed `#module`
  at the binding's `Paths().Module`, so no additional fixture work is
  required. Hand-built test releases must embed an `#module` block to
  exercise the schema-validation path.

### Changed (BREAKING) — `core.Rendered` renamed to `core.Compiled`

- `pkg/core.Rendered` (struct) → `pkg/core.Compiled`. File
  `pkg/core/rendered.go` moved to `pkg/core/compiled.go`. Fields
  (`Value`, `Release`, `Component`, `Transformer`) unchanged.
- `pkg/compile.CompileResult.Rendered` field → `Compiled`. The list
  itself (and per-element shape) is unchanged.
- Doc-only language sweep: every "render pipeline" / "render package"
  reference in package docs and specs now reads "compile pipeline" /
  "compile package" to match the slice-06 package rename.

  Mechanical migration:

  ```diff
  - var rs []*core.Rendered
  + var rs []*core.Compiled

  - for _, r := range result.Rendered {
  + for _, r := range result.Compiled {
        // r.Value is unchanged
    }
  ```

  No alias is provided — the symbol changes in this MAJOR cycle.

### Changed (BREAKING) — kernel values input is a single cue.Value

- `pkg/validate.Config(schema, values, context, name)` — `values` parameter
  changes from `[]cue.Value` to a single `cue.Value`. The kernel no longer
  unifies internally; callers pre-merge. The zero `cue.Value{}` is treated as
  "no values supplied".
- `pkg/module.ParseModuleRelease(ctx, spec, mod, values)` — `values`
  parameter changes from `[]cue.Value` to a single `cue.Value` for the same
  reason.
- `(*Kernel).ValidateConfig` and `(*Kernel).ParseModuleRelease` wrapper
  methods follow suit.
- Layering policy (CLI `-f` stack, operator ConfigMap → Secret → CR overlay,
  XR fn composition input) now lives outside the kernel. Slice 05
  (`introduce-tiered-validation`) ships `pkg/helper/values/` as the
  recommended source-positioned Tier-1 helper.

#### Migration

```diff
- merged, cfgErr := validate.Config(schema, []cue.Value{a, b, c}, "module", name)
+ stack := values.Stack{
+     {Name: "a", Source: "a.cue", Value: a},
+     {Name: "b", Source: "b.cue", Value: b},
+     {Name: "c", Source: "c.cue", Value: c},
+ }
+ merged, msErr := k.ValidateAndUnify(schema, stack)
+ // Tier-2 safety net (also runs internally in ParseModuleRelease):
+ got, cfgErr := k.ValidateConfig(schema, merged, "module", name)
```

```diff
- rel, err := k.ParseModuleRelease(ctx, spec, mod, []cue.Value{userValues})
+ rel, err := k.ParseModuleRelease(ctx, spec, mod, userValues)
```

### Removed (BREAKING)

- `pkg/validate.UnifyAndValidate` — the slice-04 transitional shim is
  removed. Slice 05 (`introduce-tiered-validation`) ships
  `pkg/helper/values.ValidateAndUnify` as the recommended Tier-1 helper;
  consumers that just need to merge without source-positioned diagnostics
  inline the two-line loop.

  Migration:

  ```diff
  - merged := validate.UnifyAndValidate([]cue.Value{a, b, c})
  - got, cfgErr := validate.Config(schema, merged, "module", name)
  + stack := values.Stack{
  +     {Name: "defaults", Source: "embedded",      Value: a},
  +     {Name: "user",     Source: "values.cue",    Value: b},
  +     {Name: "overlay",  Source: "-f overlay.cue", Value: c},
  + }
  + merged, msErr := k.ValidateAndUnify(schema, stack)
  + if msErr != nil {
  +     // per-layer source-positioned diagnostics: msErr.Errors()
  + }
  + got, cfgErr := k.ValidateConfig(schema, merged, "module", name) // Tier-2 safety net
  ```

- `pkg/provider/` — package deleted. Catalog enhancement 014's `#Platform`
  replaces `#Provider` end-to-end; `*provider.Provider` no longer appears
  on any kernel input. Construct `*platform.Platform` instead — slice 10
  ships `pkg/helper/platform.Compose` to make this a one-line migration
  from a Module registry.
- `pkg/helper/loader/file.LoadProvider` (and its `pkg/loader/` shim
  re-export) — removed alongside the provider package.
- `(*Kernel).LoadProvider` and `(*Kernel).NewRenderModule` wrappers —
  removed; both delegated to provider-only paths.
- `(*Kernel).ProcessModuleRelease` and `compile.ProcessModuleRelease`
  deprecated aliases — removed; replace with `(*Kernel).Compile` or
  `compile.CompileModuleRelease` (now Platform-based).
- `MatchInput.Provider`, `PlanInput.Provider`, `CompileInput.Provider`
  fields — removed; the `Platform` field is now required on each.

### Changed (BREAKING)

- `pkg/compile.Match(components, plat *platform.Platform, b api.Binding)` —
  the matcher now consumes `Platform.#composedTransformers` and walks
  each component's `#resources`/`#traits` keys against
  `Platform.#matchers.{resources, traits}` via the binding paths.
  `requiredLabels` matching is preserved verbatim. Multi-fulfiller is
  forbidden at the platform layer (catalog 014 D13); the matcher keeps
  a defensive ambiguity check that surfaces in `MatchPlan.Ambiguous`.
- `pkg/compile.NewModule(plat *platform.Platform, runtimeName string)` —
  signature changed; `*provider.Provider` parameter retired.
- `pkg/compile.CompileModuleRelease(ctx, rel, plat, runtimeName)` —
  signature changed; takes `*platform.Platform` instead of
  `*provider.Provider`. APIVersion mismatch error now reads
  `release "X" has "..." but platform "Y" has "..."`.
- `pkg/compile.MatchPlan` — gained an `Ambiguous []string` field for FQNs
  that resolved to more than one transformer at the platform-matchers
  layer.

### Migration

```diff
- p, err := loader.LoadProvider("kubernetes", providers)
- result, err := k.Compile(ctx, kernel.CompileInput{
-     Module: mod, ModuleRelease: rel, Provider: p, RuntimeName: "opm-cli",
- })
+ // Compose a Platform from your Module registry. Slice 10 will ship
+ // pkg/helper/platform.Compose; until then build it inline:
+ platVal, _, err := k.LoadPlatformFile(ctx, "./platform.cue", loader.LoadOptions{})
+ plat, err := k.NewPlatformFromValue(platVal)
+ result, err := k.Compile(ctx, kernel.CompileInput{
+     Module: mod, ModuleRelease: rel, Platform: plat, RuntimeName: "opm-cli",
+ })
```

## Unreleased — next MINOR

### Added

- `pkg/helper/values/` — new helper package shipping the Tier-1 layered
  values validator. `Layer{Name, Source, Value}` labels a single values
  source; `Stack []Layer` is ordered later-overrides-earlier; and
  `ValidateAndUnify(k, schema, stack) (cue.Value, *MultiSourceError)`
  validates each layer independently against the `#config` schema (partial
  mode — see `validate.ConfigPartial`), then unifies in stack order on
  success. Per-layer failures aggregate into `*MultiSourceError`, exposing
  `Errors() []LayerError` (with `LayerName`, `Source`, and the underlying
  `*oerrors.ConfigError`) and `Unwrap() []error` for stdlib `errors.Is/As`
  walks. The kernel ships an ergonomic shortcut
  `(*Kernel).ValidateAndUnify(schema, stack)` delegating to the helper.
  Frontends pass the unified result to the kernel's Tier-2 validation
  (`(*Kernel).ValidateConfig`, or downstream methods like
  `(*Kernel).ParseModuleRelease`) so both tiers run. Slice 05 of the
  kernel-redesign-around-platform enhancement; see decisions D1 and D5.
- `pkg/validate.ConfigPartial(schema, values, context, name)` — Tier-1
  building block used by `pkg/helper/values`. Same signature and error
  shape as `validate.Config` but without `cue.Concrete(true)`, so partial
  layers (overlays that intentionally leave fields unset) validate without
  noise. The merged result is still re-validated by `Config`/`ValidateConfig`
  (Tier-2) with full concreteness.

- `pkg/api.Paths.DebugValues` (`"debugValues"`) — module-internal field
  path exposing `#Module.debugValues` for frontend reads. The
  v1alpha2 binding populates it. `#ModuleDebug` is **not** a kernel
  artifact: the kernel accepts only `Module`, `ModuleRelease`, and
  `Platform`. Whether `debugValues` participates in the values stack
  is a frontend policy decision (operator: never in prod; CLI: when
  `--debug` is set; XR fn: per-composition). Slice 03 of the
  kernel-redesign-around-platform enhancement; see D6.

  Migration recipe — read debug overlays directly off the Module:

  ```go
  b, _ := api.Lookup(mod.APIVersion)
  dbg := mod.Package.LookupPath(b.Paths().DebugValues)
  // feed dbg into the helper-side values stack at the layer your frontend prefers
  ```

  No public Go type was removed; no `LoadModuleDebug` ever existed.
  The retirement is a contract sharpening — the kernel never gains
  awareness of a debug artifact, and bindings never grow a
  `Decode*Debug*Metadata` decoder (enforced by unit test in
  `pkg/api/v1alpha2/binding_test.go`).

- `pkg/helper/platform/` — new helper package shipping
  `Compose(owner, shell, modules) (*Platform, error)` and the kernel
  wrapper `(*Kernel).ComposePlatform(shell, modules)`. Builds a fully
  registered Platform by FillPath-injecting each `*module.Module` into
  `shell.Package` at `binding.Paths().Registry[<id>]` (id =
  `module.Metadata.Name`), with `enabled: true` set explicitly. Inputs
  are not mutated; calling Compose twice with the same inputs is
  idempotent. Multi-fulfiller violations (catalog enhancement
  [014](../catalog/enhancements/014-platform-construct/) D13) surface as
  `*helper/platform.MultiFulfillerError`, which carries parsed
  attribution (`FQN`, `ConflictingModules`, `ConflictingTransformers`)
  when extractable and otherwise wraps the raw CUE diagnostic via
  `Unwrap`. Slice 10 of the kernel-redesign-around-platform enhancement.

- `pkg/platform/` — new package introducing the `Platform` Go type that
  mirrors catalog enhancement
  [014-platform-construct](../catalog/enhancements/014-platform-construct/)'s
  `#Platform`. `Platform` follows the unified
  `(APIVersion, Metadata, Package)` artifact shape. Construct via
  `platform.NewPlatformFromValue(k, v)` or the kernel wrapper
  `(*Kernel).NewPlatformFromValue`. The constructor's first parameter is
  typed as the small `platform.CueContextOwner` interface (a single
  `CueContext() *cue.Context` method) so `pkg/platform` does not import
  `pkg/kernel`; `*kernel.Kernel` satisfies the interface, so call sites
  are unchanged. Mirrors the `module.NewModuleFromValue` shape. The four
  CUE-computed views
  (`#knownResources`, `#knownTraits`, `#composedTransformers`,
  `#matchers`) remain accessible only via `Package.LookupPath` using the
  new binding paths — they are intentionally not eagerly decoded into Go.
- `pkg/helper/loader/file.LoadPlatformFile(ctx, path, opts)` and
  `(*Kernel).LoadPlatformFile(ctx, path, opts)` — load a `#Platform`
  from a standalone `.cue` file or from a directory containing
  `platform.cue`. Mirrors `LoadReleaseFile` in shape, return type, and
  `LoadOptions`.
- `pkg/api.PlatformMetadata` — canonical decoded platform-level
  metadata (`Name`, `Type`, `Description`, `Labels`, `Annotations`).
  `Type` is hoisted from the top-level `#Platform.type` field into the
  metadata projection per catalog 014.
- `pkg/api.Binding.DecodePlatformMetadata(v)` and `pkg/api.Paths`
  extensions: `Registry` (`#registry`), `KnownResources`
  (`#knownResources`), `KnownTraits` (`#knownTraits`),
  `ComposedTransformers` (`#composedTransformers`), `Matchers`
  (`#matchers`). The v1alpha2 binding implements them all.
- `pkg/kernel.MatchInput.Platform`, `PlanInput.Platform`, and
  `CompileInput.Platform` — optional `*platform.Platform` fields. Today
  the phase methods continue to drive matching off `Provider` and ignore
  `Platform`. Slice 09 (`rewrite-match-around-platform`) makes
  `Platform` required and removes the `Provider` field.
- `library/testdata/platform/v1alpha2/platform.cue` — minimal Platform
  fixture for the new tests.

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
