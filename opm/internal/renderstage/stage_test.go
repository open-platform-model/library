package renderstage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cuelang.org/go/cue/load"
	"cuelang.org/go/cue/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
		Overlay: map[string]load.Source{
			filepath.Join(root, "cue.mod", "module.cue"):   load.FromString(instanceModFile),
			filepath.Join(root, "module.cue"):              load.FromString("package web_app\n\nx: 1\n"),
			filepath.Join(root, pkg, "instance.cue"):       load.FromBytes([]byte("package instance\n\ny: 2\n")),
			filepath.Join(root, pkg, "values.cue"):         load.FromString("package instance\n\nz: 3\n"),
			filepath.Join(root, pkg, "notes.md"):           load.FromString("not cue"),
			filepath.Join(root, pkg, "nested", "deep.cue"): load.FromString("package other\n"),
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

func TestStage_MaterializesOverlayAndWritesRenderModule(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "opm-registry-module", "web_app")
	inst := overlayInstance(root)
	plat := diskPlatform(t)
	dir := t.TempDir()

	staged, err := Stage(dir, inst, plat, "rt")
	require.NoError(t, err)
	assert.Equal(t, dir, staged.Dir)

	// The overlay tree landed under <dir>/instance, rebased from its root.
	instDir := filepath.Join(dir, "instance")
	for _, rel := range []string{"cue.mod/module.cue", "module.cue", "opm-synth-instance/instance.cue", "opm-synth-instance/values.cue", "opm-synth-instance/nested/deep.cue"} {
		_, err := os.Stat(filepath.Join(instDir, filepath.FromSlash(rel)))
		assert.NoError(t, err, rel)
	}
	got, err := os.ReadFile(filepath.Join(instDir, "opm-synth-instance", "instance.cue"))
	require.NoError(t, err)
	assert.Equal(t, "package instance\n\ny: 2\n", string(got))

	// The on-disk platform is referenced in place; the replacements name
	// both directories.
	assert.Equal(t, map[string]string{
		"testing.opmodel.dev/modules/web_app@v1": instDir,
		"testing.opmodel.dev/render/platform@v0": plat.Root,
	}, staged.Promotion.Replacements)

	// Generated files.
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
	assert.Equal(t, "testing.opmodel.dev/modules/web_app/opm-synth-instance@v1:instance", staged.InstanceImport)
	assert.Equal(t, "testing.opmodel.dev/render/platform@v0", staged.PlatformImport)

	// Skew rows ride along (the instance pins a newer catalog).
	require.Len(t, staged.Skew, 2)
	assert.True(t, staged.Skew[0].Newer)
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
	escaped.Overlay[filepath.Join(string(filepath.Separator), "elsewhere", "x.cue")] = load.FromString("package x\n")
	_, err = Stage(t.TempDir(), escaped, plat, "rt")
	require.ErrorContains(t, err, "outside the source root")

	// Two package clauses in one package directory.
	mixed := overlayInstance(root)
	mixed.Overlay[filepath.Join(root, "opm-synth-instance", "other.cue")] = load.FromString("package other\n")
	_, err = Stage(t.TempDir(), mixed, plat, "rt")
	require.ErrorContains(t, err, "more than one package")

	// No package clause at all.
	bare := overlayInstance(root)
	for k := range bare.Overlay {
		if strings.Contains(k, "opm-synth-instance") {
			delete(bare.Overlay, k)
		}
	}
	bare.Overlay[filepath.Join(root, "opm-synth-instance", "data.cue")] = load.FromString("a: 1\n")
	_, err = Stage(t.TempDir(), bare, plat, "rt")
	require.ErrorContains(t, err, "no package clause")
}

func TestRegistryEnv(t *testing.T) {
	assert.Nil(t, RegistryEnv(""))
	t.Setenv("CUE_REGISTRY", "old=host")
	env := RegistryEnv("new=host")
	assert.Contains(t, env, "CUE_REGISTRY=new=host")
	assert.NotContains(t, env, "CUE_REGISTRY=old=host")
	assert.Equal(t, "old=host", os.Getenv("CUE_REGISTRY"), "the process environment is never written")
}

func TestAlternatives(t *testing.T) {
	universe := []string{"x.example/cat/resources/a@v2", "x.example/cat/resources/b@v1", "x.example/cat/resources/a@v1beta1", "x.example/cat/resources/a@v1"}
	assert.Equal(t, []string{"x.example/cat/resources/a@v1beta1", "x.example/cat/resources/a@v2"},
		Alternatives(universe, "x.example/cat/resources/a@v1"))
	assert.Nil(t, Alternatives(universe, "x.example/cat/resources/c@v1"))
}
