package file_test

import (
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
)

// writeTempReleaseDir writes content to a fresh temp dir as release.cue and
// returns the dir path. LoadReleasePackage operates on the directory itself.
func writeTempReleaseDir(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "release.cue"), []byte(content), 0o644))
	return dir
}

func TestLoadReleasePackage_DetectsV1alpha2(t *testing.T) {
	dir := writeTempReleaseDir(t, `
package release
apiVersion: "opmodel.dev/v1alpha2"
kind: "ModuleRelease"
metadata: {
	name: "demo"
	namespace: "ns"
}
`)

	val, ver, err := loader.LoadReleasePackage(cuecontext.New(), dir, loader.LoadOptions{})
	require.NoError(t, err)
	assert.True(t, val.Exists())
	assert.Equal(t, apiversion.V1alpha2, ver)
}

func TestLoadReleasePackage_RejectsMissingAPIVersion(t *testing.T) {
	dir := writeTempReleaseDir(t, `
package release
kind: "ModuleRelease"
metadata: name: "demo"
`)

	_, _, err := loader.LoadReleasePackage(cuecontext.New(), dir, loader.LoadOptions{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, apiversion.ErrUnknownAPIVersion), "want ErrUnknownAPIVersion, got %v", err)
}

func TestLoadReleasePackage_RejectsUnknownLiteral(t *testing.T) {
	dir := writeTempReleaseDir(t, `
package release
apiVersion: "opmodel.dev/v9beta42"
kind: "ModuleRelease"
metadata: name: "demo"
`)

	_, _, err := loader.LoadReleasePackage(cuecontext.New(), dir, loader.LoadOptions{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, apiversion.ErrUnknownAPIVersion))
}

// TestLoadReleasePackage_MultiFile pins the multi-file release-package
// behavior that motivates the package-loader unification: release.cue and
// values.cue share a package declaration; LoadReleasePackage MUST unify
// them into one cue.Value and detect the apiVersion from the union.
func TestLoadReleasePackage_MultiFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "release.cue"), []byte(`
package release

apiVersion: "opmodel.dev/v1alpha2"
kind:       "ModuleRelease"
metadata: {
	name:      "demo"
	namespace: "ns"
}
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "values.cue"), []byte(`
package release

values: { replicas: 3 }
`), 0o644))

	val, ver, err := loader.LoadReleasePackage(cuecontext.New(), dir, loader.LoadOptions{})
	require.NoError(t, err)
	assert.True(t, val.Exists())
	assert.Equal(t, apiversion.V1alpha2, ver)

	// The unified value carries both fields.
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
// A fully-online round trip — actually resolving an import via the
// override registry — lives in opm/kernel/flow_integration_test.go, which
// is gated on the local registry being reachable.
func TestLoadReleasePackage_RegistryOptionAccepted(t *testing.T) {
	dir := writeTempReleaseDir(t, `
package release
apiVersion: "opmodel.dev/v1alpha2"
kind: "ModuleRelease"
metadata: { name: "demo", namespace: "ns" }
`)

	val, ver, err := loader.LoadReleasePackage(cuecontext.New(), dir, loader.LoadOptions{
		Registry: "testing.opmodel.dev=localhost:5000+insecure",
	})
	require.NoError(t, err, "registry override must be accepted even when no imports use it")
	assert.True(t, val.Exists())
	assert.Equal(t, apiversion.V1alpha2, ver)
}
