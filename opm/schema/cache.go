package schema

import (
	"sync"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
)

// Cache memoizes a single [Loader.Load] invocation per instance and
// exposes the resolved schema module version for diagnostics. It owns
// no goroutines and no I/O of its own; the Loader carries those
// concerns.
//
// The first [Cache.Get] invocation creates a private [cue.Context] and runs
// Loader.Load into it through sync.Once; every subsequent Get (including
// the one that loses the race) returns the same cached value or the same
// cached error. Errors are cached too — the load is never retried. To
// force a re-fetch, construct a fresh Cache with a fresh Loader.
//
// Each Cache instance owns its own memoization and its own context. The
// context is the one long-lived evaluation state a Kernel holds, and no
// accessor exposes it: a caller that must compile a value against the
// schema takes the returned value's Context. The library MUST NOT expose a
// package-level Cache singleton; long-running consumers attach the Cache
// to a Kernel (or equivalent lifetime anchor) and keep that anchor alive
// across operations.
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
// serialized via sync.Once; the call that wins creates the cache's private
// context, runs Loader.Load with it, and the rest observe the cached
// result. The context lives as long as the cached value does and is
// reachable only through that value.
//
// Returns the zero cue.Value and a non-nil error if Loader.Load fails;
// the error is cached and subsequent calls return it without re-invoking
// the Loader.
func (c *Cache) Get() (cue.Value, error) {
	c.once.Do(func() {
		ctx := cuecontext.New()
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
// "v2.0.0-alpha.4" when the default identifier resolved to that instance).
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
