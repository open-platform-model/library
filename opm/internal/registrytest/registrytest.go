// Package registrytest provides an in-memory OCI registry harness for tests
// that need to materialize catalogs without a live registry.
//
// It stands up a [modregistrytest] registry serving inline `c.#Catalog`
// fixtures under the [CatalogPrefix] module path, while opmodel.dev/core@v1
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
// while opmodel.dev (core@v1) still resolves from the public registry / warm
// workspace cache.
const CatalogPrefix = "test.example"

// CatalogFixture is one (path, version) catalog module published into the
// in-memory registry. Body is the catalog package body that follows the bare
// `c.#Catalog` line (see [BuildCatalog]).
type CatalogFixture struct {
	Path    string // module path without the @major suffix, e.g. "test.example/x/cat"
	Version string // bare SemVer, e.g. "0.1.0"
	Body    string // catalog package body (metadata + #transformers)

	// CoreVersion pins the opmodel.dev/core@v1 dependency this catalog's
	// cue.mod/module.cue declares. Empty defaults to defaultCoreVersion
	// ("v1.0.0-alpha.1"), so existing callers are unaffected. Tests exercising the
	// author-supplied-#Module-identity mechanism pin a later version (e.g.
	// "v0.5.0", or the "v0.4.0" self-cycle boundary for a negative control).
	// core still resolves from the public registry / warm workspace cache.
	CoreVersion string
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

// ModuleFixture is one (path, version) #Module published into the in-memory
// registry. File is the full module CUE file content (package clause + imports +
// the c.#Module embed and author-set metadata); Deps lists any module deps
// BEYOND opmodel.dev/core@v1 that File imports (e.g. a catalog the module
// references), keyed by major-qualified path → bare SemVer. See [BuildModuleFile].
type ModuleFixture struct {
	Path    string            // module path without @major, e.g. "test.example/x/modules/hello"
	Version string            // bare SemVer, e.g. "0.0.2"
	File    string            // full module.cue contents
	Deps    map[string]string // extra deps: "<path>@vN" → bare SemVer (core is added automatically)

	// CoreVersion pins the opmodel.dev/core@v1 dependency this module's
	// cue.mod/module.cue declares. Empty defaults to defaultCoreVersion
	// ("v1.0.0-alpha.1"), so existing callers are unaffected. Tests exercising the
	// author-supplied-#Module-identity mechanism pin a later version (e.g.
	// "v0.5.0", or the "v0.4.0" self-cycle boundary for a negative control).
	// core still resolves from the public registry / warm workspace cache.
	CoreVersion string
}

// defaultCoreVersion is the opmodel.dev/core@v1 version registrytest fixtures
// declare when ModuleFixture.CoreVersion / CatalogFixture.CoreVersion are
// empty. It preserves the historical pin so callers predating the override are
// unaffected.
const defaultCoreVersion = "v1.0.0-alpha.1"

// coreVersionOr returns v normalized to a leading "v", or defaultCoreVersion
// when v is empty.
func coreVersionOr(v string) string {
	if v == "" {
		return defaultCoreVersion
	}
	return "v" + strings.TrimPrefix(v, "v")
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
// (module + language version + the opmodel.dev/core@v1 dep) and catalog.cue
// (package body importing core and unifying c.#Catalog).
func NewCatalogRegistry(t *testing.T, fixtures ...CatalogFixture) string {
	t.Helper()

	mapfs := fstest.MapFS{}
	addCatalogs(mapfs, fixtures...)
	return buildRegistry(t, mapfs)
}

// NewModuleRegistry stands up an in-memory OCI registry serving the given module
// AND catalog fixtures from one host, configuring CUE_REGISTRY / CUE_CACHE_DIR
// exactly like [NewCatalogRegistry]. Catalogs published here are resolvable as
// transitive deps of the modules. Returns the CUE_REGISTRY mapping string.
func NewModuleRegistry(t *testing.T, modules []ModuleFixture, catalogs []CatalogFixture) string {
	t.Helper()

	mapfs := fstest.MapFS{}
	addCatalogs(mapfs, catalogs...)
	addModules(mapfs, modules...)
	return buildRegistry(t, mapfs)
}

// addCatalogs writes the modregistrytest fixture files for each catalog into
// mapfs.
func addCatalogs(mapfs fstest.MapFS, fixtures ...CatalogFixture) {
	for _, f := range fixtures {
		dir := strings.ReplaceAll(f.Path, "/", "_") + "_v" + f.Version
		pkg := f.Path[strings.LastIndex(f.Path, "/")+1:]
		mapfs[dir+"/cue.mod/module.cue"] = &fstest.MapFile{Data: fmt.Appendf(nil,
			"module: %q\nlanguage: version: \"v0.17.0-alpha.1\"\ndeps: \"opmodel.dev/core@v1\": v: %q\n",
			f.Path+"@v0", coreVersionOr(f.CoreVersion),
		)}
		mapfs[dir+"/catalog.cue"] = &fstest.MapFile{Data: []byte(
			"package " + pkg + "\n\nimport c \"opmodel.dev/core@v1\"\n\nc.#Catalog\n" + f.Body,
		)}
	}
}

// addModules writes the modregistrytest fixture files for each module into
// mapfs. Each module's cue.mod/module.cue declares opmodel.dev/core@v1 plus any
// extra Deps; the module body itself is the fixture's File verbatim.
func addModules(mapfs fstest.MapFS, modules ...ModuleFixture) {
	for _, m := range modules {
		dir := strings.ReplaceAll(m.Path, "/", "_") + "_v" + m.Version
		var deps strings.Builder
		fmt.Fprintf(&deps, "deps: \"opmodel.dev/core@v1\": v: %q\n", coreVersionOr(m.CoreVersion))
		for p, v := range m.Deps {
			fmt.Fprintf(&deps, "deps: %q: v: %q\n", p, "v"+strings.TrimPrefix(v, "v"))
		}
		mapfs[dir+"/cue.mod/module.cue"] = &fstest.MapFile{Data: fmt.Appendf(nil,
			"module: %q\nlanguage: version: \"v0.17.0-alpha.1\"\n%s",
			m.Path+"@v0", deps.String(),
		)}
		mapfs[dir+"/module.cue"] = &fstest.MapFile{Data: []byte(m.File)}
	}
}

// buildRegistry stands up the in-memory registry from mapfs and wires the test
// environment (warm core cache + in-process host for the test prefix). Returns
// the CUE_REGISTRY mapping string.
func buildRegistry(t *testing.T, mapfs fstest.MapFS) string {
	t.Helper()

	reg, err := modregistrytest.New(mapfs, "")
	require.NoError(t, err, "stand up in-memory registry")
	t.Cleanup(reg.Close)

	// SetEnv points CUE_CACHE_DIR at the warm workspace cache (core@v1
	// already extracted there) and seeds CUE_REGISTRY with PublicRegistry;
	// the combined mapping below adds the in-process host.
	schematest.SetEnv(t)
	registry := CatalogPrefix + "=" + reg.Host() + "+insecure," + schema.PublicRegistry
	t.Setenv("CUE_REGISTRY", registry)
	return registry
}

// BuildModuleFile renders a complete module.cue for a #Module that imports the
// core schema and (optionally) a single catalog, setting the author-given
// identity metadata. When catalogImport is non-empty the module imports that
// major-qualified catalog path and references its metadata under debugValues
// (an open field), forcing the loader to resolve the catalog as a transitive
// dependency. pkg is the package clause name.
func BuildModuleFile(pkg, name, modulePath, catalogImport string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "package %s\n\n", pkg)
	if catalogImport == "" {
		b.WriteString("import c \"opmodel.dev/core@v1\"\n\n")
	} else {
		b.WriteString("import (\n\tc \"opmodel.dev/core@v1\"\n")
		fmt.Fprintf(&b, "\tcat %q\n)\n\n", catalogImport)
	}
	b.WriteString("c.#Module\n")
	fmt.Fprintf(&b, "metadata: {\n\tname:       %q\n\tmodulePath: %q\n\tversion:    \"0.0.2\"\n}\n", name, modulePath)
	if catalogImport != "" {
		// Reference the imported catalog so the dep is load-bearing; debugValues
		// is an open field on #Module, so this does not trip closedness.
		b.WriteString("debugValues: catalogModulePath: cat.metadata.modulePath\n")
	}
	return b.String()
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
