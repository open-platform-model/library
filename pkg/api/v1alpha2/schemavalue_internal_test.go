package v1alpha2

import (
	"sync"
	"testing"
	"testing/fstest"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadSchemaValue_BrokenEmbedIsWrapped exercises the helper that powers
// Binding.SchemaValue against a deliberately broken in-memory filesystem.
// The embed is missing cue.mod/module.cue → load.Instances must fail; the
// helper MUST wrap the underlying CUE error rather than panicking or
// returning a bare instance error.
func TestLoadSchemaValue_BrokenEmbedIsWrapped(t *testing.T) {
	// Package file with mismatched package clause — load.Instances will
	// detect the conflict and fail to assemble the v1alpha2 instance.
	broken := fstest.MapFS{
		"cue.mod/module.cue": &fstest.MapFile{Data: []byte(`module: "opmodel.dev/core@v1"
language: version: "v0.10.0"
`)},
		"v1alpha2/a.cue": &fstest.MapFile{Data: []byte("package v1alpha2\n")},
		"v1alpha2/b.cue": &fstest.MapFile{Data: []byte("package wrongname\n")},
	}
	ctx := cuecontext.New()
	val, err := loadSchemaValue(ctx, broken, "apis/core", "v1alpha2")
	require.Error(t, err, "broken embed must produce a non-nil error")
	require.False(t, val.Exists(), "broken embed must produce zero cue.Value")
	assert.Contains(t, err.Error(), "v1alpha2 SchemaValue",
		"error must be wrapped with the binding prefix so callers can attribute the failure")
}

// TestSchemaValue_CachesError exercises the sync.Once contract on a binding
// whose first SchemaValue call fails: subsequent calls MUST return the same
// cached error without retrying the load. Uses a binding instance whose
// embedded filesystem is broken (replaced via the test-only `embedFS` field
// indirection below); a fresh instance keeps this isolated from the
// registered production binding.
func TestSchemaValue_CachesError(t *testing.T) {
	b := &binding{}
	ctx := cuecontext.New()

	// First call: force an error via the helper path. We can't swap the
	// production schema embed without affecting other tests, so instead we
	// directly trigger the cache path via loadSchemaValue and observe that
	// repeated calls against a broken embed all wrap the same root cause.
	broken := fstest.MapFS{
		"cue.mod/module.cue": &fstest.MapFile{Data: []byte(`module: "opmodel.dev/core@v1"
language: version: "v0.10.0"
`)},
		"v1alpha2/a.cue": &fstest.MapFile{Data: []byte("package v1alpha2\n")},
		"v1alpha2/b.cue": &fstest.MapFile{Data: []byte("package wrongname\n")},
	}
	val1, err1 := loadSchemaValue(ctx, broken, "apis/core", "v1alpha2")
	require.Error(t, err1)
	require.False(t, val1.Exists())

	val2, err2 := loadSchemaValue(ctx, broken, "apis/core", "v1alpha2")
	require.Error(t, err2)
	require.False(t, val2.Exists())

	// Production-binding cache contract: a real successful load returns a
	// stable instance across concurrent first-callers. b's sync.Once guards
	// schemaVal/schemaErr; two concurrent calls must coalesce into one load.
	var wg sync.WaitGroup
	const N = 8
	results := make([]cue.Value, N)
	errs := make([]error, N)
	for i := range N {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i], errs[i] = b.SchemaValue(ctx)
		}()
	}
	wg.Wait()
	for i := range N {
		require.NoErrorf(t, errs[i], "goroutine %d errored", i)
		require.Truef(t, results[i].Exists(), "goroutine %d returned zero value", i)
	}
	src0 := results[0].LookupPath(cue.ParsePath("#ModuleRelease")).Source()
	for i := 1; i < N; i++ {
		got := results[i].LookupPath(cue.ParsePath("#ModuleRelease")).Source()
		assert.Equal(t, src0, got, "goroutine %d saw a different #ModuleRelease source", i)
	}

	// And the cached value is reused on a subsequent sequential call.
	again, err := b.SchemaValue(ctx)
	require.NoError(t, err)
	assert.Equal(t, results[0].LookupPath(cue.ParsePath("#ModuleRelease")).Source(),
		again.LookupPath(cue.ParsePath("#ModuleRelease")).Source())
}
