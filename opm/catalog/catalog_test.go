package catalog_test

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/catalog"
)

func TestNewCatalogFromValue(t *testing.T) {
	v := cuecontext.New().CompileString(`kind: "Catalog"
metadata: {
	modulePath:  "test.example/catalogs/provider@v1"
	version:     "1.0.0"
	fqn:         "test.example/catalogs/provider@v1"
	description: "a test catalog"
	labels: "opm.test/tier": "provider"
}`)
	require.NoError(t, v.Err())

	c, err := catalog.NewCatalogFromValue(v)
	require.NoError(t, err)
	require.NotNil(t, c.Metadata)
	assert.Equal(t, "test.example/catalogs/provider@v1", c.Metadata.ModulePath)
	assert.Equal(t, "1.0.0", c.Metadata.Version)
	assert.Equal(t, "test.example/catalogs/provider@v1", c.Metadata.FQN)
	assert.Equal(t, "a test catalog", c.Metadata.Description)
	assert.Equal(t, map[string]string{"opm.test/tier": "provider"}, c.Metadata.Labels)
	assert.True(t, c.Package.Equals(v), "Package stores the input value unmodified")
	assert.Nil(t, c.Source, "a catalog built from a bare value carries no source")
}

// Errors return a nil *Catalog: partial values are never returned, matching
// module.NewModuleFromValue and platform.NewPlatformFromValue.
func TestNewCatalogFromValue_MissingMetadata(t *testing.T) {
	v := cuecontext.New().CompileString(`kind: "Catalog"`)
	require.NoError(t, v.Err())

	c, err := catalog.NewCatalogFromValue(v)
	require.Error(t, err)
	assert.Nil(t, c)
}
