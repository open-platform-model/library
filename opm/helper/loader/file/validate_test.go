package file_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loader "github.com/open-platform-model/library/opm/helper/loader/file"
)

// loadFn abstracts the three package loaders so one table can drive all of
// them through identical fixtures.
type loadFn func(dir string) error

func moduleLoad(dir string) error {
	_, err := loader.LoadModulePackage(cuecontext.New(), dir, loader.LoadOptions{})
	return err
}

func releaseLoad(dir string) error {
	_, err := loader.LoadReleasePackage(cuecontext.New(), dir, loader.LoadOptions{})
	return err
}

func platformLoad(dir string) error {
	_, err := loader.LoadPlatformPackage(cuecontext.New(), dir, loader.LoadOptions{})
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
kind:       "ModuleRelease"
metadata: {name: "demo", namespace: "ns"}
`,
			sentinel: loader.ErrMissingRequiredField,
		},
		{
			name: "platform missing type",
			load: platformLoad,
			content: `
package platform
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

	_, err := loader.LoadModulePackage(cuecontext.New(), dir, loader.LoadOptions{})
	require.Error(t, err)
}

// TestShapeGate_WellFormedArtifactsPass is the regression guard: a well-formed
// module, release, and platform each still load successfully.
func TestShapeGate_WellFormedArtifactsPass(t *testing.T) {
	t.Run("module", func(t *testing.T) {
		dir := writeTempModuleDir(t, `
package mod
kind:       "Module"
metadata: {name: "demo", modulePath: "example.com/modules", version: "0.1.0"}
`)
		val, err := loader.LoadModulePackage(cuecontext.New(), dir, loader.LoadOptions{})
		require.NoError(t, err)
		assert.True(t, val.Exists())
	})

	t.Run("release", func(t *testing.T) {
		dir := writeTempReleaseDir(t, `
package release
kind:       "ModuleRelease"
metadata: {name: "demo", namespace: "ns"}
#module: {kind: "Module"}
`)
		val, err := loader.LoadReleasePackage(cuecontext.New(), dir, loader.LoadOptions{})
		require.NoError(t, err)
		assert.True(t, val.Exists())
	})

	t.Run("platform", func(t *testing.T) {
		dir := writeTempPlatformDir(t, `
package platform
kind:       "Platform"
metadata: {name: "demo"}
type: "kubernetes"
#registry: {}
`)
		val, err := loader.LoadPlatformPackage(cuecontext.New(), dir, loader.LoadOptions{})
		require.NoError(t, err)
		assert.True(t, val.Exists())
	})
}
