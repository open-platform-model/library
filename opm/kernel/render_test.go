package kernel_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/compile"
	"github.com/open-platform-model/library/opm/core"
	oerrors "github.com/open-platform-model/library/opm/errors"
	loaderfile "github.com/open-platform-model/library/opm/helper/loader/file"
	"github.com/open-platform-model/library/opm/helper/synth"
	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/internal/schematest"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/platform"
)

// The render fixtures (testdata/render): a catalog and a module served by the
// in-process registry, an on-disk D5 platform importing the catalog, an
// on-disk instance importing the module, and one scenario instance package
// per outcome. Every fixture cue.mod pins core 2.0.0-alpha.7 (the D5/D17
// prerelease) explicitly.
const (
	renderPrefix  = "testing.opmodel.dev/library-render"
	renderCatPath = renderPrefix + "/cat"
	renderModPath = renderPrefix + "/web_app"
	renderTxPath  = renderCatPath + "/transformers"
)

func renderFixtureDir(t *testing.T, parts ...string) string {
	t.Helper()
	return filepath.Join(append([]string{schematest.LibraryRoot(t), "testdata", "render"}, parts...)...)
}

// newRenderKernel serves testdata/render/registry in-process and returns a
// kernel routing the fixture prefix to it (core still resolves from GHCR /
// the warm workspace cache).
func newRenderKernel(t *testing.T) *kernel.Kernel {
	t.Helper()
	mapping := registrytest.NewRegistryFromDir(t, renderFixtureDir(t, "registry"), renderPrefix)
	return kernel.New(kernel.WithRegistry(mapping))
}

func acquireRenderPlatform(t *testing.T, k *kernel.Kernel, dir string) *platform.Platform {
	t.Helper()
	p, err := k.AcquirePlatformFromDir(context.Background(), renderFixtureDir(t, dir), loaderfile.LoadOptions{})
	require.NoError(t, err, "acquiring platform fixture %s", dir)
	require.NotNil(t, p.Source)
	return p
}

func acquireRenderInstance(t *testing.T, k *kernel.Kernel, parts ...string) *module.Instance {
	t.Helper()
	inst, err := k.AcquireInstanceFromDir(context.Background(), renderFixtureDir(t, parts...), loaderfile.LoadOptions{})
	require.NoError(t, err, "acquiring instance fixture %v", parts)
	require.NotNil(t, inst.Source)
	return inst
}

func renderPairSet(pairs []kernel.RenderPair) []string {
	out := make([]string, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, p.Component+" :: "+strings.TrimPrefix(p.Transformer, renderTxPath+"/"))
	}
	sort.Strings(out)
	return out
}

func compiledSummary(t *testing.T, compiled []*core.Compiled) []string {
	t.Helper()
	out := make([]string, 0, len(compiled))
	for _, c := range compiled {
		kind, err := c.Value.LookupPath(cue.ParsePath("kind")).String()
		require.NoError(t, err)
		name, err := c.Value.LookupPath(cue.ParsePath("metadata.name")).String()
		require.NoError(t, err)
		out = append(out, c.Component+"/"+kind+"/"+name)
	}
	return out
}

func TestRender_HappyOnDiskInputs(t *testing.T) {
	k := newRenderKernel(t)
	plat := acquireRenderPlatform(t, k, "platform")
	inst := acquireRenderInstance(t, k, "instance")

	res, err := k.Render(context.Background(), kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "render-test"})
	require.NoError(t, err)

	assert.Equal(t, []string{
		"config :: configmap-transformer@0.1.0",
		"web :: deployment-transformer@0.1.0",
		"web :: service-transformer@0.1.0",
	}, renderPairSet(res.Diagnostics.Pairs))
	assert.Empty(t, res.Diagnostics.Unresolved)
	assert.Empty(t, res.Diagnostics.Unify)
	assert.Empty(t, res.Diagnostics.Unmatched)
	assert.Empty(t, res.Diagnostics.FailedPairs)
	assert.Empty(t, res.Warnings)

	// Provenance on every object; the ConfigMap list splits per item.
	assert.ElementsMatch(t, []string{
		"config/ConfigMap/app",
		"web/Deployment/web-demo-web",
		"web/Service/web-demo-web",
	}, compiledSummary(t, res.Compiled))
	for _, c := range res.Compiled {
		assert.Equal(t, "web-demo", c.Instance)
		assert.NotEmpty(t, c.Component)
		assert.True(t, strings.HasPrefix(c.Transformer, renderTxPath+"/"), c.Transformer)
		managedBy, err := c.Value.LookupPath(cue.ParsePath(`metadata.labels."app.kubernetes.io/managed-by"`)).String()
		require.NoError(t, err)
		assert.Equal(t, "render-test", managedBy, "#runtimeName reaches every rendered object through core's #context projection")
	}

	// D18 rows: every OPM path the instance requires, as data. The module
	// path is instance-only (the platform does not list it), so its row
	// carries no platform version.
	assert.Equal(t, []kernel.ResolvedVersion{
		{Path: "opmodel.dev/core@v2", ModuleVersion: "v2.0.0-alpha.7", PlatformVersion: "v2.0.0-alpha.7"},
		{Path: renderCatPath + "@v0", ModuleVersion: "v0.1.0", PlatformVersion: "v0.1.0"},
		{Path: renderModPath + "@v0", ModuleVersion: "v0.1.0"},
	}, res.Diagnostics.ResolvedVersions)
}

// synthRenderInstance acquires the fixture module from the in-process registry
// and synthesizes an overlay-mode instance from it (the CLI / operator path).
func synthRenderInstance(t *testing.T, k *kernel.Kernel, version string) *module.Instance {
	t.Helper()
	ctx := context.Background()
	mod, err := k.AcquireModuleFromRegistry(ctx, renderModPath+"@v0", "v"+version)
	require.NoError(t, err, "acquiring fixture module %s", version)
	require.True(t, mod.HasSource())
	inst, err := k.SynthesizeInstance(ctx, synth.InstanceInput{
		Module:      mod,
		Name:        "web-synth",
		Namespace:   "default",
		Values:      k.CueContext().CompileString(`{image: "nginx:1.27", replicas: 3}`),
		SchemaCache: k.SchemaCache(),
	})
	require.NoError(t, err, "synthesizing overlay-mode instance")
	require.NotNil(t, inst.Source)
	require.NotEmpty(t, inst.Source.Overlay, "synth instances are overlay-mode")
	return inst
}

func TestRender_OverlayModeInstance(t *testing.T) {
	k := newRenderKernel(t)
	plat := acquireRenderPlatform(t, k, "platform")
	inst := synthRenderInstance(t, k, "0.1.0")

	res, err := k.Render(context.Background(), kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "render-test"})
	require.NoError(t, err, "an overlay-mode instance is materialized into the staging directory and builds")
	assert.Equal(t, []string{
		"config :: configmap-transformer@0.1.0",
		"web :: deployment-transformer@0.1.0",
		"web :: service-transformer@0.1.0",
	}, renderPairSet(res.Diagnostics.Pairs))
	assert.ElementsMatch(t, []string{
		"config/ConfigMap/app",
		"web/Deployment/web-synth-web",
		"web/Service/web-synth-web",
	}, compiledSummary(t, res.Compiled))
	for _, c := range res.Compiled {
		assert.Equal(t, "web-synth", c.Instance)
	}
	// The instance's own dependency list (the module's tidied cue.mod)
	// carries the catalog and the core pin; both are rows, neither is skew.
	assert.Equal(t, []kernel.ResolvedVersion{
		{Path: "opmodel.dev/core@v2", ModuleVersion: "v2.0.0-alpha.7", PlatformVersion: "v2.0.0-alpha.7"},
		{Path: renderCatPath + "@v0", ModuleVersion: "v0.1.0", PlatformVersion: "v0.1.0"},
	}, res.Diagnostics.ResolvedVersions)
}

func TestRender_RefusesSourcelessInputs(t *testing.T) {
	k := newRenderKernel(t)
	plat := acquireRenderPlatform(t, k, "platform")
	inst := acquireRenderInstance(t, k, "instance")
	ctx := context.Background()

	bare, err := platform.NewPlatformFromValue(k, plat.Package)
	require.NoError(t, err)
	require.Nil(t, bare.Source)
	_, err = k.Render(ctx, kernel.RenderInput{Instance: inst, Platform: bare, RuntimeName: "rt"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "platform")
	assert.Contains(t, err.Error(), "no Source")

	noSrc := *inst
	noSrc.Source = nil
	_, err = k.Render(ctx, kernel.RenderInput{Instance: &noSrc, Platform: plat, RuntimeName: "rt"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "instance")
	assert.Contains(t, err.Error(), "no Source")

	_, err = k.Render(ctx, kernel.RenderInput{Instance: inst, Platform: plat})
	require.ErrorContains(t, err, "RuntimeName")
	_, err = k.Render(ctx, kernel.RenderInput{Platform: plat, RuntimeName: "rt"})
	require.ErrorContains(t, err, "Instance is required")
	_, err = k.Render(ctx, kernel.RenderInput{Instance: inst, RuntimeName: "rt"})
	require.ErrorContains(t, err, "Platform is required")
	_, err = k.Render(ctx, kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt", Skew: kernel.SkewPolicy(7)})
	require.ErrorContains(t, err, "SkewPolicy")
}

func TestRender_MissingFQN_RefusesWithAlternatives(t *testing.T) {
	k := newRenderKernel(t)
	plat := acquireRenderPlatform(t, k, "platform")
	inst := acquireRenderInstance(t, k, "scenarios", "missing")
	assert.Equal(t, "missing", inst.Source.Pkg, "a subpackage acquisition stamps the enclosing module root and the package dir")

	_, err := k.Render(context.Background(), kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
	require.Error(t, err)

	var rerr *kernel.RenderError
	require.ErrorAs(t, err, &rerr, "post-build refusals carry the decoded diagnostics")
	var agg *oerrors.UnresolvedDemandsError
	require.ErrorAs(t, err, &agg)

	byFQN := map[string]oerrors.UnresolvedDemand{}
	for _, d := range rerr.Diagnostics.Unresolved {
		byFQN[d.FQN] = d
	}
	orphan := byFQN[renderCatPath+"/resources/orphan@v1"]
	assert.Equal(t, "orphan", orphan.Component)
	assert.Equal(t, "resource", orphan.Kind)
	assert.Empty(t, orphan.Disqualified, "an empty bucket has no disqualified candidates")
	assert.Equal(t, []string{renderCatPath + "/resources/orphan@v2"}, orphan.Alternatives,
		"the platform implements the same contract base at another apiVersion")

	backup := byFQN[renderCatPath+"/traits/backup@v1"]
	assert.Equal(t, "trait", backup.Kind, "a load-bearing unhandled trait is an unresolved demand")
	assert.Empty(t, backup.Alternatives)

	// The sibling verdict stays readable beside the refusal: config-maps
	// still paired.
	assert.Equal(t, []string{"orphan :: configmap-transformer@0.1.0"}, renderPairSet(rerr.Diagnostics.Pairs))
	assert.Empty(t, rerr.Diagnostics.Unmatched)
	assert.Len(t, agg.Demands, 2)
}

func TestRender_DisqualifiedCandidateIsData(t *testing.T) {
	k := newRenderKernel(t)
	plat := acquireRenderPlatform(t, k, "platform")
	inst := acquireRenderInstance(t, k, "scenarios", "disqualified")

	_, err := k.Render(context.Background(), kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
	require.Error(t, err)
	var rerr *kernel.RenderError
	require.ErrorAs(t, err, &rerr)

	narrowFQN := renderCatPath + "/resources/narrow@v1"
	narrowTx := renderTxPath + "/narrow-transformer@0.1.0"
	require.Len(t, rerr.Diagnostics.Unify, 1, "the always-unify rung disqualified the only candidate")
	assert.Equal(t, "narrow", rerr.Diagnostics.Unify[0].Component)
	assert.Equal(t, narrowFQN, rerr.Diagnostics.Unify[0].FQN)
	assert.Contains(t, rerr.Diagnostics.Unify[0].Cause.Error(), narrowTx)

	require.Len(t, rerr.Diagnostics.Unresolved, 1)
	d := rerr.Diagnostics.Unresolved[0]
	assert.Equal(t, narrowFQN, d.FQN)
	assert.Len(t, d.Disqualified, 1, "the demand names its disqualified candidate")
	assert.Equal(t, narrowFQN, d.Disqualified[0].FQN)
	assert.Empty(t, d.Alternatives)
	assert.Equal(t, []string{"narrow"}, rerr.Diagnostics.Unmatched)

	var unmatched *compile.UnmatchedComponentsError
	require.ErrorAs(t, err, &unmatched)
	assert.Equal(t, []string{"narrow"}, unmatched.Components)
}

func TestRender_EffectivelyOptionalTraitWarns(t *testing.T) {
	k := newRenderKernel(t)
	plat := acquireRenderPlatform(t, k, "platform")
	inst := acquireRenderInstance(t, k, "scenarios", "warning")

	res, err := k.Render(context.Background(), kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
	require.NoError(t, err, "an advisory unhandled trait degrades to a warning")
	assert.Equal(t, []string{"web :: deployment-transformer@0.1.0"}, renderPairSet(res.Diagnostics.Pairs))
	sidecar := renderCatPath + "/traits/sidecar@v1"
	assert.Equal(t, map[string][]string{"web": {sidecar}}, res.Diagnostics.UnhandledTraits)
	require.Len(t, res.Warnings, 1)
	assert.Contains(t, res.Warnings[0], sidecar)
	assert.Contains(t, res.Warnings[0], "not handled by any matched transformer")
	assert.Equal(t, []string{"web/Deployment/warning-demo-web"}, compiledSummary(t, res.Compiled))
}

func TestRender_UnstatedPostureIsBuildError(t *testing.T) {
	k := newRenderKernel(t)
	plat := acquireRenderPlatform(t, k, "platform")
	inst := acquireRenderInstance(t, k, "scenarios", "unstated")

	_, err := k.Render(context.Background(), kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
	require.Error(t, err, "an unhandled trait with an unstated posture refuses (fail-closed)")
	assert.Contains(t, err.Error(), "optional", "the refusal names the trait's own optional field")
	assert.Contains(t, err.Error(), renderCatPath+"/traits/unstated@v1")
	assert.Contains(t, err.Error(), `component "web"`)
	var rerr *kernel.RenderError
	assert.False(t, errors.As(err, &rerr), "the refusal is a build error, not a diagnostics row (measured boundary, 0019 D10)")
}

func TestRender_IncompletePairRefusesNamingPair(t *testing.T) {
	k := newRenderKernel(t)
	plat := acquireRenderPlatform(t, k, "platform")
	inst := acquireRenderInstance(t, k, "scenarios", "incomplete")

	_, err := k.Render(context.Background(), kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
	require.Error(t, err)
	var rerr *kernel.RenderError
	require.ErrorAs(t, err, &rerr)
	assert.Empty(t, rerr.Diagnostics.FailedPairs, "an incomplete output is not bottom: invisible to the glue's guards")
	var terr *oerrors.TransformError
	require.ErrorAs(t, err, &terr, "the kernel's own concreteness check names the pair")
	assert.Equal(t, "hole", terr.ComponentName)
	assert.Equal(t, renderTxPath+"/incomplete-transformer@0.1.0", terr.TransformerFQN)
	assert.Contains(t, terr.Cause.Error(), "not concrete")
	assert.Contains(t, terr.Cause.Error(), "metadata.name")
	// The healthy sibling pair was matched and is not blamed.
	assert.Equal(t, []string{
		"hole :: incomplete-transformer@0.1.0",
		"web :: deployment-transformer@0.1.0",
	}, renderPairSet(rerr.Diagnostics.Pairs))
	assert.NotContains(t, err.Error(), `component "web"`)
}

func TestRender_FailingPairIsDataBesideHealthy(t *testing.T) {
	k := newRenderKernel(t)
	plat := acquireRenderPlatform(t, k, "platform")
	inst := acquireRenderInstance(t, k, "scenarios", "failing")

	_, err := k.Render(context.Background(), kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
	require.Error(t, err)
	var rerr *kernel.RenderError
	require.ErrorAs(t, err, &rerr)
	brokenTx := renderTxPath + "/broken-transformer@0.1.0"
	assert.Equal(t, []kernel.RenderPair{{Component: "crash", Transformer: brokenTx}}, rerr.Diagnostics.FailedPairs,
		"an error-class pair failure is reported as data naming the pair")
	assert.Equal(t, []string{
		"crash :: broken-transformer@0.1.0",
		"web :: deployment-transformer@0.1.0",
	}, renderPairSet(rerr.Diagnostics.Pairs), "sibling verdicts stay readable")
	var terr *oerrors.TransformError
	require.ErrorAs(t, err, &terr)
	assert.Equal(t, "crash", terr.ComponentName)
	assert.Equal(t, brokenTx, terr.TransformerFQN)
	assert.NotContains(t, err.Error(), `component "web"`)
}

func TestRender_Skew_NewerModuleWarnsByDefault(t *testing.T) {
	k := newRenderKernel(t)
	plat := acquireRenderPlatform(t, k, "platform") // carries cat 0.1.0
	inst := synthRenderInstance(t, k, "0.2.0")      // requires cat 0.2.0

	res, err := k.Render(context.Background(), kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
	require.NoError(t, err, "warn-and-render is the default")
	require.Len(t, res.Warnings, 1)
	assert.Contains(t, res.Warnings[0], renderCatPath+"@v0")
	assert.Contains(t, res.Warnings[0], "v0.2.0")
	assert.Contains(t, res.Warnings[0], "v0.1.0")
	assert.Equal(t, []kernel.ResolvedVersion{
		{Path: "opmodel.dev/core@v2", ModuleVersion: "v2.0.0-alpha.7", PlatformVersion: "v2.0.0-alpha.7"},
		{Path: renderCatPath + "@v0", ModuleVersion: "v0.2.0", PlatformVersion: "v0.1.0", Newer: true},
	}, res.Diagnostics.ResolvedVersions)
	// The platform's bytes executed: every transformer FQN carries 0.1.0.
	for _, p := range res.Diagnostics.Pairs {
		assert.True(t, strings.HasSuffix(p.Transformer, "@0.1.0"), p.Transformer)
	}
}

func TestRender_Skew_RefusePolicyStopsBeforeEvaluation(t *testing.T) {
	k := newRenderKernel(t)
	plat := acquireRenderPlatform(t, k, "platform")
	inst := synthRenderInstance(t, k, "0.2.0")

	_, err := k.Render(context.Background(), kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt", Skew: kernel.SkewRefuse})
	require.Error(t, err)
	var skew *oerrors.SkewError
	require.ErrorAs(t, err, &skew)
	assert.Equal(t, renderCatPath+"@v0", skew.Path)
	assert.Equal(t, "v0.2.0", skew.ModuleVersion)
	assert.Equal(t, "v0.1.0", skew.PlatformVersion)
	assert.Contains(t, err.Error(), "before evaluation")
	var rerr *kernel.RenderError
	assert.False(t, errors.As(err, &rerr), "no build ran")
}

func TestRender_Skew_OlderModuleIsData(t *testing.T) {
	k := newRenderKernel(t)
	plat := acquireRenderPlatform(t, k, "platform_next") // carries cat 0.2.0
	inst := synthRenderInstance(t, k, "0.1.0")           // requires cat 0.1.0

	res, err := k.Render(context.Background(), kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt", Skew: kernel.SkewRefuse})
	require.NoError(t, err, "older-than-platform is not skew, even under the refuse policy")
	assert.Empty(t, res.Warnings)
	assert.Equal(t, []kernel.ResolvedVersion{
		{Path: "opmodel.dev/core@v2", ModuleVersion: "v2.0.0-alpha.7", PlatformVersion: "v2.0.0-alpha.7"},
		{Path: renderCatPath + "@v0", ModuleVersion: "v0.1.0", PlatformVersion: "v0.2.0"},
	}, res.Diagnostics.ResolvedVersions)
	for _, p := range res.Diagnostics.Pairs {
		assert.True(t, strings.HasSuffix(p.Transformer, "@0.2.0"), "the platform's build executed: %s", p.Transformer)
	}
}

// stagingDirs lists the opm-render-* staging directories currently present
// under the process temp dir (what Render leaves behind if it fails to clean
// up).
func stagingDirs(t *testing.T) []string {
	t.Helper()
	dirs, err := filepath.Glob(filepath.Join(os.TempDir(), "opm-render-*"))
	require.NoError(t, err)
	sort.Strings(dirs)
	return dirs
}

func TestRender_RepeatedRendersShareNothing(t *testing.T) {
	k := newRenderKernel(t)
	plat := acquireRenderPlatform(t, k, "platform")
	inst := synthRenderInstance(t, k, "0.1.0")
	ctx := context.Background()

	before := stagingDirs(t)
	first, err := k.Render(ctx, kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
	require.NoError(t, err)
	second, err := k.Render(ctx, kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
	require.NoError(t, err)
	assert.Equal(t, before, stagingDirs(t), "each render removes its staging directory on return")

	assert.Equal(t, first.Diagnostics, second.Diagnostics)
	assert.Equal(t, first.Warnings, second.Warnings)
	require.Len(t, second.Compiled, len(first.Compiled))
	for i := range first.Compiled {
		a, err := first.Compiled[i].Value.MarshalJSON()
		require.NoError(t, err)
		b, err := second.Compiled[i].Value.MarshalJSON()
		require.NoError(t, err)
		assert.Equal(t, string(a), string(b), "render %d is byte-identical", i)
		assert.Equal(t, first.Compiled[i].Component, second.Compiled[i].Component)
		assert.Equal(t, first.Compiled[i].Transformer, second.Compiled[i].Transformer)
	}
	// Value.Context is deprecated for combining values, which is not what
	// happens here: it is read only to assert the D8 lifetime rule (each
	// render builds in its own context, never the Kernel's).
	assert.NotSame(t, first.Compiled[0].Value.Context(), second.Compiled[0].Value.Context(), //nolint:staticcheck // D8 lifetime assertion, not value combination
		"each render builds in its own context")
	assert.NotSame(t, k.CueContext(), first.Compiled[0].Value.Context(), "the Kernel's context is not the render context") //nolint:staticcheck // same
}

// TestRender_PairSetMatchesOldPath renders the fixture module on the new path
// and compiles an equivalent hermetic fixture on the old path (registrytest
// catalog + Materialize + Match + Compile), asserting the two pair sets agree
// once the catalog path is normalized away. The old-path fixture is built by
// the same generators the old-path suite uses, against the core the old path
// is pinned to.
func TestRender_PairSetMatchesOldPath(t *testing.T) {
	// New path.
	nk := newRenderKernel(t)
	plat := acquireRenderPlatform(t, nk, "platform")
	inst := synthRenderInstance(t, nk, "0.1.0")
	res, err := nk.Render(context.Background(), kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
	require.NoError(t, err)
	newPairs := renderPairSet(res.Diagnostics.Pairs)

	// Old path: a catalog with the same transformer/demand shape.
	const version = "0.1.0"
	catPath := registrytest.UniquePath(t, "oldpath")
	ok := newKernelWithCatalogs(t, registrytest.CatalogFixture{
		Path:    catPath,
		Version: version,
		Body: registrytest.BuildCatalog(catPath, version,
			registrytest.TxFixture{Name: "deployment-transformer", Resources: []string{"container"}},
			registrytest.TxFixture{Name: "service-transformer", Resources: []string{"container"}, Traits: []string{"expose"}},
			registrytest.TxFixture{Name: "configmap-transformer", Resources: []string{"config-maps"}},
		),
	})
	mp, err := materializePlatform(t, ok, version, catPath)
	require.NoError(t, err)
	oldInst := buildInstance(t, ok, catPath, version, "", "",
		compSpec{name: "web", resources: []string{"container"}, traits: []string{"expose"}},
		compSpec{name: "config", resources: []string{"config-maps"}},
	)
	out, err := ok.Compile(context.Background(), kernel.CompileInput{ModuleInstance: oldInst, Platform: mp, RuntimeName: "rt"})
	require.NoError(t, err)
	oldPairs := make([]string, 0)
	for _, p := range out.MatchPlan.MatchedPairs() {
		oldPairs = append(oldPairs, p.ComponentName+" :: "+strings.TrimPrefix(p.TransformerFQN, catPath+"/transformers/"))
	}
	sort.Strings(oldPairs)

	assert.Equal(t, oldPairs, newPairs, "the in-build matcher reproduces the Go matcher's pair set")
}

// The single-provider guard in-build (0010 D32/D37; library-render-cutover).
// platform_oversubscribed carries cat 0.1.0 and cat2 0.2.0, which both ship
// a transformer requiring cat's provider-fulfilled gateway contract.
func TestRender_OverSubscribedProviderRefused(t *testing.T) {
	k := newRenderKernel(t)
	plat := acquireRenderPlatform(t, k, "platform_oversubscribed")
	inst := acquireRenderInstance(t, k, "instance")

	_, err := k.Render(context.Background(), kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
	require.Error(t, err)
	var rerr *kernel.RenderError
	require.ErrorAs(t, err, &rerr, "the refusal is the gate, with the decoded diagnostics beside it")

	gateway := renderCatPath + "/resources/gateway@v1"
	var ose oerrors.OverSubscribedContractError
	require.ErrorAs(t, err, &ose)
	assert.Equal(t, gateway, ose.Key)
	assert.Equal(t, []string{renderPrefix + "/cat2@v0", renderCatPath + "@v0"}, ose.Catalogs,
		"provenance is the registry key of each supplying entry, sorted")
	assert.Equal(t, []oerrors.OverSubscribedContractError{ose}, rerr.Diagnostics.OverSubscribed)
	assert.Contains(t, err.Error(), `fulfilment "provider"`)
	assert.Contains(t, err.Error(), "exactly one provider")

	// The matching verdicts stay readable beside the refusal, and nothing
	// else is wrong with this platform: the guard is the only cause.
	assert.Equal(t, []string{
		"config :: configmap-transformer@0.1.0",
		"web :: deployment-transformer@0.1.0",
		"web :: service-transformer@0.1.0",
		"web :: " + renderPrefix + "/cat2/transformers/mirror-transformer@0.2.0",
	}, renderPairSet(rerr.Diagnostics.Pairs))
	assert.Empty(t, rerr.Diagnostics.Unresolved)
	assert.Empty(t, rerr.Diagnostics.Unmatched)
	assert.Empty(t, rerr.Diagnostics.FailedPairs)
	var unresolved *oerrors.UnresolvedDemandsError
	assert.False(t, errors.As(err, &unresolved))
}

// platform_two carries cat 0.1.0 and cat2 0.1.0: two catalogs supplying
// transformers for the container contract, whose fulfilment is the default
// (catalog). Plurality is admitted and every candidate participates.
func TestRender_CatalogFulfilledPluralityRenders(t *testing.T) {
	k := newRenderKernel(t)
	plat := acquireRenderPlatform(t, k, "platform_two")
	inst := acquireRenderInstance(t, k, "instance")

	res, err := k.Render(context.Background(), kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
	require.NoError(t, err, "catalog-fulfilled keys admit any number of suppliers")
	assert.Empty(t, res.Diagnostics.OverSubscribed)
	assert.Equal(t, []string{
		"config :: configmap-transformer@0.1.0",
		"web :: deployment-transformer@0.1.0",
		"web :: service-transformer@0.1.0",
		"web :: " + renderPrefix + "/cat2/transformers/mirror-transformer@0.1.0",
	}, renderPairSet(res.Diagnostics.Pairs))
	assert.ElementsMatch(t, []string{
		"config/ConfigMap/app",
		"web/Deployment/web-demo-web",
		"web/Mirror/web-demo-web",
		"web/Service/web-demo-web",
	}, compiledSummary(t, res.Compiled))
}
