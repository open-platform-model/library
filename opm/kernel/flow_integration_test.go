package kernel_test

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/compile"
	loader "github.com/open-platform-model/library/opm/helper/loader/file"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/schema"
)

// TestFlow_WebApp_OnOpmPlatform exercises the full Materialize → Match → Plan
// → Compile pipeline against the on-disk fixture pair:
//
//   - testdata/modules/web_app   (a core@v0 #Module consuming opm primitives:
//     Container resource, HttpRoute / Scaling / RestartPolicy / Expose traits,
//     StatelessWorkloadBlueprint)
//   - modules/opm_platform       (the canonical Kubernetes #Platform that
//     subscribes to opmodel.dev/catalogs/opm via a path-keyed #registry)
//
// The platform's subscription is materialized against the published catalog
// (opmodel.dev/catalogs/opm@0.1.0), then a #ModuleRelease is built from the
// module's debugValues and driven through Match / Plan / Compile. Transformer
// FQNs are asserted by substring so the test survives catalog version bumps.
//
// Re-enabled by enhancement 0001's library slice (the catalog repackage +
// #Subscription fixture restore). Skips under -short or when localhost:5000 is
// unreachable; OPM_FLOW_TEST_FORCE=1 turns the skip into a hard failure.
func TestFlow_WebApp_OnOpmPlatform(t *testing.T) {
	if testing.Short() {
		t.Skip("flow integration test requires the local CUE registry; skipping under -short")
	}
	skipUnlessRegistry(t)

	libraryRoot := repoLibraryRoot(t)
	platformDir := filepath.Join(libraryRoot, "modules", "opm_platform")
	moduleDir := filepath.Join(libraryRoot, "testdata", "modules", "web_app")

	registry := flowRegistry()
	t.Setenv("CUE_REGISTRY", registry)

	k := kernel.New()
	ctx := context.Background()

	// ── Load the consumer Module ─────────────────────────────────────
	modVal, err := k.LoadModulePackage(ctx, moduleDir, loader.LoadOptions{Registry: registry})
	require.NoErrorf(t, err, "loading module package from %s", moduleDir)

	mod, err := k.NewModuleFromValue(modVal)
	require.NoError(t, err, "constructing module.Module from CUE value")
	require.NotNil(t, mod)
	require.Equal(t, "web-app", mod.Metadata.Name)

	// ── Load + materialize the Platform ──────────────────────────────
	platVal, err := k.LoadPlatformPackage(ctx, platformDir, loader.LoadOptions{Registry: registry})
	require.NoErrorf(t, err, "loading platform from %s", platformDir)

	plat, err := k.NewPlatformFromValue(platVal)
	require.NoError(t, err, "constructing platform.Platform from CUE value")
	require.NotNil(t, plat)
	require.Equal(t, "kubernetes", plat.Metadata.Type)

	mp, err := k.Materialize(ctx, plat)
	require.NoError(t, err, "materializing platform against the published catalog")
	require.NotNil(t, mp)

	// ── Build a #ModuleRelease referencing the loaded Module ─────────
	//
	// The release skeleton is a plain CUE literal — CompileString has no
	// cue.mod context and cannot resolve registry-backed imports; the schema
	// constraints and the auto-fanned `components` field are filled below by
	// FillPath against the loaded module value. Mirrors cmd/flow-inspect.
	debugValues := modVal.LookupPath(schema.DebugValues)
	require.True(t, debugValues.Exists(), "web_app fixture must provide debugValues")

	releaseSkeleton := k.CueContext().CompileString(`
kind: "ModuleRelease"
metadata: {
	name:      "web-app-demo"
	namespace: "default"
	uuid:      "11111111-2222-5333-8444-555555555555"
}
`, cue.Filename("release.cue"))
	require.NoError(t, releaseSkeleton.Err(), "compiling release skeleton")

	unifiedModule := modVal.FillPath(schema.Config, debugValues)
	require.NoErrorf(t, unifiedModule.Err(), "filling values into module #config")
	moduleComponents := unifiedModule.LookupPath(cue.ParsePath("#components"))
	require.True(t, moduleComponents.Exists(), "module must expose #components after values are unified")

	releaseSpec := releaseSkeleton.
		FillPath(schema.Module, modVal).
		FillPath(schema.Values, debugValues).
		FillPath(schema.Components, moduleComponents)
	require.NoErrorf(t, releaseSpec.Err(), "building release spec from module value")

	rel, err := k.ProcessModuleRelease(ctx, releaseSpec, *mod, debugValues)
	require.NoError(t, err, "processing module release")
	require.NotNil(t, rel)
	require.Equal(t, "web-app-demo", rel.Metadata.Name)

	const runtimeName = "opm-test"

	// ── Phase 1: Match ───────────────────────────────────────────────
	t.Run("Match", func(t *testing.T) {
		plan, err := k.Match(ctx, kernel.MatchInput{ModuleRelease: rel, Platform: mp})
		require.NoError(t, err)
		require.NotNil(t, plan)

		gotPairs := matchPairsToMap(plan.MatchedPairs())

		// The deployment-transformer fires for the stateless web component
		// (Container resource + workload-type=stateless label gate); the
		// service-transformer fires because the web component carries the
		// Expose trait — pins the "two transformers pair on one component"
		// invariant.
		require.Contains(t, gotPairs, "web", "web component should pair with at least one transformer")
		assertContainsFQNSub(t, gotPairs["web"], "transformers/deployment-transformer@",
			"web should match deployment-transformer (Container + workload-type=stateless)")
		assertContainsFQNSub(t, gotPairs["web"], "transformers/service-transformer@",
			"web should also match service-transformer (Expose trait)")

		// config component carries a ConfigMaps resource → pairs with the
		// configmap-transformer (exercises the list-output renderer path).
		require.Contains(t, gotPairs, "config", "config component should pair with configmap-transformer")
		assertContainsFQNSub(t, gotPairs["config"], "transformers/configmap-transformer@",
			"config should match configmap-transformer (ConfigMaps resource)")

		assert.Empty(t, plan.Unmatched, "every component should match at least one transformer")
	})

	// ── Phase 2: Plan ────────────────────────────────────────────────
	t.Run("Plan", func(t *testing.T) {
		planResult, err := k.Plan(ctx, kernel.PlanInput{
			ModuleRelease: rel,
			Platform:      mp,
			RuntimeName:   runtimeName,
		})
		require.NoError(t, err)
		require.NotNil(t, planResult)

		assert.Empty(t, planResult.Unmatched, "every component should match at least one transformer")

		require.Len(t, planResult.Components, 2)
		byName := map[string]compile.ComponentSummary{}
		for _, c := range planResult.Components {
			byName[c.Name] = c
		}

		webSummary, ok := byName["web"]
		require.True(t, ok, "web component summary present")
		assert.Equal(t, "stateless", webSummary.Labels["core.opmodel.dev/workload-type"])
		assertContainsFQNSub(t, webSummary.ResourceFQNs, "resources/container@", "web declares the container resource")
		assertContainsFQNSub(t, webSummary.TraitFQNs, "traits/http-route@", "web declares the http-route trait")
		assertContainsFQNSub(t, webSummary.TraitFQNs, "traits/scaling@", "web declares the scaling trait")

		configSummary, ok := byName["config"]
		require.True(t, ok, "config component summary present")
		assertContainsFQNSub(t, configSummary.ResourceFQNs, "resources/config-maps@", "config declares the config-maps resource")
	})

	// ── Phase 3: Compile ─────────────────────────────────────────────
	t.Run("Compile", func(t *testing.T) {
		out, err := k.Compile(ctx, kernel.CompileInput{
			ModuleRelease: rel,
			Platform:      mp,
			RuntimeName:   runtimeName,
		})
		require.NoError(t, err)
		require.NotNil(t, out)
		require.NotNil(t, out.MatchPlan)
		assert.Empty(t, out.Unmatched)
		require.NotEmpty(t, out.Compiled, "compile must emit at least one rendered item")

		seenTransformers := map[string]int{}
		seenComponents := map[string]int{}
		for _, c := range out.Compiled {
			require.NotNil(t, c)
			require.NotEmpty(t, c.Component)
			require.NotEmpty(t, c.Transformer)
			seenComponents[c.Component]++
			seenTransformers[c.Transformer]++
		}

		// Both components fire transformers — web (deployment + service) and
		// config (configmap).
		assert.Equal(t, 2, len(seenComponents),
			"both web and config should fire transformers, got %v", seenComponents)
		assert.GreaterOrEqual(t, countFQNSub(seenTransformers, "transformers/deployment-transformer@"), 1,
			"deployment-transformer should produce at least one Compiled item")
		assert.GreaterOrEqual(t, countFQNSub(seenTransformers, "transformers/service-transformer@"), 1,
			"service-transformer should produce at least one Compiled item (Expose trait → Service)")

		// configmap-transformer emits N ConfigMaps per (component, transformer)
		// pair via its list output. The fixture's config component carries 2
		// configmap entries → 2 Compiled items from this single pair.
		assert.Equal(t, 2, countFQNSub(seenTransformers, "transformers/configmap-transformer@"),
			"configmap-transformer should emit one Compiled per configmap entry (2 entries → 2 Compiled)")
	})
}

// ── Shared helpers for the flow integration tests ────────────────────

// flowRegistry returns the CUE registry mapping the flow tests resolve imports
// through. It honors an externally-set CUE_REGISTRY (set by
// `task cue:test:flow` and CI), falling back to a split mapping that resolves
// the external core@v0 schema from GHCR while pulling the catalog and fixture
// modules from the local registry.
func flowRegistry() string {
	if v := os.Getenv("CUE_REGISTRY"); v != "" {
		return v
	}
	return "opmodel.dev/core=ghcr.io/open-platform-model,opmodel.dev=localhost:5000+insecure,registry.cue.works"
}

// matchPairsToMap groups MatchedPair entries by component name for ergonomic
// containment assertions.
func matchPairsToMap(pairs []compile.MatchedPair) map[string][]string {
	out := map[string][]string{}
	for _, p := range pairs {
		out[p.ComponentName] = append(out[p.ComponentName], p.TransformerFQN)
	}
	return out
}

// assertContainsFQNSub asserts that some element of got contains sub. FQNs
// carry the catalog SemVer (e.g. @0.1.0); substring matching keeps the
// assertions stable across version bumps.
func assertContainsFQNSub(t *testing.T, got []string, sub, msg string) {
	t.Helper()
	for _, g := range got {
		if strings.Contains(g, sub) {
			return
		}
	}
	assert.Failf(t, "missing FQN", "%s — no element contains %q; got: %v", msg, sub, got)
}

// countFQNSub sums the counts of entries in m whose key contains sub.
func countFQNSub(m map[string]int, sub string) int {
	n := 0
	for k, v := range m {
		if strings.Contains(k, sub) {
			n += v
		}
	}
	return n
}

// repoLibraryRoot resolves to the library/ directory regardless of where
// `go test` is invoked from.
func repoLibraryRoot(t *testing.T) string {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller failed")
	return filepath.Clean(filepath.Join(filepath.Dir(here), "..", ".."))
}

// skipUnlessRegistry calls t.Skip when localhost:5000 is unreachable, unless
// OPM_FLOW_TEST_FORCE=1 forces the test to run.
func skipUnlessRegistry(t *testing.T) {
	t.Helper()
	if os.Getenv("OPM_FLOW_TEST_FORCE") == "1" {
		return
	}
	conn, err := net.DialTimeout("tcp", "localhost:5000", 200*time.Millisecond)
	if err != nil {
		t.Skipf("local CUE registry not reachable on localhost:5000 (%v); start it via `task -d ../opm registry:start`", err)
	}
	_ = conn.Close()
}
