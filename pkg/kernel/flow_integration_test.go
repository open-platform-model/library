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

	"github.com/open-platform-model/library/pkg/api"
	_ "github.com/open-platform-model/library/pkg/api/v1alpha2"
	"github.com/open-platform-model/library/pkg/apiversion"
	"github.com/open-platform-model/library/pkg/compile"
	loader "github.com/open-platform-model/library/pkg/helper/loader/file"
	"github.com/open-platform-model/library/pkg/kernel"
)

// TestFlow_WebApp_OnOpmPlatform exercises the full Plan → Match → Compile
// pipeline against the on-disk fixture pair:
//
//   - testdata/modules/web_app   (a v1alpha2 #Module that consumes opm
//     primitives — Container resource, HttpRoute / Scaling / RestartPolicy
//     traits, StatelessWorkloadBlueprint)
//   - modules/opm_platform       (the canonical Kubernetes #Platform that
//     registers opmodel.dev/modules/opm via #registry)
//
// The test runs three phases against a single (Module, Platform) pair so a
// regression in any phase pinpoints the failure surface:
//
//  1. Match   — kernel.Match returns a MatchPlan; assert that the `web`
//     component pairs with deployment-transformer (label gate
//     workload-type=stateless picks it over the other workload
//     transformers) and that http-route surfaces as an unhandled-
//     trait warning (no transformer registered for it in opm).
//  2. Plan    — kernel.Plan re-runs Validate + Match + Execute (dry-run)
//     and exposes the same diagnostics plus per-component
//     summaries.
//  3. Compile — kernel.Compile renders the matched pairs via #transform.
//     Assert per-pair Compiled provenance (Component, Transformer);
//     the renderer emits one Compiled per pair, with Compiled.Value
//     carrying the transformer's #transform.output verbatim.
//
// The fixture imports resolve through the local OPM registry (publish state
// recorded in cue-versions.yml). The test skips with a clear message when
// localhost:5000 is unreachable, so a fresh checkout without the registry
// running fails loudly only at intentional invocation (`task cue:test:flow`)
// rather than as a noisy default `go test ./...` failure.
func TestFlow_WebApp_OnOpmPlatform(t *testing.T) {
	if testing.Short() {
		t.Skip("flow integration test requires the local CUE registry; skipping under -short")
	}
	skipUnlessRegistry(t)

	// Resolve fixture paths relative to this source file so the test is
	// CWD-independent (mirrors schemaModuleRoot in schema_fixture_test.go).
	libraryRoot := repoLibraryRoot(t)
	platformDir := filepath.Join(libraryRoot, "modules", "opm_platform")
	moduleDir := filepath.Join(libraryRoot, "testdata", "modules", "web_app")

	registry := "testing.opmodel.dev=localhost:5000+insecure,opmodel.dev=localhost:5000+insecure,registry.cue.works"

	k := kernel.New()
	ctx := context.Background()

	// ── Load the platform ────────────────────────────────────────────
	platVal, _, err := k.LoadPlatformFile(ctx, platformDir, loader.LoadOptions{Registry: registry})
	require.NoErrorf(t, err, "loading platform from %s", platformDir)

	plat, err := k.NewPlatformFromValue(platVal)
	require.NoError(t, err, "constructing platform.Platform from CUE value")
	require.NotNil(t, plat)
	require.Equal(t, apiversion.V1alpha2, plat.APIVersion)
	require.Equal(t, "kubernetes", plat.Metadata.Type)

	// ── Load the consumer Module ─────────────────────────────────────
	modVal, modVer, err := k.LoadModulePackage(ctx, moduleDir, loader.LoadOptions{Registry: registry})
	require.NoErrorf(t, err, "loading module package from %s", moduleDir)
	require.Equal(t, apiversion.V1alpha2, modVer)

	mod, err := k.NewModuleFromValue(modVal)
	require.NoError(t, err, "constructing module.Module from CUE value")
	require.NotNil(t, mod)
	require.Equal(t, "web-app", mod.Metadata.Name)

	// ── Build a #ModuleRelease referencing the loaded Module ─────────
	//
	// The release skeleton is concrete except for the embedded #module + values
	// fields, which we fill from the loaded module. ProcessModuleRelease then
	// validates values against #module.#config, fills them, and verifies the
	// result is fully concrete — exactly the path the CLI takes for a real
	// release.
	// Release skeleton is a plain CUE literal — no schema import. CompileString
	// has no cue.mod context and cannot resolve registry-backed imports; the
	// schema constraints (and the auto-fanned `components` field) are filled
	// in below by FillPath against the loaded module value.
	//
	// The metadata.uuid field is hard-coded to a valid v5 UUID so the
	// release passes the v1alpha2 #UUIDType regex without the schema's
	// uuid.SHA1 derivation (which our skeleton bypasses by not unifying
	// with #ModuleRelease). Any stable UUID works — the renderer only
	// stamps it onto output via #context.#moduleReleaseMetadata.uuid.
	releaseSkeleton := k.CueContext().CompileString(`
apiVersion: "opmodel.dev/v1alpha2"
kind:       "ModuleRelease"
metadata: {
	name:      "web-app-demo"
	namespace: "default"
	uuid:      "11111111-2222-5333-8444-555555555555"
}
`, cue.Filename("release.cue"))
	require.NoError(t, releaseSkeleton.Err(), "compiling release skeleton")

	binding, err := api.Lookup(modVer)
	require.NoError(t, err)
	paths := binding.Paths()

	// Read the module's debugValues and use them as the values payload — same
	// convenience the CLI's `opm module test` workflow uses when no overlay
	// is supplied.
	debugValues := modVal.LookupPath(paths.DebugValues)
	require.True(t, debugValues.Exists(), "web_app fixture must provide debugValues")

	// Fan components out of the unified module (#config := values). The
	// #ModuleRelease schema does this via a CUE comprehension when the
	// release is loaded against the schema; a CompileString skeleton cannot
	// see the schema, so we replicate the comprehension manually. web_app
	// declares no #Secret instances, so the auto-secrets branch in the
	// schema's components projection is a no-op for this fixture and skipping
	// it here matches the schema-driven result exactly.
	unifiedModule := modVal.FillPath(paths.Config, debugValues)
	require.NoErrorf(t, unifiedModule.Err(), "filling values into module #config")
	moduleComponents := unifiedModule.LookupPath(cue.ParsePath("#components"))
	require.True(t, moduleComponents.Exists(), "module must expose #components after values are unified")

	releaseSpec := releaseSkeleton.
		FillPath(paths.Module, modVal).
		FillPath(paths.Values, debugValues).
		FillPath(paths.Components, moduleComponents)
	require.NoErrorf(t, releaseSpec.Err(), "building release spec from module value")

	rel, err := k.ProcessModuleRelease(ctx, releaseSpec, *mod, debugValues)
	require.NoError(t, err, "processing module release")
	require.NotNil(t, rel)
	require.Equal(t, apiversion.V1alpha2, rel.APIVersion)
	require.Equal(t, "web-app-demo", rel.Metadata.Name)

	const runtimeName = "opm-test"

	// ── Phase 1: Match ───────────────────────────────────────────────
	t.Run("Match", func(t *testing.T) {
		plan, err := k.Match(ctx, kernel.MatchInput{ModuleRelease: rel, Platform: plat})
		require.NoError(t, err)
		require.NotNil(t, plan)

		pairs := plan.MatchedPairs()
		gotPairs := matchPairsToMap(pairs)

		// The deployment-transformer fires for the stateless web component
		// (Container resource + workload-type=stateless label gate). Other
		// container-using transformers (statefulset / daemonset / job /
		// cronjob) are filtered out because their workload-type label gates
		// don't match. The service-transformer also fires because the web
		// component carries the Expose trait — pins the "two transformers
		// pair on one component" invariant the matcher now supports.
		assert.Contains(t, gotPairs, "web",
			"web component should pair with at least one transformer")
		webMatched := gotPairs["web"]
		assertContainsFQN(t, webMatched,
			"opmodel.dev/modules/opm/transformers/deployment-transformer@v1",
			"web should match deployment-transformer (Container + workload-type=stateless)")
		assertContainsFQN(t, webMatched,
			"opmodel.dev/modules/opm/transformers/service-transformer@v1",
			"web should also match service-transformer (Expose trait)")

		// config component carries a ConfigMaps resource → pairs with the
		// configmap-transformer. Exercises the list-output renderer path
		// (N Compiled per pair) in addition to web's struct-output path.
		assert.Contains(t, gotPairs, "config",
			"config component should pair with configmap-transformer")
		assertContainsFQN(t, gotPairs["config"],
			"opmodel.dev/modules/opm/transformers/configmap-transformer@v1",
			"config should match configmap-transformer (ConfigMaps resource)")

		// http-route trait has no transformer registered in opm — should
		// surface as an unhandled-trait warning. Keeps the unhandled path
		// covered by the integration suite.
		unhandled := plan.UnhandledTraits["web"]
		assert.Contains(t, unhandled,
			"opmodel.dev/modules/opm/traits/http-route@v1",
			"http-route should be reported as unhandled (no registered transformer)")

	})

	// ── Phase 2: Plan ────────────────────────────────────────────────
	t.Run("Plan", func(t *testing.T) {
		planResult, err := k.Plan(ctx, kernel.PlanInput{
			ModuleRelease: rel,
			Platform:      plat,
			RuntimeName:   runtimeName,
		})
		require.NoError(t, err)
		require.NotNil(t, planResult)

		assert.Empty(t, planResult.Unmatched, "every component should match at least one transformer")

		// Component summary should reflect the on-disk fixture's primitives.
		// Two components: web (stateless workload + HTTP + Expose) and
		// config (ConfigMaps).
		require.Len(t, planResult.Components, 2)
		byName := map[string]compile.ComponentSummary{}
		for _, c := range planResult.Components {
			byName[c.Name] = c
		}

		webSummary, ok := byName["web"]
		require.True(t, ok, "web component summary present")
		assert.Equal(t, "stateless", webSummary.Labels["core.opmodel.dev/workload-type"])
		assert.Contains(t, webSummary.ResourceFQNs,
			"opmodel.dev/modules/opm/resources/container@v1")
		assert.Contains(t, webSummary.TraitFQNs,
			"opmodel.dev/modules/opm/traits/http-route@v1")
		assert.Contains(t, webSummary.TraitFQNs,
			"opmodel.dev/modules/opm/traits/scaling@v1")

		configSummary, ok := byName["config"]
		require.True(t, ok, "config component summary present")
		assert.Contains(t, configSummary.ResourceFQNs,
			"opmodel.dev/modules/opm/resources/config-maps@v1")

		// At least one warning — the http-route unhandled-trait advisory.
		joinedWarnings := strings.Join(planResult.Warnings, "\n")
		assert.Contains(t, joinedWarnings, "http-route@v1",
			"plan warnings should mention the unhandled http-route trait")
	})

	// ── Phase 3: Compile ─────────────────────────────────────────────
	t.Run("Compile", func(t *testing.T) {
		out, err := k.Compile(ctx, kernel.CompileInput{
			ModuleRelease: rel,
			Platform:      plat,
			RuntimeName:   runtimeName,
		})
		require.NoError(t, err)
		require.NotNil(t, out)

		// Compile MUST surface the same MatchPlan as the standalone Match
		// phase — the kernel's Plan/Compile share execution paths so
		// upstream invariants flow through.
		require.NotNil(t, out.MatchPlan)
		assert.Empty(t, out.Unmatched)

		// Provenance assertions: every Compiled item must carry the source
		// component + transformer. We don't pin the exact rendered shape
		// because the opm transformers' output (k8sappsv1.#Deployment,
		// k8scorev1.#Service) is itself the contract under integration —
		// Compiled[].Value is the rendered K8s manifest verbatim.
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

		// Both components fire transformers — web (deployment + service)
		// and config (configmap).
		assert.Equal(t, 2, len(seenComponents),
			"both web and config should fire transformers, got %v", seenComponents)
		assert.Contains(t, seenTransformers,
			"opmodel.dev/modules/opm/transformers/deployment-transformer@v1",
			"deployment-transformer should produce at least one Compiled item")
		assert.Contains(t, seenTransformers,
			"opmodel.dev/modules/opm/transformers/service-transformer@v1",
			"service-transformer should produce at least one Compiled item (Expose trait → Service)")

		// configmap-transformer emits N ConfigMaps per (component,
		// transformer) pair via its list output. The fixture's config
		// component carries 2 configmap entries → 2 Compiled items from
		// this single pair. Pins the list-output dispatch path.
		assert.Equal(t, 2, seenTransformers["opmodel.dev/modules/opm/transformers/configmap-transformer@v1"],
			"configmap-transformer should emit one Compiled per configmap entry (2 entries → 2 Compiled)")
	})
}

// matchPairsToMap groups MatchedPair entries by component name for ergonomic
// containment assertions in the Match subtest.
func matchPairsToMap(pairs []compile.MatchedPair) map[string][]string {
	out := map[string][]string{}
	for _, p := range pairs {
		out[p.ComponentName] = append(out[p.ComponentName], p.TransformerFQN)
	}
	return out
}

// repoLibraryRoot resolves to the library/ directory regardless of where
// `go test` is invoked from. Mirrors schemaModuleRoot in schema_fixture_test.go.
func repoLibraryRoot(t *testing.T) string {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller failed")
	return filepath.Clean(filepath.Join(filepath.Dir(here), "..", ".."))
}

// skipUnlessRegistry calls t.Skip when localhost:5000 (the canonical OPM
// local registry from CLAUDE.md) is unreachable. Keeps the test usable on
// fresh checkouts without forcing every contributor to start the registry.
func skipUnlessRegistry(t *testing.T) {
	t.Helper()
	if v := os.Getenv("OPM_FLOW_TEST_FORCE"); v == "1" {
		return
	}
	conn, err := net.DialTimeout("tcp", "localhost:5000", 200*time.Millisecond)
	if err != nil {
		t.Skipf("local CUE registry not reachable on localhost:5000 (%v); start it via `task -d ../opm registry:start`", err)
	}
	_ = conn.Close()
}

// assertContainsFQN checks whether a slice of transformer FQNs contains
// expected. Uses assert.Contains for the actual check but keeps the wrapper
// so the failure message names which assertion fired.
func assertContainsFQN(t *testing.T, got []string, expected, msg string) {
	t.Helper()
	assert.Containsf(t, got, expected, "%s — got: %v", msg, got)
}
