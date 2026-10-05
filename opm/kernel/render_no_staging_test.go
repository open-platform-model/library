package kernel_test

import (
	"context"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/internal/renderstage"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/module"
)

// TestRender_WritesNoStagingFile pins single-build-render "A render writes no
// staging file": a successful render, each refusal before evaluation and a
// build failure leave the test's private temp dir empty and the render root
// absent from disk.
func TestRender_WritesNoStagingFile(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		k := newRenderKernel(t)
		plat := acquireRenderPlatform(t, k, "platform")
		inst := synthRenderInstance(t, k, "0.1.0")
		root := privateTempDir(t)
		_, err := k.Render(ctx, kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
		require.NoError(t, err)
		assertNoStagingFile(t, root)
	})

	t.Run("older-core refusal", func(t *testing.T) {
		k := newRenderKernel(t)
		plat := acquireOlderCorePlatform(t, k)
		inst := acquireRenderInstance(t, k, "instance")
		root := privateTempDir(t)
		_, err := k.Render(ctx, kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
		var old *oerrors.PlatformCoreTooOldError
		require.ErrorAs(t, err, &old)
		assertNoStagingFile(t, root)
	})

	t.Run("local replacement without the opt-in", func(t *testing.T) {
		k := newRenderKernel(t)
		plat, err := k.AcquirePlatformFromDir(ctx, platformReplacingCatalog(t, catalogWithLabel(t)))
		require.NoError(t, err)
		inst := acquireRenderInstance(t, k, "instance")
		root := privateTempDir(t)
		_, err = k.Render(ctx, kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
		require.ErrorContains(t, err, "did not enable local replacements")
		assertNoStagingFile(t, root)
	})

	t.Run("dependency covered by no version or replacement", func(t *testing.T) {
		k := newRenderKernel(t)
		plat := acquireRenderPlatform(t, k, "platform")
		inst, err := k.AcquireInstanceFromDir(ctx, instanceImportingLib(t, writeLibModule(t)))
		require.NoError(t, err)
		require.NoError(t, os.Remove(filepath.Join(inst.Source.Root, "cue.mod", "local-module.cue")))
		root := privateTempDir(t)
		_, err = k.Render(ctx, kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt", LocalReplacements: true})
		require.ErrorContains(t, err, "carries no version")
		assertNoStagingFile(t, root)
	})

	t.Run("skew refusal", func(t *testing.T) {
		k := newRenderKernel(t)
		plat := acquireRenderPlatform(t, k, "platform")
		inst := synthRenderInstance(t, k, "0.2.0")
		root := privateTempDir(t)
		_, err := k.Render(ctx, kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt", Skew: kernel.SkewRefuse})
		var skew *oerrors.SkewError
		require.ErrorAs(t, err, &skew)
		assertNoStagingFile(t, root)
	})

	t.Run("build failure", func(t *testing.T) {
		k := newRenderKernel(t)
		plat := acquireRenderPlatform(t, k, "platform")
		inst := brokenImportInstance(t, synthRenderInstance(t, k, "0.1.0"))
		root := privateTempDir(t)
		_, err := k.Render(ctx, kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
		require.ErrorContains(t, err, "building render module: ")
		assertNoStagingFile(t, root)
	})
}

// TestRender_BuildErrorPositionsAreDeterministic pins that a render build
// error names the fixed render root, not a per-render temporary directory,
// so two renders of the same failing inputs report the same text.
func TestRender_BuildErrorPositionsAreDeterministic(t *testing.T) {
	k := newRenderKernel(t)
	plat := acquireRenderPlatform(t, k, "platform")
	inst := brokenImportInstance(t, synthRenderInstance(t, k, "0.1.0"))
	in := kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"}

	_, first := k.Render(context.Background(), in)
	require.Error(t, first)
	_, second := k.Render(context.Background(), in)
	require.Error(t, second)

	var rerr *kernel.RenderError
	assert.False(t, errors.As(first, &rerr), "a load failure, not a gate refusal")
	assert.Equal(t, first.Error(), second.Error(), "the two failures are byte-identical")
	assert.Contains(t, first.Error(), renderstage.RenderRoot+string(filepath.Separator), "the position names the render root")
}

// brokenImportInstance returns a copy of an overlay-mode instance whose
// package gains a file importing a module no registry serves and no
// dependency lists, so the render build fails to load. The original is not
// mutated.
func brokenImportInstance(t *testing.T, inst *module.Instance) *module.Instance {
	t.Helper()
	require.NotNil(t, inst.Source)
	require.NotNil(t, inst.Source.Overlay, "an overlay-mode instance")
	src := *inst.Source
	src.Overlay = maps.Clone(inst.Source.Overlay)
	// A synthesized instance's package is named instance.
	src.Overlay[filepath.Join(src.Root, filepath.FromSlash(src.Pkg), "broken.cue")] = []byte("package instance\n\nimport missing \"test.example/missing@v0\"\n\n_broken: missing.X\n")
	broken := *inst
	broken.Source = &src
	return &broken
}
