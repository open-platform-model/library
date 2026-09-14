package kernel_test

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/kernel"
)

// registry-module-loading spec — Kernel.AcquireModuleFromRegistry returns a
// decoded *module.Module whose staged source (synthetic root + overlay,
// including the module's own cue.mod/module.cue) is populated, so synthesis can
// build inside the module's own root. It also carries the author-set,
// self-referential metadata (the fields that regressed under the wrapper
// approach).
func TestKernel_AcquireModuleFromRegistry(t *testing.T) {
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

	k := kernel.New(kernel.WithRegistry(reg))

	m, err := k.AcquireModuleFromRegistry(context.Background(), modPath+"@v0", "v0.0.2")
	require.NoError(t, err)
	require.NotNil(t, m.Metadata)
	assert.Equal(t, "hello", m.Metadata.Name)
	assert.Equal(t, modPath+"@v0", m.Metadata.ModulePath)
	assert.Equal(t, "0.0.2", m.Metadata.Version)

	// Staged source is populated and reachable.
	require.True(t, m.HasSource(), "acquired module must carry staged source")
	assert.NotEmpty(t, m.Source.Root)
	// The module's own cue.mod/module.cue must be present in the staged overlay
	// (that file carries the tidied closure synth reuses).
	var hasModuleFile bool
	for key := range m.Source.Overlay {
		if filepath.Base(key) == "module.cue" && filepath.Base(filepath.Dir(key)) == "cue.mod" {
			hasModuleFile = true
			break
		}
	}
	assert.True(t, hasModuleFile, "staged overlay must include the module's cue.mod/module.cue")
}

// registry-module-loading spec, "Registry and directory acquisition fail
// identically": both acquire verbs build through the one evaluate-and-shape-
// gate routine, so the same malformed module wraps the same opm/errors
// sentinel whichever way it is acquired, and a well-formed module acquired
// both ways yields the same metadata.
func TestKernel_AcquireModule_RegistryAndDirAgree(t *testing.T) {
	modPath := registrytest.UniquePath(t, "modules") + "/demo"
	moduleFile := func(kind, metadata string) string {
		return "package demo\nkind: \"" + kind + "\"\nmetadata: {\n" + metadata + "}\n"
	}
	cases := map[string]struct {
		file     string
		sentinel error
	}{
		"wrong kind": {
			file:     moduleFile("Platform", "\tname: \"demo\"\n\tmodulePath: \""+modPath+"@v0\"\n\tversion: \"0.1.0\"\n"),
			sentinel: oerrors.ErrWrongKind,
		},
		"missing required field": {
			file:     moduleFile("Module", "\tname: \"demo\"\n\tversion: \"0.1.0\"\n"),
			sentinel: oerrors.ErrMissingRequiredField,
		},
		"well-formed": {
			file: moduleFile("Module", "\tname: \"demo\"\n\tmodulePath: \""+modPath+"@v0\"\n\tversion: \"0.1.0\"\n"),
		},
	}

	// One registry serves every variant under its own version so the
	// coordinates never collide in the test's private module cache. A
	// well-formed module must declare the version it is fetched by (the
	// registry path verifies identity after the gate), so each variant's
	// file is rewritten to its version before it is served and written out.
	var fixtures []registrytest.ModuleFixture
	versions := map[string]string{}
	files := map[string]string{}
	for i, name := range slices.Sorted(maps.Keys(cases)) {
		versions[name] = fmt.Sprintf("0.1.%d", i)
		files[name] = strings.Replace(cases[name].file, "version: \"0.1.0\"", "version: \""+versions[name]+"\"", 1)
		fixtures = append(fixtures, registrytest.ModuleFixture{Path: modPath, Version: versions[name], File: files[name]})
	}
	reg := registrytest.NewModuleRegistry(t, fixtures, nil)
	k := kernel.New(kernel.WithRegistry(reg))
	ctx := context.Background()

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// The dir path verifies no identity; it reads the same bytes the
			// registry serves.
			root := writeTempModuleRoot(t, "", files[name])

			fromDir, dirErr := k.AcquireModuleFromDir(ctx, root)
			fromRegistry, regErr := k.AcquireModuleFromRegistry(ctx, modPath+"@v0", "v"+versions[name])

			if tc.sentinel == nil {
				require.NoError(t, dirErr)
				require.NoError(t, regErr)
				assert.Equal(t, fromRegistry.Metadata, fromDir.Metadata, "the same module, read two ways")
				return
			}
			require.Error(t, dirErr)
			require.Error(t, regErr)
			assert.Nil(t, fromDir)
			assert.Nil(t, fromRegistry)
			assert.True(t, errors.Is(dirErr, tc.sentinel), "directory acquisition: want %v, got %v", tc.sentinel, dirErr)
			assert.True(t, errors.Is(regErr, tc.sentinel), "registry acquisition: want %v, got %v", tc.sentinel, regErr)
		})
	}
}
