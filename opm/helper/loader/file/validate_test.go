package file_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/apiversion"
	loader "github.com/open-platform-model/library/opm/helper/loader/file"
)

// loadFn abstracts the three package loaders so one table can drive all of
// them through identical fixtures.
type loadFn func(dir string) error

func moduleLoad(dir string) error {
	_, _, err := loader.LoadModulePackage(cuecontext.New(), dir, loader.LoadOptions{})
	return err
}

func releaseLoad(dir string) error {
	_, _, err := loader.LoadReleasePackage(cuecontext.New(), dir, loader.LoadOptions{})
	return err
}

func platformLoad(dir string) error {
	_, _, err := loader.LoadPlatformPackage(cuecontext.New(), dir, loader.LoadOptions{})
	return err
}

// TestShapeGate_RejectsMalformedPackages drives every loader through a fixture
// that violates one shape-gate rule and asserts the returned error wraps the
// expected sentinel.
func TestShapeGate_RejectsMalformedPackages(t *testing.T) {
	tests := []struct {
		name     string
		load     loadFn
		content  string
		sentinel error
	}{
		{
			name: "module loaded from a platform package",
			load: moduleLoad,
			content: `
package mod
apiVersion: "opmodel.dev/v1alpha2"
kind:       "Platform"
metadata: {name: "demo", modulePath: "example.com/modules", version: "0.1.0"}
`,
			sentinel: loader.ErrWrongKind,
		},
		{
			name: "module missing metadata.name",
			load: moduleLoad,
			content: `
package mod
apiVersion: "opmodel.dev/v1alpha2"
kind:       "Module"
metadata: {modulePath: "example.com/modules", version: "0.1.0"}
`,
			sentinel: loader.ErrMissingRequiredField,
		},
		{
			name: "release with a non-module #module",
			load: releaseLoad,
			content: `
package release
apiVersion: "opmodel.dev/v1alpha2"
kind:       "ModuleRelease"
metadata: {name: "demo", namespace: "ns"}
#module: {kind: "Platform"}
`,
			sentinel: loader.ErrWrongKind,
		},
		{
			name: "release missing #module",
			load: releaseLoad,
			content: `
package release
apiVersion: "opmodel.dev/v1alpha2"
kind:       "ModuleRelease"
metadata: {name: "demo", namespace: "ns"}
`,
			sentinel: loader.ErrMissingRequiredField,
		},
		{
			name: "platform registry entry pointing at a non-module",
			load: platformLoad,
			content: `
package platform
apiVersion: "opmodel.dev/v1alpha2"
kind:       "Platform"
metadata: {name: "demo"}
type: "kubernetes"
#registry: {bad: {#module: {kind: "Platform"}}}
`,
			sentinel: loader.ErrWrongKind,
		},
		{
			name: "platform missing type",
			load: platformLoad,
			content: `
package platform
apiVersion: "opmodel.dev/v1alpha2"
kind:       "Platform"
metadata: {name: "demo"}
`,
			sentinel: loader.ErrMissingRequiredField,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "artifact.cue"), []byte(tc.content), 0o644))

			err := tc.load(dir)
			require.Error(t, err)
			assert.True(t, errors.Is(err, tc.sentinel), "want %v, got %v", tc.sentinel, err)
		})
	}
}

// TestShapeGate_RejectsConflictingPackageClauses pins the CUE-loader behavior:
// two files in one directory declaring different package names is a hard
// error, surfaced rather than silently resolved to one instance.
func TestShapeGate_RejectsConflictingPackageClauses(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.cue"), []byte("package one\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.cue"), []byte("package two\n"), 0o644))

	_, _, err := loader.LoadModulePackage(cuecontext.New(), dir, loader.LoadOptions{})
	require.Error(t, err)
}

// TestShapeGate_WellFormedArtifactsPass is the regression guard: a well-formed
// module, release, and platform each still load and return the detected
// apiVersion exactly as before the shape gate was introduced.
func TestShapeGate_WellFormedArtifactsPass(t *testing.T) {
	t.Run("module", func(t *testing.T) {
		dir := writeTempModuleDir(t, `
package mod
apiVersion: "opmodel.dev/v1alpha2"
kind:       "Module"
metadata: {name: "demo", modulePath: "example.com/modules", version: "0.1.0"}
`)
		val, ver, err := loader.LoadModulePackage(cuecontext.New(), dir, loader.LoadOptions{})
		require.NoError(t, err)
		assert.True(t, val.Exists())
		assert.Equal(t, apiversion.V1alpha2, ver)
	})

	t.Run("release", func(t *testing.T) {
		dir := writeTempReleaseDir(t, `
package release
apiVersion: "opmodel.dev/v1alpha2"
kind:       "ModuleRelease"
metadata: {name: "demo", namespace: "ns"}
#module: {kind: "Module"}
`)
		val, ver, err := loader.LoadReleasePackage(cuecontext.New(), dir, loader.LoadOptions{})
		require.NoError(t, err)
		assert.True(t, val.Exists())
		assert.Equal(t, apiversion.V1alpha2, ver)
	})

	t.Run("platform", func(t *testing.T) {
		dir := writeTempPlatformDir(t, `
package platform
apiVersion: "opmodel.dev/v1alpha2"
kind:       "Platform"
metadata: {name: "demo"}
type: "kubernetes"
#registry: {}
`)
		val, ver, err := loader.LoadPlatformPackage(cuecontext.New(), dir, loader.LoadOptions{})
		require.NoError(t, err)
		assert.True(t, val.Exists())
		assert.Equal(t, apiversion.V1alpha2, ver)
	})
}
