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
)

// subKey returns the v2 #registry key for path at version — the
// major-suffixed form Materialize records in Resolved and on
// MaterializeError.Subscription.
func subKey(path, version string) string {
	major, _, _ := strings.Cut(version, ".")
	return path + "@v" + major
}

// 6.1 — happy path: a single enabled subscription. The authored `version!` is
// the selection (D14: the platform file IS the resolution) — a HIGHER version
// is published alongside it and must NOT be chosen. #composedTransformers +
// #matchers are filled and the authored version is recorded under the
// subscription key: the assertion is contract, not coincidence.
func TestMaterialize_HappyPath(t *testing.T) {
	path := registrytest.UniquePath(t, "cat")
	registry := registrytest.NewCatalogRegistry(t,
		registrytest.CatalogFixture{Path: path, Version: "0.1.0", Body: registrytest.BuildCatalog(path, "0.1.0",
			registrytest.TxFixture{Name: "deployment", Resources: []string{"container"}, Traits: []string{"replicas"}})},
		registrytest.CatalogFixture{Path: path, Version: "0.2.0", Body: registrytest.BuildCatalog(path, "0.2.0",
			registrytest.TxFixture{Name: "deployment", Resources: []string{"container"}, Traits: []string{"replicas"}})},
	)
	octx := cuecontext.New()
	p := registrytest.BuildPlatform(t, octx, fmt.Sprintf(`{ %q: {enable: true, version: "0.1.0"} }`, subKey(path, "0.1.0")))

	mp, err := Materialize(context.Background(), registrytest.NewCtxOwner(octx), registry, p)
	require.NoError(t, err)

	// The authored version — not the highest published (0.2.0) — is resolved.
	assert.Equal(t, "0.1.0", mp.Resolved[subKey(path, "0.1.0")])

	// Composed transformer reachable on the native Transformers surface
	// (implementation FQNs stay build-keyed under v2), keyed by the authored
	// build only.
	txFQN := path + "/transformers/deployment@0.1.0"
	composedKeys := composedFQNs(mp.Transformers)
	assert.Equal(t, []string{txFQN}, composedKeys, "Transformers indexes the stamped FQN of the authored build")

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
	assert.False(t, mp.Source.Package.LookupPath(cue.ParsePath("#composedTransformers")).Exists(),
		"Source.Package.#composedTransformers must remain unfilled (never FillPath-ed onto the closed platform)")
	assert.False(t, mp.Source.Package.LookupPath(cue.ParsePath("#matchers")).Exists(),
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

// D14 — an authored version absent from the registry fails materialize, and
// the error is enriched (lazy enumeration) with what IS published in the
// key's major. Previously the float silently selected something else — that
// silent divergence is the defect this change removes.
func TestMaterialize_NamedVersionAbsent(t *testing.T) {
	path := registrytest.UniquePath(t, "cat")
	registry := registrytest.NewCatalogRegistry(t,
		registrytest.CatalogFixture{Path: path, Version: "0.1.0", Body: registrytest.BuildCatalog(path, "0.1.0",
			registrytest.TxFixture{Name: "deployment", Resources: []string{"container"}})},
	)
	octx := cuecontext.New()
	p := registrytest.BuildPlatform(t, octx, fmt.Sprintf(`{ %q: {enable: true, version: "0.2.0"} }`, subKey(path, "0.2.0")))

	_, err := Materialize(context.Background(), registrytest.NewCtxOwner(octx), registry, p)
	require.Error(t, err)
	var me *oerrors.MaterializeError
	require.True(t, errors.As(err, &me), "absent named build surfaces as MaterializeError: %v", err)
	assert.Equal(t, oerrors.MaterializeKindCatalog, me.Kind)
	assert.Equal(t, subKey(path, "0.2.0"), me.Subscription)
	assert.Equal(t, "0.2.0", me.Version)
	assert.Contains(t, err.Error(), "not published", "error names the missing build")
	assert.Contains(t, err.Error(), "v0.1.0", "error carries the published list")
}

// D14 — the named build must sit in the subscription key's major; a
// disagreement fails before any registry I/O.
func TestMaterialize_MajorDisagreement(t *testing.T) {
	path := registrytest.UniquePath(t, "cat")
	registry := registrytest.NewCatalogRegistry(t,
		registrytest.CatalogFixture{Path: path, Version: "0.1.0", Body: registrytest.BuildCatalog(path, "0.1.0",
			registrytest.TxFixture{Name: "deployment", Resources: []string{"container"}})},
	)
	octx := cuecontext.New()
	p := registrytest.BuildPlatform(t, octx, fmt.Sprintf(`{ %q: {enable: true, version: "1.0.0"} }`, path+"@v0"))

	_, err := Materialize(context.Background(), registrytest.NewCtxOwner(octx), registry, p)
	require.Error(t, err)
	var me *oerrors.MaterializeError
	require.True(t, errors.As(err, &me), "major disagreement surfaces as MaterializeError: %v", err)
	assert.Equal(t, oerrors.MaterializeKindCatalog, me.Kind)
	assert.Equal(t, path+"@v0", me.Subscription)
	assert.Equal(t, "1.0.0", me.Version)
	assert.Contains(t, err.Error(), "outside the subscription key's major")
}

// D11/D9 — a pulled catalog whose metadata lies about its identity (here: a
// stale metadata.version, the measured jellyfin defect class) is refused with
// a typed IdentityError wrapped in the MaterializeError, reachable via
// errors.As.
func TestMaterialize_IdentityMismatch(t *testing.T) {
	path := registrytest.UniquePath(t, "cat")
	body := fmt.Sprintf("metadata: {\n\tmodulePath:  %q\n\tversion:     \"0.2.0\"\n\tdescription: \"stale version label\"\n}\n#transformers: {}\n", path+"@v0")
	registry := registrytest.NewCatalogRegistry(t,
		registrytest.CatalogFixture{Path: path, Version: "0.1.0", Body: body},
	)
	octx := cuecontext.New()
	p := registrytest.BuildPlatform(t, octx, fmt.Sprintf(`{ %q: {enable: true, version: "0.1.0"} }`, subKey(path, "0.1.0")))

	_, err := Materialize(context.Background(), registrytest.NewCtxOwner(octx), registry, p)
	require.Error(t, err)

	var me *oerrors.MaterializeError
	require.True(t, errors.As(err, &me), "identity mismatch surfaces as MaterializeError: %v", err)
	assert.Equal(t, oerrors.MaterializeKindCatalog, me.Kind)

	var ie oerrors.IdentityError
	require.True(t, errors.As(err, &ie), "IdentityError reachable through the MaterializeError wrap: %v", err)
	assert.Equal(t, "catalog", ie.Artifact)
	assert.Equal(t, "version", ie.Field)
	assert.Equal(t, "0.2.0", ie.Declared)
	assert.Equal(t, "0.1.0", ie.Fetched)
}

// D11 — a pulled catalog whose metadata declares a different modulePath than
// the subscription key it was pulled by is refused with a typed IdentityError
// (Field "path") wrapped in the MaterializeError.
func TestMaterialize_IdentityPathMismatch(t *testing.T) {
	path := registrytest.UniquePath(t, "cat")
	other := registrytest.UniquePath(t, "other")
	body := fmt.Sprintf("metadata: {\n\tmodulePath:  %q\n\tversion:     \"0.1.0\"\n\tdescription: \"wrong address\"\n}\n#transformers: {}\n", other+"@v0")
	registry := registrytest.NewCatalogRegistry(t,
		registrytest.CatalogFixture{Path: path, Version: "0.1.0", Body: body},
	)
	octx := cuecontext.New()
	p := registrytest.BuildPlatform(t, octx, fmt.Sprintf(`{ %q: {enable: true, version: "0.1.0"} }`, subKey(path, "0.1.0")))

	_, err := Materialize(context.Background(), registrytest.NewCtxOwner(octx), registry, p)
	require.Error(t, err)

	var ie oerrors.IdentityError
	require.True(t, errors.As(err, &ie), "IdentityError reachable through the MaterializeError wrap: %v", err)
	assert.Equal(t, "catalog", ie.Artifact)
	assert.Equal(t, "path", ie.Field)
	assert.Equal(t, other+"@v0", ie.Declared)
	assert.Equal(t, subKey(path, "0.1.0"), ie.Fetched)
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
	assert.Empty(t, mapKeys(p.Package, cue.ParsePath("#composedTransformers")),
		"input platform #composedTransformers must remain empty")
	assert.Empty(t, mapKeys(mp1.Source.Package, cue.ParsePath("#composedTransformers")),
		"Source.Package #composedTransformers must remain unfilled (federation, ADR-003)")
}
