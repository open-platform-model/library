package materialize

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"cuelang.org/go/cue"
	"cuelang.org/go/mod/modregistrytest"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/internal/schematest"
	"github.com/open-platform-model/library/opm/platform"
	"github.com/open-platform-model/library/opm/schema"
)

// testCatalogPrefix is the module-path prefix every in-memory catalog fixture
// lives under. The CUE_REGISTRY mapping routes this prefix to the in-process
// registry while opmodel.dev (core@v0) still resolves from the public
// registry / warm workspace cache.
const testCatalogPrefix = "test.example"

// catalogFixture is one (path, version) catalog module published into the
// in-memory registry. Body is the catalog package body that follows the bare
// `c.#Catalog` line (see buildCatalog).
type catalogFixture struct {
	Path    string // module path without the @major suffix, e.g. "test.example/x/cat"
	Version string // bare SemVer, e.g. "0.1.0"
	Body    string // catalog package body (metadata + #transformers)
}

// uniquePath returns a globally-unique catalog module path for the current
// test. Uniqueness matters because all tests share the warm workspace CUE
// module cache (download cache keyed by module path + version): distinct
// paths prevent one test's fixture content from shadowing another's.
func uniquePath(t *testing.T, leaf string) string {
	t.Helper()
	s := strings.ToLower(t.Name())
	s = strings.NewReplacer("/", "-", "_", "-").Replace(s)
	return testCatalogPrefix + "/" + s + "/" + leaf
}

// newCatalogRegistry stands up an in-memory OCI registry serving the given
// catalog fixtures and configures CUE_REGISTRY / CUE_CACHE_DIR for the test
// scope: the test prefix routes to the in-process host (+insecure), while
// opmodel.dev/core resolves from the public registry via the warm workspace
// cache. Returns the CUE_REGISTRY mapping string. The registry is torn down
// at test end.
//
// Fixture layout follows modregistrytest.New: one directory per (module,
// version) named "<path with / → _>_v<X.Y.Z>", each holding cue.mod/module.cue
// (module + language version + the opmodel.dev/core@v0 dep) and catalog.cue
// (package body importing core and unifying c.#Catalog).
func newCatalogRegistry(t *testing.T, fixtures ...catalogFixture) string {
	t.Helper()

	mapfs := fstest.MapFS{}
	for _, f := range fixtures {
		dir := strings.ReplaceAll(f.Path, "/", "_") + "_v" + f.Version
		pkg := f.Path[strings.LastIndex(f.Path, "/")+1:]
		mapfs[dir+"/cue.mod/module.cue"] = &fstest.MapFile{Data: []byte(fmt.Sprintf(
			"module: %q\nlanguage: version: \"v0.16.0\"\ndeps: \"opmodel.dev/core@v0\": v: \"v0.3.0\"\n",
			f.Path+"@v0",
		))}
		mapfs[dir+"/catalog.cue"] = &fstest.MapFile{Data: []byte(
			"package " + pkg + "\n\nimport c \"opmodel.dev/core@v0\"\n\nc.#Catalog\n" + f.Body,
		)}
	}

	reg, err := modregistrytest.New(mapfs, "")
	require.NoError(t, err, "stand up in-memory catalog registry")
	t.Cleanup(reg.Close)

	// SetEnv points CUE_CACHE_DIR at the warm workspace cache (core@v0
	// already extracted there) and seeds CUE_REGISTRY with PublicRegistry;
	// the combined mapping below adds the in-process catalog host.
	schematest.SetEnv(t)
	registry := testCatalogPrefix + "=" + reg.Host() + "+insecure," + schema.PublicRegistry
	t.Setenv("CUE_REGISTRY", registry)
	return registry
}

// buildCatalog renders a complete catalog package body (the text after the
// bare `c.#Catalog` line) for the given module path/version and transformer
// fixtures. The #Catalog pattern stamps each transformer's metadata.modulePath
// ("<path>/transformers") and version; this only authors name, description,
// the required-primitive maps, and a trivial transform.
func buildCatalog(path, version string, txs ...txFixture) string {
	var b strings.Builder
	fmt.Fprintf(&b, "metadata: {\n\tmodulePath:  %q\n\tversion:     %q\n\tdescription: \"test catalog\"\n}\n", path, version)
	b.WriteString("#transformers: {\n")
	for _, tx := range txs {
		fqn := fmt.Sprintf("%s/transformers/%s@%s", path, tx.Name, version)
		fmt.Fprintf(&b, "\t%q: {\n", fqn)
		b.WriteString("\t\tkind: \"ComponentTransformer\"\n")
		fmt.Fprintf(&b, "\t\tmetadata: {\n\t\t\tname:        %q\n\t\t\tdescription: %q\n\t\t}\n", tx.Name, tx.Name+" transformer")
		if len(tx.Resources) > 0 {
			b.WriteString("\t\trequiredResources: {\n")
			for _, r := range tx.Resources {
				rfqn := fmt.Sprintf("%s/resources/%s@%s", path, r, version)
				fmt.Fprintf(&b, "\t\t\t%q: {\n", rfqn)
				b.WriteString("\t\t\t\tkind: \"Resource\"\n")
				fmt.Fprintf(&b, "\t\t\t\tmetadata: {name: %q, modulePath: %q, version: %q}\n", r, path+"/resources", version)
				fmt.Fprintf(&b, "\t\t\t\tspec: %s: _\n", r)
				b.WriteString("\t\t\t}\n")
			}
			b.WriteString("\t\t}\n")
		}
		if len(tx.Traits) > 0 {
			b.WriteString("\t\trequiredTraits: {\n")
			for _, tr := range tx.Traits {
				trfqn := fmt.Sprintf("%s/traits/%s@%s", path, tr, version)
				fmt.Fprintf(&b, "\t\t\t%q: {\n", trfqn)
				b.WriteString("\t\t\t\tkind: \"Trait\"\n")
				fmt.Fprintf(&b, "\t\t\t\tmetadata: {name: %q, modulePath: %q, version: %q}\n", tr, path+"/traits", version)
				fmt.Fprintf(&b, "\t\t\t\tspec: %s: _\n", tr)
				b.WriteString("\t\t\t\tappliesTo: []\n")
				b.WriteString("\t\t\t}\n")
			}
			b.WriteString("\t\t}\n")
		}
		b.WriteString("\t\t#transform: output: {}\n")
		b.WriteString("\t}\n")
	}
	b.WriteString("}\n")
	return b.String()
}

// txFixture describes one transformer to author into a test catalog: its
// kebab name plus the short names of the resources/traits it requires (used
// to populate the #matchers reverse index).
type txFixture struct {
	Name      string
	Resources []string
	Traits    []string
}

// ctxOwner is a minimal CueContextOwner wrapping a *cue.Context, so tests can
// drive Materialize without constructing a full *kernel.Kernel (which would
// import materialize and create a cycle).
type ctxOwner struct{ ctx *cue.Context }

func (o ctxOwner) CueContext() *cue.Context { return o.ctx }

// buildPlatform builds a concrete *platform.Platform whose #registry contains
// the given map body (e.g. `{ "test.example/.../cat": {enable: true} }`),
// validated against core's #Platform. The platform value is built with octx
// so Materialize can fill catalog values (built with the same context) onto
// it. CUE_REGISTRY / CUE_CACHE_DIR must already be configured (e.g. by
// newCatalogRegistry) so #Platform resolves from the warm workspace cache.
func buildPlatform(t *testing.T, octx *cue.Context, registryBody string) *platform.Platform {
	t.Helper()
	cache := &schema.Cache{Loader: schema.OCILoader{}}
	schemaVal, err := cache.Get(octx)
	require.NoError(t, err, "load core schema")

	def := schemaVal.LookupPath(cue.ParsePath("#Platform"))
	require.True(t, def.Exists(), "#Platform definition must exist")

	concrete := octx.CompileString(`{
		kind: "Platform"
		metadata: name: "test"
		type: "kubernetes"
		#registry: ` + registryBody + `
	}`)
	require.NoError(t, concrete.Err())

	pv := def.Unify(concrete)
	require.NoError(t, pv.Validate(cue.Concrete(false)), "platform must validate against #Platform")

	p, err := platform.NewPlatformFromValue(ctxOwner{octx}, pv)
	require.NoError(t, err)
	return p
}
