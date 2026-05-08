package validate_test

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/pkg/validate"
)

func TestConfig_ZeroValueAcceptedAsNoValues(t *testing.T) {
	ctx := cuecontext.New()
	schema := ctx.CompileString(`{ replicas: int & >0 }`)
	require.NoError(t, schema.Err())

	got, err := validate.Config(schema, cue.Value{}, "module", "demo") //nolint:staticcheck // SA1019: exercising the function under test
	require.Nil(t, err, "zero cue.Value MUST be treated as 'no values' and succeed")
	assert.False(t, got.Exists(), "no values supplied -> returned cue.Value is the zero value")
}

func TestConfig_SchemaErrorReturnedAsConfigError(t *testing.T) {
	ctx := cuecontext.New()
	schema := ctx.CompileString(`{ replicas: int & >0 }`)
	require.NoError(t, schema.Err())
	bad := ctx.CompileString(`{ replicas: -1 }`)
	require.NoError(t, bad.Err())

	_, cfgErr := validate.Config(schema, bad, "module", "demo") //nolint:staticcheck // SA1019: exercising the function under test
	require.NotNil(t, cfgErr)
	assert.Equal(t, "module", cfgErr.Context)
	assert.Equal(t, "demo", cfgErr.Name)
}

func TestConfig_FieldNotAllowedReportedViaWalk(t *testing.T) {
	ctx := cuecontext.New()
	schema := ctx.CompileString(`close({ replicas: int })`)
	require.NoError(t, schema.Err())
	stray := ctx.CompileString(`{ replicas: 1, stray: "x" }`)
	require.NoError(t, stray.Err())

	_, cfgErr := validate.Config(schema, stray, "module", "demo") //nolint:staticcheck // SA1019: exercising the function under test
	require.NotNil(t, cfgErr, "field outside the closed schema MUST surface")
}

func TestUnifyAndValidate_EmptySliceProducesZeroValue(t *testing.T) {
	got := validate.UnifyAndValidate(nil) //nolint:staticcheck // SA1019: exercising the helper under test
	assert.False(t, got.Exists(), "empty slice MUST return cue.Value{}")
}

func TestUnifyAndValidate_SingleValueRoundTrips(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString(`{ replicas: 3 }`)
	require.NoError(t, v.Err())

	got := validate.UnifyAndValidate([]cue.Value{v}) //nolint:staticcheck // SA1019: exercising the helper under test
	require.True(t, got.Exists())
	r, err := got.LookupPath(cue.ParsePath("replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(3), r)
}

// TestUnifyAndValidate_ParityWithPreviousLoop confirms UnifyAndValidate
// produces the same unified value the previous slice-form Config would have
// produced internally before the slice 04 signature change.
func TestUnifyAndValidate_ParityWithPreviousLoop(t *testing.T) {
	ctx := cuecontext.New()
	a := ctx.CompileString(`{ replicas: 3, name: "demo" }`)
	require.NoError(t, a.Err())
	b := ctx.CompileString(`{ replicas: 3, image: "nginx" }`)
	require.NoError(t, b.Err())
	c := ctx.CompileString(`{ replicas: 3, name: "demo", image: "nginx", env: "prod" }`)
	require.NoError(t, c.Err())

	// Reference: the merge loop the kernel previously performed inline.
	want := a
	for _, v := range []cue.Value{b, c} {
		want = want.Unify(v)
	}

	got := validate.UnifyAndValidate([]cue.Value{a, b, c}) //nolint:staticcheck // SA1019: exercising the helper under test
	require.True(t, got.Exists())

	gotName, _ := got.LookupPath(cue.ParsePath("name")).String()
	wantName, _ := want.LookupPath(cue.ParsePath("name")).String()
	assert.Equal(t, wantName, gotName)

	gotImage, _ := got.LookupPath(cue.ParsePath("image")).String()
	wantImage, _ := want.LookupPath(cue.ParsePath("image")).String()
	assert.Equal(t, wantImage, gotImage)
}

// TestConfig_ParityWithSliceFormViaUnifyAndValidate confirms the new single-
// value Config produces equivalent output to the old slice form bridged via
// UnifyAndValidate.
func TestConfig_ParityWithSliceFormViaUnifyAndValidate(t *testing.T) {
	ctx := cuecontext.New()
	schema := ctx.CompileString(`{ replicas: int & >0, name: string, env: string }`)
	require.NoError(t, schema.Err())

	v1 := ctx.CompileString(`{ replicas: 3, name: "demo" }`)
	v2 := ctx.CompileString(`{ replicas: 3, name: "demo", env: "prod" }`)
	require.NoError(t, v1.Err())
	require.NoError(t, v2.Err())

	merged := validate.UnifyAndValidate([]cue.Value{v1, v2})      //nolint:staticcheck // SA1019: exercising the helper under test
	got, err := validate.Config(schema, merged, "module", "demo") //nolint:staticcheck // SA1019: exercising the function under test
	require.Nil(t, err)
	require.True(t, got.Exists())

	name, lookupErr := got.LookupPath(cue.ParsePath("name")).String()
	require.NoError(t, lookupErr)
	assert.Equal(t, "demo", name)

	env, lookupErr := got.LookupPath(cue.ParsePath("env")).String()
	require.NoError(t, lookupErr)
	assert.Equal(t, "prod", env)
}
