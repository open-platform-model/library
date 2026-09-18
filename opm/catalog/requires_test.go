package catalog_test

import (
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/catalog"
)

const testModFile = `module: "test.example/catalogs/provider@v1"
language: version: "v0.17.0"
deps: {
	"opmodel.dev/core@v2": v: "v2.0.0-alpha.10"
	"test.example/catalogs/base@v1": v: "v1.2.3"
}
`

// withSource wraps a bare catalog value in a source tree, the way an acquire
// verb stamps one.
func withSource(t *testing.T, src *catalog.Source) *catalog.Catalog {
	t.Helper()
	v := cuecontext.New().CompileString(catalogBody(""))
	require.NoError(t, v.Err())
	c, err := catalog.NewCatalogFromValue(v)
	require.NoError(t, err)
	c.Source = src
	return c
}

// catalog-acquisition spec, "The catalog's committed dependency requirements
// are readable": both requirements of an acquired catalog are readable as a
// path and a version, in both source modes, and the library reports no verdict
// about either.
func TestCatalog_Requires(t *testing.T) {
	want := map[string]string{
		"opmodel.dev/core@v2":           "v2.0.0-alpha.10",
		"test.example/catalogs/base@v1": "v1.2.3",
	}

	t.Run("overlay mode reads the staged module file", func(t *testing.T) {
		root := filepath.Join(string(filepath.Separator), "synthetic", "root")
		c := withSource(t, &catalog.Source{
			Root: root,
			Overlay: map[string][]byte{
				filepath.Join(root, "cue.mod", "module.cue"): []byte(testModFile),
			},
		})

		got, err := c.Requires()
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("on-disk mode reads the committed module file", func(t *testing.T) {
		root := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(root, "cue.mod"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(root, "cue.mod", "module.cue"), []byte(testModFile), 0o644))
		c := withSource(t, &catalog.Source{Root: root})

		got, err := c.Requires()
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})
}

// The return type is the whole point of the requirement: a consumer comparing
// a catalog's committed resolution against a platform's sees two strings and
// never a CUE module type.
func TestCatalog_Requires_ReturnsPlainStrings(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "synthetic", "root")
	c := withSource(t, &catalog.Source{
		Root:    root,
		Overlay: map[string][]byte{filepath.Join(root, "cue.mod", "module.cue"): []byte(testModFile)},
	})

	var got map[string]string
	got, err := c.Requires()
	require.NoError(t, err)
	for path, version := range got {
		assert.NotEmpty(t, path)
		assert.NotEmpty(t, version)
	}
}

func TestCatalog_Requires_Errors(t *testing.T) {
	t.Run("a catalog carrying no source has no committed file to read", func(t *testing.T) {
		c := withSource(t, nil)
		got, err := c.Requires()
		require.Error(t, err)
		assert.Nil(t, got)
	})

	t.Run("an absent module file is reported with its path", func(t *testing.T) {
		root := t.TempDir()
		c := withSource(t, &catalog.Source{Root: root})
		got, err := c.Requires()
		require.Error(t, err)
		assert.Nil(t, got)
		assert.Contains(t, err.Error(), "cue.mod")
	})

	t.Run("an unparseable module file is reported", func(t *testing.T) {
		root := filepath.Join(string(filepath.Separator), "synthetic", "root")
		c := withSource(t, &catalog.Source{
			Root:    root,
			Overlay: map[string][]byte{filepath.Join(root, "cue.mod", "module.cue"): []byte("this is not a module file")},
		})
		got, err := c.Requires()
		require.Error(t, err)
		assert.Nil(t, got)
	})

	t.Run("nil receiver", func(t *testing.T) {
		var c *catalog.Catalog
		got, err := c.Requires()
		require.Error(t, err)
		assert.Nil(t, got)
	})
}
