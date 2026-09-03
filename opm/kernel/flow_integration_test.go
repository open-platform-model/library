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

	loader "github.com/open-platform-model/library/opm/helper/loader/file"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/schema"
)

// TestFlow_WebApp_OnOpmPlatform exercises the render path end to end against
// the on-disk fixture pair:
//
//   - testdata/modules/web_app   (a core@v2 #Module consuming opm primitives
//     from the consolidated catalogs/opm v4 line: Container resource,
//     HttpRoute / Scaling / RestartPolicy / Expose traits, StatelessWorkload
//     blueprint via D49 versioned imports), with its import-authored
//     #ModuleInstance package under testdata/modules/web_app/instance
//   - modules/opm_platform       (the canonical Kubernetes #Platform module
//     importing the major-suffixed opmodel.dev/catalogs/opm v4 line through
//     a #CatalogEntry-form #registry, 0019 D5)
//
// Both are acquired source-carrying and rendered through Kernel.Render as one
// build; the catalog and core resolve from GHCR. Transformer FQNs are
// asserted by substring so the test survives catalog version bumps.
//
// Skips under -short or when GHCR is unreachable; OPM_FLOW_TEST_FORCE=1
// turns the skip into a hard failure.
func TestFlow_WebApp_OnOpmPlatform(t *testing.T) {
	if testing.Short() {
		t.Skip("flow integration test pulls the catalog + core schema from GHCR; skipping under -short")
	}
	skipUnlessRegistry(t)

	libraryRoot := repoLibraryRoot(t)
	platformDir := filepath.Join(libraryRoot, "modules", "opm_platform")
	moduleDir := filepath.Join(libraryRoot, "testdata", "modules", "web_app")

	registry := flowRegistry()
	t.Setenv("CUE_REGISTRY", registry)

	k := kernel.New(kernel.WithRegistry(registry))
	ctx := context.Background()
	opts := loader.LoadOptions{Registry: registry}

	// ── The consumer Module, for its identity ────────────────────────
	modVal, err := k.LoadModulePackage(ctx, moduleDir, opts)
	require.NoErrorf(t, err, "loading module package from %s", moduleDir)
	mod, err := k.NewModuleFromValue(modVal)
	require.NoError(t, err, "constructing module.Module from CUE value")
	require.Equal(t, "web_app", mod.Metadata.Name)

	// ── Acquire the Platform module ──────────────────────────────────
	plat, err := k.AcquirePlatformFromDir(ctx, platformDir, opts)
	require.NoErrorf(t, err, "acquiring platform module from %s", platformDir)
	require.Equal(t, "kubernetes", plat.Metadata.Type)
	require.NotNil(t, plat.Source)

	// ── Acquire the #ModuleInstance ──────────────────────────────────
	//
	// The instance is an import-authored package inside the fixture module
	// (testdata/modules/web_app/instance): it names the module by import,
	// so every component's #instance and #names resolve and core derives
	// metadata.uuid from the instance fqn (0019 D3).
	inst, err := k.AcquireInstanceFromDir(ctx, filepath.Join(moduleDir, "instance"), opts)
	require.NoErrorf(t, err, "acquiring instance package from %s", moduleDir)
	require.Equal(t, "web-app-demo", inst.Metadata.Name)
	require.Equal(t, "instance", inst.Source.Pkg, "the instance package sits inside the module fixture's root")

	// The processed instance resolves what the old LookupPath+FillPath
	// skeleton severed: the computed names and the derived uuid.
	fqdn := inst.Components().LookupPath(cue.ParsePath("web.#names.dns.fqdn"))
	require.NoError(t, fqdn.Err(), "web.#names.dns.fqdn must resolve on the processed instance")
	fqdnStr, err := fqdn.String()
	require.NoError(t, err)
	assert.Equal(t, "web-app-demo-web.default.svc.cluster.local", fqdnStr)
	// The v5 uuid core derives from the instance fqn; the parity oracle
	// derives the same value for the same name and namespace
	// (testdata/parity/instance), which is what makes the two fixtures'
	// rendered names directly comparable.
	assert.Equal(t, "bf5b9c54-bf4a-5cad-8cb7-77d4d526a16a", inst.Metadata.UUID,
		"metadata.uuid must be the v5 uuid core derives from the instance fqn")

	// ── Render ───────────────────────────────────────────────────────
	res, err := k.Render(ctx, kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "opm-test"})
	require.NoError(t, err, "rendering the web_app instance against the opm platform")
	require.NotEmpty(t, res.Compiled, "render must emit at least one object")

	t.Run("pairs", func(t *testing.T) {
		gotPairs := pairsByComponent(res.Diagnostics.Pairs)

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
		// configmap-transformer (exercises the list-output path).
		require.Contains(t, gotPairs, "config", "config component should pair with configmap-transformer")
		assertContainsFQNSub(t, gotPairs["config"], "transformers/configmap-transformer@",
			"config should match configmap-transformer (ConfigMaps resource)")

		assert.Empty(t, res.Diagnostics.Unmatched, "every component should match at least one transformer")
		assert.Empty(t, res.Diagnostics.Unresolved)
	})

	t.Run("rendered objects", func(t *testing.T) {
		seenTransformers := map[string]int{}
		seenComponents := map[string]int{}
		for _, c := range res.Compiled {
			require.NotNil(t, c)
			require.NotEmpty(t, c.Component)
			require.NotEmpty(t, c.Transformer)
			assert.Equal(t, "web-app-demo", c.Instance)
			seenComponents[c.Component]++
			seenTransformers[c.Transformer]++
		}

		// Both components fire transformers — web (deployment + service) and
		// config (configmap).
		assert.Equal(t, 2, len(seenComponents),
			"both web and config should fire transformers, got %v", seenComponents)
		assert.GreaterOrEqual(t, countFQNSub(seenTransformers, "transformers/deployment-transformer@"), 1,
			"deployment-transformer should produce at least one object")
		assert.GreaterOrEqual(t, countFQNSub(seenTransformers, "transformers/service-transformer@"), 1,
			"service-transformer should produce at least one object (Expose trait → Service)")

		// configmap-transformer emits N ConfigMaps per (component, transformer)
		// pair via its list output. The fixture's config component carries 2
		// configmap entries → 2 objects from this single pair.
		assert.Equal(t, 2, countFQNSub(seenTransformers, "transformers/configmap-transformer@"),
			"configmap-transformer should emit one object per configmap entry (2 entries → 2 objects)")
	})

	t.Run("resolved versions", func(t *testing.T) {
		// The instance module and the platform module pin the same catalog
		// build (D18: rows, not warnings).
		assert.Empty(t, res.Warnings)
		var catalogRow *kernel.ResolvedVersion
		for i := range res.Diagnostics.ResolvedVersions {
			if res.Diagnostics.ResolvedVersions[i].Path == "opmodel.dev/catalogs/opm@v4" {
				catalogRow = &res.Diagnostics.ResolvedVersions[i]
			}
		}
		require.NotNil(t, catalogRow, "the catalog the instance module requires is a row")
		assert.Equal(t, catalogRow.PlatformVersion, catalogRow.ModuleVersion)
		assert.False(t, catalogRow.Newer)
	})
}

// ── Shared helpers for the flow integration tests ────────────────────

// flowRegistry returns the CUE registry mapping the flow tests resolve imports
// through. It honors an externally-set CUE_REGISTRY (set by
// `task cue:test:flow` and CI), falling back to schema.PublicRegistry, which
// resolves the whole opmodel.dev prefix — the core@v2 schema *and* the
// opmodel.dev/catalogs/opm catalog — from GHCR, with cue.dev/x/k8s.io falling
// through to registry.cue.works. Pulling the catalog from GHCR (rather than a
// laptop-only localhost:5000) is what lets these tests run in CI.
func flowRegistry() string {
	if v := os.Getenv("CUE_REGISTRY"); v != "" {
		return v
	}
	return schema.PublicRegistry
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

// skipUnlessRegistry calls t.Skip when GHCR is unreachable, unless
// OPM_FLOW_TEST_FORCE=1 forces the test to run. The flow tests resolve the
// catalog and core schema from ghcr.io (see flowRegistry); probing it keeps an
// offline `go test ./...` graceful (skip, not a multi-second pull timeout)
// while CI — which sets OPM_FLOW_TEST_FORCE=1 — always runs the flow.
func skipUnlessRegistry(t *testing.T) {
	t.Helper()
	if os.Getenv("OPM_FLOW_TEST_FORCE") == "1" {
		return
	}
	conn, err := net.DialTimeout("tcp", "ghcr.io:443", 500*time.Millisecond)
	if err != nil {
		t.Skipf("GHCR not reachable (%v); the flow test pulls the catalog + core schema from ghcr.io. Set OPM_FLOW_TEST_FORCE=1 to require it", err)
	}
	_ = conn.Close()
}
