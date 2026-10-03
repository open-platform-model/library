package loader_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/internal/cueenv"
	"github.com/open-platform-model/library/opm/internal/loader"
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
func TestFetchModule_HappyPathAndTransitiveDeps(t *testing.T) {
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

	val, src, err := loader.FetchModule(
		context.Background(), cuecontext.New(), modPath+"@v0", "v0.0.2",
		cueenv.Override(reg, ""))
	require.NoError(t, err)

	// The staged tree comes back as the artifact Source type, overlay mode.
	require.NotNil(t, src, "staged source is a non-nil *module.Source")
	assert.NotEmpty(t, src.Root, "overlay keys sit under a synthetic root")
	assert.NotEmpty(t, src.Overlay, "the overlay carries the module's files")

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

// The staged overlay carries the fetched module's .cue files only: a license
// or a readme in the archive is not part of what cue/load reads and is not
// staged, while a .cue file in a subdirectory is.
func TestFetchModule_OverlayCarriesCueFilesOnly(t *testing.T) {
	base := registrytest.UniquePath(t, "app")
	modPath := base + "/hello"
	mod := registrytest.ModuleFixture{
		Path: modPath, Version: "0.0.1",
		File: "package hello\nkind: \"Module\"\nmetadata: {name: \"hello\", modulePath: \"" + modPath + "@v0\", version: \"0.0.1\"}\n",
		Extra: map[string]string{
			"LICENSE":        "MIT",
			"docs/README.md": "# hello\n",
			"sub/extra.cue":  "package extra\n",
		},
	}
	reg := registrytest.NewModuleRegistry(t, []registrytest.ModuleFixture{mod}, nil)

	_, src, err := loader.FetchModule(
		context.Background(), cuecontext.New(), modPath+"@v0", "v0.0.1",
		cueenv.Override(reg, ""))
	require.NoError(t, err)
	require.NotNil(t, src)

	var rels []string
	for key := range src.Overlay {
		rel, err := filepath.Rel(src.Root, key)
		require.NoError(t, err, "every overlay key sits under the synthetic root")
		rels = append(rels, filepath.ToSlash(rel))
	}
	sort.Strings(rels)
	assert.Equal(t, []string{"cue.mod/module.cue", "module.cue", "sub/extra.cue"}, rels)
}

// 5.3a — a registry artifact whose kind != "Module" is rejected with an error
// wrapping the SAME ErrWrongKind sentinel directory acquisition wraps, proving the
// shape gate is single-sourced across both loaders.
func TestFetchModule_WrongKind(t *testing.T) {
	base := registrytest.UniquePath(t, "app")
	modPath := base + "/wrong"
	mod := registrytest.ModuleFixture{
		Path: modPath, Version: "0.0.1",
		File: "package wrong\nkind: \"Platform\"\nmetadata: {name: \"x\", modulePath: \"y\", version: \"0.0.1\"}\n",
	}
	reg := registrytest.NewModuleRegistry(t, []registrytest.ModuleFixture{mod}, nil)

	val, _, err := loader.FetchModule(
		context.Background(), cuecontext.New(), modPath+"@v0", "v0.0.1",
		cueenv.Override(reg, ""))
	require.Error(t, err)
	assert.False(t, val.Exists(), "wrong-kind load returns a zero value")
	assert.True(t, errors.Is(err, oerrors.ErrWrongKind), "want ErrWrongKind, got %v", err)
}

// 5.3b — a module missing a required identity field (metadata.modulePath) is
// rejected with an error wrapping the shared ErrMissingRequiredField sentinel.
func TestFetchModule_MissingRequiredField(t *testing.T) {
	base := registrytest.UniquePath(t, "app")
	modPath := base + "/nomp"
	mod := registrytest.ModuleFixture{
		Path: modPath, Version: "0.0.1",
		File: "package nomp\nkind: \"Module\"\nmetadata: {name: \"x\", version: \"0.0.1\"}\n",
	}
	reg := registrytest.NewModuleRegistry(t, []registrytest.ModuleFixture{mod}, nil)

	_, _, err := loader.FetchModule(
		context.Background(), cuecontext.New(), modPath+"@v0", "v0.0.1",
		cueenv.Override(reg, ""))
	require.Error(t, err)
	assert.True(t, errors.Is(err, oerrors.ErrMissingRequiredField), "want ErrMissingRequiredField, got %v", err)
}

// 0010:D11 — a module whose metadata declares a different modulePath than the one
// it was fetched by is rejected with a typed IdentityError naming both values.
func TestFetchModule_IdentityPathMismatch(t *testing.T) {
	base := registrytest.UniquePath(t, "app")
	modPath := base + "/hello"
	otherPath := base + "/other@v0"
	mod := registrytest.ModuleFixture{
		Path: modPath, Version: "0.0.1",
		File: "package hello\nkind: \"Module\"\nmetadata: {name: \"hello\", modulePath: \"" + otherPath + "\", version: \"0.0.1\"}\n",
	}
	reg := registrytest.NewModuleRegistry(t, []registrytest.ModuleFixture{mod}, nil)

	_, _, err := loader.FetchModule(
		context.Background(), cuecontext.New(), modPath+"@v0", "v0.0.1",
		cueenv.Override(reg, ""))
	require.Error(t, err)

	var ie oerrors.IdentityError
	require.True(t, errors.As(err, &ie), "want IdentityError, got %v", err)
	assert.Equal(t, "path", ie.Field)
	assert.Equal(t, otherPath, ie.Declared)
	assert.Equal(t, modPath+"@v0", ie.Fetched)
}

// 0010:D11 — a module declaring the major-free PARENT path (the core-v0/v1 shape,
// once verified by the enhancements/0003 publishing convention) is refused
// like any other disagreement: the schema the library consumes requires the
// major-suffixed form, so there is no convention fallback and the typed error
// carries the declared parent path and the fetched path.
func TestFetchModule_IdentityMajorFreeDeclarationRefused(t *testing.T) {
	base := registrytest.UniquePath(t, "app")
	modPath := base + "/hello"
	mod := registrytest.ModuleFixture{
		Path: modPath, Version: "0.0.1",
		File: "package hello\nkind: \"Module\"\nmetadata: {name: \"hello\", modulePath: \"" + base + "\", version: \"0.0.1\"}\n",
	}
	reg := registrytest.NewModuleRegistry(t, []registrytest.ModuleFixture{mod}, nil)

	_, _, err := loader.FetchModule(
		context.Background(), cuecontext.New(), modPath+"@v0", "v0.0.1",
		cueenv.Override(reg, ""))
	require.Error(t, err, "a major-free declaration cannot equal the fetched path")

	var ie oerrors.IdentityError
	require.True(t, errors.As(err, &ie), "want IdentityError, got %v", err)
	assert.Equal(t, "path", ie.Field)
	assert.Equal(t, base, ie.Declared)
	assert.Equal(t, modPath+"@v0", ie.Fetched)
}

// 0010:D9 — a module whose metadata declares a different version than the tag it
// was fetched by is rejected with a typed IdentityError naming both values
// (the "three published jellyfin artifacts carried one label value" defect).
func TestFetchModule_IdentityVersionMismatch(t *testing.T) {
	base := registrytest.UniquePath(t, "app")
	modPath := base + "/hello"
	mod := registrytest.ModuleFixture{
		Path: modPath, Version: "0.0.1",
		File: "package hello\nkind: \"Module\"\nmetadata: {name: \"hello\", modulePath: \"" + modPath + "@v0\", version: \"9.9.9\"}\n",
	}
	reg := registrytest.NewModuleRegistry(t, []registrytest.ModuleFixture{mod}, nil)

	_, _, err := loader.FetchModule(
		context.Background(), cuecontext.New(), modPath+"@v0", "v0.0.1",
		cueenv.Override(reg, ""))
	require.Error(t, err)

	var ie oerrors.IdentityError
	require.True(t, errors.As(err, &ie), "want IdentityError, got %v", err)
	assert.Equal(t, "version", ie.Field)
	assert.Equal(t, "9.9.9", ie.Declared)
	assert.Equal(t, "0.0.1", ie.Fetched)
}

// 5.4 — an unresolvable path@version surfaces a wrapped fetch/load error
// without mutating inputs or process environment.
func TestFetchModule_Unresolvable(t *testing.T) {
	reg := registrytest.NewModuleRegistry(t, nil, nil)
	envBefore := os.Getenv("CUE_REGISTRY")

	val, _, err := loader.FetchModule(
		context.Background(), cuecontext.New(), "test.example/does/not/exist@v0", "v9.9.9",
		cueenv.Override(reg, ""))
	require.Error(t, err)
	assert.False(t, val.Exists(), "unresolvable load returns a zero value")
	assert.Equal(t, envBefore, os.Getenv("CUE_REGISTRY"))
}

// Invalid caller input (a malformed version) is wrapped, not panicked
// (NewVersion, not MustNewVersion). registry-module-loading spec, "A malformed
// version is still refused": the error is the version parse error, naming the
// version as the caller wrote it (not the canonical "vnot-a-version", which
// the wrapped cue error carries), and not a fetch error.
func TestFetchModule_BadVersionWrapped(t *testing.T) {
	_, _, err := loader.FetchModule(
		context.Background(), cuecontext.New(), "test.example/x@v0", "not-a-version",
		nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing artifact version test.example/x@v0@not-a-version:")
}

// registry-module-loading spec, "A module is acquired by a bare version" and
// "Both spellings stage under the same root": a module published at v0.0.2
// loads by "0.0.2" and by "v0.0.2", passes the identity check both times, and
// both sources share one synthetic root. The bare spelling runs first, so it
// drives the registry fetch rather than a warmed module cache.
func TestFetchModule_BareVersion(t *testing.T) {
	base := registrytest.UniquePath(t, "app")
	modPath := base + "/hello"
	mod := registrytest.ModuleFixture{
		Path: modPath, Version: "0.0.2",
		File: "package hello\nkind: \"Module\"\nmetadata: {name: \"hello\", modulePath: \"" + modPath + "@v0\", version: \"0.0.2\"}\n",
	}
	reg := registrytest.NewModuleRegistry(t, []registrytest.ModuleFixture{mod}, nil)
	env := cueenv.Override(reg, "")

	bareVal, bareSrc, err := loader.FetchModule(context.Background(), cuecontext.New(), modPath+"@v0", "0.0.2", env)
	require.NoError(t, err, "a bare version is accepted")
	prefVal, prefSrc, err := loader.FetchModule(context.Background(), cuecontext.New(), modPath+"@v0", "v0.0.2", env)
	require.NoError(t, err, "a v-prefixed version is accepted")

	assert.Equal(t, "0.0.2", lookupString(t, bareVal, "metadata.version"))
	assert.Equal(t, "0.0.2", lookupString(t, prefVal, "metadata.version"))
	require.NotNil(t, bareSrc)
	require.NotNil(t, prefSrc)
	assert.Equal(t, prefSrc.Root, bareSrc.Root, "both spellings stage under the same synthetic root")
}

// registry-module-loading spec, "A catalog is acquired by a bare version", at
// the loader: the catalog spec goes through the same canonicalisation as a
// module, bare spelling first.
func TestFetchArtifact_CatalogBareVersion(t *testing.T) {
	catPath := registrytest.UniquePath(t, "cat")
	cat := registrytest.CatalogFixture{
		Path: catPath, Version: "1.0.0",
		Body: registrytest.BuildCatalog(catPath, "1.0.0"),
	}
	reg := registrytest.NewCatalogRegistry(t, cat)
	env := cueenv.Override(reg, "")

	bareVal, bareSrc, err := loader.FetchArtifact(context.Background(), cuecontext.New(), catPath+"@v1", "1.0.0", env, loader.CatalogSpec)
	require.NoError(t, err, "a bare version is accepted")
	prefVal, prefSrc, err := loader.FetchArtifact(context.Background(), cuecontext.New(), catPath+"@v1", "v1.0.0", env, loader.CatalogSpec)
	require.NoError(t, err, "a v-prefixed version is accepted")

	assert.Equal(t, "1.0.0", lookupString(t, bareVal, "metadata.version"))
	assert.Equal(t, lookupString(t, prefVal, "metadata.modulePath"), lookupString(t, bareVal, "metadata.modulePath"))
	require.NotNil(t, bareSrc)
	require.NotNil(t, prefSrc)
	assert.Equal(t, prefSrc.Root, bareSrc.Root, "both spellings stage under the same synthetic root")
}
