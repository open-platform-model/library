package module_test

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/open-platform-model/library/opm/api/v1alpha2"
	"github.com/open-platform-model/library/opm/apiversion"
	"github.com/open-platform-model/library/opm/module"
)

func TestRelease_ConfigSchema_Reachable(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString(`
apiVersion: "opmodel.dev/v1alpha2"
kind: "ModuleRelease"
metadata: { name: "demo", namespace: "ns", uuid: "u" }
#module: {
	apiVersion: "opmodel.dev/v1alpha2"
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
		APIVersion: apiversion.V1alpha2,
		Metadata:   &module.ReleaseMetadata{Name: "demo", Namespace: "ns"},
		Package:    v,
	}

	schema := rel.ConfigSchema()
	require.True(t, schema.Exists(), "ConfigSchema must resolve on a release whose #module carries #config")

	replicas := schema.LookupPath(cue.ParsePath("replicas"))
	assert.True(t, replicas.Exists(), "ConfigSchema returned the #config subtree (replicas field reachable)")
}

func TestRelease_ConfigSchema_UnregisteredBinding(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString(`
apiVersion: "opmodel.dev/never-registered"
kind: "ModuleRelease"
metadata: { name: "demo", namespace: "ns", uuid: "u" }
#module: { #config: { replicas: int } }
`)
	require.NoError(t, v.Err())

	rel := &module.Release{
		APIVersion: apiversion.Version("opmodel.dev/never-registered"),
		Metadata:   &module.ReleaseMetadata{Name: "demo"},
		Package:    v,
	}

	assert.False(t, rel.ConfigSchema().Exists(), "unregistered binding must yield zero value, not error")
}

func TestRelease_ConfigSchema_MissingConfigPath(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString(`
apiVersion: "opmodel.dev/v1alpha2"
kind: "ModuleRelease"
metadata: { name: "demo", namespace: "ns", uuid: "u" }
#module: {
	apiVersion: "opmodel.dev/v1alpha2"
	metadata: { name: "demo-mod" }
}
`)
	require.NoError(t, v.Err())

	rel := &module.Release{
		APIVersion: apiversion.V1alpha2,
		Metadata:   &module.ReleaseMetadata{Name: "demo"},
		Package:    v,
	}

	assert.False(t, rel.ConfigSchema().Exists(), "missing #config path must yield zero value")
}

func TestRelease_ConfigSchema_NilReceiver(t *testing.T) {
	var rel *module.Release
	assert.NotPanics(t, func() {
		assert.False(t, rel.ConfigSchema().Exists())
	})
}
