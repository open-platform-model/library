// Package registrytest provides an in-memory OCI registry harness for tests
// that need catalogs and modules resolvable by import without a live
// registry.
//
// It stands up a [modregistrytest] registry serving inline `c.#Catalog` and
// `c.#Module` fixtures under the [CatalogPrefix] module path (or a committed
// fixture tree, [NewRegistryFromDir]). The CUE_REGISTRY mapping routes the
// test prefix to the in-process host and leaves every other path on the
// public registry.
//
// Every constructor points CUE_CACHE_DIR at a private per-test module cache
// ([schematest.PrivateCacheDir]): served coordinates extract into it fresh,
// so a committed fixture edited under a fixed version is always built from
// its current bytes and two test processes never read or write the same
// fixture directory; the opmodel.dev tier of that cache is the shared
// workspace cache, so opmodel.dev/core@v2 stays warm across tests and
// processes. Nothing here deletes from the shared cache: CUE's module cache
// assumes an extracted directory is immutable while any process can read it.
//
// It lives under opm/internal/ so it stays out of the library's public SemVer
// surface (kernel neutrality) while remaining importable from any opm/* test
// package. The kernel render tests and the loader tests share it so registry
// semantics never drift between them.
package registrytest

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"cuelang.org/go/mod/modregistrytest"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/internal/schematest"
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
	// to DefaultCoreVersion (the v2 line); historical tests pin a v1-era
	// version explicitly. core still resolves from the public registry / warm
	// workspace cache.
	CoreVersion string
}

// TxFixture describes one transformer to author into a test catalog: its kebab
// name plus the short names of the resources it requires (the demands the
// render build's matcher buckets on). Output is an optional inline
// `#transform.output` literal (a CUE struct or list expression); when empty it
// defaults to an empty struct. Trait demands are authored by the committed
// render fixtures (testdata/render/registry), not generated here.
type TxFixture struct {
	Name      string
	Resources []string
	Output    string // optional inline #transform.output literal; "" → "{}"
}

// UniquePath returns a catalog module path derived from the current test's
// name. Each test builds against its own private module cache, so the name
// carries no isolation duty; it makes a fixture's coordinate readable in
// diagnostics and cache listings.
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
	// import derive from it. Empty defaults to DefaultCoreVersion (the v2
	// line); historical tests pin a v1-era version explicitly. core still
	// resolves from the public registry / warm workspace cache.
	CoreVersion string
}

// DefaultCoreVersion is the opmodel.dev/core version registrytest fixtures
// declare when ModuleFixture.CoreVersion / CatalogFixture.CoreVersion are
// empty, and the version a test writes into any module file it authors
// beside them (a platform module importing a served catalog, an instance
// module importing a served module). It is the release
// [schema.DefaultSchemaModule] pins; historical tests pin earlier versions
// explicitly.
const DefaultCoreVersion = "v2.0.0-alpha.7"

// ContractAPIVersion is the contract level every generated v2 fixture
// primitive declares. Core v2 keys contracts by the primitive's own
// apiVersion (enhancement 0010 D4), not by the catalog's build version.
const ContractAPIVersion = "v1"

// PrimitiveMatchKey is the matchLabels key every generated v2 fixture
// primitive authors (valued with the primitive's short name). Mirrors the
// real catalog's shape: matching identity lives in matchLabels (0010 D36)
// with a transitional duplicate under metadata.labels.
const PrimitiveMatchKey = "opm.test/primitive"

// coreVersionOr returns v normalized to a leading "v", or DefaultCoreVersion
// when v is empty.
func coreVersionOr(v string) string {
	if v == "" {
		return DefaultCoreVersion
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
// "v2.0.0-alpha.7" → "v2"; a bare major ("v2") passes through.
func coreMajor(coreVersion string) string {
	major, _, _ := strings.Cut(coreVersion, ".")
	return major
}

// NewCatalogRegistry stands up an in-memory OCI registry serving the given
// catalog fixtures and configures CUE_REGISTRY / CUE_CACHE_DIR for the test
// scope: the test prefix routes to the in-process host (+insecure), while
// opmodel.dev/core resolves from the public registry via the shared
// workspace cache; served fixtures extract into the test's private cache.
// Returns the CUE_REGISTRY mapping string. The registry is torn down at test
// end.
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

// NewRegistryFromDir stands up an in-memory OCI registry serving the modules
// laid out under dir in modregistrytest's own layout (one directory per
// (module, version) named "<path with / → _>_v<X.Y.Z>", each a complete
// module tree with its cue.mod/module.cue) and configures CUE_REGISTRY /
// CUE_CACHE_DIR for the test scope: prefix (a module path prefix such as
// "testing.opmodel.dev/library-render") routes to the in-process host
// (+insecure) while everything else, opmodel.dev/core included, resolves from
// the public registry via the shared workspace cache. It serves COMMITTED
// fixture trees (e.g. testdata/render/registry) rather than generated ones;
// because the test's module cache is private and starts empty for every
// served coordinate, a fixture edited under a fixed version is built from
// its current bytes, never from what an earlier run cached. Returns the
// CUE_REGISTRY mapping string.
func NewRegistryFromDir(t *testing.T, dir, prefix string) string {
	t.Helper()
	require.NotEmpty(t, prefix, "a module path prefix routes the fixture tree to the in-process host")

	mapfs := fstest.MapFS{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		mapfs[filepath.ToSlash(rel)] = &fstest.MapFile{Data: data}
		return nil
	})
	require.NoError(t, err, "reading fixture registry tree %s", dir)

	reg, err := modregistrytest.New(mapfs, "")
	require.NoError(t, err, "stand up in-memory registry")
	t.Cleanup(reg.Close)
	return setEnv(t, prefix, reg.Host())
}

// setEnv wires the test environment for a registry serving prefix at host and
// returns the CUE_REGISTRY mapping: prefix routes to the in-process host
// (+insecure) and every other path to the public registry; CUE_CACHE_DIR is
// the test's private module cache, whose opmodel.dev tier is the shared
// workspace cache (see [schematest.PrivateCacheDir]). Both settings revert at
// test end.
func setEnv(t *testing.T, prefix, host string) string {
	t.Helper()
	registry := prefix + "=" + host + "+insecure," + schema.PublicRegistry
	t.Setenv("CUE_REGISTRY", registry)
	t.Setenv("CUE_CACHE_DIR", schematest.PrivateCacheDir(t))
	return registry
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
// environment ([setEnv] for [CatalogPrefix]). Returns the CUE_REGISTRY mapping
// string.
func buildRegistry(t *testing.T, mapfs fstest.MapFS) string {
	t.Helper()

	reg, err := modregistrytest.New(mapfs, "")
	require.NoError(t, err, "stand up in-memory registry")
	t.Cleanup(reg.Close)
	return setEnv(t, CatalogPrefix, reg.Host())
}

// BuildModuleFile renders a complete module.cue for a #Module that imports the
// core schema and (optionally) a single catalog, setting the author-given
// identity metadata. When catalogImport is non-empty the module imports that
// major-qualified catalog path and references its metadata under debugValues
// (an open field), forcing the loader to resolve the catalog as a transitive
// dependency. pkg is the package clause name. The emitted core import derives
// its major from [DefaultCoreVersion], matching the dep the fixture writer
// declares; a test that needs another core version hand-authors its module
// text.
//
// The metadata shape is core v2's: modulePath is the FULL module path (major
// suffix included; pass "…/modules/hello@v0") and name MUST be its snake_case
// leaf.
func BuildModuleFile(pkg, name, modulePath, catalogImport string) string {
	dep := coreDep(DefaultCoreVersion)
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
// defaulting to an empty struct). The body is core v2's catalog member shape:
// members carry `apiVersion`/`catalogVersion`/an authored `fqn`, and contract
// FQNs are keyed by [ContractAPIVersion] (transformer keys stay build-keyed).
func BuildCatalog(path, version string, txs ...TxFixture) string {
	var b strings.Builder
	major, _, _ := strings.Cut(version, ".")
	fmt.Fprintf(&b, "metadata: {\n\tmodulePath:  %q\n\tversion:     %q\n\tdescription: \"test catalog\"\n}\n", path+"@v"+major, version)
	b.WriteString("#transformers: {\n")
	for _, tx := range txs {
		fqn := fmt.Sprintf("%s/transformers/%s@%s", path, tx.Name, version)
		fmt.Fprintf(&b, "\t%q: {\n", fqn)
		b.WriteString("\t\tkind: \"ComponentTransformer\"\n")
		fmt.Fprintf(&b, "\t\tmetadata: {\n\t\t\tname:        %q\n\t\t\tdescription: %q\n\t\t\tfqn:         %q\n\t\t}\n", tx.Name, tx.Name+" transformer", fqn)
		if len(tx.Resources) > 0 {
			b.WriteString("\t\trequiredResources: {\n")
			for _, r := range tx.Resources {
				rfqn := contractFQN(path, "resources", r)
				fmt.Fprintf(&b, "\t\t\t%q: {\n", rfqn)
				b.WriteString("\t\t\t\tkind: \"Resource\"\n")
				// matchLabels is the matching identity (0010 D36); the
				// metadata.labels duplicate mirrors the real catalog's
				// transitional state (kept for descriptive reads).
				fmt.Fprintf(&b, "\t\t\t\tmetadata: {name: %q, modulePath: %q, apiVersion: %q, catalogVersion: %q, fqn: %q, labels: %q: %q}\n",
					r, path+"/resources", ContractAPIVersion, version, rfqn, PrimitiveMatchKey, r)
				fmt.Fprintf(&b, "\t\t\t\tmatchLabels: %q: %q\n", PrimitiveMatchKey, r)
				fmt.Fprintf(&b, "\t\t\t\tspec: %q: _\n", specField(r))
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
// under: [ContractAPIVersion]-keyed (enhancement 0010 D4). Test harnesses
// authoring components against generated catalogs mirror this form so demanded
// keys match the glue's buckets.
func contractFQN(path, kind, name string) string {
	return fmt.Sprintf("%s/%s/%s@%s", path, kind, name, ContractAPIVersion)
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
