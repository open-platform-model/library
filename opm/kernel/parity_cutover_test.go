package kernel_test

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/compile"
	loaderfile "github.com/open-platform-model/library/opm/helper/loader/file"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/module"
)

// The old-versus-new proof (openspec change library-render-cutover, PR 1;
// spec render-parity "The old path was proven equal to Render before
// deletion"). Every shipped parity case is rendered twice from ONE acquired
// instance: through the old path (Materialize + Compile against
// testdata/parity/opm_platform, the subscription form on the alpha.6 pin)
// and through Kernel.Render (against testdata/parity/opm_platform_next, the
// #CatalogEntry form on alpha.7). Both platforms name the same published
// catalogs/opm build, so the comparison is between two render paths over one
// set of catalog bytes.
//
// The comparison is the harness's own: compareRendered, order-sensitive
// (0019 D14). A difference that is ordering-only is recorded per case
// through the shared table's ExpectedDivergence, exactly as the oracle
// comparison records it; the cause is the same on both comparisons (the
// old path builds #context in Go and re-emits label maps sorted, while
// Render hands the transformer core's projection in CUE's evaluation
// order), so the record is shared rather than duplicated. A difference in
// any VALUE fails the case outright: that is a defect in Render or the glue
// and blocks PR 2.
//
// The probe group (parity_probe_test.go) runs the same comparison per probe
// through renderProbe below, against a #CatalogEntry platform written for
// the served probe catalog; its cases carry no label maps and agree exactly.
//
// Deleted with the old path in PR 2; the archived change is its record.
func TestParity_CutoverCompileVersusRender(t *testing.T) {
	if testing.Short() {
		t.Skip("cutover proof pulls the catalog + core schema from GHCR; skipping under -short")
	}
	skipUnlessRegistry(t)

	parityDir := filepath.Join(repoLibraryRoot(t), "testdata", "parity")
	registry := flowRegistry()
	t.Setenv("CUE_REGISTRY", registry)
	ctx := context.Background()
	opts := loaderfile.LoadOptions{Registry: registry}
	k := kernel.New(kernel.WithRegistry(registry))

	// One instance, acquired once, handed to both paths. It carries the
	// Source Render needs and is processed exactly as the old harness
	// processes it (ProcessModuleInstance with no extra values).
	inst, err := k.AcquireInstanceFromDir(ctx, filepath.Join(parityDir, "instance"), opts)
	require.NoError(t, err, "acquiring the parity instance package")

	// ── old path: subscription platform, materialized, compiled ────────
	platVal, err := k.LoadPlatformPackage(ctx, filepath.Join(parityDir, "opm_platform"), opts)
	require.NoError(t, err, "loading the subscription-form platform")
	oldPlat, err := k.NewPlatformFromValue(platVal)
	require.NoError(t, err)
	mp, err := k.Materialize(ctx, oldPlat)
	require.NoError(t, err, "materializing the subscription-form platform against the published catalog")
	out, compileErr := k.Compile(ctx, kernel.CompileInput{ModuleInstance: inst, Platform: mp, RuntimeName: parityRuntimeName})
	require.NoError(t, compileErr, "the old path must render the parity instance")

	// ── new path: #CatalogEntry platform, one build ────────────────────
	nextPlat, err := k.AcquirePlatformFromDir(ctx, filepath.Join(parityDir, "opm_platform_next"), opts)
	require.NoError(t, err, "acquiring the #CatalogEntry-form platform")
	res, err := k.Render(ctx, kernel.RenderInput{Instance: inst, Platform: nextPlat, RuntimeName: parityRuntimeName})
	require.NoError(t, err, "Render must render the parity instance")
	assert.Empty(t, res.Warnings, "the instance requires no newer build than the platform carries")

	// ── same catalog bytes on both sides ───────────────────────────────
	subscribed, err := platVal.LookupPath(cue.ParsePath(`#registry."opmodel.dev/catalogs/opm@v4".version`)).String()
	require.NoError(t, err, "the subscription form names its catalog build")
	assert.Contains(t, res.Diagnostics.ResolvedVersions, kernel.ResolvedVersion{
		Path:            "opmodel.dev/catalogs/opm@v4",
		ModuleVersion:   "v" + subscribed,
		PlatformVersion: "v" + subscribed,
	}, "the #CatalogEntry form resolves the build the subscription form pulls")

	// ── pair sets agree, with no exemption ─────────────────────────────
	oldPairs := out.MatchPlan.MatchedPairs()
	sortPairs(oldPairs)
	newPairs := make([]compile.MatchedPair, 0, len(res.Diagnostics.Pairs))
	for _, p := range res.Diagnostics.Pairs {
		newPairs = append(newPairs, compile.MatchedPair{ComponentName: p.Component, TransformerFQN: p.Transformer})
	}
	sortPairs(newPairs)
	assert.Equal(t, oldPairs, newPairs, "Compile and Render must match the same (component, transformer) pairs")
	assertRowsCoverPairs(t, shippedCases, newPairs)

	// ── cases ──────────────────────────────────────────────────────────
	for _, c := range shippedCases {
		p := compile.MatchedPair{ComponentName: c.Component, TransformerFQN: c.Transformer}
		t.Run(c.Name, func(t *testing.T) {
			assertCutover(t, c, kernelRender(out, compileErr, p), renderObjects(res, p))
		})
	}
}

// renderProbe is the probe group's Render side (parity_probe_test.go): it
// writes a #CatalogEntry-form platform for the served probe catalog as a
// nested module under dir (core alpha.7, the catalog at the build the old
// path materialized), acquires it, and renders inst against it.
func renderProbe(t *testing.T, k *kernel.Kernel, dir, mapping, catPath, version string, inst *module.Instance, pair compile.MatchedPair) parityRender {
	t.Helper()
	platDir := filepath.Join(dir, "platform")
	writeFile(t, filepath.Join(platDir, "cue.mod", "module.cue"), fmt.Sprintf(`module: "testing.opmodel.dev/library-parity-probe/platform@v0"
language: version: "v0.17.0"
deps: {
	"opmodel.dev/core@v2": v: "v2.0.0-alpha.7"
	%q: v: %q
}
`, catPath+"@v0", "v"+version))
	writeFile(t, filepath.Join(platDir, "platform.cue"), fmt.Sprintf(`package platform

import (
	core "opmodel.dev/core@v2"
	cat %q
)

core.#Platform
metadata: name: "probe"
type: "kubernetes"
#registry: %q: {
	enable:   true
	#catalog: cat
}
`, catPath+"@v0", catPath+"@v0"))

	ctx := context.Background()
	plat, err := k.AcquirePlatformFromDir(ctx, platDir, loaderfile.LoadOptions{Registry: mapping})
	require.NoError(t, err, "acquiring the #CatalogEntry-form probe platform")
	res, err := k.Render(ctx, kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: parityRuntimeName})
	if err != nil {
		return parityRender{Err: err}
	}
	return renderObjects(res, pair)
}

// renderObjects selects Render's Compiled objects for one pair, in the order
// Render returned them (the build's pair order, then output order).
func renderObjects(res *kernel.RenderResult, p compile.MatchedPair) parityRender {
	var objs []cue.Value
	for _, c := range res.Compiled {
		if c.Component == p.ComponentName && c.Transformer == p.TransformerFQN {
			objs = append(objs, c.Value)
		}
	}
	return parityRender{Objects: objs}
}

// checkCutover applies the proof's contract to one case: both paths MUST
// render; the rendered objects MUST compare equal order-sensitively, or
// differ only in ordering with the difference recorded on the case. A value
// difference is never admitted, recorded or not.
func checkCutover(c parityCase, old, next parityRender) error {
	if old.Err != nil {
		return fmt.Errorf("case %q (%s :: %s): the old path failed to render: %w", c.Name, c.Component, c.Transformer, old.Err)
	}
	if next.Err != nil {
		return fmt.Errorf("case %q (%s :: %s): Render failed: %w", c.Name, c.Component, c.Transformer, next.Err)
	}
	divergence := compareRendered(old.Objects, next.Objects)
	if divergence == "" {
		if c.ExpectedDivergence != "" {
			return fmt.Errorf("case %q (%s :: %s): expected divergence %q no longer reproduces between Compile and Render; delete this case's ExpectedDivergence entry (0019 D4)",
				c.Name, c.Component, c.Transformer, c.ExpectedDivergence)
		}
		return nil
	}
	if !orderingOnly(old.Objects, next.Objects) {
		return fmt.Errorf("case %q (%s :: %s): Compile and Render differ in a VALUE, not only in ordering; this is a defect in Render or the glue and blocks the cutover\n%s",
			c.Name, c.Component, c.Transformer, divergence)
	}
	if c.ExpectedDivergence == "" {
		return fmt.Errorf("case %q (%s :: %s): Compile and Render differ in ordering only, but the case records no divergence; record it (0019 D14)\n%s",
			c.Name, c.Component, c.Transformer, divergence)
	}
	return nil
}

// orderingOnly reports whether two object lists carry the same fields and
// values once struct field order (and, failing that, list element order) is
// disregarded. Classification only: the comparison itself is compareRendered.
func orderingOnly(a, b []cue.Value) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		ea, errA := encodeOrdered(a[i])
		eb, errB := encodeOrdered(b[i])
		if errA != nil || errB != nil {
			return false
		}
		if !equalModuloOrder(ea, eb) && !equalModuloAllOrder(ea, eb) {
			return false
		}
	}
	return true
}

// assertCutover is checkCutover as a test assertion, logging a recorded
// ordering-only divergence so the evidence stays visible in -v output.
func assertCutover(t *testing.T, c parityCase, old, next parityRender) {
	t.Helper()
	require.NoError(t, checkCutover(c, old, next))
	if c.ExpectedDivergence != "" {
		t.Logf("case %q: ordering-only divergence between Compile and Render reproduces (%s): %s",
			c.Name, c.ExpectedDivergence, firstLine(strings.TrimSpace(compareRendered(old.Objects, next.Objects))))
	}
}
