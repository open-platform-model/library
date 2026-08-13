package kernel_test

import (
	"context"
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/compile"
	oerrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/kernel"
)

// This is the always-on, fully hermetic integration harness. It drives the
// public Kernel API (Materialize → Validate → Match → Plan → Compile) against
// in-memory catalogs, with no localhost:5000 dependency. The live, real-catalog
// flow lives in flow_integration_test.go (gated by skipUnlessRegistry).
//
// Divergent-FQN, named-version-absent, major-disagreement, identity-mismatch,
// and disabled-subscription resolution are covered at the materialize-package
// level (opm/materialize/materialize_test.go);
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

		mp, err := materializePlatform(t, k, "0.1.0", path)
		require.NoError(t, err)
		require.NotNil(t, mp)
		assert.Equal(t, "0.1.0", mp.Resolved[subKey(path, "0.1.0")], "resolved version recorded")
		assert.True(t,
			mp.Matchers.LookupPath(cue.ParsePath("resources")).Exists(),
			"materialized platform carries #matchers")
	})

	t.Run("authored version selected among several published", func(t *testing.T) {
		path := registrytest.UniquePath(t, "cat")
		k := newKernelWithCatalogs(t,
			standardCatalog(path, "0.1.0"),
			standardCatalog(path, "0.2.0"),
		)

		// The authored scalar — not the highest published (0.2.0) — is pulled
		// (0010 D14: the platform file IS the resolution).
		mp, err := materializePlatform(t, k, "0.1.0", path)
		require.NoError(t, err)
		assert.Equal(t, "0.1.0", mp.Resolved[subKey(path, "0.1.0")], "authored version resolved")
	})

	t.Run("unresolvable path errors with catalog kind", func(t *testing.T) {
		published := registrytest.UniquePath(t, "cat")
		missing := registrytest.UniquePath(t, "missing")
		k := newKernelWithCatalogs(t, standardCatalog(published, "0.1.0"))

		_, err := materializePlatform(t, k, "0.1.0", missing)
		require.Error(t, err)
		var me *oerrors.MaterializeError
		require.ErrorAs(t, err, &me)
		assert.Equal(t, oerrors.MaterializeKindCatalog, me.Kind)
		assert.Equal(t, subKey(missing, "0.1.0"), me.Subscription)
	})
}

func TestIntegration_MatchPlanCompile(t *testing.T) {
	path := registrytest.UniquePath(t, "cat")
	k := newKernelWithCatalogs(t, standardCatalog(path, "0.1.0"))
	mp, err := materializePlatform(t, k, "0.1.0", path)
	require.NoError(t, err)

	inst := buildInstance(t, k, path, "0.1.0", "", "",
		compSpec{name: "web", resources: []string{"container"}},
		compSpec{name: "config", resources: []string{"config-maps"}},
	)
	ctx := context.Background()

	t.Run("Match pairs both components", func(t *testing.T) {
		plan, err := k.Match(ctx, kernel.MatchInput{ModuleInstance: inst, Platform: mp})
		require.NoError(t, err)
		pairs := matchPairsToMap(plan.MatchedPairs())
		assertContainsFQNSub(t, pairs["web"], "transformers/deployment@", "web → deployment")
		assertContainsFQNSub(t, pairs["config"], "transformers/configmap@", "config → configmap")
		assert.Empty(t, plan.Unmatched)
		assert.Empty(t, plan.Missing)
	})

	t.Run("Plan summarizes components", func(t *testing.T) {
		pr, err := k.Plan(ctx, kernel.PlanInput{ModuleInstance: inst, Platform: mp, RuntimeName: "rt"})
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
		_, err := k.Plan(ctx, kernel.PlanInput{ModuleInstance: inst, Platform: mp})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "RuntimeName must be non-empty")
	})

	t.Run("Compile dispatches struct and list outputs", func(t *testing.T) {
		out, err := k.Compile(ctx, kernel.CompileInput{ModuleInstance: inst, Platform: mp, RuntimeName: "rt"})
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
		_, err := k.Compile(ctx, kernel.CompileInput{ModuleInstance: inst, Platform: mp})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "RuntimeName must be non-empty")
	})
}

func TestIntegration_Compile_UnmatchedComponentErrors(t *testing.T) {
	path := registrytest.UniquePath(t, "cat")
	k := newKernelWithCatalogs(t, standardCatalog(path, "0.1.0"))
	mp, err := materializePlatform(t, k, "0.1.0", path)
	require.NoError(t, err)

	// "web" declares a resource short-name the catalog does not publish.
	inst := buildInstance(t, k, path, "0.1.0", "", "",
		compSpec{name: "web", resources: []string{"does-not-exist"}},
	)

	_, err = k.Compile(context.Background(), kernel.CompileInput{
		ModuleInstance: inst, Platform: mp, RuntimeName: "rt",
	})
	require.Error(t, err)
	var uce *compile.UnmatchedComponentsError
	require.ErrorAs(t, err, &uce)
	assert.Contains(t, uce.Components, "web")
}

// TestIntegration_Compile_UnresolvedDemands covers D28's demand-resolution
// gate on the public Kernel surface: undemandable resources and load-bearing
// unhandled traits fail Compile with the typed aggregate; effectively-optional
// unhandled traits stay warnings; the D4 different-apiVersion diagnostic names
// the alternative. Match stays phase-only throughout (returns the diagnosis,
// never fails on it).
func TestIntegration_Compile_UnresolvedDemands(t *testing.T) {
	ctx := context.Background()

	t.Run("undemandable resource fails Compile", func(t *testing.T) {
		path := registrytest.UniquePath(t, "cat")
		k := newKernelWithCatalogs(t, standardCatalog(path, "0.1.0"))
		mp, err := materializePlatform(t, k, "0.1.0", path)
		require.NoError(t, err)

		inst := buildInstance(t, k, path, "0.1.0", "", "",
			compSpec{name: "web", resources: []string{"container", "does-not-exist"}},
		)

		// Match is phase-only: the diagnosis is returned, not failed.
		plan, err := k.Match(ctx, kernel.MatchInput{ModuleInstance: inst, Platform: mp})
		require.NoError(t, err)
		require.NotEmpty(t, plan.Unresolved)

		_, err = k.Compile(ctx, kernel.CompileInput{ModuleInstance: inst, Platform: mp, RuntimeName: "rt"})
		require.Error(t, err)
		var agg *oerrors.UnresolvedDemandsError
		require.ErrorAs(t, err, &agg)
		var demand oerrors.UnresolvedDemand
		require.ErrorAs(t, err, &demand)
		assert.Equal(t, "web", demand.Component)
		assert.Equal(t, "resource", demand.Kind)
		assert.Contains(t, demand.FQN, "does-not-exist")
		assert.Empty(t, demand.Alternatives, "no other version of this contract exists")
	})

	t.Run("load-bearing unhandled trait fails Compile", func(t *testing.T) {
		path := registrytest.UniquePath(t, "cat")
		k := newKernelWithCatalogs(t, standardCatalog(path, "0.1.0"))
		mp, err := materializePlatform(t, k, "0.1.0", path)
		require.NoError(t, err)

		inst := buildInstance(t, k, path, "0.1.0", "", "",
			compSpec{
				name:          "web",
				resources:     []string{"container"},
				traits:        []string{"backup"},
				traitPostures: map[string]string{"backup": "bool | *false"},
			},
		)

		_, err = k.Compile(ctx, kernel.CompileInput{ModuleInstance: inst, Platform: mp, RuntimeName: "rt"})
		require.Error(t, err)
		var demand oerrors.UnresolvedDemand
		require.ErrorAs(t, err, &demand)
		assert.Equal(t, "trait", demand.Kind)
		assert.Contains(t, demand.FQN, "backup")
		assert.False(t, demand.UnstatedPosture, "posture was stated, just load-bearing")
	})

	t.Run("unstated posture fails closed", func(t *testing.T) {
		path := registrytest.UniquePath(t, "cat")
		k := newKernelWithCatalogs(t, standardCatalog(path, "0.1.0"))
		mp, err := materializePlatform(t, k, "0.1.0", path)
		require.NoError(t, err)

		inst := buildInstance(t, k, path, "0.1.0", "", "",
			compSpec{
				name:          "web",
				resources:     []string{"container"},
				traits:        []string{"backup"},
				traitPostures: map[string]string{"backup": "bool"},
			},
		)

		_, err = k.Compile(ctx, kernel.CompileInput{ModuleInstance: inst, Platform: mp, RuntimeName: "rt"})
		require.Error(t, err)
		var demand oerrors.UnresolvedDemand
		require.ErrorAs(t, err, &demand)
		assert.True(t, demand.UnstatedPosture)
		assert.Contains(t, err.Error(), "no optional posture")
	})

	t.Run("optional unhandled trait warns", func(t *testing.T) {
		path := registrytest.UniquePath(t, "cat")
		k := newKernelWithCatalogs(t, standardCatalog(path, "0.1.0"))
		mp, err := materializePlatform(t, k, "0.1.0", path)
		require.NoError(t, err)

		inst := buildInstance(t, k, path, "0.1.0", "", "",
			compSpec{
				name:      "web",
				resources: []string{"container"},
				traits:    []string{"backup"}, // default posture: bool | *true
			},
		)

		out, err := k.Compile(ctx, kernel.CompileInput{ModuleInstance: inst, Platform: mp, RuntimeName: "rt"})
		require.NoError(t, err, "an effectively-optional unhandled trait must not fail")
		require.NotEmpty(t, out.Warnings)
		assert.Contains(t, out.Warnings[0], "backup")
	})

	t.Run("different apiVersion named as alternative", func(t *testing.T) {
		path := registrytest.UniquePath(t, "cat")
		k := newKernelWithCatalogs(t, standardCatalog(path, "0.1.0"))
		mp, err := materializePlatform(t, k, "0.1.0", path)
		require.NoError(t, err)

		demanded := path + "/resources/container@v9"
		inst := buildInstance(t, k, path, "0.1.0", "", "",
			compSpec{name: "web", resourceKeys: []string{demanded}},
		)

		_, err = k.Compile(ctx, kernel.CompileInput{ModuleInstance: inst, Platform: mp, RuntimeName: "rt"})
		require.Error(t, err)
		var demand oerrors.UnresolvedDemand
		require.ErrorAs(t, err, &demand)
		assert.Equal(t, demanded, demand.FQN)
		assert.Contains(t, demand.Alternatives, resFQN(path, "container"),
			"the published contract level is named as an alternative")
		assert.Contains(t, err.Error(), "different apiVersion")
	})
}

func TestIntegration_Match_MissingFQNRecordsAlternatives(t *testing.T) {
	path := registrytest.UniquePath(t, "cat")
	// Catalog publishes the container contract at ContractAPIVersion ("v1");
	// the component below demands the same contract at an unpublished level.
	k := newKernelWithCatalogs(t, standardCatalog(path, "0.2.0"))
	mp, err := materializePlatform(t, k, "0.2.0", path)
	require.NoError(t, err)

	demanded := path + "/resources/container@v9"
	inst := buildInstance(t, k, path, "0.2.0", "", "",
		compSpec{name: "web", resourceKeys: []string{demanded}},
	)

	plan, err := k.Match(context.Background(), kernel.MatchInput{ModuleInstance: inst, Platform: mp})
	require.NoError(t, err)
	require.NotEmpty(t, plan.Missing, "demanded FQN at an unpublished contract level is a hard miss")

	var found *oerrors.MissingFQN
	for i := range plan.Missing {
		if plan.Missing[i].Component == "web" {
			found = &plan.Missing[i]
			break
		}
	}
	require.NotNil(t, found, "missing FQN recorded for web")
	assert.Equal(t, demanded, found.FQN)
	// The published contract level shares modulePath/name → surfaced as an alternative.
	assert.Contains(t, found.Alternatives, resFQN(path, "container"))
}
