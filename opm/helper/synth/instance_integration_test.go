package synth_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loaderfile "github.com/open-platform-model/library/opm/helper/loader/file"
	registryloader "github.com/open-platform-model/library/opm/helper/loader/registry"
	"github.com/open-platform-model/library/opm/helper/synth"
	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/schema"
)

// skipUnlessGHCR gates the import-based synth integration tests the same way the
// kernel flow test gates: skip under -short and when GHCR is unreachable (the
// fabricated package resolves opmodel.dev/core from ghcr.io / the warm cache),
// unless OPM_FLOW_TEST_FORCE=1 turns the skip into a hard requirement.
func skipUnlessGHCR(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("import-based synth integration test pulls opmodel.dev/core from GHCR; skipping under -short")
	}
	if os.Getenv("OPM_FLOW_TEST_FORCE") == "1" {
		return
	}
	conn, err := net.DialTimeout("tcp", "ghcr.io:443", 500*time.Millisecond)
	if err != nil {
		t.Skipf("GHCR not reachable (%v); set OPM_FLOW_TEST_FORCE=1 to require it", err)
	}
	_ = conn.Close()
}

// coreVersionSlug turns a core version like "v1.0.0-alpha.1" into a clean module-path
// segment ("v0-5-0"). Module paths must avoid "." in a path segment and MUST
// NOT be built from t.Name() (subtest names carry '@', '#', parens — all
// invalid in a CUE module path), so each fixture is keyed by this slug and the
// shared CUE module cache never shadows one core version's fixture with
// another's.
func coreVersionSlug(coreVersion string) string {
	return strings.NewReplacer(".", "-", "/", "-").Replace(coreVersion)
}

// cleanSeg sanitizes an arbitrary string (e.g. t.Name(), which carries '/',
// '#', '@', parens) into a single CUE-module-path-safe segment. Module paths
// must never be built from t.Name() verbatim.
func cleanSeg(s string) string {
	return strings.NewReplacer(
		"/", "-", "_", "-", "#", "-", "@", "-",
		"(", "-", ")", "-", ".", "-", " ", "-", "=", "-", ",", "-",
	).Replace(strings.ToLower(s))
}

// publishModuleWithBody publishes a #Module whose module.cue carries bodyFields
// (the text after the metadata block — e.g. #components / #config / debugValues)
// pinned to coreVersion, and loads it back through the registry loader into a
// *module.Module. It follows the OPM module publishing convention
// (enhancements/0003): the module is published at metadata.modulePath +
// "/" + snake_case(name) and its CUE package name is snake_case(name), while
// metadata.name keeps its (possibly hyphenated) kebab form. The path is keyed
// by the test name AND coreVersion (a clean slug, never t.Name() verbatim) so
// fixtures never collide or shadow one another in the shared CUE module cache.
// Returns the loaded module and the CUE_REGISTRY mapping.
func publishModuleWithBody(t *testing.T, ctx *cue.Context, coreVersion, name, version, bodyFields string) (*module.Module, string) {
	t.Helper()

	snake := strings.ReplaceAll(name, "-", "_")
	base := registrytest.CatalogPrefix + "/synthunit/" + cleanSeg(t.Name()) + "-" + coreVersionSlug(coreVersion)
	metaPath := base
	modPath := metaPath + "/" + snake // canonical leaf = snake_case(name)

	var file strings.Builder
	fmt.Fprintf(&file, "package %s\n\n", snake) // package name = snake_case(name)
	file.WriteString("import core \"opmodel.dev/core@v1\"\n\n")
	file.WriteString("core.#Module\n")
	fmt.Fprintf(&file, "metadata: {\n\tname:       %q\n\tmodulePath: %q\n\tversion:    %q\n}\n", name, metaPath, version)
	file.WriteString(bodyFields)

	mod := registrytest.ModuleFixture{
		Path:        modPath,
		Version:     version,
		File:        file.String(),
		CoreVersion: coreVersion,
	}
	registryMapping := registrytest.NewModuleRegistry(t, []registrytest.ModuleFixture{mod}, nil)

	res, err := registryloader.LoadModulePackageWithSource(context.Background(), ctx, modPath+"@v0", "v"+version,
		registryloader.LoadOptions{Registry: registryMapping})
	require.NoErrorf(t, err, "loading published module %s@v%s", modPath, version)

	m, err := module.NewModuleFromValue(stubOwner{ctx: ctx}, res.Value)
	require.NoError(t, err, "constructing *module.Module from loaded value")
	// Synth builds the instance inside the module's own staged root, so the
	// module must carry its source (as Kernel.AcquireModuleFromRegistry attaches).
	m.Source = &module.Source{Root: res.Root, Overlay: res.Overlay}
	require.True(t, m.HasSource(), "published fixture module must carry staged source for synth")
	require.Equal(t, metaPath, m.Metadata.ModulePath)
	require.Equal(t, name, m.Metadata.Name)
	return m, registryMapping
}

// publishImportableModule publishes a minimal #Module (empty body) pinned to
// coreVersion and loads it back — the simplest importable fixture, used by the
// positive/negative construction guards.
func publishImportableModule(t *testing.T, ctx *cue.Context, coreVersion string) (*module.Module, string) {
	t.Helper()
	return publishModuleWithBody(t, ctx, coreVersion, "hello", "0.0.2", "")
}

// pinnedCache returns a *schema.Cache pinned to an explicit core version. Under
// design D4 the cache no longer pins the synth build's core (that comes from the
// MODULE's own cue.mod/module.cue); the cache confirms #ModuleInstance is
// available and supplies the core import's major. Tests pin it to the same core
// version the fixture module is published against so the major aligns.
func pinnedCache(coreVersion string) *schema.Cache {
	return &schema.Cache{Loader: schema.OCILoader{Module: "opmodel.dev/core@" + coreVersion}}
}

// TestInstance_ImportedModule_ConstructsOnAuthorSuppliedCore is the positive half
// of the import-construction guard (tasks 4.4 / 4.9): a real published module
// referenced by IMPORT constructs a #ModuleInstance with concrete #module
// identity on core ≥ v0.5.0 (author-supplied #Module identity).
func TestInstance_ImportedModule_ConstructsOnAuthorSuppliedCore(t *testing.T) {
	skipUnlessGHCR(t)
	ctx := cuecontext.New()

	mod, _ := publishImportableModule(t, ctx, "v1.0.0-alpha.1")

	inst, err := synth.Instance(ctx, synth.InstanceInput{
		Module:      mod,
		Name:        "hello-inst",
		Namespace:   "default",
		SchemaCache: pinnedCache("v1.0.0-alpha.1"),
	})
	require.NoError(t, err, "imported-module instance must construct on author-supplied-identity core")
	require.True(t, inst.Exists())

	kind, err := inst.LookupPath(cue.ParsePath("kind")).String()
	require.NoError(t, err)
	assert.Equal(t, "ModuleInstance", kind)

	// #module identity must be concrete — this is the self-cycle boundary the
	// core fix removed; before it, re-evaluating the imported #Module resolved
	// modulePath/version to bottom.
	modPath, err := inst.LookupPath(cue.ParsePath("#module.metadata.modulePath")).String()
	require.NoError(t, err, "imported #module.metadata.modulePath must be concrete")
	assert.Equal(t, mod.Metadata.ModulePath, modPath)
	modVer, err := inst.LookupPath(cue.ParsePath("#module.metadata.version")).String()
	require.NoError(t, err, "imported #module.metadata.version must be concrete")
	assert.Equal(t, mod.Metadata.Version, modVer)
}

// The negative control TestRelease_ImportedModule_NegativeControlV040 was
// retired by enhancement 0002 (Release→Instance). It pinned core@v0.4.0 (the
// pre-self-cycle-fix boundary) to prove the positive test was non-vacuous, but
// the library now hard-targets core@v1: synth emits core.#ModuleInstance, which
// is undefined in v0.4.0, so the construction fails for a different reason than
// the self-cycle admission path it was meant to exercise. Supporting a
// pre-rename core is explicitly out of scope. See MIGRATIONS.md.

// TestInstance_DerivedFields_FromSchema pins task 4.2: every field the schema
// derives is unchanged under single-build construction — metadata.uuid is the
// canonical SHA1 (stable, namespace-divergent), components is fanned from the
// module's #components, and the standard module-instance.opmodel.dev/{name,uuid}
// labels coexist with caller labels. These are produced by CUE unification of
// the synthesized package, not by Go-side scope/fill.
func TestInstance_DerivedFields_FromSchema(t *testing.T) {
	skipUnlessGHCR(t)
	ctx := cuecontext.New()

	mod, _ := publishModuleWithBody(t, ctx, "v1.0.0-alpha.1", "demo", "0.1.0", `
#components: {
	foo: {metadata: name: "foo"}
	bar: {metadata: name: "bar"}
}
#config: {}
debugValues: {}
`)

	inst, err := synth.Instance(ctx, synth.InstanceInput{
		Module:      mod,
		Name:        "myrel",
		Namespace:   "default",
		Labels:      map[string]string{"env": "prod"},
		Annotations: map[string]string{"opmodel.dev/owner": "team-x"},
		SchemaCache: pinnedCache("v1.0.0-alpha.1"),
	})
	require.NoError(t, err)

	// metadata.uuid is the canonical SHA1 derived by the schema.
	uuid, err := inst.LookupPath(cue.ParsePath("metadata.uuid")).String()
	require.NoError(t, err)
	assert.Equal(t, expectedInstanceUUID(t, ctx, mod.Metadata.UUID, "myrel", "default"), uuid,
		"schema-derived UUID must equal uuid.SHA1(OPMNamespace, <module.uuid>:<name>:<namespace>)")

	// components fanned from #components.
	components := inst.LookupPath(cue.ParsePath("components"))
	require.True(t, components.Exists())
	assert.True(t, components.LookupPath(cue.ParsePath("foo")).Exists(), "components.foo fanned from #components.foo")
	assert.True(t, components.LookupPath(cue.ParsePath("bar")).Exists(), "components.bar fanned from #components.bar")

	// caller labels coexist with schema-stamped labels.
	labels := map[string]string{}
	require.NoError(t, inst.LookupPath(cue.ParsePath("metadata.labels")).Decode(&labels))
	assert.Equal(t, "prod", labels["env"], "caller-supplied label must be present")
	assert.Equal(t, "myrel", labels["module-instance.opmodel.dev/name"], "schema-stamped name label must coexist")
	assert.NotEmpty(t, labels["module-instance.opmodel.dev/uuid"], "schema-stamped uuid label must coexist")

	// annotations pass through unchanged.
	annotations := map[string]string{}
	require.NoError(t, inst.LookupPath(cue.ParsePath("metadata.annotations")).Decode(&annotations))
	assert.Equal(t, "team-x", annotations["opmodel.dev/owner"], "caller-supplied annotation must survive")
}

// TestInstance_NamespaceChangesUUID pins the namespace-divergence + determinism
// of the schema-derived instance UUID under single-build construction.
func TestInstance_NamespaceChangesUUID(t *testing.T) {
	skipUnlessGHCR(t)
	ctx := cuecontext.New()

	mod, _ := publishModuleWithBody(t, ctx, "v1.0.0-alpha.1", "demo", "0.1.0", "#components: {}\n#config: {}\ndebugValues: {}\n")
	cache := pinnedCache("v1.0.0-alpha.1")

	uuidFor := func(ns string) string {
		inst, err := synth.Instance(ctx, synth.InstanceInput{Module: mod, Name: "inst", Namespace: ns, SchemaCache: cache})
		require.NoError(t, err)
		u, err := inst.LookupPath(cue.ParsePath("metadata.uuid")).String()
		require.NoError(t, err)
		return u
	}

	a1, b, a2 := uuidFor("ns-a"), uuidFor("ns-b"), uuidFor("ns-a")
	assert.NotEqual(t, a1, b, "different namespaces must produce different UUIDs")
	assert.Equal(t, a1, a2, "identical inputs must produce identical UUIDs")
}

// TestInstance_EmptyValuesNotReplacedByDebugValues pins task 4.1's values rule:
// with no caller Values, the schema's values path stays unfilled and is NEVER
// backfilled from Module.debugValues.
func TestInstance_EmptyValuesNotReplacedByDebugValues(t *testing.T) {
	skipUnlessGHCR(t)
	ctx := cuecontext.New()

	// debugValues intentionally non-empty (and #config requires a sentinel) so
	// we can prove synth.Instance does NOT fall back to debugValues.
	mod, _ := publishModuleWithBody(t, ctx, "v1.0.0-alpha.1", "demo", "0.1.0", `
#components: {}
#config: {sentinel: string}
debugValues: {sentinel: "from-debug"}
`)

	inst, err := synth.Instance(ctx, synth.InstanceInput{
		Module:      mod,
		Name:        "myrel",
		Namespace:   "default",
		SchemaCache: pinnedCache("v1.0.0-alpha.1"),
		// Values omitted — MUST NOT fall back to debugValues.
	})
	require.NoError(t, err)

	values := inst.LookupPath(cue.ParsePath("values"))
	if values.Exists() {
		assert.Error(t, values.Validate(cue.Concrete(true)),
			"values path must be non-concrete when no Values were supplied (not backfilled from debugValues)")
	}
}

// TestInstance_AutoSecretsComponentInjected pins task 4.2's opm-secrets rule:
// when caller Values carry a #Secret instance, the schema injects the
// opm-secrets component — and the values merge happens in-build via the
// schema's unifiedModule, not a Go-side #config fill.
func TestInstance_AutoSecretsComponentInjected(t *testing.T) {
	skipUnlessGHCR(t)
	ctx := cuecontext.New()

	mod, _ := publishModuleWithBody(t, ctx, "v1.0.0-alpha.1", "demo", "0.1.0", `
#components: {}
#config: {
	dbPassword: core.#SecretLiteral & {
		$secretName: "app-secrets"
		$dataKey:    "db-password"
	}
}
debugValues: {dbPassword: {value: "s3cret"}}
`)

	values := ctx.CompileString(`
dbPassword: {
	$opm:        "secret"
	$secretName: "app-secrets"
	$dataKey:    "db-password"
	value:       "s3cret"
}
`)
	require.NoError(t, values.Err())

	inst, err := synth.Instance(ctx, synth.InstanceInput{
		Module:      mod,
		Name:        "myrel",
		Namespace:   "default",
		Values:      values,
		SchemaCache: pinnedCache("v1.0.0-alpha.1"),
	})
	require.NoError(t, err)

	components := inst.LookupPath(cue.ParsePath("components"))
	require.True(t, components.Exists())
	assert.True(t, components.LookupPath(cue.ParsePath(`"opm-secrets"`)).Exists(),
		"opm-secrets component must be auto-injected when the module has #Secret instances")
}

// TestInstance_BadNameFailsUnification pins that a #NameType-violating name fails
// at schema unification within the single build (a real admission error), not
// vacuously via some unrelated failure.
func TestInstance_BadNameFailsUnification(t *testing.T) {
	skipUnlessGHCR(t)
	ctx := cuecontext.New()

	mod, _ := publishImportableModule(t, ctx, "v1.0.0-alpha.1")

	_, err := synth.Instance(ctx, synth.InstanceInput{
		Module:      mod,
		Name:        "BAD-UPPER", // #NameType forbids uppercase
		Namespace:   "default",
		SchemaCache: pinnedCache("v1.0.0-alpha.1"),
	})
	require.Error(t, err, "names violating #NameType must surface as a unification error")
	assert.NotContains(t, err.Error(), "cannot find package",
		"failure must be name unification, not a module-resolution accident")
}

// TestInstance_ParityWithAuthoredPackage pins task 4.5: synth.Instance and an
// equivalent authored instance.cue package that imports the SAME published
// module construct the same instance value — same schema-derived uuid and the
// same fanned components. This is the convergence the change exists to
// guarantee: a render bug surfaces in both paths or neither.
func TestInstance_ParityWithAuthoredPackage(t *testing.T) {
	skipUnlessGHCR(t)
	ctx := cuecontext.New()

	mod, registryMapping := publishModuleWithBody(t, ctx, "v1.0.0-alpha.1", "demo", "0.1.0", `
#components: {
	foo: {metadata: name: "foo"}
}
#config: {}
debugValues: {}
`)

	// synth path.
	synthRel, err := synth.Instance(ctx, synth.InstanceInput{
		Module:      mod,
		Name:        "myrel",
		Namespace:   "default",
		SchemaCache: pinnedCache("v1.0.0-alpha.1"),
	})
	require.NoError(t, err)

	// authored path: an on-disk instance.cue importing the same published module.
	importPath := mod.Metadata.ModulePath + "/" + mod.Metadata.Name + "@v0"
	dir := t.TempDir()
	instanceSrc := `package instance

import (
	core "opmodel.dev/core@v1"
	opmModule "` + importPath + `"
)

core.#ModuleInstance

metadata: {
	name:      "myrel"
	namespace: "default"
}

#module: opmModule
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "instance.cue"), []byte(instanceSrc), 0o644))
	// A cue.mod/module.cue pinning core + the module is required for the import.
	moduleSrc := `module: "authored.opmodel.dev/instance@v0"
language: version: "v0.17.0-alpha.1"
deps: {
	"opmodel.dev/core@v1": v: "v1.0.0-alpha.1"
	"` + importPath + `": v: "v0.1.0"
}
`
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cue.mod"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cue.mod", "module.cue"), []byte(moduleSrc), 0o644))

	authoredRel, err := loaderfile.LoadInstancePackage(ctx, dir, loaderfile.LoadOptions{Registry: registryMapping})
	require.NoError(t, err, "authored instance.cue importing the published module must load")

	// Authored-path identity (task 4.7): the imported #module's identity must be
	// concrete end-to-end — this is the path that rotted invisibly before the
	// core self-cycle fix (it failed with `field not allowed`).
	for _, p := range []string{"#module.metadata.modulePath", "#module.metadata.version", "#module.metadata.fqn"} {
		v := authoredRel.LookupPath(cue.ParsePath(p))
		s, err := v.String()
		require.NoErrorf(t, err, "authored %s must be concrete", p)
		require.NotEmpty(t, s, "authored %s must be non-empty", p)
	}

	synthUUID, err := synthRel.LookupPath(cue.ParsePath("metadata.uuid")).String()
	require.NoError(t, err)
	authoredUUID, err := authoredRel.LookupPath(cue.ParsePath("metadata.uuid")).String()
	require.NoError(t, err)
	assert.Equal(t, authoredUUID, synthUUID, "synth and authored paths must derive the same instance UUID")

	for _, comp := range []string{"foo"} {
		assert.True(t, synthRel.LookupPath(cue.ParsePath("components."+comp)).Exists(), "synth components."+comp)
		assert.True(t, authoredRel.LookupPath(cue.ParsePath("components."+comp)).Exists(), "authored components."+comp)
	}
}

// TestInstance_HyphenatedNameImportsBySnakeCase pins the nameSnakeCase fix
// (enhancements/0003): a module whose metadata.name carries hyphens
// ("web-app") is published at the snake_case leaf (".../web_app") with a
// snake_case CUE package, and synth.Instance must derive that import path from
// metadata.nameSnakeCase. The previous modulePath/name derivation produced
// ".../web-app" and failed to resolve — this is the regression guard for that.
func TestInstance_HyphenatedNameImportsBySnakeCase(t *testing.T) {
	skipUnlessGHCR(t)
	ctx := cuecontext.New()

	mod, _ := publishModuleWithBody(t, ctx, "v1.0.0-alpha.1", "web-app", "0.1.0", "#components: {}\n#config: {}\ndebugValues: {}\n")

	// The kebab identity is preserved; nameSnakeCase is the derived snake form.
	require.Equal(t, "web-app", mod.Metadata.Name)
	snake, err := mod.Package.LookupPath(cue.ParsePath("metadata.nameSnakeCase")).String()
	require.NoError(t, err, "core@v0.6.0 module must expose metadata.nameSnakeCase")
	require.Equal(t, "web_app", snake)

	inst, err := synth.Instance(ctx, synth.InstanceInput{
		Module:      mod,
		Name:        "web-app-inst",
		Namespace:   "default",
		SchemaCache: pinnedCache("v1.0.0-alpha.1"),
	})
	require.NoError(t, err, "hyphenated-name module must import by its snake_case registry leaf")

	modName, err := inst.LookupPath(cue.ParsePath("#module.metadata.name")).String()
	require.NoError(t, err)
	assert.Equal(t, "web-app", modName, "imported #module keeps its kebab metadata.name")
}
