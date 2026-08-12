package materialize

import (
	"errors"
	"strconv"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/open-platform-model/library/opm/errors"
)

// topLevelKeys lists the concrete string field labels at the root of v.
func topLevelKeys(t *testing.T, v cue.Value) []string {
	t.Helper()
	it, err := v.Fields()
	require.NoError(t, err)
	var out []string
	for it.Next() {
		out = append(out, it.Selector().Unquoted())
	}
	return out
}

// syntheticBuild builds a catalogBuild whose #transformers map holds one entry
// at fqn with the given modulePath/description. Unlike the registry-backed
// fixtures, this bypasses the #Catalog pattern stamping so two builds can
// share an FQN with identical (collapsing) or divergent (conflicting) bodies —
// the only way to reach the indexCatalogs collapse branch directly.
func syntheticBuild(octx *cue.Context, sub, fqn, modulePath, desc string) catalogBuild {
	src := `{#transformers: ` + strconv.Quote(fqn) + `: {
		kind: "ComponentTransformer"
		metadata: {name: "shared", modulePath: ` + strconv.Quote(modulePath) + `, version: "1.0.0", description: ` + strconv.Quote(desc) + `}
		requiredResources: "x.example/resources/foo@1.0.0": {}
	}}`
	return catalogBuild{Subscription: sub, Version: "1.0.0", Value: octx.CompileString(src)}
}

// TestIndexCatalogs_IdenticalBuildsCollapse covers the
// "Identical builds collapse" scenario: two builds exposing byte-identical
// bodies at the same FQN unify into a single composed-map entry.
func TestIndexCatalogs_IdenticalBuildsCollapse(t *testing.T) {
	octx := cuecontext.New()
	const fqn = "x.example/transformers/shared@1.0.0"
	const mp = "x.example/transformers"
	b1 := syntheticBuild(octx, "x.example/a", fqn, mp, "same body")
	b2 := syntheticBuild(octx, "x.example/b", fqn, mp, "same body")

	composed, matchers, err := indexCatalogs(octx, []catalogBuild{b1, b2})
	require.NoError(t, err)

	assert.Equal(t, []string{fqn}, topLevelKeys(t, composed),
		"identical same-FQN bodies collapse to one composed entry")

	// The single transformer appears once in the reverse index. indexCatalogs
	// returns the bare {resources,traits} value (the #matchers prefix is added
	// only when filled onto the platform), so look up resources.<fqn> directly.
	ri := matchers.LookupPath(cue.MakePath(cue.Str("resources"), cue.Str("x.example/resources/foo@1.0.0")))
	require.True(t, ri.Exists())
	n, err := ri.Len().Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(1), n, "collapsed transformer listed once")
}

// TestIndexCatalogs_DivergentBuildsConflict covers the "Divergent builds
// conflict" scenario at the unit level: same FQN, divergent bodies → a
// MaterializeError wrapping the CUE conflict.
func TestIndexCatalogs_DivergentBuildsConflict(t *testing.T) {
	octx := cuecontext.New()
	const fqn = "x.example/transformers/shared@1.0.0"
	b1 := syntheticBuild(octx, "x.example/a", fqn, "x.example/a/transformers", "body A")
	b2 := syntheticBuild(octx, "x.example/b", fqn, "x.example/b/transformers", "body B")

	_, _, err := indexCatalogs(octx, []catalogBuild{b1, b2})
	require.Error(t, err)
	var me *oerrors.MaterializeError
	require.True(t, errors.As(err, &me), "divergence surfaces as MaterializeError: %v", err)
	assert.Equal(t, oerrors.MaterializeKindCatalog, me.Kind)
}

// syntheticGuardBuild builds a catalogBuild whose #transformers map holds one
// transformer (FQN derived from sub + name) requiring exactly one contract
// key with an explicitly declared fulfilment — the direct harness for the
// single-provider guard (0010 D32/D37).
func syntheticGuardBuild(octx *cue.Context, sub, name, requiredKey, fulfilment string) catalogBuild {
	fqn := sub + "/transformers/" + name + "@1.0.0"
	src := `{#transformers: ` + strconv.Quote(fqn) + `: {
		kind: "ComponentTransformer"
		metadata: {name: ` + strconv.Quote(name) + `, modulePath: ` + strconv.Quote(sub+"/transformers") + `, version: "1.0.0"}
		requiredResources: ` + strconv.Quote(requiredKey) + `: {
			fulfilment: ` + strconv.Quote(fulfilment) + `
		}
	}}`
	return catalogBuild{Subscription: sub, Version: "1.0.0", Value: octx.CompileString(src)}
}

// TestIndexCatalogs_SecondProviderRefused covers the guard's core scenario:
// two subscribed catalogs each supply a transformer requiring a contract
// declared fulfilment "provider" — the materialization is refused with an
// error naming both catalog paths and the key.
func TestIndexCatalogs_SecondProviderRefused(t *testing.T) {
	octx := cuecontext.New()
	const key = "x.example/resources/ingress@v1"
	a := syntheticGuardBuild(octx, "x.example/a", "nginx", key, "provider")
	b := syntheticGuardBuild(octx, "x.example/b", "traefik", key, "provider")

	_, _, err := indexCatalogs(octx, []catalogBuild{a, b})
	require.Error(t, err)
	var me *oerrors.MaterializeError
	require.True(t, errors.As(err, &me))
	assert.Equal(t, oerrors.MaterializeKindCatalog, me.Kind)
	assert.Contains(t, err.Error(), key)
	assert.Contains(t, err.Error(), "x.example/a")
	assert.Contains(t, err.Error(), "x.example/b")
}

// TestIndexCatalogs_SameCatalogProviderPlurality: exclusivity is across
// CATALOGS — two transformers of one subscribed catalog requiring the same
// provider-fulfilled key pass the guard.
func TestIndexCatalogs_SameCatalogProviderPlurality(t *testing.T) {
	octx := cuecontext.New()
	const key = "x.example/resources/ingress@v1"
	a1 := syntheticGuardBuild(octx, "x.example/a", "nginx", key, "provider")
	a2 := syntheticGuardBuild(octx, "x.example/a", "nginx-v2", key, "provider")

	_, _, err := indexCatalogs(octx, []catalogBuild{a1, a2})
	require.NoError(t, err, "one catalog supplying one provider key stays legal")
}

// TestIndexCatalogs_CatalogFulfilledPluralityAllowed: contract keys with
// fulfilment "catalog" (the default) may be supplied by any number of
// transformers from any number of catalogs — all candidates index.
func TestIndexCatalogs_CatalogFulfilledPluralityAllowed(t *testing.T) {
	octx := cuecontext.New()
	const key = "x.example/resources/volume@v1"
	a := syntheticGuardBuild(octx, "x.example/a", "csi-a", key, "catalog")
	b := syntheticGuardBuild(octx, "x.example/b", "csi-b", key, "catalog")

	_, matchers, err := indexCatalogs(octx, []catalogBuild{a, b})
	require.NoError(t, err)
	bucket := matchers.LookupPath(cue.MakePath(cue.Str("resources"), cue.Str(key)))
	require.True(t, bucket.Exists())
	n, err := bucket.Len().Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(2), n, "both candidates indexed")
}

// TestIndexCatalogs_DivergentFulfilmentRefused: embedded copies that disagree
// on a key's fulfilment are refused as divergent contract definitions —
// unifying them would mask a catalog bug.
func TestIndexCatalogs_DivergentFulfilmentRefused(t *testing.T) {
	octx := cuecontext.New()
	const key = "x.example/resources/ingress@v1"
	a := syntheticGuardBuild(octx, "x.example/a", "nginx", key, "provider")
	b := syntheticGuardBuild(octx, "x.example/b", "helper", key, "catalog")

	_, _, err := indexCatalogs(octx, []catalogBuild{a, b})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "disagree on fulfilment")
	assert.Contains(t, err.Error(), key)
	assert.Contains(t, err.Error(), "x.example/a")
	assert.Contains(t, err.Error(), "x.example/b")
}
