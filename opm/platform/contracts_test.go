package platform_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/internal/schematest"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/platform"
	"github.com/open-platform-model/library/opm/schema"
)

// The render fixtures (testdata/render) serve the catalogs whose contract
// maps these tests read: cat lists every member it defines, cat2 defines no
// member of its own and lists nothing. The six original inventory fields
// were pinned by the read-contract-inventory spike (archived at
// openspec/changes/archive/2026-09-13-read-contract-inventory/design.md);
// the comparable-predicate rows below were pinned by the
// read-comparable-predicates spike, which fixed the fixture values before
// these tests were written.
const (
	contractsPrefix = "testing.opmodel.dev/library-render"
	contractsCat    = contractsPrefix + "/cat@v0"
	contractsCat2   = contractsPrefix + "/cat2@v0"
	contractsCatV1  = contractsPrefix + "/cat@v1"
	contractsProv   = contractsPrefix + "/providers@v0"
	contractsRes    = contractsPrefix + "/cat/resources"
	contractsTraits = contractsPrefix + "/cat/traits"
	contractsTx     = contractsPrefix + "/cat/transformers"
	contractsTx2    = contractsPrefix + "/cat2/transformers"

	// The colliding-majors fixtures: maj 0.1.0 and maj 1.4.0 list the same
	// container, expose and backup keys; bprov at each major requires the
	// backup trait.
	contractsMajV0   = contractsPrefix + "/maj@v0"
	contractsMajV1   = contractsPrefix + "/maj@v1"
	contractsBprovV0 = contractsPrefix + "/bprov@v0"
	contractsBprovV1 = contractsPrefix + "/bprov@v1"
	contractsMajRes  = contractsPrefix + "/maj/resources"
	contractsMajTr   = contractsPrefix + "/maj/traits"
	contractsMajTx   = contractsPrefix + "/maj/transformers"
)

// assertNoCollisions pins the empty collision report on a platform whose
// enabled entries share no contract key.
func assertNoCollisions(t *testing.T, inv *platform.ContractInventory) {
	t.Helper()
	assert.Empty(t, inv.Collisions, "no collisions")
	assert.Empty(t, inv.CollidingEntries, "no colliding entries")
}

func contractsKernel(t *testing.T) *kernel.Kernel {
	t.Helper()
	dir := filepath.Join(schematest.LibraryRoot(t), "testdata", "render", "registry")
	return kernel.New(kernel.WithRegistry(registrytest.NewRegistryFromDir(t, dir, contractsPrefix)))
}

func acquireContractsPlatform(t *testing.T, k *kernel.Kernel, dir string) *platform.Platform {
	t.Helper()
	p, err := k.AcquirePlatformFromDir(context.Background(), dir)
	require.NoError(t, err, "acquiring platform %s", dir)
	return p
}

// platform-artifact spec, "The inventory of a healthy platform reads as
// fulfilled and routable": one enabled catalog listing every member it
// defines, its one provider-fulfilled contract (the gateway resource)
// required by exactly one transformer.
func TestContracts_ListedCatalogReadsFulfilledAndRoutable(t *testing.T) {
	k := contractsKernel(t)
	plat := acquireContractsPlatform(t, k, filepath.Join(schematest.LibraryRoot(t), "testdata", "render", "platform"))

	inv, err := plat.Contracts()
	require.NoError(t, err)
	require.NotNil(t, inv)
	assertNoCollisions(t, inv)

	// definedBy: every listed contract, to the registry key of the listing
	// catalog (its module path, never a prefix parsed off the FQN).
	assert.Len(t, inv.DefinedBy, 17)
	for fqn, by := range inv.DefinedBy {
		assert.Equal(t, contractsCat, by, "definedBy[%s]", fqn)
	}
	assert.Contains(t, inv.DefinedBy, contractsRes+"/gateway@v1")
	assert.Contains(t, inv.DefinedBy, contractsTraits+"/backup@v1")

	// requiredBy: required demands only, per defined contract; a defined
	// contract nothing requires maps to an empty list.
	assert.Len(t, inv.RequiredBy, 17)
	assert.ElementsMatch(t,
		[]string{contractsTx + "/deployment-transformer@0.1.0", contractsTx + "/service-transformer@0.1.0"},
		inv.RequiredBy[contractsRes+"/container@v1"])
	assert.Equal(t, []string{contractsTx + "/gateway-transformer@0.1.0"}, inv.RequiredBy[contractsRes+"/gateway@v1"])
	assert.Equal(t, []string{contractsTx + "/service-transformer@0.1.0"}, inv.RequiredBy[contractsTraits+"/expose@v1"],
		"the deployment transformer only optionally consumes expose: tolerance, not a requirement")
	for _, unrequired := range []string{
		contractsRes + "/orphan@v1", contractsRes + "/ladder@v2",
		contractsTraits + "/backup@v1", contractsTraits + "/sidecar@v1", contractsTraits + "/unstated@v1",
	} {
		assert.Contains(t, inv.RequiredBy, unrequired)
		assert.Empty(t, inv.RequiredBy[unrequired], "requiredBy[%s]", unrequired)
	}

	// providedBy: the one provider-fulfilled contract, to the one registry
	// entry supplying it.
	assert.Equal(t, map[string][]string{contractsRes + "/gateway@v1": {contractsCat}}, inv.ProvidedBy)

	assert.Empty(t, inv.Unfulfilled)
	assert.Empty(t, inv.OverSubscribed)
	assert.True(t, inv.Fulfilled)
	assert.True(t, inv.Routable)

	// cat's two container-requiring transformers each add a demand the
	// other lacks — deployment a required label, service a required trait —
	// so neither predicate contains the other and the platform is
	// discriminated.
	assert.Empty(t, inv.Comparable)
	assert.True(t, inv.Discriminated)
}

// platform-artifact spec, "An undiscriminated platform is reported, not
// refused": cat 0.1.0 beside cat2 0.1.0, whose mirror-transformer requires
// the catalog-fulfilled container resource alone. Its predicate therefore
// contains both of cat's container-requiring predicates. Neither catalog
// over-subscribes a provider-fulfilled contract here, so Routable stays
// true and the discrimination report is read on its own.
func TestContracts_UndiscriminatedIsReportedNotRefused(t *testing.T) {
	k := contractsKernel(t)
	plat := acquireContractsPlatform(t, k, filepath.Join(schematest.LibraryRoot(t), "testdata", "render", "platform_two"))

	inv, err := plat.Contracts()
	require.NoError(t, err, "an undiscriminated platform is a report, never a refusal")
	require.NotNil(t, inv)
	assertNoCollisions(t, inv)

	assert.ElementsMatch(t, []platform.ComparablePredicates{
		{
			Broader:   contractsTx2 + "/mirror-transformer@0.1.0",
			Narrower:  contractsTx + "/deployment-transformer@0.1.0",
			Contracts: []string{contractsRes + "/container@v1"},
		},
		{
			Broader:   contractsTx2 + "/mirror-transformer@0.1.0",
			Narrower:  contractsTx + "/service-transformer@0.1.0",
			Contracts: []string{contractsRes + "/container@v1"},
		},
	}, inv.Comparable, "deployment against service is incomparable: a required label against a required trait")
	assert.False(t, inv.Discriminated)

	// The report is orthogonal to routing: nothing here over-subscribes a
	// provider-fulfilled contract.
	assert.Equal(t, map[string][]string{contractsRes + "/gateway@v1": {contractsCat}}, inv.ProvidedBy)
	assert.Empty(t, inv.OverSubscribed)
	assert.True(t, inv.Routable)
}

// platform-artifact spec, "An over-subscribed platform is reported, not
// refused": cat 0.1.0 beside cat2 0.2.0, both supplying a transformer that
// requires the provider-fulfilled gateway contract cat lists. Acquisition
// and the accessor both succeed; the inventory names the key.
func TestContracts_OverSubscribedIsReportedNotRefused(t *testing.T) {
	k := contractsKernel(t)
	plat := acquireContractsPlatform(t, k, filepath.Join(schematest.LibraryRoot(t), "testdata", "render", "platform_oversubscribed"))

	inv, err := plat.Contracts()
	require.NoError(t, err, "an over-subscribed platform is a report, never a refusal")
	require.NotNil(t, inv)
	assertNoCollisions(t, inv)

	assert.Equal(t, []string{contractsRes + "/gateway@v1"}, inv.OverSubscribed)
	assert.False(t, inv.Routable)
	assert.Equal(t, map[string][]string{contractsRes + "/gateway@v1": {contractsCat2, contractsCat}}, inv.ProvidedBy,
		"both supplying registry keys, ascending")
	assert.ElementsMatch(t,
		[]string{contractsTx + "/gateway-transformer@0.1.0", contractsTx2 + "/gateway-transformer@0.2.0"},
		inv.RequiredBy[contractsRes+"/gateway@v1"])
	assert.ElementsMatch(t,
		[]string{contractsTx + "/deployment-transformer@0.1.0", contractsTx + "/service-transformer@0.1.0", contractsTx2 + "/mirror-transformer@0.2.0"},
		inv.RequiredBy[contractsRes+"/container@v1"],
		"the catalog-fulfilled container admits any number of suppliers")

	// cat2 defines nothing, so every defining catalog is still cat.
	assert.Len(t, inv.DefinedBy, 17)
	for fqn, by := range inv.DefinedBy {
		assert.Equal(t, contractsCat, by, "definedBy[%s]", fqn)
	}
	assert.Empty(t, inv.Unfulfilled)
	assert.True(t, inv.Fulfilled)

	// cat2 0.2.0's mirror-transformer still requires the container alone,
	// so the two undiscriminated pairs are reported here beside the
	// over-subscription: the two verdicts are independent.
	assert.ElementsMatch(t, []platform.ComparablePredicates{
		{
			Broader:   contractsTx2 + "/mirror-transformer@0.2.0",
			Narrower:  contractsTx + "/deployment-transformer@0.1.0",
			Contracts: []string{contractsRes + "/container@v1"},
		},
		{
			Broader:   contractsTx2 + "/mirror-transformer@0.2.0",
			Narrower:  contractsTx + "/service-transformer@0.1.0",
			Contracts: []string{contractsRes + "/container@v1"},
		},
	}, inv.Comparable)
	assert.False(t, inv.Discriminated)
}

// platform-artifact spec, "An unlisted demand leaves the inventory empty":
// a platform embedding only cat2, which carries transformers but lists no
// contract, derives the empty inventory: empty maps and lists, both booleans
// true, and no `defined` on the returned value (it has no field for it).
func TestContracts_UnlistedCatalogReadsEmpty(t *testing.T) {
	k := contractsKernel(t)
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cue.mod"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cue.mod", "module.cue"), []byte(`module: "testing.opmodel.dev/library-platform-test/cat2only@v0"
language: version: "v0.17.0"
deps: {
	"opmodel.dev/core@v2": v: "`+registrytest.DefaultCoreVersion+`"
	"`+contractsCat2+`": v: "v0.1.0"
	"`+contractsCat+`": v: "v0.1.0"
}
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "platform.cue"), []byte(`package platform

import (
	c "opmodel.dev/core@v2"
	cat2 "`+contractsCat2+`"
)

c.#Platform
metadata: name: "cat2-only"
type: "kubernetes"
#registry: "`+contractsCat2+`": {
	enable:   true
	#catalog: cat2
}
`), 0o644))
	plat := acquireContractsPlatform(t, k, dir)

	inv, err := plat.Contracts()
	require.NoError(t, err)
	require.NotNil(t, inv)
	assertNoCollisions(t, inv)
	assert.Empty(t, inv.DefinedBy)
	assert.Empty(t, inv.RequiredBy)
	assert.Empty(t, inv.ProvidedBy, "cat2 0.1.0 requires no provider-fulfilled contract")
	assert.Empty(t, inv.Unfulfilled)
	assert.Empty(t, inv.OverSubscribed)
	assert.True(t, inv.Fulfilled)
	assert.True(t, inv.Routable)
	assert.Empty(t, inv.Comparable)
	assert.True(t, inv.Discriminated)
}

// A disabled catalog's transformers leave the predicate fold entirely, so
// the pairs platform_two reports are gone: cat is disabled beside cat2
// 0.1.0 and the one remaining transformer has nothing to be comparable
// with.
func TestContracts_DisabledCatalogReadsDiscriminated(t *testing.T) {
	k := contractsKernel(t)
	plat := acquireContractsPlatform(t, k, filepath.Join(schematest.LibraryRoot(t), "testdata", "render", "platform_disabled"))

	inv, err := plat.Contracts()
	require.NoError(t, err)
	require.NotNil(t, inv)
	assertNoCollisions(t, inv)

	assert.Empty(t, inv.Comparable)
	assert.True(t, inv.Discriminated)
	assert.Empty(t, inv.ProvidedBy, "the disabled catalog's gateway provider is not counted")
	assert.True(t, inv.Routable)
}

// platform_providers: providers lists snapshot and ledger with no
// transformer, and archive with one label-gated provider. ProvidedBy holds
// only what some enabled transformer requires; the two unprovided keys are
// Unfulfilled because an enabled catalog defines them and nothing provides
// them.
func TestContracts_ProvidersPlatformProvidedBy(t *testing.T) {
	k := contractsKernel(t)
	plat := acquireContractsPlatform(t, k, filepath.Join(schematest.LibraryRoot(t), "testdata", "render", "platform_providers"))

	inv, err := plat.Contracts()
	require.NoError(t, err)
	require.NotNil(t, inv)
	assertNoCollisions(t, inv)

	assert.Equal(t, map[string][]string{
		contractsRes + "/gateway@v1":                     {contractsCat},
		contractsPrefix + "/providers/traits/archive@v1": {contractsProv},
	}, inv.ProvidedBy)
	assert.ElementsMatch(t, []string{
		contractsPrefix + "/providers/resources/ledger@v1",
		contractsPrefix + "/providers/traits/snapshot@v1",
	}, inv.Unfulfilled)
	assert.False(t, inv.Fulfilled)
	assert.Empty(t, inv.OverSubscribed)
	assert.True(t, inv.Routable)
}

// platform-artifact spec, "Two majors of one provider catalog are two
// providers": cat 0.1.0 and cat 1.0.0 each carry a transformer requiring
// the gateway contract cat 0.1.0 lists. Core stamps both with one
// major-free transformer path; the count is per registry entry.
func TestContracts_TwoMajorsAreTwoProviders(t *testing.T) {
	k := contractsKernel(t)
	plat := acquireContractsPlatform(t, k, filepath.Join(schematest.LibraryRoot(t), "testdata", "render", "platform_two_majors"))

	inv, err := plat.Contracts()
	require.NoError(t, err, "an over-subscribed platform is a report, never a refusal")
	require.NotNil(t, inv)
	assertNoCollisions(t, inv)

	gateway := contractsRes + "/gateway@v1"
	assert.Equal(t, []string{contractsCat, contractsCatV1}, inv.ProvidedBy[gateway])
	assert.Equal(t, []string{gateway}, inv.OverSubscribed)
	assert.False(t, inv.Routable)
}

// platform-artifact spec, "A disabled definer does not hide
// over-subscription": cat 0.1.0 (the gateway's definer) is disabled, cat2
// 0.2.0 and cat 1.0.0 both require the gateway. Nothing enabled defines it,
// so DefinedBy and RequiredBy lack the key and ProvidedBy is the one field
// naming the supplying entries.
func TestContracts_DisabledDefinerDoesNotHideOverSubscription(t *testing.T) {
	k := contractsKernel(t)
	plat := acquireContractsPlatform(t, k, filepath.Join(schematest.LibraryRoot(t), "testdata", "render", "platform_definer_disabled"))

	inv, err := plat.Contracts()
	require.NoError(t, err)
	require.NotNil(t, inv)
	assertNoCollisions(t, inv)

	gateway := contractsRes + "/gateway@v1"
	assert.Equal(t, []string{contractsCat2, contractsCatV1}, inv.ProvidedBy[gateway])
	assert.Equal(t, []string{gateway}, inv.OverSubscribed)
	assert.False(t, inv.Routable)
	assert.NotContains(t, inv.DefinedBy, gateway)
	assert.NotContains(t, inv.RequiredBy, gateway)
}

// A value carrying no #contracts (built against a core release before
// 2.0.0-alpha.9, or not a #Platform at all) is refused by the accessor with
// an error naming the missing field; construction itself does not read it.
func TestContracts_MissingInventoryErrors(t *testing.T) {
	v := cuecontext.New().CompileString(`
kind: "Platform"
metadata: name: "bare"
type: "kubernetes"
`)
	require.NoError(t, v.Err())
	plat, err := platform.NewPlatformFromValue(v)
	require.NoError(t, err, "construction never decodes the inventory")

	inv, err := plat.Contracts()
	require.Error(t, err)
	assert.Nil(t, inv)
	assert.Contains(t, err.Error(), "#contracts")
	var old *oerrors.PlatformCoreTooOldError
	require.True(t, errors.As(err, &old), "the refusal is the typed core-floor error")
	assert.Equal(t, oerrors.PlatformCoreTooOldError{Platform: "bare", Field: "#contracts", Since: "2.0.0-alpha.9", Require: "2.0.0-alpha.12"}, *old)
}

// platform-artifact spec, "An inventory that predates the
// comparable-predicate report is refused": a #contracts carrying the six
// alpha.9 fields and neither report field is a platform module pinning an
// older core. The accessor refuses it rather than defaulting the missing
// verdict, and names both the field and the release that derives it, so the
// caller knows what to re-pin to.
func TestContracts_InventoryPredatingTheReportErrors(t *testing.T) {
	v := cuecontext.New().CompileString(`
kind: "Platform"
metadata: name: "alpha9"
type: "kubernetes"
#contracts: {
	definedBy: {}
	requiredBy: {}
	unfulfilled: []
	overSubscribed: []
	fulfilled:      true
	routable:       true
}
`)
	require.NoError(t, v.Err())
	plat, err := platform.NewPlatformFromValue(v)
	require.NoError(t, err)

	inv, err := plat.Contracts()
	require.Error(t, err)
	assert.Nil(t, inv, "a missing report is never a partial inventory")
	assert.Contains(t, err.Error(), `"comparable"`)
	assert.Contains(t, err.Error(), "2.0.0-alpha.10")
	assert.Contains(t, err.Error(), "to v2.0.0-alpha.12 or later", "one re-pin, to the floor, clears every missing field")
	var old *oerrors.PlatformCoreTooOldError
	require.True(t, errors.As(err, &old), "the refusal is the typed core-floor error")
	assert.Equal(t, oerrors.PlatformCoreTooOldError{Platform: "alpha9", Field: "comparable", Since: "2.0.0-alpha.10", Require: "2.0.0-alpha.12"}, *old)
}

// platform-artifact spec, "An inventory that predates the provider count is
// refused": a #contracts carrying every report field but providedBy is a
// platform module pinning core 2.0.0-alpha.11 or older. The accessor
// refuses it with the error Kernel.Render returns for the same platform.
func TestContracts_InventoryPredatingProvidedByErrors(t *testing.T) {
	v := cuecontext.New().CompileString(`
kind: "Platform"
metadata: name: "alpha11"
type: "kubernetes"
#contracts: {
	definedBy: {}
	requiredBy: {}
	unfulfilled: []
	overSubscribed: []
	comparable: []
	fulfilled:     true
	routable:      true
	discriminated: true
}
`)
	require.NoError(t, v.Err())
	plat, err := platform.NewPlatformFromValue(v)
	require.NoError(t, err)

	inv, err := plat.Contracts()
	require.Error(t, err)
	assert.Nil(t, inv, "an inventory lacking ProvidedBy is never returned")
	var old *oerrors.PlatformCoreTooOldError
	require.True(t, errors.As(err, &old))
	assert.Equal(t, "alpha11", old.Platform)
	assert.Equal(t, "providedBy", old.Field)
	assert.Equal(t, "2.0.0-alpha.12", old.Since)
	assert.Equal(t, "2.0.0-alpha.12", old.Require)
}

// majCollisions is the collision report every colliding-majors platform
// carries: the three keys maj 0.1.0 and maj 1.4.0 both list, ascending, each
// to both registry keys.
func majCollisions() ([]string, map[string][]string) {
	keys := []string{
		contractsMajRes + "/container@v1",
		contractsMajTr + "/backup@v1",
		contractsMajTr + "/expose@v1",
	}
	entries := map[string][]string{}
	for _, k := range keys {
		entries[k] = []string{contractsMajV0, contractsMajV1}
	}
	return keys, entries
}

// assertMajCollisionFold pins what core folds on a colliding-majors
// platform: the three shared keys are collisions and are absent from
// DefinedBy and RequiredBy; maj 1.4.0's container@v2, which no other entry
// lists, folds as before.
func assertMajCollisionFold(t *testing.T, inv *platform.ContractInventory) {
	t.Helper()
	keys, entries := majCollisions()
	assert.Equal(t, keys, inv.Collisions, "the shared keys, ascending")
	assert.Equal(t, entries, inv.CollidingEntries, "each shared key to both majors, ascending")
	assert.Equal(t, map[string]string{contractsMajRes + "/container@v2": contractsMajV1}, inv.DefinedBy,
		"only the single-definer key folds into definedBy")
	assert.Equal(t, map[string][]string{
		contractsMajRes + "/container@v2": {contractsMajTx + "/deployment-v2-transformer@1.4.0"},
	}, inv.RequiredBy, "a colliding key is absent from requiredBy however many transformers require it")
	for _, k := range keys {
		assert.NotContains(t, inv.DefinedBy, k)
		assert.NotContains(t, inv.RequiredBy, k)
	}
	assert.False(t, inv.Routable, "a collision makes the platform not routable")
}

// platform-artifact spec, "Two majors sharing contract keys are reported as
// collisions": maj 0.1.0 beside maj 1.4.0. The platform acquires and the
// accessor reports the collisions rather than refusing.
func TestContracts_CollidingMajorsAreReported(t *testing.T) {
	k := contractsKernel(t)
	plat := acquireContractsPlatform(t, k, filepath.Join(schematest.LibraryRoot(t), "testdata", "render", "platform_collide"))

	inv, err := plat.Contracts()
	require.NoError(t, err, "a colliding platform is a report, never a refusal")
	require.NotNil(t, inv)

	assertMajCollisionFold(t, inv)
	assert.Empty(t, inv.ProvidedBy, "no transformer requires the provider-fulfilled backup trait here")
	assert.Empty(t, inv.OverSubscribed)
	assert.Empty(t, inv.Unfulfilled)

	// Known blind spot, pinned on purpose: core computes Fulfilled and
	// Discriminated without the colliding keys, so both read true here
	// although maj 0.1.0's deployment transformer and maj 1.4.0's bridge
	// require the colliding container key under equal predicates. A core
	// that closes the blind spot flips these, and this test is updated
	// deliberately with it.
	assert.True(t, inv.Fulfilled, "blind spot: Fulfilled ignores colliding keys")
	assert.Empty(t, inv.Comparable, "blind spot: no Comparable row shares a colliding key")
	assert.True(t, inv.Discriminated, "blind spot: Discriminated ignores colliding keys")
}

// platform-artifact spec, "A colliding key is left out of the other
// reports": platform_collide plus bprov 0.1.0, whose transformer requires
// the colliding provider-fulfilled backup trait. ProvidedBy is unaffected by
// the collision; RequiredBy and Unfulfilled leave the key out.
func TestContracts_CollidingKeyIsLeftOutOfTheOtherReports(t *testing.T) {
	k := contractsKernel(t)
	plat := acquireContractsPlatform(t, k, filepath.Join(schematest.LibraryRoot(t), "testdata", "render", "platform_collide_provider"))

	inv, err := plat.Contracts()
	require.NoError(t, err)
	require.NotNil(t, inv)

	assertMajCollisionFold(t, inv)
	backup := contractsMajTr + "/backup@v1"
	assert.Equal(t, map[string][]string{backup: {contractsBprovV0}}, inv.ProvidedBy,
		"providedBy counts the one provider whether or not the key collides")
	assert.Empty(t, inv.OverSubscribed)
	assert.NotContains(t, inv.RequiredBy, backup)
	assert.Empty(t, inv.Unfulfilled, "a colliding key is never unfulfilled")

	// Known blind spot, pinned on purpose (see
	// TestContracts_CollidingMajorsAreReported).
	assert.True(t, inv.Fulfilled, "blind spot: Fulfilled ignores colliding keys")
	assert.Empty(t, inv.Comparable, "blind spot: no Comparable row shares a colliding key")
	assert.True(t, inv.Discriminated, "blind spot: Discriminated ignores colliding keys")
}

// platform-artifact spec, "Collision and over-subscription are reported
// together": platform_collide plus bprov 0.1.0 and bprov 1.0.0, both
// requiring the colliding backup trait.
func TestContracts_CollisionAndOverSubscriptionTogether(t *testing.T) {
	k := contractsKernel(t)
	plat := acquireContractsPlatform(t, k, filepath.Join(schematest.LibraryRoot(t), "testdata", "render", "platform_collide_oversubscribed"))

	inv, err := plat.Contracts()
	require.NoError(t, err)
	require.NotNil(t, inv)

	assertMajCollisionFold(t, inv)
	backup := contractsMajTr + "/backup@v1"
	assert.Equal(t, map[string][]string{backup: {contractsBprovV0, contractsBprovV1}}, inv.ProvidedBy)
	assert.Equal(t, []string{backup}, inv.OverSubscribed, "over-subscription co-occurs with the collision")
	assert.False(t, inv.Routable)
}

// platform-artifact spec, "An inventory that predates the collision report
// reads no collision": the healthy platform re-pinned to core 2.0.0-alpha.12,
// whose #contracts carries neither collisions nor collidingEntries. The
// absence decodes as empty, never as a refusal: such a core cannot evaluate
// a colliding platform at all.
func TestContracts_InventoryPredatingTheCollisionReportReadsEmpty(t *testing.T) {
	k := contractsKernel(t)
	dir := t.TempDir()
	src := filepath.Join(schematest.LibraryRoot(t), "testdata", "render", "platform")
	for _, rel := range []string{"platform.cue", filepath.Join("cue.mod", "module.cue")} {
		data, err := os.ReadFile(filepath.Join(src, rel))
		require.NoError(t, err)
		if rel != "platform.cue" {
			require.Contains(t, string(data), `v: "`+registrytest.DefaultCoreVersion+`"`)
			data = []byte(strings.Replace(string(data), `v: "`+registrytest.DefaultCoreVersion+`"`, `v: "v2.0.0-alpha.12"`, 1))
		}
		require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(dir, rel)), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, rel), data, 0o644))
	}
	plat := acquireContractsPlatform(t, k, dir)
	require.False(t, plat.Package.LookupPath(schema.ContractsCollisions).Exists(),
		"precondition: core 2.0.0-alpha.12 carries no collision report")

	inv, err := plat.Contracts()
	require.NoError(t, err, "an absent collision report is not a refusal")
	require.NotNil(t, inv)
	assertNoCollisions(t, inv)
	assert.Len(t, inv.DefinedBy, 17, "every other field decodes as before")
	assert.Equal(t, map[string][]string{contractsRes + "/gateway@v1": {contractsCat}}, inv.ProvidedBy)
	assert.True(t, inv.Routable)
	assert.True(t, inv.Fulfilled)
	assert.True(t, inv.Discriminated)
}

// A collision report that is present but does not decode is an error, never
// read as empty: only absence is provably no collision.
func TestContracts_IllTypedCollisionReportErrors(t *testing.T) {
	v := cuecontext.New().CompileString(`
kind: "Platform"
metadata: name: "illtyped"
type: "kubernetes"
#contracts: {
	definedBy: {}
	requiredBy: {}
	providedBy: {}
	unfulfilled: []
	overSubscribed: []
	comparable: []
	fulfilled:     true
	routable:      false
	discriminated: true
	collisions:    "not a list"
	collidingEntries: {}
}
`)
	require.NoError(t, v.Err())
	plat, err := platform.NewPlatformFromValue(v)
	require.NoError(t, err)

	inv, err := plat.Contracts()
	require.Error(t, err)
	assert.Nil(t, inv)
	assert.Contains(t, err.Error(), "collisions")
	var old *oerrors.PlatformCoreTooOldError
	assert.False(t, errors.As(err, &old), "a present field that fails to decode is a decode error, not a core-floor refusal")
}
