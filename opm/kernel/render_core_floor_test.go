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
	"github.com/open-platform-model/library/opm/kernel"
)

// single-build-render spec, "An older-core platform is refused before
// staging": the served healthy platform, re-pinned to core 2.0.0-alpha.10
// (before #contracts.providedBy), acquires, but Render refuses it with the
// typed core-floor error before creating a staging directory. Not a
// *RenderError: the build never runs, and no provider count of the kernel's
// own stands in for core's.
func TestRender_OlderCorePlatformRefusedBeforeStaging(t *testing.T) {
	k := newRenderKernel(t)
	dir := t.TempDir()
	copyTree(t, renderFixtureDir(t, "platform"), dir)
	modFile := filepath.Join(dir, "cue.mod", "module.cue")
	data, err := os.ReadFile(modFile)
	require.NoError(t, err)
	require.Contains(t, string(data), `v: "v2.0.0-alpha.12"`)
	require.NoError(t, os.WriteFile(modFile,
		[]byte(strings.Replace(string(data), `v: "v2.0.0-alpha.12"`, `v: "v2.0.0-alpha.10"`, 1)), 0o644))

	plat, err := k.AcquirePlatformFromDir(context.Background(), dir)
	require.NoError(t, err, "acquisition does not read the inventory")
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
	assert.True(t, strings.HasPrefix(err.Error(), "render refused before staging: "), err.Error())

	var rerr *kernel.RenderError
	assert.False(t, errors.As(err, &rerr), "the floor refuses before the build, not through the gate")

	left, err := filepath.Glob(filepath.Join(tmp, "opm-render-*"))
	require.NoError(t, err)
	assert.Empty(t, left, "no staging directory was created")
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
