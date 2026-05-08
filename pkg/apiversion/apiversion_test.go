package apiversion_test

import (
	"errors"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/pkg/apiversion"
)

func TestDetect_RecognisedVersion(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString(`apiVersion: "opmodel.dev/v1alpha2"`)
	require.NoError(t, v.Err())

	got, err := apiversion.Detect(v)
	require.NoError(t, err)
	assert.Equal(t, apiversion.V1alpha2, got)
}

func TestDetect_UnknownLiteral(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString(`apiVersion: "opmodel.dev/v9beta42"`)
	require.NoError(t, v.Err())

	got, err := apiversion.Detect(v)
	assert.Equal(t, apiversion.Version(""), got)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apiversion.ErrUnknownAPIVersion), "want ErrUnknownAPIVersion, got %v", err)
}

func TestDetect_MissingField(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString(`kind: "Module"`)
	require.NoError(t, v.Err())

	got, err := apiversion.Detect(v)
	assert.Equal(t, apiversion.Version(""), got)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apiversion.ErrUnknownAPIVersion), "want ErrUnknownAPIVersion, got %v", err)
}

func TestDetect_NonStringField(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString(`apiVersion: 42`)
	require.NoError(t, v.Err())

	got, err := apiversion.Detect(v)
	assert.Equal(t, apiversion.Version(""), got)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apiversion.ErrUnknownAPIVersion), "want ErrUnknownAPIVersion, got %v", err)
}
