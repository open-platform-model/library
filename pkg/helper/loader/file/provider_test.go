package file_test

import (
	"errors"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/pkg/apiversion"
	loader "github.com/open-platform-model/library/pkg/helper/loader/file"
)

func TestLoadProvider_PopulatesAPIVersion(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString(`
apiVersion: "opmodel.dev/v1alpha2"
kind: "Provider"
metadata: { name: "kubernetes", version: "v0" }
`)
	require.NoError(t, v.Err())

	got, err := loader.LoadProvider("kubernetes", map[string]cue.Value{"kubernetes": v})
	require.NoError(t, err)
	assert.Equal(t, apiversion.V1alpha2, got.APIVersion)
	assert.Equal(t, "kubernetes", got.Metadata.Name)
}

func TestLoadProvider_RejectsMissingAPIVersion(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString(`metadata: name: "kubernetes"`)
	require.NoError(t, v.Err())

	_, err := loader.LoadProvider("kubernetes", map[string]cue.Value{"kubernetes": v})
	require.Error(t, err)
	assert.True(t, errors.Is(err, apiversion.ErrUnknownAPIVersion))
}
