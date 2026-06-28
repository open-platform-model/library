package kernel_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/helper/synth"
	"github.com/open-platform-model/library/opm/kernel"
)

// TestFlow_Redis_CatalogSubpackage_Regression is the FAITHFUL library#31
// regression guard, run against a real registry (the hermetic registrytest
// substrate cannot reproduce #31: its in-memory resolver walks a dependency's
// cue.mod/module.cue transitively, whereas the real modconfig resolver does NOT
// — the main module must itself declare the transitive closure, which is exactly
// the gap the old fabricated-{core, module} synth module hit).
//
// `opmodel.dev/modules/test/redis@v0.1.6` is a real published module whose
// SOURCE imports catalog subpackages it does not re-declare at the top level:
//
//	import bp "opmodel.dev/catalogs/opm/blueprints/workload"
//	import tr "opmodel.dev/catalogs/opm/traits"
//	import res "opmodel.dev/catalogs/opm/resources"
//
// Verified by hand against a real registry: a consumer declaring only
// {core, redis} fails to synthesize with
//
//	cannot find module providing package opmodel.dev/catalogs/opm/blueprints/workload
//
// while redis's OWN cue.mod/module.cue declares opmodel.dev/catalogs/opm@v1.
// Because this change builds the instance INSIDE redis's own staged root (its
// own module.cue is the build's module file), the catalog closure resolves.
//
// Gating: needs CUE_REGISTRY / OPM_REGISTRY serving redis@v0.1.6 + catalogs/opm
// + core. Skips when unset/unreachable unless OPM_FLOW_TEST_FORCE=1 (mirrors the
// other registry-backed integration tests). It is NOT hermetic by necessity.
func TestFlow_Redis_CatalogSubpackage_Regression(t *testing.T) {
	const (
		redisPath = "opmodel.dev/modules/test/redis@v0"
		redisVer  = "v0.1.6"
	)
	force := os.Getenv("OPM_FLOW_TEST_FORCE") == "1"

	registry := firstNonEmpty(os.Getenv("OPM_REGISTRY"), os.Getenv("CUE_REGISTRY"))
	if registry == "" {
		if force {
			t.Fatal("OPM_FLOW_TEST_FORCE=1 but neither OPM_REGISTRY nor CUE_REGISTRY is set")
		}
		t.Skip("no OPM_REGISTRY/CUE_REGISTRY; set one serving redis@v0.1.6 + catalogs/opm to run the #31 redis regression")
	}

	ctx := context.Background()
	k := kernel.New(kernel.WithRegistry(registry))

	mod, err := k.AcquireModuleFromRegistry(ctx, redisPath, redisVer)
	if err != nil {
		// Acquire itself resolves redis as the main module, so it succeeds even on
		// the buggy build — a failure here means the fixture/registry is missing,
		// not a #31 regression. Skip (unless forced) rather than fail spuriously.
		if force {
			t.Fatalf("OPM_FLOW_TEST_FORCE=1 but acquiring %s@%s failed: %v", redisPath, redisVer, err)
		}
		t.Skipf("redis fixture not available in %q (%v); skipping #31 regression", registry, err)
	}
	require.True(t, mod.HasSource(), "acquired redis must carry staged source")

	// THE regression assertion: synthesizing an instance of a module that imports
	// catalog SUBPACKAGES must succeed. Before this change it failed with
	// "cannot find module providing package opmodel.dev/catalogs/opm/...".
	inst, err := k.SynthesizeInstance(ctx, synth.InstanceInput{
		Module:      mod,
		Name:        "redis-inst",
		Namespace:   "default",
		Values:      k.CueContext().CompileString("{}"),
		SchemaCache: k.SchemaCache(),
	})
	if err != nil {
		assert.NotContains(t, err.Error(), "cannot find module providing package",
			"library#31: synth must resolve redis's transitive catalog imports via redis's own cue.mod/module.cue")
		require.NoError(t, err, "synthesizing a catalog-subpackage-importing module must succeed")
	}
	require.NotNil(t, inst)
	require.NotNil(t, inst.Metadata)
	assert.Equal(t, "redis-inst", inst.Metadata.Name)
	kind, err := inst.Package.LookupPath(cue.ParsePath("kind")).String()
	require.NoError(t, err)
	assert.Equal(t, "ModuleInstance", kind)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
