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

	oerrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/internal/schematest"
	"github.com/open-platform-model/library/opm/internal/synth"
	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/schema"
)

// sharedCtx is the single *cue.Context used by the guard tests in this package.
// A test's cue.Values must all live in one runtime; mixing contexts panics
// cross-runtime unification. The end-to-end synth flows (a published core-v2
// module synthesized and rendered through the kernel) live in opm/kernel's
// synth and flow tests.
var sharedCtx = cuecontext.New()

// coreVersion is the resolved core release the kernel would hand Instance;
// only its major reaches the synthesized import.
var coreVersion = schema.DefaultSchemaVersion()

// testdataSynthDir resolves the on-disk path to the testdata/synth/
// fixture directory relative to this test file.
func testdataSynthDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(schematest.LibraryRoot(t), "testdata", "synth")
}

// testModule loads a #Module from a synthtest/ fixture for the guard tests. The
// fixture imports "opmodel.dev/core@v2" which resolves through
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

	mod, err := module.NewModuleFromValue(modVal)
	require.NoErrorf(t, err, "constructing *module.Module from fixture: %v", err)
	require.NotNil(t, mod)
	require.NotEmpty(t, mod.Metadata.UUID, "schema-derived module UUID must be present")
	return mod
}

// baseModuleFixture is the minimal #Module declaration used by guard tests that
// don't need a custom #config or debugValues. Name/modulePath/version are fixed
// so the derived module UUID is stable.
const baseModuleFixture = `
package synthtest

import core "opmodel.dev/core@v2"

module: {
	core.#Module
	metadata: {
		name:       "demo"
		modulePath: "example.com/demo@v0"
		version:    "0.1.0"
	}
	#components: {}
	#config: {}
	debugValues: {}
}
`

func TestInstance_RejectsNilModule(t *testing.T) {
	_, _, err := synth.Instance(sharedCtx, coreVersion, synth.Input{
		Name:      "demo",
		Namespace: "ns",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, oerrors.ErrMissingModule), "want ErrMissingModule, got %v", err)
}

func TestInstance_RejectsEmptyName(t *testing.T) {
	mod := testModule(t, sharedCtx, baseModuleFixture)
	_, _, err := synth.Instance(sharedCtx, coreVersion, synth.Input{
		Module:    mod,
		Namespace: "ns",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, oerrors.ErrMissingName), "want ErrMissingName, got %v", err)
}

func TestInstance_RejectsEmptyNamespace(t *testing.T) {
	mod := testModule(t, sharedCtx, baseModuleFixture)
	_, _, err := synth.Instance(sharedCtx, coreVersion, synth.Input{
		Module: mod,
		Name:   "demo",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, oerrors.ErrMissingNamespace), "want ErrMissingNamespace, got %v", err)
}

// The kernel resolves the core release from its own cache; an unresolvable one
// is refused here rather than producing an importless synthesized package.
func TestInstance_RejectsEmptyCoreVersion(t *testing.T) {
	mod := testModule(t, sharedCtx, baseModuleFixture)
	_, _, err := synth.Instance(sharedCtx, "", synth.Input{
		Module:    mod,
		Name:      "demo",
		Namespace: "ns",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, oerrors.ErrSchemaUnavailable), "want ErrSchemaUnavailable, got %v", err)
}

// instance-synthesis spec — Instance constructs the instance inside the
// module's own staged source tree, so a module that was not acquired with
// source (HasSource() is false) is rejected with ErrMissingSource rather than
// silently fetching. testModule builds a source-free *module.Module, exercising
// exactly that precondition.
func TestInstance_RejectsMissingSource(t *testing.T) {
	mod := testModule(t, sharedCtx, baseModuleFixture)
	require.False(t, mod.HasSource(), "fixture module must be source-free for this guard")
	_, _, err := synth.Instance(sharedCtx, coreVersion, synth.Input{
		Module:    mod,
		Name:      "demo",
		Namespace: "ns",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, oerrors.ErrMissingSource), "want ErrMissingSource, got %v", err)
}

// instance-synthesis spec, "Failure returns no tree": every error path of
// Instance returns a nil staged tree alongside the error.
func TestInstance_FailureReturnsNoTree(t *testing.T) {
	mod := testModule(t, sharedCtx, baseModuleFixture)

	_, src, err := synth.Instance(sharedCtx, coreVersion, synth.Input{Module: mod, Namespace: "ns"})
	require.Error(t, err)
	assert.Nil(t, src, "ErrMissingName path must return no tree")

	_, src, err = synth.Instance(sharedCtx, coreVersion, synth.Input{Module: mod, Name: "demo", Namespace: "ns"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, oerrors.ErrMissingSource))
	assert.Nil(t, src, "ErrMissingSource path must return no tree")
}
