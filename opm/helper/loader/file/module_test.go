package file_test

import (
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loader "github.com/open-platform-model/library/opm/helper/loader/file"
)

// writeTempModuleDir writes a single-file CUE package under a fresh temp dir
// and returns the dir path.
func writeTempModuleDir(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "module.cue"), []byte(content), 0o644))
	return dir
}

func TestLoadModulePackage_Loads(t *testing.T) {
	dir := writeTempModuleDir(t, `
package mod
kind: "Module"
metadata: {
	name:       "demo"
	modulePath: "example.com/modules"
	version:    "0.1.0"
}
`)

	val, err := loader.LoadModulePackage(cuecontext.New(), dir, loader.LoadOptions{})
	require.NoError(t, err)
	assert.True(t, val.Exists())
}
