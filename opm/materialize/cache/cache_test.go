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
// is invariant to subscription ordering, field ordering, and enable
// defaulting, and moves when the subscribed version does.
func TestKey_StableAcrossSemanticallyIdenticalRegistries(t *testing.T) {
	octx := cuecontext.New()

	pA := buildPlatform(t, octx, `{
		"test.example/a@v1": {enable: true, version: "1.0.0"}
		"test.example/b@v1": {version: "1.2.0", enable: true}
	}`)
	// Same meaning, authored differently: subscription order swapped, enable
	// omitted (defaults true), field order swapped.
	pB := buildPlatform(t, octx, `{
		"test.example/b@v1": {enable: true, version: "1.2.0"}
		"test.example/a@v1": {version: "1.0.0"}
	}`)
	// Different meaning: b subscribes a different build.
	pC := buildPlatform(t, octx, `{
		"test.example/a@v1": {enable: true, version: "1.0.0"}
		"test.example/b@v1": {version: "1.5.0", enable: true}
	}`)

	keyA, err := cache.Key(pA)
	require.NoError(t, err)
	keyB, err := cache.Key(pB)
	require.NoError(t, err)
	keyC, err := cache.Key(pC)
	require.NoError(t, err)

	assert.Equal(t, keyA, keyB, "semantically-identical registries share a key")
	assert.NotEqual(t, keyA, keyC, "a different subscribed version yields a different key")
}

// TestKey_ByteIdenticalAcrossFilterFieldTrim pins a v2 platform's key to the
// value the PRE-TRIM code path produced (golden captured before normSub
// dropped the v1 filter fields): a v2 subscription could never populate
// range/allow/deny, so removing them must not move any live key — no cache
// invalidation on upgrade.
func TestKey_ByteIdenticalAcrossFilterFieldTrim(t *testing.T) {
	octx := cuecontext.New()
	p := buildPlatform(t, octx, `{
		"test.example/a@v1": {enable: true, version: "1.0.0"}
		"test.example/b@v1": {version: "1.2.0"}
	}`)

	key, err := cache.Key(p)
	require.NoError(t, err)
	assert.Equal(t, "e4d33d74239acc7ac19a69c0e415cd73661067d0d5500671f6db929cdf278d3c", key,
		"v2 platform key must be byte-identical to the pre-trim golden")
}
