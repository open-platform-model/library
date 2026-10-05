package kernel_test

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/platform"
)

// managedByKey is the managed-by label key. It is spelled out here because the
// kernel, its tests included, may not import opm/k8s/labels (ADR-011 item 2).
const managedByKey = "app.kubernetes.io/managed-by"

// The two runtimes' managed-by values, core's #runtimeName for each frontend.
const (
	runtimeCLI        = "opm-cli"
	runtimeController = "opm-controller"
)

// TestRender_RuntimeNameReachesOnlyManagedBy is half of the cross-runtime
// render-digest parity proof (0012:D6:R2). It renders one instance as the cli
// and as the operator and shows that the two renders differ only in the
// managed-by label value. The other half is the opm/k8s/inventory render
// digest tests, which show that the digest ignores exactly that value. The
// fence keeps the two halves apart: a full render needs the in-process
// registry under opm/internal, which the tier may not import.
func TestRender_RuntimeNameReachesOnlyManagedBy(t *testing.T) {
	k := newRenderKernel(t)
	plat := acquireRenderPlatform(t, k, "platform")
	inst := acquireRenderInstance(t, k, "instance")

	assertRuntimeNameReachesOnlyManagedBy(t, k, inst, plat)
}

// TestRender_RuntimeNameReachesOnlyManagedBy_ShippedCatalog makes the same
// check on the catalog the frontends ship. Core hands #context.#runtimeName
// to every transformer, so whether the name reaches anything but the label is
// a property of the catalog: this case renders the parity instance against
// the published catalogs/opm build, gated like TestParity_ShippedCatalog.
func TestRender_RuntimeNameReachesOnlyManagedBy_ShippedCatalog(t *testing.T) {
	if testing.Short() {
		t.Skip("pulls the catalog + core schema from GHCR; skipping under -short")
	}
	skipUnlessRegistry(t)

	parityDir := filepath.Join(repoLibraryRoot(t), "testdata", "parity")
	registry := flowRegistry()
	t.Setenv("CUE_REGISTRY", registry)
	ctx := context.Background()

	k := kernel.New(kernel.WithRegistry(registry))
	inst, err := k.AcquireInstanceFromDir(ctx, filepath.Join(parityDir, "instance"))
	require.NoError(t, err, "acquiring the parity instance")
	plat, err := k.AcquirePlatformFromDir(ctx, filepath.Join(parityDir, "opm_platform"))
	require.NoError(t, err, "acquiring the opm_platform fixture")

	assertRuntimeNameReachesOnlyManagedBy(t, k, inst, plat)
}

// assertRuntimeNameReachesOnlyManagedBy renders inst against plat as each
// runtime and asserts that the two renders carry their own runtime name as the
// managed-by value and are otherwise equal, object by object.
func assertRuntimeNameReachesOnlyManagedBy(t *testing.T, k *kernel.Kernel, inst *module.Instance, plat *platform.Platform) {
	t.Helper()
	ctx := context.Background()

	cli, err := k.Render(ctx, kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: runtimeCLI})
	require.NoError(t, err, "rendering as %s", runtimeCLI)
	ctrl, err := k.Render(ctx, kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: runtimeController})
	require.NoError(t, err, "rendering as %s", runtimeController)

	require.NotEmpty(t, cli.Compiled, "the fixture renders objects")
	require.Len(t, ctrl.Compiled, len(cli.Compiled), "both runtimes render the same number of objects")

	for i := range cli.Compiled {
		a, b := cli.Compiled[i], ctrl.Compiled[i]
		require.Equal(t,
			[3]string{a.Instance, a.Component, a.Transformer},
			[3]string{b.Instance, b.Component, b.Transformer},
			"object %d: provenance is equal", i)

		objA := decodeCompiled(t, a)
		objB := decodeCompiled(t, b)
		assert.Equal(t, runtimeCLI, blankManagedBy(t, objA, i), "object %d: the cli render stamps its runtime name", i)
		assert.Equal(t, runtimeController, blankManagedBy(t, objB, i), "object %d: the operator render stamps its runtime name", i)
		assert.Equal(t, objA, objB, "object %d: with the managed-by value blanked, the two renders are equal", i)
	}
}

// decodeCompiled exports c's value to JSON and decodes it with every number
// kept as its literal.
func decodeCompiled(t *testing.T, c *kernel.Compiled) map[string]any {
	t.Helper()
	b, err := c.Value.MarshalJSON()
	require.NoError(t, err)
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var obj map[string]any
	require.NoError(t, dec.Decode(&obj))
	require.NotNil(t, obj)
	return obj
}

// blankManagedBy returns obj's managed-by label value and sets it to "".
func blankManagedBy(t *testing.T, obj map[string]any, i int) string {
	t.Helper()
	meta, ok := obj["metadata"].(map[string]any)
	require.True(t, ok, "object %d: metadata is an object", i)
	lbls, ok := meta["labels"].(map[string]any)
	require.True(t, ok, "object %d: metadata.labels is an object", i)
	v, ok := lbls[managedByKey].(string)
	require.True(t, ok, "object %d: carries a string managed-by label", i)
	lbls[managedByKey] = ""
	return v
}
