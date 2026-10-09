package kernel_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"cuelang.org/go/mod/modregistrytest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/internal/schematest"
	"github.com/open-platform-model/library/opm/kernel"
)

// fetch-error-classification spec: every library path that returns a
// registry fetch or dependency resolution failure classifies it, through the
// public verbs.

// requireFetchError returns the *FetchError in err's chain.
func requireFetchError(t *testing.T, err error) *oerrors.FetchError {
	t.Helper()
	require.Error(t, err)
	var fe *oerrors.FetchError
	require.True(t, errors.As(err, &fe), "a *FetchError in the chain of %q", err)
	return fe
}

// freshCache points CUE_CACHE_DIR at a cache that shares nothing, so a
// dependency must be fetched and a warm cache never hides a failure.
func freshCache(t *testing.T) {
	t.Helper()
	t.Setenv("CUE_CACHE_DIR", schematest.IsolatedCacheDir(t))
}

// writeDependentModule writes a #Module-shaped directory module whose
// package imports test.example/dep@v0 at v0.0.1, so loading it must fetch
// that dependency, and returns its directory.
func writeDependentModule(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cue.mod"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cue.mod", "module.cue"), []byte(
		"module: \"fetch.example/app@v0\"\nlanguage: version: \"v0.17.0\"\n"+
			"deps: \"test.example/dep@v0\": v: \"v0.0.1\"\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.cue"), []byte(
		"package app\n\nimport \"test.example/dep\"\n\nkind: \"Module\"\nx: dep.y\n"), 0o644))
	return dir
}

func TestFetchClassify_RegistryVerbs(t *testing.T) {
	modPath := registrytest.UniquePath(t, "modules") + "/hello"
	catPath := registrytest.UniquePath(t, "cat")
	served := registrytest.NewModuleRegistry(t,
		[]registrytest.ModuleFixture{{Path: modPath, Version: "0.0.2", File: registrytest.BuildModuleFile("hello", "hello", modPath+"@v0", "")}},
		[]registrytest.CatalogFixture{{Path: catPath, Version: "0.1.0", Body: registrytest.BuildCatalog(catPath, "0.1.0")}})
	ctx := context.Background()

	acquire := map[string]func(k *kernel.Kernel, path, version string) error{
		"module": func(k *kernel.Kernel, path, version string) error {
			_, err := k.AcquireModuleFromRegistry(ctx, path, version)
			return err
		},
		"catalog": func(k *kernel.Kernel, path, version string) error {
			_, err := k.AcquireCatalogFromRegistry(ctx, path, version)
			return err
		},
	}
	paths := map[string]string{"module": modPath, "catalog": catPath}

	for label, verb := range acquire {
		path := paths[label]
		t.Run(label+"/absent version", func(t *testing.T) {
			err := verb(kernel.New(kernel.WithRegistry(served)), path+"@v0", "0.0.9")
			fe := requireFetchError(t, err)
			assert.Equal(t, oerrors.FetchNotFound, fe.Kind)
			assert.Equal(t, path+"@v0.0.9", fe.Coordinate)
			assert.NotErrorIs(t, err, oerrors.ErrTransient)
			// The text the verb returned before classification existed.
			assert.Equal(t, fmt.Sprintf("fetching %s %s@v0.0.9: module %s@v0.0.9: module not found", label, path, path), err.Error())
		})
		t.Run(label+"/unreachable registry", func(t *testing.T) {
			freshCache(t)
			err := verb(kernel.New(kernel.WithRegistry(registrytest.UnreachableRegistry(t))), path+"@v0", "0.0.2")
			fe := requireFetchError(t, err)
			assert.Equal(t, oerrors.FetchUnreachable, fe.Kind)
			assert.Equal(t, path+"@v0.0.2", fe.Coordinate)
			assert.ErrorIs(t, err, oerrors.ErrTransient)
		})
		t.Run(label+"/401", func(t *testing.T) {
			freshCache(t)
			err := verb(kernel.New(kernel.WithRegistry(registrytest.NewStatusRegistry(t, 401))), path+"@v0", "0.0.2")
			fe := requireFetchError(t, err)
			assert.Equal(t, oerrors.FetchUnauthorized, fe.Kind)
			assert.Equal(t, 401, fe.Status)
			assert.NotErrorIs(t, err, oerrors.ErrTransient)
		})
	}
}

func TestFetchClassify_DirModuleWithUnreachableDependency(t *testing.T) {
	freshCache(t)
	k := kernel.New(kernel.WithRegistry(registrytest.UnreachableRegistry(t)))
	_, err := k.AcquireModuleFromDir(context.Background(), writeDependentModule(t))
	fe := requireFetchError(t, err)
	assert.Equal(t, oerrors.FetchUnreachable, fe.Kind)
	assert.Empty(t, fe.Coordinate, "a load site does not know which module failed")
	assert.ErrorIs(t, err, oerrors.ErrTransient)
}

// A registry that uses token authentication, whose token endpoint refuses
// the token request, has answered: the failure is a refusal, never an
// unreachable registry, and it is not transient.
func TestFetchClassify_TokenEndpointRefusal(t *testing.T) {
	ctx := context.Background()
	t.Run("directory module with a dependency/401", func(t *testing.T) {
		freshCache(t)
		k := kernel.New(kernel.WithRegistry(registrytest.NewTokenRegistry(t, 401)))
		_, err := k.AcquireModuleFromDir(ctx, writeDependentModule(t))
		fe := requireFetchError(t, err)
		assert.Equal(t, oerrors.FetchUnauthorized, fe.Kind)
		assert.Equal(t, 401, fe.Status)
		assert.NotErrorIs(t, err, oerrors.ErrTransient)
	})
	t.Run("registry module/401", func(t *testing.T) {
		freshCache(t)
		k := kernel.New(kernel.WithRegistry(registrytest.NewTokenRegistry(t, 401)))
		_, err := k.AcquireModuleFromRegistry(ctx, "test.example/dep@v0", "0.0.1")
		fe := requireFetchError(t, err)
		assert.Equal(t, oerrors.FetchUnauthorized, fe.Kind)
		assert.Equal(t, 401, fe.Status)
		assert.NotErrorIs(t, err, oerrors.ErrTransient)
	})
	// CUE's registry client reports a 403 on a version lookup as not found
	// and drops the status, for the token endpoint's answer too.
	t.Run("registry module/403", func(t *testing.T) {
		freshCache(t)
		k := kernel.New(kernel.WithRegistry(registrytest.NewTokenRegistry(t, 403)))
		_, err := k.AcquireModuleFromRegistry(ctx, "test.example/dep@v0", "0.0.1")
		fe := requireFetchError(t, err)
		assert.Equal(t, oerrors.FetchNotFound, fe.Kind)
		assert.Zero(t, fe.Status)
		assert.NotErrorIs(t, err, oerrors.ErrTransient)
	})
}

// requireResolutionError returns the *ResolutionError in err's chain, and
// holds that the chain carries no *FetchError and is not transient.
func requireResolutionError(t *testing.T, err error) *oerrors.ResolutionError {
	t.Helper()
	require.Error(t, err)
	var re *oerrors.ResolutionError
	require.True(t, errors.As(err, &re), "a *ResolutionError in the chain of %q", err)
	var fe *oerrors.FetchError
	assert.False(t, errors.As(err, &fe), "no *FetchError in %q", err)
	assert.NotErrorIs(t, err, oerrors.ErrTransient)
	return re
}

// servedDependency serves test.example/dep@v0 at v0.0.1, whose one package
// is the module root, and returns the registry mapping.
func servedDependency(t *testing.T) string {
	t.Helper()
	reg, err := modregistrytest.New(fstest.MapFS{
		"test.example_dep_v0.0.1/cue.mod/module.cue": &fstest.MapFile{Data: []byte("module: \"test.example/dep@v0\"\nlanguage: version: \"v0.17.0\"\n")},
		"test.example_dep_v0.0.1/dep.cue":            &fstest.MapFile{Data: []byte("package dep\n\ny: 1\n")},
	}, "")
	require.NoError(t, err)
	t.Cleanup(reg.Close)
	return reg.Host() + "+insecure"
}

// An import no registry interaction failed is an author defect, typed as a
// *ResolutionError of kind ResolutionImportUnprovided: a package missing from
// the module's own path, an import of a module the module never declared,
// and a package missing from a declared dependency that was fetched. The
// message is the text the verb returned before the type existed.
func TestFetchClassify_UnresolvableImportIsResolutionError(t *testing.T) {
	for name, tc := range map[string]struct {
		importPath string
		deps       string
		registry   func(t *testing.T) string
	}{
		"own path":   {"fetch.example/app/missing", "", registrytest.UnreachableRegistry},
		"undeclared": {"test.example/undeclared@v0", "", registrytest.UnreachableRegistry},
		"package missing from a declared dependency": {"test.example/dep/missing", "deps: \"test.example/dep@v0\": v: \"v0.0.1\"\n", servedDependency},
	} {
		t.Run(name, func(t *testing.T) {
			freshCache(t)
			dir := t.TempDir()
			require.NoError(t, os.MkdirAll(filepath.Join(dir, "cue.mod"), 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "cue.mod", "module.cue"), []byte(
				"module: \"fetch.example/app@v0\"\nlanguage: version: \"v0.17.0\"\n"+tc.deps), 0o644))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "app.cue"), []byte(
				"package app\n\nimport m \""+tc.importPath+"\"\n\nkind: \"Module\"\nx: m.y\n"), 0o644))
			k := kernel.New(kernel.WithRegistry(tc.registry(t)))
			_, err := k.AcquireModuleFromDir(context.Background(), dir)
			re := requireResolutionError(t, err)
			assert.Equal(t, oerrors.ResolutionImportUnprovided, re.Kind)
			assert.Contains(t, err.Error(), "cannot find module providing package "+tc.importPath)
		})
	}
}

// A file-backed values source that imports a module its own cue.mod does not
// declare is the same author defect, through AcquireInstanceFromDir.
func TestFetchClassify_ValuesFileWithUndeclaredImport(t *testing.T) {
	inst := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(inst, "cue.mod"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(inst, "cue.mod", "module.cue"), []byte(
		"module: \"fetch.example/inst@v0\"\nlanguage: version: \"v0.17.0\"\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(inst, "inst.cue"), []byte("package inst\n\nkind: \"ModuleInstance\"\n"), 0o644))

	values := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(values, "cue.mod"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(values, "cue.mod", "module.cue"), []byte(
		"module: \"values.example/site@v0\"\nlanguage: version: \"v0.17.0\"\n"), 0o644))
	path := filepath.Join(values, "values.cue")
	require.NoError(t, os.WriteFile(path, []byte(
		"package values\n\nimport defaults \"test.example/undeclared@v0\"\n\nvalues: sentinel: defaults.sentinel\n"), 0o644))

	freshCache(t)
	k := kernel.New(kernel.WithRegistry(registrytest.UnreachableRegistry(t)))
	src, err := k.LoadSourceFromFile(path)
	require.NoError(t, err)
	_, err = k.AcquireInstanceFromDir(context.Background(), inst, src)
	re := requireResolutionError(t, err)
	assert.Equal(t, oerrors.ResolutionImportUnprovided, re.Kind)
	assert.Contains(t, err.Error(), "cannot find module providing package test.example/undeclared@v0")
}

// An evaluation error is never a fetch failure, and the shape-gate sentinels
// still match.
func TestFetchClassify_EvaluationErrorStaysPlain(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cue.mod"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cue.mod", "module.cue"), []byte(
		"module: \"fetch.example/inst@v0\"\nlanguage: version: \"v0.17.0\"\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "inst.cue"), []byte(
		"package inst\n\nkind: \"ModuleInstance\"\na: 1\na: 2\n"), 0o644))
	_, err := kernel.New().AcquireInstanceFromDir(context.Background(), dir)
	require.Error(t, err)
	var fe *oerrors.FetchError
	assert.False(t, errors.As(err, &fe), "no *FetchError in %q", err)
	assert.NotErrorIs(t, err, oerrors.ErrTransient)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "inst.cue"), []byte("package inst\n\nkind: \"Module\"\n"), 0o644))
	_, err = kernel.New().AcquireInstanceFromDir(context.Background(), dir)
	assert.ErrorIs(t, err, oerrors.ErrWrongKind)
	assert.False(t, errors.As(err, &fe))
}

func TestFetchClassify_SynthesizeWithUnreachableDependency(t *testing.T) {
	_, mod := publishSynthModule(t, "demo", "0.1.0", "#components: {}\n#config: {}\ndebugValues: {}\n")
	freshCache(t)
	offline := kernel.New(kernel.WithRegistry(registrytest.UnreachableRegistry(t)))
	_, err := offline.SynthesizeInstance(context.Background(), kernel.InstanceInput{Module: mod, Name: "demo", Namespace: "default"})
	fe := requireFetchError(t, err)
	assert.Equal(t, oerrors.FetchUnreachable, fe.Kind)
	assert.ErrorIs(t, err, oerrors.ErrTransient)
}

func TestFetchClassify_ValuesFileWithUnreachableImport(t *testing.T) {
	inst := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(inst, "cue.mod"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(inst, "cue.mod", "module.cue"), []byte(
		"module: \"fetch.example/inst@v0\"\nlanguage: version: \"v0.17.0\"\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(inst, "inst.cue"), []byte("package inst\n\nkind: \"ModuleInstance\"\n"), 0o644))

	freshCache(t)
	k := kernel.New(kernel.WithRegistry(registrytest.UnreachableRegistry(t)))
	src, err := k.LoadSourceFromFile(writeValuesModule(t, "test.example/defaults@v0", "v0.0.1"))
	require.NoError(t, err)
	_, err = k.AcquireInstanceFromDir(context.Background(), inst, src)
	fe := requireFetchError(t, err)
	assert.Equal(t, oerrors.FetchUnreachable, fe.Kind)
	assert.ErrorIs(t, err, oerrors.ErrTransient)
}

// The render module load: a render whose catalog and core must be fetched
// from an unreachable registry is transient, through Render's own wrap.
func TestFetchClassify_RenderWithUnreachableDependency(t *testing.T) {
	k := newRenderKernel(t)
	plat := acquireRenderPlatform(t, k, "platform")
	inst := acquireRenderInstance(t, k, "instance")

	freshCache(t)
	offline := kernel.New(kernel.WithRegistry(registrytest.UnreachableRegistry(t)))
	_, err := offline.Render(context.Background(), kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "render-test"})
	fe := requireFetchError(t, err)
	assert.Equal(t, oerrors.FetchUnreachable, fe.Kind)
	assert.ErrorIs(t, err, oerrors.ErrTransient)
	assert.Contains(t, err.Error(), "building render module: ")
}
