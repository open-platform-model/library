package module_test

import (
	"strings"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/schema"
)

func TestNewModuleFromValue_SuccessPath(t *testing.T) {
	v := cuecontext.New().CompileString(`
kind: "Module"
metadata: {
	name: "demo-mod"
	modulePath: "example.com/m"
	version: "1.0.0"
	fqn: "example.com/m/demo-mod:1.0.0"
	uuid: "11111111-1111-1111-1111-111111111111"
}
`)
	require.NoError(t, v.Err())

	mod, err := module.NewModuleFromValue(v)
	require.NoError(t, err)
	require.NotNil(t, mod)

	require.NotNil(t, mod.Metadata)
	assert.Equal(t, "demo-mod", mod.Metadata.Name)
	assert.Equal(t, "example.com/m/demo-mod:1.0.0", mod.Metadata.FQN)
	assert.True(t, mod.Package.Equals(v), "Package set unchanged from input")
	assert.Nil(t, mod.Source, "value-constructed module must carry no Source")
	assert.False(t, mod.HasSource())
}

func TestNewModuleFromValue_MissingMetadata(t *testing.T) {
	v := cuecontext.New().CompileString(`kind: "Module"`)
	require.NoError(t, v.Err())

	mod, err := module.NewModuleFromValue(v)
	require.Error(t, err)
	assert.Nil(t, mod)
	assert.Contains(t, err.Error(), "metadata field is required")
}

// artifact-types spec, "Constructor Helpers from cue.Value": a module built
// from a bare value carries no Source. That the kernel exposes no wrapper for
// the constructor is pinned by TestKernel_ExportedSurface in opm/kernel: a
// frontend holding a value calls the package constructor directly, and one
// wanting a source-carrying module uses an acquire verb.
func TestNewModuleFromValue_NoSource(t *testing.T) {
	v := cuecontext.New().CompileString(`
kind: "Module"
metadata: {
	name: "demo-mod"
	modulePath: "example.com/m"
	version: "1.0.0"
	fqn: "example.com/m/demo-mod:1.0.0"
	uuid: "11111111-1111-1111-1111-111111111111"
}
`)
	require.NoError(t, v.Err())

	mod, err := module.NewModuleFromValue(v)
	require.NoError(t, err)
	assert.Equal(t, "demo-mod", mod.Metadata.Name)
	assert.Nil(t, mod.Source, "a value-constructed module carries no staged source")
	assert.False(t, mod.HasSource())
}

// TestNewModuleFromValue_DecodeFailureNamedOnce pins the schema-dispatch
// scenario "A decode failure names its artifact once": the constructor
// returns the decoder's error without a second prefix.
func TestNewModuleFromValue_DecodeFailureNamedOnce(t *testing.T) {
	v := cuecontext.New().CompileString(`kind: "Module", metadata: name: 1`)
	require.NoError(t, v.Err())

	mod, err := module.NewModuleFromValue(v)
	require.Error(t, err)
	assert.Nil(t, mod)
	assert.Equal(t, 1, strings.Count(err.Error(), "decoding module metadata:"), err.Error())
}

// artifact-types, "Debug values present" and "Debug values absent".
func TestModule_DebugValues(t *testing.T) {
	ctx := cuecontext.New()
	with := ctx.CompileString(`
kind: "Module"
metadata: name: "demo-mod"
debugValues: replicas: 1
`)
	require.NoError(t, with.Err())
	mod, err := module.NewModuleFromValue(with)
	require.NoError(t, err)
	dv := mod.DebugValues()
	require.True(t, dv.Exists())
	assert.True(t, dv.Equals(with.LookupPath(schema.DebugValues)))

	without := ctx.CompileString(`
kind: "Module"
metadata: name: "demo-mod"
`)
	require.NoError(t, without.Err())
	mod, err = module.NewModuleFromValue(without)
	require.NoError(t, err)
	assert.False(t, mod.DebugValues().Exists(), "a module that declares none")

	var nilMod *module.Module
	assert.NotPanics(t, func() { assert.False(t, nilMod.DebugValues().Exists()) })
}
