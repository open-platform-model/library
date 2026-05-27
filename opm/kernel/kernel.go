package kernel

import (
	"io"
	"log/slog"
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/open-platform-model/library/opm/schema"
)

// Kernel is the public anchor type for the OPM runtime. It owns a
// [*cue.Context] for its lifetime and carries the cross-cutting
// dependencies (logger, tracer, clock, schema cache) used by every
// kernel operation.
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
	logger       *slog.Logger
	tracer       trace.Tracer
	clock        Clock
	schemaLoader schema.Loader
	schemaCache  *schema.Cache
}

// Option configures a [Kernel] at construction time. Options compose via
// the functional-options pattern; new options can be added in MINOR
// releases without breaking existing call sites.
type Option func(*Kernel)

// Clock is the kernel's view of wall-clock time. The interface is
// intentionally minimal: future slices may consult [Clock.Now] for
// deterministic rendering when render becomes time-dependent. Pass a
// fake [Clock] via [WithClock] in tests that need to pin time.
type Clock interface {
	Now() time.Time
}

// systemClock is the default [Clock] backed by [time.Now].
type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

// New constructs a [Kernel] with default dependencies and applies the
// supplied options. Defaults are:
//
//   - cue.Context: a fresh [cuecontext.New]
//   - Logger:      a no-op [*slog.Logger] (writes are discarded)
//   - Tracer:      a no-op OpenTelemetry tracer
//   - Clock:       wall-clock time via [time.Now]
//   - SchemaCache: a fresh [*schema.Cache] backed by zero-value
//     [schema.OCILoader]; resolves opmodel.dev/core@v0 against
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
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		tracer: noop.NewTracerProvider().Tracer(""),
		clock:  systemClock{},
	}
	for _, opt := range opts {
		opt(k)
	}
	// One Cache per Kernel. WithSchemaLoader sets schemaLoader; absent
	// the option, the zero-value OCILoader resolves opmodel.dev/core@v0
	// against the process environment.
	loader := k.schemaLoader
	if loader == nil {
		loader = schema.OCILoader{}
	}
	k.schemaCache = &schema.Cache{Loader: loader}
	return k
}

// WithLogger overrides the kernel's internal [*slog.Logger]. The logger
// is used for kernel-internal diagnostics only; it is intentionally not
// exposed back to callers.
func WithLogger(l *slog.Logger) Option {
	return func(k *Kernel) {
		if l != nil {
			k.logger = l
		}
	}
}

// WithTracer overrides the kernel's internal OpenTelemetry [trace.Tracer].
// The tracer is used to emit spans for kernel operations once those slices
// land; in this slice it is a passive slot.
func WithTracer(t trace.Tracer) Option {
	return func(k *Kernel) {
		if t != nil {
			k.tracer = t
		}
	}
}

// WithClock overrides the kernel's [Clock]. Use a fake clock in tests that
// need time pinned to a specific instant.
func WithClock(c Clock) Option {
	return func(k *Kernel) {
		if c != nil {
			k.clock = c
		}
	}
}

// WithSchemaLoader configures the [schema.Loader] used to populate the
// kernel's [*schema.Cache]. Omitting this option defaults to a
// zero-value [schema.OCILoader] that resolves opmodel.dev/core@v0 via
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
// Typical use: pass to [synth.ReleaseInput.SchemaCache] before calling
// release synthesis, or read [schema.Cache.ResolvedVersion] for
// diagnostics after a schema-touching operation has run.
func (k *Kernel) SchemaCache() *schema.Cache {
	return k.schemaCache
}
