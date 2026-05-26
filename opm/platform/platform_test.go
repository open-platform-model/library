package platform_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/kernel"
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

	p, err := platform.NewPlatformFromValue(k, v)
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

	p, err := platform.NewPlatformFromValue(k, v)
	require.Error(t, err)
	assert.Nil(t, p)
	assert.Contains(t, err.Error(), "platform metadata field is required")
}

// TestKernelWrapper_NewPlatformFromValue confirms the kernel wrapper produces
// the same result as the free constructor.
func TestKernelWrapper_NewPlatformFromValue(t *testing.T) {
	k := kernel.New()
	v := k.CueContext().CompileString(`
kind: "Platform"
metadata: name: "wrapper"
type: "kubernetes"
`)
	require.NoError(t, v.Err())

	got, err := k.NewPlatformFromValue(v)
	require.NoError(t, err)
	want, err := platform.NewPlatformFromValue(k, v)
	require.NoError(t, err)
	assert.Equal(t, want.Metadata.Name, got.Metadata.Name)
	assert.Equal(t, want.Metadata.Type, got.Metadata.Type)
}
