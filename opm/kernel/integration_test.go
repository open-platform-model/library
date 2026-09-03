package kernel_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loaderfile "github.com/open-platform-model/library/opm/helper/loader/file"
	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/kernel"
)

// This is the always-on, fully hermetic integration harness. It drives the
// public Kernel API (AcquireModuleFromRegistry → SynthesizeInstance →
// AcquirePlatformFromDir → Render) against in-memory catalogs and modules,
// with no localhost:5000 dependency. The live, real-catalog flow lives in
// flow_integration_test.go (gated by skipUnlessRegistry); every matching
// and execution outcome (unresolved demands, disqualified candidates, trait
// postures, failing and incomplete pairs, the single-provider guard) is
// exercised by render_test.go against the committed testdata/render
// fixtures.

func TestIntegration_Render(t *testing.T) {
	const version = "0.1.0"
	catPath := registrytest.UniquePath(t, "cat")
	modPath := registrytest.UniquePath(t, "modules") + "/two_app"
	mapping := registrytest.NewModuleRegistry(t,
		[]registrytest.ModuleFixture{{Path: modPath, Version: version, File: twoComponentModuleFile(modPath, catPath, version)}},
		[]registrytest.CatalogFixture{standardCatalog(catPath, version)},
	)
	k := kernel.New(kernel.WithRegistry(mapping))
	inst := synthesizeInstance(t, k, modPath, version, "demo")
	plat := acquireCatalogPlatform(t, k, mapping, catPath, version)

	res, err := k.Render(context.Background(), kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
	require.NoError(t, err)

	t.Run("pairs both components", func(t *testing.T) {
		pairs := pairsByComponent(res.Diagnostics.Pairs)
		assertContainsFQNSub(t, pairs["web"], "transformers/deployment@", "web → deployment")
		assertContainsFQNSub(t, pairs["config"], "transformers/configmap@", "config → configmap")
		assert.Empty(t, res.Diagnostics.Unmatched)
		assert.Empty(t, res.Diagnostics.Unresolved)
		assert.Empty(t, res.Diagnostics.Unify)
	})

	t.Run("dispatches struct and list outputs", func(t *testing.T) {
		perComp := map[string]int{}
		for _, c := range res.Compiled {
			assert.Equal(t, "demo", c.Instance)
			perComp[c.Component]++
		}
		assert.Equal(t, 1, perComp["web"], "struct output → one Compiled")
		assert.Equal(t, 2, perComp["config"], "two-element list output → two Compiled")
	})
}

// TestIntegration_Render_PinnedCatalogBuildExecutes pins 0010 D14 on the
// D5 shape: the platform module's cue.mod names the exact catalog build the
// render executes, not the highest published one.
func TestIntegration_Render_PinnedCatalogBuildExecutes(t *testing.T) {
	catPath := registrytest.UniquePath(t, "cat")
	modPath := registrytest.UniquePath(t, "modules") + "/two_app"
	mapping := registrytest.NewModuleRegistry(t,
		[]registrytest.ModuleFixture{{Path: modPath, Version: "0.1.0", File: twoComponentModuleFile(modPath, catPath, "0.1.0")}},
		[]registrytest.CatalogFixture{standardCatalog(catPath, "0.1.0"), standardCatalog(catPath, "0.2.0")},
	)
	k := kernel.New(kernel.WithRegistry(mapping))
	inst := synthesizeInstance(t, k, modPath, "0.1.0", "demo")
	plat := acquireCatalogPlatform(t, k, mapping, catPath, "0.1.0")

	res, err := k.Render(context.Background(), kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
	require.NoError(t, err)
	require.NotEmpty(t, res.Diagnostics.Pairs)
	for _, p := range res.Diagnostics.Pairs {
		assert.True(t, strings.HasSuffix(p.Transformer, "@0.1.0"), "the pinned build executed, not the newest published: %s", p.Transformer)
	}
}

// TestIntegration_Render_UnpublishedCatalogFailsAtAcquire pins where an
// unresolvable catalog surfaces on the D5 shape: the platform module's
// import does not resolve, so acquisition fails naming the path and no
// render is attempted.
func TestIntegration_Render_UnpublishedCatalogFailsAtAcquire(t *testing.T) {
	published := registrytest.UniquePath(t, "cat")
	missing := registrytest.UniquePath(t, "missing")
	k, mapping := newKernelWithCatalogs(t, standardCatalog(published, "0.1.0"))

	platDir := writeCatalogPlatform(t, t.TempDir(), missing, "0.1.0")
	plat, err := k.AcquirePlatformFromDir(context.Background(), platDir, loaderfile.LoadOptions{Registry: mapping})
	require.Error(t, err)
	assert.Nil(t, plat)
	assert.Contains(t, err.Error(), missing, "the failure names the catalog path the platform imports")
	assert.Contains(t, err.Error(), filepath.Base(platDir))
}
