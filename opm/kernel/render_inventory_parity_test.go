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
// inventory (Contracts(), core's #contracts) and renders its classified
// instance fixture against it, then asserts the render's over-subscription
// rows are exactly the inventory's verdict: the same keys, a routable
// platform exactly when there are no rows and no collision, and every row
// naming the registry keys the inventory's ProvidedBy holds for its key. The render may also refuse for an
// unrelated reason (an unresolved demand on a platform that disables a
// catalog); the rows stay decodable on the refusal, so neither the render
// outcome nor Discriminated is asserted. The render's collision rows equal
// the inventory's CollidingEntries, its decoded routable verdict equals the
// inventory's, and no refusal carries the not-routable catch-all.
//
// The platforms are found by glob, so a platform fixture added later joins
// the tripwire; a directory the table below does not classify fails the
// test, so nobody adds one without stating its expected verdict.
func TestRender_InventoryParity(t *testing.T) {
	gateway := renderCatPath + "/resources/gateway@v1"

	majPrefix := renderPrefix + "/maj"
	backup := majPrefix + "/traits/backup@v1"

	// Every served platform, classified: a control carries the rows it
	// renders today; a pinned bad fixture carries the verdict both counts
	// must reach. nil rows means none. instance is the instance fixture
	// rendered against it ("instance" when empty), and routable is the
	// inventory's expected verdict: a colliding platform carries no
	// over-subscription row and still reads not routable.
	type classified struct {
		instance string
		rows     []oerrors.OverSubscribedContract
		routable bool
	}
	expected := map[string]classified{
		"platform":           {routable: true},
		"platform_next":      {routable: true},
		"platform_two":       {routable: true},
		"platform_providers": {routable: true},
		"platform_disabled":  {routable: true},
		"platform_oversubscribed": {rows: []oerrors.OverSubscribedContract{
			{Key: gateway, Catalogs: []string{renderPrefix + "/cat2@v0", renderCatPath + "@v0"}},
		}},
		// Two majors of one catalog are two providers.
		"platform_two_majors": {rows: []oerrors.OverSubscribedContract{
			{Key: gateway, Catalogs: []string{renderCatPath + "@v0", renderCatPath + "@v1"}},
		}},
		// A disabled definer does not hide over-subscription.
		"platform_definer_disabled": {rows: []oerrors.OverSubscribedContract{
			{Key: gateway, Catalogs: []string{renderPrefix + "/cat2@v0", renderCatPath + "@v1"}},
		}},
		// Two majors listing the same keys collide: not routable with no
		// over-subscription row.
		"platform_collide":          {instance: "instance_maj0"},
		"platform_collide_provider": {instance: "instance_maj0"},
		// A collision and an over-subscription together.
		"platform_collide_oversubscribed": {instance: "instance_maj0", rows: []oerrors.OverSubscribedContract{
			{Key: backup, Catalogs: []string{renderPrefix + "/bprov@v0", renderPrefix + "/bprov@v1"}},
		}},
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
		require.True(t, ok, "served platform %s is not classified in this test's table: add its instance, expected over-subscription rows and routable verdict", name)
	}
	require.Len(t, names, len(expected), "every classified platform is served")

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			k := newRenderKernel(t)
			plat := acquireRenderPlatform(t, k, name)
			want := expected[name]
			instance := want.instance
			if instance == "" {
				instance = "instance"
			}
			inst := acquireRenderInstance(t, k, instance)

			inv, err := plat.Contracts()
			require.NoError(t, err, "the inventory decodes")

			res, rerr := k.Render(context.Background(), kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
			var diag kernel.RenderDiagnostics
			if rerr == nil {
				diag = res.Diagnostics
			} else {
				var re *kernel.RenderError
				require.True(t, errors.As(rerr, &re),
					"a refusal after the build carries decodable diagnostics, got: %v", rerr)
				diag = re.Diagnostics
				var nr *oerrors.NotRoutableError
				assert.False(t, errors.As(rerr, &nr), "no served platform raises the not-routable catch-all")
			}
			rows := diag.OverSubscribed

			// Collision rows equal the inventory's CollidingEntries, and the
			// decoded routable verdict equals the inventory's.
			collided := map[string][]string{}
			for _, c := range diag.Collisions {
				collided[c.Key] = c.Catalogs
			}
			assert.Equal(t, inv.CollidingEntries, collided, "the render's collision rows are the inventory's CollidingEntries")
			assert.Equal(t, inv.Routable, diag.Routable, "the decoded routable diagnostic is the inventory's")

			rowKeys := make([]string, 0, len(rows))
			for _, r := range rows {
				rowKeys = append(rowKeys, r.Key)
			}
			over := append([]string{}, inv.OverSubscribed...)
			sort.Strings(over)
			assert.Equal(t, over, rowKeys, "the inventory's over-subscribed keys are the render's rows")
			assert.Equal(t, len(rows) == 0 && len(inv.Collisions) == 0, inv.Routable,
				"the inventory is routable exactly when the render has no over-subscription row and there is no collision")
			for _, r := range rows {
				assert.Equal(t, inv.ProvidedBy[r.Key], r.Catalogs,
					"row %s names exactly the registry keys the inventory's providedBy holds", r.Key)
			}

			assert.Equal(t, want.routable, inv.Routable, "the pinned routable verdict")
			if len(want.rows) == 0 {
				assert.Empty(t, rows, "no over-subscription rows")
				return
			}
			assert.Equal(t, want.rows, rows, "the pinned over-subscription rows")
		})
	}
}
