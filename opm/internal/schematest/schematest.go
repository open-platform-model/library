// Package schematest is a test-only helper for constructing
// [schema.Cache] instances against the workspace-local CUE module cache
// and for handing tests a CUE module cache directory to build in.
//
// Tests across opm/schema/, opm/helper/synth/, and opm/kernel/ exercise
// the real OCILoader code path; no test-only Loader exists in the
// library. The workspace cache directory (library/.cue-cache/) is
// gitignored. First test run on a fresh checkout fetches
// opmodel.dev/core@v2 from CUE_REGISTRY (default schema.PublicRegistry →
// GHCR); subsequent runs hit the workspace cache.
//
// # Two cache tiers
//
// The module cache a test builds against has two tiers:
//
//   - The shared workspace cache ([WorkspaceCacheDir]) holds the
//     opmodel.dev namespace: opmodel.dev/core and the GHCR catalogs. It is
//     shared by every test and every test process, and nothing in the test
//     tree ever deletes from it: CUE's module cache assumes an extracted
//     directory is immutable while any process can read it, and packages
//     under `go test ./...` run as parallel processes.
//   - A private per-test cache ([PrivateCacheDir]) holds the coordinates a
//     test's in-process registry serves (test.example, testing.opmodel.dev).
//     It starts empty for every served coordinate, so a committed fixture
//     edited under a fixed version is always built from its current bytes,
//     and it is removed when the test ends. Inside it the opmodel.dev
//     subtrees are symlinks into the shared cache, so core is fetched once
//     and reused everywhere through CUE's own lock-protected fetch path.
//
// Which helper a test uses:
//
//   - Tests that serve fixtures through opm/internal/registrytest get the
//     private tier automatically; its constructors call [PrivateCacheDir].
//   - Tests that need only opmodel.dev (schema cache tests, file-loader
//     tests, synth unit tests, the flow and live tests) call [SetEnv] or
//     [NewCache] and build against the shared cache directly.
//
// This package is under opm/internal/ — only opm/* packages may import it.
package schematest

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"cuelang.org/go/mod/modcache"

	"github.com/open-platform-model/library/opm/schema"
)

// sharedNamespace is the module-path namespace a private cache resolves
// through the shared workspace cache rather than privately: the
// opmodel.dev directory of each cache tier, holding core and the
// published catalogs. Served fixture prefixes are sibling top-level
// directories and never touch it.
const sharedNamespace = "opmodel.dev"

// cacheTiers are the two subtrees of a CUE module cache under mod/:
// extracted module trees and download artifacts (zip, mod, lock).
var cacheTiers = [...]string{"extract", "download"}

// LibraryRoot returns the absolute path to the library/ directory
// (the one containing go.mod, opm/, testdata/, …) resolved via
// runtime.Caller relative to this file.
func LibraryRoot(t testing.TB) string {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("schematest: runtime.Caller(0) failed")
	}
	// opm/internal/schematest/schematest.go → library/
	return filepath.Clean(filepath.Join(filepath.Dir(here), "..", "..", ".."))
}

// WorkspaceCacheDir returns the absolute path to library/.cue-cache/ —
// the per-workspace CUE module cache shared by every test and test
// process. The directory is created lazily by [cue/load.Instances] on
// first use; tests do not need to mkdir it. Nothing in the test tree
// deletes from it (see the package doc).
func WorkspaceCacheDir(t testing.TB) string {
	t.Helper()
	return filepath.Join(LibraryRoot(t), ".cue-cache")
}

// PrivateCacheDir returns a CUE module cache directory owned by the
// current test: a fresh temp root holding mod/extract and mod/download,
// each with its opmodel.dev entry symlinked to the same directory
// of [WorkspaceCacheDir] (created empty first when the shared cache is
// cold). Every other coordinate a build under this cache resolves is
// extracted privately, so the cache is empty for a test's served fixtures
// and is removed at test end, read-only extracted directories included.
//
// The caller points CUE_CACHE_DIR at the returned root (registrytest's
// constructors do). Tests that resolve only opmodel.dev use [SetEnv]
// instead.
func PrivateCacheDir(t testing.TB) string {
	t.Helper()
	root := t.TempDir()
	// Registered after t.TempDir so it runs first (LIFO) and TempDir's own
	// removal then finds nothing left; a plain removal fails on the 0555
	// directories cue/load extracts, and modcache's variant restores write
	// permission first.
	t.Cleanup(func() {
		if err := modcache.RemoveAll(root); err != nil {
			t.Errorf("schematest: removing private cache %s: %v", root, err)
		}
	})
	shared := WorkspaceCacheDir(t)
	for _, tier := range cacheTiers {
		target := filepath.Join(shared, "mod", tier, sharedNamespace)
		if err := os.MkdirAll(target, 0o777); err != nil {
			t.Fatalf("schematest: creating shared %s tier: %v", tier, err)
		}
		private := filepath.Join(root, "mod", tier)
		if err := os.MkdirAll(private, 0o777); err != nil {
			t.Fatalf("schematest: creating private %s tier: %v", tier, err)
		}
		if err := os.Symlink(target, filepath.Join(private, sharedNamespace)); err != nil {
			t.Fatalf("schematest: linking %s tier to the shared cache: %v", tier, err)
		}
	}
	return root
}

// SetEnv configures CUE_REGISTRY and CUE_CACHE_DIR for the test scope
// via t.Setenv. Registry defaults to [schema.PublicRegistry]; the cache
// directory is the shared [WorkspaceCacheDir]. The settings revert at
// test scope (t.Cleanup semantics).
func SetEnv(t testing.TB) {
	t.Helper()
	t.Setenv("CUE_REGISTRY", schema.PublicRegistry)
	t.Setenv("CUE_CACHE_DIR", WorkspaceCacheDir(t))
}

// NewCache returns a fresh *schema.Cache backed by a zero-value
// [schema.OCILoader]. It also configures CUE_REGISTRY and CUE_CACHE_DIR
// via [SetEnv] so the loader resolves opmodel.dev/core@v2 against the
// public registry into the workspace-local cache.
//
// Memoization is per-call: distinct tests get distinct caches to keep
// per-test state explicit. Tests that need to share a Cache across
// multiple synth calls within one test should hold the returned pointer
// for the duration of the test.
func NewCache(t testing.TB) *schema.Cache {
	t.Helper()
	SetEnv(t)
	return &schema.Cache{Loader: schema.OCILoader{}}
}
