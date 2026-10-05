package registrytest_test

import (
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/internal/schematest"
	"github.com/open-platform-model/library/opm/schema"
)

// TestDefaultCoreVersion_IsTheDefaultSchemaRelease pins the schema-dispatch
// scenario "Served fixtures pin the default release": every render fixture
// under testdata/render declares the release schema.DefaultSchemaModule pins
// (registrytest.DefaultCoreVersion is derived from it) in its cue.mod. Those
// trees are pinned as text, so a default move re-pins them by hand or by the
// cascade, and this walk is the guard on that re-pin. A fixture pinning
// another core would be served against a default kernel that was never
// verified on it; the platformmodule build canary catches only the default
// running ahead of the fixtures, so the reverse drift is asserted here.
func TestDefaultCoreVersion_IsTheDefaultSchemaRelease(t *testing.T) {
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
		assert.Equal(t, schema.DefaultSchemaVersion(), string(m[1]),
			"testdata/render/%s pins a core release the default kernel was not verified against", rel)
		return nil
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, seen, 13, "the render fixture set carries at least thirteen cue.mod files")
}

// TestNewStatusRegistry_AnswersEveryRequestWithItsStatus pins the helper the
// fetch-classification tests drive: every path gets the status, with an OCI
// error code where the status has one.
func TestNewStatusRegistry_AnswersEveryRequestWithItsStatus(t *testing.T) {
	for status, code := range map[int]string{401: "UNAUTHORIZED", 403: "DENIED", 404: "NAME_UNKNOWN", 429: "TOOMANYREQUESTS", 503: ""} {
		reg := registrytest.NewStatusRegistry(t, status)
		require.True(t, strings.HasSuffix(reg, "+insecure"), "the mapping is insecure: %s", reg)
		resp, err := http.Get("http://" + strings.TrimSuffix(reg, "+insecure") + "/v2/any/manifests/v0.0.1")
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, resp.Body.Close())
		require.NoError(t, err)
		assert.Equal(t, status, resp.StatusCode)
		if code == "" {
			assert.Empty(t, body, "a %d carries no OCI error body", status)
		} else {
			assert.Contains(t, string(body), `"code":"`+code+`"`)
		}
	}
}

// TestUnreachableRegistry_RefusesTheConnection pins that the returned address
// refuses a connection rather than answering.
func TestUnreachableRegistry_RefusesTheConnection(t *testing.T) {
	reg := registrytest.UnreachableRegistry(t)
	require.True(t, strings.HasSuffix(reg, "+insecure"))
	_, err := net.Dial("tcp", strings.TrimSuffix(reg, "+insecure"))
	require.Error(t, err, "nothing listens on the reserved port")
}
