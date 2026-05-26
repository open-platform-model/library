package kernel_test

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/helper/synth"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/module"
)

// synthKernel is the single Kernel shared by every synth-using test in this
// package (this file plus flow_synth_integration_test.go).
//
// The schema package caches its loaded cue.Value against the first
// *cue.Context that calls SchemaValue. Tests that build releases from typed
// inputs MUST route every synth call through this kernel so the cached
// cue.Value lives in the same runtime as every other cue.Value the test
// produces — otherwise synth.Release panics with "incompatible runtime"
// inside CompileString(... cue.Scope(...)). Each kernel.New() in a separate
// test would seat a different first context in the schema cache and trip
// that contract.
var synthKernel = kernel.New()

// kernelSynthApisCoreDir resolves apis/core relative to this test file.
// Mirrors opm/helper/synth/release_test.go's apisCoreDir.
func kernelSynthApisCoreDir(t *testing.T) string {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	require.True(t, ok)
	// opm/kernel/ → library/
	libRoot := filepath.Clean(filepath.Join(filepath.Dir(here), "..", ".."))
	return filepath.Join(libRoot, "apis", "core")
}

// synthTestModule builds a *module.Module against the Kernel's context by
// loading a synthetic fixture rooted at apis/core. The Kernel's CueContext
// MUST own every cue.Value the synth path consumes (schema.SchemaValue caches
// against the first context it sees), so this helper threads k.CueContext()
// through load.Instances.
func synthTestModule(t *testing.T, k *kernel.Kernel, src string) *module.Module {
	t.Helper()

	moduleRoot := kernelSynthApisCoreDir(t)
	fixturePath := filepath.Join(moduleRoot, "synthtest", "fixture.cue")
	cfg := &load.Config{
		Dir: filepath.Join(moduleRoot, "synthtest"),
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
	k := synthKernel
	mod := synthTestModule(t, k, kernelSynthBaseFixture)

	values := k.CueContext().CompileString(`sentinel: "from-values"`)
	require.NoError(t, values.Err())

	rel, err := k.SynthesizeRelease(context.Background(), synth.ReleaseInput{
		Module:    mod,
		Name:      "myrel",
		Namespace: "default",
		Values:    values,
	})
	require.NoError(t, err)
	require.NotNil(t, rel)

	assert.Equal(t, "myrel", rel.Metadata.Name)
	assert.Equal(t, "default", rel.Metadata.Namespace)
	assert.NotEmpty(t, rel.Metadata.UUID, "release UUID must be schema-derived")
}

func TestKernel_SynthesizeRelease_NilModuleRejectedBeforeValidation(t *testing.T) {
	k := synthKernel
	_, err := k.SynthesizeRelease(context.Background(), synth.ReleaseInput{
		Name:      "myrel",
		Namespace: "default",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, synth.ErrMissingModule),
		"nil Module must error from synth.Release, not ProcessModuleRelease")
}

func TestKernel_SynthesizeRelease_UnconcreteRejected(t *testing.T) {
	k := synthKernel
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
		Module:    mod,
		Name:      "myrel",
		Namespace: "default",
		// Values omitted; the schema's `required!: string` has no default,
		// so the concreteness check downstream MUST reject.
	})
	require.Error(t, err)
}

func TestKernel_SynthesizeRelease_UsesKernelContext(t *testing.T) {
	k := synthKernel
	mod := synthTestModule(t, k, kernelSynthBaseFixture)
	values := k.CueContext().CompileString(`sentinel: "from-values"`)
	require.NoError(t, values.Err())

	rel, err := k.SynthesizeRelease(context.Background(), synth.ReleaseInput{
		Module:    mod,
		Name:      "myrel",
		Namespace: "default",
		Values:    values,
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
