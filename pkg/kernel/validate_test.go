package kernel_test

import (
	"testing"

	"cuelang.org/go/cue"
	cueerrors "cuelang.org/go/cue/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/pkg/kernel"
)

func TestKernel_ValidateConfig_ZeroValueIsNoOp(t *testing.T) {
	k := kernel.New()
	schema := k.CueContext().CompileString(`{ replicas: int & >0 }`)
	require.NoError(t, schema.Err())

	got, err := k.ValidateConfig(schema, cue.Value{})
	require.NoError(t, err, "zero cue.Value MUST be treated as 'no values' and succeed")
	assert.False(t, got.Exists(), "no values supplied → returned cue.Value is the zero value")
}

func TestKernel_ValidateConfig_SchemaErrorReturnsCueError(t *testing.T) {
	k := kernel.New()
	schema := k.CueContext().CompileString(`{ replicas: int & >0 }`)
	require.NoError(t, schema.Err())
	bad := k.CueContext().CompileString(`{ replicas: -1 }`)
	require.NoError(t, bad.Err())

	_, vErr := k.ValidateConfig(schema, bad)
	require.Error(t, vErr)
	// Error MUST be walkable as a CUE error tree.
	require.NotEmpty(t, cueerrors.Errors(vErr), "validation error MUST yield at least one cueerrors.Error")
}

func TestKernel_ValidateConfig_FieldNotAllowed(t *testing.T) {
	k := kernel.New()
	schema := k.CueContext().CompileString(`close({ replicas: int })`)
	require.NoError(t, schema.Err())
	stray := k.CueContext().CompileString(`{ replicas: 1, stray: "x" }`)
	require.NoError(t, stray.Err())

	_, vErr := k.ValidateConfig(schema, stray)
	require.Error(t, vErr, "field outside the closed schema MUST surface")
}

func TestKernel_ValidateConfigPartial_MissingFieldPassesPartialFailsFull(t *testing.T) {
	k := kernel.New()
	// Schema requires both `replicas` and `name` to be concrete.
	schema := k.CueContext().CompileString(`{ replicas: int & >0, name: string }`)
	require.NoError(t, schema.Err())
	// Partial value sets only `replicas`; `name` is missing.
	partial := k.CueContext().CompileString(`{ replicas: 3 }`)
	require.NoError(t, partial.Err())

	_, partialErr := k.ValidateConfigPartial(schema, partial)
	require.NoError(t, partialErr, "partial validation MUST allow missing required fields")

	_, fullErr := k.ValidateConfig(schema, partial)
	require.Error(t, fullErr, "full Tier-2 validation MUST flag missing required field")
}

func TestKernel_ValidateConfigPartial_TypeErrorStillSurfaces(t *testing.T) {
	k := kernel.New()
	schema := k.CueContext().CompileString(`{ replicas: int & >0, name: string }`)
	require.NoError(t, schema.Err())
	// `replicas` set but with the wrong type — partial validation MUST flag it.
	wrongType := k.CueContext().CompileString(`{ replicas: "three" }`)
	require.NoError(t, wrongType.Err())

	_, vErr := k.ValidateConfigPartial(schema, wrongType)
	require.Error(t, vErr, "partial validation still flags type errors on fields that ARE set")
	require.NotEmpty(t, cueerrors.Errors(vErr))
}

func TestKernel_ValidateConfigPartial_ZeroValueIsNoOp(t *testing.T) {
	k := kernel.New()
	schema := k.CueContext().CompileString(`{ replicas: int & >0 }`)
	require.NoError(t, schema.Err())

	got, err := k.ValidateConfigPartial(schema, cue.Value{})
	require.NoError(t, err)
	assert.False(t, got.Exists())
}

func TestKernel_ValidateConfigDetailed_EmptySourcesReturnsZeroNil(t *testing.T) {
	k := kernel.New()
	schema := k.CueContext().CompileString(`{ replicas: int & >0 }`)
	require.NoError(t, schema.Err())

	got, err := k.ValidateConfigDetailed(schema, nil)
	require.NoError(t, err)
	assert.False(t, got.Exists(), "empty sources MUST return zero cue.Value, nil error")
}

func TestKernel_ValidateConfigDetailed_SingleSourceSuccess(t *testing.T) {
	k := kernel.New()
	schema := k.CueContext().CompileString(`{ replicas: int & >0 }`)
	require.NoError(t, schema.Err())
	v, err := k.LoadSourceFromString("user.cue", "user", `replicas: 3`)
	require.NoError(t, err)

	merged, vErr := k.ValidateConfigDetailed(schema, []kernel.Source{v})
	require.NoError(t, vErr)
	assert.True(t, merged.Exists())
}

func TestKernel_ValidateConfigDetailed_TwoSourceUnifySuccess(t *testing.T) {
	k := kernel.New()
	schema := k.CueContext().CompileString(`{ replicas: int & >0, image: string }`)
	require.NoError(t, schema.Err())

	a, err := k.LoadSourceFromString("defaults.cue", "defaults", `replicas: 1`)
	require.NoError(t, err)
	b, err := k.LoadSourceFromString("user.cue", "user", `image: "nginx"`)
	require.NoError(t, err)

	merged, vErr := k.ValidateConfigDetailed(schema, []kernel.Source{a, b})
	require.NoError(t, vErr)
	require.True(t, merged.Exists())

	// Sanity: both fields present in the merged value.
	rep, _ := merged.LookupPath(cue.ParsePath("replicas")).Int64()
	assert.Equal(t, int64(1), rep)
}

func TestKernel_ValidateConfigDetailed_ConflictSurfacesBothPositions(t *testing.T) {
	k := kernel.New()
	schema := k.CueContext().CompileString(`{ replicas: int & >0 }`)
	require.NoError(t, schema.Err())

	a, err := k.LoadSourceFromString("a.cue", "layer-a", `replicas: 3`)
	require.NoError(t, err)
	b, err := k.LoadSourceFromString("b.cue", "layer-b", `replicas: 5`)
	require.NoError(t, err)

	_, vErr := k.ValidateConfigDetailed(schema, []kernel.Source{a, b})
	require.Error(t, vErr, "conflicting concrete values MUST fail")

	filenames := map[string]bool{}
	for _, ce := range cueerrors.Errors(vErr) {
		for _, pos := range cueerrors.Positions(ce) {
			if pos.IsValid() {
				filenames[pos.Filename()] = true
			}
		}
	}
	assert.True(t, filenames["a.cue"], "diagnostics MUST cite the originating Source.Origin (a.cue)")
	assert.True(t, filenames["b.cue"], "diagnostics MUST cite the originating Source.Origin (b.cue)")
}

func TestKernel_ValidateConfigDetailed_PartialSkipsConcreteButRunsWalkDisallowed(t *testing.T) {
	k := kernel.New()
	// Closed schema requiring two fields. Stray field is disallowed.
	schema := k.CueContext().CompileString(`close({ replicas: int & >0, name: string })`)
	require.NoError(t, schema.Err())

	// Single source with only `replicas` set + a stray field.
	src, err := k.LoadSourceFromString("draft.cue", "draft", `{replicas: 1, stray: "x"}`)
	require.NoError(t, err)

	// Without Partial: missing `name` AND stray field both fail.
	_, fullErr := k.ValidateConfigDetailed(schema, []kernel.Source{src})
	require.Error(t, fullErr, "concrete check fails on missing required field")

	// With Partial: missing `name` ignored; stray STILL surfaces (walkDisallowed).
	_, partErr := k.ValidateConfigDetailed(schema, []kernel.Source{src}, kernel.Partial())
	require.Error(t, partErr, "Partial() does NOT silence walkDisallowed disallowed-field errors")

	// Verify the partial error is specifically about the stray field, not missing-name.
	gotStrayMessage := false
	for _, ce := range cueerrors.Errors(partErr) {
		f, _ := ce.Msg()
		if f == "field not allowed" {
			gotStrayMessage = true
		}
	}
	assert.True(t, gotStrayMessage, "Partial mode error MUST include 'field not allowed' for stray field")
}
