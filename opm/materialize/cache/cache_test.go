package cache_test

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/internal/schematest"
	"github.com/open-platform-model/library/opm/materialize"
	"github.com/open-platform-model/library/opm/materialize/cache"
	"github.com/open-platform-model/library/opm/platform"
	"github.com/open-platform-model/library/opm/schema"
)

type ctxOwner struct{ ctx *cue.Context }

func (o ctxOwner) CueContext() *cue.Context { return o.ctx }

// buildPlatform builds a *platform.Platform with the given #registry body,
// validated against core's #Platform. No catalog registry server is needed —
// Key derivation only reads the platform's own #registry value.
func buildPlatform(t *testing.T, octx *cue.Context, registryBody string) *platform.Platform {
	t.Helper()
	schematest.SetEnv(t)
	sc := &schema.Cache{Loader: schema.OCILoader{}}
	schemaVal, err := sc.Get(octx)
	require.NoError(t, err)

	def := schemaVal.LookupPath(cue.ParsePath("#Platform"))
	concrete := octx.CompileString(`{
		kind: "Platform"
		metadata: name: "t"
		type: "kubernetes"
		#registry: ` + registryBody + `
	}`)
	require.NoError(t, concrete.Err())
	pv := def.Unify(concrete)
	require.NoError(t, pv.Validate(cue.Concrete(false)))

	p, err := platform.NewPlatformFromValue(ctxOwner{octx}, pv)
	require.NoError(t, err)
	return p
}

func TestLRU_RoundTrip(t *testing.T) {
	c := cache.NewLRU(2)
	mp := &materialize.MaterializedPlatform{Resolved: map[string]string{"test.example/a": "1.0.0"}}

	_, ok := c.Get("k1")
	assert.False(t, ok, "miss before put")

	c.Put("k1", mp)
	got, ok := c.Get("k1")
	require.True(t, ok, "hit after put")
	assert.Same(t, mp, got)
}

func TestLRU_Eviction(t *testing.T) {
	c := cache.NewLRU(2)
	a := &materialize.MaterializedPlatform{Resolved: map[string]string{"a": "1"}}
	b := &materialize.MaterializedPlatform{Resolved: map[string]string{"b": "1"}}
	d := &materialize.MaterializedPlatform{Resolved: map[string]string{"d": "1"}}

	c.Put("a", a)
	c.Put("b", b)
	_, _ = c.Get("a") // touch a → b is now least-recently-used
	c.Put("d", d)     // evicts b

	_, ok := c.Get("b")
	assert.False(t, ok, "least-recently-used entry evicted")
	_, ok = c.Get("a")
	assert.True(t, ok, "recently-used entry retained")
	_, ok = c.Get("d")
	assert.True(t, ok, "newest entry retained")
}

func TestLRU_ZeroCapacityDisabled(t *testing.T) {
	c := cache.NewLRU(0)
	c.Put("k", &materialize.MaterializedPlatform{})
	_, ok := c.Get("k")
	assert.False(t, ok, "non-positive capacity stores nothing")
}

// TestKey_StableAcrossSemanticallyIdenticalRegistries asserts the derived key
// is invariant to field ordering, enable defaulting, and allow/deny ordering.
func TestKey_StableAcrossSemanticallyIdenticalRegistries(t *testing.T) {
	octx := cuecontext.New()

	pA := buildPlatform(t, octx, `{
		"test.example/a": {enable: true}
		"test.example/b": {filter: {range: ">=1.0.0", allow: ["1.2.0", "1.1.0"]}}
	}`)
	// Same meaning, authored differently: subscription order swapped, enable
	// omitted (defaults true), allow list reordered, filter fields reordered.
	pB := buildPlatform(t, octx, `{
		"test.example/b": {filter: {allow: ["1.1.0", "1.2.0"], range: ">=1.0.0"}}
		"test.example/a": {}
	}`)
	// Different meaning: a's range narrowed.
	pC := buildPlatform(t, octx, `{
		"test.example/a": {enable: true}
		"test.example/b": {filter: {range: ">=2.0.0", allow: ["1.2.0", "1.1.0"]}}
	}`)

	keyA, err := cache.Key(pA)
	require.NoError(t, err)
	keyB, err := cache.Key(pB)
	require.NoError(t, err)
	keyC, err := cache.Key(pC)
	require.NoError(t, err)

	assert.Equal(t, keyA, keyB, "semantically-identical registries share a key")
	assert.NotEqual(t, keyA, keyC, "a different filter yields a different key")
}
