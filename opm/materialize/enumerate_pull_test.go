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
