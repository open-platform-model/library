## Context

Catalog 014 introduces `#Platform` as the deployment-target artifact: it carries platform identity (`metadata`, `type`), platform context (`#ctx`, typed by enhancement 016), and a `#registry` map of `#ModuleRegistration` entries. Computed CUE views over `#registry` produce:

- `#knownResources` / `#knownTraits` / `#knownClaims` (catalogs of available primitives).
- `#composedTransformers` (every registered Module's published transformers, keyed by FQN).
- `#matchers.{resources, traits, claims}` (reverse index from primitive FQN → list of fulfilling transformers; multi-fulfiller forbidden via `_invalid` constraint).

The kernel matcher (slice 09) consumes `#composedTransformers` and `#matchers` to walk a consumer Module's demand. Before it can, the kernel needs a Go-level handle on a `Platform` value.

This slice is the type + loader landing pad. It does not change matcher behavior. The matcher continues to work against `*provider.Provider`. Once `*Platform` is constructable, slice 09 swaps the matcher's input.

Coordination: `add-multi-apiversion-support` (the prerequisite) provides the binding registry. This slice extends each binding with the platform-specific paths and decoders. Note: the Claim-related views (`#knownClaims`, `claims` sub-map of `#matchers`) are intentionally NOT exposed by the binding in this slice — Claim support is deferred per umbrella scope.

## Goals / Non-Goals

**Goals:**
- A `Platform` Go type matching the uniform shape from slice 02.
- A constructor `NewPlatformFromValue` that detects `apiVersion`, decodes `Metadata`, and stamps the `APIVersion` field — mirroring slice 02's `NewModuleFromValue` exactly.
- A loader helper `LoadPlatformFile(ctx, path, opts)` that mirrors `LoadReleaseFile`.
- Binding extensions: `Paths().Registry`, `Paths().ComposedTransformers`, `Paths().Matchers`, `Paths().KnownResources`, `Paths().KnownTraits`. `DecodePlatformMetadata`.
- Phase input structs (slice 06) gain an optional `Platform *Platform` field with godoc explaining it becomes required after slice 09.
- The kernel does NOT yet match against the Platform. Slice 09 does.

**Non-Goals:**
- Implementing `#PlatformMatch` walking. Slice 09.
- Removing `*provider.Provider`. Slice 09.
- Composing a Platform from a list of Modules in Go. That is `pkg/helper/platform/Compose` in slice 10.
- Claim-related paths or decoders (`#knownClaims`, claim-side matchers, `#ModuleTransformer` extension, `#resolution` writeback). Deferred per umbrella scope.

## Decisions

**`pkg/platform/`, not `pkg/module/platform.go`.** Reason: Platform is a distinct artifact type (catalog 014 retires Provider). A separate package is consistent with `pkg/provider/`. Future tests, helpers, and types specific to Platform have a clear home.

**`PlatformMetadata.Type`.** Catalog 014 has a top-level `type` field on `#Platform` (e.g. `"kubernetes"`, `"crossplane"`). Decoded into `Metadata.Type`. Per-binding decoder owns the field; today's decoder treats it as a free string (matching catalog 014's "informational for now" note).

**No Go-level decoded `Registry`, `Matchers`, etc.** All computed views remain accessible only via `Package.LookupPath(binding.Paths()....)`. Reason: those views are CUE-computed, often large, and consumers (slice 09 matcher) iterate them lazily. Decoding eagerly would force allocation patterns that slice 09 does not need.

**`LoadPlatformFile` mirrors `LoadReleaseFile`.** Same signature shape, same `LoadOptions`, same return: `(cue.Value, parentDir, error)`. Reason: parity helps downstream consumers; differences would create per-artifact knowledge consumers must memorize.

**Binding test coverage gates this slice.** Each binding's `Paths()` and `DecodePlatformMetadata` get explicit tests. Reason: the matcher rewrite (slice 09) trusts these paths to be correct; spec-shape regressions there would be hard to debug.

**Optional `Platform` field on inputs.** `MatchInput.Platform`, `PlanInput.Platform`, `CompileInput.Platform` are added with godoc clearly stating they're optional today and become required after slice 09. Slice 09 also removes the `Provider` field.

## Risks / Trade-offs

**Risk — premature complexity.** Adding `Platform` before slice 09 uses it could be wasted scaffolding. Mitigation: the type + loader are needed by `cli` and `opm-operator` to start constructing Platform artifacts in tests, and slice 09 reuses them directly. Net effect: net positive even before slice 09 lands.

**Risk — binding extension scope creep.** Adding too many path constants now invites bindings to be "complete" for hypothetical future paths. Mitigation: this slice ships only the Resource/Trait paths plus computed-view paths the matcher needs. Claim paths are explicitly deferred.

**Trade-off — `Type` field is a free string.** Catalog 014 says the field is informational; future enforcement (e.g. registered Modules must declare compatibility with the Platform's type) is deferred to a follow-up. The Go type matches.

**Trade-off — keeping `Provider` and `Platform` in parallel.** Two artifact types with overlapping purpose for one slice. Mitigation: clear godoc, slice 09 explicitly retires `Provider`. This is the smallest interim duplication that preserves slice independence.
