package kernel_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loader "github.com/open-platform-model/library/opm/helper/loader/file"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/schema"
)

// TestIntegration_Live_ValidateRealConfig loads the real web_app fixture from
// disk and validates its authored debugValues against its real #config schema.
//
// This is the live counterpart to the hermetic Validate cases: it exercises the
// filesystem loader (LoadModulePackage) and validation against the published
// core@v0 schema and real catalog primitives — paths the in-memory harness
// deliberately bypasses. Gated like the flow tests: skipped under -short or when
// localhost:5000 is unreachable; OPM_FLOW_TEST_FORCE=1 makes the skip a failure.
func TestIntegration_Live_ValidateRealConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("live integration test requires the local CUE registry; skipping under -short")
	}
	skipUnlessRegistry(t)

	moduleDir := filepath.Join(repoLibraryRoot(t), "testdata", "modules", "web_app")
	registry := flowRegistry()
	t.Setenv("CUE_REGISTRY", registry)

	k := kernel.New()
	ctx := context.Background()

	modVal, err := k.LoadModulePackage(ctx, moduleDir, loader.LoadOptions{Registry: registry})
	require.NoErrorf(t, err, "loading module package from %s", moduleDir)

	mod, err := k.NewModuleFromValue(modVal)
	require.NoError(t, err)
	require.Equal(t, "web-app", mod.Metadata.Name)

	debugValues := modVal.LookupPath(schema.DebugValues)
	require.True(t, debugValues.Exists(), "web_app fixture must provide debugValues")

	out, err := k.ValidateModuleValues(mod, debugValues)
	require.NoError(t, err, "real debugValues must satisfy the real #config schema")
	assert.True(t, out.Exists())
}
