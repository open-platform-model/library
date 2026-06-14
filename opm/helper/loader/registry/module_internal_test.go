package registry

import (
	"context"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/build"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/mod/modconfig"
	"cuelang.org/go/mod/module"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/internal/registrytest"
)

// 1.2 (footgun pinned) — re-verify in-library, against the library's pinned
// cuelang.org/go version and modregistrytest substrate, the two halves of D2:
//
//   - the Overlay load (FS nil) resolves a catalog-importing module's
//     transitive dependency, and
//   - the obvious-looking alternative — pinning load.Config.FS to the fetched
//     module's SourceLoc.FS — FAILS on that same transitive dependency.
//
// This is the negative result the loader's load-bearing comment cites; the test
// exists so the Overlay approach is never "simplified" to FS-pinning.
func TestOverlayResolvesDepsButFSPinningFails(t *testing.T) {
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

	env := registryEnv(reg)
	resolver, err := modconfig.NewRegistry(&modconfig.Config{Env: env})
	require.NoError(t, err)
	mv, err := module.NewVersion(modPath+"@v0", "v0.0.2")
	require.NoError(t, err)
	loc, err := resolver.Fetch(context.Background(), mv)
	require.NoError(t, err)

	octx := cuecontext.New()

	// Positive: Overlay (FS nil) — the catalog dep resolves.
	synthRoot, overlay, err := overlayFromSource(loc, modPath+"@v0", "v0.0.2")
	require.NoError(t, err)
	overlayInsts := load.Instances([]string{"."}, &load.Config{
		Dir: synthRoot, ModuleRoot: synthRoot, Overlay: overlay, Env: env,
	})
	require.Len(t, overlayInsts, 1)
	require.NoError(t, overlayInsts[0].Err, "Overlay load must resolve the transitive catalog dep")
	require.NoError(t, octx.BuildInstance(overlayInsts[0]).Err())

	// Negative: pinning load.Config.FS to the single fetched module FS — the
	// loader then reads ALL source through that one FS, so the catalog dep
	// (in a separate cache dir) is unreadable.
	fsInsts := load.Instances([]string{"."}, &load.Config{
		FS: loc.FS, Dir: loc.Dir, ModuleRoot: loc.Dir, Env: env,
	})
	fsErr := loadOrBuildErr(octx, fsInsts)
	require.Error(t, fsErr, "FS-pinned load must fail on a transitive dependency")
	// With FS pinned to the single fetched module FS, every import is read
	// through that one FS — so the FIRST dependency outside it is unresolvable.
	// In-library that is opmodel.dev/core@v0 (resolved from the module cache by
	// the Overlay load above, but invisible through the pinned FS); the catalog
	// would fail the same way. The footgun signature is "cannot find package".
	assert.Contains(t, fsErr.Error(), "cannot find package",
		"FS-pinned load fails to resolve dependencies that live outside the pinned FS")
}

// loadOrBuildErr returns the first load error across instances, or the build
// error of the first instance, or nil if both succeed.
func loadOrBuildErr(octx *cue.Context, instances []*build.Instance) error {
	if len(instances) == 0 {
		return errNoInstances
	}
	if instances[0].Err != nil {
		return instances[0].Err
	}
	return octx.BuildInstance(instances[0]).Err()
}

var errNoInstances = errString("load.Instances returned no instances")

type errString string

func (e errString) Error() string { return string(e) }
