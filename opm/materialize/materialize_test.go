package materialize

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/schema"
)

// subKey returns the v2 #registry key for path at version — the
// major-suffixed form Materialize records in Resolved and on
// MaterializeError.Subscription.
func subKey(path, version string) string {
	major, _, _ := strings.Cut(version, ".")
	return path + "@v" + major
}

// 6.1 — happy path: a single enabled subscription. The highest published
// version is pulled, #composedTransformers + #matchers are filled, and the
// resolved version is recorded under the subscription key.
func TestMaterialize_HappyPath(t *testing.T) {
	path := registrytest.UniquePath(t, "cat")
	registry := registrytest.NewCatalogRegistry(t,
		registrytest.CatalogFixture{Path: path, Version: "0.1.0", Body: registrytest.BuildCatalog(path, "0.1.0",
			registrytest.TxFixture{Name: "deployment", Resources: []string{"container"}, Traits: []string{"replicas"}})},
		registrytest.CatalogFixture{Path: path, Version: "0.2.0", Body: registrytest.BuildCatalog(path, "0.2.0",
			registrytest.TxFixture{Name: "deployment", Resources: []string{"container"}, Traits: []string{"replicas"}})},
	)
	octx := cuecontext.New()
	p := registrytest.BuildPlatform(t, octx, fmt.Sprintf(`{ %q: {enable: true, version: "0.2.0"} }`, subKey(path, "0.2.0")))

	mp, err := Materialize(context.Background(), registrytest.NewCtxOwner(octx), registry, p)
	require.NoError(t, err)

	// Highest version (0.2.0) selected.
	assert.Equal(t, "0.2.0", mp.Resolved[subKey(path, "0.2.0")])

	// Composed transformer reachable on the native Transformers surface
	// (implementation FQNs stay build-keyed under v2).
	txFQN := path + "/transformers/deployment@0.2.0"
	composedKeys := composedFQNs(mp.Transformers)
	assert.Equal(t, []string{txFQN}, composedKeys, "Transformers indexes the stamped FQN")

	// Reverse index: resource contract FQN → transformer (contract FQNs are
	// apiVersion-keyed under v2).
	resFQN := path + "/resources/container@" + registrytest.ContractAPIVersion
	ri := mp.Matchers.LookupPath(cue.ParsePath("resources")).LookupPath(cue.MakePath(cue.Str(resFQN)))
	require.True(t, ri.Exists(), "Matchers.resources[%s] present", resFQN)
	n, err := ri.Len().Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(1), n, "one transformer references the resource")

	// Reverse index: trait contract FQN → transformer.
	traitFQN := path + "/traits/replicas@" + registrytest.ContractAPIVersion
	ti := mp.Matchers.LookupPath(cue.ParsePath("traits")).LookupPath(cue.MakePath(cue.Str(traitFQN)))
	assert.True(t, ti.Exists(), "Matchers.traits[%s] present", traitFQN)
}

// TestMaterialize_DoesNotFillClosedPlatform locks the federation seam shut: even
// on a successful materialization that produces a non-empty Transformers map,
// Materialize MUST NOT FillPath #composedTransformers or #matchers onto the
// closed c.#Platform (Source.Package). If a future change reintroduces the
// closed-fill, these assertions fail. (ADR-003 / federate-materialize-transformers.)
func TestMaterialize_DoesNotFillClosedPlatform(t *testing.T) {
	path := registrytest.UniquePath(t, "cat")
	registry := registrytest.NewCatalogRegistry(t,
		registrytest.CatalogFixture{Path: path, Version: "0.1.0", Body: registrytest.BuildCatalog(path, "0.1.0",
			registrytest.TxFixture{Name: "deployment", Resources: []string{"container"}, Traits: []string{"replicas"}})},
	)
	octx := cuecontext.New()
	p := registrytest.BuildPlatform(t, octx, fmt.Sprintf(`{ %q: {enable: true, version: "0.1.0"} }`, subKey(path, "0.1.0")))

	mp, err := Materialize(context.Background(), registrytest.NewCtxOwner(octx), registry, p)
	require.NoError(t, err)

	// Sanity: the native surface IS populated...
	require.NotEmpty(t, composedFQNs(mp.Transformers), "Transformers must carry the composed map")

	// ...while the closed platform spec stays unfilled.
	assert.False(t, mp.Source.Package.LookupPath(schema.ComposedTransformers).Exists(),
		"Source.Package.#composedTransformers must remain unfilled (never FillPath-ed onto the closed platform)")
	assert.False(t, mp.Source.Package.LookupPath(schema.Matchers).Exists(),
		"Source.Package.#matchers must remain unfilled (never FillPath-ed onto the closed platform)")
}

// composedFQNs returns the sorted top-level FQN keys of a native Transformers
// composed map (the federated surface replacing the old
// #composedTransformers-on-Package lookup).
func composedFQNs(composed cue.Value) []string {
	it, err := composed.Fields()
	if err != nil {
		return nil
	}
	var keys []string
	for it.Next() {
		keys = append(keys, it.Selector().Unquoted())
	}
	sort.Strings(keys)
	return keys
}

// NOTE (library-core-retarget): the platform-driven range/allow/deny and
// prerelease-range tests were removed with the v2 retarget — core v2's
// #Subscription carries a required scalar `version` and no `filter`, so those
// platforms are no longer authorable. The Go-side filterVersions semantics
// (which survive until the subscription-collapse slice deletes them) stay
// pinned at the unit level in filter_test.go.

// 6.3a — divergent same-FQN bodies across two catalogs surface as a
// MaterializeError.
func TestMaterialize_DivergentFQNConflicts(t *testing.T) {
	const sharedKey = "shared.example/transformers/shared@1.0.0"
	body := func(path, desc string) string {
		return fmt.Sprintf(`metadata: {modulePath: %q, version: "0.1.0", description: "c"}
#transformers: {
	%q: {
		kind: "ComponentTransformer"
		metadata: {name: "shared", description: %q, fqn: %q}
		#transform: output: {}
	}
}
`, path+"@v0", sharedKey, desc, sharedKey)
	}
	pathA := registrytest.UniquePath(t, "cata")
	pathB := registrytest.UniquePath(t, "catb")
	registry := registrytest.NewCatalogRegistry(t,
		registrytest.CatalogFixture{Path: pathA, Version: "0.1.0", Body: body(pathA, "from A")},
		registrytest.CatalogFixture{Path: pathB, Version: "0.1.0", Body: body(pathB, "from B")},
	)
	octx := cuecontext.New()
	p := registrytest.BuildPlatform(t, octx, fmt.Sprintf(`{ %q: {enable: true, version: "0.1.0"}, %q: {enable: true, version: "0.1.0"} }`,
		subKey(pathA, "0.1.0"), subKey(pathB, "0.1.0")))

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
	p := registrytest.BuildPlatform(t, octx, fmt.Sprintf(`{ %q: {enable: true, version: "0.1.0"} }`, subKey(missing, "0.1.0")))

	_, err := Materialize(context.Background(), registrytest.NewCtxOwner(octx), registry, p)
	require.Error(t, err)
	var me *oerrors.MaterializeError
	require.True(t, errors.As(err, &me), "unresolvable path surfaces as MaterializeError: %v", err)
	assert.Equal(t, oerrors.MaterializeKindCatalog, me.Kind)
	assert.Equal(t, subKey(missing, "0.1.0"), me.Subscription)
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
	p := registrytest.BuildPlatform(t, octx, fmt.Sprintf(`{ %q: {enable: true, version: "0.1.0"}, %q: {enable: false, version: "0.1.0"} }`,
		subKey(enabled, "0.1.0"), subKey(disabled, "0.1.0")))

	mp1, err := Materialize(context.Background(), registrytest.NewCtxOwner(octx), registry, p)
	require.NoError(t, err)

	// Disabled subscription contributes nothing.
	assert.NotContains(t, mp1.Resolved, subKey(disabled, "0.1.0"), "disabled subscription not resolved")
	keys1 := composedFQNs(mp1.Transformers)
	assert.Equal(t, []string{enabled + "/transformers/deployment@0.1.0"}, keys1)
	assert.NotContains(t, keys1, disabled+"/transformers/service@0.1.0")

	// Idempotent: a second call produces the same selection.
	mp2, err := Materialize(context.Background(), registrytest.NewCtxOwner(octx), registry, p)
	require.NoError(t, err)
	assert.Equal(t, mp1.Resolved, mp2.Resolved)
	assert.Equal(t, keys1, composedFQNs(mp2.Transformers))

	// Non-mutating: the source platform is never filled — the federated surfaces
	// live on MaterializedPlatform, not on the closed platform spec.
	assert.Empty(t, mapKeys(p.Package, schema.ComposedTransformers),
		"input platform #composedTransformers must remain empty")
	assert.Empty(t, mapKeys(mp1.Source.Package, schema.ComposedTransformers),
		"Source.Package #composedTransformers must remain unfilled (federation, ADR-003)")
}
