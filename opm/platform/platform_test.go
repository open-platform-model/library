package platform_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/platform"
)

func TestNewPlatformFromValue_SuccessPath(t *testing.T) {
	k := kernel.New()
	v := k.CueContext().CompileString(`
kind: "Platform"
metadata: {
	name: "demo-platform"
	description: "demo"
	labels: env: "dev"
	annotations: owner: "team"
}
type: "kubernetes"
`)
	require.NoError(t, v.Err())

	p, err := platform.NewPlatformFromValue(v)
	require.NoError(t, err)
	require.NotNil(t, p)

	require.NotNil(t, p.Metadata)
	assert.Equal(t, "demo-platform", p.Metadata.Name)
	assert.Equal(t, "kubernetes", p.Metadata.Type)
	assert.Equal(t, "demo", p.Metadata.Description)
	assert.Equal(t, map[string]string{"env": "dev"}, p.Metadata.Labels)
	assert.Equal(t, map[string]string{"owner": "team"}, p.Metadata.Annotations)
	assert.True(t, p.Package.Equals(v), "Package set unchanged from input")
}

// TestNewPlatformFromValue_MissingMetadata exercises the malformed-metadata
// path: the decoder treats an absent metadata field as fatal.
func TestNewPlatformFromValue_MissingMetadata(t *testing.T) {
	k := kernel.New()
	v := k.CueContext().CompileString(`
kind: "Platform"
type: "kubernetes"
`)
	require.NoError(t, v.Err())

	p, err := platform.NewPlatformFromValue(v)
	require.Error(t, err)
	assert.Nil(t, p)
	assert.Contains(t, err.Error(), "platform metadata field is required")
}

// artifact-types spec, "No kernel constructor wrappers": a frontend holding a
// value calls the package constructor directly; the kernel wraps neither
// constructor.
func TestNewPlatformFromValue_NoKernelWrapper(t *testing.T) {
	_, found := reflect.TypeOf(kernel.New()).MethodByName("NewPlatformFromValue")
	assert.False(t, found, "*kernel.Kernel must not wrap platform.NewPlatformFromValue")
}

// TestNewPlatformFromValue_NoSource pins the platform-artifact scenario
// "Value-constructed platform has no source": a platform built from a bare
// cue.Value carries no staged source tree.
func TestNewPlatformFromValue_NoSource(t *testing.T) {
	k := kernel.New()
	v := k.CueContext().CompileString(`
kind: "Platform"
metadata: name: "no-source"
type: "kubernetes"
`)
	require.NoError(t, v.Err())

	p, err := platform.NewPlatformFromValue(v)
	require.NoError(t, err)
	require.NotNil(t, p)
	assert.Nil(t, p.Source, "value-constructed platform must carry no Source")

	// platform.Source aliases module.Source: assigning one type to the other
	// needs no conversion.
	p.Source = &module.Source{Root: "/x"}
	var src *platform.Source = p.Source //nolint:staticcheck // the explicit type IS the assertion: alias identity
	assert.Equal(t, "/x", src.Root)
}
