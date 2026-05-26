// Package schema is the kernel's single source of truth for OPM schema-side
// knowledge: CUE paths, metadata decoders, the transformer-context builder,
// and the cached embedded-schema loader.
//
// Schema is unversioned. The library ships exactly one CUE schema vendored
// under apis/core/. Every consumer that previously dispatched through a
// per-version binding now reaches directly for the free functions and
// package-level path vars in this package.
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
// # Embedded schema loader
//
// SchemaValue returns the embedded OPM schema as a cue.Value. The first call
// builds the package via cuelang.org/go/cue/load with a synthetic overlay
// (no CUE_REGISTRY, no filesystem read); subsequent calls return the cached
// value. Schema-load failures are cached too — the load is never retried.
package schema
