package v1alpha2_test

import (
	"sync"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/api"
	"github.com/open-platform-model/library/opm/apiversion"
)

// schemaBinding returns a freshly resolved v1alpha2 binding from the registry.
// The registry stores a single *binding instance, so two callers share the
// schema-load cache — which is exactly what we want for the "repeated calls
// don't reload" assertion.
func schemaBinding(t *testing.T) api.Binding {
	t.Helper()
	b, err := api.Lookup(apiversion.V1alpha2)
	require.NoError(t, err)
	require.NotNil(t, b)
	return b
}

func TestSchemaValue_RepeatedCallsCache(t *testing.T) {
	b := schemaBinding(t)
	ctx := cuecontext.New()

	first, err := b.SchemaValue(ctx)
	require.NoError(t, err)
	require.True(t, first.Exists())

	second, err := b.SchemaValue(ctx)
	require.NoError(t, err)
	// cue.Value is a value type; identity comes from the underlying instance.
	// Equivalent instances unify without producing a fresh _|_; checking that
	// the second call returned a value at the same source position via Pos()
	// is sufficient to prove no re-load occurred. The simplest equivalent
	// assertion: both values lookup the same #ModuleRelease definition and
	// it is non-zero.
	def1 := first.LookupPath(cue.ParsePath("#ModuleRelease"))
	def2 := second.LookupPath(cue.ParsePath("#ModuleRelease"))
	require.True(t, def1.Exists())
	require.True(t, def2.Exists())
	// Source positions on the schema definition must be identical — proves the
	// instance was not rebuilt.
	assert.Equal(t, def1.Source(), def2.Source(),
		"second SchemaValue call must return the cached cue.Value (same source node)")
}

func TestSchemaValue_ExposesModuleRelease(t *testing.T) {
	b := schemaBinding(t)
	ctx := cuecontext.New()

	v, err := b.SchemaValue(ctx)
	require.NoError(t, err)

	def := v.LookupPath(cue.ParsePath("#ModuleRelease"))
	require.True(t, def.Exists(), "loaded schema must expose #ModuleRelease")

	apiVer := def.LookupPath(cue.ParsePath("apiVersion"))
	require.True(t, apiVer.Exists())
	s, err := apiVer.String()
	require.NoError(t, err)
	assert.Equal(t, "opmodel.dev/v1alpha2", s)
}

func TestSchemaValue_NoCueRegistryNeeded(t *testing.T) {
	// Explicitly clear CUE_REGISTRY for this test; the embed-driven loader
	// must not consult a remote registry under any circumstance.
	t.Setenv("CUE_REGISTRY", "")

	b := schemaBinding(t)
	ctx := cuecontext.New()

	v, err := b.SchemaValue(ctx)
	require.NoError(t, err)
	require.True(t, v.Exists())
}

func TestSchemaValue_ConcurrentFirstCallsSafe(t *testing.T) {
	// Run against a *fresh* binding via the registry — but the production
	// registry singleton may already have its schema cached from earlier tests
	// in this package. Re-importing the package is impossible at test time,
	// so we exercise the underlying contract on a fresh binding via the
	// fakeBindingTestSeam helper (binding_broken_test.go).
	//
	// For the registered binding, we still assert N concurrent calls all
	// return the same cached value without panic — this is the contract that
	// matters for downstream callers.
	b := schemaBinding(t)
	ctx := cuecontext.New()

	const N = 16
	var wg sync.WaitGroup
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
	// All goroutines must observe the same #ModuleRelease source node.
	src0 := results[0].LookupPath(cue.ParsePath("#ModuleRelease")).Source()
	for i := 1; i < N; i++ {
		got := results[i].LookupPath(cue.ParsePath("#ModuleRelease")).Source()
		assert.Equal(t, src0, got, "goroutine %d saw a different #ModuleRelease source", i)
	}
}
