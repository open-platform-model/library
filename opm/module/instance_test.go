package module_test

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/schema"
)

func TestInstance_ConfigSchema_Reachable(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString(`
kind: "ModuleInstance"
metadata: { name: "demo", namespace: "ns", uuid: "u" }
#module: {
	kind: "Module"
	metadata: {
		name: "demo-mod"
		modulePath: "example.com/m"
		version: "1.0.0"
		fqn: "example.com/m/demo-mod:1.0.0"
		uuid: "11111111-1111-1111-1111-111111111111"
	}
	#config: {
		replicas: int & >0
		name: string
	}
}
`)
	require.NoError(t, v.Err())

	inst := &module.Instance{
		Metadata: &module.InstanceMetadata{Name: "demo", Namespace: "ns"},
		Package:  v,
	}

	cfg := inst.ConfigSchema()
	require.True(t, cfg.Exists(), "ConfigSchema must resolve on an instance whose #module carries #config")

	replicas := cfg.LookupPath(cue.ParsePath("replicas"))
	assert.True(t, replicas.Exists(), "ConfigSchema returned the #config subtree (replicas field reachable)")
}

func TestInstance_ConfigSchema_MissingConfigPath(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString(`
kind: "ModuleInstance"
metadata: { name: "demo", namespace: "ns", uuid: "u" }
#module: {
	metadata: { name: "demo-mod" }
}
`)
	require.NoError(t, v.Err())

	inst := &module.Instance{
		Metadata: &module.InstanceMetadata{Name: "demo"},
		Package:  v,
	}

	assert.False(t, inst.ConfigSchema().Exists(), "missing #config path must yield zero value")
}

func TestInstance_ConfigSchema_NilReceiver(t *testing.T) {
	var inst *module.Instance
	assert.NotPanics(t, func() {
		assert.False(t, inst.ConfigSchema().Exists())
	})
}

const instanceWithModule = `
kind: "ModuleInstance"
metadata: { name: "demo", namespace: "ns", uuid: "u" }
#module: {
	kind: "Module"
	metadata: {
		name: "demo-mod"
		modulePath: "example.com/m"
		version: "1.0.0"
		fqn: "example.com/m/demo-mod:1.0.0"
		uuid: "11111111-1111-1111-1111-111111111111"
	}
}
values: { replicas: 3, image: "nginx:1.27" }
`

// artifact-types, "Module metadata reads the package": a struct literal
// whose Package carries an embedded #module, with no processing step run.
func TestInstance_ModuleMetadata_WellFormed(t *testing.T) {
	v := cuecontext.New().CompileString(instanceWithModule)
	require.NoError(t, v.Err())
	inst := &module.Instance{Package: v}

	meta := inst.ModuleMetadata()
	require.NotNil(t, meta)
	assert.Equal(t, "demo-mod", meta.Name)
	assert.Equal(t, "example.com/m", meta.ModulePath)
	assert.Equal(t, "1.0.0", meta.Version)
	assert.Equal(t, "example.com/m/demo-mod:1.0.0", meta.FQN)
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", meta.UUID)
}

// artifact-types, "Module metadata is nil when it cannot be read".
func TestInstance_ModuleMetadata_NilWhenUnreadable(t *testing.T) {
	ctx := cuecontext.New()
	noModule := ctx.CompileString(`kind: "ModuleInstance", metadata: name: "demo"`)
	require.NoError(t, noModule.Err())
	assert.Nil(t, (&module.Instance{Package: noModule}).ModuleMetadata(), "no #module")

	undecodable := ctx.CompileString(`#module: metadata: { name: 5, version: "1.0.0" }`)
	require.NoError(t, undecodable.Err())
	assert.Nil(t, (&module.Instance{Package: undecodable}).ModuleMetadata(), "metadata that does not decode, no partial fill")

	var inst *module.Instance
	assert.NotPanics(t, func() { assert.Nil(t, inst.ModuleMetadata()) }, "nil receiver")
}

// artifact-types, "Values accessor" and "Values on a nil instance is the
// zero value".
func TestInstance_Values(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString(instanceWithModule)
	require.NoError(t, v.Err())
	inst := &module.Instance{Package: v}

	values := inst.Values()
	require.True(t, values.Exists())
	assert.True(t, values.Equals(v.LookupPath(schema.Values)), "Values is the schema.Values subtree")
	replicas, err := values.LookupPath(cue.ParsePath("replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(3), replicas)

	noValues := ctx.CompileString(`kind: "ModuleInstance", metadata: name: "demo"`)
	require.NoError(t, noValues.Err())
	assert.False(t, (&module.Instance{Package: noValues}).Values().Exists(), "no values field")

	var nilInst *module.Instance
	assert.NotPanics(t, func() { assert.False(t, nilInst.Values().Exists()) })
}
