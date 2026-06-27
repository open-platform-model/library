package kernel_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/kernel"
)

func TestKernel_ValidateModuleValues_OK(t *testing.T) {
	k := kernel.New()
	f := newPhaseFixture(t, k)
	values := k.CueContext().CompileString(`{ replicas: 3, name: "demo" }`)
	require.NoError(t, values.Err())

	merged, err := k.ValidateModuleValues(f.mod, values)
	require.NoError(t, err)
	assert.True(t, merged.Exists())
}

func TestKernel_ValidateModuleValues_Error(t *testing.T) {
	k := kernel.New()
	f := newPhaseFixture(t, k)
	bad := k.CueContext().CompileString(`{ replicas: -1, name: "demo" }`)
	require.NoError(t, bad.Err())

	_, err := k.ValidateModuleValues(f.mod, bad)
	require.Error(t, err)
}

func TestKernel_ValidateModuleValuesPartial_AllowsMissingFields(t *testing.T) {
	k := kernel.New()
	f := newPhaseFixture(t, k)
	// Only replicas set; name missing — partial mode allows it.
	partial := k.CueContext().CompileString(`{ replicas: 3 }`)
	require.NoError(t, partial.Err())

	_, err := k.ValidateModuleValuesPartial(f.mod, partial)
	require.NoError(t, err, "partial MUST allow missing required fields")

	// Concrete check via the non-partial counterpart MUST fail.
	_, fullErr := k.ValidateModuleValues(f.mod, partial)
	require.Error(t, fullErr, "concrete check MUST flag missing required field")
}

func TestKernel_ValidateModuleValuesDetailed_LayeredSuccess(t *testing.T) {
	k := kernel.New()
	f := newPhaseFixture(t, k)

	a, err := k.LoadSourceFromString("defaults.cue", "defaults", `replicas: 1`)
	require.NoError(t, err)
	b, err := k.LoadSourceFromString("user.cue", "user", `name: "prod"`)
	require.NoError(t, err)

	merged, vErr := k.ValidateModuleValuesDetailed(f.mod, []kernel.Source{a, b})
	require.NoError(t, vErr)
	assert.True(t, merged.Exists())
}

func TestKernel_ValidateInstanceValues_OK(t *testing.T) {
	k := kernel.New()
	f := newPhaseFixture(t, k)
	values := k.CueContext().CompileString(`{ replicas: 3, name: "demo" }`)
	require.NoError(t, values.Err())

	merged, err := k.ValidateInstanceValues(f.inst, values)
	require.NoError(t, err)
	assert.True(t, merged.Exists())
}

func TestKernel_ValidateInstanceValuesPartial_AllowsMissingFields(t *testing.T) {
	k := kernel.New()
	f := newPhaseFixture(t, k)
	partial := k.CueContext().CompileString(`{ replicas: 3 }`)
	require.NoError(t, partial.Err())

	_, err := k.ValidateInstanceValuesPartial(f.inst, partial)
	require.NoError(t, err)
}

func TestKernel_ValidateInstanceValuesDetailed_LayeredSuccess(t *testing.T) {
	k := kernel.New()
	f := newPhaseFixture(t, k)

	a, err := k.LoadSourceFromString("a.cue", "a", `replicas: 2`)
	require.NoError(t, err)
	b, err := k.LoadSourceFromString("b.cue", "b", `name: "inst"`)
	require.NoError(t, err)

	merged, vErr := k.ValidateInstanceValuesDetailed(f.inst, []kernel.Source{a, b}, kernel.Partial())
	require.NoError(t, vErr)
	assert.True(t, merged.Exists())
}
