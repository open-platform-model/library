package kernel_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loaderfile "github.com/open-platform-model/library/opm/helper/loader/file"
	"github.com/open-platform-model/library/opm/kernel"
)

// Render-parity harness (enhancement 0019 D1/D4/D14; openspec changes
// library-parity-harness and library-render-cutover). Every case is rendered
// twice from the SAME fixture bytes and the SAME dependency pins: once
// through the kernel's public path (AcquireInstanceFromDir +
// AcquirePlatformFromDir → Render, one build) and once by plain CUE
// unification in one build (testdata/parity/oracle/render.cue). The oracle
// is the reference.
//
// Shipped group: the web_app instance against the published catalogs/opm
// build the opm_platform fixture imports. Probe group
// (parity_probe_test.go): transformers that read #component.#names and
// #moduleInstance. Every case compares the whole rendered value,
// order-sensitively, and carries no expected divergence: with #context
// projected by core (0019 D12) there is no runtime-built value left to
// exclude.
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
	// The instance enters by IMPORT on both sides: this is the same package
	// the oracle imports, never a LookupPath+FillPath reconstruction. The
	// platform is its own module (testdata/parity/opm_platform) importing
	// the same published catalog build the parity module pins.
	k := kernel.New(kernel.WithRegistry(registry))
	opts := loaderfile.LoadOptions{Registry: registry}

	inst, err := k.AcquireInstanceFromDir(ctx, filepath.Join(parityDir, "instance"), opts)
	require.NoError(t, err, "acquiring the import-authored instance package")
	plat, err := k.AcquirePlatformFromDir(ctx, filepath.Join(parityDir, "opm_platform"), opts)
	require.NoError(t, err, "acquiring the opm_platform fixture copy")

	res, renderErr := k.Render(ctx, kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: parityRuntimeName})

	// ── oracle side ──────────────────────────────────────────────────
	oracle := loadOracle(t, parityDir, "./shipped", registry)

	// ── one catalog build on both sides (0019 OQ3, executable) ───────
	// The oracle imports the catalog through the parity module's cue.mod;
	// Render resolves it through the platform module's. The two pins must
	// name one build, or the comparison is between two catalogs.
	oracleCatalog, err := oracle.LookupPath(cue.ParsePath("catalogVersion")).String()
	require.NoError(t, err, "the oracle reports the catalog build it resolved")
	require.NoError(t, renderErr, "Render must render the parity instance")
	assert.Contains(t, res.Diagnostics.ResolvedVersions, kernel.ResolvedVersion{
		Path:            "opmodel.dev/catalogs/opm@v4",
		ModuleVersion:   "v" + oracleCatalog,
		PlatformVersion: "v" + oracleCatalog,
	}, "the platform module and the parity module must pin the catalog build the oracle resolved")

	// ── pair sets (spec: "Matched pair sets agree") ──────────────────
	pairs := oraclePairs(t, oracle)
	assertPairSetsAgree(t, res.Diagnostics.Pairs, pairs)

	// ── cases ────────────────────────────────────────────────────────
	assertRowsCoverPairs(t, shippedCases, pairs)
	for _, c := range shippedCases {
		p := kernel.RenderPair{Component: c.Component, Transformer: c.Transformer}
		t.Run(c.Name, func(t *testing.T) {
			assertParity(t, c, kernelRender(res, renderErr, p), oracleRender(oracle, p))
		})
	}
}

const shippedCatalogPrefix = "opmodel.dev/catalogs/opm/transformers/"

// shippedCases is the table for the shipped group, one row per pair the
// oracle matches for the web_app fixture. The oracle's pair list is
// asserted to equal this set, so a new pair cannot go untested. Every row
// is structural with no expected divergence (0019 D4: the table is empty of
// divergences once the enhancement is implemented).
var shippedCases = []parityCase{
	{
		Name:        "config :: configmap-transformer",
		Instance:    "testdata/parity/instance",
		Component:   "config",
		Transformer: shippedCatalogPrefix + "configmap-transformer@4.0.1",
		Equality:    equalityStructural,
	},
	{
		Name:        "web :: deployment-transformer",
		Instance:    "testdata/parity/instance",
		Component:   "web",
		Transformer: shippedCatalogPrefix + "deployment-transformer@4.0.1",
		Equality:    equalityStructural,
	},
	{
		Name:        "web :: hpa-transformer",
		Instance:    "testdata/parity/instance",
		Component:   "web",
		Transformer: shippedCatalogPrefix + "hpa-transformer@4.0.1",
		Equality:    equalityStructural,
	},
	{
		Name:        "web :: http-route-transformer",
		Instance:    "testdata/parity/instance",
		Component:   "web",
		Transformer: shippedCatalogPrefix + "http-route-transformer@4.0.1",
		Equality:    equalityStructural,
	},
	{
		// The guarded-env component (0019 D14): the env MAP assembled from
		// plain fields, a feature-guarded block and a comprehension, folded
		// into the Kubernetes env LIST by the deployment transformer, so any
		// hoisting of comprehension-produced fields reaches rendered bytes.
		Name:        "worker :: deployment-transformer",
		Instance:    "testdata/parity/instance",
		Component:   "worker",
		Transformer: shippedCatalogPrefix + "deployment-transformer@4.0.1",
		Equality:    equalityStructural,
	},
	{
		Name:        "worker :: hpa-transformer",
		Instance:    "testdata/parity/instance",
		Component:   "worker",
		Transformer: shippedCatalogPrefix + "hpa-transformer@4.0.1",
		Equality:    equalityStructural,
	},
	{
		Name:        "web :: service-transformer",
		Instance:    "testdata/parity/instance",
		Component:   "web",
		Transformer: shippedCatalogPrefix + "service-transformer@4.0.1",
		Equality:    equalityStructural,
	},
}

// assertRowsCoverPairs requires the case table and the oracle's pair list to
// name the same (component, transformer) set.
func assertRowsCoverPairs(t *testing.T, rows []parityCase, pairs []kernel.RenderPair) {
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
func oraclePairs(t *testing.T, oracle cue.Value) []kernel.RenderPair {
	t.Helper()
	var raw []struct {
		Component   string `json:"component"`
		Transformer string `json:"transformer"`
	}
	require.NoError(t, oracle.LookupPath(cue.ParsePath("pairs")).Decode(&raw), "decoding oracle pairs")
	pairs := make([]kernel.RenderPair, 0, len(raw))
	for _, r := range raw {
		pairs = append(pairs, kernel.RenderPair{Component: r.Component, Transformer: r.Transformer})
	}
	sortPairs(pairs)
	return pairs
}

// oracleRender reads the oracle's rendered output for one pair, normalised
// to one value per object the way Render splits a list output.
func oracleRender(oracle cue.Value, p kernel.RenderPair) parityRender {
	key := p.Component + " :: " + p.Transformer
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

// kernelRender selects Render's objects for one pair, in the order Render
// returned them (the build's pair order, then output order). A render error
// is attributed to every pair, because the gate reports refusals as one
// aggregate.
func kernelRender(res *kernel.RenderResult, renderErr error, p kernel.RenderPair) parityRender {
	if renderErr != nil {
		return parityRender{Err: renderErr}
	}
	var objs []cue.Value
	for _, c := range res.Compiled {
		if c.Component == p.Component && c.Transformer == p.Transformer {
			objs = append(objs, c.Value)
		}
	}
	return parityRender{Objects: objs}
}

// assertPairSetsAgree compares Render's matched pairs with the oracle's.
// No exemption exists (0019 D10): the always-unify rung is plain unification
// on both sides, so the two sets must be equal.
func assertPairSetsAgree(t *testing.T, kernelPairs, oracle []kernel.RenderPair) {
	t.Helper()
	kset := pairSet(kernelPairs)
	oset := pairSet(oracle)
	var onlyKernel, onlyOracle []string
	for k := range kset {
		if !oset[k] {
			onlyKernel = append(onlyKernel, k)
		}
	}
	for k := range oset {
		if !kset[k] {
			onlyOracle = append(onlyOracle, k)
		}
	}
	sort.Strings(onlyKernel)
	sort.Strings(onlyOracle)
	assert.Emptyf(t, onlyKernel, "pairs Render matched and the oracle did not")
	assert.Emptyf(t, onlyOracle, "pairs the oracle matched and Render did not")
}

func pairSet(ps []kernel.RenderPair) map[string]bool {
	m := make(map[string]bool, len(ps))
	for _, p := range ps {
		m[p.Component+" :: "+p.Transformer] = true
	}
	return m
}

func sortPairs(ps []kernel.RenderPair) {
	sort.Slice(ps, func(i, j int) bool {
		if ps[i].Component != ps[j].Component {
			return ps[i].Component < ps[j].Component
		}
		return ps[i].Transformer < ps[j].Transformer
	})
}
