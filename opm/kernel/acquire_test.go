package kernel_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	cueerrors "cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/format"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/internal/schematest"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/module"
)

const acquirePlatformFixture = `
package platform
kind: "Platform"
metadata: {
	name: "acquire-platform"
	labels: env: "test"
}
type: "kubernetes"
`

// acquireInstanceFixture is a self-contained, fully concrete instance
// package: no imports, so the acquire path needs no registry.
const acquireInstanceFixture = `
package instance
kind: "ModuleInstance"
metadata: {
	name:      "acquire-demo"
	namespace: "ns"
}
#module: {kind: "Module"}
values: {replicas: 3}
`

// platform-artifact spec, "Acquired platform carries its source".
func TestKernel_AcquirePlatformFromDir_CarriesSource(t *testing.T) {
	dir := schematest.WritePlatformDir(t, acquirePlatformFixture)
	k := kernel.New()
	ctx := context.Background()

	plat, err := k.AcquirePlatformFromDir(ctx, dir)
	require.NoError(t, err)
	require.NotNil(t, plat)

	require.NotNil(t, plat.Metadata)
	assert.Equal(t, "acquire-platform", plat.Metadata.Name)
	assert.Equal(t, "kubernetes", plat.Metadata.Type)

	// platform-artifact spec, "Frontends stop composing load and construct":
	// the raw value is the artifact's Package field, not a second verb.
	require.True(t, plat.Package.Exists())
	assert.Equal(t, "acquire-platform", lookupString(t, plat.Package, "metadata.name"))

	absDir, err := filepath.Abs(dir)
	require.NoError(t, err)
	require.NotNil(t, plat.Source)
	assert.Equal(t, absDir, plat.Source.Root)
	assert.Empty(t, plat.Source.Pkg, "root package")
	assert.Nil(t, plat.Source.Overlay, "on-disk mode")
}

// A relative directory path is stamped as its absolute form.
func TestKernel_AcquirePlatformFromDir_RelativePathAbsolutized(t *testing.T) {
	dir := schematest.WritePlatformDir(t, acquirePlatformFixture)
	parent, leaf := filepath.Split(dir)
	t.Chdir(parent)

	plat, err := kernel.New().AcquirePlatformFromDir(context.Background(), leaf)
	require.NoError(t, err)
	assert.True(t, filepath.IsAbs(plat.Source.Root))
	assert.Equal(t, dir, plat.Source.Root)
}

// platform-artifact spec, "Registry override honored": the option reaches the
// loader (a platform with no registry-backed imports still loads).
func TestKernel_AcquirePlatformFromDir_RegistryOverride(t *testing.T) {
	dir := schematest.WritePlatformDir(t, acquirePlatformFixture)

	plat, err := kernel.New().AcquirePlatformFromDir(context.Background(), dir)
	require.NoError(t, err, "registry override must be accepted even when no imports use it")
	require.NotNil(t, plat.Source)
}

// platform-artifact spec, "Shape-gate failures propagate": the loader's
// sentinels wrap through unchanged and no partial platform is returned.
func TestKernel_AcquirePlatformFromDir_Errors(t *testing.T) {
	k := kernel.New()
	ctx := context.Background()

	t.Run("missing directory", func(t *testing.T) {
		plat, err := k.AcquirePlatformFromDir(ctx, filepath.Join(t.TempDir(), "nope"))
		require.Error(t, err)
		assert.True(t, errors.Is(err, fs.ErrNotExist), "got %v", err)
		assert.Nil(t, plat)
	})

	t.Run("wrong kind", func(t *testing.T) {
		dir := schematest.WritePlatformDir(t, `
package platform
kind: "Module"
metadata: name: "not-a-platform"
type: "kubernetes"
`)
		plat, err := k.AcquirePlatformFromDir(ctx, dir)
		require.Error(t, err)
		assert.True(t, errors.Is(err, oerrors.ErrWrongKind), "got %v", err)
		assert.Nil(t, plat)
	})

	t.Run("missing type", func(t *testing.T) {
		dir := schematest.WritePlatformDir(t, `
package platform
kind: "Platform"
metadata: name: "typeless"
`)
		plat, err := k.AcquirePlatformFromDir(ctx, dir)
		require.Error(t, err)
		assert.True(t, errors.Is(err, oerrors.ErrMissingRequiredField), "got %v", err)
		assert.Nil(t, plat)
	})
}

// artifact-types spec, "Acquired instance carries its source".
func TestKernel_AcquireInstanceFromDir_CarriesSource(t *testing.T) {
	dir := schematest.WriteInstanceDir(t, acquireInstanceFixture)
	k := kernel.New()
	ctx := context.Background()

	inst, err := k.AcquireInstanceFromDir(ctx, dir)
	require.NoError(t, err)
	require.NotNil(t, inst)

	require.NotNil(t, inst.Metadata)
	assert.Equal(t, "acquire-demo", inst.Metadata.Name)
	assert.Equal(t, "ns", inst.Metadata.Namespace)

	require.True(t, inst.Package.Exists())
	assert.Equal(t, "acquire-demo", lookupString(t, inst.Package, "metadata.name"))

	absDir, err := filepath.Abs(dir)
	require.NoError(t, err)
	require.NotNil(t, inst.Source)
	assert.Equal(t, absDir, inst.Source.Root)
	assert.Empty(t, inst.Source.Pkg, "root package")
	assert.Nil(t, inst.Source.Overlay, "on-disk mode")
}

// artifact-types spec, "Validation failures propagate": a non-concrete
// package fails the kernel's concreteness check and yields no instance.
func TestKernel_AcquireInstanceFromDir_NonConcreteRejected(t *testing.T) {
	dir := schematest.WriteInstanceDir(t, `
package instance
kind: "ModuleInstance"
metadata: {
	name:      "draft"
	namespace: "ns"
}
#module: {kind: "Module"}
values: {replicas: int}
`)

	inst, err := kernel.New().AcquireInstanceFromDir(context.Background(), dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not fully concrete")
	assert.Nil(t, inst)
}

// artifact-types spec, "Loader failures propagate".
func TestKernel_AcquireInstanceFromDir_LoaderErrors(t *testing.T) {
	k := kernel.New()
	ctx := context.Background()

	t.Run("missing directory", func(t *testing.T) {
		inst, err := k.AcquireInstanceFromDir(ctx, filepath.Join(t.TempDir(), "nope"))
		require.Error(t, err)
		assert.True(t, errors.Is(err, fs.ErrNotExist), "got %v", err)
		assert.Nil(t, inst)
	})

	t.Run("wrong kind", func(t *testing.T) {
		dir := schematest.WriteInstanceDir(t, `
package instance
kind: "Platform"
metadata: {name: "not-an-instance", namespace: "ns"}
#module: {kind: "Module"}
`)
		inst, err := k.AcquireInstanceFromDir(ctx, dir)
		require.Error(t, err)
		assert.True(t, errors.Is(err, oerrors.ErrWrongKind), "got %v", err)
		assert.Nil(t, inst)
	})

	t.Run("missing required field", func(t *testing.T) {
		dir := schematest.WriteInstanceDir(t, `
package instance
kind: "ModuleInstance"
metadata: {namespace: "ns"}
#module: {kind: "Module"}
`)
		inst, err := k.AcquireInstanceFromDir(ctx, dir)
		require.Error(t, err)
		assert.True(t, errors.Is(err, oerrors.ErrMissingRequiredField), "got %v", err)
		assert.Nil(t, inst)
	})
}

func dirListing(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	require.NoError(t, filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		out = append(out, rel)
		return nil
	}))
	return out
}

// artifact-types spec, "Extra values layer onto the package": two sources
// unify with the package's own values in one build, the Source turns to
// overlay mode carrying the on-disk files plus the rendered values file,
// and the caller's directory is untouched.
func TestKernel_AcquireInstanceFromDir_WithSources_Layers(t *testing.T) {
	k := newRenderKernel(t)
	dir := renderFixtureDir(t, "instance_partial")
	before := dirListing(t, dir)

	inst, err := k.AcquireInstanceFromDir(context.Background(), dir,
		mustSource(t, k, "/values/a.cue", `replicas: 3`),
		mustSource(t, k, "/values/b.cue", `image: "nginx:1.27"`))
	require.NoError(t, err)
	require.NotNil(t, inst)

	replicas, err := inst.Package.LookupPath(cue.ParsePath("values.replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(3), replicas, "source a's field")
	image, err := inst.Package.LookupPath(cue.ParsePath("values.image")).String()
	require.NoError(t, err)
	assert.Equal(t, "nginx:1.27", image, "the package's own field, restated by source b")
	assert.Equal(t, "web-partial", inst.Metadata.Name)

	require.NotNil(t, inst.Source)
	assert.Equal(t, dir, inst.Source.Root)
	assert.Empty(t, inst.Source.Pkg)
	require.NotNil(t, inst.Source.Overlay, "overlay mode")
	assert.Contains(t, inst.Source.Overlay, filepath.Join(dir, "instance.cue"))
	assert.Contains(t, inst.Source.Overlay, filepath.Join(dir, "cue.mod", "module.cue"))
	assert.Contains(t, inst.Source.Overlay, filepath.Join(dir, "opm-values.cue"))
	assert.Len(t, inst.Source.Overlay, 3)

	assert.Equal(t, before, dirListing(t, dir), "the caller's directory is never written to")
}

// The overlay mirrors the on-disk tree: layering a source that adds nothing
// beyond what the package states yields the very same Package the plain
// acquire returns, so no file class cue/load reads from disk is missed.
func TestKernel_AcquireInstanceFromDir_WithSources_MirrorsDisk(t *testing.T) {
	k := newRenderKernel(t)
	dir := renderFixtureDir(t, "instance_partial")
	ctx := context.Background()

	plain, err := k.AcquireInstanceFromDir(ctx, dir)
	require.NoError(t, err)
	assert.Nil(t, plain.Source.Overlay, "no sources: on-disk mode unchanged")

	layered, err := k.AcquireInstanceFromDir(ctx, dir,
		mustSource(t, k, "/values/same.cue", `image: "nginx:1.27"`))
	require.NoError(t, err)
	// cue.Value.Equals is false even across two plain acquires of this
	// fixture (definitions and hidden fields); the exported syntax is the
	// faithful comparison.
	assert.Equal(t, exportedSyntax(t, plain.Package), exportedSyntax(t, layered.Package), "layered package differs from the on-disk build")
	assert.Equal(t, plain.Source.Root, layered.Source.Root)
	assert.Equal(t, plain.Source.Pkg, layered.Source.Pkg)

	// No trailing sources is the plain path.
	same, err := k.AcquireInstanceFromDir(ctx, dir)
	require.NoError(t, err)
	assert.Nil(t, same.Source.Overlay)
}

// A module-less package (no cue.mod, no imports) layers without a registry.
func TestKernel_AcquireInstanceFromDir_WithSources_ModuleLess(t *testing.T) {
	dir := schematest.WriteInstanceDir(t, acquireInstanceFixture)
	k := kernel.New()
	inst, err := k.AcquireInstanceFromDir(context.Background(), dir,
		mustSource(t, k, "/values/extra.cue", `tag: "v1"`))
	require.NoError(t, err)
	replicas, err := inst.Package.LookupPath(cue.ParsePath("values.replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(3), replicas)
	tag, err := inst.Package.LookupPath(cue.ParsePath("values.tag")).String()
	require.NoError(t, err)
	assert.Equal(t, "v1", tag)
	absDir, _ := filepath.Abs(dir)
	assert.Equal(t, absDir, inst.Source.Root)
	assert.Len(t, inst.Source.Overlay, 2)
}

// artifact-types spec, "Conflicting extra values fail at acquisition": the
// error names the conflicting path with positions attributable to the
// values source, and no partial instance is returned.
func TestKernel_AcquireInstanceFromDir_WithSources_ConflictAttributed(t *testing.T) {
	k := newRenderKernel(t)
	dir := renderFixtureDir(t, "instance_partial")
	ctx := context.Background()

	t.Run("against the package's own values", func(t *testing.T) {
		inst, err := k.AcquireInstanceFromDir(ctx, dir,
			mustSource(t, k, "/values/prod.cue", `image: "nginx:1.28"`))
		require.Error(t, err)
		assert.Nil(t, inst)
		assert.Contains(t, err.Error(), "image")
		assert.True(t, positionsName(err, "/values/prod.cue"), "no position names the source: %v", err)
	})

	t.Run("against the module's #config", func(t *testing.T) {
		inst, err := k.AcquireInstanceFromDir(ctx, dir,
			mustSource(t, k, "/values/bad.cue", `replicas: "three"`))
		require.Error(t, err)
		assert.Nil(t, inst)
		assert.Contains(t, err.Error(), "replicas")
		assert.True(t, positionsName(err, "/values/bad.cue"), "no position names the source: %v", err)
	})

	t.Run("between sources", func(t *testing.T) {
		inst, err := k.AcquireInstanceFromDir(ctx, dir,
			mustSource(t, k, "/values/a.cue", `replicas: 2`),
			mustSource(t, k, "/values/b.cue", `replicas: 3`))
		require.Error(t, err)
		assert.Nil(t, inst)
		assert.True(t, positionsName(err, "/values/a.cue") || positionsName(err, "/values/b.cue"), "no position names a source: %v", err)
	})
}

// acquireModuleFixture is a self-contained, import-free #Module package: the
// acquire path needs no registry to build it.
const acquireModuleFixture = `
package mod
kind: "Module"
metadata: {
	name:       "demo"
	modulePath: "example.com/modules/demo@v0"
	version:    "0.1.0"
}
`

// writeTempModuleRoot lays out a CUE module root holding the module package
// at pkgDir (empty for the root package) and returns the root.
func writeTempModuleRoot(t *testing.T, pkgDir, content string) string {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "cue.mod"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "cue.mod", "module.cue"),
		[]byte("module: \"example.com/modules/demo@v0\"\nlanguage: version: \"v0.17.0\"\n"), 0o644))
	dir := root
	if pkgDir != "" {
		dir = filepath.Join(root, pkgDir)
		require.NoError(t, os.MkdirAll(dir, 0o755))
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "module.cue"), []byte(content), 0o644))
	return root
}

// overlayKeys returns the overlay's keys relative to root, sorted.
func overlayKeys(t *testing.T, src *module.Source) []string {
	t.Helper()
	out := make([]string, 0, len(src.Overlay))
	for key := range src.Overlay {
		rel, err := filepath.Rel(src.Root, key)
		require.NoError(t, err)
		out = append(out, filepath.ToSlash(rel))
	}
	sort.Strings(out)
	return out
}

// artifact-types spec, "Acquired module carries an overlay source": every
// .cue file under the module root and nothing else, an empty Pkg at the
// root, and HasSource() true so the module is a SynthesizeInstance input.
func TestKernel_AcquireModuleFromDir_CarriesOverlaySource(t *testing.T) {
	root := writeTempModuleRoot(t, "", acquireModuleFixture)
	// A non-.cue sibling must not enter the overlay: the overlay carries .cue
	// files only, and an embedded data file is read from disk beneath it.
	require.NoError(t, os.WriteFile(filepath.Join(root, "README.md"), []byte("not cue"), 0o644))

	k := kernel.New()
	mod, err := k.AcquireModuleFromDir(context.Background(), root)
	require.NoError(t, err)
	require.NotNil(t, mod)

	require.NotNil(t, mod.Metadata)
	assert.Equal(t, "demo", mod.Metadata.Name)
	assert.Equal(t, "example.com/modules/demo@v0", mod.Metadata.ModulePath)
	assert.Equal(t, "0.1.0", mod.Metadata.Version)

	require.True(t, mod.Package.Exists(), "Package is the built module package")
	assert.Equal(t, "demo", lookupString(t, mod.Package, "metadata.name"))

	require.NotNil(t, mod.Source)
	assert.Equal(t, root, mod.Source.Root)
	assert.Empty(t, mod.Source.Pkg, "root package")
	assert.Equal(t, []string{"cue.mod/module.cue", "module.cue"}, overlayKeys(t, mod.Source))
	assert.True(t, mod.HasSource(), "an acquired module satisfies the synthesis precondition")
}

// A package in a subdirectory names its module root and a relative Pkg; the
// overlay still spans the whole root, so the module file drives resolution.
func TestKernel_AcquireModuleFromDir_Subpackage(t *testing.T) {
	root := writeTempModuleRoot(t, "sub", acquireModuleFixture)

	mod, err := kernel.New().AcquireModuleFromDir(context.Background(), filepath.Join(root, "sub"))
	require.NoError(t, err)
	require.NotNil(t, mod.Source)
	assert.Equal(t, root, mod.Source.Root)
	assert.Equal(t, "sub", mod.Source.Pkg)
	assert.Equal(t, []string{"cue.mod/module.cue", "sub/module.cue"}, overlayKeys(t, mod.Source))
}

// A relative directory path is stamped as its absolute form.
func TestKernel_AcquireModuleFromDir_RelativePathAbsolutized(t *testing.T) {
	root := writeTempModuleRoot(t, "", acquireModuleFixture)
	parent, leaf := filepath.Split(root)
	t.Chdir(parent)

	mod, err := kernel.New().AcquireModuleFromDir(context.Background(), leaf)
	require.NoError(t, err)
	assert.True(t, filepath.IsAbs(mod.Source.Root))
	assert.Equal(t, root, mod.Source.Root)
}

// artifact-types spec, "Shape-gate failures propagate": each sentinel wraps
// through unchanged and no partial module is returned.
func TestKernel_AcquireModuleFromDir_Errors(t *testing.T) {
	k := kernel.New()
	ctx := context.Background()

	t.Run("missing directory", func(t *testing.T) {
		mod, err := k.AcquireModuleFromDir(ctx, filepath.Join(t.TempDir(), "nope"))
		require.Error(t, err)
		assert.True(t, errors.Is(err, fs.ErrNotExist), "got %v", err)
		assert.Nil(t, mod)
	})

	t.Run("non-struct root", func(t *testing.T) {
		root := writeTempModuleRoot(t, "", "package mod\n\n\"just a string\"\n")
		mod, err := k.AcquireModuleFromDir(ctx, root)
		require.Error(t, err)
		assert.True(t, errors.Is(err, oerrors.ErrInvalidPackage), "got %v", err)
		assert.Nil(t, mod)
	})

	t.Run("wrong kind", func(t *testing.T) {
		root := writeTempModuleRoot(t, "", `
package mod
kind: "Platform"
metadata: {
	name:       "not-a-module"
	modulePath: "example.com/modules/demo@v0"
	version:    "0.1.0"
}
`)
		mod, err := k.AcquireModuleFromDir(ctx, root)
		require.Error(t, err)
		assert.True(t, errors.Is(err, oerrors.ErrWrongKind), "got %v", err)
		assert.Nil(t, mod)
	})

	t.Run("missing required field", func(t *testing.T) {
		root := writeTempModuleRoot(t, "", `
package mod
kind: "Module"
metadata: {
	modulePath: "example.com/modules/demo@v0"
	version:    "0.1.0"
}
`)
		mod, err := k.AcquireModuleFromDir(ctx, root)
		require.Error(t, err)
		assert.True(t, errors.Is(err, oerrors.ErrMissingRequiredField), "got %v", err)
		assert.Contains(t, err.Error(), "metadata.name")
		assert.Nil(t, mod)
	})
}

// artifact-types spec, "Acquired module synthesizes like a registry module":
// the render fixture module is served from the in-process registry and read
// from the very tree that registry serves; both acquisitions synthesize and
// render to the same set of objects.
func TestKernel_AcquireModuleFromDir_SynthesizesLikeRegistryModule(t *testing.T) {
	k := newRenderKernel(t)
	ctx := context.Background()

	fromDir, err := k.AcquireModuleFromDir(ctx, renderFixtureDir(t, "registry", "testing.opmodel.dev_library-render_web_app_v0.1.0"))
	require.NoError(t, err)
	require.True(t, fromDir.HasSource())

	fromRegistry, err := k.AcquireModuleFromRegistry(ctx, renderModPath+"@v0", "v0.1.0")
	require.NoError(t, err)
	assert.Equal(t, fromRegistry.Metadata, fromDir.Metadata, "the same module, read two ways")

	plat := acquireRenderPlatform(t, k, "platform")
	values := mustSource(t, k, "values.cue", `{image: "nginx:1.27", replicas: 3}`)

	renderObjects := func(t *testing.T, mod *module.Module) []string {
		t.Helper()
		inst, err := k.SynthesizeInstance(ctx, kernel.InstanceInput{
			Module:    mod,
			Name:      "web-synth",
			Namespace: "default",
			Values:    []kernel.Source{values},
		})
		require.NoError(t, err)
		res, err := k.Render(ctx, kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
		require.NoError(t, err)
		return compiledSummary(t, res.Compiled)
	}

	want := renderObjects(t, fromRegistry)
	assert.ElementsMatch(t, []string{
		"config/ConfigMap/app",
		"web/Deployment/web-synth-web",
		"web/Service/web-synth-web",
	}, want, "the registry-acquired module renders the fixture's objects")
	assert.ElementsMatch(t, want, renderObjects(t, fromDir))
}

// exportedSyntax renders v (definitions and hidden fields included) as
// formatted CUE source for a structural comparison.
func exportedSyntax(t *testing.T, v cue.Value) string {
	t.Helper()
	out, err := format.Node(v.Syntax(cue.Final(), cue.Concrete(false), cue.Definitions(true), cue.Hidden(true)))
	require.NoError(t, err)
	return string(out)
}

// positionsName reports whether any position in the CUE error tree carried
// by err names filename.
func positionsName(err error, filename string) bool {
	var cerr cueerrors.Error
	if !errors.As(err, &cerr) {
		return false
	}
	for _, e := range cueerrors.Errors(cerr) {
		for _, pos := range cueerrors.Positions(e) {
			if pos.Filename() == filename {
				return true
			}
		}
	}
	return false
}

// copyRenderInstance copies a render fixture instance (its cue.mod included)
// into a temp dir, so a test can add its own values files to the package.
func copyRenderInstance(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.CopyFS(dir, os.DirFS(renderFixtureDir(t, name))))
	return dir
}

func writeValuesFile(t *testing.T, dir, body string) string {
	t.Helper()
	path := filepath.Join(dir, "values.cue")
	require.NoError(t, os.WriteFile(path, []byte("package instance\n\nvalues: {"+body+"}\n"), 0o600))
	return path
}

// The package's own values are checked against #config on every acquire, the
// same as -f sources: a key the schema does not have is refused at the file
// that holds it, with and without trailing sources.
func TestKernel_AcquireInstanceFromDir_OwnValues_UnknownKeyRefused(t *testing.T) {
	k := newRenderKernel(t)
	ctx := context.Background()

	cases := []struct {
		name    string
		sources []kernel.Source
	}{
		{name: "no sources"},
		{name: "with a clean source", sources: []kernel.Source{mustSource(t, k, "/values/ok.cue", `replicas: 4`)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := copyRenderInstance(t, "instance_partial")
			valuesFile := writeValuesFile(t, dir, "replcas: 5")

			inst, err := k.AcquireInstanceFromDir(ctx, dir, tc.sources...)
			require.Error(t, err)
			assert.Nil(t, inst)
			assert.Contains(t, err.Error(), "field not allowed")
			assert.True(t, hasErrorPath(err, "values.replcas"), "no error names values.replcas: %v", err)
			assert.True(t, positionsName(err, valuesFile), "no position names values.cue: %v", err)
		})
	}

	t.Run("in instance.cue", func(t *testing.T) {
		dir := copyRenderInstance(t, "instance_partial")
		path := filepath.Join(dir, "instance.cue")
		body, err := os.ReadFile(path)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(path, append(body, []byte("\nvalues: replcas: 5\n")...), 0o600))

		inst, err := k.AcquireInstanceFromDir(ctx, dir)
		require.Error(t, err)
		assert.Nil(t, inst)
		assert.Contains(t, err.Error(), "field not allowed")
		assert.True(t, hasErrorPath(err, "values.replcas"), "no error names values.replcas: %v", err)
		assert.True(t, positionsName(err, filepath.Join(dir, "instance.cue")), "no position names instance.cue: %v", err)
	})

	t.Run("a -f source keeps its own attribution", func(t *testing.T) {
		dir := copyRenderInstance(t, "instance_partial")
		inst, err := k.AcquireInstanceFromDir(ctx, dir,
			mustSource(t, k, "/values/typo.cue", `replcas: 5`))
		require.Error(t, err)
		assert.Nil(t, inst)
		assert.Contains(t, err.Error(), "field not allowed")
		assert.True(t, hasErrorPath(err, "values.replcas"), "no error names values.replcas: %v", err)
		assert.True(t, positionsName(err, "/values/typo.cue"), "no position names the source: %v", err)
	})
}

// A value of the wrong type in the package's own values is refused, whether
// or not a component reads the setting.
func TestKernel_AcquireInstanceFromDir_OwnValues_TypeMismatchRefused(t *testing.T) {
	k := newRenderKernel(t)
	dir := copyRenderInstance(t, "instance_partial")
	writeValuesFile(t, dir, `replicas: "two"`)

	inst, err := k.AcquireInstanceFromDir(context.Background(), dir)
	require.Error(t, err)
	assert.Nil(t, inst)
	assert.True(t, positionsName(err, filepath.Join(dir, "values.cue")), "no position names values.cue: %v", err)
	assert.Contains(t, err.Error(), "replicas")
}

// Valid values pass the values check even when they are not concrete: only a
// key the schema lacks or a violated type or constraint is refused by it. A
// default-carrying value is concrete once defaults resolve and acquires; a
// bare constraint is valid for #config, so the refusal is the instance's
// concreteness gate, never a values error.
func TestKernel_AcquireInstanceFromDir_OwnValues_ValidNonConcreteAcquires(t *testing.T) {
	k := newRenderKernel(t)
	ctx := context.Background()

	t.Run("a default acquires, with and without sources", func(t *testing.T) {
		dir := copyRenderInstance(t, "instance_partial")
		writeValuesFile(t, dir, "replicas: int | *3")

		inst, err := k.AcquireInstanceFromDir(ctx, dir)
		require.NoError(t, err)
		require.NotNil(t, inst)

		inst, err = k.AcquireInstanceFromDir(ctx, dir, mustSource(t, k, "/values/a.cue", `image: "nginx:1.27"`))
		require.NoError(t, err)
		require.NotNil(t, inst)
	})

	t.Run("a bare constraint is not a values error", func(t *testing.T) {
		dir := copyRenderInstance(t, "instance_partial")
		writeValuesFile(t, dir, "replicas: >=1")

		for _, sources := range [][]kernel.Source{nil, {mustSource(t, k, "/values/a.cue", `image: "nginx:1.27"`)}} {
			inst, err := k.AcquireInstanceFromDir(ctx, dir, sources...)
			require.Error(t, err)
			assert.Nil(t, inst)
			assert.Contains(t, err.Error(), "not fully concrete")
			assert.NotContains(t, err.Error(), "field not allowed")
		}
	})
}

// hasErrorPath reports whether any error in err's CUE tree is at path.
func hasErrorPath(err error, path string) bool {
	var cerr cueerrors.Error
	if !errors.As(err, &cerr) {
		return false
	}
	for _, e := range cueerrors.Errors(cerr) {
		if strings.Join(e.Path(), ".") == path {
			return true
		}
	}
	return false
}

// artifact-types spec, "A missing directory fails before any tree read" and
// "A file path is refused as not a directory": every directory verb checks
// the path first, so the text is the stat check's and never the walker's.
func TestKernel_AcquireFromDir_PathErrors(t *testing.T) {
	k := kernel.New()
	ctx := context.Background()
	extra := mustSource(t, k, "/values/extra.cue", `tag: "v1"`)

	verbs := []struct {
		name    string
		label   string
		acquire func(dir string) (any, error)
	}{
		{"module", "module", func(dir string) (any, error) {
			mod, err := k.AcquireModuleFromDir(ctx, dir)
			if mod == nil {
				return nil, err
			}
			return mod, err
		}},
		{"catalog", "catalog", func(dir string) (any, error) {
			cat, err := k.AcquireCatalogFromDir(ctx, dir)
			if cat == nil {
				return nil, err
			}
			return cat, err
		}},
		{"platform", "platform", func(dir string) (any, error) {
			plat, err := k.AcquirePlatformFromDir(ctx, dir)
			if plat == nil {
				return nil, err
			}
			return plat, err
		}},
		{"instance", "instance", func(dir string) (any, error) {
			inst, err := k.AcquireInstanceFromDir(ctx, dir)
			if inst == nil {
				return nil, err
			}
			return inst, err
		}},
		{"instance with values", "instance", func(dir string) (any, error) {
			inst, err := k.AcquireInstanceFromDir(ctx, dir, extra)
			if inst == nil {
				return nil, err
			}
			return inst, err
		}},
	}

	for _, v := range verbs {
		t.Run(v.name+"/missing directory", func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "nope")
			got, err := v.acquire(dir)
			require.Error(t, err)
			assert.Nil(t, got)
			assert.True(t, strings.HasPrefix(err.Error(), `accessing `+v.label+` directory "`+dir+`": `), "got %v", err)
			assert.True(t, errors.Is(err, fs.ErrNotExist), "got %v", err)
			assert.NotContains(t, err.Error(), "reading module tree")
		})
		t.Run(v.name+"/regular file", func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "file.cue")
			require.NoError(t, os.WriteFile(file, []byte("package x\n"), 0o644))
			got, err := v.acquire(file)
			require.Error(t, err)
			assert.Nil(t, got)
			assert.Equal(t, v.label+` path "`+file+`" is not a directory`, err.Error())
		})
	}
}

// artifact-types spec, "A module embedding a non-CUE file acquires from a
// directory": the embed attribute reads the data file beside the package,
// whether the package is the module root or a subdirectory, and the stamped
// overlay still carries .cue files only.
func TestKernel_AcquireModuleFromDir_EmbedsNonCUEFile(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"cue.mod/module.cue": "module: \"example.com/modules/demo@v0\"\nlanguage: version: \"v0.17.0\"\n",
		"module.cue": `@extern(embed)

package mod

kind: "Module"
metadata: {
	name:       "demo"
	modulePath: "example.com/modules/demo@v0"
	version:    "0.1.0"
}
data: _ @embed(file="data.json")
`,
		"data.json": `{"greeting": "hello"}`,
		"sub/module.cue": `@extern(embed)

package sub

kind: "Module"
metadata: {
	name:       "demo-sub"
	modulePath: "example.com/modules/demo@v0"
	version:    "0.1.0"
}
data: _ @embed(file="d.json")
`,
		"sub/d.json": `{"greeting": "from sub"}`,
	}
	for rel, body := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, []byte(body), 0o644))
	}

	k := kernel.New()
	ctx := context.Background()

	mod, err := k.AcquireModuleFromDir(ctx, root)
	require.NoError(t, err)
	assert.Equal(t, "hello", lookupString(t, mod.Package, "data.greeting"))
	assert.Equal(t, []string{"cue.mod/module.cue", "module.cue", "sub/module.cue"}, overlayKeys(t, mod.Source))

	sub, err := k.AcquireModuleFromDir(ctx, filepath.Join(root, "sub"))
	require.NoError(t, err)
	assert.Equal(t, "from sub", lookupString(t, sub.Package, "data.greeting"))
	assert.Equal(t, "sub", sub.Source.Pkg)
	assert.Equal(t, []string{"cue.mod/module.cue", "module.cue", "sub/module.cue"}, overlayKeys(t, sub.Source))
}
