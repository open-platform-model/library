package cueenv

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"cuelang.org/go/mod/modcache"
	"cuelang.org/go/mod/modconfig"
	"cuelang.org/go/mod/modfile"
	"cuelang.org/go/mod/modregistry"
	"cuelang.org/go/mod/module"
)

// Registry is a Kernel's one registry client: the resolver and the OCI
// transport behind it (a *modregistry.Client), built on first use and shared
// by every operation started from it. It holds no module cache. Each
// operation gets its own ([Registry.Operation]), because CUE's module cache
// keeps every fetch error with the module version it was for, and a cache
// shared for the life of a long-running process would serve one transient
// failure to every later fetch of that version.
//
// The client is built from the registry mapping the Registry was given (an
// empty mapping reads CUE_REGISTRY from the process environment) and the
// credentials configuration, both read when the client is built. A
// construction error is returned to the operation that needed the client and
// is not kept: the next operation tries again. A registry call that fails
// (Fetch, ModFile or ModuleVersions; not a FetchFromCache miss, which is
// normal during resolution) drops the client, so the next operation builds a
// fresh resolver and transport and nothing the transport remembered (it keeps
// a credentials read error per host) outlives that operation.
//
// A Registry is safe for concurrent use.
type Registry struct {
	mapping string

	mu     sync.Mutex
	client *modregistry.Client

	// Test seams, set through SetHooksForTest; nil means the real behaviour.
	newClient func(env []string) (*modregistry.Client, error)
	wrapOp    func(modconfig.CachedRegistry) modconfig.CachedRegistry
}

// NewRegistry returns a Registry for mapping (CUE_REGISTRY syntax; empty
// reads the process CUE_REGISTRY). It builds nothing and reads nothing.
func NewRegistry(mapping string) *Registry {
	return &Registry{mapping: mapping}
}

// NewClient builds a registry client the way modconfig.NewRegistry builds
// the one inside its module cache: a resolver for env (nil reads the process
// environment) and a modregistry client over it. It is the constructor a
// Registry uses unless a test replaces it.
func NewClient(env []string) (*modregistry.Client, error) {
	resolver, err := modconfig.NewResolver(&modconfig.Config{Env: env})
	if err != nil {
		return nil, err
	}
	return modregistry.NewClientWithResolver(resolver), nil
}

// SetHooksForTest replaces the client constructor and wraps each
// operation's registry, so a test can count constructions and calls. Either
// may be nil to keep the real behaviour. It is for tests only and must be
// called before the Registry is used.
func (r *Registry) SetHooksForTest(newClient func(env []string) (*modregistry.Client, error), wrapOp func(modconfig.CachedRegistry) modconfig.CachedRegistry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.newClient = newClient
	r.wrapOp = wrapOp
}

// Operation starts one operation: its environment is read now
// ([Override] of the mapping), and the returned registry is what that
// operation hands to every load and fetch it runs. On its first use it takes
// the shared client, building it if needed, and wraps it in a fresh module
// cache over the cache directory that environment names (CUE_CACHE_DIR, else
// the user cache directory's cue subdirectory, the rule cue itself uses).
func (r *Registry) Operation() *Operation {
	return &Operation{r: r, env: Override(r.mapping, "")}
}

// shared returns the client, building it from env when there is none. A
// construction error is returned and not kept.
func (r *Registry) shared(env []string) (*modregistry.Client, func(modconfig.CachedRegistry) modconfig.CachedRegistry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.client == nil {
		build := r.newClient
		if build == nil {
			build = NewClient
		}
		c, err := build(env)
		if err != nil {
			return nil, nil, err
		}
		r.client = c
	}
	return r.client, r.wrapOp, nil
}

// drop forgets c if it is still the shared client, so the next operation
// builds a new one. A concurrent operation already holding c keeps using it.
func (r *Registry) drop(c *modregistry.Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.client == c {
		r.client = nil
	}
}

// Operation is one kernel operation's registry: a modconfig.CachedRegistry
// over the Registry's shared client and a module cache of its own. Its setup
// runs once, on first use; an error from it is that operation's alone. It is
// safe for concurrent use within the operation.
type Operation struct {
	r   *Registry
	env []string

	once   sync.Once
	client *modregistry.Client
	reg    modconfig.CachedRegistry
	err    error
}

var _ modconfig.CachedRegistry = (*Operation)(nil)

// Env returns the environment slice this operation was started with: nil
// when the Registry has no mapping (the process environment), else a copy of
// the process environment with CUE_REGISTRY replaced. A caller passes it as
// load.Config.Env beside the registry, so one operation reads one
// environment.
func (o *Operation) Env() []string { return o.env }

// Init sets the operation up: it takes the shared client (building it when
// there is none) and opens the module cache. The error, a client
// construction error or an unusable cache directory, is the one
// modconfig.NewRegistry would return for the same environment.
func (o *Operation) Init() error {
	o.once.Do(func() {
		client, wrap, err := o.r.shared(o.env)
		if err != nil {
			o.err = err
			return
		}
		dir, err := cacheDir(o.env)
		if err != nil {
			o.err = err
			return
		}
		cache, err := modcache.New(client, dir)
		if err != nil {
			o.err = err
			return
		}
		o.client = client
		o.reg = cache
		if wrap != nil {
			o.reg = wrap(cache)
		}
	})
	return o.err
}

// failed drops the shared client after a failed registry call.
func (o *Operation) failed(err error) {
	if err != nil {
		o.r.drop(o.client)
	}
}

// ModFile implements modconfig.Registry.
func (o *Operation) ModFile(ctx context.Context, mv module.Version) (*modfile.File, error) {
	if err := o.Init(); err != nil {
		return nil, err
	}
	f, err := o.reg.ModFile(ctx, mv)
	o.failed(err)
	return f, err
}

// Fetch implements modconfig.Registry.
func (o *Operation) Fetch(ctx context.Context, mv module.Version) (module.SourceLoc, error) {
	if err := o.Init(); err != nil {
		return module.SourceLoc{}, err
	}
	loc, err := o.reg.Fetch(ctx, mv)
	o.failed(err)
	return loc, err
}

// ModuleVersions implements modconfig.Registry.
func (o *Operation) ModuleVersions(ctx context.Context, mpath string) ([]string, error) {
	if err := o.Init(); err != nil {
		return nil, err
	}
	versions, err := o.reg.ModuleVersions(ctx, mpath)
	o.failed(err)
	return versions, err
}

// FetchFromCache implements modconfig.CachedRegistry. A miss is an error
// here and is normal during resolution, so no error drops the client.
func (o *Operation) FetchFromCache(mv module.Version) (module.SourceLoc, error) {
	if err := o.Init(); err != nil {
		return module.SourceLoc{}, err
	}
	return o.reg.FetchFromCache(mv)
}

// cacheDir returns the CUE cache directory env names, by the rule
// modconfig.NewRegistry applies: CUE_CACHE_DIR, else the cue subdirectory of
// the user cache directory. A nil env reads the process environment.
func cacheDir(env []string) (string, error) {
	if dir := getenv(env, "CUE_CACHE_DIR"); dir != "" {
		return dir, nil
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine system cache directory: %v", err)
	}
	return filepath.Join(dir, "cue"), nil
}

// getenv reads key from env the way modconfig does: the last entry wins, and
// a nil env is the process environment.
func getenv(env []string, key string) string {
	if env == nil {
		return os.Getenv(key)
	}
	for _, kv := range slices.Backward(env) {
		if v, ok := strings.CutPrefix(kv, key+"="); ok {
			return v
		}
	}
	return ""
}
