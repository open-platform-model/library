package kernel_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/kernel"
)

// TestRender_InventoryParity is the tripwire that keeps one provider count.
// For every served render platform it reads the platform's contract
// inventory (Contracts(), core's #contracts) and renders the `instance`
// fixture against it, then asserts the render's over-subscription rows are
// exactly the inventory's verdict: the same keys, a routable platform
// exactly when there are no rows, and every row naming the registry keys
// the inventory's ProvidedBy holds for its key. The render may also refuse for an
// unrelated reason (an unresolved demand on a platform that disables a
// catalog); the rows stay decodable on the refusal, so neither the render
// outcome nor Discriminated is asserted.
//
// The platforms are found by glob, so a platform fixture added later joins
// the tripwire; a directory the table below does not classify fails the
// test, so nobody adds one without stating its expected verdict.
func TestRender_InventoryParity(t *testing.T) {
	gateway := renderCatPath + "/resources/gateway@v1"

	// Every served platform, classified: a control carries the rows it
	// renders today; a pinned bad fixture carries the verdict both counts
	// must reach. nil rows means none.
	expected := map[string][]oerrors.OverSubscribedContract{
		"platform":           nil,
		"platform_next":      nil,
		"platform_two":       nil,
		"platform_providers": nil,
		"platform_disabled":  nil,
		"platform_oversubscribed": {
			{Key: gateway, Catalogs: []string{renderPrefix + "/cat2@v0", renderCatPath + "@v0"}},
		},
		// Two majors of one catalog are two providers.
		"platform_two_majors": {
			{Key: gateway, Catalogs: []string{renderCatPath + "@v0", renderCatPath + "@v1"}},
		},
		// A disabled definer does not hide over-subscription.
		"platform_definer_disabled": {
			{Key: gateway, Catalogs: []string{renderPrefix + "/cat2@v0", renderCatPath + "@v1"}},
		},
	}

	dirs, err := filepath.Glob(renderFixtureDir(t, "platform*"))
	require.NoError(t, err)
	var names []string
	for _, d := range dirs {
		if fi, statErr := os.Stat(d); statErr == nil && fi.IsDir() {
			names = append(names, filepath.Base(d))
		}
	}
	require.NotEmpty(t, names, "the glob finds the served platforms")
	for _, name := range names {
		_, ok := expected[name]
		require.True(t, ok, "served platform %s is not classified in this test's table: add its expected over-subscription rows", name)
	}
	require.Len(t, names, len(expected), "every classified platform is served")

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			k := newRenderKernel(t)
			plat := acquireRenderPlatform(t, k, name)
			inst := acquireRenderInstance(t, k, "instance")

			inv, err := plat.Contracts()
			require.NoError(t, err, "the inventory decodes")

			res, rerr := k.Render(context.Background(), kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
			var rows []oerrors.OverSubscribedContract
			if rerr == nil {
				rows = res.Diagnostics.OverSubscribed
			} else {
				var re *kernel.RenderError
				require.True(t, errors.As(rerr, &re),
					"a refusal after the build carries decodable diagnostics, got: %v", rerr)
				rows = re.Diagnostics.OverSubscribed
			}

			rowKeys := make([]string, 0, len(rows))
			for _, r := range rows {
				rowKeys = append(rowKeys, r.Key)
			}
			over := append([]string{}, inv.OverSubscribed...)
			sort.Strings(over)
			assert.Equal(t, over, rowKeys, "the inventory's over-subscribed keys are the render's rows")
			assert.Equal(t, len(rows) == 0, inv.Routable, "the inventory is routable exactly when the render has no rows")
			for _, r := range rows {
				assert.Equal(t, inv.ProvidedBy[r.Key], r.Catalogs,
					"row %s names exactly the registry keys the inventory's providedBy holds", r.Key)
			}

			want := expected[name]
			if len(want) == 0 {
				assert.Empty(t, rows, "no over-subscription rows")
				assert.True(t, inv.Routable)
				return
			}
			assert.Equal(t, want, rows, "the pinned over-subscription rows")
			assert.False(t, inv.Routable, "the pinned platform is not routable")
		})
	}
}
