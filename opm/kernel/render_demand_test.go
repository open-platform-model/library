package kernel_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/internal/corepath"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/module"
)

// single-build-render spec, "A render reports every contract its instance
// requires" (0013:D24).

const (
	containerFQN  = renderCatPath + "/resources/container@v1"
	configMapsFQN = renderCatPath + "/resources/config-maps@v1"
	orphanFQN     = renderCatPath + "/resources/orphan@v1"
	exposeFQN     = renderCatPath + "/traits/expose@v1"
	sidecarFQN    = renderCatPath + "/traits/sidecar@v1"
	backupFQN     = renderCatPath + "/traits/backup@v1"
)

// renderDemand renders the fixture at parts on platformDir and returns the
// demand from whichever of RenderResult and *RenderError came back. ok is
// false when the render returned a plain error, which carries no demand.
func renderDemand(t *testing.T, k *kernel.Kernel, platformDir string, skip bool, parts ...string) (demand []string, res *kernel.RenderResult, ok bool, err error) {
	t.Helper()
	plat := acquireRenderPlatform(t, k, platformDir)
	inst := acquireRenderInstance(t, k, parts...)
	res, err = k.Render(context.Background(), kernel.RenderInput{
		Instance: inst, Platform: plat, RuntimeName: "rt", SkipUnprovided: skip,
	})
	if res != nil {
		return res.Diagnostics.RequiredContracts, res, true, err
	}
	var rerr *kernel.RenderError
	if errors.As(err, &rerr) {
		return rerr.Diagnostics.RequiredContracts, nil, true, err
	}
	return nil, nil, false, err
}

// "Demand on a successful render" and "A component without traits": web and
// worker share the container key, worker attaches no #traits at all.
func TestRenderDemand_SuccessSortedAndDeduplicated(t *testing.T) {
	k := newRenderKernel(t)
	demand, res, ok, err := renderDemand(t, k, "platform", false, "scenarios", "shared_demand")
	require.NoError(t, err, "an absent #traits does not fail the render")
	require.True(t, ok)
	require.NotNil(t, res)
	assert.Equal(t, []string{configMapsFQN, containerFQN, exposeFQN}, demand,
		"every resource and trait key once, in byte order")
}

// The happy-path instance reports its two components' keys.
func TestRenderDemand_HappyPathInstance(t *testing.T) {
	k := newRenderKernel(t)
	demand, _, ok, err := renderDemand(t, k, "platform", false, "instance")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, []string{configMapsFQN, containerFQN, exposeFQN}, demand)
}

// "No components".
func TestRenderDemand_NoComponentsIsEmptyNotNil(t *testing.T) {
	k := newRenderKernel(t)
	demand, res, ok, err := renderDemand(t, k, "platform", false, "scenarios", "empty")
	require.NoError(t, err)
	require.True(t, ok)
	require.NotNil(t, res)
	require.NotNil(t, demand, "an instance with no components reports an empty, non-nil list")
	assert.Empty(t, demand)
}

// "Demand on a refusal": the unresolved orphan resource is listed beside the
// keys that did resolve.
func TestRenderDemand_OnRenderError(t *testing.T) {
	k := newRenderKernel(t)
	demand, res, ok, err := renderDemand(t, k, "platform", false, "scenarios", "missing")
	require.Error(t, err)
	assert.Nil(t, res)
	require.True(t, ok, "the gate's refusal is a *RenderError carrying the demand, got: %v", err)
	assert.Equal(t, []string{configMapsFQN, orphanFQN, backupFQN}, demand)
}

// "An omitted component's demand is reported": under the switch, ledger is
// omitted for its unprovided resource, and its keys stay on the demand.
func TestRenderDemand_OmittedComponentCounts(t *testing.T) {
	k := newRenderKernel(t)
	demand, res, ok, err := renderDemand(t, k, "platform_providers", true, "scenarios", "unprovided")
	require.NoError(t, err)
	require.True(t, ok)
	require.NotNil(t, res)
	require.NotEmpty(t, skippedFor(res.Diagnostics.Skipped, "ledger", ledgerFQN), "ledger is omitted")
	for _, p := range res.Diagnostics.Pairs {
		require.NotEqual(t, "ledger", p.Component, "an omitted component renders nothing")
	}
	assert.Contains(t, demand, ledgerFQN, "the omitted component's resource is demand")
	assert.Contains(t, demand, snapshotFQN, "the skipped trait is demand")
	assert.True(t, slices.IsSorted(demand))
}

// "A component with traits and no resources": an empty #resources map
// beside an attached trait. Nothing matches it, so the render refuses, and
// the refusal carries the trait key.
func TestRenderDemand_TraitsOnlyComponent(t *testing.T) {
	k := newRenderKernel(t)
	demand, _, ok, err := renderDemand(t, k, "platform", false, "scenarios", "traits_only")
	require.Error(t, err)
	require.True(t, ok, "the refusal is a *RenderError, got: %v", err)
	assert.Equal(t, []string{sidecarFQN}, demand)
}

// walkDeclaredContracts copies the operator's demand walk:
// opm-operator internal/render/demand.go, declaredContracts and contractKeys,
// at opm-operator origin/main 9b83611. The paths are built locally: the
// library's opm/schema carries no component #resources or #traits path
// (schema-dispatch, "Matcher and transformer paths are gone").
func walkDeclaredContracts(inst *module.Instance) ([]string, error) {
	resources := cue.MakePath(cue.Def("resources"))
	traits := cue.MakePath(cue.Def("traits"))
	contracts := make([]string, 0)
	if inst == nil {
		return contracts, nil
	}
	components := inst.Package.LookupPath(corepath.Components)
	if !components.Exists() {
		return contracts, nil
	}
	if err := components.Err(); err != nil {
		return nil, err
	}
	iter, err := components.Fields()
	if err != nil {
		return nil, err
	}
	for iter.Next() {
		for _, declared := range []cue.Path{resources, traits} {
			demands := iter.Value().LookupPath(declared)
			if !demands.Exists() {
				continue
			}
			fields, err := demands.Fields()
			if err != nil {
				return nil, fmt.Errorf("component %q: %w", iter.Selector().Unquoted(), err)
			}
			for fields.Next() {
				contracts = append(contracts, fields.Selector().Unquoted())
			}
		}
	}
	slices.Sort(contracts)
	return slices.Compact(contracts), nil
}

// "The demand equals the declared contracts of the instance": for every
// render fixture instance (none has synthesised components), the build's
// demand equals the operator walk it replaces. Every scenario package on
// disk is classified, so a new one must either join the parity set or be
// named as a render that never reaches its diagnostics.
func TestRenderDemand_ParityWithOperatorWalk(t *testing.T) {
	// Scenarios whose render returns a plain error before or at the
	// diagnostics decode, so no demand is reported. bad_traits and
	// no_resources are not acquirable at all.
	excluded := map[string]bool{"unstated": true, "no_resources": true, "bad_traits": true}
	notAcquirable := map[string]bool{"no_resources": true, "bad_traits": true}
	onProviders := map[string]bool{
		"unprovided": true, "provided_unmatched": true, "optional_unprovided": true,
		"skipped_beside_refused": true, "omitted_refused": true,
	}

	entries, err := os.ReadDir(renderFixtureDir(t, "scenarios"))
	require.NoError(t, err)
	type fixture struct {
		platform string
		parts    []string
	}
	var set []fixture
	seenExcluded := map[string]bool{}
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "cue.mod" {
			continue
		}
		name := e.Name()
		if excluded[name] {
			seenExcluded[name] = true
			continue
		}
		plat := "platform"
		if onProviders[name] {
			plat = "platform_providers"
		}
		set = append(set, fixture{plat, []string{"scenarios", name}})
	}
	require.Equal(t, excluded, seenExcluded, "every excluded scenario exists on disk")
	set = append(set,
		fixture{"platform", []string{"instance"}},
		fixture{"platform_collide", []string{"instance_maj0"}},
		fixture{"platform_collide", []string{"instance_bk0"}},
	)

	for _, f := range set {
		t.Run(fmt.Sprint(f.parts), func(t *testing.T) {
			k := newRenderKernel(t)
			inst := acquireRenderInstance(t, k, f.parts...)
			want, err := walkDeclaredContracts(inst)
			require.NoError(t, err)
			got, _, ok, rerr := renderDemand(t, k, f.platform, false, f.parts...)
			require.True(t, ok, "the render reaches its diagnostics, got: %v", rerr)
			assert.Equal(t, want, got)
		})
	}

	// The exclusions are asserted, not assumed: each renders to a plain
	// error and carries no demand.
	for name := range excluded {
		t.Run("excluded/"+name, func(t *testing.T) {
			k := newRenderKernel(t)
			plat := acquireRenderPlatform(t, k, "platform")
			var inst *module.Instance
			if notAcquirable[name] {
				_, aerr := k.AcquireInstanceFromDir(context.Background(), renderFixtureDir(t, "scenarios", name))
				require.Error(t, aerr, "%s is not acquirable", name)
				inst = scenarioLiteralInstance(t, name)
			} else {
				inst = acquireRenderInstance(t, k, "scenarios", name)
			}
			res, err := k.Render(context.Background(), kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
			require.Error(t, err)
			assert.Nil(t, res)
			var rerr *kernel.RenderError
			assert.False(t, errors.As(err, &rerr), "%s refuses with a plain error, so it carries no demand", name)
		})
	}
}
