package cueenv_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"cuelang.org/go/mod/modregistry"
	"cuelang.org/go/mod/module"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/internal/cueenv"
	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/internal/schematest"
)

const renderPrefix = "testing.opmodel.dev/library-render"

// renderModule is a module version the committed render fixture registry
// serves.
var renderModule = module.MustNewVersion(renderPrefix+"/web_app@v0", "v0.1.0")

// servedMapping serves the committed render fixture registry for this test
// (with a private, empty module cache) and returns its CUE_REGISTRY mapping.
func servedMapping(t *testing.T) string {
	t.Helper()
	return registrytest.NewRegistryFromDir(t, filepath.Join(schematest.LibraryRoot(t), "testdata", "render", "registry"), renderPrefix)
}

// countingRegistry returns a Registry for mapping whose client constructor
// counts its calls and fails while fail returns true.
func countingRegistry(mapping string, fail func(n int64) bool) (*cueenv.Registry, *atomic.Int64) {
	var n atomic.Int64
	r := cueenv.NewRegistry(mapping)
	r.SetHooksForTest(func(env []string) (*modregistry.Client, error) {
		c := n.Add(1)
		if fail != nil && fail(c) {
			return nil, errors.New("constructor failed on purpose")
		}
		return cueenv.NewClient(env)
	}, nil)
	return r, &n
}

func TestRegistry_NewBuildsNothing(t *testing.T) {
	_, n := countingRegistry("example.com=localhost:5000", nil)
	assert.Zero(t, n.Load(), "NewRegistry constructs no client")
}

func TestRegistry_OneConstructionAcrossOperations(t *testing.T) {
	r, n := countingRegistry("example.com=localhost:5000+insecure", nil)
	for range 5 {
		op := r.Operation()
		require.NoError(t, op.Init())
		require.NoError(t, op.Init(), "Init is idempotent within an operation")
	}
	assert.Equal(t, int64(1), n.Load(), "every operation shares one client")
}

func TestRegistry_ConstructionErrorIsRetried(t *testing.T) {
	r, n := countingRegistry("example.com=localhost:5000+insecure", func(c int64) bool { return c == 1 })
	err := r.Operation().Init()
	require.ErrorContains(t, err, "constructor failed on purpose")
	require.NoError(t, r.Operation().Init(), "the next operation builds the client again")
	assert.Equal(t, int64(2), n.Load())
}

// A real construction error (an invalid mapping) reads as modconfig's.
func TestRegistry_InvalidMappingFailsInit(t *testing.T) {
	err := cueenv.NewRegistry("not a registry mapping !!!").Operation().Init()
	require.ErrorContains(t, err, "bad value for registry")
}

func TestRegistry_ConcurrentFirstUseConstructsOnce(t *testing.T) {
	r, n := countingRegistry("example.com=localhost:5000+insecure", nil)
	const workers = 16
	errs := make([]error, workers)
	var wg sync.WaitGroup
	for i := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs[i] = r.Operation().Init()
		}()
	}
	wg.Wait()
	for i, err := range errs {
		require.NoError(t, err, "worker %d", i)
	}
	assert.Equal(t, int64(1), n.Load())
}

// kernel-runtime "A transient fetch failure is not remembered": the registry
// refuses the first request and serves every later one. The operation that
// saw the refusal fails; the next operation on the same Registry fetches.
func TestRegistry_TransientFailureIsNotRemembered(t *testing.T) {
	mapping := registrytest.RefuseFirst(t, servedMapping(t), renderPrefix, 1)
	r, n := countingRegistry(mapping, nil)
	ctx := context.Background()

	first := r.Operation()
	_, err := first.Fetch(ctx, renderModule)
	require.ErrorContains(t, err, "503")
	_, err = first.Fetch(ctx, renderModule)
	require.Error(t, err, "within one operation the failure stands, as with cue/load's own registry")

	_, err = r.Operation().Fetch(ctx, renderModule)
	require.NoError(t, err, "a later operation fetches again")
	assert.Equal(t, int64(2), n.Load(), "the failed call dropped the client, so it was built again")
}

// kernel-runtime "A cancelled fetch is not remembered".
func TestRegistry_CancelledFetchIsNotRemembered(t *testing.T) {
	r, _ := countingRegistry(servedMapping(t), nil)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := r.Operation().Fetch(cancelled, renderModule)
	require.ErrorIs(t, err, context.Canceled)

	_, err = r.Operation().Fetch(context.Background(), renderModule)
	require.NoError(t, err, "a later operation with a live context fetches")
}

// A cache miss is the normal answer of FetchFromCache during resolution; it
// does not drop the client.
func TestRegistry_CacheMissKeepsTheClient(t *testing.T) {
	r, n := countingRegistry(servedMapping(t), nil)
	_, err := r.Operation().FetchFromCache(renderModule)
	require.ErrorIs(t, err, modregistry.ErrNotFound)
	require.NoError(t, r.Operation().Init())
	assert.Equal(t, int64(1), n.Load())
}

// kernel-runtime "The cache directory is read for each operation": a change
// to CUE_CACHE_DIR between operations reaches the next one.
func TestRegistry_CacheDirIsReadPerOperation(t *testing.T) {
	r, _ := countingRegistry(servedMapping(t), nil)
	ctx := context.Background()

	_, err := r.Operation().Fetch(ctx, renderModule)
	require.NoError(t, err)

	fresh := schematest.IsolatedCacheDir(t)
	t.Setenv("CUE_CACHE_DIR", fresh)
	op := r.Operation()
	loc, err := op.Fetch(ctx, renderModule)
	require.NoError(t, err)
	root, ok := loc.FS.(module.OSRootFS)
	require.True(t, ok)
	rel, err := filepath.Rel(fresh, root.OSRoot())
	require.NoError(t, err)
	assert.True(t, filepath.IsLocal(rel), "the second fetch extracted under the new cache dir, got %s", root.OSRoot())
	entries, err := os.ReadDir(filepath.Join(fresh, "mod"))
	require.NoError(t, err)
	assert.NotEmpty(t, entries)
}

// Env is the slice the operation was started with: nil without a mapping,
// and the mapping's CUE_REGISTRY otherwise.
func TestOperation_Env(t *testing.T) {
	assert.Nil(t, cueenv.NewRegistry("").Operation().Env())
	env := cueenv.NewRegistry("example.com=localhost:5000").Operation().Env()
	var got []string
	for _, kv := range env {
		if v, ok := strings.CutPrefix(kv, "CUE_REGISTRY="); ok {
			got = append(got, v)
		}
	}
	assert.Equal(t, []string{"example.com=localhost:5000"}, got)
}
