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
