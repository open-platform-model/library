package platformmodule_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/mod/modfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loaderfile "github.com/open-platform-model/library/opm/helper/loader/file"
	"github.com/open-platform-model/library/opm/helper/platformmodule"
	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/internal/schematest"
	"github.com/open-platform-model/library/opm/kernel"
)

// The render fixture registry (testdata/render/registry) serves two
// catalogs: cat2 imports cat, so cat2's published module file requires cat
// transitively. core resolves from the warm workspace cache.
const (
	fixturePrefix = "testing.opmodel.dev/library-render"
	fixtureCat    = fixturePrefix + "/cat@v0"
	fixtureCat2   = fixturePrefix + "/cat2@v0"
)

// platform-module-generation spec, "End-to-end against the in-process
// registry" and "Registry source carries the caller's configuration": the
// closure derived from the served module files and the files Generate emits
// build through the kernel's shape-gated platform loader with no further
// tidy, the stamped version agrees with the catalog's readout, and the
// catalog's transitive dependency is pinned.
func TestGenerate_BuildsThroughTheKernel(t *testing.T) {
	root := schematest.LibraryRoot(t)
	mapping := registrytest.NewRegistryFromDir(t, filepath.Join(root, "testdata", "render", "registry"), fixturePrefix)
	ctx := context.Background()

	// The registry source is built from explicit configuration: the mapping
	// the test wired, a client type of the caller's choosing, and the
	// environment carrying CUE_CACHE_DIR (the test's private module cache,
	// whose opmodel.dev tier is the shared workspace cache).
	src, err := platformmodule.NewRegistry(platformmodule.RegistryConfig{
		Registry:   mapping,
		ClientType: "library-platformmodule-test",
		Env:        os.Environ(),
	})
	require.NoError(t, err)

	entries := []platformmodule.Entry{{Path: fixtureCat2, Version: "0.1.0", Enable: true}}
	deps, err := platformmodule.Closure(ctx, src, platformmodule.Roots(entries))
	require.NoError(t, err)
	pinned := map[string]string{}
	for _, d := range deps {
		pinned[d.Path] = d.Version
	}
	assert.Equal(t, "v0.1.0", pinned[fixtureCat2])
	assert.Equal(t, "v0.1.0", pinned[fixtureCat], "closure does not pin the catalog's transitive dependency: %v", deps)
	assert.Equal(t, registrytest.DefaultCoreVersion, pinned[platformmodule.CorePath])

	files, err := platformmodule.Generate(platformmodule.Input{
		Name:       "cluster",
		Type:       "kubernetes",
		ModulePath: "opmodel.dev/platforms/cluster@v0",
		Entries:    entries,
		Deps:       deps,
	})
	require.NoError(t, err)
	dir := t.TempDir()
	require.NoError(t, files.WriteTo(dir))

	k := kernel.New(kernel.WithRegistry(mapping))
	plat, err := k.AcquirePlatformFromDir(ctx, dir, loaderfile.LoadOptions{Registry: mapping})
	require.NoError(t, err, "generated module does not build:\n%s\n%s", files[platformmodule.ModuleFileName], files[platformmodule.PlatformFileName])
	require.NotNil(t, plat.Source)
	assert.Equal(t, dir, plat.Source.Root)

	got, err := plat.Package.LookupPath(cue.ParsePath(`#registry."` + fixtureCat2 + `".version`)).String()
	require.NoError(t, err, "reading the entry's derived version")
	assert.Equal(t, "0.1.0", got)

	mf, err := modfile.Parse(files[platformmodule.ModuleFileName], platformmodule.ModuleFileName)
	require.NoError(t, err)
	require.Contains(t, mf.Deps, fixtureCat, "module file does not pin the transitive dependency")
	assert.Equal(t, "v0.1.0", mf.Deps[fixtureCat].Version)
}
