package kernel_test

import (
	"context"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/format"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/schema"
)

// TestIntegration_Live_ValidateRealConfig loads the real web_app fixture from
// disk and validates its authored debugValues against its real #config schema.
//
// This is the live counterpart to the hermetic Validate cases: it exercises
// the filesystem loader (LoadModulePackage) and validation against the
// published core@v2 schema and the real catalogs/opm v2 primitives — paths
// the in-memory harness deliberately bypasses. Gated like the flow tests:
// skipped under -short or when GHCR is unreachable; OPM_FLOW_TEST_FORCE=1
// makes the skip a failure.
func TestIntegration_Live_ValidateRealConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("live integration test pulls the catalog + core schema from GHCR; skipping under -short")
	}
	skipUnlessRegistry(t)

	moduleDir := filepath.Join(repoLibraryRoot(t), "testdata", "modules", "web_app")
	registry := flowRegistry()
	t.Setenv("CUE_REGISTRY", registry)

	k := kernel.New()
	ctx := context.Background()

	mod, err := k.AcquireModuleFromDir(ctx, moduleDir)
	require.NoErrorf(t, err, "acquiring module from %s", moduleDir)
	require.Equal(t, "web_app", mod.Metadata.Name)

	debugValues := mod.Package.LookupPath(schema.DebugValues)
	require.True(t, debugValues.Exists(), "web_app fixture must provide debugValues")
	// A Source carries bytes: render the module's own debugValues back to CUE
	// source, the way a frontend layering a debug overlay would hand them in.
	rendered, err := format.Node(debugValues.Syntax(cue.Final(), cue.Concrete(false)))
	require.NoError(t, err)
	src, err := k.LoadSourceFromBytes(moduleDir, rendered)
	require.NoError(t, err)

	out, err := k.ValidateConfigDetailed(mod.ConfigSchema(), []kernel.Source{src})
	require.NoError(t, err, "real debugValues must satisfy the real #config schema")
	assert.True(t, out.Exists())
}
