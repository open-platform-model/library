package module_test

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/schema"
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

	mod, err := module.NewModuleFromValue(k, v)
	require.NoError(t, err)
	require.NotNil(t, mod)

	require.NotNil(t, mod.Metadata)
	assert.Equal(t, "demo-mod", mod.Metadata.Name)
	assert.Equal(t, "example.com/m/demo-mod:1.0.0", mod.Metadata.FQN)
	assert.True(t, mod.Package.Equals(v), "Package set unchanged from input")
}

func TestNewModuleFromValue_MissingMetadata(t *testing.T) {
	k := kernel.New()
	v := k.CueContext().CompileString(`kind: "Module"`)
	require.NoError(t, v.Err())

	mod, err := module.NewModuleFromValue(k, v)
	require.Error(t, err)
	assert.Nil(t, mod)
	assert.Contains(t, err.Error(), "metadata field is required")
}

func TestNewInstanceFromValue_SuccessPath(t *testing.T) {
	k := kernel.New()
	v := k.CueContext().CompileString(`
kind: "ModuleInstance"
metadata: {
	name: "demo"
	namespace: "ns"
	uuid: "11111111-1111-1111-1111-111111111111"
	labels: app: "x"
}
#module: {
	kind: "Module"
	metadata: {
		name: "demo-mod"
		modulePath: "example.com/m"
		version: "1.0.0"
		fqn: "example.com/m/demo-mod:1.0.0"
		uuid: "22222222-2222-2222-2222-222222222222"
	}
}
`)
	require.NoError(t, v.Err())

	inst, err := module.NewInstanceFromValue(k, v)
	require.NoError(t, err)
	require.NotNil(t, inst)

	require.NotNil(t, inst.Metadata)
	assert.Equal(t, "demo", inst.Metadata.Name)
	assert.Equal(t, "ns", inst.Metadata.Namespace)
	assert.True(t, inst.Package.Equals(v), "Package set unchanged from input")

	// Spec scenario: the instance's referenced module is reachable via
	// Package.LookupPath(schema.Module).
	moduleRef := inst.Package.LookupPath(schema.Module)
	require.True(t, moduleRef.Exists(), "instance's #module reference must be reachable via schema.Module")
	moduleName, err := moduleRef.LookupPath(schema.Metadata).LookupPath(cue.ParsePath("name")).String()
	require.NoError(t, err)
	assert.Equal(t, "demo-mod", moduleName)
}

func TestNewInstanceFromValue_MissingMetadata(t *testing.T) {
	k := kernel.New()
	v := k.CueContext().CompileString(`kind: "ModuleInstance"`)
	require.NoError(t, v.Err())

	inst, err := module.NewInstanceFromValue(k, v)
	require.Error(t, err)
	assert.Nil(t, inst)
	assert.Contains(t, err.Error(), "metadata field is required")
}

// TestKernelWrapper_NewModuleFromValue confirms the kernel wrapper produces
// the same result as the free constructor — the wrapper is the user-facing
// entry point per the unified-artifact-shape design.
func TestKernelWrapper_NewModuleFromValue(t *testing.T) {
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

	got, err := k.NewModuleFromValue(v)
	require.NoError(t, err)
	want, err := module.NewModuleFromValue(k, v)
	require.NoError(t, err)
	assert.Equal(t, want.Metadata.Name, got.Metadata.Name)
}

// TestInstance_ModuleFQNFromPackage confirms ModuleFQN reads through the
// schema's ModuleMetadataPath on Package rather than a cached struct field.
func TestInstance_ModuleFQNFromPackage(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString(`
kind: "ModuleInstance"
metadata: { name: "demo", namespace: "ns", uuid: "u" }
#moduleMetadata: {
	name: "demo-mod"
	modulePath: "example.com/m"
	version: "1.2.3"
	fqn: "example.com/m/demo-mod:1.2.3"
	uuid: "mm"
}
`)
	require.NoError(t, v.Err())

	inst := &module.Instance{
		Metadata: &module.InstanceMetadata{Name: "demo", Namespace: "ns"},
		Package:  v,
	}
	assert.Equal(t, "example.com/m/demo-mod:1.2.3", inst.ModuleFQN())
	assert.Equal(t, "1.2.3", inst.ModuleVersion())
}
