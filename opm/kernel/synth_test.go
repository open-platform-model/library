package kernel_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/helper/synth"
	"github.com/open-platform-model/library/opm/internal/schematest"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/module"
)

// newSynthKernel returns a fresh kernel.Kernel configured with the
// workspace-local CUE cache. Each test gets its own Kernel so the
// schema-cache lifetime is per-test and explicit. Tests in this file
// MUST route every synth call through the returned Kernel — the
// Kernel's *cue.Context owns every cue.Value the synth path consumes,
// and a second Kernel in the same test would seat a different runtime
// in its own *schema.Cache, tripping the "values are not from the same
// runtime" contract inside cue.Value.FillPath.
func newSynthKernel(t *testing.T) *kernel.Kernel {
	t.Helper()
	schematest.SetEnv(t)
	return kernel.New()
}

// testdataSynthDir resolves the on-disk path to library/testdata/synth/
// relative to this test file.
func testdataSynthDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(schematest.LibraryRoot(t), "testdata", "synth")
}

// synthTestModule builds a *module.Module against the Kernel's context
// by loading a synthtest fixture rooted at library/testdata/synth/. The
// Kernel's CueContext owns every cue.Value the synth path consumes, so
// the helper threads k.CueContext() through load.Instances.
func synthTestModule(t *testing.T, k *kernel.Kernel, src string) *module.Module {
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
	require.Len(t, insts, 1)
	require.NoError(t, insts[0].Err)

	pkg := k.CueContext().BuildInstance(insts[0])
	require.NoErrorf(t, pkg.Err(), "building fixture: %v", pkg.Err())
	modVal := pkg.LookupPath(cue.ParsePath("module"))
	require.True(t, modVal.Exists())
	mod, err := k.NewModuleFromValue(modVal)
	require.NoError(t, err)
	return mod
}

const kernelSynthBaseFixture = `
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
	#config: { sentinel: string | *"ok" }
	debugValues: { sentinel: "from-debug" }
}
`

func TestKernel_SynthesizeRelease_HappyPath(t *testing.T) {
	k := newSynthKernel(t)
	mod := synthTestModule(t, k, kernelSynthBaseFixture)

	values := k.CueContext().CompileString(`sentinel: "from-values"`)
	require.NoError(t, values.Err())

	rel, err := k.SynthesizeRelease(context.Background(), synth.ReleaseInput{
		Module:      mod,
		Name:        "myrel",
		Namespace:   "default",
		Values:      values,
		SchemaCache: k.SchemaCache(),
	})
	require.NoError(t, err)
	require.NotNil(t, rel)

	assert.Equal(t, "myrel", rel.Metadata.Name)
	assert.Equal(t, "default", rel.Metadata.Namespace)
	assert.NotEmpty(t, rel.Metadata.UUID, "release UUID must be schema-derived")
}

func TestKernel_SynthesizeRelease_NilModuleRejectedBeforeValidation(t *testing.T) {
	k := newSynthKernel(t)
	_, err := k.SynthesizeRelease(context.Background(), synth.ReleaseInput{
		Name:        "myrel",
		Namespace:   "default",
		SchemaCache: k.SchemaCache(),
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, synth.ErrMissingModule),
		"nil Module must error from synth.Release, not ProcessModuleRelease")
}

func TestKernel_SynthesizeRelease_UnconcreteRejected(t *testing.T) {
	k := newSynthKernel(t)
	// Module declares a required #config field with no default. Omitting
	// Values means ProcessModuleRelease's concreteness check must fail.
	mod := synthTestModule(t, k, `
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
	#config: { required!: string }
	debugValues: { required: "from-debug" }
}
`)

	_, err := k.SynthesizeRelease(context.Background(), synth.ReleaseInput{
		Module:      mod,
		Name:        "myrel",
		Namespace:   "default",
		SchemaCache: k.SchemaCache(),
		// Values omitted; the schema's `required!: string` has no default,
		// so the concreteness check downstream MUST reject.
	})
	require.Error(t, err)
}

// TestKernel_SynthesizeRelease_DefaultSchemaCache asserts that omitting
// SchemaCache on ReleaseInput falls back to the kernel-owned cache via
// SynthesizeRelease's defaulting.
func TestKernel_SynthesizeRelease_DefaultSchemaCache(t *testing.T) {
	k := newSynthKernel(t)
	mod := synthTestModule(t, k, kernelSynthBaseFixture)

	values := k.CueContext().CompileString(`sentinel: "from-values"`)
	require.NoError(t, values.Err())

	rel, err := k.SynthesizeRelease(context.Background(), synth.ReleaseInput{
		Module:    mod,
		Name:      "myrel",
		Namespace: "default",
		Values:    values,
		// SchemaCache intentionally omitted; SynthesizeRelease fills it from
		// k.SchemaCache().
	})
	require.NoError(t, err)
	require.NotNil(t, rel)
}

func TestKernel_SynthesizeRelease_UsesKernelContext(t *testing.T) {
	k := newSynthKernel(t)
	mod := synthTestModule(t, k, kernelSynthBaseFixture)
	values := k.CueContext().CompileString(`sentinel: "from-values"`)
	require.NoError(t, values.Err())

	rel, err := k.SynthesizeRelease(context.Background(), synth.ReleaseInput{
		Module:      mod,
		Name:        "myrel",
		Namespace:   "default",
		Values:      values,
		SchemaCache: k.SchemaCache(),
	})
	require.NoError(t, err)
	require.NotNil(t, rel)

	// Cross-runtime sanity check: a cue.Value built with k.CueContext() must
	// unify with rel.Package without triggering "values are not from the
	// same runtime". The unification succeeds only if both values share the
	// kernel's context, proving SynthesizeRelease threaded the kernel's
	// *cue.Context end-to-end.
	probe := k.CueContext().CompileString(`metadata: name: "myrel"`)
	require.NoError(t, probe.Err())
	merged := rel.Package.Unify(probe)
	require.NoError(t, merged.Err())
}
