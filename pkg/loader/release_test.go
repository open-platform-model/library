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

// writeTempCUEFile writes content to a fresh temp dir as release.cue and
// returns the dir path. Used so LoadReleaseFile sees a real on-disk artifact
// without depending on the workspace catalog.
func writeTempCUEFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "release.cue")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func TestLoadReleaseFile_DetectsV1alpha2(t *testing.T) {
	path := writeTempCUEFile(t, `
package release
apiVersion: "opmodel.dev/v1alpha2"
kind: "ModuleRelease"
metadata: {
	name: "demo"
	namespace: "ns"
}
`)

	val, dir, ver, err := loader.LoadReleaseFile(cuecontext.New(), path, loader.LoadOptions{})
	require.NoError(t, err)
	assert.True(t, val.Exists())
	assert.NotEmpty(t, dir)
	assert.Equal(t, apiversion.V1alpha2, ver)
}

func TestLoadReleaseFile_RejectsMissingAPIVersion(t *testing.T) {
	path := writeTempCUEFile(t, `
package release
kind: "ModuleRelease"
metadata: name: "demo"
`)

	_, _, _, err := loader.LoadReleaseFile(cuecontext.New(), path, loader.LoadOptions{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, apiversion.ErrUnknownAPIVersion), "want ErrUnknownAPIVersion, got %v", err)
}

func TestLoadReleaseFile_RejectsUnknownLiteral(t *testing.T) {
	path := writeTempCUEFile(t, `
package release
apiVersion: "opmodel.dev/v9beta42"
kind: "ModuleRelease"
metadata: name: "demo"
`)

	_, _, _, err := loader.LoadReleaseFile(cuecontext.New(), path, loader.LoadOptions{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, apiversion.ErrUnknownAPIVersion))
}
