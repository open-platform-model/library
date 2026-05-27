// Package schematest is a test-only helper for constructing
// [schema.Cache] instances against the workspace-local CUE module cache.
//
// Tests across opm/schema/, opm/helper/synth/, and opm/kernel/ exercise
// the real OCILoader code path; no test-only Loader exists in the
// library. The workspace cache directory (library/.cue-cache/) is
// gitignored. First test run on a fresh checkout fetches
// opmodel.dev/core@v0 from CUE_REGISTRY (default schema.PublicRegistry →
// GHCR); subsequent runs hit the workspace cache.
//
// This package is under opm/internal/ — only opm/* packages may import it.
package schematest

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/open-platform-model/library/opm/schema"
)

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
// the per-workspace CUE module cache used by tests. The directory is
// created lazily by [cue/load.Instances] on first use; tests do not
// need to mkdir it.
func WorkspaceCacheDir(t testing.TB) string {
	t.Helper()
	return filepath.Join(LibraryRoot(t), ".cue-cache")
}

// SetEnv configures CUE_REGISTRY and CUE_CACHE_DIR for the test scope
// via t.Setenv. Registry defaults to [schema.PublicRegistry]; the cache
// directory is [WorkspaceCacheDir]. The settings revert at test scope
// (t.Cleanup semantics).
func SetEnv(t testing.TB) {
	t.Helper()
	t.Setenv("CUE_REGISTRY", schema.PublicRegistry)
	t.Setenv("CUE_CACHE_DIR", WorkspaceCacheDir(t))
}

// NewCache returns a fresh *schema.Cache backed by a zero-value
// [schema.OCILoader]. It also configures CUE_REGISTRY and CUE_CACHE_DIR
// via [SetEnv] so the loader resolves opmodel.dev/core@v0 against the
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
