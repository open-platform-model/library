package renderstage

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/parser"
	"cuelang.org/go/mod/modfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

	// The generated file must parse as CUE.
	_, err = parser.ParseFile(RenderFileName, glue)
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

// stagedFiles lists every file under dir, slash-separated and relative to
// it, sorted: what one render leaves on disk.
func stagedFiles(t *testing.T, dir string) []string {
	t.Helper()
	var files []string
	require.NoError(t, filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			rel, err := filepath.Rel(dir, p)
			require.NoError(t, err)
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	}))
	sort.Strings(files)
	return files
}

func TestStage_ServesOverlayFromMemoryAndWritesRenderModule(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "opm-registry-module", "web_app")
	inst := overlayInstance(root)
	plat := diskPlatform(t)
	dir := t.TempDir()

	staged, err := Stage(dir, inst, plat, "rt", false)
	require.NoError(t, err)
	assert.Equal(t, dir, staged.Dir)

	// The overlay tree is re-keyed under <dir>/instance (the directory the
	// local-module.cue replacement names) on Staged.Overlay, every entry
	// included, and nothing of it is written.
	instDir := filepath.Join(dir, "instance")
	_, err = os.Stat(instDir)
	assert.True(t, os.IsNotExist(err), "no instance/ directory is written for an overlay-mode input")
	require.Len(t, staged.Overlay, len(inst.Overlay))
	for _, rel := range []string{"cue.mod/module.cue", "module.cue", "opm-synth-instance/instance.cue", "opm-synth-instance/values.cue", "opm-synth-instance/notes.md", "opm-synth-instance/nested/deep.cue"} {
		assert.Contains(t, staged.Overlay, filepath.Join(instDir, filepath.FromSlash(rel)), rel)
	}
	assert.Equal(t, "package instance\n\ny: 2\n", string(staged.Overlay[filepath.Join(instDir, "opm-synth-instance", "instance.cue")]))
	assert.Equal(t, []string{"cue.mod/local-module.cue", "cue.mod/module.cue", RenderFileName}, stagedFiles(t, dir), "the staging directory holds only the generated render module")

	// Generated files. The on-disk platform is referenced in place through
	// the local-module.cue replacement.
	moduleCue, err := os.ReadFile(filepath.Join(dir, "cue.mod", "module.cue"))
	require.NoError(t, err)
	assert.Contains(t, string(moduleCue), `module: "`+RenderModulePath+`"`)
	assert.Contains(t, string(moduleCue), `"opmodel.dev/catalogs/opm@v4"`)
	localCue, err := os.ReadFile(filepath.Join(dir, "cue.mod", "local-module.cue"))
	require.NoError(t, err)
	assert.Contains(t, string(localCue), "replaceWith")
	glue, err := os.ReadFile(filepath.Join(dir, RenderFileName))
	require.NoError(t, err)
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
	dir := t.TempDir()

	staged, err := Stage(dir, inst, plat, "rt", false)
	require.NoError(t, err)

	assert.Equal(t, []string{"cue.mod/local-module.cue", "cue.mod/module.cue", RenderFileName}, stagedFiles(t, dir))
	for _, name := range []string{"instance", "platform"} {
		_, err := os.Stat(filepath.Join(dir, name))
		assert.True(t, os.IsNotExist(err), "%s/ is served from memory, never written", name)
	}
	assert.Len(t, staged.Overlay, len(inst.Overlay)+len(plat.Overlay))
	assert.Contains(t, staged.Overlay, filepath.Join(dir, "platform", "platform.cue"))
	assert.Contains(t, staged.Overlay, filepath.Join(dir, "platform", "cue.mod", "module.cue"))
	localCue, err := os.ReadFile(filepath.Join(dir, "cue.mod", "local-module.cue"))
	require.NoError(t, err)
	assert.Contains(t, string(localCue), filepath.Join(dir, "platform"), "the replacement names the in-memory directory")
}

func TestStage_OnDiskInputsCarryNoOverlay(t *testing.T) {
	fixture := filepath.Join(schematest.LibraryRoot(t), "testdata", "render")
	inst := &module.Source{Root: filepath.Join(fixture, "instance")}
	plat := &module.Source{Root: filepath.Join(fixture, "platform")}
	dir := t.TempDir()

	staged, err := Stage(dir, inst, plat, "rt", false)
	require.NoError(t, err)
	assert.Empty(t, staged.Overlay, "on-disk inputs are referenced in place")
	assert.Equal(t, []string{"cue.mod/local-module.cue", "cue.mod/module.cue", RenderFileName}, stagedFiles(t, dir))
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
	dir := t.TempDir()

	staged, err := Stage(dir, inst, plat, "rt", false)
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(dir, "instance"))
	require.True(t, os.IsNotExist(err), "the instance tree is not written")
	require.Len(t, staged.Overlay, len(inst.Overlay))

	// Build with the process environment: the registry helper set
	// CUE_REGISTRY and CUE_CACHE_DIR for this test.
	built, err := Build(cuecontext.New(), staged, nil)
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

	_, err := Stage(t.TempDir(), nil, plat, "rt", false)
	require.ErrorContains(t, err, "instance carries no source")
	_, err = Stage(t.TempDir(), overlayInstance(root), nil, "rt", false)
	require.ErrorContains(t, err, "platform carries no source")
	_, err = Stage(t.TempDir(), overlayInstance(root), plat, "", false)
	require.ErrorContains(t, err, "runtime name")

	// An overlay entry outside its root is refused rather than written
	// somewhere else.
	escaped := overlayInstance(root)
	escaped.Overlay[filepath.Join(string(filepath.Separator), "elsewhere", "x.cue")] = []byte("package x\n")
	_, err = Stage(t.TempDir(), escaped, plat, "rt", false)
	require.ErrorContains(t, err, "outside the source root")

	// Two package clauses in one package directory.
	mixed := overlayInstance(root)
	mixed.Overlay[filepath.Join(root, "opm-synth-instance", "other.cue")] = []byte("package other\n")
	_, err = Stage(t.TempDir(), mixed, plat, "rt", false)
	require.ErrorContains(t, err, "more than one package")

	// No package clause at all.
	bare := overlayInstance(root)
	for k := range bare.Overlay {
		if strings.Contains(k, "opm-synth-instance") {
			delete(bare.Overlay, k)
		}
	}
	bare.Overlay[filepath.Join(root, "opm-synth-instance", "data.cue")] = []byte("a: 1\n")
	_, err = Stage(t.TempDir(), bare, plat, "rt", false)
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
	"opmodel.dev/core@v2": v: "v2.0.0-alpha.7"
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

// readPair returns the staged cue.mod pair's bytes.
func readPair(t *testing.T, dir string) (moduleCue, localCue []byte) {
	t.Helper()
	moduleCue, err := os.ReadFile(filepath.Join(dir, "cue.mod", "module.cue"))
	require.NoError(t, err)
	localCue, err = os.ReadFile(filepath.Join(dir, "cue.mod", "local-module.cue"))
	require.NoError(t, err)
	return moduleCue, localCue
}

func TestStage_RefusesLocalReplacementsUnlessEnabled(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "opm-registry-module", "web_app")
	catDir := t.TempDir()

	// The platform's file carries a replacement: refused, nothing written.
	plat := withLocalFile(t, diskPlatform(t), `deps: "opmodel.dev/catalogs/opm@v4": replaceWith: "`+catDir+`"
`)
	dir := t.TempDir()
	_, err := Stage(dir, overlayInstance(root), plat, "rt", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `platform "testing.opmodel.dev/render/platform@v0"`)
	assert.Contains(t, err.Error(), "cue.mod/local-module.cue")
	assert.Empty(t, stagedFiles(t, dir), "refused before anything is written")

	// The instance's file (overlay mode) carries one: same refusal.
	inst := withLocalFile(t, overlayInstance(root), `deps: "example.com/helpers@v1": replaceWith: "./helpers"
`)
	dir = t.TempDir()
	_, err = Stage(dir, inst, diskPlatform(t), "rt", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `instance "testing.opmodel.dev/modules/web_app@v1"`)
	assert.Contains(t, err.Error(), "cue.mod/local-module.cue")
	assert.Empty(t, stagedFiles(t, dir))

	// A file with no replacement is not a redirection: it stages.
	noRepl := withLocalFile(t, diskPlatform(t), "deps: \"opmodel.dev/core@v2\": v: \"v2.0.0-alpha.7\"\n")
	staged, err := Stage(t.TempDir(), overlayInstance(root), noRepl, "rt", false)
	require.NoError(t, err)
	assert.Nil(t, staged.Replacements)
}

func TestStage_AbsentLocalFileStagesIdenticallyEitherWay(t *testing.T) {
	fixture := filepath.Join(schematest.LibraryRoot(t), "testdata", "render")
	inst := &module.Source{Root: filepath.Join(fixture, "instance")}
	plat := &module.Source{Root: filepath.Join(fixture, "platform")}

	off := t.TempDir()
	stagedOff, err := Stage(off, inst, plat, "rt", false)
	require.NoError(t, err)
	on := t.TempDir()
	stagedOn, err := Stage(on, inst, plat, "rt", true)
	require.NoError(t, err)

	moduleOff, localOff := readPair(t, off)
	moduleOn, localOn := readPair(t, on)
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
	dir := t.TempDir()
	staged, err := Stage(dir, inst, plat, "rt", true)
	require.NoError(t, err)

	helpersDir := filepath.Join(root, "helpers")
	assert.Equal(t, []ReplacementRow{
		{Path: "example.com/helpers@v1", Target: helpersDir, By: "instance"},
		{Path: "opmodel.dev/catalogs/opm@v4", Target: catDir, By: "platform"},
	}, staged.Replacements, "the platform's replacement and the instance-only one; the instance's catalog replacement is inert")
	assert.Equal(t, []string{"cue.mod/local-module.cue", "cue.mod/module.cue", RenderFileName}, stagedFiles(t, dir))

	// The written pair carries exactly those targets, as cue/load reads it.
	moduleCue, localCue := readPair(t, dir)
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
	assert.Equal(t, filepath.Join(dir, "instance"), eff.Deps["testing.opmodel.dev/modules/web_app@v1"].ReplaceWith)
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
	"opmodel.dev/core@v2": v: "v2.0.0-alpha.7"
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

	built, err := Build(cuecontext.New(), &Staged{Dir: dir}, nil)
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
