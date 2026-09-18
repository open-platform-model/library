package registrytest_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/internal/schematest"
	"github.com/open-platform-model/library/opm/schema"
)

// TestMajor accepts both spellings a fixture writer holds: the bare version
// a served fixture is published at and the v-prefixed core release.
func TestMajor(t *testing.T) {
	for in, want := range map[string]string{
		"0.1.0":           "v0",
		"v0.1.0":          "v0",
		"2.0.0-alpha.10":  "v2",
		"v2.0.0-alpha.10": "v2",
		"v2":              "v2",
	} {
		assert.Equal(t, want, registrytest.Major(in), "Major(%q)", in)
	}
}

// TestDefaultCoreVersion_IsTheDefaultSchemaRelease pins the schema-dispatch
// scenario "Served fixtures pin the default release" in both directions: the
// version every registrytest fixture declares is the release
// schema.DefaultSchemaModule pins, and every render fixture under
// testdata/render declares it in its cue.mod. A fixture pinning another core
// would be served against a default kernel that was never verified on it;
// the platformmodule build canary catches only the default running ahead of
// the fixtures, so the reverse drift is asserted here.
func TestDefaultCoreVersion_IsTheDefaultSchemaRelease(t *testing.T) {
	assert.Equal(t, schema.DefaultSchemaVersion(), registrytest.DefaultCoreVersion,
		"registrytest.DefaultCoreVersion must move with schema.DefaultSchemaModule")

	root := filepath.Join(schematest.LibraryRoot(t), "testdata", "render")
	coreDep := regexp.MustCompile(`(?s)"opmodel\.dev/core@v2":\s*\{\s*v:\s*"([^"]+)"`)
	var seen int
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "module.cue" || filepath.Base(filepath.Dir(path)) != "cue.mod" {
			return nil
		}
		seen++
		src, err := os.ReadFile(path)
		require.NoError(t, err)
		m := coreDep.FindSubmatch(src)
		require.NotNil(t, m, "%s declares no opmodel.dev/core@v2 dependency", path)
		rel, _ := filepath.Rel(root, path)
		assert.Equal(t, registrytest.DefaultCoreVersion, string(m[1]),
			"testdata/render/%s pins a core release the default kernel was not verified against", rel)
		return nil
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, seen, 13, "the render fixture set carries at least thirteen cue.mod files")
}
