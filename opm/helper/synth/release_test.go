package synth_test

import (
	"errors"
	"path/filepath"
	"runtime"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/helper/synth"
	"github.com/open-platform-model/library/opm/module"
)

// sharedCtx is the single *cue.Context used by every synth test in this
// package. The schema package caches its loaded schema against the first
// *cue.Context that calls SchemaValue. If tests use multiple contexts, the
// second context's values cannot unify with the schema (the schema is bound
// to the first context's runtime), triggering "values are not from the same
// runtime" panics inside cue.Value.FillPath.
var sharedCtx = cuecontext.New()

// apisCoreDir resolves the on-disk path to the apis/core CUE module from
// this test file's location.
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
// caller-supplied src; it imports "opmodel.dev/core@v0" so the local
// module's #Module definition resolves without any registry access.
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

import core "opmodel.dev/core@v0"

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
// assertion is the drift sentinel for module_release.cue.
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

func TestRelease_EmptyValuesLeavesPathUnfilled(t *testing.T) {
	ctx := sharedCtx
	// debugValues intentionally non-empty so we can prove the synth helper
	// does NOT fall back to it.
	mod := testModule(t, ctx, `
package synthtest

import core "opmodel.dev/core@v0"

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

func TestRelease_AutoSecretsComponentInjected(t *testing.T) {
	ctx := sharedCtx
	mod := testModule(t, ctx, `
package synthtest

import core "opmodel.dev/core@v0"

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

func TestRelease_NoCueRegistryNeeded(t *testing.T) {
	// synth.Release goes through schema.SchemaValue, which is entirely
	// off-disk (embed + overlay). Verify the contract by explicitly clearing
	// CUE_REGISTRY before the synth call.
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

func TestRelease_ComponentsAreFannedBySchema(t *testing.T) {
	ctx := sharedCtx
	mod := testModule(t, ctx, `
package synthtest

import core "opmodel.dev/core@v0"

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
