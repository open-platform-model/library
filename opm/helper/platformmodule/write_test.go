package platformmodule

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// platform-module-generation spec, "Write into an empty directory".
func TestFiles_WriteTo(t *testing.T) {
	files, err := Generate(twoCatalogInput())
	require.NoError(t, err)
	dir := t.TempDir()
	require.NoError(t, files.WriteTo(dir))

	for name, want := range files {
		got, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
		require.NoError(t, err, "%s was not written", name)
		assert.Equal(t, want, got, "%s bytes differ", name)
	}
	info, err := os.Stat(filepath.Join(dir, "platform.cue"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o644), info.Mode().Perm())

	// Parents are created: the target need not exist yet.
	nested := filepath.Join(t.TempDir(), "gen", "1")
	require.NoError(t, files.WriteTo(nested))
	_, err = os.Stat(filepath.Join(nested, "cue.mod", "module.cue"))
	require.NoError(t, err)
}

// platform-module-generation spec, "Path escape refused": the write fails
// naming the file and writes nothing, even when other files are valid.
func TestFiles_WriteTo_RefusesEscape(t *testing.T) {
	for _, escape := range []string{"../escape.cue", "cue.mod/../../escape.cue", "/abs/escape.cue", "..", "."} {
		t.Run(escape, func(t *testing.T) {
			dir := t.TempDir()
			err := Files{
				PlatformFileName: []byte("package platform\n"),
				escape:           []byte("nope"),
			}.WriteTo(dir)
			require.Error(t, err)
			assert.Contains(t, err.Error(), escape)
			entries, readErr := os.ReadDir(dir)
			require.NoError(t, readErr)
			assert.Empty(t, entries, "a refused write must leave the directory untouched")
		})
	}
}

func TestFiles_WriteTo_RequiresDir(t *testing.T) {
	require.Error(t, Files{}.WriteTo(""))
}
