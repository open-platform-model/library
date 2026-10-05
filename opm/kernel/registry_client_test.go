package kernel_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"cuelang.org/go/mod/modconfig"
	"cuelang.org/go/mod/modfile"
	"cuelang.org/go/mod/modregistry"
	"cuelang.org/go/mod/module"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/internal/schematest"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/schema"
)

// clientProbe counts constructions of a Kernel's shared registry client and
// the registry calls each operation makes through it.
type clientProbe struct {
	constructions atomic.Int64
	fail          func(n int64) bool // nil: never fail

	mu  sync.Mutex
	ops []*countingRegistry
}

// install puts the probe on k.
func (p *clientProbe) install(k *kernel.Kernel) {
	k.SetRegistryHooksForTest(func(env []string) (*modregistry.Client, error) {
		n := p.constructions.Add(1)
		if p.fail != nil && p.fail(n) {
			return nil, errors.New("registry client construction failed on purpose")
		}
		return kernel.NewRegistryClientForTest(env)
	}, func(reg modconfig.CachedRegistry) modconfig.CachedRegistry {
		c := &countingRegistry{CachedRegistry: reg}
		p.mu.Lock()
		p.ops = append(p.ops, c)
		p.mu.Unlock()
		return c
	})
}

// operations returns the number of operations that used the client and the
// registry calls each made, in the order they first used it.
func (p *clientProbe) operations() []int64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	calls := make([]int64, len(p.ops))
	for i, op := range p.ops {
		calls[i] = op.calls.Load()
	}
	return calls
}

// countingRegistry counts every call one operation makes through its
// registry.
type countingRegistry struct {
	modconfig.CachedRegistry
	calls atomic.Int64
}

func (c *countingRegistry) ModFile(ctx context.Context, mv module.Version) (*modfile.File, error) {
	c.calls.Add(1)
	return c.CachedRegistry.ModFile(ctx, mv)
}

func (c *countingRegistry) Fetch(ctx context.Context, mv module.Version) (module.SourceLoc, error) {
	c.calls.Add(1)
	return c.CachedRegistry.Fetch(ctx, mv)
}

func (c *countingRegistry) ModuleVersions(ctx context.Context, mpath string) ([]string, error) {
	c.calls.Add(1)
	return c.CachedRegistry.ModuleVersions(ctx, mpath)
}

func (c *countingRegistry) FetchFromCache(mv module.Version) (module.SourceLoc, error) {
	c.calls.Add(1)
	return c.CachedRegistry.FetchFromCache(mv)
}

// probedRenderKernel is newRenderKernel with a clientProbe installed.
func probedRenderKernel(t *testing.T, fail func(n int64) bool) (*kernel.Kernel, *clientProbe) {
	t.Helper()
	k := newRenderKernel(t)
	p := &clientProbe{fail: fail}
	p.install(k)
	return k, p
}

// writeProbeValuesFile writes a values file to a directory of its own and returns
// its path, for a file-backed Source.
func writeProbeValuesFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "values.cue")
	require.NoError(t, os.WriteFile(path, []byte("values: {image: \"nginx:1.27\", replicas: 3}\n"), 0o644))
	return path
}

// kernel-runtime "Construction builds no client".
func TestRegistryClient_ConstructionBuildsNone(t *testing.T) {
	k := kernel.New(kernel.WithRegistry("example.com=localhost:5000+insecure"))
	p := &clientProbe{}
	p.install(k)
	assert.Zero(t, p.constructions.Load())
	assert.Empty(t, p.operations())
}

// kernel-runtime "Operations on one Kernel build one client": a registry
// acquire, a platform directory acquire, a synthesis with a file-backed
// values source and a render construct the client once, and the fetch and
// the render build both resolve through it.
func TestRegistryClient_OperationsOnOneKernelBuildOne(t *testing.T) {
	k, p := probedRenderKernel(t, nil)
	ctx := context.Background()

	mod, err := k.AcquireModuleFromRegistry(ctx, renderModPath+"@v0", "v0.1.0")
	require.NoError(t, err)
	afterFetch := p.operations()
	require.Len(t, afterFetch, 1, "the registry acquire is one operation")
	assert.Positive(t, afterFetch[0], "the fetch resolved through the shared client")

	plat, err := k.AcquirePlatformFromDir(ctx, renderFixtureDir(t, "platform"))
	require.NoError(t, err)

	values, err := k.LoadSourceFromFile(writeProbeValuesFile(t))
	require.NoError(t, err)
	inst, err := k.SynthesizeInstance(ctx, kernel.InstanceInput{Module: mod, Name: "web-synth", Namespace: "default", Values: []kernel.Source{values}})
	require.NoError(t, err)

	beforeRender := len(p.operations())
	_, err = k.Render(ctx, kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
	require.NoError(t, err)
	ops := p.operations()
	require.Len(t, ops, beforeRender+1, "the render build is one more operation through the client")
	assert.Positive(t, ops[len(ops)-1], "the render build resolved its dependencies through the shared client")

	assert.Equal(t, int64(1), p.constructions.Load(), "one client for every operation")
}

// kernel-runtime "A failed construction is retried" and "A construction
// failure keeps its wording".
func TestRegistryClient_FailedConstructionIsRetried(t *testing.T) {
	k, p := probedRenderKernel(t, func(n int64) bool { return n == 1 })
	ctx := context.Background()

	_, err := k.AcquireModuleFromRegistry(ctx, renderModPath+"@v0", "v0.1.0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "building module registry resolver: registry client construction failed on purpose")
	var fe *oerrors.FetchError
	assert.False(t, errors.As(err, &fe), "a construction failure is not a fetch failure")

	_, err = k.AcquireModuleFromRegistry(ctx, renderModPath+"@v0", "v0.1.0")
	require.NoError(t, err, "the next operation builds the client again")
	assert.Equal(t, int64(2), p.constructions.Load())
}

// kernel-runtime "A construction failure keeps its wording", with the real
// constructor and a mapping it refuses.
func TestRegistryClient_InvalidMappingKeepsTheWording(t *testing.T) {
	k := kernel.New(kernel.WithRegistry("not a registry mapping !!!"))
	_, err := k.AcquireModuleFromRegistry(context.Background(), renderModPath+"@v0", "v0.1.0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "building module registry resolver: bad value for registry")
	var fe *oerrors.FetchError
	assert.False(t, errors.As(err, &fe))
}

// kernel-runtime "A transient fetch failure is not remembered": the registry
// refuses the first request and serves the rest; the second acquire of the
// same version on the same Kernel succeeds.
func TestRegistryClient_TransientFetchFailureIsNotRemembered(t *testing.T) {
	served := registrytest.NewRegistryFromDir(t, renderFixtureDir(t, "registry"), renderPrefix)
	k := kernel.New(kernel.WithRegistry(registrytest.RefuseFirst(t, served, renderPrefix, 1)))
	ctx := context.Background()

	_, err := k.AcquireModuleFromRegistry(ctx, renderModPath+"@v0", "v0.1.0")
	var fe *oerrors.FetchError
	require.ErrorAs(t, err, &fe, "the refusal is a fetch failure")
	assert.ErrorIs(t, err, oerrors.ErrTransient)

	mod, err := k.AcquireModuleFromRegistry(ctx, renderModPath+"@v0", "v0.1.0")
	require.NoError(t, err, "the same Kernel fetches the version on the next acquire")
	assert.Equal(t, "0.1.0", mod.Metadata.Version)
}

// kernel-runtime "A cancelled fetch is not remembered": the module cache is
// cold, so the first acquire's fetch observes the cancelled context.
func TestRegistryClient_CancelledFetchIsNotRemembered(t *testing.T) {
	k := newRenderKernel(t)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := k.AcquireModuleFromRegistry(cancelled, renderModPath+"@v0", "v0.1.0")
	require.ErrorIs(t, err, context.Canceled)

	_, err = k.AcquireModuleFromRegistry(context.Background(), renderModPath+"@v0", "v0.1.0")
	require.NoError(t, err, "a live acquire on the same Kernel fetches")
}

// kernel-runtime "The cache directory is read for each operation".
func TestRegistryClient_CacheDirIsReadPerOperation(t *testing.T) {
	k := newRenderKernel(t)
	ctx := context.Background()
	_, err := k.AcquireModuleFromRegistry(ctx, renderModPath+"@v0", "v0.1.0")
	require.NoError(t, err)

	fresh := schematest.PrivateCacheDir(t)
	t.Setenv("CUE_CACHE_DIR", fresh)
	_, err = k.AcquireModuleFromRegistry(ctx, renderModPath+"@v0", "v0.1.0")
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(fresh, "mod", "extract", filepath.FromSlash(renderModPath)+"@v0.1.0"))
	require.NoError(t, err, "the second acquire fetched into the cache directory set after the first")
}

// kernel-runtime "Concurrent operations share the client": acquires,
// syntheses and renders on one Kernel at once, the client's first use
// included, construct it once and agree with the sequential run.
func TestRegistryClient_ConcurrentOperationsShareTheClient(t *testing.T) {
	k, p := probedRenderKernel(t, nil)
	ctx := context.Background()

	run := func() (string, error) {
		mod, err := k.AcquireModuleFromRegistry(ctx, renderModPath+"@v0", "v0.1.0")
		if err != nil {
			return "", err
		}
		inst, err := k.SynthesizeInstance(ctx, kernel.InstanceInput{
			Module: mod, Name: "web-synth", Namespace: "default",
			Values: []kernel.Source{mustSource(t, k, "values.cue", `{image: "nginx:1.27", replicas: 3}`)},
		})
		if err != nil {
			return "", err
		}
		plat, err := k.AcquirePlatformFromDir(ctx, renderFixtureDir(t, "platform"))
		if err != nil {
			return "", err
		}
		res, err := k.Render(ctx, kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s %s %d", inst.Metadata.UUID, plat.Metadata.Name, len(res.Compiled)), nil
	}

	const n = 6
	results := make([]string, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i], errs[i] = run()
		}()
	}
	wg.Wait()
	want, err := run()
	require.NoError(t, err)
	for i := range n {
		require.NoError(t, errs[i], "goroutine %d", i)
		assert.Equal(t, want, results[i], "goroutine %d", i)
	}
	assert.Equal(t, int64(1), p.constructions.Load(), "one client across every concurrent operation")
}

// kernel-runtime "The schema loader keeps its own client": a default schema
// cache load makes no call through the Kernel's client.
func TestRegistryClient_SchemaLoaderKeepsItsOwn(t *testing.T) {
	schematest.SetEnv(t)
	k := kernel.New(kernel.WithRegistry(schema.PublicRegistry))
	p := &clientProbe{}
	p.install(k)

	_, err := k.SchemaCache().Get()
	require.NoError(t, err)
	assert.Zero(t, p.constructions.Load(), "the schema load built no kernel client")
	assert.Empty(t, p.operations())
}

// A Kernel not built by New (the zero value) has no shared client; its loads
// let cue/load build its own registry, and no nil registry reaches them.
func TestRegistryClient_ZeroKernelStillRenders(t *testing.T) {
	k := newRenderKernel(t)
	plat := acquireRenderPlatform(t, k, "platform")
	inst := acquireRenderInstance(t, k, "instance")

	var zero kernel.Kernel
	res, err := zero.Render(context.Background(), kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
	require.NoError(t, err, "the zero Kernel renders through cue/load's own registry (CUE_REGISTRY set by the fixture)")
	assert.NotEmpty(t, res.Compiled)
}
