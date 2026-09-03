// Package schema is the kernel's single source of truth for OPM schema-side
// knowledge: CUE paths, metadata decoders, and the OCI-backed schema loader.
//
// Schema is unversioned at the package level. The library consumes exactly
// one OPM CUE schema package, opmodel.dev/core at the release
// [DefaultSchemaModule] pins, resolved through CUE's module system against
// CUE_REGISTRY. There is no in-tree schema mirror.
//
// # Path inventory
//
// CUE paths are exported as package-level cue.Path variables (Metadata,
// Components, Values, Config, Module, DebugValues).
// Callers use schema.X verbatim — there is no Paths() accessor, no struct,
// no lookup. The inventory is exactly what Go code reads: matching and
// execution happen inside the render build, in CUE, and read nothing by
// path from Go.
//
// # Metadata decoders
//
// DecodeModuleMetadata, DecodeInstanceMetadata, and DecodePlatformMetadata —
// one per artifact the kernel accepts — take a raw artifact-root cue.Value
// and return the canonical decoded struct. Missing metadata is fatal.
//
// # Schema loader and cache
//
// Loader is the strategy interface for resolving the schema; OCILoader is
// the sole public implementation, fetching [DefaultSchemaModule] through
// CUE's module system. Cache memoizes a single Loader.Load per instance
// (sync.Once-guarded) and exposes ResolvedVersion for diagnostics.
//
// Long-running consumers attach the Cache to a Kernel (via
// kernel.WithSchemaLoader) and reuse the kernel-owned cache via
// kernel.SchemaCache(). The library auto-applies no CUE_REGISTRY default;
// callers opt in by setting CUE_REGISTRY to schema.PublicRegistry (or to
// a private mirror) before the first Cache.Get.
package schema
