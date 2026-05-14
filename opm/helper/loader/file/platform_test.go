package file_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/apiversion"
	loader "github.com/open-platform-model/library/opm/helper/loader/file"
	"github.com/open-platform-model/library/opm/kernel"
)

const platformFixture = `
package platform
apiVersion: "opmodel.dev/v1alpha2"
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

func TestLoadPlatformPackage_DetectsV1alpha2(t *testing.T) {
	dir := writeTempPlatformDir(t, platformFixture)

	val, ver, err := loader.LoadPlatformPackage(cuecontext.New(), dir, loader.LoadOptions{})
	require.NoError(t, err)
	assert.True(t, val.Exists())
	assert.Equal(t, apiversion.V1alpha2, ver)

	name, err := val.LookupPath(cue.ParsePath("metadata.name")).String()
	require.NoError(t, err)
	assert.Equal(t, "demo-platform", name)
}

func TestLoadPlatformPackage_RejectsMissingAPIVersion(t *testing.T) {
	dir := writeTempPlatformDir(t, `
package platform
kind: "Platform"
metadata: name: "demo-platform"
`)

	_, _, err := loader.LoadPlatformPackage(cuecontext.New(), dir, loader.LoadOptions{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, apiversion.ErrUnknownAPIVersion), "want ErrUnknownAPIVersion, got %v", err)
}

// TestLoadPlatformPackage_RegistryOptionAccepted confirms that a non-empty
// LoadOptions.Registry is plumbed through to load.Config without aborting
// the load on a platform that does not import any registry-backed modules.
// A fully-online round trip — actually resolving an import via the override
// registry — lives in opm/kernel/flow_integration_test.go, which is gated
// on the local registry being reachable.
func TestLoadPlatformPackage_RegistryOptionAccepted(t *testing.T) {
	dir := writeTempPlatformDir(t, platformFixture)

	val, ver, err := loader.LoadPlatformPackage(cuecontext.New(), dir, loader.LoadOptions{
		Registry: "testing.opmodel.dev=localhost:5000+insecure",
	})
	require.NoError(t, err, "registry override must be accepted even when no imports use it")
	assert.True(t, val.Exists())
	assert.Equal(t, apiversion.V1alpha2, ver)
}

func TestLoadPlatformPackage_NotADirectory(t *testing.T) {
	dir := writeTempPlatformDir(t, platformFixture)
	filePath := filepath.Join(dir, "platform.cue")

	_, _, err := loader.LoadPlatformPackage(cuecontext.New(), filePath, loader.LoadOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not a directory")
}

func TestLoadPlatformPackage_MissingPath(t *testing.T) {
	_, _, err := loader.LoadPlatformPackage(cuecontext.New(), "/no/such/path", loader.LoadOptions{})
	require.Error(t, err)
}

// TestKernelWrapper_LoadPlatformPackage locks the spec scenario "Kernel
// wrapper exists" — calling (k *Kernel).LoadPlatformPackage must produce the
// same cue.Value and apiVersion as the helper invoked with k.CueContext().
func TestKernelWrapper_LoadPlatformPackage(t *testing.T) {
	dir := writeTempPlatformDir(t, platformFixture)

	k := kernel.New()
	wrapVal, wrapVer, err := k.LoadPlatformPackage(context.Background(), dir, loader.LoadOptions{})
	require.NoError(t, err)

	helperVal, helperVer, err := loader.LoadPlatformPackage(k.CueContext(), dir, loader.LoadOptions{})
	require.NoError(t, err)

	assert.Equal(t, helperVer, wrapVer)
	assert.True(t, wrapVal.Equals(helperVal), "wrapper and helper must yield equal cue.Values when given the same context")
}

// TestLoadPlatformPackage_RepoFixture exercises the canonical fixture under
// library/testdata/platform/v1alpha2/ — the directory the binding-path tests
// also use, so a single concrete artifact covers loader + decoder paths.
func TestLoadPlatformPackage_RepoFixture(t *testing.T) {
	// The test runs from opm/helper/loader/file/. Walk up to the repo root
	// and into testdata/.
	fixtureDir := filepath.Join("..", "..", "..", "..", "testdata", "platform", "v1alpha2")
	abs, err := filepath.Abs(fixtureDir)
	require.NoError(t, err)
	if _, err := os.Stat(abs); err != nil {
		t.Skipf("fixture not found at %s: %v", abs, err)
	}

	val, ver, err := loader.LoadPlatformPackage(cuecontext.New(), abs, loader.LoadOptions{})
	require.NoError(t, err)
	assert.True(t, val.Exists())
	assert.Equal(t, apiversion.V1alpha2, ver)

	name, err := val.LookupPath(cue.ParsePath("metadata.name")).String()
	require.NoError(t, err)
	assert.Equal(t, "test-platform", name)
}
