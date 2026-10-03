package kernel_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/platform"
)

// single-build-render spec, "An older-core platform is refused before
// staging": the served healthy platform, re-pinned to core 2.0.0-alpha.10
// (before #contracts.providedBy), acquires, but Render refuses it with the
// typed core-floor error before creating a staging directory. Not a
// *RenderError: the build never runs, and no provider count of the kernel's
// own stands in for core's.
func TestRender_OlderCorePlatformRefusedBeforeStaging(t *testing.T) {
	k := newRenderKernel(t)
	plat := acquireOlderCorePlatform(t, k)
	inst := acquireRenderInstance(t, k, "instance")

	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)

	res, err := k.Render(context.Background(), kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
	require.Error(t, err)
	assert.Nil(t, res)

	var old *oerrors.PlatformCoreTooOldError
	require.True(t, errors.As(err, &old), "the refusal is the typed core-floor error, got: %v", err)
	assert.Equal(t, "render-fixture", old.Platform)
	assert.Equal(t, "providedBy", old.Field)
	assert.Equal(t, "2.0.0-alpha.12", old.Since)
	assert.Equal(t, "2.0.0-alpha.12", old.Require)
	assert.True(t, strings.HasPrefix(err.Error(), "render refused before staging: "), err.Error())

	var rerr *kernel.RenderError
	assert.False(t, errors.As(err, &rerr), "the floor refuses before the build, not through the gate")

	left, err := filepath.Glob(filepath.Join(tmp, "opm-render-*"))
	require.NoError(t, err)
	assert.Empty(t, left, "no staging directory was created")
}

// acquireOlderCorePlatform acquires the served healthy platform re-pinned to
// core 2.0.0-alpha.10, before #contracts.providedBy. Acquisition does not
// read the inventory, so it succeeds.
func acquireOlderCorePlatform(t *testing.T, k *kernel.Kernel) *platform.Platform {
	t.Helper()
	dir := t.TempDir()
	copyTree(t, renderFixtureDir(t, "platform"), dir)
	modFile := filepath.Join(dir, "cue.mod", "module.cue")
	data, err := os.ReadFile(modFile)
	require.NoError(t, err)
	pin := `v: "` + registrytest.DefaultCoreVersion + `"`
	require.Contains(t, string(data), pin)
	require.NoError(t, os.WriteFile(modFile,
		[]byte(strings.Replace(string(data), pin, `v: "v2.0.0-alpha.10"`, 1)), 0o644))

	plat, err := k.AcquirePlatformFromDir(context.Background(), dir)
	require.NoError(t, err, "acquisition does not read the inventory")
	return plat
}

// Render and Contracts() refuse the same unnamed older-core platform with
// the same message: both pass the raw metadata name (empty) and the error
// words it as <unnamed>, whether no metadata was decoded or the name is
// empty. The re-pin target is the floor the kernel enforces.
func TestRender_UnnamedOlderCorePlatformSameMessageAsContracts(t *testing.T) {
	k := newRenderKernel(t)
	acquired := acquireOlderCorePlatform(t, k)
	inst := acquireRenderInstance(t, k, "instance")

	for name, md := range map[string]*platform.PlatformMetadata{
		"no metadata": nil,
		"empty name":  {},
	} {
		t.Run(name, func(t *testing.T) {
			plat := &platform.Platform{Metadata: md, Package: acquired.Package, Source: acquired.Source}

			_, renderErr := k.Render(context.Background(), kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
			var fromRender *oerrors.PlatformCoreTooOldError
			require.True(t, errors.As(renderErr, &fromRender), "got: %v", renderErr)

			_, contractsErr := plat.Contracts()
			var fromContracts *oerrors.PlatformCoreTooOldError
			require.True(t, errors.As(contractsErr, &fromContracts), "got: %v", contractsErr)

			assert.Equal(t, *fromContracts, *fromRender)
			assert.Empty(t, fromRender.Platform)
			assert.Equal(t, fromContracts.Error(), fromRender.Error())
			assert.Equal(t,
				`platform <unnamed> carries no "providedBy" (core derives it from release 2.0.0-alpha.12 on): re-pin opmodel.dev/core in the platform module to v2.0.0-alpha.12 or later`,
				fromRender.Error())
		})
	}
}

// single-build-render spec, "A platform shared by concurrent renders stays
// race-free": Render's core floor reads the platform's Package (a lookup and
// a presence test). One acquired platform and one acquired instance are
// shared by several goroutines rendering on one Kernel; every render passes
// the floor and produces the same objects. Run under -race (task test).
func TestRender_SharedPlatformConcurrentRenders(t *testing.T) {
	k := newRenderKernel(t)
	plat := acquireRenderPlatform(t, k, "platform")
	inst := acquireRenderInstance(t, k, "instance")
	ctx := context.Background()

	first, err := k.Render(ctx, kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
	require.NoError(t, err)
	want := compiledSummary(t, first.Compiled)
	require.NotEmpty(t, want)

	const n = 8
	got := make([]*kernel.RenderResult, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			got[i], errs[i] = k.Render(ctx, kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
		}(i)
	}
	wg.Wait()
	for i := range n {
		require.NoError(t, errs[i], "render %d", i)
		assert.ElementsMatch(t, want, compiledSummary(t, got[i].Compiled), "render %d produces the same objects", i)
	}
}

// single-build-render spec, "A platform shared by concurrent renders stays
// race-free", from the first render: the cold sibling of
// TestRender_SharedPlatformConcurrentRenders. The goroutines render a
// platform and instance that were never rendered before, so they race the
// platform Package's first core-floor lookup, and theirs are the first
// renders of the test. The expected objects come from a render of a
// separately acquired pair after the race; every acquisition builds into its
// own context, so that baseline touches nothing the shared pair holds. One
// kernel serves both pairs because the render kernel's registry environment
// is per test. Run under -race (task test).
func TestRender_SharedPlatformConcurrentRendersCold(t *testing.T) {
	k := newRenderKernel(t)
	ctx := context.Background()

	plat := acquireRenderPlatform(t, k, "platform")
	inst := acquireRenderInstance(t, k, "instance")

	const n = 8
	got := make([]*kernel.RenderResult, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			got[i], errs[i] = k.Render(ctx, kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
		}(i)
	}
	wg.Wait()

	baseline, err := k.Render(ctx, kernel.RenderInput{
		Instance:    acquireRenderInstance(t, k, "instance"),
		Platform:    acquireRenderPlatform(t, k, "platform"),
		RuntimeName: "rt",
	})
	require.NoError(t, err)
	want := compiledSummary(t, baseline.Compiled)
	require.NotEmpty(t, want)
	for i := range n {
		require.NoError(t, errs[i], "render %d", i)
		assert.ElementsMatch(t, want, compiledSummary(t, got[i].Compiled), "render %d produces the same objects", i)
	}
}
