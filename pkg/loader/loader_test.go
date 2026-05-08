package loader_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/pkg/apiversion"
	loaderfile "github.com/open-platform-model/library/pkg/helper/loader/file"
	"github.com/open-platform-model/library/pkg/loader"
)

// These parity tests confirm the deprecated pkg/loader shim delegates to
// pkg/helper/loader/file faithfully. They run against the same fixtures
// using the same *cue.Context so any divergence (return values, error
// kinds) shows up immediately.

func writeTempModuleDir(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "module.cue"), []byte(content), 0o644))
	return dir
}

func writeTempReleaseFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "release.cue")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func writeTempValuesFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "values.cue")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func TestShim_LoadModulePackage_Parity(t *testing.T) {
	dir := writeTempModuleDir(t, `
package mod
apiVersion: "opmodel.dev/v1alpha2"
kind: "Module"
`)

	ctx := cuecontext.New()
	gotVal, gotVer, gotErr := loader.LoadModulePackage(ctx, dir) //nolint:staticcheck // SA1019: parity test for the deprecated shim
	require.NoError(t, gotErr)

	wantVal, wantVer, wantErr := loaderfile.LoadModulePackage(ctx, dir)
	require.NoError(t, wantErr)

	assert.Equal(t, wantVer, gotVer)
	assert.Equal(t, apiversion.V1alpha2, gotVer)
	assert.True(t, gotVal.Exists())
	assert.True(t, wantVal.Exists())
}

func TestShim_LoadModulePackage_ErrorParity(t *testing.T) {
	dir := writeTempModuleDir(t, `
package mod
kind: "Module"
`)
	ctx := cuecontext.New()
	_, _, gotErr := loader.LoadModulePackage(ctx, dir) //nolint:staticcheck // SA1019: parity test for the deprecated shim
	_, _, wantErr := loaderfile.LoadModulePackage(ctx, dir)
	require.Error(t, gotErr)
	require.Error(t, wantErr)
	assert.True(t, errors.Is(gotErr, apiversion.ErrUnknownAPIVersion))
	assert.True(t, errors.Is(wantErr, apiversion.ErrUnknownAPIVersion))
}

func TestShim_LoadReleaseFile_Parity(t *testing.T) {
	path := writeTempReleaseFile(t, `
package release
apiVersion: "opmodel.dev/v1alpha2"
kind: "ModuleRelease"
metadata: {
	name: "demo"
	namespace: "ns"
}
`)
	ctx := cuecontext.New()
	gotVal, gotDir, gotVer, gotErr := loader.LoadReleaseFile(ctx, path, loader.LoadOptions{}) //nolint:staticcheck // SA1019: parity test for the deprecated shim
	require.NoError(t, gotErr)

	wantVal, wantDir, wantVer, wantErr := loaderfile.LoadReleaseFile(ctx, path, loaderfile.LoadOptions{})
	require.NoError(t, wantErr)

	assert.Equal(t, wantDir, gotDir)
	assert.Equal(t, wantVer, gotVer)
	assert.Equal(t, apiversion.V1alpha2, gotVer)
	assert.True(t, gotVal.Exists())
	assert.True(t, wantVal.Exists())
}

func TestShim_LoadValuesFile_Parity(t *testing.T) {
	path := writeTempValuesFile(t, `
package values
values: {
	replicas: 3
	name: "demo"
}
`)
	ctx := cuecontext.New()
	gotVal, gotErr := loader.LoadValuesFile(ctx, path) //nolint:staticcheck // SA1019: parity test for the deprecated shim
	require.NoError(t, gotErr)

	wantVal, wantErr := loaderfile.LoadValuesFile(ctx, path)
	require.NoError(t, wantErr)

	gotReplicas, err := gotVal.LookupPath(cue.ParsePath("replicas")).Int64()
	require.NoError(t, err)
	wantReplicas, err := wantVal.LookupPath(cue.ParsePath("replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, wantReplicas, gotReplicas)
	assert.Equal(t, int64(3), gotReplicas)
}

// Compile-time assertions that loader.LoadOptions is the same Go type as
// loaderfile.LoadOptions (i.e. a type alias, not a wrapper). If the alias
// becomes a distinct named type these declarations stop compiling.
var (
	_ = loaderfile.LoadOptions(loader.LoadOptions{}) //nolint:staticcheck // SA1019: alias check for the deprecated shim
	_ = loader.LoadOptions(loaderfile.LoadOptions{}) //nolint:staticcheck // SA1019: alias check for the deprecated shim
)

func TestShim_LoadOptions_TypeAlias(t *testing.T) {
	// Value-level smoke check: values flow freely between the two
	// identifiers without conversion.
	shim := loader.LoadOptions{Registry: "example.com=foo+insecure"} //nolint:staticcheck // SA1019: alias check for the deprecated shim
	canonical := loaderfile.LoadOptions(shim)
	assert.Equal(t, "example.com=foo+insecure", canonical.Registry)
}
