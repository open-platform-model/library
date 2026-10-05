package catalog_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/catalog"
	"github.com/open-platform-model/library/opm/internal/modversion"
	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/schema"
)

// requireRegistry skips a test that resolves core from GHCR under -short or
// when GHCR is unreachable, the gate the kernel's registry-backed tests use.
// OPM_FLOW_TEST_FORCE=1 makes an unreachable registry a failure instead.
func requireRegistry(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("resolves core from GHCR; skipped under -short")
	}
	if os.Getenv("OPM_FLOW_TEST_FORCE") == "1" {
		return
	}
	conn, err := net.DialTimeout("tcp", "ghcr.io:443", 500*time.Millisecond)
	if err != nil {
		t.Skipf("GHCR not reachable (%v); this test resolves core from ghcr.io. Set OPM_FLOW_TEST_FORCE=1 to require it", err)
	}
	_ = conn.Close()
}

const (
	parityCatalogPath = "test.example/catalogs/provider"
	parityCatalogVer  = "1.0.0"
	parityBasePath    = "test.example/catalogs/base"
)

// parityDemand is one entry of a transformer's demand map. The arm names the
// map (requiredTraits, requiredResources, optionalTraits, optionalResources);
// an empty fulfilment authors none, so core's "catalog" default applies.
type parityDemand struct {
	arm, name, fulfilment string
}

type parityTransformer struct {
	name    string
	labels  bool // author requiredLabels, for a transformer with no demand map
	demands []parityDemand
}

// contractFQN keys a demand by the contract's own metadata.fqn, so the core
// release that binds member keys to metadata.fqn (ADR-013, decision j3) leaves
// it valid.
func contractFQN(arm, name string) string {
	kind := "traits"
	if strings.HasSuffix(arm, "Resources") {
		kind = "resources"
	}
	return parityBasePath + "/" + kind + "/" + name + "@v1"
}

// parityCatalogBody renders a #Catalog body from transformers, in the shape
// core's #ComponentTransformer, #Trait and #Resource accept.
func parityCatalogBody(txs []parityTransformer) string {
	var b strings.Builder
	fmt.Fprintf(&b, "metadata: {\n\tmodulePath:  %q\n\tversion:     %q\n\tdescription: \"parity provider catalog\"\n}\n",
		parityCatalogPath+"@"+modversion.Major(parityCatalogVer), parityCatalogVer)
	if len(txs) == 0 {
		return b.String()
	}
	b.WriteString("#transformers: {\n")
	for _, tx := range txs {
		impl := fmt.Sprintf("%s/transformers/%s@%s", parityCatalogPath, tx.name, parityCatalogVer)
		fmt.Fprintf(&b, "\t%q: {\n\t\tkind: \"ComponentTransformer\"\n", impl)
		fmt.Fprintf(&b, "\t\tmetadata: {name: %q, description: %q, fqn: %q}\n", tx.name, tx.name+" transformer", impl)
		if tx.labels {
			b.WriteString("\t\trequiredLabels: \"opm.test/workload\": \"stateless\"\n")
		}
		for _, d := range tx.demands {
			fqn := contractFQN(d.arm, d.name)
			kind, segment := "Trait", "traits"
			if strings.HasSuffix(d.arm, "Resources") {
				kind, segment = "Resource", "resources"
			}
			fmt.Fprintf(&b, "\t\t%s: %q: {\n\t\t\tkind: %q\n", d.arm, fqn, kind)
			fmt.Fprintf(&b, "\t\t\tmetadata: {name: %q, modulePath: %q, apiVersion: \"v1\", catalogVersion: \"1.0.0\", fqn: %q}\n",
				d.name, parityBasePath+"/"+segment, fqn)
			if d.fulfilment != "" {
				fmt.Fprintf(&b, "\t\t\tfulfilment: %q\n", d.fulfilment)
			}
			fmt.Fprintf(&b, "\t\t\tspec: %s: _\n", d.name)
			if kind == "Trait" {
				b.WriteString("\t\t\tappliesTo: []\n")
			}
			b.WriteString("\t\t}\n")
		}
		b.WriteString("\t\t#transform: output: {}\n\t}\n")
	}
	b.WriteString("}\n")
	return b.String()
}

// writeParityCatalogDir lays out a one-package catalog module that embeds
// c.#Catalog, commits to core at coreVersion, and holds body.
func writeParityCatalogDir(t *testing.T, coreVersion, body string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cue.mod"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cue.mod", "module.cue"), fmt.Appendf(nil,
		"module: %q\nlanguage: version: \"v0.17.0\"\ndeps: \"opmodel.dev/core@v2\": v: %q\n",
		parityCatalogPath+"@"+modversion.Major(parityCatalogVer), coreVersion), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "catalog.cue"), fmt.Appendf(nil,
		"package provider\n\nimport c \"opmodel.dev/core@v2\"\n\nc.#Catalog\n%s", body), 0o644))
	return dir
}

// assertParity requires the catalog to carry core's provides, already sorted
// and free of duplicates. It asserts that core's field equals the deprecated
// fold over the same #transformers, and that Provides returns the field, so a
// Provides that never decodes cannot pass by comparing the fold with itself.
// It returns the answer.
func assertParity(t *testing.T, cat *catalog.Catalog) []string {
	t.Helper()
	field := cat.Package.LookupPath(schema.CatalogProvides)
	require.True(t, field.Exists(), "a catalog built against the pinned core carries provides")
	var raw []string
	require.NoError(t, field.Decode(&raw))
	require.True(t, sort.StringsAreSorted(raw), "core sorts provides: %v", raw)
	for i := 1; i < len(raw); i++ {
		require.NotEqual(t, raw[i-1], raw[i], "core deduplicates provides: %v", raw)
	}

	got, err := cat.Provides()
	require.NoError(t, err)
	fold, err := catalog.ProvidesFold(cat)
	require.NoError(t, err)
	require.NotNil(t, raw, "core's provides decodes to a list")
	assert.Equal(t, fold, raw, "core's provider set and the fallback agree element for element")
	assert.Equal(t, raw, got, "Provides returns core's provider set")

	// On core's own output the field and the fold are equal, so the checks
	// above cannot tell a Provides that decodes from one that always folds.
	// The same catalog with its #transformers dropped can: under its real
	// committed core pin, Provides must still answer core's field.
	stripped, err := catalog.NewCatalogFromValue(cuecontext.New().CompileString("{}").
		FillPath(schema.Metadata, cat.Package.LookupPath(schema.Metadata)).
		FillPath(schema.CatalogProvides, field))
	require.NoError(t, err)
	stripped.Source = cat.Source
	fromField, err := stripped.Provides()
	require.NoError(t, err)
	assert.Equal(t, raw, fromField, "Provides reads core's field, not #transformers")
	return got
}

// catalog-acquisition spec, "Core's provider set and the fallback agree"
// (ADR-012's parity test): for catalogs built against the core the library
// pins, the decoded provides equals the fold.
func TestCatalog_Provides_ParityWithFold(t *testing.T) {
	requireRegistry(t)
	t.Setenv("CUE_REGISTRY", schema.PublicRegistry)
	ctx := context.Background()

	backup := func(arm, f string) parityDemand { return parityDemand{arm: arm, name: "backup", fulfilment: f} }
	restore := func(arm, f string) parityDemand { return parityDemand{arm: arm, name: "restore", fulfilment: f} }
	volumes := func(arm, f string) parityDemand { return parityDemand{arm: arm, name: "volumes", fulfilment: f} }
	container := func(arm, f string) parityDemand { return parityDemand{arm: arm, name: "container", fulfilment: f} }

	// The unit table's shapes, rebuilt as real #Catalog artifacts.
	shapes := []struct {
		name string
		txs  []parityTransformer
		want []string
	}{
		{
			name: "provider-fulfilled demands from both maps",
			txs: []parityTransformer{{name: "schedule", demands: []parityDemand{
				restore("requiredTraits", "provider"), backup("requiredTraits", "provider"), volumes("requiredResources", "provider"),
			}}},
			want: []string{contractFQN("requiredResources", "volumes"), contractFQN("requiredTraits", "backup"), contractFQN("requiredTraits", "restore")},
		},
		{
			name: "a catalog-fulfilled demand is not collected",
			txs: []parityTransformer{{name: "schedule", demands: []parityDemand{
				backup("requiredTraits", "provider"), container("requiredResources", "catalog"),
			}}},
			want: []string{contractFQN("requiredTraits", "backup")},
		},
		{
			name: "no provider-fulfilled demand derives the empty set",
			txs: []parityTransformer{{name: "schedule", demands: []parityDemand{
				container("requiredResources", "catalog"), restore("requiredTraits", "catalog"),
			}}},
			want: []string{},
		},
		{name: "no transformers derives the empty set", want: []string{}},
		{
			name: "a transformer with no demand maps contributes nothing",
			txs:  []parityTransformer{{name: "schedule", labels: true}},
			want: []string{},
		},
		{
			// Adjusted: through core an unauthored fulfilment defaults to
			// "catalog" rather than being absent; the case keeps "not
			// collected".
			name: "an unauthored fulfilment takes core's catalog default",
			txs: []parityTransformer{{name: "schedule", demands: []parityDemand{
				container("requiredResources", ""), backup("requiredTraits", "provider"),
			}}},
			want: []string{contractFQN("requiredTraits", "backup")},
		},
		{
			name: "one contract required by two transformers appears once",
			txs: []parityTransformer{
				{name: "schedule", demands: []parityDemand{backup("requiredTraits", "provider")}},
				{name: "prebackup", demands: []parityDemand{backup("requiredTraits", "provider")}},
			},
			want: []string{contractFQN("requiredTraits", "backup")},
		},
		{
			name: "an optional demand is tolerance, not fulfilment",
			txs: []parityTransformer{{name: "schedule", demands: []parityDemand{
				backup("optionalTraits", "provider"), volumes("optionalResources", "provider"),
			}}},
			want: []string{},
		},
	}

	t.Run("unit shapes", func(t *testing.T) {
		k := kernel.New()
		for _, tt := range shapes {
			t.Run(tt.name, func(t *testing.T) {
				dir := writeParityCatalogDir(t, registrytest.DefaultCoreVersion, parityCatalogBody(tt.txs))
				cat, err := k.AcquireCatalogFromDir(ctx, dir)
				require.NoError(t, err)
				assert.Equal(t, tt.want, assertParity(t, cat))
			})
		}
	})

	t.Run("render registry catalogs", func(t *testing.T) {
		const prefix = "testing.opmodel.dev/library-render"
		root := filepath.Join("..", "..", "testdata", "render", "registry")
		mapping := registrytest.NewRegistryFromDir(t, root, prefix)
		k := kernel.New(kernel.WithRegistry(mapping))

		entries, err := os.ReadDir(root)
		require.NoError(t, err)
		moduleLine := regexp.MustCompile(`(?m)^module:\s*"([^"]+)"`)
		embeds := regexp.MustCompile(`(?m)^c\.#Catalog\s*$`)

		var catalogs, nonEmpty int
		for _, e := range entries {
			if !e.IsDir() || !isCatalogDir(t, filepath.Join(root, e.Name()), embeds) {
				continue
			}
			modFile, err := os.ReadFile(filepath.Join(root, e.Name(), "cue.mod", "module.cue"))
			require.NoError(t, err)
			m := moduleLine.FindSubmatch(modFile)
			require.NotNil(t, m, "%s names its module", e.Name())
			version := e.Name()[strings.LastIndex(e.Name(), "_v")+1:]

			catalogs++
			t.Run(e.Name(), func(t *testing.T) {
				cat, err := k.AcquireCatalogFromRegistry(ctx, string(m[1]), version)
				require.NoError(t, err)
				if len(assertParity(t, cat)) > 0 {
					nonEmpty++
				}
			})
		}
		require.Positive(t, catalogs, "the render registry serves catalogs")
		require.Positive(t, nonEmpty, "at least one render registry catalog provides a contract, so the fixtures cover more than the empty set")
	})
}

// isCatalogDir reports whether the module tree at dir has a top-level .cue
// file embedding c.#Catalog.
func isCatalogDir(t *testing.T, dir string, embeds *regexp.Regexp) bool {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*.cue"))
	require.NoError(t, err)
	for _, f := range files {
		src, err := os.ReadFile(f)
		require.NoError(t, err)
		if embeds.Match(src) {
			return true
		}
	}
	return false
}

// catalog-acquisition spec, "A catalog built against an older core falls
// back": a catalog committing to the core release before
// schema.ProvidesSince carries no derived provides, and Provides answers it
// through the fold. One it authors beside its embedded #Catalog, which an
// older core admits, is not read.
func TestCatalog_Provides_OldCatalogFallsBackToFold(t *testing.T) {
	requireRegistry(t)
	t.Setenv("CUE_REGISTRY", schema.PublicRegistry)
	ctx := context.Background()

	// The release before ProvidesSince; the literal is the point
	// (.cascade-frozen).
	const olderCore = "v2.0.0-beta.2"
	cmp, err := modversion.Compare(olderCore, "v"+schema.ProvidesSince)
	require.NoError(t, err)
	require.Negative(t, cmp, "the fixture's core predates schema.ProvidesSince")

	body := parityCatalogBody([]parityTransformer{{name: "schedule", demands: []parityDemand{
		{arm: "requiredTraits", name: "backup", fulfilment: "provider"},
		{arm: "requiredResources", name: "container", fulfilment: "catalog"},
	}}})
	want := []string{contractFQN("requiredTraits", "backup")}
	k := kernel.New()

	t.Run("an older catalog carries no provides", func(t *testing.T) {
		cat, err := k.AcquireCatalogFromDir(ctx, writeParityCatalogDir(t, olderCore, body))
		require.NoError(t, err)
		assert.False(t, cat.Package.LookupPath(schema.CatalogProvides).Exists(), "core before ProvidesSince derives no provides")

		got, err := cat.Provides()
		require.NoError(t, err)
		assert.Equal(t, want, got)
		fold, err := catalog.ProvidesFold(cat)
		require.NoError(t, err)
		assert.Equal(t, fold, got)
	})

	t.Run("an authored provides on an older core is not read", func(t *testing.T) {
		bogus := parityBasePath + "/traits/bogus@v1"
		cat, err := k.AcquireCatalogFromDir(ctx, writeParityCatalogDir(t, olderCore, body+fmt.Sprintf("provides: [%q]\n", bogus)))
		require.NoError(t, err)
		var authored []string
		require.NoError(t, cat.Package.LookupPath(schema.CatalogProvides).Decode(&authored),
			"an embedding admits a field declared beside it")
		assert.Equal(t, []string{bogus}, authored)

		got, err := cat.Provides()
		require.NoError(t, err)
		assert.Equal(t, want, got, "the committed core pin decides, so the fold answers")
	})
}
