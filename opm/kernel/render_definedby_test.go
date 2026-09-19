package kernel_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/kernel"
)

// The defining catalog on an unresolved-demand row (single-build-render,
// "Unresolved demands are diagnosed with alternatives"; 0015:D18): the row
// carries the registry key of the enabled catalog whose contract maps list the
// demanded key, read inside the build from platform.#contracts.definedBy, and
// the refusal's wording distinguishes "defined by a catalog, implemented by
// nothing" from "implemented at a different apiVersion" from "no enabled
// catalog defines it". The gate is unchanged: every case below refuses exactly
// as it did before the field.

const renderCatKey = renderCatPath + "@v0"

func unresolvedByFQN(rows []oerrors.UnresolvedDemand) map[string]oerrors.UnresolvedDemand {
	out := map[string]oerrors.UnresolvedDemand{}
	for _, d := range rows {
		out[d.FQN] = d
	}
	return out
}

// "A defined but unimplemented contract names its catalog": the load-bearing
// backup trait cat lists in #traits and no transformer requires carries cat's
// registry key and no alternatives, and the message names the catalog beside
// the component and key. The orphan@v1 demand on the same instance is the
// alternatives case with the defining catalog named alongside.
func TestRender_DefinedCatalogNamedOnUnresolvedRow(t *testing.T) {
	k := newRenderKernel(t)
	plat := acquireRenderPlatform(t, k, "platform")
	inst := acquireRenderInstance(t, k, "scenarios", "missing")

	built, _, err := k.RenderForTest(context.Background(), kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
	require.Error(t, err)
	assertGateAgrees(t, built, true)
	var rerr *kernel.RenderError
	require.ErrorAs(t, err, &rerr)
	var agg *oerrors.UnresolvedDemandsError
	require.ErrorAs(t, err, &agg)
	byFQN := unresolvedByFQN(rerr.Diagnostics.Unresolved)

	backup := byFQN[renderCatPath+"/traits/backup@v1"]
	assert.Equal(t, renderCatKey, backup.DefinedBy, "the registry key of the listing catalog, never a prefix of the FQN")
	assert.Empty(t, backup.Alternatives)
	assert.Contains(t, agg.Error(),
		`component "orphan": unresolved trait demand "`+renderCatPath+`/traits/backup@v1": defined by "`+renderCatKey+`" and nothing on this platform implements it`)

	orphan := byFQN[renderCatPath+"/resources/orphan@v1"]
	assert.Equal(t, renderCatKey, orphan.DefinedBy)
	assert.Equal(t, []string{renderCatPath + "/resources/orphan@v2"}, orphan.Alternatives)
	assert.Contains(t, agg.Error(),
		`unresolved resource demand "`+renderCatPath+`/resources/orphan@v1": defined by "`+renderCatKey+`", implemented at a different apiVersion`)
	assert.NotContains(t, agg.Error(), "no enabled catalog defines")
}

// "Undemandable resource fails the render": a contract no catalog lists or
// implements (authored inline by the scenario) refuses with a row carrying
// no defining catalog and no alternatives, and the message says no enabled
// catalog defines the contract.
func TestRender_UnlistedDemandNamesNoCatalog(t *testing.T) {
	k := newRenderKernel(t)
	plat := acquireRenderPlatform(t, k, "platform")
	inst := acquireRenderInstance(t, k, "scenarios", "unlisted")

	built, _, err := k.RenderForTest(context.Background(), kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
	require.Error(t, err)
	assertGateAgrees(t, built, true)
	var rerr *kernel.RenderError
	require.ErrorAs(t, err, &rerr)
	var agg *oerrors.UnresolvedDemandsError
	require.ErrorAs(t, err, &agg)

	require.Len(t, rerr.Diagnostics.Unresolved, 1)
	d := rerr.Diagnostics.Unresolved[0]
	assert.Equal(t, "stray", d.Component)
	assert.Equal(t, renderPrefix+"/elsewhere/resources/stray@v1", d.FQN)
	assert.Empty(t, d.DefinedBy, "no enabled catalog lists the key")
	assert.Empty(t, d.Alternatives)
	assert.Empty(t, d.Disqualified)
	assert.Contains(t, agg.Error(),
		`component "stray": unresolved resource demand "`+renderPrefix+`/elsewhere/resources/stray@v1": no enabled catalog defines this contract`)
	assert.NotContains(t, agg.Error(), "defined by")
}

// "A disabled catalog defines nothing": on platform_disabled the only catalog
// listing the missing scenario's demands is a registry entry with
// `enable: false`, so every unresolved row carries no defining catalog even
// though the catalog's members are imported and present in the build.
func TestRender_DisabledCatalogDefinesNothing(t *testing.T) {
	k := newRenderKernel(t)
	plat := acquireRenderPlatform(t, k, "platform_disabled")
	inst := acquireRenderInstance(t, k, "scenarios", "missing")

	// The inventory agrees before the render does: a disabled entry
	// contributes nothing to definedBy.
	inv, err := plat.Contracts()
	require.NoError(t, err)
	assert.Empty(t, inv.DefinedBy)

	built, _, err := k.RenderForTest(context.Background(), kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
	require.Error(t, err)
	assertGateAgrees(t, built, true)
	var rerr *kernel.RenderError
	require.ErrorAs(t, err, &rerr)
	var agg *oerrors.UnresolvedDemandsError
	require.ErrorAs(t, err, &agg)

	byFQN := unresolvedByFQN(rerr.Diagnostics.Unresolved)
	for _, fqn := range []string{
		renderCatPath + "/resources/orphan@v1",
		renderCatPath + "/resources/config-maps@v1",
		renderCatPath + "/traits/backup@v1",
	} {
		d, ok := byFQN[fqn]
		require.True(t, ok, "unresolved row for %s", fqn)
		assert.Empty(t, d.DefinedBy, "%s is listed by a disabled entry only", fqn)
		assert.Empty(t, d.Alternatives, "%s: the disabled catalog's transformers are not candidates", fqn)
	}
	assert.Contains(t, agg.Error(), "no enabled catalog defines this contract")
	assert.NotContains(t, agg.Error(), "defined by")
}
