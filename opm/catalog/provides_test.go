package catalog_test

import (
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/catalog"
	"github.com/open-platform-model/library/opm/schema"
)

// newCatalog compiles a catalog-shaped value from CUE text and wraps it. The
// text is not built against core, so it carries no `provides` unless a case
// authors one, and the value carries no Source: Provides then runs the
// deprecated fold, and the first table here pins the fold's behaviour —
// ordering, deduplication, the empty set, the skipped fulfilments — without
// paying a registry resolution per case. That the paths agree with the shape
// core actually publishes is pinned end to end by the acquisition tests in
// opm/kernel, which build real #Catalog artifacts against core.
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

// catalog-acquisition spec, "Core's provider set is read when the catalog
// carries it" and "An unreadable provider set is reported": a value carrying
// a `provides` field and no Source is answered from the field, normalised,
// and a field that is not a concrete list of strings is an error naming it.
func TestCatalog_Provides_DecodesCoreField(t *testing.T) {
	// The transformers provide backupTrait; the authored field disagrees, so
	// an answer of restoreTrait proves the decode ran and the fold did not.
	transformers := `#transformers: "` + scheduleImpl + `": requiredTraits: "` + backupTrait + `": fulfilment: "provider"
`
	tests := []struct {
		name     string
		provides string
		want     []string
	}{
		{name: "a present field wins over disagreeing transformers", provides: `["` + restoreTrait + `"]`, want: []string{restoreTrait}},
		{name: "an unsorted, duplicated field is normalised", provides: `["` + restoreTrait + `", "` + backupTrait + `", "` + restoreTrait + `"]`, want: []string{backupTrait, restoreTrait}},
		{name: "an empty field is a non-nil empty set", provides: `[]`, want: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newCatalog(t, catalogBody(transformers+"provides: "+tt.provides+"\n"))
			got, err := c.Provides()
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.want, got)
		})
	}

	for _, tt := range []struct{ name, provides string }{
		{name: "a non-concrete element is reported", provides: `[string]`},
		{name: "a field that is not a list is reported", provides: `"` + backupTrait + `"`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := newCatalog(t, catalogBody(transformers+"provides: "+tt.provides+"\n"))
			got, err := c.Provides()
			require.Error(t, err)
			assert.Nil(t, got, "no partial set is returned")
			assert.Contains(t, err.Error(), "provides")
		})
	}

	// The shared #transformers check: core guards each demand map with
	// `!= _|_`, so a #transformers that does not evaluate could leave a
	// well-formed `provides` behind. The decode path must still report it.
	t.Run("a #transformers that does not evaluate is reported over a present field", func(t *testing.T) {
		// newCatalog refuses a value that does not evaluate, so build it here.
		v := cuecontext.New().CompileString(catalogBody("#transformers: x: kind: 1 & 2\nprovides: []\n"))
		c, err := catalog.NewCatalogFromValue(v)
		require.NoError(t, err)
		got, err := c.Provides()
		require.Error(t, err)
		assert.Nil(t, got, "no partial set is returned")
		assert.Contains(t, err.Error(), "#transformers did not evaluate")
	})
}

// providerModFile is a catalog module file committing to core at version,
// or to no core at all when version is empty.
func providerModFile(version string) string {
	f := "module: \"test.example/catalogs/provider@v1\"\nlanguage: version: \"v0.17.0\"\n"
	if version != "" {
		f += "deps: \"opmodel.dev/core@v2\": v: \"" + version + "\"\n"
	}
	return f
}

// withModFile wraps a catalog value in an on-disk source tree whose committed
// module file is modFile, the way an acquire verb stamps one.
func withModFile(t *testing.T, body, modFile string) *catalog.Catalog {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "cue.mod"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "cue.mod", "module.cue"), []byte(modFile), 0o644))
	c := newCatalog(t, body)
	c.Source = &catalog.Source{Root: root}
	return c
}

// catalog-acquisition spec, "A catalog built against an older core falls
// back": when the catalog carries a Source, its committed core pin decides
// the path. An older core does not derive `provides`, so one authored beside
// an embedded #Catalog is not read; from schema.ProvidesSince on the field
// is. Offline: the values are literals, only the module file is on disk.
func TestCatalog_Provides_CorePinDecidesThePath(t *testing.T) {
	// The transformers provide backupTrait; the authored field says
	// restoreTrait, so the answer names the path that ran.
	body := catalogBody(`#transformers: "` + scheduleImpl + `": requiredTraits: "` + backupTrait + `": fulfilment: "provider"
provides: ["` + restoreTrait + `"]
`)

	// v2.0.0-beta.2 is the release before ProvidesSince; the literal is the
	// point (.cascade-frozen).
	const olderCore = "v2.0.0-beta.2"

	tests := []struct {
		name    string
		modFile string
		want    []string
	}{
		{name: "an older core pin runs the fallback even over an authored field", modFile: providerModFile(olderCore), want: []string{backupTrait}},
		{name: "a core pin at ProvidesSince decodes the field", modFile: providerModFile("v" + schema.ProvidesSince), want: []string{restoreTrait}},
		{name: "a module file with no core requirement decides by presence", modFile: providerModFile(""), want: []string{restoreTrait}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := withModFile(t, body, tt.modFile)
			got, err := c.Provides()
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	// The module file reader already refuses a core version that is not
	// canonical SemVer, so this case fails inside Requires; the version
	// comparison after it is a defensive branch no module file reaches.
	t.Run("a module file whose core pin is not a version is reported", func(t *testing.T) {
		c := withModFile(t, body, providerModFile("not-a-version"))
		got, err := c.Provides()
		require.Error(t, err)
		assert.Nil(t, got, "no partial set is returned")
		assert.Contains(t, err.Error(), "core pin")
	})

	t.Run("a committed module file that cannot be read is reported", func(t *testing.T) {
		c := withModFile(t, body, "module: [")
		got, err := c.Provides()
		require.Error(t, err)
		assert.Nil(t, got, "no partial set is returned")
		assert.Contains(t, err.Error(), "core pin")
	})
}
