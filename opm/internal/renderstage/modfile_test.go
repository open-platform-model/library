package renderstage

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

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
	overlay := map[string][]byte{
		filepath.Join(root, "cue.mod", "module.cue"): []byte(instanceModFile),
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

	_, err = ReadModFile(&module.Source{Root: root, Overlay: map[string][]byte{}})
	require.Error(t, err, "an overlay without cue.mod/module.cue is not a module")
	_, err = ReadModFile(nil)
	require.Error(t, err)
}

// overlayWithLocal stages instanceModFile plus the given local-module.cue
// (absent when empty) as an overlay-mode source under root.
func overlayWithLocal(root, local string) *module.Source {
	overlay := map[string][]byte{
		filepath.Join(root, "cue.mod", "module.cue"): []byte(instanceModFile),
	}
	if local != "" {
		overlay[filepath.Join(root, "cue.mod", "local-module.cue")] = []byte(local)
	}
	return &module.Source{Root: root, Overlay: overlay}
}

func TestReadLocalModFile_AbsentIsNil(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "opm-registry-module", "x")
	src := overlayWithLocal(root, "")
	base, err := ReadModFile(src)
	require.NoError(t, err)
	local, err := ReadLocalModFile(src, base)
	require.NoError(t, err)
	assert.Nil(t, local, "no local-module.cue is the normal case")

	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cue.mod"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cue.mod", "module.cue"), []byte(platformModFile), 0o644))
	disk := &module.Source{Root: dir}
	base, err = ReadModFile(disk)
	require.NoError(t, err)
	local, err = ReadLocalModFile(disk, base)
	require.NoError(t, err)
	assert.Nil(t, local)

	_, err = ReadLocalModFile(nil, base)
	require.Error(t, err)
	_, err = ReadLocalModFile(src, &ModFile{Module: base.Module})
	require.ErrorContains(t, err, "parsed module file", "a hand-built ModFile carries no parsed form to parse the local view against")
}

func TestReadLocalModFile_Targets(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "opm-registry-module", "x")
	src := overlayWithLocal(root, `deps: {
	"opmodel.dev/catalogs/opm@v4": replaceWith: "../catalog_opm"
	"example.com/helpers@v1": replaceWith: "/abs/helpers"
	"opmodel.dev/core@v2": replaceWith: "fork.example/core@v2"
	"fork.example/core@v2": v: "v2.1.0"
	"lib.example/never@v0": replaceWith: "./lib"
}
`)
	base, err := ReadModFile(src)
	require.NoError(t, err)
	local, err := ReadLocalModFile(src, base)
	require.NoError(t, err)
	require.NotNil(t, local)

	assert.Equal(t, map[string]string{
		"opmodel.dev/catalogs/opm@v4": filepath.Join(root, "..", "catalog_opm"),
		"example.com/helpers@v1":      "/abs/helpers",
		"opmodel.dev/core@v2":         "fork.example/core@v2",
		"lib.example/never@v0":        filepath.Join(root, "lib"),
	}, local.Replacements, "relative directories resolve against the module root, absolute ones and module paths pass verbatim")
	assert.Equal(t, filepath.Join(string(filepath.Separator), "opm-registry-module", "catalog_opm"), local.Replacements["opmodel.dev/catalogs/opm@v4"], "resolution is a clean join")

	// The listed entries: versions and markers inherited from module.cue
	// where the local file omits them, the replace-only path version-less,
	// the module-path target with its own version, and no replacement
	// target on any of them.
	assert.Equal(t, Dep{Version: "v4.3.0"}, local.Deps["opmodel.dev/catalogs/opm@v4"])
	assert.Equal(t, Dep{Version: "v1.0.0", Default: true}, local.Deps["example.com/helpers@v1"])
	assert.Equal(t, Dep{Version: "v2.0.0-alpha.7"}, local.Deps["opmodel.dev/core@v2"])
	assert.Equal(t, Dep{Version: "v2.1.0"}, local.Deps["fork.example/core@v2"])
	assert.Equal(t, Dep{}, local.Deps["lib.example/never@v0"])
	assert.Len(t, local.Deps, 5)
}

func TestReadLocalModFile_MalformedNamesTheInput(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "opm-registry-module", "x")

	// A version-less entry the module file does not list and nothing
	// replaces is refused by cue's own local-file rule.
	src := overlayWithLocal(root, "deps: \"lib.example/never@v0\": {}\n")
	base, err := ReadModFile(src)
	require.NoError(t, err)
	_, err = ReadLocalModFile(src, base)
	require.Error(t, err)
	assert.Contains(t, err.Error(), filepath.Join(root, "cue.mod", "local-module.cue"), "the error names the input's own file")
	assert.Contains(t, err.Error(), "lib.example/never@v0")

	// A module path disagreeing with module.cue.
	src = overlayWithLocal(root, "module: \"other.example/m@v0\"\n")
	base, err = ReadModFile(src)
	require.NoError(t, err)
	_, err = ReadLocalModFile(src, base)
	require.Error(t, err)
	assert.Contains(t, err.Error(), filepath.Join(root, "cue.mod", "local-module.cue"))

	// Not CUE at all.
	src = overlayWithLocal(root, "deps: {\n")
	base, err = ReadModFile(src)
	require.NoError(t, err)
	_, err = ReadLocalModFile(src, base)
	require.Error(t, err)
	assert.Contains(t, err.Error(), filepath.Join(root, "cue.mod", "local-module.cue"))
}

func TestParseModFile_AcceptsVersionlessDependency(t *testing.T) {
	mf := mustParse(t, `module: "x.example/m@v0"
language: version: "v0.17.0"
deps: "lib.example/never@v0": {}
`, "m/cue.mod/module.cue")
	assert.Equal(t, Dep{}, mf.Deps["lib.example/never@v0"], "a path a local replacement serves may carry no version; promotion decides whether that is acceptable")
}

func TestIsOPMPath(t *testing.T) {
	assert.True(t, IsOPMPath("opmodel.dev/core@v2"))
	assert.True(t, IsOPMPath("testing.opmodel.dev/modules/x@v1"))
	assert.False(t, IsOPMPath("cue.dev/x/k8s.io@v0"))
	assert.False(t, IsOPMPath("notopmodel.dev/x@v0"))
	assert.False(t, IsOPMPath("example.com/opmodel.dev@v0"))
}

func TestPromote_PlatformWinsSharedPath(t *testing.T) {
	p, err := Promote(mustParse(t, platformModFile, "p"), mustParse(t, instanceModFile, "i"), nil, nil, "/tmp/plat", "/tmp/inst")
	require.NoError(t, err)
	assert.Equal(t, "v4.2.0", p.Deps["opmodel.dev/catalogs/opm@v4"].Version, "the platform's entry wins the shared catalog path")
	assert.Equal(t, "v2.0.0-alpha.7", p.Deps["opmodel.dev/core@v2"].Version)
	assert.Equal(t, "v0.17.1", p.Language, "language.version is the inputs' maximum")
}

func TestPromote_InstanceOnlyPathSurvives(t *testing.T) {
	p, err := Promote(mustParse(t, platformModFile, "p"), mustParse(t, instanceModFile, "i"), nil, nil, "/tmp/plat", "/tmp/inst")
	require.NoError(t, err)
	assert.Equal(t, Dep{Version: "v1.0.0", Default: true}, p.Deps["example.com/helpers@v1"], "an instance-only path joins with its own marker")
	assert.Equal(t, []string{
		"cue.dev/x/k8s.io@v0",
		"example.com/helpers@v1",
		"opmodel.dev/catalogs/opm@v4",
		"opmodel.dev/core@v2",
		"testing.opmodel.dev/modules/web_app@v1",
		"testing.opmodel.dev/render/platform@v0",
	}, p.SortedPaths())
}

func TestPromote_DefaultMajorPreserved(t *testing.T) {
	p, err := Promote(mustParse(t, platformModFile, "p"), mustParse(t, instanceModFile, "i"), nil, nil, "/tmp/plat", "/tmp/inst")
	require.NoError(t, err)
	assert.True(t, p.Deps["cue.dev/x/k8s.io@v0"].Default, "the platform's default-major marker survives promotion")

	data, err := p.ModuleFile()
	require.NoError(t, err)
	written, err := modfile.Parse(data, "render/cue.mod/module.cue")
	require.NoError(t, err)
	assert.Equal(t, RenderModulePath, written.QualifiedModule())
	assert.Equal(t, map[string]string{
		"cue.dev/x/k8s.io":                    "v0",
		"example.com/helpers":                 "v1",
		"render.opmodel.dev/build":            "v0",
		"testing.opmodel.dev/modules/web_app": "v1",
		"testing.opmodel.dev/render/platform": "v0",
	}, written.DefaultMajorVersions(), "both inputs are marked default so their unqualified self-imports resolve")
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
	p, err := Promote(plat, inst, nil, nil, "/tmp/plat", "/tmp/inst")
	require.NoError(t, err)
	assert.True(t, p.Deps["cue.dev/x/k8s.io@v0"].Default)
	assert.False(t, p.Deps["cue.dev/x/k8s.io@v1"].Default, "two defaults for one root path would be refused by cue/load; the platform's wins")
}

func TestPromote_RefusesSameModulePath(t *testing.T) {
	mf := mustParse(t, platformModFile, "p")
	_, err := Promote(mf, mf, nil, nil, "/a", "/b")
	require.ErrorContains(t, err, "same module path")
}

func TestLocalModuleFile_CarriesPromotedListAndReplacements(t *testing.T) {
	plat := mustParse(t, platformModFile, "p")
	inst := mustParse(t, instanceModFile, "i")
	p, err := Promote(plat, inst, nil, nil, "/tmp/plat", "/tmp/inst")
	require.NoError(t, err)

	moduleData, err := p.ModuleFile()
	require.NoError(t, err)
	base, err := modfile.Parse(moduleData, "cue.mod/module.cue")
	require.NoError(t, err)

	localData, err := p.LocalModuleFile()
	require.NoError(t, err)
	eff, err := modfile.ParseLocal(localData, "cue.mod/local-module.cue", base)
	require.NoError(t, err, "cue/load must accept the local-module.cue exactly as written:\n%s", localData)

	// The main-module view is the promoted list, the two inputs' entries
	// served from their directories.
	assert.Equal(t, "v4.2.0", eff.Deps["opmodel.dev/catalogs/opm@v4"].Version)
	assert.True(t, eff.Deps["cue.dev/x/k8s.io@v0"].Default)
	assert.Equal(t, "/tmp/plat", eff.Deps["testing.opmodel.dev/render/platform@v0"].ReplaceWith)
	assert.Equal(t, "/tmp/inst", eff.Deps["testing.opmodel.dev/modules/web_app@v1"].ReplaceWith)
	assert.Len(t, eff.Deps, len(p.Deps))
}

func TestPromote_InputsListedAsDefaultMarkedReplacements(t *testing.T) {
	p, err := Promote(mustParse(t, platformModFile, "p"), mustParse(t, instanceModFile, "i"), nil, nil, "/tmp/plat", "/tmp/inst")
	require.NoError(t, err)
	assert.Equal(t, Dep{Version: "v0.0.0", Default: true}, p.Deps["testing.opmodel.dev/render/platform@v0"])
	assert.Equal(t, Dep{Version: "v1.0.0", Default: true}, p.Deps["testing.opmodel.dev/modules/web_app@v1"],
		"the instance module's own path is listed with a placeholder version and marked default")
	assert.Equal(t, map[string]string{
		"testing.opmodel.dev/modules/web_app@v1": "/tmp/inst",
		"testing.opmodel.dev/render/platform@v0": "/tmp/plat",
	}, p.Replacements, "the replacements name both input directories")

	// cue/load accepts the pair exactly as written and reads both defaults.
	moduleData, err := p.ModuleFile()
	require.NoError(t, err)
	base, err := modfile.Parse(moduleData, "cue.mod/module.cue")
	require.NoError(t, err)
	assert.Equal(t, "v1", base.DefaultMajorVersions()["testing.opmodel.dev/modules/web_app"])
	localData, err := p.LocalModuleFile()
	require.NoError(t, err)
	eff, err := modfile.ParseLocal(localData, "cue.mod/local-module.cue", base)
	require.NoError(t, err)
	assert.Equal(t, "/tmp/inst", eff.Deps["testing.opmodel.dev/modules/web_app@v1"].ReplaceWith)
	assert.True(t, eff.Deps["testing.opmodel.dev/modules/web_app@v1"].Default)
}

func TestPromote_InputDefaultYieldsToPromotedDefault(t *testing.T) {
	// The platform already marks another major of the instance module's own
	// root path default: two defaults for one root would be refused by
	// cue/load, so the input's entry is listed without the marker.
	plat := mustParse(t, `module: "p.example/p@v0"
language: version: "v0.17.0"
deps: "i.example/i@v1": {v: "v1.0.0", default: true}
`, "p")
	inst := mustParse(t, `module: "i.example/i@v0"
language: version: "v0.17.0"
`, "i")
	p, err := Promote(plat, inst, nil, nil, "/tmp/plat", "/tmp/inst")
	require.NoError(t, err)
	assert.Equal(t, Dep{Version: "v0.0.0"}, p.Deps["i.example/i@v0"])
	assert.True(t, p.Deps["i.example/i@v1"].Default)
	assert.Equal(t, Dep{Version: "v0.0.0", Default: true}, p.Deps["p.example/p@v0"])
}

func TestReplacedVersion(t *testing.T) {
	for path, want := range map[string]string{"a.example/m@v0": "v0.0.0", "a.example/m@v12": "v12.0.0"} {
		got, err := ReplacedVersion(path)
		require.NoError(t, err, path)
		assert.Equal(t, want, got)
	}
	for _, bad := range []string{"a.example/m", "a.example/m@1", "a.example/m@v0.1.0"} {
		_, err := ReplacedVersion(bad)
		assert.Error(t, err, bad)
	}
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
	p, err := Promote(plat, inst, nil, nil, "/tmp/plat", "/tmp/inst")
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

// ── Local replacements (render-local-replacements) ──────────────────

// localView parses a local-module.cue against a module file and returns the
// view, with a relative directory resolved against root.
func localView(t *testing.T, root, modSrc, localSrc string) (*ModFile, *LocalModFile) {
	t.Helper()
	src := &module.Source{Root: root, Overlay: map[string][]byte{
		filepath.Join(root, "cue.mod", "module.cue"):       []byte(modSrc),
		filepath.Join(root, "cue.mod", "local-module.cue"): []byte(localSrc),
	}}
	base, err := ReadModFile(src)
	require.NoError(t, err)
	local, err := ReadLocalModFile(src, base)
	require.NoError(t, err)
	require.NotNil(t, local)
	return base, local
}

// parseWrittenPair parses the promotion's module.cue and local-module.cue
// exactly as cue/load would, returning the effective main-module view.
func parseWrittenPair(t *testing.T, p *Promotion) *modfile.File {
	t.Helper()
	moduleData, err := p.ModuleFile()
	require.NoError(t, err)
	base, err := modfile.Parse(moduleData, "cue.mod/module.cue")
	require.NoError(t, err)
	localData, err := p.LocalModuleFile()
	require.NoError(t, err)
	eff, err := modfile.ParseLocal(localData, "cue.mod/local-module.cue", base)
	require.NoError(t, err, "cue/load must accept the pair exactly as written:\n%s\n%s", moduleData, localData)
	return eff
}

func TestPromote_PlatformReplacementIsHonoured(t *testing.T) {
	plat, platLocal := localView(t, "/plat", platformModFile, `deps: "opmodel.dev/catalogs/opm@v4": replaceWith: "../catalog_opm"
`)
	inst := mustParse(t, instanceModFile, "i")
	p, err := Promote(plat, inst, platLocal, nil, "/plat", "/inst")
	require.NoError(t, err)

	assert.Equal(t, "/catalog_opm", p.Replacements["opmodel.dev/catalogs/opm@v4"], "the platform's relative directory was resolved against its root")
	assert.Equal(t, "v4.2.0", p.Deps["opmodel.dev/catalogs/opm@v4"].Version, "the replaced path keeps its pinned version")
	assert.Equal(t, []ReplacementRow{{Path: "opmodel.dev/catalogs/opm@v4", Target: "/catalog_opm", By: "platform"}}, p.Rows)
	eff := parseWrittenPair(t, p)
	assert.Equal(t, "/catalog_opm", eff.Deps["opmodel.dev/catalogs/opm@v4"].ReplaceWith)
	assert.Equal(t, "v4.2.0", eff.Deps["opmodel.dev/catalogs/opm@v4"].Version)
}

func TestPromote_InstanceReplacementOnPlatformPathIsInert(t *testing.T) {
	plat := mustParse(t, platformModFile, "p")
	inst, instLocal := localView(t, "/inst", instanceModFile, `deps: "opmodel.dev/catalogs/opm@v4": replaceWith: "/mine/catalog"
`)
	p, err := Promote(plat, inst, nil, instLocal, "/plat", "/inst")
	require.NoError(t, err)

	assert.NotContains(t, p.Replacements, "opmodel.dev/catalogs/opm@v4", "the platform names the path: its pinned bytes execute")
	assert.Equal(t, "v4.2.0", p.Deps["opmodel.dev/catalogs/opm@v4"].Version)
	assert.Empty(t, p.Rows, "an inert replacement is not a row")
	eff := parseWrittenPair(t, p)
	assert.Empty(t, eff.Deps["opmodel.dev/catalogs/opm@v4"].ReplaceWith)
}

func TestPromote_InstanceReplacementOnInstanceOnlyPathIsHonoured(t *testing.T) {
	plat := mustParse(t, platformModFile, "p")
	inst, instLocal := localView(t, "/inst", instanceModFile, `deps: "example.com/helpers@v1": replaceWith: "./helpers"
`)
	p, err := Promote(plat, inst, nil, instLocal, "/plat", "/inst")
	require.NoError(t, err)

	assert.Equal(t, "/inst/helpers", p.Replacements["example.com/helpers@v1"])
	assert.Equal(t, Dep{Version: "v1.0.0", Default: true}, p.Deps["example.com/helpers@v1"], "version and marker are the instance's own")
	assert.Equal(t, []ReplacementRow{{Path: "example.com/helpers@v1", Target: "/inst/helpers", By: "instance"}}, p.Rows)
	eff := parseWrittenPair(t, p)
	assert.Equal(t, "/inst/helpers", eff.Deps["example.com/helpers@v1"].ReplaceWith)
	assert.True(t, eff.Deps["example.com/helpers@v1"].Default)
}

func TestPromote_ReplaceOnlyDependencyGetsPlaceholder(t *testing.T) {
	plat := mustParse(t, platformModFile, "p")
	inst, instLocal := localView(t, "/inst", `module: "testing.opmodel.dev/modules/web_app@v1"
language: version: "v0.17.0"
deps: {
	"opmodel.dev/core@v2": v: "v2.0.0-alpha.7"
	"lib.example/never@v0": {}
	"testing.opmodel.dev/fixtures/never@v3": {}
}
`, `deps: {
	"lib.example/never@v0": replaceWith: "/lib"
	"testing.opmodel.dev/fixtures/never@v3": replaceWith: "./fixtures"
}
`)
	p, err := Promote(plat, inst, nil, instLocal, "/plat", "/inst")
	require.NoError(t, err)

	assert.Equal(t, Dep{Version: "v0.0.0"}, p.Deps["lib.example/never@v0"], "a version-less replaced path is listed with the placeholder of its major")
	assert.Equal(t, Dep{Version: "v3.0.0"}, p.Deps["testing.opmodel.dev/fixtures/never@v3"])
	assert.Equal(t, []ReplacementRow{
		{Path: "lib.example/never@v0", Target: "/lib", By: "instance"},
		{Path: "testing.opmodel.dev/fixtures/never@v3", Target: "/inst/fixtures", By: "instance"},
	}, p.Rows)

	// cue/load accepts the pair, and the OPM-namespace path is covered.
	eff := parseWrittenPair(t, p)
	assert.Equal(t, "/lib", eff.Deps["lib.example/never@v0"].ReplaceWith)
	assert.Equal(t, "v0.0.0", eff.Deps["lib.example/never@v0"].Version)
	written, err := p.ModuleFile()
	require.NoError(t, err)
	require.NoError(t, VerifyCoverage(written, "cue.mod/module.cue", map[string]*ModFile{"platform": plat, "instance": inst}))
}

func TestPromote_ModulePathReplacementCarriesItsTarget(t *testing.T) {
	plat := mustParse(t, platformModFile, "p")
	inst, instLocal := localView(t, "/inst", instanceModFile, `deps: {
	"example.com/helpers@v1": replaceWith: "fork.example/helpers@v1"
	"fork.example/helpers@v1": v: "v1.2.0"
}
`)
	p, err := Promote(plat, inst, nil, instLocal, "/plat", "/inst")
	require.NoError(t, err)

	assert.Equal(t, "fork.example/helpers@v1", p.Replacements["example.com/helpers@v1"], "a module path passes verbatim")
	assert.Equal(t, Dep{Version: "v1.2.0"}, p.Deps["fork.example/helpers@v1"], "the target entry the local file lists joins the promoted list with its version")
	assert.Equal(t, []ReplacementRow{{Path: "example.com/helpers@v1", Target: "fork.example/helpers@v1", By: "instance"}}, p.Rows)
	eff := parseWrittenPair(t, p)
	assert.Equal(t, "fork.example/helpers@v1", eff.Deps["example.com/helpers@v1"].ReplaceWith)
	assert.Equal(t, "v1.2.0", eff.Deps["fork.example/helpers@v1"].Version)
}

func TestPromote_RowsAreSortedAcrossInputs(t *testing.T) {
	plat, platLocal := localView(t, "/plat", platformModFile, `deps: "opmodel.dev/catalogs/opm@v4": replaceWith: "/cat"
`)
	inst, instLocal := localView(t, "/inst", instanceModFile, `deps: {
	"example.com/helpers@v1": replaceWith: "/helpers"
	"opmodel.dev/catalogs/opm@v4": replaceWith: "/mine"
}
`)
	p, err := Promote(plat, inst, platLocal, instLocal, "/plat", "/inst")
	require.NoError(t, err)
	assert.Equal(t, []ReplacementRow{
		{Path: "example.com/helpers@v1", Target: "/helpers", By: "instance"},
		{Path: "opmodel.dev/catalogs/opm@v4", Target: "/cat", By: "platform"},
	}, p.Rows, "path order; the instance's catalog replacement is inert")
	assert.Equal(t, "/cat", p.Replacements["opmodel.dev/catalogs/opm@v4"])
	assert.Len(t, p.Replacements, 4, "two inputs, two honoured replacements")
}

func TestPromote_RefusesReplacingAnInput(t *testing.T) {
	plat := mustParse(t, platformModFile, "p")
	inst, instLocal := localView(t, "/inst", instanceModFile, `deps: "testing.opmodel.dev/render/platform@v0": replaceWith: "/elsewhere"
`)
	_, err := Promote(plat, inst, nil, instLocal, "/plat", "/inst")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "testing.opmodel.dev/render/platform@v0")
	assert.Contains(t, err.Error(), "render input")
}

func TestPromote_RefusesVersionlessDependencyWithoutReplacement(t *testing.T) {
	plat := mustParse(t, platformModFile, "p")
	inst := mustParse(t, `module: "testing.opmodel.dev/modules/web_app@v1"
language: version: "v0.17.0"
deps: {
	"opmodel.dev/core@v2": v: "v2.0.0-alpha.7"
	"lib.example/never@v0": {}
}
`, "i")

	// No local view at all.
	_, err := Promote(plat, inst, nil, nil, "/plat", "/inst")
	require.Error(t, err)
	assert.Equal(t, `instance dependency "lib.example/never@v0" carries no version and no promoted local replacement covers it`, err.Error(),
		"the refusal names the path and the input, and is raised before modfile formatting")
	assert.NotContains(t, err.Error(), "formatting")

	// A local view that replaces some other path does not cover it.
	_, instLocal := localView(t, "/inst", `module: "testing.opmodel.dev/modules/web_app@v1"
language: version: "v0.17.0"
deps: {
	"opmodel.dev/core@v2": v: "v2.0.0-alpha.7"
	"lib.example/never@v0": {}
	"other.example/x@v0": v: "v0.1.0"
}
`, `deps: "other.example/x@v0": replaceWith: "/x"
`)
	_, err = Promote(plat, inst, nil, instLocal, "/plat", "/inst")
	require.ErrorContains(t, err, `instance dependency "lib.example/never@v0"`)

	// The platform's own version-less entry is named as the platform's.
	platBare := mustParse(t, `module: "testing.opmodel.dev/render/platform@v0"
language: version: "v0.17.0"
deps: "opmodel.dev/catalogs/opm@v4": {}
`, "p")
	_, err = Promote(platBare, mustParse(t, instanceModFile, "i"), nil, nil, "/plat", "/inst")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `platform dependency "opmodel.dev/catalogs/opm@v4"`)

	// Version-less on the instance, versioned on the platform: the
	// platform's entry wins the shared path and nothing is refused.
	instShared := mustParse(t, `module: "testing.opmodel.dev/modules/web_app@v1"
language: version: "v0.17.0"
deps: "opmodel.dev/catalogs/opm@v4": {}
`, "i")
	p, err := Promote(plat, instShared, nil, nil, "/plat", "/inst")
	require.NoError(t, err)
	assert.Equal(t, "v4.2.0", p.Deps["opmodel.dev/catalogs/opm@v4"].Version)
}

func TestPromote_NilViewsMatchTheOldShape(t *testing.T) {
	plat := mustParse(t, platformModFile, "p")
	inst := mustParse(t, instanceModFile, "i")
	p, err := Promote(plat, inst, nil, nil, "/tmp/plat", "/tmp/inst")
	require.NoError(t, err)
	assert.Nil(t, p.Rows)
	assert.Equal(t, map[string]string{
		"testing.opmodel.dev/modules/web_app@v1": "/tmp/inst",
		"testing.opmodel.dev/render/platform@v0": "/tmp/plat",
	}, p.Replacements)
}

func TestCompareSkew_VersionlessReplacedPathIsNotCompared(t *testing.T) {
	// A platform developing a never-published catalog lists it version-less
	// and replaces it; the instance pins a published build. Neither side is
	// newer: there is no version to compare, and the row carries what each
	// side wrote.
	plat := mustParse(t, `module: "testing.opmodel.dev/render/platform@v0"
language: version: "v0.17.0"
deps: "opmodel.dev/catalogs/opm@v4": {}
`, "p")
	inst := mustParse(t, instanceModFile, "i")
	rows, err := CompareSkew(plat, inst)
	require.NoError(t, err)
	assert.Equal(t, []VersionRow{
		{Path: "opmodel.dev/catalogs/opm@v4", ModuleVersion: "v4.3.0", PlatformVersion: ""},
		{Path: "opmodel.dev/core@v2", ModuleVersion: "v2.0.0-alpha.7", PlatformVersion: ""},
	}, rows)
}
