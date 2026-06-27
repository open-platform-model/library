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

// 5.1 + 5.2 — a published core@v1 module that imports a catalog loads by
// path@version with its author-set, self-referential metadata intact (the
// fields that regressed under the operator's wrapper approach), and its
// transitive catalog dependency resolves through the in-memory Overlay load.
func TestLoadModulePackage_HappyPathAndTransitiveDeps(t *testing.T) {
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
		File: registrytest.BuildModuleFile("hello", "hello", modMetaPath, catPath+"@v0"),
		Deps: map[string]string{catPath + "@v0": "0.1.0"},
	}
	reg := registrytest.NewModuleRegistry(t, []registrytest.ModuleFixture{mod}, []registrytest.CatalogFixture{cat})

	envBefore := os.Getenv("CUE_REGISTRY")

	val, err := registry.LoadModulePackage(
		context.Background(), cuecontext.New(), modPath+"@v0", "v0.0.2",
		registry.LoadOptions{Registry: reg})
	require.NoError(t, err)

	// Self-referential metadata is preserved (no "field not allowed").
	assert.Equal(t, "hello", lookupString(t, val, "metadata.name"))
	assert.Equal(t, modMetaPath, lookupString(t, val, "metadata.modulePath"))
	assert.Equal(t, "0.0.2", lookupString(t, val, "metadata.version"))

	// Transitive catalog dependency resolved through the Overlay load.
	assert.Equal(t, catPath, lookupString(t, val, "debugValues.catalogModulePath"),
		"module's imported catalog must resolve via the module's own cue.mod/module.cue")

	// The loader does not mutate process environment state (Principle I).
	assert.Equal(t, envBefore, os.Getenv("CUE_REGISTRY"))
}

// 5.3a — a registry artifact whose kind != "Module" is rejected with an error
// wrapping the SAME ErrWrongKind sentinel exposed from loader/file, proving the
// shape gate is single-sourced across both loaders.
func TestLoadModulePackage_WrongKind(t *testing.T) {
	base := registrytest.UniquePath(t, "app")
	modPath := base + "/wrong"
	mod := registrytest.ModuleFixture{
		Path: modPath, Version: "0.0.1",
		File: "package wrong\nkind: \"Platform\"\nmetadata: {name: \"x\", modulePath: \"y\", version: \"0.0.1\"}\n",
	}
	reg := registrytest.NewModuleRegistry(t, []registrytest.ModuleFixture{mod}, nil)

	val, err := registry.LoadModulePackage(
		context.Background(), cuecontext.New(), modPath+"@v0", "v0.0.1",
		registry.LoadOptions{Registry: reg})
	require.Error(t, err)
	assert.False(t, val.Exists(), "wrong-kind load returns a zero value")
	assert.True(t, errors.Is(err, loaderfile.ErrWrongKind), "want ErrWrongKind, got %v", err)
}

// 5.3b — a module missing a required identity field (metadata.modulePath) is
// rejected with an error wrapping the shared ErrMissingRequiredField sentinel.
func TestLoadModulePackage_MissingRequiredField(t *testing.T) {
	base := registrytest.UniquePath(t, "app")
	modPath := base + "/nomp"
	mod := registrytest.ModuleFixture{
		Path: modPath, Version: "0.0.1",
		File: "package nomp\nkind: \"Module\"\nmetadata: {name: \"x\", version: \"0.0.1\"}\n",
	}
	reg := registrytest.NewModuleRegistry(t, []registrytest.ModuleFixture{mod}, nil)

	_, err := registry.LoadModulePackage(
		context.Background(), cuecontext.New(), modPath+"@v0", "v0.0.1",
		registry.LoadOptions{Registry: reg})
	require.Error(t, err)
	assert.True(t, errors.Is(err, loaderfile.ErrMissingRequiredField), "want ErrMissingRequiredField, got %v", err)
}

// 5.4 — an unresolvable path@version surfaces a wrapped fetch/load error
// without mutating inputs or process environment.
func TestLoadModulePackage_Unresolvable(t *testing.T) {
	reg := registrytest.NewModuleRegistry(t, nil, nil)
	envBefore := os.Getenv("CUE_REGISTRY")

	val, err := registry.LoadModulePackage(
		context.Background(), cuecontext.New(), "test.example/does/not/exist@v0", "v9.9.9",
		registry.LoadOptions{Registry: reg})
	require.Error(t, err)
	assert.False(t, val.Exists(), "unresolvable load returns a zero value")
	assert.Equal(t, envBefore, os.Getenv("CUE_REGISTRY"))
}

// Invalid caller input (a malformed version) is wrapped, not panicked
// (NewVersion, not MustNewVersion).
func TestLoadModulePackage_BadVersionWrapped(t *testing.T) {
	_, err := registry.LoadModulePackage(
		context.Background(), cuecontext.New(), "test.example/x@v0", "not-a-version",
		registry.LoadOptions{})
	require.Error(t, err)
}
