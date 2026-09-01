package kernel_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

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
// package fails in the validated entry point and yields no instance.
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
