package kernel

import (
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"

	"github.com/open-platform-model/library/opm/schema"
)

// Kernel is the public anchor type for the OPM runtime. It owns a
// [*cue.Context] and a [*schema.Cache] for its lifetime.
//
// Kernel is NOT safe for concurrent use across method calls — see the
// package documentation for the one-Kernel-per-goroutine pattern.
//
// The Kernel owns exactly one [*schema.Cache] for its lifetime. Long-
// running consumers (operator, server) MUST keep the Kernel alive
// across operations to reuse the in-process schema cache; constructing
// a fresh Kernel per request pays the schema-fetch cost on every cold
// disk cache. The CUE module cache on disk is shared across Kernels.
type Kernel struct {
	cueCtx       *cue.Context
	schemaLoader schema.Loader
	schemaCache  *schema.Cache
	registry     string
}

// Option configures a [Kernel] at construction time. Options compose via
// the functional-options pattern; new options can be added in MINOR
// instances without breaking existing call sites. The provided options
// are [WithSchemaLoader] and [WithRegistry]; the Kernel exposes no
// injection slot that no kernel operation reads.
type Option func(*Kernel)

// New constructs a [Kernel] with default dependencies and applies the
// supplied options. Defaults are:
//
//   - cue.Context: a fresh [cuecontext.New]
//   - SchemaCache: a fresh [*schema.Cache] backed by zero-value
//     [schema.OCILoader]; resolves opmodel.dev/core@v2 against
//     CUE_REGISTRY / CUE_CACHE_DIR from the process environment
//
// New never returns nil. The returned Kernel is NOT safe for concurrent
// use across method calls.
//
// New does NOT trigger a schema load. The first [Kernel] method that
// needs the schema invokes [Cache.Get] internally, which performs the
// fetch lazily.
func New(opts ...Option) *Kernel {
	k := &Kernel{
		cueCtx: cuecontext.New(),
	}
	for _, opt := range opts {
		opt(k)
	}
	// One Cache per Kernel. WithSchemaLoader sets schemaLoader; absent
	// the option, the zero-value OCILoader resolves opmodel.dev/core@v2
	// against the process environment.
	loader := k.schemaLoader
	if loader == nil {
		loader = schema.OCILoader{}
	}
	k.schemaCache = &schema.Cache{Loader: loader}
	return k
}

// WithSchemaLoader configures the [schema.Loader] used to populate the
// kernel's [*schema.Cache]. Omitting this option defaults to a
// zero-value [schema.OCILoader] that resolves opmodel.dev/core@v2 via
// CUE_REGISTRY / CUE_CACHE_DIR from the process environment.
//
// The Kernel wraps the supplied Loader in a fresh Cache; callers cannot
// inject a pre-built Cache. This guarantees one Kernel = one Cache, so
// no two Kernels accidentally share memoization. Multi-Kernel cache
// sharing is intentionally not exposed and may be added later as a
// non-breaking addition.
//
// A nil Loader is ignored (the default OCILoader applies).
func WithSchemaLoader(l schema.Loader) Option {
	return func(k *Kernel) {
		if l != nil {
			k.schemaLoader = l
		}
	}
}

// WithRegistry sets the OCI registry mapping (CUE_REGISTRY syntax, e.g.
// "opmodel.dev=ghcr.io/open-platform-model") used for catalog resolution
// during [Kernel.Materialize]. The materialize flow uses the same mapping when
// it resolves opmodel.dev/core for the schema.
//
// Omitting this option (or passing an empty string) inherits CUE_REGISTRY from
// the process environment; the kernel applies no built-in default registry —
// the same stance as the schema loader. The mapping is never written back to
// the process environment; it is plumbed into the load configuration for the
// operation only.
func WithRegistry(registry string) Option {
	return func(k *Kernel) {
		k.registry = registry
	}
}

// CueContext returns the [*cue.Context] owned by this Kernel.
//
// Advanced: most callers do not need this. Use it only when building
// [cue.Value]s outside the kernel (typically tests or programmatic CUE
// construction). Values built with this context are safe to pass back
// into Kernel methods. The same [*cue.Context] is returned for the
// lifetime of the Kernel.
func (k *Kernel) CueContext() *cue.Context {
	return k.cueCtx
}

// SchemaCache returns the [*schema.Cache] owned by this Kernel. The same
// pointer is returned for the lifetime of the Kernel; callers MAY hold
// it across operations to ensure cache reuse.
//
// Calling SchemaCache does NOT trigger a schema load. Only the first
// [schema.Cache.Get] invocation contacts CUE; the load is lazy and
// memoized.
//
// Typical use: pass to [synth.InstanceInput.SchemaCache] before calling
// instance synthesis, or read [schema.Cache.ResolvedVersion] for
// diagnostics after a schema-touching operation has run.
func (k *Kernel) SchemaCache() *schema.Cache {
	return k.schemaCache
}
