package catalog_test

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/catalog"
)

// newCatalog compiles a catalog-shaped value from CUE text and wraps it. The
// text is not built against core: Provides reads the artifact by path, so the
// unit cases here pin the fold's behaviour — ordering, deduplication, the
// empty set, the skipped fulfilments — without paying a registry resolution
// per case. That the paths agree with the shape core actually publishes is
// pinned end to end by the acquisition tests in opm/kernel, which build real
// #Catalog artifacts against core.
func newCatalog(t *testing.T, body string) *catalog.Catalog {
	t.Helper()
	v := cuecontext.New().CompileString(body)
	require.NoError(t, v.Err())
	c, err := catalog.NewCatalogFromValue(v)
	require.NoError(t, err)
	return c
}

// catalogBody wraps a #transformers block in the metadata every catalog
// carries, so a case only has to author the part under test.
func catalogBody(transformers string) string {
	return `kind: "Catalog"
metadata: {
	modulePath: "test.example/catalogs/provider@v1"
	version:    "1.0.0"
	fqn:        "test.example/catalogs/provider@v1"
}
` + transformers
}

const (
	backupTrait   = "test.example/catalogs/base/traits/backup@v1"
	restoreTrait  = "test.example/catalogs/base/traits/restore@v1"
	volumeRes     = "test.example/catalogs/base/resources/volumes@v1"
	containerRes  = "test.example/catalogs/base/resources/container@v1"
	scheduleImpl  = "test.example/catalogs/provider/transformers/schedule@1.0.0"
	prebackupImpl = "test.example/catalogs/provider/transformers/prebackup@1.0.0"
)

// catalog-acquisition spec, "The provider-fulfilled contract set is derived
// from the catalog": the fold collects every contract a transformer of this
// catalog REQUIRES whose value carries fulfilment "provider", from both demand
// maps, deduplicated across transformers and deterministically ordered. Any
// other fulfilment, an absent fulfilment, an optional demand and a catalog
// with no transformers at all contribute nothing and fail nothing.
func TestCatalog_Provides(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []string
	}{
		{
			name: "provider-fulfilled demands from both maps are collected and sorted",
			body: catalogBody(`#transformers: {
	"` + scheduleImpl + `": {
		requiredTraits: {
			"` + restoreTrait + `": fulfilment: "provider"
			"` + backupTrait + `": fulfilment:  "provider"
		}
		requiredResources: "` + volumeRes + `": fulfilment: "provider"
	}
}`),
			// Sorted, not authored order: restore is written before backup
			// above, and the resource demand after both traits.
			want: []string{volumeRes, backupTrait, restoreTrait},
		},
		{
			name: "a demand carrying another fulfilment is not collected",
			body: catalogBody(`#transformers: {
	"` + scheduleImpl + `": {
		requiredTraits: "` + backupTrait + `": fulfilment:        "provider"
		requiredResources: "` + containerRes + `": fulfilment:     "catalog"
	}
}`),
			want: []string{backupTrait},
		},
		{
			name: "a catalog whose transformers require no provider contract derives the empty set",
			body: catalogBody(`#transformers: {
	"` + scheduleImpl + `": {
		requiredResources: "` + containerRes + `": fulfilment: "catalog"
		requiredTraits: "` + restoreTrait + `": fulfilment:    "catalog"
	}
}`),
			want: []string{},
		},
		{
			name: "a catalog shipping no transformers derives the empty set",
			body: catalogBody(""),
			want: []string{},
		},
		{
			name: "a transformer with no demand maps contributes nothing",
			body: catalogBody(`#transformers: "` + scheduleImpl + `": {
	requiredLabels: "opm.test/workload": "stateless"
}`),
			want: []string{},
		},
		{
			name: "a demand carrying no fulfilment at all is not collected",
			body: catalogBody(`#transformers: "` + scheduleImpl + `": {
	requiredResources: "` + containerRes + `": kind: "Resource"
	requiredTraits: "` + backupTrait + `": fulfilment: "provider"
}`),
			want: []string{backupTrait},
		},
		{
			name: "one contract required by two transformers appears once",
			body: catalogBody(`#transformers: {
	"` + scheduleImpl + `": requiredTraits: "` + backupTrait + `": fulfilment:  "provider"
	"` + prebackupImpl + `": requiredTraits: "` + backupTrait + `": fulfilment: "provider"
}`),
			want: []string{backupTrait},
		},
		{
			name: "an optional demand is tolerance, not fulfilment",
			body: catalogBody(`#transformers: "` + scheduleImpl + `": {
	optionalTraits: "` + backupTrait + `": fulfilment:    "provider"
	optionalResources: "` + volumeRes + `": fulfilment: "provider"
}`),
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newCatalog(t, tt.body)

			got, err := c.Provides()
			require.NoError(t, err, "deriving is a report: it refuses nothing")
			assert.Equal(t, tt.want, got)

			// The order is part of the contract, so a consumer comparing a
			// derivation against a claimed list never has to sort first.
			again, err := c.Provides()
			require.NoError(t, err)
			assert.Equal(t, got, again, "two derivations of one catalog are equal element for element")
		})
	}
}

// A fulfilment that is present but does not read as a concrete string means
// the contract was built against a core that means something else by the
// field. The design calls for that to break loudly rather than fold away
// silently, and the error names the contract and the transformer requiring it.
func TestCatalog_Provides_NonConcreteFulfilmentIsReported(t *testing.T) {
	c := newCatalog(t, catalogBody(`#transformers: "`+scheduleImpl+`": {
	requiredTraits: "`+backupTrait+`": fulfilment: string
}`))

	got, err := c.Provides()
	require.Error(t, err)
	assert.Nil(t, got, "no partial set is returned")
	assert.Contains(t, err.Error(), backupTrait)
	assert.Contains(t, err.Error(), scheduleImpl)
}

func TestCatalog_Provides_NilReceiver(t *testing.T) {
	var c *catalog.Catalog
	got, err := c.Provides()
	require.Error(t, err)
	assert.Nil(t, got)
}
