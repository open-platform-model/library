package platform_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/open-platform-model/library/pkg/api/v1alpha2"
	"github.com/open-platform-model/library/pkg/apiversion"
	"github.com/open-platform-model/library/pkg/kernel"
	"github.com/open-platform-model/library/pkg/platform"
)

func TestNewPlatformFromValue_SuccessPath(t *testing.T) {
	k := kernel.New()
	v := k.CueContext().CompileString(`
apiVersion: "opmodel.dev/v1alpha2"
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

	assert.Equal(t, apiversion.V1alpha2, p.APIVersion, "APIVersion stamped from Package")
	require.NotNil(t, p.Metadata)
	assert.Equal(t, "demo-platform", p.Metadata.Name)
	assert.Equal(t, "kubernetes", p.Metadata.Type)
	assert.Equal(t, "demo", p.Metadata.Description)
	assert.Equal(t, map[string]string{"env": "dev"}, p.Metadata.Labels)
	assert.Equal(t, map[string]string{"owner": "team"}, p.Metadata.Annotations)
	assert.True(t, p.Package.Equals(v), "Package set unchanged from input")
}

func TestNewPlatformFromValue_UnknownAPIVersion(t *testing.T) {
	k := kernel.New()
	v := k.CueContext().CompileString(`
apiVersion: "opmodel.dev/v9beta42"
kind: "Platform"
metadata: name: "demo"
type: "kubernetes"
`)
	require.NoError(t, v.Err())

	p, err := platform.NewPlatformFromValue(k, v)
	require.Error(t, err)
	assert.Nil(t, p, "no partial platform on detection failure")
	assert.True(t, errors.Is(err, apiversion.ErrUnknownAPIVersion))
}

func TestNewPlatformFromValue_MissingAPIVersion(t *testing.T) {
	k := kernel.New()
	v := k.CueContext().CompileString(`metadata: name: "demo"`)
	require.NoError(t, v.Err())

	p, err := platform.NewPlatformFromValue(k, v)
	require.Error(t, err)
	assert.Nil(t, p)
	assert.True(t, errors.Is(err, apiversion.ErrUnknownAPIVersion))
}

// TestNewPlatformFromValue_MissingMetadata exercises the malformed-metadata
// path: the v1alpha2 decoder treats an absent metadata field as fatal.
func TestNewPlatformFromValue_MissingMetadata(t *testing.T) {
	k := kernel.New()
	v := k.CueContext().CompileString(`
apiVersion: "opmodel.dev/v1alpha2"
kind: "Platform"
type: "kubernetes"
`)
	require.NoError(t, v.Err())

	p, err := platform.NewPlatformFromValue(k, v)
	require.Error(t, err)
	assert.Nil(t, p)
	// Error chain: NewPlatformFromValue → decoder-side message.
	assert.Contains(t, err.Error(), "platform metadata field is required")
}

// TestKernelWrapper_NewPlatformFromValue confirms the kernel wrapper produces
// the same result as the free constructor.
func TestKernelWrapper_NewPlatformFromValue(t *testing.T) {
	k := kernel.New()
	v := k.CueContext().CompileString(`
apiVersion: "opmodel.dev/v1alpha2"
kind: "Platform"
metadata: name: "wrapper"
type: "kubernetes"
`)
	require.NoError(t, v.Err())

	got, err := k.NewPlatformFromValue(v)
	require.NoError(t, err)
	want, err := platform.NewPlatformFromValue(k, v)
	require.NoError(t, err)
	assert.Equal(t, want.APIVersion, got.APIVersion)
	assert.Equal(t, want.Metadata.Name, got.Metadata.Name)
	assert.Equal(t, want.Metadata.Type, got.Metadata.Type)
}
