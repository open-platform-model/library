package kernel_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	cueerrors "cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/format"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loaderfile "github.com/open-platform-model/library/opm/helper/loader/file"
	"github.com/open-platform-model/library/opm/kernel"
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

	plat, err := k.AcquirePlatformFromDir(ctx, dir, loaderfile.LoadOptions{})
	require.NoError(t, err)
	require.NotNil(t, plat)

	require.NotNil(t, plat.Metadata)
	assert.Equal(t, "acquire-platform", plat.Metadata.Name)
	assert.Equal(t, "kubernetes", plat.Metadata.Type)

	want, err := k.LoadPlatformPackage(ctx, dir, loaderfile.LoadOptions{})
	require.NoError(t, err)
	assert.True(t, plat.Package.Equals(want), "Package is what LoadPlatformPackage returns for the directory")

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

	plat, err := kernel.New().AcquirePlatformFromDir(context.Background(), leaf, loaderfile.LoadOptions{})
	require.NoError(t, err)
	assert.True(t, filepath.IsAbs(plat.Source.Root))
	assert.Equal(t, dir, plat.Source.Root)
}

// platform-artifact spec, "Registry override honored": the option reaches the
// loader (a platform with no registry-backed imports still loads).
func TestKernel_AcquirePlatformFromDir_RegistryOverride(t *testing.T) {
	dir := writeTempPlatformDir(t, acquirePlatformFixture)

	plat, err := kernel.New().AcquirePlatformFromDir(context.Background(), dir, loaderfile.LoadOptions{
		Registry: "testing.opmodel.dev=localhost:5000+insecure",
	})
	require.NoError(t, err, "registry override must be accepted even when no imports use it")
	require.NotNil(t, plat.Source)
}

// platform-artifact spec, "Shape-gate failures propagate": the loader's
// sentinels wrap through unchanged and no partial platform is returned.
func TestKernel_AcquirePlatformFromDir_Errors(t *testing.T) {
	k := kernel.New()
	ctx := context.Background()

	t.Run("missing directory", func(t *testing.T) {
		plat, err := k.AcquirePlatformFromDir(ctx, filepath.Join(t.TempDir(), "nope"), loaderfile.LoadOptions{})
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
		plat, err := k.AcquirePlatformFromDir(ctx, dir, loaderfile.LoadOptions{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, loaderfile.ErrWrongKind), "got %v", err)
		assert.Nil(t, plat)
	})

	t.Run("missing type", func(t *testing.T) {
		dir := writeTempPlatformDir(t, `
package platform
kind: "Platform"
metadata: name: "typeless"
`)
		plat, err := k.AcquirePlatformFromDir(ctx, dir, loaderfile.LoadOptions{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, loaderfile.ErrMissingRequiredField), "got %v", err)
		assert.Nil(t, plat)
	})
}

// artifact-types spec, "Acquired instance carries its source".
func TestKernel_AcquireInstanceFromDir_CarriesSource(t *testing.T) {
	dir := writeTempInstanceDir(t, acquireInstanceFixture)
	k := kernel.New()
	ctx := context.Background()

	inst, err := k.AcquireInstanceFromDir(ctx, dir, loaderfile.LoadOptions{})
	require.NoError(t, err)
	require.NotNil(t, inst)

	require.NotNil(t, inst.Metadata)
	assert.Equal(t, "acquire-demo", inst.Metadata.Name)
	assert.Equal(t, "ns", inst.Metadata.Namespace)

	want, err := k.LoadInstancePackage(ctx, dir, loaderfile.LoadOptions{})
	require.NoError(t, err)
	assert.True(t, inst.Package.Equals(want), "Package is what LoadInstancePackage returns for the directory")

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

	inst, err := kernel.New().AcquireInstanceFromDir(context.Background(), dir, loaderfile.LoadOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not fully concrete")
	assert.Nil(t, inst)
}

// artifact-types spec, "Loader failures propagate".
func TestKernel_AcquireInstanceFromDir_LoaderErrors(t *testing.T) {
	k := kernel.New()
	ctx := context.Background()

	t.Run("missing directory", func(t *testing.T) {
		inst, err := k.AcquireInstanceFromDir(ctx, filepath.Join(t.TempDir(), "nope"), loaderfile.LoadOptions{})
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
		inst, err := k.AcquireInstanceFromDir(ctx, dir, loaderfile.LoadOptions{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, loaderfile.ErrWrongKind), "got %v", err)
		assert.Nil(t, inst)
	})

	t.Run("missing required field", func(t *testing.T) {
		dir := writeTempInstanceDir(t, `
package instance
kind: "ModuleInstance"
metadata: {namespace: "ns"}
#module: {kind: "Module"}
`)
		inst, err := k.AcquireInstanceFromDir(ctx, dir, loaderfile.LoadOptions{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, loaderfile.ErrMissingRequiredField), "got %v", err)
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
func TestKernel_AcquireInstanceFromDir_WithValues_Layers(t *testing.T) {
	k := newRenderKernel(t)
	dir := renderFixtureDir(t, "instance_partial")
	before := dirListing(t, dir)

	inst, err := k.AcquireInstanceFromDir(context.Background(), dir, loaderfile.LoadOptions{},
		kernel.WithValues(
			mustSource(t, k, "/values/a.cue", `replicas: 3`),
			mustSource(t, k, "/values/b.cue", `image: "nginx:1.27"`),
		))
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
func TestKernel_AcquireInstanceFromDir_WithValues_MirrorsDisk(t *testing.T) {
	k := newRenderKernel(t)
	dir := renderFixtureDir(t, "instance_partial")
	ctx := context.Background()

	plain, err := k.AcquireInstanceFromDir(ctx, dir, loaderfile.LoadOptions{})
	require.NoError(t, err)
	assert.Nil(t, plain.Source.Overlay, "no option: on-disk mode unchanged")

	layered, err := k.AcquireInstanceFromDir(ctx, dir, loaderfile.LoadOptions{},
		kernel.WithValues(mustSource(t, k, "/values/same.cue", `image: "nginx:1.27"`)))
	require.NoError(t, err)
	// cue.Value.Equals is false even across two plain acquires of this
	// fixture (definitions and hidden fields); the exported syntax is the
	// faithful comparison.
	assert.Equal(t, exportedSyntax(t, plain.Package), exportedSyntax(t, layered.Package), "layered package differs from the on-disk build")
	assert.Equal(t, plain.Source.Root, layered.Source.Root)
	assert.Equal(t, plain.Source.Pkg, layered.Source.Pkg)

	// An empty option list is the plain path.
	same, err := k.AcquireInstanceFromDir(ctx, dir, loaderfile.LoadOptions{}, kernel.WithValues())
	require.NoError(t, err)
	assert.Nil(t, same.Source.Overlay)
}

// A module-less package (no cue.mod, no imports) layers without a registry.
func TestKernel_AcquireInstanceFromDir_WithValues_ModuleLess(t *testing.T) {
	dir := writeTempInstanceDir(t, acquireInstanceFixture)
	k := kernel.New()
	inst, err := k.AcquireInstanceFromDir(context.Background(), dir, loaderfile.LoadOptions{},
		kernel.WithValues(mustSource(t, k, "/values/extra.cue", `tag: "v1"`)))
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
func TestKernel_AcquireInstanceFromDir_WithValues_ConflictAttributed(t *testing.T) {
	k := newRenderKernel(t)
	dir := renderFixtureDir(t, "instance_partial")
	ctx := context.Background()

	t.Run("against the package's own values", func(t *testing.T) {
		inst, err := k.AcquireInstanceFromDir(ctx, dir, loaderfile.LoadOptions{},
			kernel.WithValues(mustSource(t, k, "/values/prod.cue", `image: "nginx:1.28"`)))
		require.Error(t, err)
		assert.Nil(t, inst)
		assert.Contains(t, err.Error(), "image")
		assert.True(t, positionsName(err, "/values/prod.cue"), "no position names the source: %v", err)
	})

	t.Run("against the module's #config", func(t *testing.T) {
		inst, err := k.AcquireInstanceFromDir(ctx, dir, loaderfile.LoadOptions{},
			kernel.WithValues(mustSource(t, k, "/values/bad.cue", `replicas: "three"`)))
		require.Error(t, err)
		assert.Nil(t, inst)
		assert.Contains(t, err.Error(), "replicas")
		assert.True(t, positionsName(err, "/values/bad.cue"), "no position names the source: %v", err)
	})

	t.Run("between sources", func(t *testing.T) {
		inst, err := k.AcquireInstanceFromDir(ctx, dir, loaderfile.LoadOptions{},
			kernel.WithValues(
				mustSource(t, k, "/values/a.cue", `replicas: 2`),
				mustSource(t, k, "/values/b.cue", `replicas: 3`),
			))
		require.Error(t, err)
		assert.Nil(t, inst)
		assert.True(t, positionsName(err, "/values/a.cue") || positionsName(err, "/values/b.cue"), "no position names a source: %v", err)
	})
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
