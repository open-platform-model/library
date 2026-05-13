package module_test

import (
	"errors"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/api"
	_ "github.com/open-platform-model/library/opm/api/v1alpha2"
	"github.com/open-platform-model/library/opm/apiversion"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/module"
)

func TestNewModuleFromValue_SuccessPath(t *testing.T) {
	k := kernel.New()
	v := k.CueContext().CompileString(`
apiVersion: "opmodel.dev/v1alpha2"
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

	assert.Equal(t, apiversion.V1alpha2, mod.APIVersion, "APIVersion stamped from Package")
	require.NotNil(t, mod.Metadata)
	assert.Equal(t, "demo-mod", mod.Metadata.Name)
	assert.Equal(t, "example.com/m/demo-mod:1.0.0", mod.Metadata.FQN)
	assert.True(t, mod.Package.Equals(v), "Package set unchanged from input")
}

func TestNewModuleFromValue_UnknownAPIVersion(t *testing.T) {
	k := kernel.New()
	v := k.CueContext().CompileString(`
apiVersion: "opmodel.dev/v9beta42"
kind: "Module"
metadata: { name: "x", modulePath: "y", version: "1", fqn: "y/x:1", uuid: "u" }
`)
	require.NoError(t, v.Err())

	mod, err := module.NewModuleFromValue(k, v)
	require.Error(t, err)
	assert.Nil(t, mod, "no partial module on detection failure")
	assert.True(t, errors.Is(err, apiversion.ErrUnknownAPIVersion))
}

func TestNewModuleFromValue_MissingAPIVersion(t *testing.T) {
	k := kernel.New()
	v := k.CueContext().CompileString(`metadata: name: "demo"`)
	require.NoError(t, v.Err())

	mod, err := module.NewModuleFromValue(k, v)
	require.Error(t, err)
	assert.Nil(t, mod)
	assert.True(t, errors.Is(err, apiversion.ErrUnknownAPIVersion))
}

func TestNewReleaseFromValue_SuccessPath(t *testing.T) {
	k := kernel.New()
	v := k.CueContext().CompileString(`
apiVersion: "opmodel.dev/v1alpha2"
kind: "ModuleRelease"
metadata: {
	name: "demo"
	namespace: "ns"
	uuid: "11111111-1111-1111-1111-111111111111"
	labels: app: "x"
}
#module: {
	apiVersion: "opmodel.dev/v1alpha2"
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

	rel, err := module.NewReleaseFromValue(k, v)
	require.NoError(t, err)
	require.NotNil(t, rel)

	assert.Equal(t, apiversion.V1alpha2, rel.APIVersion)
	require.NotNil(t, rel.Metadata)
	assert.Equal(t, "demo", rel.Metadata.Name)
	assert.Equal(t, "ns", rel.Metadata.Namespace)
	assert.True(t, rel.Package.Equals(v), "Package set unchanged from input")

	// Spec scenario: the release's referenced module is reachable via
	// Package.LookupPath(binding.Paths().Module).
	b, err := api.Lookup(apiversion.V1alpha2)
	require.NoError(t, err)
	moduleRef := rel.Package.LookupPath(b.Paths().Module)
	require.True(t, moduleRef.Exists(), "release's #module reference must be reachable via Paths().Module")
	moduleName, err := moduleRef.LookupPath(b.Paths().Metadata).LookupPath(cue.ParsePath("name")).String()
	require.NoError(t, err)
	assert.Equal(t, "demo-mod", moduleName)
}

func TestNewReleaseFromValue_UnknownAPIVersion(t *testing.T) {
	k := kernel.New()
	v := k.CueContext().CompileString(`
apiVersion: "opmodel.dev/never-registered"
kind: "ModuleRelease"
metadata: { name: "demo", namespace: "ns" }
`)
	require.NoError(t, v.Err())

	rel, err := module.NewReleaseFromValue(k, v)
	require.Error(t, err)
	assert.Nil(t, rel)
	assert.True(t, errors.Is(err, apiversion.ErrUnknownAPIVersion))
}

// TestKernelWrapper_NewModuleFromValue confirms the kernel wrapper produces
// the same result as the free constructor — the wrapper is the user-facing
// entry point per the unified-artifact-shape design.
func TestKernelWrapper_NewModuleFromValue(t *testing.T) {
	k := kernel.New()
	v := k.CueContext().CompileString(`
apiVersion: "opmodel.dev/v1alpha2"
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
	assert.Equal(t, want.APIVersion, got.APIVersion)
	assert.Equal(t, want.Metadata.Name, got.Metadata.Name)
}

// TestRelease_ModuleFQNFromPackage confirms ModuleFQN reads through the
// binding's ModuleMetadata path on Package rather than a cached struct field.
func TestRelease_ModuleFQNFromPackage(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString(`
apiVersion: "opmodel.dev/v1alpha2"
kind: "ModuleRelease"
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

	rel := &module.Release{
		APIVersion: apiversion.V1alpha2,
		Metadata:   &module.ReleaseMetadata{Name: "demo", Namespace: "ns"},
		Package:    v,
	}
	assert.Equal(t, "example.com/m/demo-mod:1.2.3", rel.ModuleFQN())
	assert.Equal(t, "1.2.3", rel.ModuleVersion())
}
