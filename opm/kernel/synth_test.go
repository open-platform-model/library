package kernel_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/helper/synth"
	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/internal/schematest"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/module"
)

// newSynthKernel returns a fresh kernel.Kernel configured with the
// workspace-local CUE cache. Used by guard tests that fail before any synth
// build runs (so they need no published module).
func newSynthKernel(t *testing.T) *kernel.Kernel {
	t.Helper()
	schematest.SetEnv(t)
	return kernel.New()
}

// publishSynthModule publishes a #Module (bodyFields is the text after the
// metadata block) to an in-memory registry against the default (v2) core,
// then returns a Kernel wired to that registry plus the module loaded back
// through Kernel.AcquireModuleFromRegistry. This mirrors how a frontend acquires
// a module before synthesizing an instance: synth.Instance imports the module
// by its registry path, so the module MUST be resolvable from a registry — a
// locally-built value no longer works.
//
// Per core v2's identity rules (D8), the module is published at its
// snake_case leaf, metadata.name IS that leaf, and metadata.modulePath is the
// full module path with the major suffix.
func publishSynthModule(t *testing.T, name, version, bodyFields string) (*kernel.Kernel, *module.Module) {
	t.Helper()

	snake := strings.ReplaceAll(name, "-", "_")
	metaPath := registrytest.UniquePath(t, "modules")
	modPath := metaPath + "/" + snake

	var file strings.Builder
	fmt.Fprintf(&file, "package %s\n\n", snake)
	file.WriteString("import core \"opmodel.dev/core@v2\"\n\n")
	file.WriteString("core.#Module\n")
	fmt.Fprintf(&file, "metadata: {\n\tname:       %q\n\tmodulePath: %q\n\tversion:    %q\n}\n", snake, modPath+"@v0", version)
	file.WriteString(bodyFields)

	reg := registrytest.NewModuleRegistry(t, []registrytest.ModuleFixture{{
		Path:    modPath,
		Version: version,
		File:    file.String(),
	}}, nil)

	k := kernel.New(kernel.WithRegistry(reg))
	// Acquire WITH source: synth builds the instance inside the module's own
	// staged root, so the module must carry its source.
	mod, err := k.AcquireModuleFromRegistry(context.Background(), modPath+"@v0", "v"+version)
	require.NoErrorf(t, err, "acquiring published module %s@v%s", modPath, version)
	require.True(t, mod.HasSource(), "acquired module must carry staged source for synth")
	return k, mod
}

const kernelSynthConfigBody = "#components: {}\n#config: {sentinel: string | *\"ok\"}\ndebugValues: {sentinel: \"from-debug\"}\n"

func TestKernel_SynthesizeInstance_HappyPath(t *testing.T) {
	k, mod := publishSynthModule(t, "demo", "0.1.0", kernelSynthConfigBody)

	values := k.CueContext().CompileString(`sentinel: "from-values"`)
	require.NoError(t, values.Err())

	inst, err := k.SynthesizeInstance(context.Background(), synth.InstanceInput{
		Module:      mod,
		Name:        "myrel",
		Namespace:   "default",
		Values:      values,
		SchemaCache: k.SchemaCache(),
	})
	require.NoError(t, err)
	require.NotNil(t, inst)

	assert.Equal(t, "myrel", inst.Metadata.Name)
	assert.Equal(t, "default", inst.Metadata.Namespace)
	assert.NotEmpty(t, inst.Metadata.UUID, "instance UUID must be schema-derived")
}

func TestKernel_SynthesizeInstance_NilModuleRejectedBeforeValidation(t *testing.T) {
	k := newSynthKernel(t)
	_, err := k.SynthesizeInstance(context.Background(), synth.InstanceInput{
		Name:        "myrel",
		Namespace:   "default",
		SchemaCache: k.SchemaCache(),
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, synth.ErrMissingModule),
		"nil Module must error from synth.Instance, not the concreteness check")
}

func TestKernel_SynthesizeInstance_UnconcreteRejected(t *testing.T) {
	// Module declares a required #config field with no default. Omitting
	// Values means the kernel's concreteness check must fail.
	k, mod := publishSynthModule(t, "demo", "0.1.0",
		"#components: {}\n#config: {required!: string}\ndebugValues: {required: \"from-debug\"}\n")

	_, err := k.SynthesizeInstance(context.Background(), synth.InstanceInput{
		Module:      mod,
		Name:        "myrel",
		Namespace:   "default",
		SchemaCache: k.SchemaCache(),
		// Values omitted; the schema's `required!: string` has no default,
		// so the concreteness check downstream MUST reject.
	})
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "cannot find package",
		"failure must be the concreteness check, not module resolution")
}

// TestKernel_SynthesizeInstance_DefaultSchemaCache asserts that omitting
// SchemaCache on InstanceInput falls back to the kernel-owned cache via
// SynthesizeInstance's defaulting.
func TestKernel_SynthesizeInstance_DefaultSchemaCache(t *testing.T) {
	k, mod := publishSynthModule(t, "demo", "0.1.0", kernelSynthConfigBody)

	values := k.CueContext().CompileString(`sentinel: "from-values"`)
	require.NoError(t, values.Err())

	inst, err := k.SynthesizeInstance(context.Background(), synth.InstanceInput{
		Module:    mod,
		Name:      "myrel",
		Namespace: "default",
		Values:    values,
		// SchemaCache intentionally omitted; SynthesizeInstance fills it from
		// k.SchemaCache().
	})
	require.NoError(t, err)
	require.NotNil(t, inst)
}

func TestKernel_SynthesizeInstance_UsesKernelContext(t *testing.T) {
	k, mod := publishSynthModule(t, "demo", "0.1.0", kernelSynthConfigBody)
	values := k.CueContext().CompileString(`sentinel: "from-values"`)
	require.NoError(t, values.Err())

	inst, err := k.SynthesizeInstance(context.Background(), synth.InstanceInput{
		Module:      mod,
		Name:        "myrel",
		Namespace:   "default",
		Values:      values,
		SchemaCache: k.SchemaCache(),
	})
	require.NoError(t, err)
	require.NotNil(t, inst)

	// Cross-runtime sanity check: a cue.Value built with k.CueContext() must
	// unify with inst.Package without triggering "values are not from the
	// same runtime". The unification succeeds only if both values share the
	// kernel's context, proving SynthesizeInstance threaded the kernel's
	// *cue.Context end-to-end.
	probe := k.CueContext().CompileString(`metadata: name: "myrel"`)
	require.NoError(t, probe.Err())
	merged := inst.Package.Unify(probe)
	require.NoError(t, merged.Err())
}

// instance-synthesis spec, "Synthesized instance carries the tree":
// SynthesizeInstance stamps the tree synth.Instance built onto
// Instance.Source — overlay mode, rooted at the module's staged root, with
// the synthesized package under the reserved subdirectory.
func TestKernel_SynthesizeInstance_CarriesSource(t *testing.T) {
	k, mod := publishSynthModule(t, "demo", "0.1.0", kernelSynthConfigBody)
	values := k.CueContext().CompileString(`sentinel: "from-values"`)
	require.NoError(t, values.Err())

	inst, err := k.SynthesizeInstance(context.Background(), synth.InstanceInput{
		Module:    mod,
		Name:      "myrel",
		Namespace: "default",
		Values:    values,
	})
	require.NoError(t, err)
	require.NotNil(t, inst)

	require.NotNil(t, inst.Source, "synthesized instance must carry its staged tree")
	assert.Equal(t, mod.Source.Root, inst.Source.Root, "Root is the module's staged root")
	assert.Equal(t, "opm-synth-instance", inst.Source.Pkg, "Pkg is the reserved instance subdirectory")
	assert.Contains(t, inst.Source.Overlay, filepath.Join(inst.Source.Root, inst.Source.Pkg, "instance.cue"))
	assert.Contains(t, inst.Source.Overlay, filepath.Join(inst.Source.Root, inst.Source.Pkg, "values.cue"))
	assert.Contains(t, inst.Source.Overlay, filepath.Join(inst.Source.Root, "cue.mod", "module.cue"),
		"the module's own module file stays in the tree")
}

// The failure surface is unchanged: a synthesis that fails validation returns
// no instance, so there is nothing to stamp.
func TestKernel_SynthesizeInstance_FailureReturnsNoInstance(t *testing.T) {
	k, mod := publishSynthModule(t, "demo", "0.1.0",
		"#components: {}\n#config: {required!: string}\ndebugValues: {required: \"from-debug\"}\n")

	inst, err := k.SynthesizeInstance(context.Background(), synth.InstanceInput{
		Module: mod, Name: "myrel", Namespace: "default",
	})
	require.Error(t, err)
	assert.Nil(t, inst)
}
