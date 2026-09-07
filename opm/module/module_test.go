package module_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/module"
)

func TestNewModuleFromValue_SuccessPath(t *testing.T) {
	k := kernel.New()
	v := k.CueContext().CompileString(`
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
	k := kernel.New()
	v := k.CueContext().CompileString(`kind: "Module"`)
	require.NoError(t, v.Err())

	mod, err := module.NewModuleFromValue(v)
	require.Error(t, err)
	assert.Nil(t, mod)
	assert.Contains(t, err.Error(), "metadata field is required")
}

// artifact-types spec, "Constructor Helpers from cue.Value": a module built
// from a bare value carries no Source, and the kernel exposes no wrapper for
// the constructor — a frontend holding a value calls the package constructor
// directly, and one wanting a source-carrying module uses an acquire verb.
func TestNewModuleFromValue_NoSourceAndNoKernelWrapper(t *testing.T) {
	k := kernel.New()
	v := k.CueContext().CompileString(`
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

	_, found := reflect.TypeOf(k).MethodByName("NewModuleFromValue")
	assert.False(t, found, "*kernel.Kernel must not wrap module.NewModuleFromValue")
}
