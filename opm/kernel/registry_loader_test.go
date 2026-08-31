package kernel_test

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

// TestKernel_NoLoadModuleFromRegistryMethod pins the absence of the value-only
// registry wrapper (library-dead-symbol-sweep): AcquireModuleFromRegistry is
// the single kernel entry point for a published module. The method reappearing
// here is a deliberate act, not drift.
func TestKernel_NoLoadModuleFromRegistryMethod(t *testing.T) {
	_, found := reflect.TypeOf(&kernel.Kernel{}).MethodByName("LoadModuleFromRegistry")
	assert.False(t, found, "*kernel.Kernel must not expose LoadModuleFromRegistry; use AcquireModuleFromRegistry")
}
