package file_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loader "github.com/open-platform-model/library/opm/helper/loader/file"
	"github.com/open-platform-model/library/opm/kernel"
)

const platformFixture = `
package platform
kind: "Platform"
metadata: {
	name: "demo-platform"
}
type: "kubernetes"
`

// writeTempPlatformDir writes content to a fresh temp dir as platform.cue and
// returns the dir path. LoadPlatformPackage operates on the directory itself.
func writeTempPlatformDir(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "platform.cue"), []byte(content), 0o644))
	return dir
}

func TestLoadPlatformPackage_Loads(t *testing.T) {
	dir := writeTempPlatformDir(t, platformFixture)

	val, err := loader.LoadPlatformPackage(cuecontext.New(), dir, loader.LoadOptions{})
	require.NoError(t, err)
	assert.True(t, val.Exists())

	name, err := val.LookupPath(cue.ParsePath("metadata.name")).String()
	require.NoError(t, err)
	assert.Equal(t, "demo-platform", name)
}

// TestLoadPlatformPackage_RegistryOptionAccepted confirms that a non-empty
// LoadOptions.Registry is plumbed through to load.Config without aborting
// the load on a platform that does not import any registry-backed modules.
func TestLoadPlatformPackage_RegistryOptionAccepted(t *testing.T) {
	dir := writeTempPlatformDir(t, platformFixture)

	val, err := loader.LoadPlatformPackage(cuecontext.New(), dir, loader.LoadOptions{
		Registry: "testing.opmodel.dev=localhost:5000+insecure",
	})
	require.NoError(t, err, "registry override must be accepted even when no imports use it")
	assert.True(t, val.Exists())
}

func TestLoadPlatformPackage_NotADirectory(t *testing.T) {
	dir := writeTempPlatformDir(t, platformFixture)
	filePath := filepath.Join(dir, "platform.cue")

	_, err := loader.LoadPlatformPackage(cuecontext.New(), filePath, loader.LoadOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not a directory")
}

func TestLoadPlatformPackage_MissingPath(t *testing.T) {
	_, err := loader.LoadPlatformPackage(cuecontext.New(), "/no/such/path", loader.LoadOptions{})
	require.Error(t, err)
}

// TestKernelWrapper_LoadPlatformPackage locks the spec scenario "Kernel
// wrapper exists" — calling (k *Kernel).LoadPlatformPackage must produce the
// same cue.Value as the helper invoked with k.CueContext().
func TestKernelWrapper_LoadPlatformPackage(t *testing.T) {
	dir := writeTempPlatformDir(t, platformFixture)

	k := kernel.New()
	wrapVal, err := k.LoadPlatformPackage(context.Background(), dir, loader.LoadOptions{})
	require.NoError(t, err)

	helperVal, err := loader.LoadPlatformPackage(k.CueContext(), dir, loader.LoadOptions{})
	require.NoError(t, err)

	assert.True(t, wrapVal.Equals(helperVal), "wrapper and helper must yield equal cue.Values when given the same context")
}
