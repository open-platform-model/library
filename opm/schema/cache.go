package schema

import (
	"sync"

	"cuelang.org/go/cue"
)

// Cache memoizes a single [Loader.Load] invocation per instance and
// exposes the resolved schema module version for diagnostics. It owns
// no goroutines and no I/O of its own; the Loader carries those
// concerns.
//
// The first [Cache.Get] invocation runs Loader.Load through sync.Once;
// every subsequent Get (including the one that loses the race) returns
// the same cached value or the same cached error. Errors are cached too —
// the load is never retried. To force a re-fetch, construct a fresh
// Cache with a fresh Loader.
//
// Each Cache instance owns its own memoization. The library MUST NOT
// expose a package-level Cache singleton; long-running consumers attach
// the Cache to a Kernel (or equivalent lifetime anchor) and keep that
// anchor alive across operations.
type Cache struct {
	// Loader is the strategy used to resolve and build the schema value.
	// Required.
	Loader Loader

	once sync.Once
	val  cue.Value
	err  error
	ver  string
}

// Get returns the schema [cue.Value], invoking the underlying Loader at
// most once per Cache instance. Concurrent first-call invocations are
// serialized via sync.Once; the call that wins runs Loader.Load and the
// rest observe the cached result.
//
// Returns the zero cue.Value and a non-nil error if Loader.Load fails;
// the error is cached and subsequent calls return it without re-invoking
// the Loader.
func (c *Cache) Get(ctx *cue.Context) (cue.Value, error) {
	c.once.Do(func() {
		if vl, ok := c.Loader.(versionedLoader); ok {
			c.val, c.ver, c.err = vl.loadVersioned(ctx)
			return
		}
		c.val, c.err = c.Loader.Load(ctx)
	})
	return c.val, c.err
}

// ResolvedVersion returns the schema module version that the underlying
// Loader resolved during the first successful [Cache.Get] (e.g.
// "v0.3.0" when the default identifier resolved to that release).
//
// Returns the empty string before the first successful Get, after a
// failed Get, or when the Loader does not surface a resolved version.
// The value is diagnostic-only: callers SHOULD log it but MUST NOT
// branch behavior on it.
func (c *Cache) ResolvedVersion() string {
	return c.ver
}

// versionedLoader is an internal-only interface a Loader may satisfy to
// surface the resolved schema module version to [Cache.Get]. Only
// [OCILoader] implements it today.
type versionedLoader interface {
	loadVersioned(ctx *cue.Context) (cue.Value, string, error)
}
