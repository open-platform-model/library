package kernel_test

import (
	"context"
	"os"
	"reflect"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/internal/schematest"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/platform"
	"github.com/open-platform-model/library/opm/schema"
)

// buildEmptyRegistryPlatform builds a valid #Platform with an empty #registry
// using k's own *cue.Context (so a filled materialized value shares the
// context). No subscriptions means Materialize performs no registry I/O.
func buildEmptyRegistryPlatform(t *testing.T, k *kernel.Kernel) *platform.Platform {
	t.Helper()
	octx := k.CueContext()
	schemaVal, err := k.SchemaCache().Get(octx)
	require.NoError(t, err)
	def := schemaVal.LookupPath(cue.ParsePath("#Platform"))
	concrete := octx.CompileString(`{
		kind: "Platform"
		metadata: name: "t"
		type: "kubernetes"
		#registry: {}
	}`)
	require.NoError(t, concrete.Err())
	pv := def.Unify(concrete)
	require.NoError(t, pv.Validate(cue.Concrete(false)))
	p, err := platform.NewPlatformFromValue(k, pv)
	require.NoError(t, err)
	return p
}

// TestKernel_MaterializeDelegates asserts (*Kernel).Materialize delegates to
// opm/materialize using the kernel's context, returning a MaterializedPlatform
// that wraps the source and answers the matcher's path constants.
func TestKernel_MaterializeDelegates(t *testing.T) {
	schematest.SetEnv(t)
	k := kernel.New()
	p := buildEmptyRegistryPlatform(t, k)

	mp, err := k.Materialize(context.Background(), p)
	require.NoError(t, err)
	require.NotNil(t, mp)

	assert.Same(t, p, mp.Source, "MaterializedPlatform wraps the source platform")
	assert.Empty(t, mp.Resolved, "no subscriptions resolved")
	assert.True(t, mp.Package.LookupPath(schema.ComposedTransformers).Exists(),
		"#composedTransformers slot filled (empty)")
}

// TestKernel_WithRegistryDoesNotMutateEnv asserts WithRegistry plumbs the
// mapping into the operation without writing process CUE_REGISTRY.
func TestKernel_WithRegistryDoesNotMutateEnv(t *testing.T) {
	schematest.SetEnv(t)
	before := os.Getenv("CUE_REGISTRY")
	require.NotEmpty(t, before)

	k := kernel.New(kernel.WithRegistry("different=ghcr.io/elsewhere"))
	p := buildEmptyRegistryPlatform(t, k)

	_, err := k.Materialize(context.Background(), p)
	require.NoError(t, err)

	assert.Equal(t, before, os.Getenv("CUE_REGISTRY"),
		"WithRegistry MUST NOT mutate process CUE_REGISTRY")
}

// TestKernel_HoldsNoMaterializeCache asserts the "Kernel holds no cache"
// requirement structurally: no field on the Kernel struct references a
// materialize cache type. Memoization is opt-in via opm/materialize/cache,
// wired by consumers, never by the kernel (Principle I, D14).
func TestKernel_HoldsNoMaterializeCache(t *testing.T) {
	rt := reflect.TypeOf(*kernel.New())
	for i := range rt.NumField() {
		typeName := rt.Field(i).Type.String()
		assert.NotContains(t, strings.ToLower(typeName), "materializecache",
			"Kernel field %q must not hold a materialize cache", rt.Field(i).Name)
	}
}
