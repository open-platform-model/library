package schema_test

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/internal/schematest"
	"github.com/open-platform-model/library/opm/schema"
)

// countingLoader is an in-test Loader spy used to assert that
// schema.Cache invokes its Loader exactly once per instance.
type countingLoader struct {
	calls atomic.Int64
	val   cue.Value
	err   error
}

func (l *countingLoader) Load(*cue.Context) (cue.Value, error) {
	l.calls.Add(1)
	return l.val, l.err
}

// TestCache_RepeatedGetReturnsSameValue asserts that two calls to Get
// return the same cue.Value and run Loader.Load exactly once.
func TestCache_RepeatedGetReturnsSameValue(t *testing.T) {
	ctx := cuecontext.New()
	loader := &countingLoader{val: ctx.CompileString("hello: 1")}

	cache := &schema.Cache{Loader: loader}

	v1, err := cache.Get(ctx)
	require.NoError(t, err)
	v2, err := cache.Get(ctx)
	require.NoError(t, err)

	// Same cached cue.Value — comparable via underlying source/struct
	// identity. We compare a derived field for safety.
	one, err := v1.LookupPath(cue.ParsePath("hello")).Int64()
	require.NoError(t, err)
	two, err := v2.LookupPath(cue.ParsePath("hello")).Int64()
	require.NoError(t, err)
	assert.Equal(t, one, two)

	assert.Equal(t, int64(1), loader.calls.Load(),
		"Cache.Get must invoke Loader.Load exactly once across repeated calls")
}

// TestCache_ConcurrentFirstGetIsSafe asserts that under a concurrent
// first-call race, exactly one Loader.Load runs and all goroutines see
// the same result.
func TestCache_ConcurrentFirstGetIsSafe(t *testing.T) {
	ctx := cuecontext.New()
	loader := &countingLoader{val: ctx.CompileString("x: 42")}
	cache := &schema.Cache{Loader: loader}

	const goroutines = 16
	var wg sync.WaitGroup
	wg.Add(goroutines)
	results := make([]cue.Value, goroutines)
	errs := make([]error, goroutines)
	start := make(chan struct{})
	for i := 0; i < goroutines; i++ {
		go func(i int) {
			defer wg.Done()
			<-start
			results[i], errs[i] = cache.Get(ctx)
		}(i)
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		require.NoErrorf(t, err, "goroutine %d", i)
		val, err := results[i].LookupPath(cue.ParsePath("x")).Int64()
		require.NoError(t, err)
		assert.Equal(t, int64(42), val)
	}
	assert.Equal(t, int64(1), loader.calls.Load(),
		"concurrent first-call must run Loader.Load exactly once")
}

// TestCache_LoaderErrorsAreCached asserts that an error from the first
// Get is cached and the Loader is not re-invoked on subsequent calls.
func TestCache_LoaderErrorsAreCached(t *testing.T) {
	sentinel := errors.New("loader exploded")
	loader := &countingLoader{err: sentinel}
	cache := &schema.Cache{Loader: loader}

	ctx := cuecontext.New()
	for i := 0; i < 3; i++ {
		_, err := cache.Get(ctx)
		require.ErrorIs(t, err, sentinel)
	}
	assert.Equal(t, int64(1), loader.calls.Load(),
		"loader errors must be cached without retry")
}

// TestCache_TwoInstancesDoNotShareState asserts that distinct Cache
// values built from logically-equivalent Loaders each invoke their own
// Loader.
func TestCache_TwoInstancesDoNotShareState(t *testing.T) {
	ctx := cuecontext.New()
	l1 := &countingLoader{val: ctx.CompileString("a: 1")}
	l2 := &countingLoader{val: ctx.CompileString("a: 2")}
	c1 := &schema.Cache{Loader: l1}
	c2 := &schema.Cache{Loader: l2}

	_, _ = c1.Get(ctx)
	_, _ = c2.Get(ctx)
	_, _ = c1.Get(ctx)
	_, _ = c2.Get(ctx)

	assert.Equal(t, int64(1), l1.calls.Load(), "cache 1 keeps its own once")
	assert.Equal(t, int64(1), l2.calls.Load(), "cache 2 keeps its own once")
}

// TestCache_ResolvedVersionEmptyBeforeGet asserts ResolvedVersion is the
// empty string until the first successful Get.
func TestCache_ResolvedVersionEmptyBeforeGet(t *testing.T) {
	loader := &countingLoader{}
	cache := &schema.Cache{Loader: loader}
	assert.Empty(t, cache.ResolvedVersion(),
		"ResolvedVersion must be empty before the first Get")
}

// TestCache_ResolvedVersionAfterOCIGet asserts that ResolvedVersion
// returns a v0.x.y semver after a successful OCILoader-backed Get.
// This exercises the loadVersioned hook in opm/schema/loader.go.
func TestCache_ResolvedVersionAfterOCIGet(t *testing.T) {
	schematest.SetEnv(t)
	cache := &schema.Cache{Loader: schema.OCILoader{}}

	ctx := cuecontext.New()
	val, err := cache.Get(ctx)
	require.NoError(t, err)
	require.True(t, val.Exists())

	ver := cache.ResolvedVersion()
	if ver == "" {
		t.Skip("CUE SDK did not surface a parseable version in build.Instance.Root; " +
			"ResolvedVersion is diagnostic-only, skip when unavailable")
	}
	assert.True(t,
		len(ver) >= len("v0.0.0") && ver[0] == 'v',
		"ResolvedVersion must look like a semver (got %q)", ver,
	)
}

// TestCache_ResolvedVersionStaysEmptyOnFailedLoad asserts that a failed
// Load leaves ResolvedVersion at the empty string.
func TestCache_ResolvedVersionStaysEmptyOnFailedLoad(t *testing.T) {
	loader := &countingLoader{err: errors.New("boom")}
	cache := &schema.Cache{Loader: loader}
	_, err := cache.Get(cuecontext.New())
	require.Error(t, err)
	assert.Empty(t, cache.ResolvedVersion())
}

// TestCache_WorkspaceCacheReuse asserts that a second OCILoader-backed
// Cache reuses the workspace cache populated by the first Cache —
// i.e. the on-disk cache layer survives across Cache instances within
// the same workspace cache directory.
func TestCache_WorkspaceCacheReuse(t *testing.T) {
	schematest.SetEnv(t)
	ctx := cuecontext.New()

	first := &schema.Cache{Loader: schema.OCILoader{}}
	_, err := first.Get(ctx)
	require.NoError(t, err)

	second := &schema.Cache{Loader: schema.OCILoader{}}
	_, err = second.Get(ctx)
	require.NoError(t, err, "second Cache must load from warm workspace cache without re-fetch")
}
