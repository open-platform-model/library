// Package schema is the kernel's single source of truth for OPM schema-side
// knowledge: CUE paths, metadata decoders, the transformer-context builder,
// and the OCI-backed schema loader.
//
// Schema is unversioned at the package level. The library consumes exactly
// one OPM CUE schema package — opmodel.dev/core@v0, resolved through CUE's
// module system against CUE_REGISTRY. There is no in-tree schema mirror.
//
// # Path inventory
//
// CUE paths are exported as package-level cue.Path variables (Metadata,
// Components, Config, Module, ModuleMetadata, Registry, …). Callers use
// schema.X verbatim — there is no Paths() accessor, no struct, no lookup.
//
// # Metadata decoders
//
// DecodeModuleMetadata, DecodeReleaseMetadata, DecodeProviderMetadata, and
// DecodePlatformMetadata accept a raw artifact-root cue.Value and return the
// canonical decoded struct. Missing metadata is fatal for module / release /
// platform; provider metadata falls back to a caller-supplied name.
//
// # Transformer context
//
// BuildTransformerContext constructs the #TransformerContext value for a
// single (release, component, transformer) tuple. The caller is responsible
// for filling the returned value at schema.Context on the unified
// transformer.
//
// # Schema loader and cache
//
// Loader is the strategy interface for resolving the schema; OCILoader is
// the sole public implementation, fetching opmodel.dev/core@v0 through
// CUE's module system. Cache memoizes a single Loader.Load per instance
// (sync.Once-guarded) and exposes ResolvedVersion for diagnostics.
//
// Long-running consumers attach the Cache to a Kernel (via
// kernel.WithSchemaLoader) and reuse the kernel-owned cache via
// kernel.SchemaCache(). The library auto-applies no CUE_REGISTRY default;
// callers opt in by setting CUE_REGISTRY to schema.PublicRegistry (or to
// a private mirror) before the first Cache.Get.
package schema
