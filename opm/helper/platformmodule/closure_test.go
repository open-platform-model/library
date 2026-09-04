package platformmodule

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"cuelang.org/go/mod/modfile"
	"cuelang.org/go/mod/module"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeSource is a fixture module graph: module version string -> the paths
// and versions that module requires. A version absent from the map is an
// unpublished build. It satisfies "Fixture graph without a registry": no
// network, no cache.
type fakeSource struct {
	graph map[string][]Dep
	calls []string
}

func (f *fakeSource) ModFile(_ context.Context, mv module.Version) (*modfile.File, error) {
	f.calls = append(f.calls, mv.String())
	deps, ok := f.graph[mv.String()]
	if !ok {
		return nil, fmt.Errorf("module %s: module not found", mv)
	}
	mf := &modfile.File{Module: mv.Path(), Deps: map[string]*modfile.Dep{}}
	for _, d := range deps {
		mf.Deps[d.Path] = &modfile.Dep{Version: d.Version}
	}
	if err := mf.Init(); err != nil {
		return nil, err
	}
	return mf, nil
}

// fixtureGraph pins core at coreVersion (alpha.7) as the root while both
// catalogs require the older alpha.6; the opm catalog requires
// cue.dev/x/k8s.io transitively.
func fixtureGraph() *fakeSource {
	return &fakeSource{graph: map[string][]Dep{
		"opmodel.dev/core@v2.0.0-alpha.7": nil,
		"opmodel.dev/core@v2.0.0-alpha.6": nil,
		"opmodel.dev/catalogs/opm@v4.0.1": {
			{Path: "cue.dev/x/k8s.io@v0", Version: "v0.10.0"},
			{Path: CorePath, Version: "v2.0.0-alpha.6"},
		},
		"opmodel.dev/catalogs/k8s@v1.0.0-alpha.2": {
			{Path: CorePath, Version: "v2.0.0-alpha.6"},
		},
		"cue.dev/x/k8s.io@v0.10.0": nil,
	}}
}

// fixtureRoots derives roots pinned at the fixture graph's core version, so
// the tests hold whatever release the kernel default advances to.
func fixtureRoots(entries ...Entry) []Dep {
	return Roots(entries, WithCoreVersion(coreVersion))
}

// platform-module-generation spec, "Transitive dependency pinned".
func TestClosure_TransitiveDependencyIsPinned(t *testing.T) {
	src := fixtureGraph()
	got, err := Closure(context.Background(), src, fixtureRoots(
		Entry{Path: opmPath, Version: "4.0.1", Enable: true},
		Entry{Path: k8sPath, Version: "1.0.0-alpha.2", Enable: false},
	))
	require.NoError(t, err)
	// Sorted by path; core resolves to the root's alpha.7 (the maximum over
	// the catalogs' alpha.6 requirement); k8s.io joins from the opm catalog.
	assert.Equal(t, []Dep{
		{Path: "cue.dev/x/k8s.io@v0", Version: "v0.10.0"},
		{Path: k8sPath, Version: "v1.0.0-alpha.2"},
		{Path: opmPath, Version: "v4.0.1"},
		{Path: CorePath, Version: "v2.0.0-alpha.7"},
	}, got)
}

// platform-module-generation spec, "Root wins a shared path" (and its
// converse: a transitive requirement newer than the root raises the pin,
// exactly as `cue mod tidy` would).
func TestClosure_RootsParticipateInTheMaximum(t *testing.T) {
	t.Run("root newer than requirement", func(t *testing.T) {
		got, err := Closure(context.Background(), fixtureGraph(), fixtureRoots(Entry{Path: k8sPath, Version: "1.0.0-alpha.2", Enable: true}))
		require.NoError(t, err)
		assert.Equal(t, []Dep{
			{Path: k8sPath, Version: "v1.0.0-alpha.2"},
			{Path: CorePath, Version: "v2.0.0-alpha.7"},
		}, got)
	})
	t.Run("requirement newer than root", func(t *testing.T) {
		src := fixtureGraph()
		src.graph["opmodel.dev/core@v2.0.0-alpha.8"] = nil
		src.graph["opmodel.dev/catalogs/opm@v4.0.1"] = []Dep{
			{Path: CorePath, Version: "v2.0.0-alpha.8"},
		}
		got, err := Closure(context.Background(), src, fixtureRoots(Entry{Path: opmPath, Version: "4.0.1", Enable: true}))
		require.NoError(t, err)
		assert.Equal(t, []Dep{
			{Path: opmPath, Version: "v4.0.1"},
			{Path: CorePath, Version: "v2.0.0-alpha.8"},
		}, got)
	})
}

// platform-module-generation spec, "Unpublished build refused": the error
// names the module path and version, and no partial list is returned.
func TestClosure_UnpublishedBuildNamesPathAndVersion(t *testing.T) {
	t.Run("root", func(t *testing.T) {
		got, err := Closure(context.Background(), fixtureGraph(), fixtureRoots(Entry{Path: opmPath, Version: "4.9.9", Enable: true}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "opmodel.dev/catalogs/opm@v4.9.9")
		assert.Contains(t, err.Error(), "module not found")
		assert.Nil(t, got)
	})
	t.Run("transitive", func(t *testing.T) {
		src := fixtureGraph()
		delete(src.graph, "cue.dev/x/k8s.io@v0.10.0")
		got, err := Closure(context.Background(), src, fixtureRoots(Entry{Path: opmPath, Version: "4.0.1", Enable: true}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cue.dev/x/k8s.io@v0.10.0")
		assert.Nil(t, got)
	})
}

func TestClosure_EachVersionFetchedOnce(t *testing.T) {
	src := fixtureGraph()
	_, err := Closure(context.Background(), src, fixtureRoots(
		Entry{Path: opmPath, Version: "4.0.1", Enable: true},
		Entry{Path: k8sPath, Version: "1.0.0-alpha.2", Enable: true},
	))
	require.NoError(t, err)
	seen := map[string]int{}
	for _, c := range src.calls {
		seen[c]++
	}
	for mv, n := range seen {
		assert.Equal(t, 1, n, "%s fetched %d times", mv, n)
	}
	// Both catalogs require core alpha.6 and the root names alpha.7: both
	// versions are walked (their requirements count), each once.
	for _, want := range []string{"opmodel.dev/core@v2.0.0-alpha.6", "opmodel.dev/core@v2.0.0-alpha.7"} {
		assert.Equal(t, 1, seen[want], "%s not walked exactly once: %v", want, src.calls)
	}
}

// A local replacement (the "local" path, standing for packages held in
// cue.mod/{gen,pkg,usr}) is not a published dependency and never enters the
// closure.
func TestClosure_SkipsLocalReplacements(t *testing.T) {
	src := fixtureGraph()
	src.graph["opmodel.dev/catalogs/opm@v4.0.1"] = append(src.graph["opmodel.dev/catalogs/opm@v4.0.1"],
		Dep{Path: "local", Version: ""})
	got, err := Closure(context.Background(), src, fixtureRoots(Entry{Path: opmPath, Version: "4.0.1", Enable: true}))
	require.NoError(t, err)
	for _, d := range got {
		assert.NotEqual(t, "local", d.Path)
	}
	assert.Len(t, got, 3)
}

func TestClosure_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Closure(ctx, fixtureGraph(), fixtureRoots())
	assert.True(t, errors.Is(err, context.Canceled), "got %v", err)
}

func TestClosure_NilSource(t *testing.T) {
	_, err := Closure(context.Background(), nil, fixtureRoots())
	require.Error(t, err)
}

func TestClosure_InvalidRootRefused(t *testing.T) {
	_, err := Closure(context.Background(), fixtureGraph(), []Dep{{Path: opmPath, Version: "not-a-version"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dependency root opmodel.dev/catalogs/opm@v4@not-a-version")
}
