package schematest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// test-fixture-registry spec, "The opmodel.dev namespace resolves through
// the shared tier": both tiers of a private cache link their opmodel.dev
// entry to the shared workspace cache's own directory.
func TestPrivateCacheDir_LinksSharedNamespace(t *testing.T) {
	root := PrivateCacheDir(t)
	shared := WorkspaceCacheDir(t)

	for _, tier := range cacheTiers {
		link := filepath.Join(root, "mod", tier, sharedNamespace)
		fi, err := os.Lstat(link)
		require.NoError(t, err, "%s tier", tier)
		assert.NotZero(t, fi.Mode()&os.ModeSymlink, "%s: the namespace entry is a symlink", link)

		target, err := os.Readlink(link)
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(shared, "mod", tier, sharedNamespace), target)

		resolved, err := filepath.EvalSymlinks(link)
		require.NoError(t, err, "the shared directory exists (created when the shared cache is cold)")
		assert.True(t, isWithin(resolved, shared), "%s resolves inside %s", resolved, shared)
	}
}

// test-fixture-registry spec, "Private cache is cleaned up": the test's own
// cleanup removes the root even after cue/load has left read-only extracted
// directories in it, and the removal stops at the symlinks (the shared tier
// survives).
func TestPrivateCacheDir_CleanupRemovesReadOnlyTree(t *testing.T) {
	var root string
	t.Run("owner", func(t *testing.T) {
		root = PrivateCacheDir(t)
		// The shape cue/load leaves behind: 0555 directories, 0444 files.
		extracted := filepath.Join(root, "mod", "extract", "test.example", "x@v0.1.0")
		modDir := filepath.Join(extracted, "cue.mod")
		require.NoError(t, os.MkdirAll(modDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(modDir, "module.cue"), []byte("module: \"test.example/x@v0\"\n"), 0o444))
		require.NoError(t, os.Chmod(modDir, 0o555))
		require.NoError(t, os.Chmod(extracted, 0o555))
	})

	_, err := os.Stat(root)
	assert.True(t, os.IsNotExist(err), "private cache root %s removed by the owning test's cleanup", root)

	shared := WorkspaceCacheDir(t)
	for _, tier := range cacheTiers {
		_, err := os.Stat(filepath.Join(shared, "mod", tier, sharedNamespace))
		assert.NoError(t, err, "shared %s tier survives the private cache's removal", tier)
	}
}

// isWithin reports whether p is root or a path below it.
func isWithin(p, root string) bool {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
