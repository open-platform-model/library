package file_test

import (
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loader "github.com/open-platform-model/library/opm/helper/loader/file"
)

// writeTempReleaseDir writes content to a fresh temp dir as release.cue and
// returns the dir path. LoadReleasePackage operates on the directory itself.
func writeTempReleaseDir(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "release.cue"), []byte(content), 0o644))
	return dir
}

func TestLoadReleasePackage_Loads(t *testing.T) {
	dir := writeTempReleaseDir(t, `
package release
kind: "ModuleRelease"
metadata: {
	name: "demo"
	namespace: "ns"
}
#module: {kind: "Module"}
`)

	val, err := loader.LoadReleasePackage(cuecontext.New(), dir, loader.LoadOptions{})
	require.NoError(t, err)
	assert.True(t, val.Exists())
}

// TestLoadReleasePackage_MultiFile pins the multi-file release-package
// behavior that motivates the package-loader unification: release.cue and
// values.cue share a package declaration; LoadReleasePackage MUST unify
// them into one cue.Value.
func TestLoadReleasePackage_MultiFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "release.cue"), []byte(`
package release

kind:       "ModuleRelease"
metadata: {
	name:      "demo"
	namespace: "ns"
}
#module: {kind: "Module"}
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "values.cue"), []byte(`
package release

values: { replicas: 3 }
`), 0o644))

	val, err := loader.LoadReleasePackage(cuecontext.New(), dir, loader.LoadOptions{})
	require.NoError(t, err)
	assert.True(t, val.Exists())

	name, err := val.LookupPath(cue.ParsePath("metadata.name")).String()
	require.NoError(t, err)
	assert.Equal(t, "demo", name)

	replicas, err := val.LookupPath(cue.ParsePath("values.replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(3), replicas)
}

// TestLoadReleasePackage_RegistryOptionAccepted confirms that a non-empty
// LoadOptions.Registry is plumbed through to load.Config without aborting
// the load on a release that does not import any registry-backed modules.
func TestLoadReleasePackage_RegistryOptionAccepted(t *testing.T) {
	dir := writeTempReleaseDir(t, `
package release
kind: "ModuleRelease"
metadata: { name: "demo", namespace: "ns" }
#module: {kind: "Module"}
`)

	val, err := loader.LoadReleasePackage(cuecontext.New(), dir, loader.LoadOptions{
		Registry: "testing.opmodel.dev=localhost:5000+insecure",
	})
	require.NoError(t, err, "registry override must be accepted even when no imports use it")
	assert.True(t, val.Exists())
}
