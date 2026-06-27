package synth_test

import (
	"errors"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/helper/synth"
	"github.com/open-platform-model/library/opm/internal/schematest"
	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/schema"
)

// sharedCtx is the single *cue.Context used by the guard tests in this package.
// The schema cache must produce values in the same runtime that a test's other
// cue.Values live in; mixing contexts panics cross-runtime unification. The
// import-based integration tests (instance_integration_test.go) construct their
// own per-test contexts.
var sharedCtx = cuecontext.New()

// testdataSynthDir resolves the on-disk path to the testdata/synth/
// fixture directory relative to this test file.
func testdataSynthDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(schematest.LibraryRoot(t), "testdata", "synth")
}

// testModule loads a #Module from a synthtest/ fixture for the guard tests. The
// fixture imports "opmodel.dev/core@v1" which resolves through
// testdata/cue.mod/module.cue's deps against CUE_REGISTRY (configured by
// schematest.SetEnv). Guard tests only need a well-formed *module.Module to feed
// the required-input checks; they return before any synthesized build runs, so
// the module need not be published to a registry.
func testModule(t *testing.T, ctx *cue.Context, src string) *module.Module {
	t.Helper()
	schematest.SetEnv(t)

	moduleRoot := testdataSynthDir(t)
	fixturePath := filepath.Join(moduleRoot, "fixture.cue")
	cfg := &load.Config{
		Dir: moduleRoot,
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

// baseModuleFixture is the minimal #Module declaration used by guard tests that
// don't need a custom #config or debugValues. Name/modulePath/version are fixed
// so the derived module UUID is stable.
const baseModuleFixture = `
package synthtest

import core "opmodel.dev/core@v1"

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

// newCache builds a fresh *schema.Cache via the workspace-local cache helper.
// Each test gets its own cache so memoization scope is explicit.
func newCache(t *testing.T) *schema.Cache {
	t.Helper()
	return schematest.NewCache(t)
}

func TestInstance_RejectsNilModule(t *testing.T) {
	ctx := sharedCtx
	_, err := synth.Instance(ctx, synth.InstanceInput{
		Name:        "demo",
		Namespace:   "ns",
		SchemaCache: newCache(t),
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, synth.ErrMissingModule), "want ErrMissingModule, got %v", err)
}

func TestInstance_RejectsEmptyName(t *testing.T) {
	ctx := sharedCtx
	mod := testModule(t, ctx, baseModuleFixture)
	_, err := synth.Instance(ctx, synth.InstanceInput{
		Module:      mod,
		Namespace:   "ns",
		SchemaCache: newCache(t),
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, synth.ErrMissingName), "want ErrMissingName, got %v", err)
}

func TestInstance_RejectsEmptyNamespace(t *testing.T) {
	ctx := sharedCtx
	mod := testModule(t, ctx, baseModuleFixture)
	_, err := synth.Instance(ctx, synth.InstanceInput{
		Module:      mod,
		Name:        "demo",
		SchemaCache: newCache(t),
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, synth.ErrMissingNamespace), "want ErrMissingNamespace, got %v", err)
}

func TestInstance_RejectsNilSchemaCache(t *testing.T) {
	ctx := sharedCtx
	mod := testModule(t, ctx, baseModuleFixture)
	_, err := synth.Instance(ctx, synth.InstanceInput{
		Module:    mod,
		Name:      "demo",
		Namespace: "ns",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, synth.ErrMissingSchemaCache),
		"want ErrMissingSchemaCache, got %v", err)
}

// expectedInstanceUUID computes the canonical instance UUID through CUE so the
// derived-field tests stay in lockstep with the schema's definition. Failing
// this assertion is the drift sentinel for module_instance.cue.
func expectedInstanceUUID(t *testing.T, ctx *cue.Context, moduleUUID, name, namespace string) string {
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
