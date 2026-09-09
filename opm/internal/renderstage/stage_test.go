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

	staged, err := Stage(dir, inst, plat, "rt")
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

	staged, err := Stage(dir, inst, plat, "rt")
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

	staged, err := Stage(dir, inst, plat, "rt")
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

	staged, err := Stage(dir, inst, plat, "rt")
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

	_, err := Stage(t.TempDir(), nil, plat, "rt")
	require.ErrorContains(t, err, "instance carries no source")
	_, err = Stage(t.TempDir(), overlayInstance(root), nil, "rt")
	require.ErrorContains(t, err, "platform carries no source")
	_, err = Stage(t.TempDir(), overlayInstance(root), plat, "")
	require.ErrorContains(t, err, "runtime name")

	// An overlay entry outside its root is refused rather than written
	// somewhere else.
	escaped := overlayInstance(root)
	escaped.Overlay[filepath.Join(string(filepath.Separator), "elsewhere", "x.cue")] = []byte("package x\n")
	_, err = Stage(t.TempDir(), escaped, plat, "rt")
	require.ErrorContains(t, err, "outside the source root")

	// Two package clauses in one package directory.
	mixed := overlayInstance(root)
	mixed.Overlay[filepath.Join(root, "opm-synth-instance", "other.cue")] = []byte("package other\n")
	_, err = Stage(t.TempDir(), mixed, plat, "rt")
	require.ErrorContains(t, err, "more than one package")

	// No package clause at all.
	bare := overlayInstance(root)
	for k := range bare.Overlay {
		if strings.Contains(k, "opm-synth-instance") {
			delete(bare.Overlay, k)
		}
	}
	bare.Overlay[filepath.Join(root, "opm-synth-instance", "data.cue")] = []byte("a: 1\n")
	_, err = Stage(t.TempDir(), bare, plat, "rt")
	require.ErrorContains(t, err, "no package clause")
}
