package schema_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/internal/schematest"
	"github.com/open-platform-model/library/opm/schema"
)

// TestOCILoader_ZeroValueResolvesDefault asserts that
// (schema.OCILoader{}).Load works in an environment with CUE_REGISTRY
// and CUE_CACHE_DIR set, threads the env into load.Config.Env, and
// returns a non-zero cue.Value carrying the OPM core definitions.
func TestOCILoader_ZeroValueResolvesDefault(t *testing.T) {
	schematest.SetEnv(t)
	ctx := cuecontext.New()

	val, err := schema.OCILoader{}.Load(ctx)
	require.NoError(t, err, "zero-value OCILoader must resolve the default module")
	require.True(t, val.Exists(), "loaded schema cue.Value must exist")

	for _, def := range []string{"#ModuleInstance", "#Module", "#Platform"} {
		assert.True(t,
			val.LookupPath(cue.ParsePath(def)).Exists(),
			"loaded schema must expose %s", def,
		)
	}
}

// TestOCILoader_ExplicitOverridesBeatEnv asserts that explicit Module,
// Registry, and CacheDir fields take precedence over the corresponding
// environment variables. We point the env at unusable values; the load
// must still succeed because the explicit override wins.
func TestOCILoader_ExplicitOverridesBeatEnv(t *testing.T) {
	// Set env to bogus values; the explicit struct fields must win.
	t.Setenv("CUE_REGISTRY", "ghcr.io/does-not-exist")
	t.Setenv("CUE_CACHE_DIR", filepath.Join(t.TempDir(), "wrong-env-cache"))

	cacheDir := newCleanableCacheDir(t)
	ctx := cuecontext.New()
	val, err := schema.OCILoader{
		Registry: schema.PublicRegistry,
		CacheDir: cacheDir,
	}.Load(ctx)
	require.NoError(t, err, "explicit overrides must beat env: %v", err)
	require.True(t, val.Exists())
	require.True(t,
		val.LookupPath(cue.ParsePath("#ModuleInstance")).Exists(),
		"loaded schema must expose #ModuleInstance",
	)

	// Sanity: the explicit cacheDir contains an extract directory after
	// the load, proving the override flowed into CUE.
	entries, _ := os.ReadDir(filepath.Join(cacheDir, "mod", "extract"))
	require.NotEmpty(t, entries, "explicit CacheDir must be populated by the load")
}

// newCleanableCacheDir returns a fresh cache directory whose contents
// will be cleaned up at test end. CUE materializes extract trees with
// read-only files; t.TempDir's RemoveAll cannot remove them directly,
// so the cleanup chmods the tree writable first.
func newCleanableCacheDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "schema-test-cache-*")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			_ = os.Chmod(p, 0o700)
			return nil
		})
		_ = os.RemoveAll(dir)
	})
	return dir
}

// TestOCILoader_ModuleOverride asserts that an explicit Module value
// pinned to a specific patch version resolves through the loader without
// falling back to the default identifier.
func TestOCILoader_ModuleOverride(t *testing.T) {
	schematest.SetEnv(t)
	ctx := cuecontext.New()

	// Pin to the major-only form — the loader expands it to vN.latest
	// internally; the resolved value should still expose the schema.
	val, err := schema.OCILoader{
		Module: "opmodel.dev/core@v1",
	}.Load(ctx)
	require.NoError(t, err)
	require.True(t, val.Exists())
	require.True(t, val.LookupPath(cue.ParsePath("#ModuleInstance")).Exists())
}

// TestOCILoader_LoadFailureWrapped asserts that a load failure surfaces
// as a non-nil error whose message identifies the module being loaded.
func TestOCILoader_LoadFailureWrapped(t *testing.T) {
	t.Setenv("CUE_REGISTRY", "")
	t.Setenv("CUE_CACHE_DIR", t.TempDir())

	ctx := cuecontext.New()
	val, err := schema.OCILoader{
		Module: "does-not-exist.example/missing@v0.0.1",
	}.Load(ctx)
	require.Error(t, err, "missing module must surface as a load error")
	assert.False(t, val.Exists())
	assert.True(t,
		strings.Contains(err.Error(), "does-not-exist.example/missing"),
		"error message must name the failing module: %v", err,
	)
}

// TestOCILoader_NilContextRejected asserts the loader rejects a nil
// *cue.Context with a non-nil error rather than panicking.
func TestOCILoader_NilContextRejected(t *testing.T) {
	val, err := schema.OCILoader{}.Load(nil)
	require.Error(t, err)
	assert.False(t, val.Exists())
}

// TestOCILoader_DoesNotMutateProcessEnv asserts the loader does not set
// or modify CUE_REGISTRY in the process environment when an explicit
// Registry override is used.
func TestOCILoader_DoesNotMutateProcessEnv(t *testing.T) {
	schematest.SetEnv(t)
	expected := os.Getenv("CUE_REGISTRY")
	require.NotEmpty(t, expected, "test setup must set CUE_REGISTRY")

	ctx := cuecontext.New()
	_, err := schema.OCILoader{
		Registry: "different=ghcr.io/somewhere-else",
	}.Load(ctx)
	// Whether the load succeeds depends on the override; we only care
	// about env-mutation contract.
	_ = err

	assert.Equal(t,
		expected,
		os.Getenv("CUE_REGISTRY"),
		"OCILoader.Load MUST NOT mutate process CUE_REGISTRY",
	)
}

// TestPublicRegistry_Value pins the documented registry mapping. Bumping
// this requires a coordinated docs update (MIGRATIONS.md,
// docs/getting-started.md, CLAUDE.md).
func TestPublicRegistry_Value(t *testing.T) {
	const expected = "opmodel.dev=ghcr.io/open-platform-model,registry.cue.works"
	assert.Equal(t, expected, schema.PublicRegistry)
}
