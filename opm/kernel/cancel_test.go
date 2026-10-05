package kernel_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/kernel"
)

// kernel-runtime spec, "Kernel verbs check cancellation at entry and between
// stages": every verb is first shown to succeed on its inputs with a live
// context, so the cancelled call fails on the context and on nothing else.

func cancelledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

// assertBareCanceled asserts err is the context's own error, unwrapped.
func assertBareCanceled(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, context.Canceled, err, "the bare context error, not a wrap of it")
}

func TestCancel_DirectoryVerbs(t *testing.T) {
	k := newRenderKernel(t)
	live := context.Background()
	moduleDir := renderFixtureDir(t, "registry", "testing.opmodel.dev_library-render_web_app_v0.1.0")
	catalogDir := renderFixtureDir(t, "registry", "testing.opmodel.dev_library-render_cat_v0.1.0")

	t.Run("AcquireModuleFromDir", func(t *testing.T) {
		_, err := k.AcquireModuleFromDir(live, moduleDir)
		require.NoError(t, err)
		mod, err := k.AcquireModuleFromDir(cancelledContext(), moduleDir)
		assertBareCanceled(t, err)
		assert.Nil(t, mod)
	})
	t.Run("AcquireCatalogFromDir", func(t *testing.T) {
		_, err := k.AcquireCatalogFromDir(live, catalogDir)
		require.NoError(t, err)
		cat, err := k.AcquireCatalogFromDir(cancelledContext(), catalogDir)
		assertBareCanceled(t, err)
		assert.Nil(t, cat)
	})
	t.Run("AcquirePlatformFromDir", func(t *testing.T) {
		_, err := k.AcquirePlatformFromDir(live, renderFixtureDir(t, "platform"))
		require.NoError(t, err)
		plat, err := k.AcquirePlatformFromDir(cancelledContext(), renderFixtureDir(t, "platform"))
		assertBareCanceled(t, err)
		assert.Nil(t, plat)
	})
	t.Run("AcquireInstanceFromDir", func(t *testing.T) {
		_, err := k.AcquireInstanceFromDir(live, renderFixtureDir(t, "instance"))
		require.NoError(t, err)
		inst, err := k.AcquireInstanceFromDir(cancelledContext(), renderFixtureDir(t, "instance"))
		assertBareCanceled(t, err)
		assert.Nil(t, inst)
	})
	t.Run("AcquireInstanceFromDir with values sources", func(t *testing.T) {
		values := mustSource(t, k, "values.cue", `{replicas: 2}`)
		_, err := k.AcquireInstanceFromDir(live, renderFixtureDir(t, "instance"), values)
		require.NoError(t, err)
		inst, err := k.AcquireInstanceFromDir(cancelledContext(), renderFixtureDir(t, "instance"), values)
		assertBareCanceled(t, err)
		assert.Nil(t, inst)
	})
}

func TestCancel_SynthesizeInstance(t *testing.T) {
	k := newRenderKernel(t)
	mod, err := k.AcquireModuleFromRegistry(context.Background(), renderModPath+"@v0", "v0.1.0")
	require.NoError(t, err)
	in := kernel.InstanceInput{
		Module:    mod,
		Name:      "web-synth",
		Namespace: "default",
		Values:    []kernel.Source{mustSource(t, k, "values.cue", `{image: "nginx:1.27", replicas: 3}`)},
	}
	_, err = k.SynthesizeInstance(context.Background(), in)
	require.NoError(t, err)

	inst, err := k.SynthesizeInstance(cancelledContext(), in)
	assertBareCanceled(t, err)
	assert.Nil(t, inst)

	t.Run("argument errors come first", func(t *testing.T) {
		_, err := k.SynthesizeInstance(cancelledContext(), kernel.InstanceInput{Name: "web-synth", Namespace: "default"})
		assert.ErrorIs(t, err, oerrors.ErrMissingModule)
		assert.NotErrorIs(t, err, context.Canceled)
	})
}

// A fetch served from the module cache makes no request, so the check after
// the fetch is what observes the cancellation.
func TestCancel_RegistryFetchServedFromCache(t *testing.T) {
	k := newRenderKernel(t)
	_, err := k.AcquireModuleFromRegistry(context.Background(), renderModPath+"@v0", "v0.1.0")
	require.NoError(t, err, "the warm-up acquire caches the coordinate")
	_, err = k.AcquireCatalogFromRegistry(context.Background(), renderCatPath+"@v0", "v0.1.0")
	require.NoError(t, err, "the warm-up acquire caches the coordinate")

	mod, err := k.AcquireModuleFromRegistry(cancelledContext(), renderModPath+"@v0", "v0.1.0")
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, mod)

	cat, err := k.AcquireCatalogFromRegistry(cancelledContext(), renderCatPath+"@v0", "v0.1.0")
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, cat)
}

func TestCancel_ExpiredDeadline(t *testing.T) {
	k := newRenderKernel(t)
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	plat, err := k.AcquirePlatformFromDir(ctx, renderFixtureDir(t, "platform"))
	require.Error(t, err)
	assert.True(t, errors.Is(err, context.DeadlineExceeded))
	assert.Equal(t, context.DeadlineExceeded, err)
	assert.Nil(t, plat)
}

// countingCtx is a context whose Err reports nil for its first n calls and
// context.Canceled after that, so a test can cancel a verb at each of its
// stage checks in turn. Done never closes: only the verb's own checks see
// the cancellation, which is what the test counts.
type countingCtx struct {
	context.Context
	left int
}

func (c *countingCtx) Err() error {
	if c.left > 0 {
		c.left--
		return nil
	}
	return context.Canceled
}

// assertStageChecks cancels call at its first, second, ... check until it
// succeeds. Every cancelled call must return the bare context.Canceled (the
// artifact's nil-ness is asserted by call), and the number of checks the verb
// made before it succeeded must be want, so dropping any one check fails the
// test.
func assertStageChecks(t *testing.T, want int, call func(ctx context.Context) error) {
	t.Helper()
	for n := 0; n <= want+1; n++ {
		err := call(&countingCtx{Context: context.Background(), left: n})
		if err == nil {
			assert.Equal(t, want, n, "stage checks before the verb succeeded")
			return
		}
		assert.Equal(t, context.Canceled, err, "cancelled at check %d: the bare context error", n+1)
	}
	t.Fatalf("the verb did not succeed after %d checks", want+1)
}

// Each count names the checks the verb makes, in order.
func TestCancel_EveryStageCheck(t *testing.T) {
	k := newRenderKernel(t)
	moduleDir := renderFixtureDir(t, "registry", "testing.opmodel.dev_library-render_web_app_v0.1.0")
	catalogDir := renderFixtureDir(t, "registry", "testing.opmodel.dev_library-render_cat_v0.1.0")
	values := mustSource(t, k, "values.cue", `{replicas: 2}`)

	// entry, after the directory is read, after the package is built.
	t.Run("AcquireModuleFromDir", func(t *testing.T) {
		assertStageChecks(t, 3, func(ctx context.Context) error {
			mod, err := k.AcquireModuleFromDir(ctx, moduleDir)
			if err != nil {
				assert.Nil(t, mod)
			}
			return err
		})
	})
	t.Run("AcquireCatalogFromDir", func(t *testing.T) {
		assertStageChecks(t, 3, func(ctx context.Context) error {
			cat, err := k.AcquireCatalogFromDir(ctx, catalogDir)
			if err != nil {
				assert.Nil(t, cat)
			}
			return err
		})
	})
	t.Run("AcquirePlatformFromDir", func(t *testing.T) {
		assertStageChecks(t, 3, func(ctx context.Context) error {
			plat, err := k.AcquirePlatformFromDir(ctx, renderFixtureDir(t, "platform"))
			if err != nil {
				assert.Nil(t, plat)
			}
			return err
		})
	})
	// entry, after the directory is read, after the package is built (twice:
	// the shared build step and the verb), after the values check.
	t.Run("AcquireInstanceFromDir", func(t *testing.T) {
		assertStageChecks(t, 5, func(ctx context.Context) error {
			inst, err := k.AcquireInstanceFromDir(ctx, renderFixtureDir(t, "instance"))
			if err != nil {
				assert.Nil(t, inst)
			}
			return err
		})
	})
	// entry, after the directory is read, after the values sources are
	// merged, after the package is built, after the values check.
	t.Run("AcquireInstanceFromDir with values sources", func(t *testing.T) {
		assertStageChecks(t, 5, func(ctx context.Context) error {
			inst, err := k.AcquireInstanceFromDir(ctx, renderFixtureDir(t, "instance"), values)
			if err != nil {
				assert.Nil(t, inst)
			}
			return err
		})
	})

	// The registry verbs are served from the warm cache, so the fetch makes
	// no request: after the fetch, after the package is built.
	_, err := k.AcquireModuleFromRegistry(context.Background(), renderModPath+"@v0", "v0.1.0")
	require.NoError(t, err)
	_, err = k.AcquireCatalogFromRegistry(context.Background(), renderCatPath+"@v0", "v0.1.0")
	require.NoError(t, err)
	t.Run("AcquireModuleFromRegistry", func(t *testing.T) {
		assertStageChecks(t, 2, func(ctx context.Context) error {
			mod, err := k.AcquireModuleFromRegistry(ctx, renderModPath+"@v0", "v0.1.0")
			if err != nil {
				assert.Nil(t, mod)
			}
			return err
		})
	})
	t.Run("AcquireCatalogFromRegistry", func(t *testing.T) {
		assertStageChecks(t, 2, func(ctx context.Context) error {
			cat, err := k.AcquireCatalogFromRegistry(ctx, renderCatPath+"@v0", "v0.1.0")
			if err != nil {
				assert.Nil(t, cat)
			}
			return err
		})
	})

	// entry, after the core version resolves, after the values sources are
	// merged, after the package is built, after the values check.
	t.Run("SynthesizeInstance", func(t *testing.T) {
		mod, err := k.AcquireModuleFromRegistry(context.Background(), renderModPath+"@v0", "v0.1.0")
		require.NoError(t, err)
		in := kernel.InstanceInput{
			Module:    mod,
			Name:      "web-synth",
			Namespace: "default",
			Values:    []kernel.Source{mustSource(t, k, "values.cue", `{image: "nginx:1.27", replicas: 3}`)},
		}
		assertStageChecks(t, 5, func(ctx context.Context) error {
			inst, err := k.SynthesizeInstance(ctx, in)
			if err != nil {
				assert.Nil(t, inst)
			}
			return err
		})
	})

	// after the input checks, after staging, after the render build.
	t.Run("Render", func(t *testing.T) {
		in := kernel.RenderInput{
			Instance:    acquireRenderInstance(t, k, "instance"),
			Platform:    acquireRenderPlatform(t, k, "platform"),
			RuntimeName: "render-test",
		}
		assertStageChecks(t, 3, func(ctx context.Context) error {
			res, err := k.Render(ctx, in)
			if err != nil {
				assert.Nil(t, res)
			}
			return err
		})
	})
}
