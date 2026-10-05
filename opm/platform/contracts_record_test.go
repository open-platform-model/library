package platform_test

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/internal/schematest"
	"github.com/open-platform-model/library/opm/platform"
)

// The tests below pin the platform-artifact requirement "A platform records
// its core floor and contract inventory at construction".

// recordedInventorySrc is a current-core #contracts carrying every report
// field, one Comparable row, a defined contract nothing requires (an empty
// requiredBy list) and an empty collision report.
const recordedInventorySrc = `
kind: "Platform"
metadata: name: "recorded"
type: "kubernetes"
#contracts: {
	definedBy: {"r/a@v1": "cat@v0", "r/b@v1": "cat@v0"}
	requiredBy: {"r/a@v1": ["t/x@0.1.0", "t/y@0.1.0"], "r/b@v1": []}
	providedBy: {"r/g@v1": ["cat@v0", "cat2@v0"]}
	unfulfilled: []
	overSubscribed: ["r/g@v1"]
	comparable: [{broader: "t/x@0.1.0", narrower: "t/y@0.1.0", contracts: ["r/a@v1"]}]
	fulfilled:     true
	routable:      false
	discriminated: false
	collisions: []
	collidingEntries: {}
}
`

func compilePlatformValue(t *testing.T, src string) cue.Value {
	t.Helper()
	v := cuecontext.New().CompileString(src)
	require.NoError(t, v.Err())
	return v
}

// "Contracts and the floor no longer read Package": an acquired
// current-core platform keeps its inventory and its floor after its Package
// is replaced with the zero value.
func TestContracts_RecordedAtConstructionSurvivesZeroPackage(t *testing.T) {
	k := contractsKernel(t)
	dir := filepath.Join(schematest.LibraryRoot(t), "testdata", "render", "platform")
	plat := acquireContractsPlatform(t, k, dir)

	// Zero Package before the first read, so only a record taken at
	// construction can answer.
	plat.Package = cue.Value{}

	want, err := acquireContractsPlatform(t, k, dir).Contracts()
	require.NoError(t, err)

	after, err := plat.Contracts()
	require.NoError(t, err, "the recorded inventory is returned without reading Package")
	assert.Equal(t, want, after)
	assert.NoError(t, plat.CoreFloor(), "the recorded floor is read without reading Package")
}

// "Each call returns its own inventory": changing a map entry, truncating a
// slice and changing a Comparable row's Contracts in one result leaves the
// next call's result as the decode produced it, nil and empty kept apart.
func TestContracts_EachCallReturnsItsOwnInventory(t *testing.T) {
	plat, err := platform.NewPlatformFromValue(compilePlatformValue(t, recordedInventorySrc))
	require.NoError(t, err)

	// want comes from an independent decode, so a shared record would
	// not change it along with got.
	other, err := platform.NewPlatformFromValue(compilePlatformValue(t, recordedInventorySrc))
	require.NoError(t, err)
	want, err := other.Contracts()
	require.NoError(t, err)
	got, err := plat.Contracts()
	require.NoError(t, err)
	require.Equal(t, want, got)
	first, err := plat.Contracts()
	require.NoError(t, err)
	assert.NotSame(t, first, got, "each call returns a new inventory")

	got.DefinedBy["r/a@v1"] = "changed"
	got.RequiredBy["r/a@v1"][0] = "changed"
	got.RequiredBy["r/a@v1"] = got.RequiredBy["r/a@v1"][:1]
	got.ProvidedBy["r/g@v1"][1] = "changed"
	got.OverSubscribed[0] = "changed"
	got.Comparable[0].Contracts[0] = "changed"
	got.Comparable[0].Broader = "changed"
	got.CollidingEntries["new"] = []string{"x"}
	got.Collisions = append(got.Collisions, "new")

	next, err := plat.Contracts()
	require.NoError(t, err)
	assert.Equal(t, want, next, "a caller's change never reaches the next call")

	// The decoded empty values stay empty, not nil.
	assert.NotNil(t, next.Unfulfilled)
	assert.Empty(t, next.Unfulfilled)
	assert.NotNil(t, next.RequiredBy["r/b@v1"])
	assert.Empty(t, next.RequiredBy["r/b@v1"])
	assert.NotNil(t, next.Collisions)
	assert.NotNil(t, next.CollidingEntries)
}

// "The zero platform reports no contracts": a zero Platform has no
// #contracts; Contracts refuses with field #contracts and the floor is unmet.
func TestContracts_ZeroPlatform(t *testing.T) {
	plat := &platform.Platform{}

	inv, err := plat.Contracts()
	assert.Nil(t, inv)
	var old *oerrors.PlatformCoreTooOldError
	require.True(t, errors.As(err, &old), "got: %v", err)
	assert.Equal(t, oerrors.PlatformCoreTooOldError{Field: "#contracts", Since: "2.0.0-alpha.9", Require: "2.0.0-alpha.12"}, *old)

	var floor *oerrors.PlatformCoreTooOldError
	require.True(t, errors.As(plat.CoreFloor(), &floor))
	assert.Equal(t, oerrors.PlatformCoreTooOldError{Field: "providedBy", Since: "2.0.0-alpha.12", Require: "2.0.0-alpha.12"}, *floor)
}

// "A hand-built platform keeps its results": a struct literal over a valid
// value decodes, on its first call, the inventory the constructor records.
func TestContracts_StructLiteralMatchesConstructed(t *testing.T) {
	v := compilePlatformValue(t, recordedInventorySrc)
	built, err := platform.NewPlatformFromValue(v)
	require.NoError(t, err)
	literal := &platform.Platform{Metadata: built.Metadata, Package: v}

	want, err := built.Contracts()
	require.NoError(t, err)
	got, err := literal.Contracts()
	require.NoError(t, err)
	assert.Equal(t, want, got)
	assert.NoError(t, literal.CoreFloor())

	// The literal decoded once: a later change to its Package is not seen.
	literal.Package = cue.Value{}
	again, err := literal.Contracts()
	require.NoError(t, err)
	assert.Equal(t, want, again)
}

// "A contracts value that does not evaluate still constructs": the
// refusal is recorded, never returned from construction, and the floor,
// a presence test, still holds.
func TestContracts_NonEvaluatingContractsStillConstructs(t *testing.T) {
	v := cuecontext.New().CompileString(`
kind: "Platform"
metadata: name: "conflict"
type: "kubernetes"
#contracts: {
	providedBy: {}
	definedBy: a: "x"
	definedBy: a: "y"
}
`)
	plat, err := platform.NewPlatformFromValue(v)
	require.NoError(t, err, "a contracts refusal never fails construction")

	inv, err := plat.Contracts()
	assert.Nil(t, inv)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "platform #contracts did not evaluate")
	assert.Contains(t, err.Error(), "conflicting values")
	assert.NoError(t, plat.CoreFloor(), "providedBy is present, so the floor is met")
}

// "Concurrent reads of one platform are race-free": goroutines read one
// constructed platform and one struct literal over the same Package at the
// same time. Run under -race (task test runs opm/platform with it).
func TestContracts_ConcurrentReadsAreRaceFree(t *testing.T) {
	k := contractsKernel(t)
	built := acquireContractsPlatform(t, k, filepath.Join(schematest.LibraryRoot(t), "testdata", "render", "platform"))
	literal := &platform.Platform{Metadata: built.Metadata, Package: built.Package, Source: built.Source}
	want, err := built.Contracts()
	require.NoError(t, err)

	const n = 8
	type result struct {
		inv   *platform.ContractInventory
		err   error
		floor error
	}
	results := make([]result, 2*n)
	var wg sync.WaitGroup
	for i := range 2 * n {
		plat := built
		if i%2 == 1 {
			plat = literal
		}
		wg.Add(1)
		go func(i int, plat *platform.Platform) {
			defer wg.Done()
			results[i].floor = plat.CoreFloor()
			results[i].inv, results[i].err = plat.Contracts()
		}(i, plat)
	}
	wg.Wait()
	for i, r := range results {
		require.NoError(t, r.err, "goroutine %d", i)
		assert.NoError(t, r.floor, "goroutine %d", i)
		assert.Equal(t, want, r.inv, "goroutine %d", i)
	}
}
