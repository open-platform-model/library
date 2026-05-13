package synth_test

import (
	"errors"
	"io/fs"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/api"
	_ "github.com/open-platform-model/library/opm/api/v1alpha2"
	"github.com/open-platform-model/library/opm/apiversion"
	"github.com/open-platform-model/library/opm/helper/synth"
	"github.com/open-platform-model/library/opm/module"
)

// brokenSchemaVer is the unique apiVersion used to register a stub binding
// whose SchemaValue always fails. Living at package scope so the registry
// stub is created at most once per test binary (api.Register panics on
// duplicate; the registry has no public Unregister hook).
const brokenSchemaVer = apiversion.Version("opmodel.dev/synthtest-broken")

// errBrokenSchema is the sentinel returned by brokenSchemaBinding.SchemaValue.
// TestRelease_WrapsSchemaLoadFailure asserts synth.Release wraps it (via %w)
// so callers can match the underlying failure with errors.Is.
var errBrokenSchema = errors.New("synth_test: broken schema load")

// registerBrokenOnce guards the one-shot api.Register call so the binding
// survives across all tests in this package without triggering the registry's
// duplicate-registration panic on subsequent test runs in the same binary.
var registerBrokenOnce sync.Once

// brokenSchemaBinding is a minimal api.Binding stub whose SchemaValue
// returns errBrokenSchema. Every other method panics — the only synth path
// these tests exercise is the schema-load wrap.
type brokenSchemaBinding struct {
	ver apiversion.Version
}

func (b *brokenSchemaBinding) Version() apiversion.Version { return b.ver }
func (b *brokenSchemaBinding) Paths() api.Paths            { return api.Paths{} }
func (b *brokenSchemaBinding) DecodeModuleMetadata(cue.Value) (*api.ModuleMetadata, error) {
	panic("unused in broken-schema test")
}
func (b *brokenSchemaBinding) DecodeReleaseMetadata(cue.Value) (*api.ReleaseMetadata, error) {
	panic("unused in broken-schema test")
}
func (b *brokenSchemaBinding) DecodeProviderMetadata(cue.Value, string) (*api.ProviderMetadata, error) {
	panic("unused in broken-schema test")
}
func (b *brokenSchemaBinding) DecodePlatformMetadata(cue.Value) (*api.PlatformMetadata, error) {
	panic("unused in broken-schema test")
}
func (b *brokenSchemaBinding) BuildTransformerContext(*cue.Context, api.ReleaseView, string, cue.Value, string) (cue.Value, []string, error) {
	panic("unused in broken-schema test")
}
func (b *brokenSchemaBinding) EmbeddedSchema() fs.FS { return nil }
func (b *brokenSchemaBinding) SchemaValue(*cue.Context) (cue.Value, error) {
	return cue.Value{}, errBrokenSchema
}

// sharedCtx is the single *cue.Context used by every synth test in this
// package. The v1alpha2 binding caches its loaded schema against the first
// *cue.Context that calls SchemaValue. If tests use multiple contexts, the
// second context's values cannot unify with the schema (the schema is bound
// to the first context's runtime), triggering "values are not from the same
// runtime" panics inside cue.Value.FillPath.
//
// The "one Kernel (one *cue.Context) per process" pattern this honours is
// documented on api.Binding.SchemaValue.
var sharedCtx = cuecontext.New()

// apisCoreDir resolves the on-disk path to the apis/core CUE module from
// this test file's location. The synthtest fixtures load through
// load.Instances rooted at this directory so `import core
// "opmodel.dev/core/v1alpha2@v1"` resolves to the local schema without a
// registry round-trip — exactly the pattern schema_fixture_test.go uses.
func apisCoreDir(t *testing.T) string {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	require.True(t, ok)
	// opm/helper/synth/ → repo/library/
	libRoot := filepath.Clean(filepath.Join(filepath.Dir(here), "..", "..", ".."))
	return filepath.Join(libRoot, "apis", "core")
}

// testModule loads a #Module from a synthetic in-overlay fixture rooted under
// apis/core. The overlay provides a synthtest/ package whose source is the
// caller-supplied src; it imports "opmodel.dev/core/v1alpha2@v1" so the local
// module's #Module definition resolves without any registry access. The
// fixture MUST declare `module: <expr>` at the top of the package; this helper
// returns the *module.Module built from that field.
func testModule(t *testing.T, ctx *cue.Context, src string) *module.Module {
	t.Helper()

	moduleRoot := apisCoreDir(t)
	fixturePath := filepath.Join(moduleRoot, "synthtest", "fixture.cue")
	cfg := &load.Config{
		Dir: filepath.Join(moduleRoot, "synthtest"),
		Overlay: map[string]load.Source{
			fixturePath: load.FromString(src),
		},
	}
	insts := load.Instances([]string{"."}, cfg)
	require.Len(t, insts, 1, "synth test fixture must produce exactly one instance")
	require.NoErrorf(t, insts[0].Err, "loading synth test fixture: %v", insts[0].Err)

	pkg := ctx.BuildInstance(insts[0])
	require.NoErrorf(t, pkg.Err(), "building synth test fixture: %v", pkg.Err())

	modVal := pkg.LookupPath(cue.ParsePath("module"))
	require.True(t, modVal.Exists(), "synth test fixture must declare top-level `module:`")
	require.NoErrorf(t, modVal.Err(), "fixture module field error: %v", modVal.Err())

	mod, err := module.NewModuleFromValue(stubOwner{ctx: ctx}, modVal)
	require.NoErrorf(t, err, "constructing *module.Module from fixture: %v", err)
	require.NotNil(t, mod)
	require.NotEmpty(t, mod.Metadata.UUID, "schema-derived module UUID must be present")
	return mod
}

// stubOwner satisfies module.CueContextOwner — module.NewModuleFromValue does
// not actually use the context (the value already carries one); the interface
// exists only to keep opm/module's import surface narrow.
type stubOwner struct{ ctx *cue.Context }

func (s stubOwner) CueContext() *cue.Context { return s.ctx }

// baseModuleFixture is the minimal #Module declaration used by tests that
// don't need a custom #config or debugValues. Name/modulePath/version are
// fixed so the derived module UUID is stable across tests.
const baseModuleFixture = `
package synthtest

import core "opmodel.dev/core/v1alpha2@v1"

module: {
	core.#Module
	metadata: {
		name:       "demo"
		modulePath: "example.com/demo"
		version:    "0.1.0"
	}
	#components: {}
	#config: {}
	debugValues: {}
}
`

// ────────────────────────────── 5.1 ──────────────────────────────

func TestRelease_RejectsNilModule(t *testing.T) {
	ctx := sharedCtx
	_, err := synth.Release(ctx, synth.ReleaseInput{Name: "demo", Namespace: "ns"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, synth.ErrMissingModule), "want ErrMissingModule, got %v", err)
}

func TestRelease_RejectsEmptyName(t *testing.T) {
	ctx := sharedCtx
	mod := testModule(t, ctx, baseModuleFixture)
	_, err := synth.Release(ctx, synth.ReleaseInput{Module: mod, Namespace: "ns"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, synth.ErrMissingName), "want ErrMissingName, got %v", err)
}

func TestRelease_RejectsEmptyNamespace(t *testing.T) {
	ctx := sharedCtx
	mod := testModule(t, ctx, baseModuleFixture)
	_, err := synth.Release(ctx, synth.ReleaseInput{Module: mod, Name: "demo"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, synth.ErrMissingNamespace), "want ErrMissingNamespace, got %v", err)
}

// ────────────────────────────── 5.2 ──────────────────────────────

func TestRelease_StampsCanonicalFields(t *testing.T) {
	ctx := sharedCtx
	mod := testModule(t, ctx, baseModuleFixture)

	rel, err := synth.Release(ctx, synth.ReleaseInput{
		Module:    mod,
		Name:      "myrel",
		Namespace: "default",
	})
	require.NoError(t, err)
	require.True(t, rel.Exists())

	apiVer, err := rel.LookupPath(cue.ParsePath("apiVersion")).String()
	require.NoError(t, err)
	assert.Equal(t, "opmodel.dev/v1alpha2", apiVer)

	kind, err := rel.LookupPath(cue.ParsePath("kind")).String()
	require.NoError(t, err)
	assert.Equal(t, "ModuleRelease", kind)

	uuid, err := rel.LookupPath(cue.ParsePath("metadata.uuid")).String()
	require.NoError(t, err)
	expected := expectedReleaseUUID(t, ctx, mod.Metadata.UUID, "myrel", "default")
	assert.Equal(t, expected, uuid, "schema-derived UUID must equal uuid.SHA1(OPMNamespace, <module.uuid>:<name>:<namespace>)")
}

// expectedReleaseUUID computes the canonical release UUID through CUE so the
// test stays in lockstep with the schema's definition. Failing this
// assertion is the drift sentinel for module_release.cue:19.
func expectedReleaseUUID(t *testing.T, ctx *cue.Context, moduleUUID, name, namespace string) string {
	t.Helper()
	src := `
import cue_uuid "uuid"
OPMNamespace: "11bc6112-a6e8-4021-bec9-b3ad246f9466"
out: cue_uuid.SHA1(OPMNamespace, "` + moduleUUID + `:` + name + `:` + namespace + `")
`
	v := ctx.CompileString(src)
	require.NoError(t, v.Err())
	s, err := v.LookupPath(cue.ParsePath("out")).String()
	require.NoError(t, err)
	return s
}

// ────────────────────────────── 5.3 ──────────────────────────────

func TestRelease_NamespaceChangesUUID(t *testing.T) {
	ctx := sharedCtx
	mod := testModule(t, ctx, baseModuleFixture)

	rel1, err := synth.Release(ctx, synth.ReleaseInput{Module: mod, Name: "rel", Namespace: "ns-a"})
	require.NoError(t, err)
	rel2, err := synth.Release(ctx, synth.ReleaseInput{Module: mod, Name: "rel", Namespace: "ns-b"})
	require.NoError(t, err)

	uuid1, err := rel1.LookupPath(cue.ParsePath("metadata.uuid")).String()
	require.NoError(t, err)
	uuid2, err := rel2.LookupPath(cue.ParsePath("metadata.uuid")).String()
	require.NoError(t, err)
	assert.NotEqual(t, uuid1, uuid2, "different namespaces must produce different UUIDs")

	rel3, err := synth.Release(ctx, synth.ReleaseInput{Module: mod, Name: "rel", Namespace: "ns-a"})
	require.NoError(t, err)
	uuid3, err := rel3.LookupPath(cue.ParsePath("metadata.uuid")).String()
	require.NoError(t, err)
	assert.Equal(t, uuid1, uuid3, "identical inputs must produce identical UUIDs")
}

// ────────────────────────────── 5.4 ──────────────────────────────

func TestRelease_CallerLabelsCoexistWithStampedLabels(t *testing.T) {
	ctx := sharedCtx
	mod := testModule(t, ctx, baseModuleFixture)

	rel, err := synth.Release(ctx, synth.ReleaseInput{
		Module:    mod,
		Name:      "myrel",
		Namespace: "default",
		Labels:    map[string]string{"env": "prod"},
	})
	require.NoError(t, err)

	labels := map[string]string{}
	require.NoError(t, rel.LookupPath(cue.ParsePath("metadata.labels")).Decode(&labels))

	assert.Equal(t, "prod", labels["env"], "caller-supplied label must be present")
	assert.Equal(t, "myrel", labels["module-release.opmodel.dev/name"], "schema-stamped name label must coexist")
	assert.NotEmpty(t, labels["module-release.opmodel.dev/uuid"], "schema-stamped uuid label must coexist")
}

// ────────────────────────────── 5.5 ──────────────────────────────

func TestRelease_EmptyValuesLeavesPathUnfilled(t *testing.T) {
	ctx := sharedCtx
	// debugValues intentionally non-empty so we can prove the synth helper
	// does NOT fall back to it.
	mod := testModule(t, ctx, `
package synthtest

import core "opmodel.dev/core/v1alpha2@v1"

module: {
	core.#Module
	metadata: {
		name:       "demo"
		modulePath: "example.com/demo"
		version:    "0.1.0"
	}
	#components: {}
	#config: { sentinel: string }
	debugValues: { sentinel: "from-debug" }
}
`)

	rel, err := synth.Release(ctx, synth.ReleaseInput{
		Module:    mod,
		Name:      "myrel",
		Namespace: "default",
		// Values intentionally omitted — synth.Release MUST NOT fall back to
		// debugValues. The values path stays open; concreteness is the
		// downstream kernel's responsibility.
	})
	require.NoError(t, err)

	values := rel.LookupPath(cue.ParsePath("values"))
	if values.Exists() {
		err := values.Validate(cue.Concrete(true))
		assert.Error(t, err, "values path must be non-concrete when no Values were supplied")
	}
}

// ────────────────────────────── 5.6 ──────────────────────────────

func TestRelease_AutoSecretsComponentInjected(t *testing.T) {
	ctx := sharedCtx
	// Module declares a #Secret instance in its config schema. The release
	// schema's _autoSecrets comprehension discovers it and adds the
	// opm-secrets component automatically.
	mod := testModule(t, ctx, `
package synthtest

import core "opmodel.dev/core/v1alpha2@v1"

module: {
	core.#Module
	metadata: {
		name:       "demo"
		modulePath: "example.com/demo"
		version:    "0.1.0"
	}
	#components: {}
	#config: {
		dbPassword: core.#SecretLiteral & {
			$secretName: "app-secrets"
			$dataKey:    "db-password"
		}
	}
	debugValues: {
		dbPassword: { value: "s3cret" }
	}
}
`)

	// Concrete values supplied at release time — the literal secret's value.
	values := ctx.CompileString(`
dbPassword: {
	$opm:        "secret"
	$secretName: "app-secrets"
	$dataKey:    "db-password"
	value:       "s3cret"
}
`)
	require.NoError(t, values.Err())

	rel, err := synth.Release(ctx, synth.ReleaseInput{
		Module:    mod,
		Name:      "myrel",
		Namespace: "default",
		Values:    values,
	})
	require.NoError(t, err)

	components := rel.LookupPath(cue.ParsePath("components"))
	require.True(t, components.Exists())
	secretsComp := components.LookupPath(cue.ParsePath(`"opm-secrets"`))
	require.True(t, secretsComp.Exists(),
		"opm-secrets component must be auto-injected when module has #Secret instances")
}

// ────────────────────────────── 5.7 ──────────────────────────────

func TestRelease_NoCueRegistryNeeded(t *testing.T) {
	// synth.Release goes through binding.SchemaValue, which is entirely
	// off-disk (embed + overlay). Verify the contract by explicitly clearing
	// CUE_REGISTRY before the synth call. Note: test-module construction
	// above uses load.Instances against the local apis/core module — that's
	// a same-module reference, not a registry round-trip, so no network
	// access is required.
	t.Setenv("CUE_REGISTRY", "")
	ctx := sharedCtx
	mod := testModule(t, ctx, baseModuleFixture)

	rel, err := synth.Release(ctx, synth.ReleaseInput{
		Module:    mod,
		Name:      "myrel",
		Namespace: "default",
	})
	require.NoError(t, err)
	require.True(t, rel.Exists())

	uuid, err := rel.LookupPath(cue.ParsePath("metadata.uuid")).String()
	require.NoError(t, err)
	assert.NotEmpty(t, uuid, "UUID must be schema-derived even with CUE_REGISTRY unset")
}

// ────────────────── post-verification additions ──────────────────

// TestRelease_ComponentsAreFannedBySchema covers release-synthesis's
// "Components are fanned by schema comprehension" scenario directly at the
// synth-package layer (independent of the registry-gated flow integration
// test). A module declaring two #components — foo and bar — must produce a
// release whose components field contains both keys.
func TestRelease_ComponentsAreFannedBySchema(t *testing.T) {
	ctx := sharedCtx
	mod := testModule(t, ctx, `
package synthtest

import core "opmodel.dev/core/v1alpha2@v1"

module: {
	core.#Module
	metadata: {
		name:       "demo"
		modulePath: "example.com/demo"
		version:    "0.1.0"
	}
	#components: {
		foo: { metadata: name: "foo" }
		bar: { metadata: name: "bar" }
	}
	#config: {}
	debugValues: {}
}
`)

	rel, err := synth.Release(ctx, synth.ReleaseInput{
		Module:    mod,
		Name:      "myrel",
		Namespace: "default",
	})
	require.NoError(t, err)

	components := rel.LookupPath(cue.ParsePath("components"))
	require.True(t, components.Exists())
	require.True(t, components.LookupPath(cue.ParsePath("foo")).Exists(),
		"components.foo must be fanned from #components.foo")
	require.True(t, components.LookupPath(cue.ParsePath("bar")).Exists(),
		"components.bar must be fanned from #components.bar")
}

// TestRelease_AnnotationsPassThrough covers release-synthesis's "Annotations
// are passed through unchanged" scenario. Caller-supplied annotations must
// appear verbatim under metadata.annotations.
func TestRelease_AnnotationsPassThrough(t *testing.T) {
	ctx := sharedCtx
	mod := testModule(t, ctx, baseModuleFixture)

	rel, err := synth.Release(ctx, synth.ReleaseInput{
		Module:      mod,
		Name:        "myrel",
		Namespace:   "default",
		Annotations: map[string]string{"opmodel.dev/owner": "team-x"},
	})
	require.NoError(t, err)

	annotations := map[string]string{}
	require.NoError(t, rel.LookupPath(cue.ParsePath("metadata.annotations")).Decode(&annotations))
	assert.Equal(t, "team-x", annotations["opmodel.dev/owner"],
		"caller-supplied annotation must survive into metadata.annotations unchanged")
}

// TestRelease_RejectsBadName covers release-synthesis's "Unification error
// returned" scenario — a Name violating #NameType (uppercase characters) must
// produce a non-nil error wrapped with the unification context.
func TestRelease_RejectsBadName(t *testing.T) {
	ctx := sharedCtx
	mod := testModule(t, ctx, baseModuleFixture)

	_, err := synth.Release(ctx, synth.ReleaseInput{
		Module:    mod,
		Name:      "BAD-UPPER", // #NameType forbids uppercase
		Namespace: "default",
	})
	require.Error(t, err,
		"Names violating #NameType must surface as a unification error")
}

// TestRelease_WrapsSchemaLoadFailure covers release-synthesis's "Schema load
// failure surfaces as a wrapped error" scenario at the synth layer. A
// stub binding whose SchemaValue returns a sentinel error must cause
// synth.Release to return an error wrapping the binding's error.
func TestRelease_WrapsSchemaLoadFailure(t *testing.T) {
	// Register a stub binding once per test binary. The api registry has no
	// public Unregister hook, but the apiVersion we use is unique to this
	// test so it cannot conflict with anything else in the suite.
	registerBrokenOnce.Do(func() {
		api.Register(&brokenSchemaBinding{ver: brokenSchemaVer})
	})

	mod := &module.Module{
		APIVersion: brokenSchemaVer,
		Metadata:   &module.ModuleMetadata{Name: "broken", FQN: "x/broken:0.1.0", UUID: "11111111-1111-1111-1111-111111111111"},
		Package:    sharedCtx.CompileString(`apiVersion: "` + string(brokenSchemaVer) + `"`),
	}

	_, err := synth.Release(sharedCtx, synth.ReleaseInput{
		Module:    mod,
		Name:      "myrel",
		Namespace: "default",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, errBrokenSchema,
		"synth.Release must wrap the binding's schema-load error")
	assert.Contains(t, err.Error(), "loading schema",
		"error message must attribute the failure to schema loading")
}
