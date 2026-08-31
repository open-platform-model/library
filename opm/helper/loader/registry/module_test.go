package registry_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/open-platform-model/library/opm/errors"
	loaderfile "github.com/open-platform-model/library/opm/helper/loader/file"
	registry "github.com/open-platform-model/library/opm/helper/loader/registry"
	"github.com/open-platform-model/library/opm/internal/registrytest"
)

func lookupString(t *testing.T, v cue.Value, path string) string {
	t.Helper()
	s, err := v.LookupPath(cue.ParsePath(path)).String()
	require.NoError(t, err, "lookup %s", path)
	return s
}

// 5.1 + 5.2 — a published core@v2 module that imports a catalog loads by
// path@version with its author-set, self-referential metadata intact (the
// fields that regressed under the operator's wrapper approach), and its
// transitive catalog dependency resolves through the in-memory Overlay load.
func TestLoadModulePackageWithSource_HappyPathAndTransitiveDeps(t *testing.T) {
	base := registrytest.UniquePath(t, "app")
	catPath := base + "/cat"
	modMetaPath := base + "/modules"
	modPath := modMetaPath + "/hello"

	cat := registrytest.CatalogFixture{
		Path: catPath, Version: "0.1.0",
		Body: registrytest.BuildCatalog(catPath, "0.1.0",
			registrytest.TxFixture{Name: "deployment", Resources: []string{"container"}}),
	}
	mod := registrytest.ModuleFixture{
		Path: modPath, Version: "0.0.2",
		File: registrytest.BuildModuleFile("hello", "hello", modPath+"@v0", catPath+"@v0"),
		Deps: map[string]string{catPath + "@v0": "0.1.0"},
	}
	reg := registrytest.NewModuleRegistry(t, []registrytest.ModuleFixture{mod}, []registrytest.CatalogFixture{cat})

	envBefore := os.Getenv("CUE_REGISTRY")

	res, err := registry.LoadModulePackageWithSource(
		context.Background(), cuecontext.New(), modPath+"@v0", "v0.0.2",
		registry.LoadOptions{Registry: reg})
	require.NoError(t, err)
	val := res.Value

	// Self-referential metadata is preserved (no "field not allowed").
	assert.Equal(t, "hello", lookupString(t, val, "metadata.name"))
	assert.Equal(t, modPath+"@v0", lookupString(t, val, "metadata.modulePath"))
	assert.Equal(t, "0.0.2", lookupString(t, val, "metadata.version"))

	// Transitive catalog dependency resolved through the Overlay load. A v2
	// catalog's metadata.modulePath carries its major suffix.
	assert.Equal(t, catPath+"@v0", lookupString(t, val, "debugValues.catalogModulePath"),
		"module's imported catalog must resolve via the module's own cue.mod/module.cue")

	// The loader does not mutate process environment state (Principle I).
	assert.Equal(t, envBefore, os.Getenv("CUE_REGISTRY"))
}

// 5.3a — a registry artifact whose kind != "Module" is rejected with an error
// wrapping the SAME ErrWrongKind sentinel exposed from loader/file, proving the
// shape gate is single-sourced across both loaders.
func TestLoadModulePackageWithSource_WrongKind(t *testing.T) {
	base := registrytest.UniquePath(t, "app")
	modPath := base + "/wrong"
	mod := registrytest.ModuleFixture{
		Path: modPath, Version: "0.0.1",
		File: "package wrong\nkind: \"Platform\"\nmetadata: {name: \"x\", modulePath: \"y\", version: \"0.0.1\"}\n",
	}
	reg := registrytest.NewModuleRegistry(t, []registrytest.ModuleFixture{mod}, nil)

	res, err := registry.LoadModulePackageWithSource(
		context.Background(), cuecontext.New(), modPath+"@v0", "v0.0.1",
		registry.LoadOptions{Registry: reg})
	require.Error(t, err)
	assert.False(t, res.Value.Exists(), "wrong-kind load returns a zero value")
	assert.True(t, errors.Is(err, loaderfile.ErrWrongKind), "want ErrWrongKind, got %v", err)
}

// 5.3b — a module missing a required identity field (metadata.modulePath) is
// rejected with an error wrapping the shared ErrMissingRequiredField sentinel.
func TestLoadModulePackageWithSource_MissingRequiredField(t *testing.T) {
	base := registrytest.UniquePath(t, "app")
	modPath := base + "/nomp"
	mod := registrytest.ModuleFixture{
		Path: modPath, Version: "0.0.1",
		File: "package nomp\nkind: \"Module\"\nmetadata: {name: \"x\", version: \"0.0.1\"}\n",
	}
	reg := registrytest.NewModuleRegistry(t, []registrytest.ModuleFixture{mod}, nil)

	_, err := registry.LoadModulePackageWithSource(
		context.Background(), cuecontext.New(), modPath+"@v0", "v0.0.1",
		registry.LoadOptions{Registry: reg})
	require.Error(t, err)
	assert.True(t, errors.Is(err, loaderfile.ErrMissingRequiredField), "want ErrMissingRequiredField, got %v", err)
}

// D11 — a module whose metadata declares a different modulePath than the one
// it was fetched by is rejected with a typed IdentityError naming both values.
func TestLoadModulePackageWithSource_IdentityPathMismatch(t *testing.T) {
	base := registrytest.UniquePath(t, "app")
	modPath := base + "/hello"
	otherPath := base + "/other@v0"
	mod := registrytest.ModuleFixture{
		Path: modPath, Version: "0.0.1",
		File: "package hello\nkind: \"Module\"\nmetadata: {name: \"hello\", modulePath: \"" + otherPath + "\", version: \"0.0.1\"}\n",
	}
	reg := registrytest.NewModuleRegistry(t, []registrytest.ModuleFixture{mod}, nil)

	_, err := registry.LoadModulePackageWithSource(
		context.Background(), cuecontext.New(), modPath+"@v0", "v0.0.1",
		registry.LoadOptions{Registry: reg})
	require.Error(t, err)

	var ie oerrors.IdentityError
	require.True(t, errors.As(err, &ie), "want IdentityError, got %v", err)
	assert.Equal(t, "module", ie.Artifact)
	assert.Equal(t, "path", ie.Field)
	assert.Equal(t, otherPath, ie.Declared)
	assert.Equal(t, modPath+"@v0", ie.Fetched)
}

// D11, older-line carve-out — a core-v0/v1-shaped module declares the
// major-free PARENT path (the enhancements/0003 publishing convention; the
// v0/v1 schema cannot express the major-suffixed form). The identity check
// verifies the convention instead of full-path equality, preserving the
// "Self-referential core@v0 metadata is preserved" scenario.
func TestLoadModulePackageWithSource_IdentityMajorFreeParentPath(t *testing.T) {
	base := registrytest.UniquePath(t, "app")
	modPath := base + "/hello"
	mod := registrytest.ModuleFixture{
		Path: modPath, Version: "0.0.1",
		File: "package hello\nkind: \"Module\"\nmetadata: {name: \"hello\", modulePath: \"" + base + "\", version: \"0.0.1\"}\n",
	}
	reg := registrytest.NewModuleRegistry(t, []registrytest.ModuleFixture{mod}, nil)

	_, err := registry.LoadModulePackageWithSource(
		context.Background(), cuecontext.New(), modPath+"@v0", "v0.0.1",
		registry.LoadOptions{Registry: reg})
	require.NoError(t, err, "major-free parent-path declaration must satisfy the identity check")
}

// D11, older-line carve-out — a major-free declaration that is NOT the fetched
// path's parent is still a lie and is refused.
func TestLoadModulePackageWithSource_IdentityMajorFreeParentMismatch(t *testing.T) {
	base := registrytest.UniquePath(t, "app")
	modPath := base + "/hello"
	mod := registrytest.ModuleFixture{
		Path: modPath, Version: "0.0.1",
		File: "package hello\nkind: \"Module\"\nmetadata: {name: \"hello\", modulePath: \"" + base + "/elsewhere\", version: \"0.0.1\"}\n",
	}
	reg := registrytest.NewModuleRegistry(t, []registrytest.ModuleFixture{mod}, nil)

	_, err := registry.LoadModulePackageWithSource(
		context.Background(), cuecontext.New(), modPath+"@v0", "v0.0.1",
		registry.LoadOptions{Registry: reg})
	require.Error(t, err)

	var ie oerrors.IdentityError
	require.True(t, errors.As(err, &ie), "want IdentityError, got %v", err)
	assert.Equal(t, "path", ie.Field)
	assert.Equal(t, base+"/elsewhere", ie.Declared)
	assert.Equal(t, modPath+"@v0", ie.Fetched)
}

// D9 — a module whose metadata declares a different version than the tag it
// was fetched by is rejected with a typed IdentityError naming both values
// (the "three published jellyfin artifacts carried one label value" defect).
func TestLoadModulePackageWithSource_IdentityVersionMismatch(t *testing.T) {
	base := registrytest.UniquePath(t, "app")
	modPath := base + "/hello"
	mod := registrytest.ModuleFixture{
		Path: modPath, Version: "0.0.1",
		File: "package hello\nkind: \"Module\"\nmetadata: {name: \"hello\", modulePath: \"" + modPath + "@v0\", version: \"9.9.9\"}\n",
	}
	reg := registrytest.NewModuleRegistry(t, []registrytest.ModuleFixture{mod}, nil)

	_, err := registry.LoadModulePackageWithSource(
		context.Background(), cuecontext.New(), modPath+"@v0", "v0.0.1",
		registry.LoadOptions{Registry: reg})
	require.Error(t, err)

	var ie oerrors.IdentityError
	require.True(t, errors.As(err, &ie), "want IdentityError, got %v", err)
	assert.Equal(t, "module", ie.Artifact)
	assert.Equal(t, "version", ie.Field)
	assert.Equal(t, "9.9.9", ie.Declared)
	assert.Equal(t, "0.0.1", ie.Fetched)
}

// 5.4 — an unresolvable path@version surfaces a wrapped fetch/load error
// without mutating inputs or process environment.
func TestLoadModulePackageWithSource_Unresolvable(t *testing.T) {
	reg := registrytest.NewModuleRegistry(t, nil, nil)
	envBefore := os.Getenv("CUE_REGISTRY")

	res, err := registry.LoadModulePackageWithSource(
		context.Background(), cuecontext.New(), "test.example/does/not/exist@v0", "v9.9.9",
		registry.LoadOptions{Registry: reg})
	require.Error(t, err)
	assert.False(t, res.Value.Exists(), "unresolvable load returns a zero value")
	assert.Equal(t, envBefore, os.Getenv("CUE_REGISTRY"))
}

// Invalid caller input (a malformed version) is wrapped, not panicked
// (NewVersion, not MustNewVersion).
func TestLoadModulePackageWithSource_BadVersionWrapped(t *testing.T) {
	_, err := registry.LoadModulePackageWithSource(
		context.Background(), cuecontext.New(), "test.example/x@v0", "not-a-version",
		registry.LoadOptions{})
	require.Error(t, err)
}
