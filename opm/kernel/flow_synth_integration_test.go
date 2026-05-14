package kernel_test

import (
	"context"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/api"
	_ "github.com/open-platform-model/library/opm/api/v1alpha2"
	"github.com/open-platform-model/library/opm/apiversion"
	loader "github.com/open-platform-model/library/opm/helper/loader/file"
	"github.com/open-platform-model/library/opm/helper/synth"
	"github.com/open-platform-model/library/opm/kernel"
)

// TestFlow_WebApp_SynthPath_OnOpmPlatform mirrors TestFlow_WebApp_OnOpmPlatform
// but constructs the #ModuleRelease through Kernel.SynthesizeRelease instead
// of the hand-rolled CompileString + FillPath skeleton. It asserts the
// synthesized release matches the same transformers and renders the same
// Compiled-item shape as the file-loaded equivalent — pinning the synth
// helper as a drop-in replacement for the skeleton pattern.
//
// Skips under -short or when localhost:5000 is unreachable, matching the
// gating in the file-driven test.
func TestFlow_WebApp_SynthPath_OnOpmPlatform(t *testing.T) {
	if testing.Short() {
		t.Skip("flow integration test requires the local CUE registry; skipping under -short")
	}
	skipUnlessRegistry(t)

	libraryRoot := repoLibraryRoot(t)
	platformDir := filepath.Join(libraryRoot, "modules", "opm_platform")
	moduleDir := filepath.Join(libraryRoot, "testdata", "modules", "web_app")

	registry := "testing.opmodel.dev=localhost:5000+insecure,opmodel.dev=localhost:5000+insecure,registry.cue.works"
	t.Setenv("CUE_REGISTRY", registry)

	// Share synthKernel with the other synth tests in this package so the
	// v1alpha2 binding's SchemaValue cache lives in a single *cue.Context.
	// Crossing contexts would panic with "incompatible runtime" inside
	// synth.Release's scope unification.
	k := synthKernel
	ctx := context.Background()

	platVal, _, err := k.LoadPlatformPackage(ctx, platformDir, loader.LoadOptions{Registry: registry})
	require.NoErrorf(t, err, "loading platform from %s", platformDir)

	plat, err := k.NewPlatformFromValue(platVal)
	require.NoError(t, err)
	require.Equal(t, apiversion.V1alpha2, plat.APIVersion)

	modVal, modVer, err := k.LoadModulePackage(ctx, moduleDir, loader.LoadOptions{Registry: registry})
	require.NoErrorf(t, err, "loading module package from %s", moduleDir)
	require.Equal(t, apiversion.V1alpha2, modVer)

	mod, err := k.NewModuleFromValue(modVal)
	require.NoError(t, err)
	require.Equal(t, "web-app", mod.Metadata.Name)

	// Read debugValues off the module to use as the release's values payload —
	// same convenience pattern the CLI's `opm module test` workflow uses.
	binding, err := api.Lookup(mod.APIVersion)
	require.NoError(t, err)
	debugValues := modVal.LookupPath(binding.Paths().DebugValues)
	require.True(t, debugValues.Exists(), "web_app fixture must provide debugValues")

	// Build the release via SynthesizeRelease — the entire spec construction
	// (uuid, components fan-out, label stamping, opm-secrets discovery) flows
	// from schema unification rather than the hand-rolled skeleton used in
	// the parallel test.
	rel, err := k.SynthesizeRelease(ctx, synth.ReleaseInput{
		Module:    mod,
		Name:      "web-app-demo",
		Namespace: "default",
		Values:    debugValues,
	})
	require.NoErrorf(t, err, "synthesizing release")
	require.NotNil(t, rel)
	require.Equal(t, apiversion.V1alpha2, rel.APIVersion)
	require.Equal(t, "web-app-demo", rel.Metadata.Name)
	require.Equal(t, "default", rel.Metadata.Namespace)
	require.NotEmpty(t, rel.Metadata.UUID, "release UUID must be schema-derived")

	// The synthesized release must pass Match against the platform — every
	// component reaches at least one transformer.
	plan, err := k.Match(ctx, kernel.MatchInput{ModuleRelease: rel, Platform: plat})
	require.NoError(t, err)
	require.NotNil(t, plan)
	pairs := matchPairsToMap(plan.MatchedPairs())
	assert.Contains(t, pairs, "web", "web component must pair with at least one transformer")
	assertContainsFQN(t, pairs["web"],
		"opmodel.dev/modules/opm/transformers/deployment-transformer@v1",
		"web must match deployment-transformer (Container + workload-type=stateless)")
	assertContainsFQN(t, pairs["web"],
		"opmodel.dev/modules/opm/transformers/service-transformer@v1",
		"web must match service-transformer (Expose trait)")
	assert.Contains(t, pairs, "config",
		"config component must pair with configmap-transformer")
	assertContainsFQN(t, pairs["config"],
		"opmodel.dev/modules/opm/transformers/configmap-transformer@v1",
		"config must match configmap-transformer (ConfigMaps resource)")

	// The Compile output must include at least one Compiled item per
	// (component, transformer) pair surfaced by Match. This mirrors the
	// invariant the file-driven flow test pins.
	out, err := k.Compile(ctx, kernel.CompileInput{
		ModuleRelease: rel,
		Platform:      plat,
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
	assert.Contains(t, seenTransformers,
		"opmodel.dev/modules/opm/transformers/deployment-transformer@v1",
		"synth path must drive deployment-transformer")
	assert.Contains(t, seenTransformers,
		"opmodel.dev/modules/opm/transformers/service-transformer@v1",
		"synth path must drive service-transformer (Expose trait)")
	assert.Equal(t, 2, seenTransformers["opmodel.dev/modules/opm/transformers/configmap-transformer@v1"],
		"synth path must emit one Compiled per configmap entry (2 entries → 2 Compiled)")
}

// silence unused-import linter when cue is otherwise referenced only inside
// binding.Paths().DebugValues — keep the cue import explicit so future
// extensions can lookup additional paths without re-importing.
var _ = cue.Value{}
