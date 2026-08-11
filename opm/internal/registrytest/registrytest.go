// Package registrytest provides an in-memory OCI registry harness for tests
// that need to materialize catalogs without a live registry.
//
// It stands up a [modregistrytest] registry serving inline `c.#Catalog`
// fixtures under the [CatalogPrefix] module path, while opmodel.dev/core@v2
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
// while opmodel.dev (core@v2) still resolves from the public registry / warm
// workspace cache.
const CatalogPrefix = "test.example"

// CatalogFixture is one (path, version) catalog module published into the
// in-memory registry. Body is the catalog package body that follows the bare
// `c.#Catalog` line (see [BuildCatalog]).
type CatalogFixture struct {
	Path    string // module path without the @major suffix, e.g. "test.example/x/cat"
	Version string // bare SemVer, e.g. "0.1.0"
	Body    string // catalog package body (metadata + #transformers)

	// CoreVersion pins the opmodel.dev/core dependency this catalog's
	// cue.mod/module.cue declares — the dep line's major, the emitted core
	// import, and the generated body shape all derive from it. Empty defaults
	// to defaultCoreVersion (the v2 line); historical tests pin a v1-era
	// version explicitly. core still resolves from the public registry / warm
	// workspace cache.
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
// BEYOND opmodel.dev/core that File imports (e.g. a catalog the module
// references), keyed by major-qualified path → bare SemVer. See [BuildModuleFile].
type ModuleFixture struct {
	Path    string            // module path without @major, e.g. "test.example/x/modules/hello"
	Version string            // bare SemVer, e.g. "0.0.2"
	File    string            // full module.cue contents
	Deps    map[string]string // extra deps: "<path>@vN" → bare SemVer (core is added automatically)

	// CoreVersion pins the opmodel.dev/core dependency this module's
	// cue.mod/module.cue declares — the dep line's major and the emitted core
	// import derive from it. Empty defaults to defaultCoreVersion (the v2
	// line); historical tests pin a v1-era version explicitly. core still
	// resolves from the public registry / warm workspace cache.
	CoreVersion string
}

// defaultCoreVersion is the opmodel.dev/core version registrytest fixtures
// declare when ModuleFixture.CoreVersion / CatalogFixture.CoreVersion are
// empty. It tracks the library's default schema line (core v2, matching
// [schema.DefaultSchemaModule]); historical tests pin earlier versions
// explicitly.
const defaultCoreVersion = "v2.0.0-alpha.4"

// ContractAPIVersion is the contract level every generated v2 fixture
// primitive declares. Core v2 keys contracts by the primitive's own
// apiVersion (enhancement 0010 D4), not by the catalog's build version.
const ContractAPIVersion = "v1"

// coreVersionOr returns v normalized to a leading "v", or defaultCoreVersion
// when v is empty.
func coreVersionOr(v string) string {
	if v == "" {
		return defaultCoreVersion
	}
	return "v" + strings.TrimPrefix(v, "v")
}

// coreDep returns the major-qualified core module path for a full core version:
// "v1.0.0-alpha.1" → "opmodel.dev/core@v1". The emitted import line and the
// declared dep are both derived from it so they can never disagree on the
// major.
func coreDep(coreVersion string) string {
	return "opmodel.dev/core@" + coreMajor(coreVersion)
}

// coreMajor returns the bare major of a (normalized) core version:
// "v2.0.0-alpha.4" → "v2"; a bare major ("v2") passes through.
func coreMajor(coreVersion string) string {
	major, _, _ := strings.Cut(coreVersion, ".")
	return major
}

// coreIsV2 reports whether coreVersion sits on the v2 line or later — the
// shape watershed for generated fixture bodies (v1-era metadata carries
// `version`; v2 carries `apiVersion`/`catalogVersion`/authored `fqn`).
func coreIsV2(coreVersion string) bool {
	major := coreMajor(coreVersion)
	return major != "v0" && major != "v1"
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
// (module + language version + the opmodel.dev/core dep) and catalog.cue
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
		// The module's major suffix must match the published version's major.
		major, _, _ := strings.Cut(f.Version, ".")
		core := coreVersionOr(f.CoreVersion)
		mapfs[dir+"/cue.mod/module.cue"] = &fstest.MapFile{Data: fmt.Appendf(nil,
			"module: %q\nlanguage: version: \"v0.17.0\"\ndeps: %q: v: %q\n",
			f.Path+"@v"+major, coreDep(core), core,
		)}
		mapfs[dir+"/catalog.cue"] = &fstest.MapFile{Data: []byte(
			"package " + pkg + "\n\nimport c \"" + coreDep(core) + "\"\n\nc.#Catalog\n" + f.Body,
		)}
	}
}

// addModules writes the modregistrytest fixture files for each module into
// mapfs. Each module's cue.mod/module.cue declares opmodel.dev/core plus any
// extra Deps; the module body itself is the fixture's File verbatim.
func addModules(mapfs fstest.MapFS, modules ...ModuleFixture) {
	for _, m := range modules {
		dir := strings.ReplaceAll(m.Path, "/", "_") + "_v" + m.Version
		core := coreVersionOr(m.CoreVersion)
		var deps strings.Builder
		fmt.Fprintf(&deps, "deps: %q: v: %q\n", coreDep(core), core)
		for p, v := range m.Deps {
			fmt.Fprintf(&deps, "deps: %q: v: %q\n", p, "v"+strings.TrimPrefix(v, "v"))
		}
		// The module's major suffix must match the published version's major.
		major, _, _ := strings.Cut(m.Version, ".")
		mapfs[dir+"/cue.mod/module.cue"] = &fstest.MapFile{Data: fmt.Appendf(nil,
			"module: %q\nlanguage: version: \"v0.17.0\"\n%s",
			m.Path+"@v"+major, deps.String(),
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

	// SetEnv points CUE_CACHE_DIR at the warm workspace cache (core@v2
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
// dependency. pkg is the package clause name. The emitted core import derives
// its major from [defaultCoreVersion]; use [BuildModuleFileCore] to pin
// another core version.
func BuildModuleFile(pkg, name, modulePath, catalogImport string) string {
	return BuildModuleFileCore(defaultCoreVersion, pkg, name, modulePath, catalogImport)
}

// BuildModuleFileCore is [BuildModuleFile] with an explicit core version (a
// full version like "v1.0.0-alpha.1" or a bare major like "v1"); the emitted
// core import derives its major from it, matching the dep the fixture writer
// declares for the same CoreVersion.
//
// The metadata shape follows the core major: on v2, modulePath is the FULL
// module path (major suffix included — pass "…/modules/hello@v0") and name
// MUST be its snake_case leaf; on v1, modulePath is the major-free parent
// path (v1's metadata semantics).
func BuildModuleFileCore(coreVersion, pkg, name, modulePath, catalogImport string) string {
	core := coreVersionOr(coreVersion)
	dep := coreDep(core)
	var b strings.Builder
	fmt.Fprintf(&b, "package %s\n\n", pkg)
	if catalogImport == "" {
		fmt.Fprintf(&b, "import c %q\n\n", dep)
	} else {
		fmt.Fprintf(&b, "import (\n\tc %q\n", dep)
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
// defaulting to an empty struct). The body shape follows the catalog member
// shape of the core major derived from [defaultCoreVersion]; use
// [BuildCatalogCore] to author against another core version.
func BuildCatalog(path, version string, txs ...TxFixture) string {
	return BuildCatalogCore(defaultCoreVersion, path, version, txs...)
}

// BuildCatalogCore is [BuildCatalog] with an explicit core version (full
// version or bare major), selecting the catalog member shape of that core
// major: v1-era members carry `version` in metadata and version-keyed
// contract FQNs; v2 members carry `apiVersion`/`catalogVersion`/authored
// `fqn` and contract FQNs keyed by [ContractAPIVersion] (transformer keys
// stay build-keyed in both).
func BuildCatalogCore(coreVersion, path, version string, txs ...TxFixture) string {
	v2 := coreIsV2(coreVersionOr(coreVersion))

	var b strings.Builder
	if v2 {
		major, _, _ := strings.Cut(version, ".")
		fmt.Fprintf(&b, "metadata: {\n\tmodulePath:  %q\n\tversion:     %q\n\tdescription: \"test catalog\"\n}\n", path+"@v"+major, version)
	} else {
		fmt.Fprintf(&b, "metadata: {\n\tmodulePath:  %q\n\tversion:     %q\n\tdescription: \"test catalog\"\n}\n", path, version)
	}
	b.WriteString("#transformers: {\n")
	for _, tx := range txs {
		fqn := fmt.Sprintf("%s/transformers/%s@%s", path, tx.Name, version)
		fmt.Fprintf(&b, "\t%q: {\n", fqn)
		b.WriteString("\t\tkind: \"ComponentTransformer\"\n")
		if v2 {
			fmt.Fprintf(&b, "\t\tmetadata: {\n\t\t\tname:        %q\n\t\t\tdescription: %q\n\t\t\tfqn:         %q\n\t\t}\n", tx.Name, tx.Name+" transformer", fqn)
		} else {
			fmt.Fprintf(&b, "\t\tmetadata: {\n\t\t\tname:        %q\n\t\t\tdescription: %q\n\t\t}\n", tx.Name, tx.Name+" transformer")
		}
		if len(tx.Resources) > 0 {
			b.WriteString("\t\trequiredResources: {\n")
			for _, r := range tx.Resources {
				rfqn := contractFQN(v2, path, "resources", r, version)
				fmt.Fprintf(&b, "\t\t\t%q: {\n", rfqn)
				b.WriteString("\t\t\t\tkind: \"Resource\"\n")
				if v2 {
					fmt.Fprintf(&b, "\t\t\t\tmetadata: {name: %q, modulePath: %q, apiVersion: %q, catalogVersion: %q, fqn: %q}\n",
						r, path+"/resources", ContractAPIVersion, version, rfqn)
				} else {
					fmt.Fprintf(&b, "\t\t\t\tmetadata: {name: %q, modulePath: %q, version: %q}\n", r, path+"/resources", version)
				}
				fmt.Fprintf(&b, "\t\t\t\tspec: %q: _\n", specField(r))
				b.WriteString("\t\t\t}\n")
			}
			b.WriteString("\t\t}\n")
		}
		if len(tx.Traits) > 0 {
			b.WriteString("\t\trequiredTraits: {\n")
			for _, tr := range tx.Traits {
				trfqn := contractFQN(v2, path, "traits", tr, version)
				fmt.Fprintf(&b, "\t\t\t%q: {\n", trfqn)
				b.WriteString("\t\t\t\tkind: \"Trait\"\n")
				if v2 {
					fmt.Fprintf(&b, "\t\t\t\tmetadata: {name: %q, modulePath: %q, apiVersion: %q, catalogVersion: %q, fqn: %q}\n",
						tr, path+"/traits", ContractAPIVersion, version, trfqn)
					b.WriteString("\t\t\t\toptional: bool | *true\n")
				} else {
					fmt.Fprintf(&b, "\t\t\t\tmetadata: {name: %q, modulePath: %q, version: %q}\n", tr, path+"/traits", version)
				}
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

// contractFQN returns the FQN a generated fixture catalog keys a primitive
// under: build-version-keyed on the v1 line, [ContractAPIVersion]-keyed on v2
// (enhancement 0010 D4). Test harnesses authoring components against
// generated catalogs mirror the v2 form so demanded keys match the matcher
// index.
func contractFQN(v2 bool, path, kind, name, version string) string {
	if v2 {
		return fmt.Sprintf("%s/%s/%s@%s", path, kind, name, ContractAPIVersion)
	}
	return fmt.Sprintf("%s/%s/%s@%s", path, kind, name, version)
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
