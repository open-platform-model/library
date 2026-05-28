package materialize

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/internal/schematest"
	"github.com/open-platform-model/library/opm/schema"
)

// TestSpike_FillPathOntoOptionalClosedSlots de-risks design.md D2/Q1: can
// FillPath populate the optional, hidden #composedTransformers / #matchers
// slots of a closed #Platform value such that the matcher's path constants
// (schema.ComposedTransformers / schema.MatchersResources / MatchersTraits)
// read the filled content back?
//
// If this passes, D2 (single filled value) holds and Materialize can fill a
// copy of Source.Package. If it fails, the design falls back to accessor
// methods on MaterializedPlatform.
func TestSpike_FillPathOntoOptionalClosedSlots(t *testing.T) {
	cache := schematest.NewCache(t)
	ctx := cuecontextForTest(t)

	schemaVal, err := cache.Get(ctx)
	require.NoError(t, err, "load core schema")

	platformDef := schemaVal.LookupPath(cue.ParsePath("#Platform"))
	require.True(t, platformDef.Exists(), "#Platform definition must exist")

	// A minimal concrete #Platform spec (the authored input shape).
	concrete := ctx.CompileString(`{
		kind: "Platform"
		metadata: name: "spike"
		type: "kubernetes"
		#registry: {}
	}`)
	require.NoError(t, concrete.Err())

	plat := platformDef.Unify(concrete)
	require.NoError(t, plat.Validate(cue.Concrete(false)), "unified platform must be valid")

	// Build a #composedTransformers map with one FQN-keyed entry that is a
	// full #ComponentTransformer (unify the schema def with concrete fields).
	transformerDef := schemaVal.LookupPath(cue.ParsePath("#ComponentTransformer"))
	require.True(t, transformerDef.Exists())
	txConcrete := ctx.CompileString(`{
		kind: "ComponentTransformer"
		metadata: {
			modulePath:  "test.example/cat/transformers"
			version:     "1.0.0"
			name:        "demo"
			description: "spike transformer"
		}
		#transform: output: {}
	}`)
	require.NoError(t, txConcrete.Err())
	tx := transformerDef.Unify(txConcrete)
	require.NoError(t, tx.Validate(cue.Concrete(false)), "transformer must be valid")

	const fqn = "test.example/cat/transformers/demo@1.0.0"
	composed := ctx.CompileString("{}").FillPath(cue.MakePath(cue.Str(fqn)), tx)
	require.NoError(t, composed.Err())

	matchers := ctx.CompileString(`{resources: {}, traits: {}}`)
	matchers = matchers.FillPath(
		cue.MakePath(cue.Str("resources"), cue.Str("test.example/cat/resources/foo@1.0.0")),
		ctx.NewList(tx),
	)
	require.NoError(t, matchers.Err())

	// Fill onto a copy of the platform value using the matcher's own path
	// constants — the exact paths rewrite-match-materialized will read.
	filled := plat.FillPath(schema.ComposedTransformers, composed)
	filled = filled.FillPath(schema.Matchers, matchers)
	require.NoError(t, filled.Err(), "fill must not error")

	// Read back through the matcher's path constants.
	ct := filled.LookupPath(schema.ComposedTransformers)
	require.True(t, ct.Exists(), "filled #composedTransformers must be readable")
	entry := ct.LookupPath(cue.MakePath(cue.Str(fqn)))
	assert.True(t, entry.Exists(), "filled FQN entry must be present")

	mr := filled.LookupPath(schema.MatchersResources)
	require.True(t, mr.Exists(), "filled #matchers.resources must be readable")
	rentry := mr.LookupPath(cue.MakePath(cue.Str("test.example/cat/resources/foo@1.0.0")))
	assert.True(t, rentry.Exists(), "reverse-index entry must be present")

	// Original platform value must remain unfilled (FillPath is non-mutating).
	assert.False(t, plat.LookupPath(schema.ComposedTransformers).LookupPath(cue.MakePath(cue.Str(fqn))).Exists(),
		"FillPath must not mutate the source value")
}

// cuecontextForTest returns a fresh *cue.Context for spike use.
func cuecontextForTest(t *testing.T) *cue.Context {
	t.Helper()
	return cuecontext.New()
}
