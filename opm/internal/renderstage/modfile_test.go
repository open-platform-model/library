package renderstage

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/cue/parser"
	"cuelang.org/go/mod/modfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/module"
)

const platformModFile = `module: "testing.opmodel.dev/render/platform@v0"
language: version: "v0.17.0"
deps: {
	"cue.dev/x/k8s.io@v0": {
		v:       "v0.11.0"
		default: true
	}
	"opmodel.dev/catalogs/opm@v4": v: "v4.2.0"
	"opmodel.dev/core@v2": v: "v2.0.0-alpha.7"
}
`

const instanceModFile = `module: "testing.opmodel.dev/modules/web_app@v1"
language: version: "v0.17.1"
deps: {
	"opmodel.dev/catalogs/opm@v4": v: "v4.3.0"
	"opmodel.dev/core@v2": v: "v2.0.0-alpha.7"
	"example.com/helpers@v1": {
		v:       "v1.0.0"
		default: true
	}
}
`

func mustParse(t *testing.T, src, name string) *ModFile {
	t.Helper()
	mf, err := ParseModFile([]byte(src), name)
	require.NoError(t, err)
	return mf
}

// Test-only views of the two dependency maps in lexical order.
func (mf *ModFile) SortedPaths() []string { return sortedKeys(mf.Deps) }

func (p *Promotion) SortedPaths() []string { return sortedKeys(p.Deps) }

func sortedKeys(deps map[string]Dep) []string {
	paths := make([]string, 0, len(deps))
	for path := range deps {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func TestParseModFile_KeepsDefaultMajorMarkers(t *testing.T) {
	mf := mustParse(t, platformModFile, "platform/cue.mod/module.cue")
	assert.Equal(t, "testing.opmodel.dev/render/platform@v0", mf.Module)
	assert.Equal(t, "v0.17.0", mf.Language)
	assert.Equal(t, Dep{Version: "v0.11.0", Default: true}, mf.Deps["cue.dev/x/k8s.io@v0"])
	assert.Equal(t, Dep{Version: "v4.2.0"}, mf.Deps["opmodel.dev/catalogs/opm@v4"])
	assert.Equal(t, []string{"cue.dev/x/k8s.io@v0", "opmodel.dev/catalogs/opm@v4", "opmodel.dev/core@v2"}, mf.SortedPaths())
}

func TestParseModFile_RefusesNonCanonical(t *testing.T) {
	_, err := ParseModFile([]byte(`module: "x.example/m@v0"
language: version: "v0.17.0"
deps: "opmodel.dev/core": v: "v2.0.0"
`), "bad.cue")
	require.Error(t, err, "a dependency path without its major is not a tidied resolution")

	_, err = ParseModFile([]byte(`module: "x.example/m@v0"
language: version: "v0.17.0"
deps: "opmodel.dev/core@v2": {v: "v2.0.0", replaceWith: "../core"}
`), "bad.cue")
	require.ErrorContains(t, err, "replaceWith")
}

func TestReadModFile_OverlayAndDisk(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "opm-registry-module", "x")
	overlay := map[string]load.Source{
		filepath.Join(root, "cue.mod", "module.cue"): load.FromString(instanceModFile),
	}
	mf, err := ReadModFile(&module.Source{Root: root, Overlay: overlay})
	require.NoError(t, err)
	assert.Equal(t, "testing.opmodel.dev/modules/web_app@v1", mf.Module)

	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cue.mod"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cue.mod", "module.cue"), []byte(platformModFile), 0o644))
	mf, err = ReadModFile(&module.Source{Root: dir})
	require.NoError(t, err)
	assert.Equal(t, "testing.opmodel.dev/render/platform@v0", mf.Module)

	_, err = ReadModFile(&module.Source{Root: root, Overlay: map[string]load.Source{}})
	require.Error(t, err, "an overlay without cue.mod/module.cue is not a module")
	_, err = ReadModFile(nil)
	require.Error(t, err)
}

func TestSourceBytes_EveryConstructor(t *testing.T) {
	b, err := sourceBytes(load.FromString("a: 1\n"))
	require.NoError(t, err)
	assert.Equal(t, "a: 1\n", string(b))

	b, err = sourceBytes(load.FromBytes([]byte("b: 2\n")))
	require.NoError(t, err)
	assert.Equal(t, "b: 2\n", string(b))

	f, err := parser.ParseFile("x.cue", "c: 3\n")
	require.NoError(t, err)
	b, err = sourceBytes(load.FromFile(f))
	require.NoError(t, err)
	assert.Equal(t, "c: 3\n", string(b))

	_, err = sourceBytes(load.FromFile((*ast.File)(nil)))
	require.Error(t, err)
	_, err = sourceBytes(nil)
	require.Error(t, err)
}

func TestIsOPMPath(t *testing.T) {
	assert.True(t, IsOPMPath("opmodel.dev/core@v2"))
	assert.True(t, IsOPMPath("testing.opmodel.dev/modules/x@v1"))
	assert.False(t, IsOPMPath("cue.dev/x/k8s.io@v0"))
	assert.False(t, IsOPMPath("notopmodel.dev/x@v0"))
	assert.False(t, IsOPMPath("example.com/opmodel.dev@v0"))
}

func TestPromote_PlatformWinsSharedPath(t *testing.T) {
	p, err := Promote(mustParse(t, platformModFile, "p"), mustParse(t, instanceModFile, "i"), "/tmp/plat", "/tmp/inst")
	require.NoError(t, err)
	assert.Equal(t, "v4.2.0", p.Deps["opmodel.dev/catalogs/opm@v4"].Version, "the platform's entry wins the shared catalog path")
	assert.Equal(t, "v2.0.0-alpha.7", p.Deps["opmodel.dev/core@v2"].Version)
	assert.Equal(t, "v0.17.1", p.Language, "language.version is the inputs' maximum")
}

func TestPromote_InstanceOnlyPathSurvives(t *testing.T) {
	p, err := Promote(mustParse(t, platformModFile, "p"), mustParse(t, instanceModFile, "i"), "/tmp/plat", "/tmp/inst")
	require.NoError(t, err)
	assert.Equal(t, Dep{Version: "v1.0.0", Default: true}, p.Deps["example.com/helpers@v1"], "an instance-only path joins with its own marker")
	assert.Equal(t, []string{
		"cue.dev/x/k8s.io@v0",
		"example.com/helpers@v1",
		"opmodel.dev/catalogs/opm@v4",
		"opmodel.dev/core@v2",
	}, p.SortedPaths())
}

func TestPromote_DefaultMajorPreserved(t *testing.T) {
	p, err := Promote(mustParse(t, platformModFile, "p"), mustParse(t, instanceModFile, "i"), "/tmp/plat", "/tmp/inst")
	require.NoError(t, err)
	assert.True(t, p.Deps["cue.dev/x/k8s.io@v0"].Default, "the platform's default-major marker survives promotion")

	data, err := p.ModuleFile()
	require.NoError(t, err)
	written, err := modfile.Parse(data, "render/cue.mod/module.cue")
	require.NoError(t, err)
	assert.Equal(t, RenderModulePath, written.QualifiedModule())
	assert.Equal(t, map[string]string{"cue.dev/x/k8s.io": "v0", "example.com/helpers": "v1", "render.opmodel.dev/build": "v0"}, written.DefaultMajorVersions())
	assert.Equal(t, "v0.17.1", written.Language.Version)
}

func TestPromote_InstanceDefaultYieldsToPlatformDefault(t *testing.T) {
	plat := mustParse(t, `module: "p.example/p@v0"
language: version: "v0.17.0"
deps: "cue.dev/x/k8s.io@v0": {v: "v0.11.0", default: true}
`, "p")
	inst := mustParse(t, `module: "i.example/i@v0"
language: version: "v0.17.0"
deps: "cue.dev/x/k8s.io@v1": {v: "v1.0.0", default: true}
`, "i")
	p, err := Promote(plat, inst, "/tmp/plat", "/tmp/inst")
	require.NoError(t, err)
	assert.True(t, p.Deps["cue.dev/x/k8s.io@v0"].Default)
	assert.False(t, p.Deps["cue.dev/x/k8s.io@v1"].Default, "two defaults for one root path would be refused by cue/load; the platform's wins")
}

func TestPromote_RefusesSameModulePath(t *testing.T) {
	mf := mustParse(t, platformModFile, "p")
	_, err := Promote(mf, mf, "/a", "/b")
	require.ErrorContains(t, err, "same module path")
}

func TestLocalModuleFile_CarriesPromotedListAndReplacements(t *testing.T) {
	plat := mustParse(t, platformModFile, "p")
	inst := mustParse(t, instanceModFile, "i")
	p, err := Promote(plat, inst, "/tmp/plat", "/tmp/inst")
	require.NoError(t, err)

	moduleData, err := p.ModuleFile()
	require.NoError(t, err)
	base, err := modfile.Parse(moduleData, "cue.mod/module.cue")
	require.NoError(t, err)

	localData, err := p.LocalModuleFile()
	require.NoError(t, err)
	eff, err := modfile.ParseLocal(localData, "cue.mod/local-module.cue", base)
	require.NoError(t, err, "cue/load must accept the local-module.cue exactly as written:\n%s", localData)

	// The main-module view is the promoted list plus the two replace-only
	// placeholders.
	assert.Equal(t, "v4.2.0", eff.Deps["opmodel.dev/catalogs/opm@v4"].Version)
	assert.True(t, eff.Deps["cue.dev/x/k8s.io@v0"].Default)
	assert.Equal(t, "/tmp/plat", eff.Deps["testing.opmodel.dev/render/platform@v0"].ReplaceWith)
	assert.Equal(t, "/tmp/inst", eff.Deps["testing.opmodel.dev/modules/web_app@v1"].ReplaceWith)
	assert.Len(t, eff.Deps, len(p.Deps)+2)
}

// TestVerifyCoverage_DoctoredPromotionRefuses is the sole coverage of the
// D13 refusal invariant: Promote cannot produce an uncovered OPM path by
// construction (every input entry is copied into the list), so the only way
// to exercise the tripwire is to doctor the promotion after the fact. The
// kernel runs VerifyCoverage inside Stage, before skew is compared and
// before any policy is consulted, so the refusal is policy-independent.
func TestVerifyCoverage_DoctoredPromotionRefuses(t *testing.T) {
	plat := mustParse(t, platformModFile, "p")
	inst := mustParse(t, instanceModFile, "i")
	p, err := Promote(plat, inst, "/tmp/plat", "/tmp/inst")
	require.NoError(t, err)
	inputs := map[string]*ModFile{"platform": plat, "instance": inst}

	good, err := p.ModuleFile()
	require.NoError(t, err)
	require.NoError(t, VerifyCoverage(good, "cue.mod/module.cue", inputs))

	// Doctor the promotion: drop the catalog path both inputs require.
	delete(p.Deps, "opmodel.dev/catalogs/opm@v4")
	bad, err := p.ModuleFile()
	require.NoError(t, err)
	err = VerifyCoverage(bad, "cue.mod/module.cue", inputs)
	require.Error(t, err)
	var cov *CoverageError
	require.ErrorAs(t, err, &cov)
	assert.Equal(t, "opmodel.dev/catalogs/opm@v4", cov.Path)
	assert.Equal(t, []string{"instance", "platform"}, cov.RequiredBy)
	assert.Contains(t, err.Error(), "opmodel.dev/catalogs/opm@v4")

	// A dropped non-OPM path is not the invariant's business.
	delete(p.Deps, "example.com/helpers@v1")
	p.Deps["opmodel.dev/catalogs/opm@v4"] = Dep{Version: "v4.2.0"}
	ok, err := p.ModuleFile()
	require.NoError(t, err)
	require.NoError(t, VerifyCoverage(ok, "cue.mod/module.cue", inputs))
}

func TestCompareSkew_NewerOlderAndEqual(t *testing.T) {
	plat := mustParse(t, platformModFile, "p")

	newer := mustParse(t, instanceModFile, "i")
	rows, err := CompareSkew(plat, newer)
	require.NoError(t, err)
	require.Equal(t, []VersionRow{
		{Path: "opmodel.dev/catalogs/opm@v4", ModuleVersion: "v4.3.0", PlatformVersion: "v4.2.0", Newer: true},
		{Path: "opmodel.dev/core@v2", ModuleVersion: "v2.0.0-alpha.7", PlatformVersion: "v2.0.0-alpha.7"},
	}, rows, "non-OPM paths are not compared; the newer catalog pin is flagged")

	older := mustParse(t, `module: "testing.opmodel.dev/modules/old@v1"
language: version: "v0.17.0"
deps: {
	"opmodel.dev/catalogs/opm@v4": v: "v4.1.0"
	"opmodel.dev/core@v2": v: "v2.0.0-alpha.6"
	"testing.opmodel.dev/fixtures/only@v0": v: "v0.3.0"
}
`, "i")
	rows, err = CompareSkew(plat, older)
	require.NoError(t, err)
	require.Equal(t, []VersionRow{
		{Path: "opmodel.dev/catalogs/opm@v4", ModuleVersion: "v4.1.0", PlatformVersion: "v4.2.0"},
		{Path: "opmodel.dev/core@v2", ModuleVersion: "v2.0.0-alpha.6", PlatformVersion: "v2.0.0-alpha.7"},
		{Path: "testing.opmodel.dev/fixtures/only@v0", ModuleVersion: "v0.3.0"},
	}, rows, "older-than-platform and instance-only paths are rows with no Newer flag")
	for _, r := range rows {
		assert.False(t, r.Newer)
	}

	_, err = CompareSkew(nil, older)
	require.Error(t, err)
}
