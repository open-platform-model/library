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

	loader "github.com/open-platform-model/library/pkg/helper/loader/file"
	"github.com/open-platform-model/library/pkg/kernel"
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

// writeTempPlatformFile writes a platform.cue under a fresh temp dir and
// returns the dir path. Mirrors writeTempCUEFile in release_test.go.
func writeTempPlatformFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "platform.cue"), []byte(content), 0o644))
	return dir
}

func TestLoadPlatformFile_DirectFilePath(t *testing.T) {
	dir := writeTempPlatformFile(t, platformFixture)
	filePath := filepath.Join(dir, "platform.cue")

	val, parent, err := loader.LoadPlatformFile(cuecontext.New(), filePath, loader.LoadOptions{})
	require.NoError(t, err)
	assert.True(t, val.Exists())
	assert.Equal(t, dir, parent)

	name, err := val.LookupPath(cue.ParsePath("metadata.name")).String()
	require.NoError(t, err)
	assert.Equal(t, "demo-platform", name)
}

func TestLoadPlatformFile_DirectoryPath(t *testing.T) {
	dir := writeTempPlatformFile(t, platformFixture)

	val, parent, err := loader.LoadPlatformFile(cuecontext.New(), dir, loader.LoadOptions{})
	require.NoError(t, err)
	assert.True(t, val.Exists())
	assert.Equal(t, dir, parent)

	typ, err := val.LookupPath(cue.ParsePath("type")).String()
	require.NoError(t, err)
	assert.Equal(t, "kubernetes", typ)
}

func TestLoadPlatformFile_DirectoryWithoutPlatformCue(t *testing.T) {
	dir := t.TempDir()

	_, _, err := loader.LoadPlatformFile(cuecontext.New(), dir, loader.LoadOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not contain platform.cue")
}

func TestLoadPlatformFile_MissingPath(t *testing.T) {
	_, _, err := loader.LoadPlatformFile(cuecontext.New(), "/no/such/path/platform.cue", loader.LoadOptions{})
	require.Error(t, err)
}

func TestLoadPlatformFile_EmptyPath(t *testing.T) {
	_, _, err := loader.LoadPlatformFile(cuecontext.New(), "", loader.LoadOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must not be empty")
}

// TestKernelWrapper_LoadPlatformFile locks the spec scenario "Kernel
// wrapper exists" — calling (k *Kernel).LoadPlatformFile must produce the
// same cue.Value and parentDir as the helper invoked with k.CueContext().
func TestKernelWrapper_LoadPlatformFile(t *testing.T) {
	dir := writeTempPlatformFile(t, platformFixture)

	k := kernel.New()
	wrapVal, wrapDir, err := k.LoadPlatformFile(context.Background(), dir, loader.LoadOptions{})
	require.NoError(t, err)

	helperVal, helperDir, err := loader.LoadPlatformFile(k.CueContext(), dir, loader.LoadOptions{})
	require.NoError(t, err)

	assert.Equal(t, wrapDir, helperDir)
	wrapName, err := wrapVal.LookupPath(cue.ParsePath("metadata.name")).String()
	require.NoError(t, err)
	helperName, err := helperVal.LookupPath(cue.ParsePath("metadata.name")).String()
	require.NoError(t, err)
	assert.Equal(t, helperName, wrapName)
	assert.True(t, wrapVal.Equals(helperVal), "wrapper and helper must yield equal cue.Values when given the same context")
}

// TestLoadPlatformFile_RepoFixture exercises the canonical fixture under
// library/testdata/platform/v1alpha2/ — the file the binding-path tests
// also use, so a single concrete artifact covers loader + decoder paths.
func TestLoadPlatformFile_RepoFixture(t *testing.T) {
	// The test runs from pkg/helper/loader/file/. Walk up to the repo root
	// and into testdata/.
	fixtureDir := filepath.Join("..", "..", "..", "..", "testdata", "platform", "v1alpha2")
	abs, err := filepath.Abs(fixtureDir)
	require.NoError(t, err)
	if _, err := os.Stat(abs); err != nil {
		t.Skipf("fixture not found at %s: %v", abs, err)
	}

	val, _, err := loader.LoadPlatformFile(cuecontext.New(), abs, loader.LoadOptions{})
	require.NoError(t, err)
	assert.True(t, val.Exists())

	name, err := val.LookupPath(cue.ParsePath("metadata.name")).String()
	require.NoError(t, err)
	assert.Equal(t, "test-platform", name)
}
