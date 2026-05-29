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
