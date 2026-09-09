package kernel_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"cuelang.org/go/cue"
	cueerrors "cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/format"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/open-platform-model/library/opm/errors"
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

func writeTempPlatformDir(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "platform.cue"), []byte(content), 0o644))
	return dir
}

// platform-artifact spec, "Acquired platform carries its source".
func TestKernel_AcquirePlatformFromDir_CarriesSource(t *testing.T) {
	dir := writeTempPlatformDir(t, acquirePlatformFixture)
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
	dir := writeTempPlatformDir(t, acquirePlatformFixture)
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
	dir := writeTempPlatformDir(t, acquirePlatformFixture)

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
		dir := writeTempPlatformDir(t, `
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
		dir := writeTempPlatformDir(t, `
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
	dir := writeTempInstanceDir(t, acquireInstanceFixture)
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
	dir := writeTempInstanceDir(t, `
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
		dir := writeTempInstanceDir(t, `
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
		dir := writeTempInstanceDir(t, `
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
	dir := writeTempInstanceDir(t, acquireInstanceFixture)
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
	// A non-.cue sibling must not enter the overlay: cue/load does not read it.
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
