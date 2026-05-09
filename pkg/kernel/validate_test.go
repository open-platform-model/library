package kernel_test

import (
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/pkg/kernel"
)

func TestKernel_ValidateConfig_ZeroValueIsNoOp(t *testing.T) {
	k := kernel.New()
	schema := k.CueContext().CompileString(`{ replicas: int & >0 }`)
	require.NoError(t, schema.Err())

	got, err := k.ValidateConfig(schema, cue.Value{}, "module", "demo")
	require.Nil(t, err, "zero cue.Value MUST be treated as 'no values' and succeed")
	assert.False(t, got.Exists(), "no values supplied → returned cue.Value is the zero value")
}

func TestKernel_ValidateConfig_SchemaErrorReturnsConfigError(t *testing.T) {
	k := kernel.New()
	schema := k.CueContext().CompileString(`{ replicas: int & >0 }`)
	require.NoError(t, schema.Err())
	bad := k.CueContext().CompileString(`{ replicas: -1 }`)
	require.NoError(t, bad.Err())

	_, cfgErr := k.ValidateConfig(schema, bad, "module", "demo")
	require.NotNil(t, cfgErr)
	assert.Equal(t, "module", cfgErr.Context)
	assert.Equal(t, "demo", cfgErr.Name)
}

func TestKernel_ValidateConfig_FieldNotAllowed(t *testing.T) {
	k := kernel.New()
	schema := k.CueContext().CompileString(`close({ replicas: int })`)
	require.NoError(t, schema.Err())
	stray := k.CueContext().CompileString(`{ replicas: 1, stray: "x" }`)
	require.NoError(t, stray.Err())

	_, cfgErr := k.ValidateConfig(schema, stray, "module", "demo")
	require.NotNil(t, cfgErr, "field outside the closed schema MUST surface")
}

func TestKernel_ValidateConfigPartial_MissingFieldPassesPartialFailsFull(t *testing.T) {
	k := kernel.New()
	// Schema requires both `replicas` and `name` to be concrete.
	schema := k.CueContext().CompileString(`{ replicas: int & >0, name: string }`)
	require.NoError(t, schema.Err())
	// Partial value sets only `replicas`; `name` is missing.
	partial := k.CueContext().CompileString(`{ replicas: 3 }`)
	require.NoError(t, partial.Err())

	_, partialErr := k.ValidateConfigPartial(schema, partial, "values", "layer-a")
	require.Nil(t, partialErr, "partial validation MUST allow missing required fields")

	_, fullErr := k.ValidateConfig(schema, partial, "module", "demo")
	require.NotNil(t, fullErr, "full Tier-2 validation MUST flag missing required field")
}

func TestKernel_ValidateConfigPartial_TypeErrorStillSurfaces(t *testing.T) {
	k := kernel.New()
	schema := k.CueContext().CompileString(`{ replicas: int & >0, name: string }`)
	require.NoError(t, schema.Err())
	// `replicas` set but with the wrong type — partial validation MUST flag it.
	wrongType := k.CueContext().CompileString(`{ replicas: "three" }`)
	require.NoError(t, wrongType.Err())

	_, cfgErr := k.ValidateConfigPartial(schema, wrongType, "values", "layer-a")
	require.NotNil(t, cfgErr, "partial validation still flags type errors on fields that ARE set")
	assert.Equal(t, "values", cfgErr.Context)
	assert.Equal(t, "layer-a", cfgErr.Name)
}

func TestKernel_ValidateConfigPartial_ZeroValueIsNoOp(t *testing.T) {
	k := kernel.New()
	schema := k.CueContext().CompileString(`{ replicas: int & >0 }`)
	require.NoError(t, schema.Err())

	got, err := k.ValidateConfigPartial(schema, cue.Value{}, "values", "layer-a")
	require.Nil(t, err)
	assert.False(t, got.Exists())
}
