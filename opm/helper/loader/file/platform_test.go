package file_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loader "github.com/open-platform-model/library/opm/helper/loader/file"
	"github.com/open-platform-model/library/opm/internal/schematest"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/schema"
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

// writePlatformModule writes a platform module importing core (resolved from
// the warm workspace cache at the release schema.DefaultSchemaModule pins)
// whose #registry is registryBody, plus the extra package-level source in
// extra, and returns the module directory.
func writePlatformModule(t *testing.T, registryBody, extra string) string {
	t.Helper()
	dir := t.TempDir()
	coreVersion := strings.TrimPrefix(schema.DefaultSchemaModule, "opmodel.dev/core@")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cue.mod"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cue.mod", "module.cue"), []byte(fmt.Sprintf(`module: "testing.opmodel.dev/loader-platform-test@v0"
language: version: "v0.17.0"
deps: "opmodel.dev/core@v2": v: %q
`, coreVersion)), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "platform.cue"), []byte(fmt.Sprintf(`package platform

import c "opmodel.dev/core@v2"

c.#Platform
metadata: name: "gated"
type: "kubernetes"
#registry: %s
%s`, registryBody, extra)), 0o644))
	return dir
}

// helper-packages spec, "Registry entry with no embedded catalog rejected":
// the retired subscription shape (a version scalar, no #catalog) is
// incomplete exactly where the embedded catalog would have completed it
// (core derives `version` from it, 0019 D5), and the gate names the entry.
func TestLoadPlatformPackage_SubscriptionShapedRegistryRejected(t *testing.T) {
	schematest.SetEnv(t)
	dir := writePlatformModule(t, `"example.test/cat@v0": {enable: true, version: "0.1.0"}`, "")

	_, err := loader.LoadPlatformPackage(cuecontext.New(), dir, loader.LoadOptions{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, loader.ErrMissingRequiredField), "got %v", err)
	assert.Contains(t, err.Error(), "example.test/cat@v0", "the refusal names the entry")
	assert.Contains(t, err.Error(), "version", "the refusal names the field the embedded catalog would supply")
	assert.Contains(t, err.Error(), "#registry")
}

// helper-packages spec, "Registry entry with no embedded catalog rejected",
// second half: the same entry with a catalog embedded passes, its `version`
// derived from the catalog's stamped identity. An empty #registry passes too.
func TestLoadPlatformPackage_EmbeddedCatalogRegistryPasses(t *testing.T) {
	schematest.SetEnv(t)
	dir := writePlatformModule(t, `"example.test/cat@v0": {enable: true, #catalog: _cat}`, `
_cat: c.#Catalog & {
	metadata: {
		modulePath:  "example.test/cat@v0"
		version:     "0.1.0"
		description: "inline catalog"
	}
	#transformers: {}
}
`)

	val, err := loader.LoadPlatformPackage(cuecontext.New(), dir, loader.LoadOptions{})
	require.NoError(t, err)
	version, err := val.LookupPath(cue.ParsePath(`#registry."example.test/cat@v0".version`)).String()
	require.NoError(t, err)
	assert.Equal(t, "0.1.0", version, "core derives the entry version from the embedded catalog")

	empty := writePlatformModule(t, `{}`, "")
	_, err = loader.LoadPlatformPackage(cuecontext.New(), empty, loader.LoadOptions{})
	require.NoError(t, err, "a platform carrying no catalogs is complete")
}
