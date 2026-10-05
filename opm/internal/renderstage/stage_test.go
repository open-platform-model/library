package renderstage

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/parser"
	"cuelang.org/go/mod/modfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/internal/loader"
	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/internal/schematest"
	"github.com/open-platform-model/library/opm/internal/sourcetree"
	"github.com/open-platform-model/library/opm/module"
)

func TestImportPath(t *testing.T) {
	cases := []struct {
		mod, pkgDir, pkgName, want string
	}{
		{"testing.opmodel.dev/x/web_app@v1", "", "web_app", "testing.opmodel.dev/x/web_app@v1"},
		{"testing.opmodel.dev/x/web_app@v1", "opm-synth-instance", "instance", "testing.opmodel.dev/x/web_app/opm-synth-instance@v1:instance"},
		{"testing.opmodel.dev/x/scenarios@v0", "missing", "missing", "testing.opmodel.dev/x/scenarios/missing@v0"},
		{"testing.opmodel.dev/x/platform@v0", ".", "platform", "testing.opmodel.dev/x/platform@v0"},
		{"testing.opmodel.dev/x/platform@v0", "", "opm_platform", "testing.opmodel.dev/x/platform@v0:opm_platform"},
		// The two inputs Stage generates import paths for: a synthesized
		// instance package under its module and an on-disk root platform.
		{"testing.opmodel.dev/modules/web_app@v1", "opm-synth-instance", "instance", "testing.opmodel.dev/modules/web_app/opm-synth-instance@v1:instance"},
		{"testing.opmodel.dev/render/platform@v0", "", "platform", "testing.opmodel.dev/render/platform@v0"},
	}
	for _, c := range cases {
		got, err := ImportPath(c.mod, c.pkgDir, c.pkgName)
		require.NoError(t, err)
		assert.Equal(t, c.want, got)
	}
	_, err := ImportPath("no-major.example/x", "", "x")
	require.Error(t, err)
	_, err = ImportPath("x.example/x@v0", "", "")
	require.Error(t, err)
}

func TestRenderGlue_QuotesCallerStrings(t *testing.T) {
	glue, err := RenderGlue(GlueInputs{
		InstancePath: "testing.opmodel.dev/x/web_app/opm-synth-instance@v1:instance",
		PlatformPath: "testing.opmodel.dev/x/platform@v0",
		RuntimeName:  `opm-"cli"\n`,
	})
	require.NoError(t, err)
	src := string(glue)
	assert.Contains(t, src, `instance "testing.opmodel.dev/x/web_app/opm-synth-instance@v1:instance"`)
	assert.Contains(t, src, `platform "testing.opmodel.dev/x/platform@v0"`)
	assert.Contains(t, src, `_runtimeName: "opm-\"cli\"\\n"`, "the runtime name is a CUE string literal, never raw interpolation")
	assert.NotContains(t, src, "<<", "every template slot was filled")
	assert.Contains(t, src, "_skipUnprovided: false", "the skip switch defaults off, as a bool literal")

	// The generated file must parse as CUE.
	_, err = parser.ParseFile(RenderFileName, glue)
	require.NoError(t, err)

	on, err := RenderGlue(GlueInputs{InstancePath: "a@v0", PlatformPath: "b@v0", RuntimeName: "rt", SkipUnprovided: true})
	require.NoError(t, err)
	assert.Contains(t, string(on), "_skipUnprovided: true", "the switch renders as the literal true")
	assert.NotContains(t, string(on), "_skipUnprovided: false")
	_, err = parser.ParseFile(RenderFileName, on)
	require.NoError(t, err)

	_, err = RenderGlue(GlueInputs{InstancePath: "a@v0", PlatformPath: "b@v0"})
	require.Error(t, err, "runtime name is required")
	_, err = RenderGlue(GlueInputs{InstancePath: "a@v0", RuntimeName: "rt"})
	require.Error(t, err)
}

// overlayInstance stages a minimal overlay-mode instance module: the module's
// own root files plus an instance package in a subdirectory, the shape
// synth.Instance produces.
func overlayInstance(root string) *module.Source {
	pkg := "opm-synth-instance"
	return &module.Source{
		Root: root,
		Pkg:  pkg,
		Overlay: map[string][]byte{
			filepath.Join(root, "cue.mod", "module.cue"):   []byte(instanceModFile),
			filepath.Join(root, "module.cue"):              []byte("package web_app\n\nx: 1\n"),
			filepath.Join(root, pkg, "instance.cue"):       []byte([]byte("package instance\n\ny: 2\n")),
			filepath.Join(root, pkg, "values.cue"):         []byte("package instance\n\nz: 3\n"),
			filepath.Join(root, pkg, "notes.md"):           []byte("not cue"),
			filepath.Join(root, pkg, "nested", "deep.cue"): []byte("package other\n"),
		},
	}
}

func diskPlatform(t *testing.T) *module.Source {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cue.mod"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cue.mod", "module.cue"), []byte(platformModFile), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "platform.cue"), []byte("package platform\n\np: 1\n"), 0o644))
	return &module.Source{Root: dir}
}

// generatedFiles lists the overlay entries of staged that are the generated
// render module (everything not under instance/ or platform/),
// slash-separated, relative to RenderRoot and sorted. It also asserts that
// RenderRoot does not exist on disk: staging writes nothing.
func generatedFiles(t *testing.T, staged *Staged) []string {
	t.Helper()
	assertAbsent(t, RenderRoot)
	var files []string
	for key := range staged.Overlay {
		rel, err := filepath.Rel(RenderRoot, key)
		require.NoError(t, err)
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, "instance/") || strings.HasPrefix(rel, "platform/") {
			continue
		}
		files = append(files, rel)
	}
	sort.Strings(files)
	return files
}

// renderModule is what generatedFiles returns for every staged render.
var renderModule = []string{"cue.mod/local-module.cue", "cue.mod/module.cue", RenderFileName}

// served returns the overlay entry at the slash path rel under RenderRoot.
func served(t *testing.T, staged *Staged, rel string) []byte {
	t.Helper()
	data, ok := staged.Overlay[filepath.Join(RenderRoot, filepath.FromSlash(rel))]
	require.True(t, ok, "%s is served from the overlay", rel)
	return data
}

func TestStage_ServesOverlayFromMemoryAndStagesRenderModule(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "opm-registry-module", "web_app")
	inst := overlayInstance(root)
	plat := diskPlatform(t)

	staged, err := Stage(inst, plat, "rt", StageOptions{})
	require.NoError(t, err)
	assert.Equal(t, RenderRoot, staged.Dir)

	// The overlay tree is re-keyed under <RenderRoot>/instance (the
	// directory the local-module.cue replacement names) on Staged.Overlay,
	// every entry included, beside the generated module, and nothing of it
	// is written.
	instDir := filepath.Join(RenderRoot, "instance")
	require.Len(t, staged.Overlay, len(inst.Overlay)+len(renderModule))
	for _, rel := range []string{"cue.mod/module.cue", "module.cue", "opm-synth-instance/instance.cue", "opm-synth-instance/values.cue", "opm-synth-instance/notes.md", "opm-synth-instance/nested/deep.cue"} {
		assert.Contains(t, staged.Overlay, filepath.Join(instDir, filepath.FromSlash(rel)), rel)
	}
	assert.Equal(t, "package instance\n\ny: 2\n", string(staged.Overlay[filepath.Join(instDir, "opm-synth-instance", "instance.cue")]))
	assert.Equal(t, renderModule, generatedFiles(t, staged), "the overlay carries the generated render module beside the instance")

	// Generated files. The on-disk platform is referenced in place through
	// the local-module.cue replacement.
	moduleCue := served(t, staged, "cue.mod/module.cue")
	assert.Contains(t, string(moduleCue), `module: "`+RenderModulePath+`"`)
	assert.Contains(t, string(moduleCue), `"opmodel.dev/catalogs/opm@v4"`)
	localCue := served(t, staged, "cue.mod/local-module.cue")
	assert.Contains(t, string(localCue), "replaceWith")
	assert.Contains(t, string(localCue), plat.Root, "the on-disk platform is served from its own directory")
	glue := served(t, staged, RenderFileName)
	assert.Contains(t, string(glue), `instance "testing.opmodel.dev/modules/web_app/opm-synth-instance@v1:instance"`)
	assert.Contains(t, string(glue), `platform "testing.opmodel.dev/render/platform@v0"`)

	// Skew rows ride along (the instance pins a newer catalog).
	require.Len(t, staged.Skew, 2)
	assert.True(t, staged.Skew[0].Newer)
}

// rekeyed returns the .cue files under dir as an overlay-mode source keyed
// under root, a path that exists nowhere on disk.
func rekeyed(t *testing.T, dir, root string) *module.Source {
	t.Helper()
	files, err := sourcetree.OverlayFromDir(dir)
	require.NoError(t, err)
	overlay := make(map[string][]byte, len(files))
	for p, data := range files {
		rel, err := filepath.Rel(dir, p)
		require.NoError(t, err)
		overlay[filepath.Join(root, rel)] = data
	}
	return &module.Source{Root: root, Overlay: overlay}
}

func TestStage_OverlayInputsLeaveOnlyTheRenderModule(t *testing.T) {
	inst := overlayInstance(filepath.Join(string(filepath.Separator), "opm-registry-module", "web_app"))
	plat := rekeyed(t, diskPlatform(t).Root, filepath.Join(string(filepath.Separator), "opm-registry-module", "platform"))

	staged, err := Stage(inst, plat, "rt", StageOptions{})
	require.NoError(t, err)

	assert.Equal(t, renderModule, generatedFiles(t, staged))
	assert.Len(t, staged.Overlay, len(inst.Overlay)+len(plat.Overlay)+len(renderModule))
	assert.Contains(t, staged.Overlay, filepath.Join(RenderRoot, "platform", "platform.cue"))
	assert.Contains(t, staged.Overlay, filepath.Join(RenderRoot, "platform", "cue.mod", "module.cue"))
	localCue := served(t, staged, "cue.mod/local-module.cue")
	assert.Contains(t, string(localCue), filepath.Join(RenderRoot, "platform"), "the replacement names the in-memory directory")
}

func TestStage_OnDiskInputsCarryOnlyTheRenderModule(t *testing.T) {
	fixture := filepath.Join(schematest.LibraryRoot(t), "testdata", "render")
	inst := &module.Source{Root: filepath.Join(fixture, "instance")}
	plat := &module.Source{Root: filepath.Join(fixture, "platform")}

	staged, err := Stage(inst, plat, "rt", StageOptions{})
	require.NoError(t, err)
	assert.Len(t, staged.Overlay, len(renderModule), "on-disk inputs are referenced in place; only the generated module is served")
	assert.Equal(t, renderModule, generatedFiles(t, staged))
}

// TestStageBuild_OverlayInstanceServedFromMemory is the change's spike
// (overlay-served-in-memory): cue/load serves a local-module.cue directory
// replacement that exists only in load.Config.Overlay. The on-disk render
// fixture instance is re-keyed under a synthetic root so no file of it can be
// read from disk, staged against the on-disk fixture platform, and built
// with the catalog and module served by the in-process registry.
func TestStageBuild_OverlayInstanceServedFromMemory(t *testing.T) {
	fixture := filepath.Join(schematest.LibraryRoot(t), "testdata", "render")
	registrytest.NewRegistryFromDir(t, filepath.Join(fixture, "registry"), "testing.opmodel.dev/library-render")

	inst := rekeyed(t, filepath.Join(fixture, "instance"), sourcetree.SyntheticRoot("testing.opmodel.dev/library-render/instance", "v0.0.0"))
	plat := &module.Source{Root: filepath.Join(fixture, "platform")}

	staged, err := Stage(inst, plat, "rt", StageOptions{})
	require.NoError(t, err)
	require.Len(t, staged.Overlay, len(inst.Overlay)+len(renderModule))
	assertAbsent(t, RenderRoot)

	// Build with the process environment: the registry helper set
	// CUE_REGISTRY and CUE_CACHE_DIR for this test.
	built, err := Build(cuecontext.New(), staged, loader.Options{})
	require.NoError(t, err, "cue/load serves the replacement directory from the overlay")
	require.NoError(t, built.Err())

	components := built.LookupPath(cue.MakePath(cue.Hid("_components", RenderModulePath+":render")))
	require.NoError(t, components.Err())
	require.True(t, components.Exists(), "_components resolves through the in-memory instance import")
	fields, err := components.Fields()
	require.NoError(t, err)
	var names []string
	for fields.Next() {
		names = append(names, fields.Selector().String())
	}
	assert.ElementsMatch(t, []string{"web", "config"}, names)
}

func TestStage_RefusesBadInputs(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "opm-registry-module", "web_app")
	plat := diskPlatform(t)

	_, err := Stage(nil, plat, "rt", StageOptions{})
	require.ErrorContains(t, err, "instance carries no source")
	_, err = Stage(overlayInstance(root), nil, "rt", StageOptions{})
	require.ErrorContains(t, err, "platform carries no source")
	_, err = Stage(overlayInstance(root), plat, "", StageOptions{})
	require.ErrorContains(t, err, "runtime name")

	// An overlay entry outside its root is refused rather than written
	// somewhere else.
	escaped := overlayInstance(root)
	escaped.Overlay[filepath.Join(string(filepath.Separator), "elsewhere", "x.cue")] = []byte("package x\n")
	_, err = Stage(escaped, plat, "rt", StageOptions{})
	require.ErrorContains(t, err, "outside the source root")

	// Two package clauses in one package directory.
	mixed := overlayInstance(root)
	mixed.Overlay[filepath.Join(root, "opm-synth-instance", "other.cue")] = []byte("package other\n")
	_, err = Stage(mixed, plat, "rt", StageOptions{})
	require.ErrorContains(t, err, "more than one package")

	// No package clause at all.
	bare := overlayInstance(root)
	for k := range bare.Overlay {
		if strings.Contains(k, "opm-synth-instance") {
			delete(bare.Overlay, k)
		}
	}
	bare.Overlay[filepath.Join(root, "opm-synth-instance", "data.cue")] = []byte("a: 1\n")
	_, err = Stage(bare, plat, "rt", StageOptions{})
	require.ErrorContains(t, err, "no package clause")
}

// ── Local replacements (render-local-replacements) ──────────────────

// libModulePath is a module no registry serves: the render can only resolve
// it through a directory replacement.
const libModulePath = "test.example/lib@v0"

// writeLibModule writes a never-published module under a temp directory: one
// package exporting the image the instance copy reads.
func writeLibModule(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"cue.mod/module.cue": "module: \"" + libModulePath + "\"\nlanguage: version: \"v0.17.0\"\n",
		"lib.cue":            "package lib\n\nImage: \"nginx:from-lib\"\n",
	})
	return dir
}

// catalogWithLabel copies the fixture catalog (0.1.0) into a temp directory
// and stamps one extra label on its deployment transformer's output, so a
// render that evaluated the copy's bytes is distinguishable from one that
// evaluated the published build.
func catalogWithLabel(t *testing.T, fixture string) string {
	t.Helper()
	dir := t.TempDir()
	copyTree(t, filepath.Join(fixture, "registry", "testing.opmodel.dev_library-render_cat_v0.1.0"), dir)
	catalog := filepath.Join(dir, "catalog.cue")
	src, err := os.ReadFile(catalog)
	require.NoError(t, err)
	// The deployment transformer is the one whose labels line is aligned
	// with four spaces; the service and configmap transformers use one.
	const anchor = "labels:    #context.labels"
	require.Contains(t, string(src), anchor, "the fixture catalog's deployment transformer")
	patched := strings.Replace(string(src), anchor,
		"labels: {\n\t\t\t\t\t\tfor k, v in #context.labels {(k): v}\n\t\t\t\t\t\t\"render.test/catalog\": \"local\"\n\t\t\t\t\t}", 1)
	require.NoError(t, os.WriteFile(catalog, []byte(patched), 0o644))
	return dir
}

// instanceImportingLib copies the fixture instance into a temp directory,
// makes it import the never-published lib module for its image, lists that
// module version-less in cue.mod/module.cue and replaces it with libDir in
// cue.mod/local-module.cue: the shape a developer's checkout has (the local
// file is the whole main-module view when the instance is built on its own,
// so it lists every dependency, versions inherited where omitted).
func instanceImportingLib(t *testing.T, fixture, libDir string) string {
	t.Helper()
	dir := t.TempDir()
	copyTree(t, filepath.Join(fixture, "instance"), dir)
	writeFiles(t, dir, map[string]string{
		"cue.mod/module.cue": `module: "testing.opmodel.dev/library-render/instance@v0"
language: version: "v0.17.0"
deps: {
	"opmodel.dev/core@v2": v: "` + registrytest.DefaultCoreVersion + `"
	"` + libModulePath + `": {}
	"testing.opmodel.dev/library-render/cat@v0": v: "v0.1.0"
	"testing.opmodel.dev/library-render/web_app@v0": v: "v0.1.0"
}
`,
		"cue.mod/local-module.cue": `deps: {
	"opmodel.dev/core@v2": {}
	"` + libModulePath + `": replaceWith: "` + libDir + `"
	"testing.opmodel.dev/library-render/cat@v0": {}
	"testing.opmodel.dev/library-render/web_app@v0": {}
}
`,
		"instance.cue": `package instance

import (
	c "opmodel.dev/core@v2"
	lib "test.example/lib@v0"
	webapp "testing.opmodel.dev/library-render/web_app@v0"
)

c.#ModuleInstance

metadata: {
	name:      "web-local"
	namespace: "default"
}

#module: webapp

values: {
	image:    lib.Image
	replicas: 2
}
`,
	})
	return dir
}

// writeFiles writes slash-relative paths under dir.
func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	}
}

// copyTree copies every regular file under src to the same relative path
// under dst.
func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	require.NoError(t, filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		out := filepath.Join(dst, rel)
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		return os.WriteFile(out, data, 0o644)
	}))
}

// renderedOutput returns rendered."<component> :: <transformer>".output of a
// built render module.
func renderedOutput(t *testing.T, built cue.Value, component, transformer string) cue.Value {
	t.Helper()
	out := built.LookupPath(cue.MakePath(cue.Str("rendered"), cue.Str(component+" :: "+transformer), cue.Str("output")))
	require.NoError(t, out.Err())
	require.True(t, out.Exists(), "rendered output for %s :: %s", component, transformer)
	return out
}

// withLocalFile adds a cue.mod/local-module.cue to a source in either mode.
func withLocalFile(t *testing.T, src *module.Source, content string) *module.Source {
	t.Helper()
	path := filepath.Join(src.Root, "cue.mod", "local-module.cue")
	if src.Overlay != nil {
		src.Overlay[path] = []byte(content)
		return src
	}
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return src
}

// readPair returns the staged cue.mod pair's bytes, as the build is served
// them.
func readPair(t *testing.T, staged *Staged) (moduleCue, localCue []byte) {
	t.Helper()
	return served(t, staged, "cue.mod/module.cue"), served(t, staged, "cue.mod/local-module.cue")
}

func TestStage_RefusesLocalReplacementsUnlessEnabled(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "opm-registry-module", "web_app")
	catDir := t.TempDir()

	// The platform's file carries a replacement: refused, nothing written.
	plat := withLocalFile(t, diskPlatform(t), `deps: "opmodel.dev/catalogs/opm@v4": replaceWith: "`+catDir+`"
`)
	_, err := Stage(overlayInstance(root), plat, "rt", StageOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `platform "testing.opmodel.dev/render/platform@v0"`)
	assert.Contains(t, err.Error(), "cue.mod/local-module.cue")
	assertAbsent(t, RenderRoot)

	// The instance's file (overlay mode) carries one: same refusal.
	inst := withLocalFile(t, overlayInstance(root), `deps: "example.com/helpers@v1": replaceWith: "./helpers"
`)
	_, err = Stage(inst, diskPlatform(t), "rt", StageOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `instance "testing.opmodel.dev/modules/web_app@v1"`)
	assert.Contains(t, err.Error(), "cue.mod/local-module.cue")
	assertAbsent(t, RenderRoot)

	// A file with no replacement is not a redirection: it stages.
	noRepl := withLocalFile(t, diskPlatform(t), "deps: \"opmodel.dev/core@v2\": v: \"v2.0.0-beta.1\"\n")
	staged, err := Stage(overlayInstance(root), noRepl, "rt", StageOptions{})
	require.NoError(t, err)
	assert.Nil(t, staged.Replacements)
}

func TestStage_AbsentLocalFileStagesIdenticallyEitherWay(t *testing.T) {
	fixture := filepath.Join(schematest.LibraryRoot(t), "testdata", "render")
	inst := &module.Source{Root: filepath.Join(fixture, "instance")}
	plat := &module.Source{Root: filepath.Join(fixture, "platform")}

	stagedOff, err := Stage(inst, plat, "rt", StageOptions{})
	require.NoError(t, err)
	stagedOn, err := Stage(inst, plat, "rt", StageOptions{LocalReplacements: true})
	require.NoError(t, err)

	moduleOff, localOff := readPair(t, stagedOff)
	moduleOn, localOn := readPair(t, stagedOn)
	assert.Equal(t, string(moduleOff), string(moduleOn), "module.cue is byte-identical")
	assert.Equal(t, string(localOff), string(localOn), "local-module.cue is byte-identical")
	assert.Nil(t, stagedOff.Replacements)
	assert.Nil(t, stagedOn.Replacements)
	assert.Equal(t, stagedOff.Skew, stagedOn.Skew)
}

func TestStage_HonoursLocalReplacements(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "opm-registry-module", "web_app")
	catDir := t.TempDir()
	plat := withLocalFile(t, diskPlatform(t), `deps: "opmodel.dev/catalogs/opm@v4": replaceWith: "`+catDir+`"
`)
	inst := withLocalFile(t, overlayInstance(root), `deps: {
	"example.com/helpers@v1": replaceWith: "./helpers"
	"opmodel.dev/catalogs/opm@v4": replaceWith: "/mine"
}
`)
	staged, err := Stage(inst, plat, "rt", StageOptions{LocalReplacements: true})
	require.NoError(t, err)

	helpersDir := filepath.Join(root, "helpers")
	assert.Equal(t, []ReplacementRow{
		{Path: "example.com/helpers@v1", Target: helpersDir, By: "instance"},
		{Path: "opmodel.dev/catalogs/opm@v4", Target: catDir, By: "platform"},
	}, staged.Replacements, "the platform's replacement and the instance-only one; the instance's catalog replacement is inert")
	assert.Equal(t, renderModule, generatedFiles(t, staged))

	// The served pair carries exactly those targets, as cue/load reads it.
	moduleCue, localCue := readPair(t, staged)
	assert.Contains(t, string(localCue), `replaceWith: "`+catDir+`"`)
	assert.Contains(t, string(localCue), `replaceWith: "`+helpersDir+`"`)
	assert.NotContains(t, string(localCue), "/mine")
	base, err := modfile.Parse(moduleCue, "cue.mod/module.cue")
	require.NoError(t, err)
	eff, err := modfile.ParseLocal(localCue, "cue.mod/local-module.cue", base)
	require.NoError(t, err)
	assert.Equal(t, catDir, eff.Deps["opmodel.dev/catalogs/opm@v4"].ReplaceWith)
	assert.Equal(t, "v4.2.0", eff.Deps["opmodel.dev/catalogs/opm@v4"].Version, "the replaced path keeps the platform's pin")
	assert.Equal(t, helpersDir, eff.Deps["example.com/helpers@v1"].ReplaceWith)
	assert.Equal(t, plat.Root, eff.Deps["testing.opmodel.dev/render/platform@v0"].ReplaceWith)
	assert.Equal(t, filepath.Join(RenderRoot, "instance"), eff.Deps["testing.opmodel.dev/modules/web_app@v1"].ReplaceWith)
}

// TestStageBuild_LocalReplacementsResolveInOneBuild is the change's spike
// (render-local-replacements 1.1): a hand-written render module whose
// local-module.cue replaces (a) an instance-only path with a directory
// holding a never-published module and (b) the platform's catalog path with
// a copy of the fixture catalog carrying one changed label. It pins that
// cue/load resolves both inside the one render build: the instance's import
// is served from (a) although the instance itself enters through a
// directory replacement, and the deployment's bytes come from (b) although
// the platform (and the registry-served module) pin the published 0.1.0.
// No promotion code is involved: the staging directory is written by hand.
func TestStageBuild_LocalReplacementsResolveInOneBuild(t *testing.T) {
	fixture := filepath.Join(schematest.LibraryRoot(t), "testdata", "render")
	registrytest.NewRegistryFromDir(t, filepath.Join(fixture, "registry"), "testing.opmodel.dev/library-render")

	libDir := writeLibModule(t)
	catDir := catalogWithLabel(t, fixture)
	instDir := instanceImportingLib(t, fixture, libDir)
	platDir := filepath.Join(fixture, "platform")

	dir := t.TempDir()
	glue, err := RenderGlue(GlueInputs{
		InstancePath: "testing.opmodel.dev/library-render/instance@v0",
		PlatformPath: "testing.opmodel.dev/library-render/platform@v0",
		RuntimeName:  "spike",
	})
	require.NoError(t, err)
	writeFiles(t, dir, map[string]string{
		"cue.mod/module.cue": `module: "` + RenderModulePath + `"
language: version: "v0.17.0"
deps: {
	"opmodel.dev/core@v2": v: "` + registrytest.DefaultCoreVersion + `"
	"` + libModulePath + `": v: "v0.0.0"
	"testing.opmodel.dev/library-render/cat@v0": v: "v0.1.0"
	"testing.opmodel.dev/library-render/instance@v0": {v: "v0.0.0", default: true}
	"testing.opmodel.dev/library-render/platform@v0": {v: "v0.0.0", default: true}
	"testing.opmodel.dev/library-render/web_app@v0": v: "v0.1.0"
}
`,
		"cue.mod/local-module.cue": `deps: {
	"` + libModulePath + `": replaceWith: "` + libDir + `"
	"testing.opmodel.dev/library-render/cat@v0": replaceWith: "` + catDir + `"
	"testing.opmodel.dev/library-render/instance@v0": replaceWith: "` + instDir + `"
	"testing.opmodel.dev/library-render/platform@v0": replaceWith: "` + platDir + `"
}
`,
		RenderFileName: string(glue),
	})

	built, err := Build(cuecontext.New(), &Staged{Dir: dir}, loader.Options{})
	require.NoError(t, err, "cue/load serves both replacement directories inside the one build")
	require.NoError(t, built.Err())

	deployment := renderedOutput(t, built, "web", "testing.opmodel.dev/library-render/cat/transformers/deployment-transformer@0.1.0")
	image, err := deployment.LookupPath(cue.ParsePath("spec.image")).String()
	require.NoError(t, err)
	assert.Equal(t, "nginx:from-lib", image, "(a) the instance's import resolved from the never-published module's directory")
	label, err := deployment.LookupPath(cue.ParsePath(`metadata.labels."render.test/catalog"`)).String()
	require.NoError(t, err)
	assert.Equal(t, "local", label, "(b) the deployment's bytes came from the replaced catalog directory, not the published build")
}

// ── Spike: the render module served from memory ─────────────────────

// componentNames returns the field names of the render module's _components.
func componentNames(t *testing.T, built cue.Value) []string {
	t.Helper()
	components := built.LookupPath(cue.MakePath(cue.Hid("_components", RenderModulePath+":render")))
	require.NoError(t, components.Err())
	require.True(t, components.Exists())
	fields, err := components.Fields()
	require.NoError(t, err)
	var names []string
	for fields.Next() {
		names = append(names, fields.Selector().String())
	}
	sort.Strings(names)
	return names
}

func assertAbsent(t *testing.T, path string) {
	t.Helper()
	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err), "%s must not exist on disk", path)
}

// TestStageBuild_RenderModuleServedFromMemory pins the assumption in-memory
// staging rests on: cue/load v0.17 reads cue.mod/module.cue,
// cue.mod/local-module.cue and the glue of the main module, and every
// replacement directory, from load.Config.Overlay under RenderRoot, a root
// that does not exist on disk.
func TestStageBuild_RenderModuleServedFromMemory(t *testing.T) {
	fixture := filepath.Join(schematest.LibraryRoot(t), "testdata", "render")
	registrytest.NewRegistryFromDir(t, filepath.Join(fixture, "registry"), "testing.opmodel.dev/library-render")

	t.Run("overlay instance, on-disk platform", func(t *testing.T) {
		inst := rekeyed(t, filepath.Join(fixture, "instance"), sourcetree.SyntheticRoot("testing.opmodel.dev/library-render/instance", "v0.0.0"))
		plat := &module.Source{Root: filepath.Join(fixture, "platform")}
		staged, err := Stage(inst, plat, "rt", StageOptions{})
		require.NoError(t, err)
		assert.Equal(t, RenderRoot, staged.Dir)
		assertAbsent(t, RenderRoot)

		built, err := Build(cuecontext.New(), staged, loader.Options{})
		require.NoError(t, err, "cue/load serves the whole render module from the overlay")
		require.NoError(t, built.Err())
		assert.Equal(t, []string{"config", "web"}, componentNames(t, built))
		assertAbsent(t, RenderRoot)
	})

	t.Run("both inputs overlay, built concurrently", func(t *testing.T) {
		inst := rekeyed(t, filepath.Join(fixture, "instance"), sourcetree.SyntheticRoot("testing.opmodel.dev/library-render/instance", "v0.0.0"))
		plat := rekeyed(t, filepath.Join(fixture, "platform"), sourcetree.SyntheticRoot("testing.opmodel.dev/library-render/platform", "v0.0.0"))
		staged, err := Stage(inst, plat, "rt", StageOptions{})
		require.NoError(t, err)
		assertAbsent(t, RenderRoot)

		const n = 4
		results := make([][]string, n)
		errs := make([]error, n)
		var wg sync.WaitGroup
		for i := range n {
			wg.Add(1)
			go func() {
				defer wg.Done()
				built, err := Build(cuecontext.New(), staged, loader.Options{})
				if err == nil {
					err = built.Err()
				}
				errs[i] = err
				if err == nil {
					results[i] = componentNames(t, built)
				}
			}()
		}
		wg.Wait()
		for i := range n {
			require.NoError(t, errs[i], "goroutine %d", i)
			assert.Equal(t, []string{"config", "web"}, results[i], "goroutine %d", i)
		}
		assertAbsent(t, RenderRoot)
	})
}

// TestStageBuild_LocalReplacementsServedFromMemory is
// TestStageBuild_LocalReplacementsResolveInOneBuild with the hand-written
// render module served from an overlay under RenderRoot, which does not exist:
// directory replacements of the instance, the platform, a never-published
// module and the catalog all resolve from disk while the main module comes
// from memory. Leaving local-module.cue out of the overlay makes the
// instance import unresolvable, so the overlay's file is the one read.
func TestStageBuild_LocalReplacementsServedFromMemory(t *testing.T) {
	fixture := filepath.Join(schematest.LibraryRoot(t), "testdata", "render")
	registrytest.NewRegistryFromDir(t, filepath.Join(fixture, "registry"), "testing.opmodel.dev/library-render")

	libDir := writeLibModule(t)
	catDir := catalogWithLabel(t, fixture)
	instDir := instanceImportingLib(t, fixture, libDir)
	platDir := filepath.Join(fixture, "platform")

	glue, err := RenderGlue(GlueInputs{
		InstancePath: "testing.opmodel.dev/library-render/instance@v0",
		PlatformPath: "testing.opmodel.dev/library-render/platform@v0",
		RuntimeName:  "spike",
	})
	require.NoError(t, err)
	overlay := map[string][]byte{
		filepath.Join(RenderRoot, "cue.mod", "module.cue"): []byte(`module: "` + RenderModulePath + `"
language: version: "v0.17.0"
deps: {
	"opmodel.dev/core@v2": v: "` + registrytest.DefaultCoreVersion + `"
	"` + libModulePath + `": v: "v0.0.0"
	"testing.opmodel.dev/library-render/cat@v0": v: "v0.1.0"
	"testing.opmodel.dev/library-render/instance@v0": {v: "v0.0.0", default: true}
	"testing.opmodel.dev/library-render/platform@v0": {v: "v0.0.0", default: true}
	"testing.opmodel.dev/library-render/web_app@v0": v: "v0.1.0"
}
`),
		filepath.Join(RenderRoot, "cue.mod", "local-module.cue"): []byte(`deps: {
	"` + libModulePath + `": replaceWith: "` + libDir + `"
	"testing.opmodel.dev/library-render/cat@v0": replaceWith: "` + catDir + `"
	"testing.opmodel.dev/library-render/instance@v0": replaceWith: "` + instDir + `"
	"testing.opmodel.dev/library-render/platform@v0": replaceWith: "` + platDir + `"
}
`),
		filepath.Join(RenderRoot, RenderFileName): glue,
	}

	built, err := Build(cuecontext.New(), &Staged{Dir: RenderRoot, Overlay: overlay}, loader.Options{})
	require.NoError(t, err, "cue/load serves the replacement directories with the main module in memory")
	require.NoError(t, built.Err())
	deployment := renderedOutput(t, built, "web", "testing.opmodel.dev/library-render/cat/transformers/deployment-transformer@0.1.0")
	label, err := deployment.LookupPath(cue.ParsePath(`metadata.labels."render.test/catalog"`)).String()
	require.NoError(t, err)
	assert.Equal(t, "local", label, "the deployment's bytes came from the replaced catalog directory")
	assertAbsent(t, RenderRoot)

	// Negative control: without the overlay's local-module.cue the
	// instance import has nowhere to resolve from.
	delete(overlay, filepath.Join(RenderRoot, "cue.mod", "local-module.cue"))
	_, err = Build(cuecontext.New(), &Staged{Dir: RenderRoot, Overlay: overlay}, loader.Options{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "testing.opmodel.dev/library-render/instance@v0")
	assertAbsent(t, RenderRoot)
}
