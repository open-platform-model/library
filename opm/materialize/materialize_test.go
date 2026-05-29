package materialize

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/schema"
)

// 6.1 — happy path: a single enabled subscription, no filter. Highest version
// is pulled, #composedTransformers + #matchers are filled, and the resolved
// version is recorded.
func TestMaterialize_HappyPath(t *testing.T) {
	path := registrytest.UniquePath(t, "cat")
	registry := registrytest.NewCatalogRegistry(t,
		registrytest.CatalogFixture{Path: path, Version: "0.1.0", Body: registrytest.BuildCatalog(path, "0.1.0",
			registrytest.TxFixture{Name: "deployment", Resources: []string{"container"}, Traits: []string{"replicas"}})},
		registrytest.CatalogFixture{Path: path, Version: "0.2.0", Body: registrytest.BuildCatalog(path, "0.2.0",
			registrytest.TxFixture{Name: "deployment", Resources: []string{"container"}, Traits: []string{"replicas"}})},
	)
	octx := cuecontext.New()
	p := registrytest.BuildPlatform(t, octx, fmt.Sprintf(`{ %q: {enable: true} }`, path))

	mp, err := Materialize(context.Background(), registrytest.NewCtxOwner(octx), registry, p)
	require.NoError(t, err)

	// Highest version (0.2.0) selected with no filter.
	assert.Equal(t, "0.2.0", mp.Resolved[path])

	// Composed transformer reachable via the matcher's path constant.
	txFQN := path + "/transformers/deployment@0.2.0"
	composedKeys := mapKeys(mp.Package, schema.ComposedTransformers)
	assert.Equal(t, []string{txFQN}, composedKeys, "#composedTransformers indexes the stamped FQN")

	// Reverse index: resource FQN → transformer.
	resFQN := path + "/resources/container@0.2.0"
	ri := mp.Package.LookupPath(schema.MatchersResources).LookupPath(cue.MakePath(cue.Str(resFQN)))
	require.True(t, ri.Exists(), "#matchers.resources[%s] present", resFQN)
	n, err := ri.Len().Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(1), n, "one transformer references the resource")

	// Reverse index: trait FQN → transformer.
	traitFQN := path + "/traits/replicas@0.2.0"
	ti := mp.Package.LookupPath(schema.MatchersTraits).LookupPath(cue.MakePath(cue.Str(traitFQN)))
	assert.True(t, ti.Exists(), "#matchers.traits[%s] present", traitFQN)
}

// 6.2 — range / allow / deny survivor selection.
func TestMaterialize_RangeAllowDeny(t *testing.T) {
	tests := []struct {
		name       string
		filterBody string
		wantVers   []string // bare versions expected in the composed map
	}{
		{
			name:       "range restricts",
			filterBody: `filter: {range: ">=0.1.0 <0.2.0"}`,
			wantVers:   []string{"0.1.0", "0.1.1"},
		},
		{
			name:       "deny excludes in-range",
			filterBody: `filter: {range: ">=0.1.0 <0.2.0", deny: ["0.1.1"]}`,
			wantVers:   []string{"0.1.0"},
		},
		{
			name:       "allow includes out-of-range",
			filterBody: `filter: {range: ">=0.1.0 <0.2.0", allow: ["0.2.0"]}`,
			wantVers:   []string{"0.1.0", "0.1.1", "0.2.0"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := registrytest.UniquePath(t, "cat")
			var fixtures []registrytest.CatalogFixture
			for _, v := range []string{"0.1.0", "0.1.1", "0.2.0"} {
				fixtures = append(fixtures, registrytest.CatalogFixture{Path: path, Version: v,
					Body: registrytest.BuildCatalog(path, v, registrytest.TxFixture{Name: "deployment", Resources: []string{"container"}})})
			}
			registry := registrytest.NewCatalogRegistry(t, fixtures...)
			octx := cuecontext.New()
			p := registrytest.BuildPlatform(t, octx, fmt.Sprintf(`{ %q: {enable: true, %s} }`, path, tt.filterBody))

			mp, err := Materialize(context.Background(), registrytest.NewCtxOwner(octx), registry, p)
			require.NoError(t, err)

			var want []string
			for _, v := range tt.wantVers {
				want = append(want, path+"/transformers/deployment@"+v)
			}
			assert.ElementsMatch(t, want, mapKeys(mp.Package, schema.ComposedTransformers))
		})
	}
}

// 6.3a — divergent same-FQN bodies across two catalogs surface as a
// MaterializeError.
func TestMaterialize_DivergentFQNConflicts(t *testing.T) {
	const sharedKey = "shared.example/transformers/shared@1.0.0"
	body := func(path, desc string) string {
		return fmt.Sprintf(`metadata: {modulePath: %q, version: "0.1.0", description: "c"}
#transformers: {
	%q: {
		kind: "ComponentTransformer"
		metadata: {name: "shared", description: %q}
		#transform: output: {}
	}
}
`, path, sharedKey, desc)
	}
	pathA := registrytest.UniquePath(t, "cata")
	pathB := registrytest.UniquePath(t, "catb")
	registry := registrytest.NewCatalogRegistry(t,
		registrytest.CatalogFixture{Path: pathA, Version: "0.1.0", Body: body(pathA, "from A")},
		registrytest.CatalogFixture{Path: pathB, Version: "0.1.0", Body: body(pathB, "from B")},
	)
	octx := cuecontext.New()
	p := registrytest.BuildPlatform(t, octx, fmt.Sprintf(`{ %q: {enable: true}, %q: {enable: true} }`, pathA, pathB))

	_, err := Materialize(context.Background(), registrytest.NewCtxOwner(octx), registry, p)
	require.Error(t, err)
	var me *oerrors.MaterializeError
	require.True(t, errors.As(err, &me), "divergence surfaces as MaterializeError: %v", err)
	assert.Equal(t, oerrors.MaterializeKindCatalog, me.Kind)
}

// 6.3b — an unresolvable subscription path surfaces as MaterializeError{catalog}.
func TestMaterialize_UnresolvablePath(t *testing.T) {
	published := registrytest.UniquePath(t, "cat")
	registry := registrytest.NewCatalogRegistry(t,
		registrytest.CatalogFixture{Path: published, Version: "0.1.0", Body: registrytest.BuildCatalog(published, "0.1.0",
			registrytest.TxFixture{Name: "deployment", Resources: []string{"container"}})},
	)
	missing := registrytest.UniquePath(t, "missing")
	octx := cuecontext.New()
	p := registrytest.BuildPlatform(t, octx, fmt.Sprintf(`{ %q: {enable: true} }`, missing))

	_, err := Materialize(context.Background(), registrytest.NewCtxOwner(octx), registry, p)
	require.Error(t, err)
	var me *oerrors.MaterializeError
	require.True(t, errors.As(err, &me), "unresolvable path surfaces as MaterializeError: %v", err)
	assert.Equal(t, oerrors.MaterializeKindCatalog, me.Kind)
	assert.Equal(t, missing, me.Subscription)
}

// 6.4 — enable:false is skipped; Materialize is idempotent and does not mutate
// its input platform.
func TestMaterialize_DisabledIdempotentNonMutating(t *testing.T) {
	enabled := registrytest.UniquePath(t, "on")
	disabled := registrytest.UniquePath(t, "off")
	registry := registrytest.NewCatalogRegistry(t,
		registrytest.CatalogFixture{Path: enabled, Version: "0.1.0", Body: registrytest.BuildCatalog(enabled, "0.1.0",
			registrytest.TxFixture{Name: "deployment", Resources: []string{"container"}})},
		registrytest.CatalogFixture{Path: disabled, Version: "0.1.0", Body: registrytest.BuildCatalog(disabled, "0.1.0",
			registrytest.TxFixture{Name: "service", Resources: []string{"port"}})},
	)
	octx := cuecontext.New()
	p := registrytest.BuildPlatform(t, octx, fmt.Sprintf(`{ %q: {enable: true}, %q: {enable: false} }`, enabled, disabled))

	mp1, err := Materialize(context.Background(), registrytest.NewCtxOwner(octx), registry, p)
	require.NoError(t, err)

	// Disabled subscription contributes nothing.
	assert.NotContains(t, mp1.Resolved, disabled, "disabled subscription not resolved")
	keys1 := mapKeys(mp1.Package, schema.ComposedTransformers)
	assert.Equal(t, []string{enabled + "/transformers/deployment@0.1.0"}, keys1)
	assert.NotContains(t, keys1, disabled+"/transformers/service@0.1.0")

	// Idempotent: a second call produces the same selection.
	mp2, err := Materialize(context.Background(), registrytest.NewCtxOwner(octx), registry, p)
	require.NoError(t, err)
	assert.Equal(t, mp1.Resolved, mp2.Resolved)
	assert.Equal(t, keys1, mapKeys(mp2.Package, schema.ComposedTransformers))

	// Non-mutating: the source platform's slots stay empty.
	assert.Empty(t, mapKeys(p.Package, schema.ComposedTransformers),
		"input platform #composedTransformers must remain empty")
}
