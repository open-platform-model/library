// Package registrytest provides an in-memory OCI registry harness for tests
// that need to materialize catalogs without a live registry.
//
// It stands up a [modregistrytest] registry serving inline `c.#Catalog`
// fixtures under the [CatalogPrefix] module path, while opmodel.dev/core@v0
// still resolves from the warm workspace cache (via
// [schematest.SetEnv]). The CUE_REGISTRY mapping routes the test prefix to the
// in-process host and leaves every other path on the public registry.
//
// It lives under opm/internal/ so it stays out of the library's public SemVer
// surface (kernel neutrality) while remaining importable from any opm/* test
// package. The materialize tests and the kernel integration harness share it
// so registry semantics never drift between them.
package registrytest

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

// CatalogPrefix is the module-path prefix every in-memory catalog fixture lives
// under. The CUE_REGISTRY mapping routes this prefix to the in-process registry
// while opmodel.dev (core@v0) still resolves from the public registry / warm
// workspace cache.
const CatalogPrefix = "test.example"

// CatalogFixture is one (path, version) catalog module published into the
// in-memory registry. Body is the catalog package body that follows the bare
// `c.#Catalog` line (see [BuildCatalog]).
type CatalogFixture struct {
	Path    string // module path without the @major suffix, e.g. "test.example/x/cat"
	Version string // bare SemVer, e.g. "0.1.0"
	Body    string // catalog package body (metadata + #transformers)
}

// TxFixture describes one transformer to author into a test catalog: its kebab
// name plus the short names of the resources/traits it requires (used to
// populate the #matchers reverse index). Output is an optional inline
// `#transform.output` literal (a CUE struct or list expression); when empty it
// defaults to an empty struct.
type TxFixture struct {
	Name      string
	Resources []string
	Traits    []string
	Output    string // optional inline #transform.output literal; "" → "{}"
}

// UniquePath returns a globally-unique catalog module path for the current
// test. Uniqueness matters because all tests share the warm workspace CUE
// module cache (download cache keyed by module path + version): distinct paths
// prevent one test's fixture content from shadowing another's.
func UniquePath(t *testing.T, leaf string) string {
	t.Helper()
	s := strings.ToLower(t.Name())
	s = strings.NewReplacer("/", "-", "_", "-").Replace(s)
	return CatalogPrefix + "/" + s + "/" + leaf
}

// NewCatalogRegistry stands up an in-memory OCI registry serving the given
// catalog fixtures and configures CUE_REGISTRY / CUE_CACHE_DIR for the test
// scope: the test prefix routes to the in-process host (+insecure), while
// opmodel.dev/core resolves from the public registry via the warm workspace
// cache. Returns the CUE_REGISTRY mapping string. The registry is torn down at
// test end.
//
// Fixture layout follows modregistrytest.New: one directory per (module,
// version) named "<path with / → _>_v<X.Y.Z>", each holding cue.mod/module.cue
// (module + language version + the opmodel.dev/core@v0 dep) and catalog.cue
// (package body importing core and unifying c.#Catalog).
func NewCatalogRegistry(t *testing.T, fixtures ...CatalogFixture) string {
	t.Helper()

	mapfs := fstest.MapFS{}
	for _, f := range fixtures {
		dir := strings.ReplaceAll(f.Path, "/", "_") + "_v" + f.Version
		pkg := f.Path[strings.LastIndex(f.Path, "/")+1:]
		mapfs[dir+"/cue.mod/module.cue"] = &fstest.MapFile{Data: fmt.Appendf(nil,
			"module: %q\nlanguage: version: \"v0.16.0\"\ndeps: \"opmodel.dev/core@v0\": v: \"v0.3.0\"\n",
			f.Path+"@v0",
		)}
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
	registry := CatalogPrefix + "=" + reg.Host() + "+insecure," + schema.PublicRegistry
	t.Setenv("CUE_REGISTRY", registry)
	return registry
}

// BuildCatalog renders a complete catalog package body (the text after the bare
// `c.#Catalog` line) for the given module path/version and transformer
// fixtures. The #Catalog pattern stamps each transformer's metadata.modulePath
// ("<path>/transformers") and version; this only authors name, description, the
// required-primitive maps, and the transform output (from [TxFixture.Output],
// defaulting to an empty struct).
func BuildCatalog(path, version string, txs ...TxFixture) string {
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
				fmt.Fprintf(&b, "\t\t\t\tspec: %q: _\n", specField(r))
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
				fmt.Fprintf(&b, "\t\t\t\tspec: %q: _\n", specField(tr))
				b.WriteString("\t\t\t\tappliesTo: []\n")
				b.WriteString("\t\t\t}\n")
			}
			b.WriteString("\t\t}\n")
		}
		out := strings.TrimSpace(tx.Output)
		if out == "" {
			out = "{}"
		}
		fmt.Fprintf(&b, "\t\t#transform: output: %s\n", out)
		b.WriteString("\t}\n")
	}
	b.WriteString("}\n")
	return b.String()
}

// specField returns the camelCase field name core's #Resource / #Trait require
// under `spec`. The core schema constrains `spec` to a single field named
// strings.ToCamel(#KebabToPascal(metadata.name)); for a kebab-case resource
// name like "config-maps" that is "configMaps". Mirroring it here lets
// fixtures use real multi-word primitive names without tripping the schema's
// "field not allowed" closedness check.
func specField(name string) string {
	parts := strings.Split(name, "-")
	var b strings.Builder
	for i, p := range parts {
		if p == "" {
			continue
		}
		if i == 0 {
			b.WriteString(p)
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]))
		b.WriteString(p[1:])
	}
	return b.String()
}

// CtxOwner is a minimal materialize.CueContextOwner wrapping a *cue.Context, so
// tests can drive Materialize without constructing a full *kernel.Kernel (which
// would create an import cycle through materialize).
type CtxOwner struct{ ctx *cue.Context }

// NewCtxOwner wraps ctx as a CueContextOwner.
func NewCtxOwner(ctx *cue.Context) CtxOwner { return CtxOwner{ctx: ctx} }

// CueContext returns the wrapped context.
func (o CtxOwner) CueContext() *cue.Context { return o.ctx }

// BuildPlatform builds a concrete *platform.Platform whose #registry contains
// the given map body (e.g. `{ "test.example/.../cat": {enable: true} }`),
// validated against core's #Platform. The platform value is built with octx so
// Materialize can fill catalog values (built with the same context) onto it.
// CUE_REGISTRY / CUE_CACHE_DIR must already be configured (e.g. by
// [NewCatalogRegistry]) so #Platform resolves from the warm workspace cache.
func BuildPlatform(t *testing.T, octx *cue.Context, registryBody string) *platform.Platform {
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

	p, err := platform.NewPlatformFromValue(CtxOwner{octx}, pv)
	require.NoError(t, err)
	return p
}
