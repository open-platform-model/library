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

// writeTempInstanceDir writes content to a fresh temp dir as instance.cue and
// returns the dir path. LoadInstancePackage operates on the directory itself.
func writeTempInstanceDir(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "instance.cue"), []byte(content), 0o644))
	return dir
}

func TestLoadInstancePackage_Loads(t *testing.T) {
	dir := writeTempInstanceDir(t, `
package instance
kind: "ModuleInstance"
metadata: {
	name: "demo"
	namespace: "ns"
}
#module: {kind: "Module"}
`)

	val, err := loader.LoadInstancePackage(cuecontext.New(), dir, loader.LoadOptions{})
	require.NoError(t, err)
	assert.True(t, val.Exists())
}

// TestLoadInstancePackage_MultiFile pins the multi-file instance-package
// behavior that motivates the package-loader unification: instance.cue and
// values.cue share a package declaration; LoadInstancePackage MUST unify
// them into one cue.Value.
func TestLoadInstancePackage_MultiFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "instance.cue"), []byte(`
package instance

kind:       "ModuleInstance"
metadata: {
	name:      "demo"
	namespace: "ns"
}
#module: {kind: "Module"}
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "values.cue"), []byte(`
package instance

values: { replicas: 3 }
`), 0o644))

	val, err := loader.LoadInstancePackage(cuecontext.New(), dir, loader.LoadOptions{})
	require.NoError(t, err)
	assert.True(t, val.Exists())

	name, err := val.LookupPath(cue.ParsePath("metadata.name")).String()
	require.NoError(t, err)
	assert.Equal(t, "demo", name)

	replicas, err := val.LookupPath(cue.ParsePath("values.replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(3), replicas)
}

// TestLoadInstancePackage_RegistryOptionAccepted confirms that a non-empty
// LoadOptions.Registry is plumbed through to load.Config without aborting
// the load on an instance that does not import any registry-backed modules.
func TestLoadInstancePackage_RegistryOptionAccepted(t *testing.T) {
	dir := writeTempInstanceDir(t, `
package instance
kind: "ModuleInstance"
metadata: { name: "demo", namespace: "ns" }
#module: {kind: "Module"}
`)

	val, err := loader.LoadInstancePackage(cuecontext.New(), dir, loader.LoadOptions{
		Registry: "testing.opmodel.dev=localhost:5000+insecure",
	})
	require.NoError(t, err, "registry override must be accepted even when no imports use it")
	assert.True(t, val.Exists())
}
