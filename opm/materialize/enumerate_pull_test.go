package materialize

import (
	"context"
	"fmt"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/schema"
)

// TestEnumerate_MajorSuffixScopesDiagnosticList locks enumeration's
// major-scoping in its post-D14 role: enumeration is DIAGNOSTIC-ONLY (it never
// selects — the authored version! does), and the published list shown to the
// user when a named build is missing is scoped to the subscription key's
// major, so a v2 key's diagnostic never mixes in another line's versions. The
// published set models the real catalogs/opm repo: stable v0 tags beside
// v1/v2 prereleases.
func TestEnumerate_MajorSuffixScopesDiagnosticList(t *testing.T) {
	path := registrytest.UniquePath(t, "cat")
	var fixtures []registrytest.CatalogFixture
	for _, v := range []string{"0.5.2", "0.6.0", "1.0.0-alpha.9", "2.0.0-alpha.1", "2.0.0-alpha.2"} {
		fixtures = append(fixtures, registrytest.CatalogFixture{
			Path: path, Version: v, Body: registrytest.BuildCatalog(path, v),
		})
	}
	registry := registrytest.NewCatalogRegistry(t, fixtures...)
	env := resolverEnv(registry)

	// A @v2 key lists only the v2 line — stable v0.6.0 never appears in a v2
	// subscription's diagnostic.
	scoped, err := enumerateVersions(context.Background(), env, path+"@v2")
	require.NoError(t, err)
	assert.Equal(t, []string{"v2.0.0-alpha.1", "v2.0.0-alpha.2"}, scoped, "@v2 key lists only v2 versions")

	// A major-free key (core-v1 form) keeps whole-repo enumeration.
	all, err := enumerateVersions(context.Background(), env, path)
	require.NoError(t, err)
	assert.Equal(t, []string{"v0.5.2", "v0.6.0", "v1.0.0-alpha.9", "v2.0.0-alpha.1", "v2.0.0-alpha.2"}, all,
		"major-free key lists every published version")

	// The wired diagnostic path: a missing named build reports the scoped
	// list; a build present in the list leaves the pull error untouched.
	pullErr := fmt.Errorf("module not found")
	enriched := pullFailureDiagnostic(context.Background(), env, path+"@v2", "2.0.0-alpha.9", pullErr)
	assert.ErrorIs(t, enriched, pullErr, "diagnostic wraps the original pull error")
	assert.Contains(t, enriched.Error(), "not published")
	assert.Contains(t, enriched.Error(), "v2.0.0-alpha.2", "diagnostic carries the major-scoped published list")
	assert.NotContains(t, enriched.Error(), "v0.6.0", "another line's versions never appear")

	same := pullFailureDiagnostic(context.Background(), env, path+"@v2", "2.0.0-alpha.2", pullErr)
	assert.Equal(t, pullErr, same, "a published named build leaves the pull error unenriched")
}

// TestSpike_EnumeratePullRealCatalog de-risks the still-open item from
// design.md Research & Decisions: the real c.#Catalog shape (importing
// the default core line, pattern-stamped #transformers) pushed to the in-memory
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
