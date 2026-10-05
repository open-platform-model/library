package renderstage

import (
	"path/filepath"
	"sort"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/internal/loader"
	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/internal/schematest"
	"github.com/open-platform-model/library/opm/module"
)

// The skip-unprovided verdict inside the build (render-skips-unprovided-
// provider-demands spike): the glue marks every unresolved row with
// `unprovided`, and under the switch moves those rows onto `skipped`,
// omitting a component with a skipped resource demand from pairs and
// unmatched. Asserted on the built value, with no kernel decode, so the
// module's own verdicts and gate are what is pinned.

const (
	skipPrefix    = "testing.opmodel.dev/library-render"
	skipProviders = skipPrefix + "/providers"
	skipSnapshot  = skipProviders + "/traits/snapshot@v1"
	skipLedger    = skipProviders + "/resources/ledger@v1"
	skipArchive   = skipProviders + "/traits/archive@v1"
	skipDeploy    = skipPrefix + "/cat/transformers/deployment-transformer@0.1.0"
)

type skipRow struct {
	Component        string   `json:"component"`
	Kind             string   `json:"kind"`
	FQN              string   `json:"fqn"`
	DefinedBy        string   `json:"definedBy"`
	Alternatives     []string `json:"alternatives"`
	Unprovided       *bool    `json:"unprovided,omitempty"`
	ComponentOmitted *bool    `json:"componentOmitted,omitempty"`
}

type skipDiagnostics struct {
	Pairs []struct {
		Component   string `json:"component"`
		Transformer string `json:"transformer"`
	} `json:"pairs"`
	Unmatched []struct {
		Component string `json:"component"`
	} `json:"unmatched"`
	Unresolved []skipRow `json:"unresolved"`
	Skipped    []skipRow `json:"skipped"`
}

// buildScenario stages the named scenario package against platformDir with
// the switch set, builds it once, and returns the built value and its
// decoded diagnostics.
func buildScenario(t *testing.T, scenario, platformDir string, skip bool) (cue.Value, skipDiagnostics) {
	t.Helper()
	fixture := filepath.Join(schematest.LibraryRoot(t), "testdata", "render")
	registrytest.NewRegistryFromDir(t, filepath.Join(fixture, "registry"), skipPrefix)

	inst := &module.Source{Root: filepath.Join(fixture, "scenarios"), Pkg: scenario}
	plat := &module.Source{Root: filepath.Join(fixture, platformDir)}
	staged, err := Stage(inst, plat, "rt", StageOptions{SkipUnprovided: skip})
	require.NoError(t, err)

	built, err := Build(cuecontext.New(), staged, loader.Options{})
	require.NoError(t, err)

	dv := built.LookupPath(cue.ParsePath("diagnostics"))
	require.True(t, dv.Exists())
	require.NoError(t, dv.Validate(cue.Concrete(true)), "every verdict is concrete: no cycle, no unresolved default")
	var d skipDiagnostics
	require.NoError(t, dv.Decode(&d))
	return built, d
}

func gateOf(t *testing.T, built cue.Value) (bool, error) {
	t.Helper()
	gate := built.LookupPath(cue.ParsePath("gate"))
	require.True(t, gate.Exists())
	if err := gate.Err(); err != nil {
		return false, err
	}
	return gate.Bool()
}

func pairComponents(d skipDiagnostics) []string {
	seen := map[string]bool{}
	for _, p := range d.Pairs {
		seen[p.Component] = true
	}
	out := make([]string, 0, len(seen))
	for c := range seen {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

func rowsFor(rows []skipRow, component, fqn string) []skipRow {
	var out []skipRow
	for _, r := range rows {
		if r.Component == component && r.FQN == fqn {
			out = append(out, r)
		}
	}
	return out
}

func TestSkipUnprovided_SwitchOnSkipsAndOmits(t *testing.T) {
	built, d := buildScenario(t, "unprovided", "platform_providers", true)

	ok, err := gateOf(t, built)
	require.NoError(t, err, "every unresolved demand here is unprovided, so the gate passes")
	assert.True(t, ok)

	assert.Empty(t, d.Unresolved)
	assert.Empty(t, d.Unmatched, "an omitted component is not unmatched")
	assert.Equal(t, []string{"app", "healthy"}, pairComponents(d), "ledger contributes no pair")

	// Resource rows first, then trait rows, as `unresolved` orders them.
	require.Len(t, d.Skipped, 3)
	assert.Equal(t, "ledger", d.Skipped[0].Component)
	assert.Equal(t, "resource", d.Skipped[0].Kind)
	assert.Equal(t, skipLedger, d.Skipped[0].FQN)

	app := rowsFor(d.Skipped, "app", skipSnapshot)
	require.Len(t, app, 1)
	assert.Equal(t, "trait", app[0].Kind)
	assert.Equal(t, skipProviders+"@v0", app[0].DefinedBy)
	assert.Empty(t, app[0].Alternatives)
	require.NotNil(t, app[0].ComponentOmitted)
	assert.False(t, *app[0].ComponentOmitted, "a skipped trait leaves its component rendering")

	for _, fqn := range []string{skipLedger, skipSnapshot} {
		rows := rowsFor(d.Skipped, "ledger", fqn)
		require.Len(t, rows, 1, fqn)
		require.NotNil(t, rows[0].ComponentOmitted)
		assert.True(t, *rows[0].ComponentOmitted, "every skipped row of an omitted component carries the flag: %s", fqn)
	}

	// The omitted component's pair is not executed either.
	rendered := built.LookupPath(cue.ParsePath("rendered"))
	assert.False(t, rendered.LookupPath(cue.MakePath(cue.Str("ledger :: "+skipDeploy))).Exists())
	assert.True(t, rendered.LookupPath(cue.MakePath(cue.Str("app :: "+skipDeploy))).Exists())
}

func TestSkipUnprovided_SwitchOffRefusesAndMarks(t *testing.T) {
	built, d := buildScenario(t, "unprovided", "platform_providers", false)

	_, err := gateOf(t, built)
	require.Error(t, err, "with the switch off the gate refuses")
	assert.Empty(t, d.Skipped)
	assert.Equal(t, []string{"app", "healthy", "ledger"}, pairComponents(d), "the pair set is as without the switch")

	require.Len(t, d.Unresolved, 3)
	for _, r := range d.Unresolved {
		require.NotNil(t, r.Unprovided, "%s/%s", r.Component, r.FQN)
		assert.True(t, *r.Unprovided, "%s/%s is provider-fulfilled with no provider", r.Component, r.FQN)
	}
}

func TestSkipUnprovided_ProvidedButUnmatchedRefuses(t *testing.T) {
	for _, skip := range []bool{false, true} {
		built, d := buildScenario(t, "provided_unmatched", "platform_providers", skip)

		_, err := gateOf(t, built)
		require.Error(t, err, "skip=%v: a provider exists, so the demand is not skippable", skip)
		assert.Empty(t, d.Skipped, "skip=%v", skip)
		rows := rowsFor(d.Unresolved, "vault", skipArchive)
		require.Len(t, rows, 1, "skip=%v", skip)
		require.NotNil(t, rows[0].Unprovided)
		assert.False(t, *rows[0].Unprovided, "skip=%v", skip)
	}
}

// A catalog-fulfilled demand is never marked: the missing scenario's backup
// trait and orphan resource default to catalog fulfilment.
func TestSkipUnprovided_CatalogFulfilledNotMarked(t *testing.T) {
	built, d := buildScenario(t, "missing", "platform", true)

	_, err := gateOf(t, built)
	require.Error(t, err)
	assert.Empty(t, d.Skipped)
	require.Len(t, d.Unresolved, 2)
	for _, r := range d.Unresolved {
		require.NotNil(t, r.Unprovided)
		assert.False(t, *r.Unprovided, "%s defaults to catalog fulfilment", r.FQN)
	}
}
