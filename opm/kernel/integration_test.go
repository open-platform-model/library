package kernel_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/compile"
	oerrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/schema"
)

// This is the always-on, fully hermetic integration harness. It drives the
// public Kernel API (Materialize → Validate → Match → Plan → Compile) against
// in-memory catalogs, with no localhost:5000 dependency. The live, real-catalog
// flow lives in flow_integration_test.go (gated by skipUnlessRegistry).
//
// Divergent-FQN, range/allow/deny, and disabled-subscription resolution are
// covered at the materialize-package level (opm/materialize/materialize_test.go);
// this harness focuses on the kernel surface and the Match→Compile path.

// standardCatalog is the common two-transformer catalog: "deployment" requires
// the container resource and emits a single struct (→ 1 Compiled), "configmap"
// requires the config-maps resource and emits a two-element list (→ 2 Compiled).
func standardCatalog(path, version string) registrytest.CatalogFixture {
	return registrytest.CatalogFixture{
		Path:    path,
		Version: version,
		Body: registrytest.BuildCatalog(path, version,
			registrytest.TxFixture{
				Name:      "deployment",
				Resources: []string{"container"},
				Output:    `{ kind: "Deployment" }`,
			},
			registrytest.TxFixture{
				Name:      "configmap",
				Resources: []string{"config-maps"},
				Output:    `[ {kind: "ConfigMap", n: 1}, {kind: "ConfigMap", n: 2} ]`,
			},
		),
	}
}

func TestIntegration_Materialize(t *testing.T) {
	t.Run("happy single subscription", func(t *testing.T) {
		path := registrytest.UniquePath(t, "cat")
		k := newKernelWithCatalogs(t, standardCatalog(path, "0.1.0"))

		mp, err := materializePlatform(t, k, path)
		require.NoError(t, err)
		require.NotNil(t, mp)
		assert.Equal(t, "0.1.0", mp.Resolved[path], "resolved version recorded")
		assert.True(t,
			mp.Package.LookupPath(schema.MatchersResources).Exists(),
			"materialized platform carries #matchers")
	})

	t.Run("highest version selected", func(t *testing.T) {
		path := registrytest.UniquePath(t, "cat")
		k := newKernelWithCatalogs(t,
			standardCatalog(path, "0.1.0"),
			standardCatalog(path, "0.2.0"),
		)

		mp, err := materializePlatform(t, k, path)
		require.NoError(t, err)
		assert.Equal(t, "0.2.0", mp.Resolved[path], "no filter → highest SemVer")
	})

	t.Run("unresolvable path errors with catalog kind", func(t *testing.T) {
		published := registrytest.UniquePath(t, "cat")
		missing := registrytest.UniquePath(t, "missing")
		k := newKernelWithCatalogs(t, standardCatalog(published, "0.1.0"))

		_, err := materializePlatform(t, k, missing)
		require.Error(t, err)
		var me *oerrors.MaterializeError
		require.ErrorAs(t, err, &me)
		assert.Equal(t, oerrors.MaterializeKindCatalog, me.Kind)
		assert.Equal(t, missing, me.Subscription)
	})
}

func TestIntegration_MatchPlanCompile(t *testing.T) {
	path := registrytest.UniquePath(t, "cat")
	k := newKernelWithCatalogs(t, standardCatalog(path, "0.1.0"))
	mp, err := materializePlatform(t, k, path)
	require.NoError(t, err)

	rel := buildRelease(t, k, path, "0.1.0", "", "",
		compSpec{name: "web", resources: []string{"container"}},
		compSpec{name: "config", resources: []string{"config-maps"}},
	)
	ctx := context.Background()

	t.Run("Match pairs both components", func(t *testing.T) {
		plan, err := k.Match(ctx, kernel.MatchInput{ModuleRelease: rel, Platform: mp})
		require.NoError(t, err)
		pairs := matchPairsToMap(plan.MatchedPairs())
		assertContainsFQNSub(t, pairs["web"], "transformers/deployment@", "web → deployment")
		assertContainsFQNSub(t, pairs["config"], "transformers/configmap@", "config → configmap")
		assert.Empty(t, plan.Unmatched)
		assert.Empty(t, plan.Missing)
	})

	t.Run("Plan summarizes components", func(t *testing.T) {
		pr, err := k.Plan(ctx, kernel.PlanInput{ModuleRelease: rel, Platform: mp, RuntimeName: "rt"})
		require.NoError(t, err)
		assert.Empty(t, pr.Unmatched)
		require.Len(t, pr.Components, 2)
		byName := map[string]compile.ComponentSummary{}
		for _, c := range pr.Components {
			byName[c.Name] = c
		}
		assertContainsFQNSub(t, byName["web"].ResourceFQNs, "resources/container@", "web declares container")
		assertContainsFQNSub(t, byName["config"].ResourceFQNs, "resources/config-maps@", "config declares config-maps")
	})

	t.Run("Plan requires runtime name", func(t *testing.T) {
		_, err := k.Plan(ctx, kernel.PlanInput{ModuleRelease: rel, Platform: mp})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "RuntimeName must be non-empty")
	})

	t.Run("Compile dispatches struct and list outputs", func(t *testing.T) {
		out, err := k.Compile(ctx, kernel.CompileInput{ModuleRelease: rel, Platform: mp, RuntimeName: "rt"})
		require.NoError(t, err)
		require.NotNil(t, out)
		assert.Empty(t, out.Unmatched)

		perComp := map[string]int{}
		for _, c := range out.Compiled {
			perComp[c.Component]++
		}
		assert.Equal(t, 1, perComp["web"], "struct output → one Compiled")
		assert.Equal(t, 2, perComp["config"], "two-element list output → two Compiled")
	})

	t.Run("Compile requires runtime name", func(t *testing.T) {
		_, err := k.Compile(ctx, kernel.CompileInput{ModuleRelease: rel, Platform: mp})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "RuntimeName must be non-empty")
	})
}

func TestIntegration_Compile_UnmatchedComponentErrors(t *testing.T) {
	path := registrytest.UniquePath(t, "cat")
	k := newKernelWithCatalogs(t, standardCatalog(path, "0.1.0"))
	mp, err := materializePlatform(t, k, path)
	require.NoError(t, err)

	// "web" declares a resource short-name the catalog does not publish.
	rel := buildRelease(t, k, path, "0.1.0", "", "",
		compSpec{name: "web", resources: []string{"does-not-exist"}},
	)

	_, err = k.Compile(context.Background(), kernel.CompileInput{
		ModuleRelease: rel, Platform: mp, RuntimeName: "rt",
	})
	require.Error(t, err)
	var uce *compile.UnmatchedComponentsError
	require.ErrorAs(t, err, &uce)
	assert.Contains(t, uce.Components, "web")
}

func TestIntegration_Match_MissingFQNRecordsAlternatives(t *testing.T) {
	path := registrytest.UniquePath(t, "cat")
	// Catalog publishes 0.2.0 only; the component below demands the 0.1.0 FQN.
	k := newKernelWithCatalogs(t, standardCatalog(path, "0.2.0"))
	mp, err := materializePlatform(t, k, path)
	require.NoError(t, err)

	rel := buildRelease(t, k, path, "0.1.0", "", "",
		compSpec{name: "web", resources: []string{"container"}},
	)

	plan, err := k.Match(context.Background(), kernel.MatchInput{ModuleRelease: rel, Platform: mp})
	require.NoError(t, err)
	require.NotEmpty(t, plan.Missing, "demanded FQN at an unpublished version is a hard miss")

	var found *oerrors.MissingFQN
	for i := range plan.Missing {
		if plan.Missing[i].Component == "web" {
			found = &plan.Missing[i]
			break
		}
	}
	require.NotNil(t, found, "missing FQN recorded for web")
	assert.Equal(t, resFQN(path, "container", "0.1.0"), found.FQN)
	// The published 0.2.0 FQN shares modulePath/name → surfaced as an alternative.
	assert.Contains(t, found.Alternatives, resFQN(path, "container", "0.2.0"))
}
