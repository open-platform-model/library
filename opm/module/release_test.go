package module_test

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/module"
)

func TestRelease_ConfigSchema_Reachable(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString(`
kind: "ModuleRelease"
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

	rel := &module.Release{
		Metadata: &module.ReleaseMetadata{Name: "demo", Namespace: "ns"},
		Package:  v,
	}

	cfg := rel.ConfigSchema()
	require.True(t, cfg.Exists(), "ConfigSchema must resolve on a release whose #module carries #config")

	replicas := cfg.LookupPath(cue.ParsePath("replicas"))
	assert.True(t, replicas.Exists(), "ConfigSchema returned the #config subtree (replicas field reachable)")
}

func TestRelease_ConfigSchema_MissingConfigPath(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString(`
kind: "ModuleRelease"
metadata: { name: "demo", namespace: "ns", uuid: "u" }
#module: {
	metadata: { name: "demo-mod" }
}
`)
	require.NoError(t, v.Err())

	rel := &module.Release{
		Metadata: &module.ReleaseMetadata{Name: "demo"},
		Package:  v,
	}

	assert.False(t, rel.ConfigSchema().Exists(), "missing #config path must yield zero value")
}

func TestRelease_ConfigSchema_NilReceiver(t *testing.T) {
	var rel *module.Release
	assert.NotPanics(t, func() {
		assert.False(t, rel.ConfigSchema().Exists())
	})
}
