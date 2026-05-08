package v1alpha2_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/pkg/api"
	"github.com/open-platform-model/library/pkg/apiversion"
)

// schemaSourceDir resolves the on-disk path of apis/core/v1alpha2 relative to
// this test file's location. Using runtime.Caller keeps the test working when
// `go test ./...` is invoked from any directory.
func schemaSourceDir(t *testing.T) string {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller failed")
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(here), "..", "..", ".."))
	return filepath.Join(repoRoot, "apis", "core", "v1alpha2")
}

// listEmbeddedFiles returns the sorted list of regular files in the embedded
// FS, with paths normalised to slash-form relative paths.
func listEmbeddedFiles(t *testing.T, fsys fs.FS) []string {
	t.Helper()
	var files []string
	require.NoError(t, fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		files = append(files, path)
		return nil
	}))
	sort.Strings(files)
	return files
}

// listOnDiskSchemaFiles returns the sorted list of files matching the embed
// pattern (*.cue and cue.mod/module.cue) under the on-disk schema dir.
func listOnDiskSchemaFiles(t *testing.T, dir string) []string {
	t.Helper()
	var files []string

	cueGlob, err := filepath.Glob(filepath.Join(dir, "*.cue"))
	require.NoError(t, err)
	for _, p := range cueGlob {
		rel, err := filepath.Rel(dir, p)
		require.NoError(t, err)
		files = append(files, filepath.ToSlash(rel))
	}

	modPath := filepath.Join(dir, "cue.mod", "module.cue")
	if _, statErr := os.Stat(modPath); statErr == nil {
		files = append(files, "cue.mod/module.cue")
	}

	sort.Strings(files)
	return files
}

func TestEmbeddedSchema_Available(t *testing.T) {
	got, err := api.EmbeddedSchema(apiversion.V1alpha2)
	require.NoError(t, err)
	require.NotNil(t, got)
}

func TestEmbeddedSchema_FileSetMatchesDisk(t *testing.T) {
	fsys, err := api.EmbeddedSchema(apiversion.V1alpha2)
	require.NoError(t, err)

	dir := schemaSourceDir(t)
	wantFiles := listOnDiskSchemaFiles(t, dir)
	gotFiles := listEmbeddedFiles(t, fsys)
	assert.Equal(t, wantFiles, gotFiles, "embedded file list must match the on-disk schema source")
}

func TestEmbeddedSchema_BytesMatchDisk(t *testing.T) {
	fsys, err := api.EmbeddedSchema(apiversion.V1alpha2)
	require.NoError(t, err)

	dir := schemaSourceDir(t)
	for _, rel := range listOnDiskSchemaFiles(t, dir) {
		t.Run(rel, func(t *testing.T) {
			diskBytes, err := os.ReadFile(filepath.Join(dir, rel))
			require.NoError(t, err)
			fsBytes, err := fs.ReadFile(fsys, rel)
			require.NoError(t, err)
			assert.Equal(t, diskBytes, fsBytes, "byte mismatch for %s", rel)
		})
	}
}
