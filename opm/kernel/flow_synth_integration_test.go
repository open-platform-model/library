package kernel_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loader "github.com/open-platform-model/library/opm/helper/loader/file"
	"github.com/open-platform-model/library/opm/helper/synth"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/schema"
)

// TestFlow_WebApp_SynthPath_OnOpmPlatform mirrors TestFlow_WebApp_OnOpmPlatform
// but constructs the #ModuleRelease through Kernel.SynthesizeRelease instead of
// the hand-rolled CompileString + FillPath skeleton. It asserts the synthesized
// release matches the same transformers and renders the same Compiled-item
// shape as the file-loaded equivalent — pinning the synth helper as a drop-in
// replacement for the skeleton pattern.
//
// Skips under -short or when GHCR is unreachable, matching the gating in the
// file-driven test.
func TestFlow_WebApp_SynthPath_OnOpmPlatform(t *testing.T) {
	if testing.Short() {
		t.Skip("flow integration test pulls the catalog + core schema from GHCR; skipping under -short")
	}
	skipUnlessRegistry(t)

	libraryRoot := repoLibraryRoot(t)
	platformDir := filepath.Join(libraryRoot, "modules", "opm_platform")
	moduleDir := filepath.Join(libraryRoot, "testdata", "modules", "web_app")

	registry := flowRegistry()
	t.Setenv("CUE_REGISTRY", registry)

	k := kernel.New()
	ctx := context.Background()

	// ── Load + materialize the Platform ──────────────────────────────
	platVal, err := k.LoadPlatformPackage(ctx, platformDir, loader.LoadOptions{Registry: registry})
	require.NoErrorf(t, err, "loading platform from %s", platformDir)

	plat, err := k.NewPlatformFromValue(platVal)
	require.NoError(t, err)
	require.Equal(t, "kubernetes", plat.Metadata.Type)

	mp, err := k.Materialize(ctx, plat)
	require.NoError(t, err, "materializing platform against the published catalog")
	require.NotNil(t, mp)

	// ── Load the consumer Module ─────────────────────────────────────
	modVal, err := k.LoadModulePackage(ctx, moduleDir, loader.LoadOptions{Registry: registry})
	require.NoErrorf(t, err, "loading module package from %s", moduleDir)

	mod, err := k.NewModuleFromValue(modVal)
	require.NoError(t, err)
	require.Equal(t, "web-app", mod.Metadata.Name)

	// Read debugValues off the module to use as the release's values payload.
	debugValues := modVal.LookupPath(schema.DebugValues)
	require.True(t, debugValues.Exists(), "web_app fixture must provide debugValues")

	// Build the release via SynthesizeRelease — the entire spec construction
	// (uuid, components fan-out, label stamping) flows from schema unification
	// rather than the hand-rolled skeleton used in the parallel test.
	rel, err := k.SynthesizeRelease(ctx, synth.ReleaseInput{
		Module:    mod,
		Name:      "web-app-demo",
		Namespace: "default",
		Values:    debugValues,
	})
	require.NoErrorf(t, err, "synthesizing release")
	require.NotNil(t, rel)
	require.Equal(t, "web-app-demo", rel.Metadata.Name)
	require.Equal(t, "default", rel.Metadata.Namespace)
	require.NotEmpty(t, rel.Metadata.UUID, "release UUID must be schema-derived")

	// The synthesized release must pass Match against the materialized platform
	// — every component reaches at least one transformer.
	plan, err := k.Match(ctx, kernel.MatchInput{ModuleRelease: rel, Platform: mp})
	require.NoError(t, err)
	require.NotNil(t, plan)
	pairs := matchPairsToMap(plan.MatchedPairs())
	require.Contains(t, pairs, "web", "web component must pair with at least one transformer")
	assertContainsFQNSub(t, pairs["web"], "transformers/deployment-transformer@",
		"web must match deployment-transformer (Container + workload-type=stateless)")
	assertContainsFQNSub(t, pairs["web"], "transformers/service-transformer@",
		"web must match service-transformer (Expose trait)")
	require.Contains(t, pairs, "config", "config component must pair with configmap-transformer")
	assertContainsFQNSub(t, pairs["config"], "transformers/configmap-transformer@",
		"config must match configmap-transformer (ConfigMaps resource)")

	// The Compile output must include at least one Compiled item per
	// (component, transformer) pair surfaced by Match.
	out, err := k.Compile(ctx, kernel.CompileInput{
		ModuleRelease: rel,
		Platform:      mp,
		RuntimeName:   "opm-test",
	})
	require.NoError(t, err)
	require.NotNil(t, out)
	require.NotEmpty(t, out.Compiled,
		"compile must emit at least one rendered item from the synthesized release")

	seenTransformers := map[string]int{}
	seenComponents := map[string]int{}
	for _, c := range out.Compiled {
		require.NotNil(t, c)
		require.NotEmpty(t, c.Component)
		require.NotEmpty(t, c.Transformer)
		seenComponents[c.Component]++
		seenTransformers[c.Transformer]++
	}
	assert.Equal(t, 2, len(seenComponents),
		"both web and config should fire transformers from the synthesized release, got %v", seenComponents)
	assert.Equal(t, 2, countFQNSub(seenTransformers, "transformers/configmap-transformer@"),
		"configmap-transformer should emit one Compiled per configmap entry (2 entries → 2 Compiled)")
}
