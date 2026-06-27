package module_test

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/module"
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
