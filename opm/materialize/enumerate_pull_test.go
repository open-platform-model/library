package materialize

import (
	"context"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/schema"
)

// TestEnumerate_MajorSuffixScopesSelection locks the fourth address-seam
// adaptation (library-core-retarget): a major-suffixed subscription key scopes
// enumeration — and therefore the no-filter default selection — to its major,
// while a major-free key keeps today's whole-repo behaviour. The published
// list models the real catalogs/opm repo: stable v0 tags beside v1/v2
// prereleases. Without scoping, a v2 key's no-filter default would select
// stable v0.6.0 — a core-v0 catalog pulled into a v2 platform.
func TestEnumerate_MajorSuffixScopesSelection(t *testing.T) {
	path := registrytest.UniquePath(t, "cat")
	var fixtures []registrytest.CatalogFixture
	for _, v := range []string{"0.5.2", "0.6.0", "1.0.0-alpha.9", "2.0.0-alpha.1", "2.0.0-alpha.2"} {
		fixtures = append(fixtures, registrytest.CatalogFixture{
			Path: path, Version: v, Body: registrytest.BuildCatalog(path, v),
		})
	}
	registry := registrytest.NewCatalogRegistry(t, fixtures...)
	env := resolverEnv(registry)

	// A @v2 key enumerates only the v2 line; the no-filter default falls back
	// to the highest alpha on the prerelease-only scope. Stable v0.6.0 is
	// never a candidate.
	scoped, err := enumerateVersions(context.Background(), env, path+"@v2")
	require.NoError(t, err)
	assert.Equal(t, []string{"v2.0.0-alpha.1", "v2.0.0-alpha.2"}, scoped, "@v2 key lists only v2 versions")
	survivors, err := filterVersions(scoped, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"v2.0.0-alpha.2"}, survivors, "no-filter default selects the highest v2 alpha")

	// A major-free key keeps whole-repo enumeration and the stable default.
	all, err := enumerateVersions(context.Background(), env, path)
	require.NoError(t, err)
	assert.Equal(t, []string{"v0.5.2", "v0.6.0", "v1.0.0-alpha.9", "v2.0.0-alpha.1", "v2.0.0-alpha.2"}, all,
		"major-free key lists every published version")
	survivors, err = filterVersions(all, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"v0.6.0"}, survivors, "no-filter default keeps selecting the highest stable")
}

// TestSpike_EnumeratePullRealCatalog de-risks the still-open item from
// design.md Research & Decisions: the real c.#Catalog shape (importing
// opmodel.dev/core@v1, pattern-stamped #transformers) pushed to the in-memory
// registry, enumerated, pulled, and read. The earlier spike used a simplified
// stand-in; this exercises the production shape.
func TestSpike_EnumeratePullRealCatalog(t *testing.T) {
	path := registrytest.UniquePath(t, "cat")
	registry := registrytest.NewCatalogRegistry(t,
		registrytest.CatalogFixture{Path: path, Version: "0.1.0", Body: registrytest.BuildCatalog(path, "0.1.0",
			registrytest.TxFixture{Name: "deployment", Resources: []string{"container"}, Traits: []string{"replicas"}})},
		registrytest.CatalogFixture{Path: path, Version: "0.2.0", Body: registrytest.BuildCatalog(path, "0.2.0",
			registrytest.TxFixture{Name: "deployment", Resources: []string{"container"}})},
	)
	env := resolverEnv(registry)

	// Enumerate: both versions, v-prefixed and sorted.
	versions, err := enumerateVersions(context.Background(), env, path)
	require.NoError(t, err)
	assert.Equal(t, []string{"v0.1.0", "v0.2.0"}, versions, "ModuleVersions lists both, v-prefixed and sorted")

	// Pull v0.1.0 and read the real #Catalog shape.
	octx := cuecontext.New()
	val, err := pullCatalog(octx, env, path, "v0.1.0")
	require.NoError(t, err, "pull v0.1.0")

	meta := val.LookupPath(schema.Metadata)
	require.True(t, meta.Exists(), "#Catalog.metadata reachable")
	ver, _ := meta.LookupPath(cue.ParsePath("version")).String()
	assert.Equal(t, "0.1.0", ver, "catalog metadata.version is bare SemVer")

	txs := val.LookupPath(schema.Transformers)
	require.True(t, txs.Exists(), "#transformers map reachable")

	fqn := path + "/transformers/deployment@0.1.0"
	entry := txs.LookupPath(cue.MakePath(cue.Str(fqn)))
	assert.True(t, entry.Exists(), "stamped transformer FQN %q present", fqn)

	// Distinct versions resolve to distinct content: v0.2.0 dropped the trait.
	val2, err := pullCatalog(octx, env, path, "v0.2.0")
	require.NoError(t, err, "pull v0.2.0")
	fqn2 := path + "/transformers/deployment@0.2.0"
	assert.True(t, val2.LookupPath(schema.Transformers).LookupPath(cue.MakePath(cue.Str(fqn2))).Exists(),
		"v0.2.0 transformer FQN %q present", fqn2)
}
