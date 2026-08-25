package kernel_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/compile"
	loaderfile "github.com/open-platform-model/library/opm/helper/loader/file"
	"github.com/open-platform-model/library/opm/kernel"
)

// Render-parity harness (enhancement 0019 D1/D4/D14; openspec change
// library-parity-harness). Every case is rendered twice from the SAME fixture
// bytes and the SAME dependency pins: once through the kernel's public path
// (LoadModulePackage → Materialize → LoadInstancePackage →
// ProcessModuleInstance → Compile) and once by plain CUE unification in one
// build (testdata/parity/oracle/render.cue). The oracle is the reference.
//
// Shipped group: the web_app instance against the published catalogs/opm
// build the opm_platform fixture subscribes to. No shipped transformer reads
// a definition field, so this group is expected to agree; it guards
// regressions and is where an ordering divergence (D14) would surface.
// Probe group (parity_probe_test.go): transformers that read #component.#names
// and #moduleInstance, which is where the kernel diverges today.
//
// Gating mirrors the flow tests: skipped under -short and when GHCR is
// unreachable; OPM_FLOW_TEST_FORCE=1 makes the skip a failure.

// parityRuntimeName is the #runtimeName both renderers use. The oracle fixes
// it in testdata/parity/shipped/shipped.cue (`#runtime`).
const parityRuntimeName = "parity"

func TestParity_ShippedCatalog(t *testing.T) {
	if testing.Short() {
		t.Skip("parity harness pulls the catalog + core schema from GHCR; skipping under -short")
	}
	skipUnlessRegistry(t)

	parityDir := filepath.Join(repoLibraryRoot(t), "testdata", "parity")
	registry := flowRegistry()
	t.Setenv("CUE_REGISTRY", registry)
	ctx := context.Background()

	// ── kernel side ──────────────────────────────────────────────────
	k := kernel.New()
	opts := loaderfile.LoadOptions{Registry: registry}

	modVal, err := k.LoadModulePackage(ctx, filepath.Join(parityDir, "web_app"), opts)
	require.NoError(t, err, "loading web_app fixture copy")
	mod, err := k.NewModuleFromValue(modVal)
	require.NoError(t, err)

	platVal, err := k.LoadPlatformPackage(ctx, filepath.Join(parityDir, "opm_platform"), opts)
	require.NoError(t, err, "loading opm_platform fixture copy")
	plat, err := k.NewPlatformFromValue(platVal)
	require.NoError(t, err)
	mp, err := k.Materialize(ctx, plat)
	require.NoError(t, err, "materializing the platform against the published catalog")

	// The instance enters by IMPORT on both sides: this is the same package
	// the oracle imports, never a LookupPath+FillPath reconstruction.
	instVal, err := k.LoadInstancePackage(ctx, filepath.Join(parityDir, "instance"), opts)
	require.NoError(t, err, "loading the import-authored instance package")
	inst, err := k.ProcessModuleInstance(ctx, instVal, *mod, cue.Value{})
	require.NoError(t, err, "processing the instance")

	plan, err := k.Match(ctx, kernel.MatchInput{ModuleInstance: inst, Platform: mp})
	require.NoError(t, err, "kernel match")
	out, compileErr := k.Compile(ctx, kernel.CompileInput{ModuleInstance: inst, Platform: mp, RuntimeName: parityRuntimeName})

	// ── oracle side ──────────────────────────────────────────────────
	oracle := loadOracle(t, parityDir, "./shipped", registry)

	// ── pair sets (spec: "Matched pair sets agree, with one stated exemption")
	assertPairSetsAgree(t, plan, oraclePairs(t, oracle))

	// ── cases ────────────────────────────────────────────────────────
	pairs := oraclePairs(t, oracle)
	assertRowsCoverPairs(t, shippedCases, pairs)
	for _, c := range shippedCases {
		p := compile.MatchedPair{ComponentName: c.Component, TransformerFQN: c.Transformer}
		t.Run(c.Name, func(t *testing.T) {
			assertParity(t, c, kernelRender(out, compileErr, p), oracleRender(oracle, p))
		})
	}
}

// divergenceContextLabelOrder names the one divergence the shipped group
// exhibits, measured on the harness's first run (2026-08-24): every rendered
// object that carries the context-derived label set has the SAME labels and
// values on both sides, in a different order. opm/schema/context.go decodes
// metadata.labels into a Go map and re-encodes it, so the kernel hands the
// transformer a sorted map where unification hands it CUE's evaluation
// order (0019 D14: the natural order is the contract). Retired by the D12
// projection slice, when #context stops being built in Go.
const divergenceContextLabelOrder = "Go-built #context re-emits label maps sorted (opm/schema/context.go); CUE keeps evaluation order (0019 D12/D14)"

const shippedCatalogPrefix = "opmodel.dev/catalogs/opm/transformers/"

// shippedCases is the table for the shipped group, one row per pair the
// oracle matches for the web_app fixture. The oracle's pair list is
// asserted to equal this set, so a new pair cannot go untested.
var shippedCases = []parityCase{
	{
		Name:               "config :: configmap-transformer",
		Instance:           "testdata/parity/instance",
		Component:          "config",
		Transformer:        shippedCatalogPrefix + "configmap-transformer@2.0.0-alpha.3",
		Equality:           equalityOutputFieldsOnly,
		ExpectedDivergence: divergenceContextLabelOrder,
	},
	{
		Name:               "web :: deployment-transformer",
		Instance:           "testdata/parity/instance",
		Component:          "web",
		Transformer:        shippedCatalogPrefix + "deployment-transformer@2.0.0-alpha.3",
		Equality:           equalityOutputFieldsOnly,
		ExpectedDivergence: divergenceContextLabelOrder,
	},
	{
		// The HPA carries no context-derived labels, so it is the one
		// shipped pair that agrees today.
		Name:        "web :: hpa-transformer",
		Instance:    "testdata/parity/instance",
		Component:   "web",
		Transformer: shippedCatalogPrefix + "hpa-transformer@2.0.0-alpha.3",
		Equality:    equalityOutputFieldsOnly,
	},
	{
		Name:               "web :: http-route-transformer",
		Instance:           "testdata/parity/instance",
		Component:          "web",
		Transformer:        shippedCatalogPrefix + "http-route-transformer@2.0.0-alpha.3",
		Equality:           equalityOutputFieldsOnly,
		ExpectedDivergence: divergenceContextLabelOrder,
	},
	{
		// The guarded-env component. Its second cause, the finalization
		// hoisting of comprehension-produced env entries, was retired by
		// library-component-fill (no transformer receives a finalized value
		// any more); only the label order every deployment shows remains.
		Name:               "worker :: deployment-transformer",
		Instance:           "testdata/parity/instance",
		Component:          "worker",
		Transformer:        shippedCatalogPrefix + "deployment-transformer@2.0.0-alpha.3",
		Equality:           equalityOutputFieldsOnly,
		ExpectedDivergence: divergenceContextLabelOrder,
	},
	{
		Name:        "worker :: hpa-transformer",
		Instance:    "testdata/parity/instance",
		Component:   "worker",
		Transformer: shippedCatalogPrefix + "hpa-transformer@2.0.0-alpha.3",
		Equality:    equalityOutputFieldsOnly,
	},
	{
		Name:               "web :: service-transformer",
		Instance:           "testdata/parity/instance",
		Component:          "web",
		Transformer:        shippedCatalogPrefix + "service-transformer@2.0.0-alpha.3",
		Equality:           equalityOutputFieldsOnly,
		ExpectedDivergence: divergenceContextLabelOrder,
	},
}

// assertRowsCoverPairs requires the case table and the oracle's pair list to
// name the same (component, transformer) set.
func assertRowsCoverPairs(t *testing.T, rows []parityCase, pairs []compile.MatchedPair) {
	t.Helper()
	rowSet := map[string]bool{}
	for _, r := range rows {
		rowSet[r.Component+" :: "+r.Transformer] = true
	}
	pset := pairSet(pairs)
	for k := range pset {
		assert.Truef(t, rowSet[k], "oracle matched pair %s has no row in the case table", k)
	}
	for k := range rowSet {
		assert.Truef(t, pset[k], "case table row %s is not a pair the oracle matches", k)
	}
}

// loadOracle builds the oracle package at pkg (relative to dir) in its own
// cue.Context with the given registry mapping, and returns the built value.
// A fresh context per oracle keeps the two renderers from sharing anything
// but the fixture bytes.
func loadOracle(t *testing.T, dir, pkg, registry string) cue.Value {
	t.Helper()
	env := append(os.Environ(), "CUE_REGISTRY="+registry)
	insts := load.Instances([]string{pkg}, &load.Config{Dir: dir, Env: env})
	require.Len(t, insts, 1, "oracle package %s", pkg)
	require.NoError(t, insts[0].Err, "loading oracle package %s", pkg)
	v := cuecontext.New().BuildInstance(insts[0])
	require.NoError(t, v.Err(), "building oracle package %s", pkg)
	return v
}

// oraclePairs decodes the oracle's `pairs` list.
func oraclePairs(t *testing.T, oracle cue.Value) []compile.MatchedPair {
	t.Helper()
	var raw []struct {
		Component   string `json:"component"`
		Transformer string `json:"transformer"`
	}
	require.NoError(t, oracle.LookupPath(cue.ParsePath("pairs")).Decode(&raw), "decoding oracle pairs")
	pairs := make([]compile.MatchedPair, 0, len(raw))
	for _, r := range raw {
		pairs = append(pairs, compile.MatchedPair{ComponentName: r.Component, TransformerFQN: r.Transformer})
	}
	sortPairs(pairs)
	return pairs
}

// oracleRender reads the oracle's rendered output for one pair, normalised
// to one value per object the way compile.Execute flattens a list output.
func oracleRender(oracle cue.Value, p compile.MatchedPair) parityRender {
	key := p.ComponentName + " :: " + p.TransformerFQN
	v := oracle.LookupPath(cue.MakePath(cue.Str("rendered"), cue.Str(key)))
	if !v.Exists() {
		return parityRender{Err: fmt.Errorf("oracle rendered no entry for %q", key)}
	}
	if err := v.Validate(cue.Concrete(true)); err != nil {
		return parityRender{Err: fmt.Errorf("oracle output for %q is not concrete: %w", key, err)}
	}
	switch v.Kind() {
	case cue.StructKind:
		return parityRender{Objects: []cue.Value{v}}
	case cue.ListKind:
		iter, err := v.List()
		if err != nil {
			return parityRender{Err: err}
		}
		var objs []cue.Value
		for iter.Next() {
			objs = append(objs, iter.Value())
		}
		return parityRender{Objects: objs}
	default:
		return parityRender{Err: fmt.Errorf("oracle output for %q has kind %s", key, v.Kind())}
	}
}

// kernelRender selects the kernel's Compiled objects for one pair, in the
// order the kernel returned them. A Compile error is attributed to every
// pair, because compile.Execute reports pair errors as one aggregate.
func kernelRender(out *compile.CompileResult, compileErr error, p compile.MatchedPair) parityRender {
	if compileErr != nil {
		return parityRender{Err: compileErr}
	}
	var objs []cue.Value
	for _, c := range out.Compiled {
		if c.Component == p.ComponentName && c.Transformer == p.TransformerFQN {
			objs = append(objs, c.Value)
		}
	}
	return parityRender{Objects: objs}
}

// assertPairSetsAgree compares the kernel's matched pairs with the oracle's.
// A pair the oracle admits and the kernel refused through the always-unify
// rung (0010 D30 carve-out; deleted by 0019 D10) is the single exemption:
// reported, never counted as agreement.
func assertPairSetsAgree(t *testing.T, plan *compile.MatchPlan, oracle []compile.MatchedPair) {
	t.Helper()
	kernelPairs := plan.MatchedPairs()
	sortPairs(kernelPairs)

	unified := map[string]bool{}
	for _, u := range plan.Unify {
		unified[u.Component] = true
	}

	kset := pairSet(kernelPairs)
	oset := pairSet(oracle)
	var onlyKernel, onlyOracle, exempt []string
	for k := range kset {
		if !oset[k] {
			onlyKernel = append(onlyKernel, k)
		}
	}
	for k := range oset {
		if kset[k] {
			continue
		}
		comp, _, _ := strings.Cut(k, " :: ")
		if unified[comp] {
			exempt = append(exempt, k)
			continue
		}
		onlyOracle = append(onlyOracle, k)
	}
	sort.Strings(onlyKernel)
	sort.Strings(onlyOracle)
	sort.Strings(exempt)
	for _, e := range exempt {
		t.Logf("pair-set exemption (always-unify refusal, 0019 D10): kernel refused %s", e)
	}
	assert.Emptyf(t, onlyKernel, "pairs the kernel matched and the oracle did not")
	assert.Emptyf(t, onlyOracle, "pairs the oracle matched and the kernel did not (no always-unify refusal recorded)")
}

func pairSet(ps []compile.MatchedPair) map[string]bool {
	m := make(map[string]bool, len(ps))
	for _, p := range ps {
		m[p.ComponentName+" :: "+p.TransformerFQN] = true
	}
	return m
}

func sortPairs(ps []compile.MatchedPair) {
	sort.Slice(ps, func(i, j int) bool {
		if ps[i].ComponentName != ps[j].ComponentName {
			return ps[i].ComponentName < ps[j].ComponentName
		}
		return ps[i].TransformerFQN < ps[j].TransformerFQN
	})
}
