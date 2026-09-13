package platform_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/internal/schematest"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/platform"
)

// The render fixtures (testdata/render) serve the catalogs whose contract
// maps these tests read: cat lists every member it defines, cat2 defines no
// member of its own and lists nothing. Values pinned by the read-contract-
// inventory spike (design.md § `defined` is not decoded).
const (
	contractsPrefix = "testing.opmodel.dev/library-render"
	contractsCat    = contractsPrefix + "/cat@v0"
	contractsCat2   = contractsPrefix + "/cat2@v0"
	contractsRes    = contractsPrefix + "/cat/resources"
	contractsTraits = contractsPrefix + "/cat/traits"
	contractsTx     = contractsPrefix + "/cat/transformers"
	contractsTx2    = contractsPrefix + "/cat2/transformers"
)

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

	assert.Empty(t, inv.Unfulfilled)
	assert.Empty(t, inv.OverSubscribed)
	assert.True(t, inv.Fulfilled)
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

	assert.Equal(t, []string{contractsRes + "/gateway@v1"}, inv.OverSubscribed)
	assert.False(t, inv.Routable)
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
	assert.Empty(t, inv.DefinedBy)
	assert.Empty(t, inv.RequiredBy)
	assert.Empty(t, inv.Unfulfilled)
	assert.Empty(t, inv.OverSubscribed)
	assert.True(t, inv.Fulfilled)
	assert.True(t, inv.Routable)
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
}
