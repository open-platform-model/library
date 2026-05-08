package loader_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/pkg/apiversion"
	"github.com/open-platform-model/library/pkg/loader"
)

// writeTempModuleDir writes a single-file CUE package under a fresh temp dir
// and returns the dir path. The file uses package name "mod" so load.Instances
// resolves it as a single-package directory without needing cue.mod.
func writeTempModuleDir(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "module.cue"), []byte(content), 0o644))
	return dir
}

func TestLoadModulePackage_DetectsV1alpha2(t *testing.T) {
	dir := writeTempModuleDir(t, `
package mod
apiVersion: "opmodel.dev/v1alpha2"
kind: "Module"
`)

	val, ver, err := loader.LoadModulePackage(cuecontext.New(), dir)
	require.NoError(t, err)
	assert.True(t, val.Exists())
	assert.Equal(t, apiversion.V1alpha2, ver)
}

func TestLoadModulePackage_RejectsMissingAPIVersion(t *testing.T) {
	dir := writeTempModuleDir(t, `
package mod
kind: "Module"
`)

	_, _, err := loader.LoadModulePackage(cuecontext.New(), dir)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apiversion.ErrUnknownAPIVersion))
}
