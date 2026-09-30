package kernel_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/kernel"
)

// single-build-render spec, "A caller may skip unprovided provider-fulfilled
// demands". platform_providers carries cat and the providers fixture catalog,
// which lists the provider-fulfilled snapshot trait and ledger resource with
// no transformer, and the archive trait with one label-gated provider.
const (
	renderProvidersPath = renderPrefix + "/providers"
	renderProvidersKey  = renderProvidersPath + "@v0"
	snapshotFQN         = renderProvidersPath + "/traits/snapshot@v1"
	ledgerFQN           = renderProvidersPath + "/resources/ledger@v1"
	archiveFQN          = renderProvidersPath + "/traits/archive@v1"
)

// skipRender is one render of a scenario with the switch set; gateAgrees
// asserts the render module's own gate against the kernel's verdict.
type skipRender struct {
	res        *kernel.RenderResult
	err        error
	gateAgrees func(refused bool)
}

// renderScenario renders the named scenario package (the plain fixture
// instance when scenario is empty) against platformDir.
func renderScenario(t *testing.T, platformDir, scenario string, skip bool) skipRender {
	t.Helper()
	k := newRenderKernel(t)
	plat := acquireRenderPlatform(t, k, platformDir)
	parts := []string{"scenarios", scenario}
	if scenario == "" {
		parts = []string{"instance"}
	}
	inst := acquireRenderInstance(t, k, parts...)
	built, res, err := k.RenderForTest(context.Background(), kernel.RenderInput{
		Instance: inst, Platform: plat, RuntimeName: "rt", SkipUnprovided: skip,
	})
	return skipRender{res: res, err: err, gateAgrees: func(refused bool) { assertGateAgrees(t, built, refused) }}
}

func skippedFor(rows []kernel.SkippedDemand, component, fqn string) []kernel.SkippedDemand {
	var out []kernel.SkippedDemand
	for _, r := range rows {
		if r.Component == component && r.FQN == fqn {
			out = append(out, r)
		}
	}
	return out
}

// "Switch off refuses an unprovided trait as before" and "The module's own
// gate agrees under the switch" (switch off).
func TestRenderSkip_SwitchOffRefusesAndMarks(t *testing.T) {
	r := renderScenario(t, "platform_providers", "unprovided", false)
	err := r.err
	require.Error(t, err)
	r.gateAgrees(true)

	var rerr *kernel.RenderError
	require.ErrorAs(t, err, &rerr)
	var agg *oerrors.UnresolvedDemandsError
	require.ErrorAs(t, err, &agg)
	assert.Empty(t, rerr.Diagnostics.Skipped, "no skipped row with the switch off")

	snap := unresolvedByFQN(rowsOfComponent(rerr.Diagnostics.Unresolved, "app"))[snapshotFQN]
	assert.True(t, snap.Unprovided, "the unprovided trait row is marked")
	assert.Equal(t, renderProvidersKey, snap.DefinedBy)
	for _, d := range rerr.Diagnostics.Unresolved {
		assert.True(t, d.Unprovided, "%s/%s", d.Component, d.FQN)
	}
	assert.Equal(t, rerr.Diagnostics.Unresolved, agg.Demands, "the cause carries the diagnostics rows unchanged")
}

// "An unprovided trait is skipped and its component renders", "An unprovided
// resource omits its component", "A skipped demand is a row" and the gate
// agreeing with the switch on.
func TestRenderSkip_SwitchOnSkipsTraitAndOmitsComponent(t *testing.T) {
	r := renderScenario(t, "platform_providers", "unprovided", true)
	res, err := r.res, r.err
	require.NoError(t, err)
	r.gateAgrees(false)

	d := res.Diagnostics
	assert.Empty(t, d.Unresolved)
	assert.Empty(t, d.Unmatched, "the omitted component is not reported unmatched")

	// The trait: app renders every pair it matched; one trait row, not omitted.
	app := skippedFor(d.Skipped, "app", snapshotFQN)
	require.Len(t, app, 1)
	assert.Equal(t, kernel.SkippedDemand{
		Component: "app", FQN: snapshotFQN, Kind: "trait",
		DefinedBy: renderProvidersKey, Alternatives: []string{}, ComponentOmitted: false,
	}, app[0])

	// The resource: ledger renders nothing, although its container matches
	// the deployment transformer; every ledger row carries the flag.
	ledger := skippedFor(d.Skipped, "ledger", ledgerFQN)
	require.Len(t, ledger, 1)
	assert.Equal(t, "resource", ledger[0].Kind)
	assert.Equal(t, renderProvidersKey, ledger[0].DefinedBy)
	for _, r := range d.Skipped {
		if r.Component == "ledger" {
			assert.True(t, r.ComponentOmitted, "%s", r.FQN)
		}
	}
	require.Len(t, d.Skipped, 3, "ledger resource, app trait, ledger trait")
	assert.Equal(t, "resource", d.Skipped[0].Kind, "resource rows come first, in build order")

	assert.Equal(t, []string{
		"app :: deployment-transformer@0.1.0",
		"healthy :: deployment-transformer@0.1.0",
	}, renderPairSet(d.Pairs))
	assert.ElementsMatch(t, []string{
		"app/Deployment/unprovided-demo-app",
		"healthy/Deployment/unprovided-demo-healthy",
	}, compiledSummary(t, res.Compiled))
	for _, c := range res.Compiled {
		assert.NotEqual(t, "ledger", c.Component, "no object renders for the omitted component")
	}
}

// "A catalog-fulfilled demand still refuses": the missing scenario's backup
// trait and orphan resource default to catalog fulfilment.
func TestRenderSkip_CatalogFulfilledStillRefuses(t *testing.T) {
	r := renderScenario(t, "platform", "missing", true)
	err := r.err
	require.Error(t, err)
	r.gateAgrees(true)

	var rerr *kernel.RenderError
	require.ErrorAs(t, err, &rerr)
	var agg *oerrors.UnresolvedDemandsError
	require.ErrorAs(t, err, &agg)
	backup, ok := unresolvedByFQN(rerr.Diagnostics.Unresolved)[renderCatPath+"/traits/backup@v1"]
	require.True(t, ok)
	assert.False(t, backup.Unprovided)
	assert.Empty(t, rerr.Diagnostics.Skipped)
	assert.NotContains(t, agg.Error(), "provider-fulfilled")
}

// "A provider that exists but does not match still refuses".
func TestRenderSkip_ProvidedButUnmatchedStillRefuses(t *testing.T) {
	for _, skip := range []bool{false, true} {
		r := renderScenario(t, "platform_providers", "provided_unmatched", skip)
		err := r.err
		require.Error(t, err, "skip=%v", skip)
		r.gateAgrees(true)

		var rerr *kernel.RenderError
		require.ErrorAs(t, err, &rerr)
		archive, ok := unresolvedByFQN(rerr.Diagnostics.Unresolved)[archiveFQN]
		require.True(t, ok, "skip=%v", skip)
		assert.False(t, archive.Unprovided, "skip=%v: a provider exists", skip)
		assert.Empty(t, rerr.Diagnostics.Skipped, "skip=%v", skip)
	}
}

// "Over-subscription still refuses".
func TestRenderSkip_OverSubscriptionStillRefuses(t *testing.T) {
	r := renderScenario(t, "platform_oversubscribed", "", true)
	err := r.err
	require.Error(t, err)
	r.gateAgrees(true)

	var ose *oerrors.OverSubscribedContractsError
	require.ErrorAs(t, err, &ose)
	require.Len(t, ose.Contracts, 1)
	assert.Equal(t, renderCatPath+"/resources/gateway@v1", ose.Contracts[0].Key)
	var unresolved *oerrors.UnresolvedDemandsError
	assert.False(t, errors.As(err, &unresolved))
	var rerr *kernel.RenderError
	require.ErrorAs(t, err, &rerr)
	assert.Empty(t, rerr.Diagnostics.Skipped)
}

// rowsOfComponent keeps the unresolved rows of one component.
func rowsOfComponent(rows []oerrors.UnresolvedDemand, component string) []oerrors.UnresolvedDemand {
	var out []oerrors.UnresolvedDemand
	for _, r := range rows {
		if r.Component == component {
			out = append(out, r)
		}
	}
	return out
}

// "An unhandled trait whose effective optional is true SHALL remain an
// unhandled-trait table entry, never a skipped row": the snapshot trait made
// optional at the attachment site renders as advisory under both switch
// values.
func TestRenderSkip_OptionalUnprovidedTraitStaysUnhandled(t *testing.T) {
	for _, skip := range []bool{false, true} {
		r := renderScenario(t, "platform_providers", "optional_unprovided", skip)
		require.NoError(t, r.err, "skip=%v", skip)
		r.gateAgrees(false)

		d := r.res.Diagnostics
		assert.Equal(t, map[string][]string{"app": {snapshotFQN}}, d.UnhandledTraits, "skip=%v", skip)
		assert.Empty(t, d.Skipped, "skip=%v: an effectively-optional trait is never a skipped row", skip)
		assert.Empty(t, d.Unresolved, "skip=%v", skip)
		assert.Equal(t, []string{"app :: deployment-transformer@0.1.0"}, renderPairSet(d.Pairs), "skip=%v", skip)
		assert.Equal(t, []string{"app/Deployment/optional-unprovided-demo-app"}, compiledSummary(t, r.res.Compiled), "skip=%v", skip)
	}
}

// "The skipped-demand rows SHALL be readable on the diagnostics beside such
// a refusal": app's unprovided snapshot trait is skipped while vault's
// label-gated archive trait still refuses.
func TestRenderSkip_SkippedRowsBesideRefusal(t *testing.T) {
	r := renderScenario(t, "platform_providers", "skipped_beside_refused", true)
	err := r.err
	require.Error(t, err)
	r.gateAgrees(true)

	var rerr *kernel.RenderError
	require.ErrorAs(t, err, &rerr)
	var agg *oerrors.UnresolvedDemandsError
	require.ErrorAs(t, err, &agg)

	require.Len(t, agg.Demands, 1, "only the standing refusal is in the cause")
	assert.Equal(t, "vault", agg.Demands[0].Component)
	assert.Equal(t, archiveFQN, agg.Demands[0].FQN)
	assert.False(t, agg.Demands[0].Unprovided, "a provider exists")
	assert.Equal(t, rerr.Diagnostics.Unresolved, agg.Demands)

	assert.Equal(t, []kernel.SkippedDemand{{
		Component: "app", FQN: snapshotFQN, Kind: "trait",
		DefinedBy: renderProvidersKey, Alternatives: []string{}, ComponentOmitted: false,
	}}, rerr.Diagnostics.Skipped, "the skipped row is readable beside the refusal")
}

// design.md "An omitted component drops its warnings but keeps every other
// refusal": ledger is omitted for its unprovided resource, so its advisory
// sidecar leaves the unhandled-trait table while web's stays; its
// catalog-fulfilled backup and provided-but-unmatched archive still refuse.
func TestRenderSkip_OmittedComponentDropsWarningsKeepsRefusals(t *testing.T) {
	r := renderScenario(t, "platform_providers", "omitted_refused", true)
	err := r.err
	require.Error(t, err)
	r.gateAgrees(true)

	var rerr *kernel.RenderError
	require.ErrorAs(t, err, &rerr)
	var agg *oerrors.UnresolvedDemandsError
	require.ErrorAs(t, err, &agg)
	d := rerr.Diagnostics

	sidecar := renderCatPath + "/traits/sidecar@v1"
	assert.Equal(t, map[string][]string{"web": {sidecar}}, d.UnhandledTraits,
		"the omitted component's advisory trait is not on the table")

	rows := unresolvedByFQN(d.Unresolved)
	require.Len(t, d.Unresolved, 2)
	for _, fqn := range []string{renderCatPath + "/traits/backup@v1", archiveFQN} {
		row, ok := rows[fqn]
		require.True(t, ok, "%s still refuses on the omitted component", fqn)
		assert.Equal(t, "ledger", row.Component)
		assert.False(t, row.Unprovided, "%s", fqn)
	}
	assert.Equal(t, d.Unresolved, agg.Demands)

	require.Len(t, d.Skipped, 1)
	assert.Equal(t, "ledger", d.Skipped[0].Component)
	assert.Equal(t, ledgerFQN, d.Skipped[0].FQN)
	assert.True(t, d.Skipped[0].ComponentOmitted)
	assert.Empty(t, d.Unmatched, "the omitted component is not reported unmatched")
	assert.Equal(t, []string{"web :: deployment-transformer@0.1.0"}, renderPairSet(d.Pairs))
}
