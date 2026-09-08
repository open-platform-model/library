package kernel_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/internal/schematest"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/schema"
)

// newSynthKernel returns a fresh kernel.Kernel configured with the
// workspace-local CUE cache. Used by guard tests that fail before any synth
// build runs (so they need no published module).
func newSynthKernel(t *testing.T) *kernel.Kernel {
	t.Helper()
	schematest.SetEnv(t)
	return kernel.New()
}

// synthModuleFixture authors a #Module (bodyFields is the text after the
// metadata block) against the default (v2) core as an in-memory registry
// fixture, returning the module's major-free path and the fixture.
//
// Per core v2's identity rules (D8), the module is published at its
// snake_case leaf, metadata.name IS that leaf, and metadata.modulePath is the
// full module path with the major suffix.
func synthModuleFixture(t *testing.T, name, version, bodyFields string) (string, registrytest.ModuleFixture) {
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

	return modPath, registrytest.ModuleFixture{Path: modPath, Version: version, File: file.String()}
}

// acquireSynthModule loads the published module back through
// Kernel.AcquireModuleFromRegistry, WITH source: synth builds the instance
// inside the module's own staged root, so the module must carry its source.
func acquireSynthModule(t *testing.T, k *kernel.Kernel, modPath, version string) *module.Module {
	t.Helper()
	mod, err := k.AcquireModuleFromRegistry(context.Background(), modPath+"@v0", "v"+version)
	require.NoErrorf(t, err, "acquiring published module %s@v%s", modPath, version)
	require.True(t, mod.HasSource(), "acquired module must carry staged source for synth")
	return mod
}

// publishSynthModule publishes a #Module (bodyFields is the text after the
// metadata block) to an in-memory registry against the default (v2) core,
// then returns a Kernel wired to that registry (plus opts) and the module
// loaded back through Kernel.AcquireModuleFromRegistry. This mirrors how a
// frontend acquires a module before synthesizing an instance: synthesis
// imports the module by its registry path, so the module MUST be resolvable
// from a registry — a locally-built value no longer works.
func publishSynthModule(t *testing.T, name, version, bodyFields string, opts ...kernel.Option) (*kernel.Kernel, *module.Module) {
	t.Helper()

	modPath, fixture := synthModuleFixture(t, name, version, bodyFields)
	reg := registrytest.NewModuleRegistry(t, []registrytest.ModuleFixture{fixture}, nil)

	k := kernel.New(append([]kernel.Option{kernel.WithRegistry(reg)}, opts...)...)
	return k, acquireSynthModule(t, k, modPath, version)
}

const kernelSynthConfigBody = "#components: {}\n#config: {sentinel: string | *\"ok\"}\ndebugValues: {sentinel: \"from-debug\"}\n"

func TestKernel_SynthesizeInstance_HappyPath(t *testing.T) {
	k, mod := publishSynthModule(t, "demo", "0.1.0", kernelSynthConfigBody)

	values := k.CueContext().CompileString(`sentinel: "from-values"`)
	require.NoError(t, values.Err())

	inst, err := k.SynthesizeInstance(context.Background(), kernel.InstanceInput{
		Module:    mod,
		Name:      "myrel",
		Namespace: "default",
		Values:    []kernel.Source{{Value: values, Origin: "values.cue"}},
	})
	require.NoError(t, err)
	require.NotNil(t, inst)

	assert.Equal(t, "myrel", inst.Metadata.Name)
	assert.Equal(t, "default", inst.Metadata.Namespace)
	assert.NotEmpty(t, inst.Metadata.UUID, "instance UUID must be schema-derived")
}

func TestKernel_SynthesizeInstance_NilModuleRejectedBeforeValidation(t *testing.T) {
	k := newSynthKernel(t)
	_, err := k.SynthesizeInstance(context.Background(), kernel.InstanceInput{
		Name:      "myrel",
		Namespace: "default",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, oerrors.ErrMissingModule),
		"nil Module must be refused before any build, not by the concreteness check")
}

// instance-synthesis spec, "Subpackage module is refused": the synthesized
// package imports the module by its module path, which resolves to the
// module's root package, so a module acquired from a subdirectory cannot be
// synthesized. The refusal names the requirement and runs no build.
func TestKernel_SynthesizeInstance_SubpackageModuleRefused(t *testing.T) {
	k := newSynthKernel(t)
	root := writeTempModuleRoot(t, "sub", acquireModuleFixture)

	mod, err := k.AcquireModuleFromDir(context.Background(), filepath.Join(root, "sub"))
	require.NoError(t, err)
	require.Equal(t, "sub", mod.Source.Pkg)

	inst, err := k.SynthesizeInstance(context.Background(), kernel.InstanceInput{
		Module:    mod,
		Name:      "myrel",
		Namespace: "default",
	})
	require.Error(t, err)
	assert.Nil(t, inst)
	assert.Contains(t, err.Error(), "root package")
	assert.Contains(t, err.Error(), `"sub"`)

	// The refusal precedes the build: nothing in the message comes from CUE.
	assert.NotContains(t, err.Error(), "cannot find package")
}

func TestKernel_SynthesizeInstance_UnconcreteRejected(t *testing.T) {
	// Module declares a required #config field with no default. Omitting
	// Values means the kernel's concreteness check must fail.
	k, mod := publishSynthModule(t, "demo", "0.1.0",
		"#components: {}\n#config: {required!: string}\ndebugValues: {required: \"from-debug\"}\n")

	_, err := k.SynthesizeInstance(context.Background(), kernel.InstanceInput{
		Module:    mod,
		Name:      "myrel",
		Namespace: "default",
		// Values omitted; the schema's `required!: string` has no default,
		// so the concreteness check downstream MUST reject.
	})
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "cannot find package",
		"failure must be the concreteness check, not module resolution")
}

// instance-synthesis spec, "Instance synthesis input": InstanceInput carries
// no schema cache — the Kernel's own is the only one — so a caller supplies
// identity and values and nothing else. schema-dispatch spec, "Synthesis on a
// pinned kernel loads no schema": the default loader pins an exact release,
// so the core import major is read off the pin and the cache stays unloaded.
func TestKernel_SynthesizeInstance_PinnedKernelLoadsNoSchema(t *testing.T) {
	k, mod := publishSynthModule(t, "demo", "0.1.0", kernelSynthConfigBody)

	inst, err := k.SynthesizeInstance(context.Background(), kernel.InstanceInput{
		Module:    mod,
		Name:      "myrel",
		Namespace: "default",
		Values:    []kernel.Source{mustSource(t, k, "values.cue", `sentinel: "from-values"`)},
	})
	require.NoError(t, err)
	require.NotNil(t, inst)

	_, found := reflect.TypeOf(kernel.InstanceInput{}).FieldByName("SchemaCache")
	assert.False(t, found, "InstanceInput carries no schema cache; the Kernel owns the only one")
	assert.Empty(t, k.SchemaCache().ResolvedVersion(), "a pinned kernel synthesizes without loading the schema")
	assert.Contains(t, synthesizedInstanceFile(t, inst), `"opmodel.dev/core@v2"`,
		"the synthesized package imports core at the pinned release's major")
}

// schema-dispatch spec, "Synthesis on a bare-major kernel resolves through
// the cache": a loader naming only the major has no release to read, so the
// kernel loads the schema once and derives the import major from what it
// resolved. Resolving the bare major reaches the public registry for the
// line's latest release, like the schema loader tests do.
func TestKernel_SynthesizeInstance_BareMajorKernelResolvesThroughCache(t *testing.T) {
	k, mod := publishSynthModule(t, "demo", "0.1.0", kernelSynthConfigBody,
		kernel.WithSchemaLoader(schema.OCILoader{Module: "opmodel.dev/core@v2"}))

	inst, err := k.SynthesizeInstance(context.Background(), kernel.InstanceInput{
		Module:    mod,
		Name:      "myrel",
		Namespace: "default",
		Values:    []kernel.Source{mustSource(t, k, "values.cue", `sentinel: "from-values"`)},
	})
	require.NoError(t, err)
	require.NotNil(t, inst)

	resolved := k.SchemaCache().ResolvedVersion()
	assert.True(t, strings.HasPrefix(resolved, "v2."), "the cache resolved a v2 release, got %q", resolved)
	assert.Contains(t, synthesizedInstanceFile(t, inst), `"opmodel.dev/core@v2"`,
		"the synthesized package imports core at the resolved release's major")
}

// The kernel no longer checks the schema it resolved for #ModuleInstance
// before synthesizing: the core the module's own cue.mod resolves inside the
// build is the one the synthesized package imports, and a core lacking the
// definition fails there, at the import that needs it, with CUE's own error
// naming it. The stand-in core carries #Module (so the module acquires) and
// nothing else.
func TestKernel_SynthesizeInstance_CoreWithoutModuleInstanceFailsInBuild(t *testing.T) {
	const coreWithoutModuleInstance = `package core

#Module: {
	kind: "Module"
	metadata: {
		name!:       string
		modulePath!: string
		version!:    string
	}
	...
}
`
	modPath, fixture := synthModuleFixture(t, "demo", "0.1.0", kernelSynthConfigBody)
	reg := registrytest.NewRegistryWithCore(t, coreWithoutModuleInstance, fixture)
	k := kernel.New(kernel.WithRegistry(reg))
	mod := acquireSynthModule(t, k, modPath, "0.1.0")

	inst, err := k.SynthesizeInstance(context.Background(), kernel.InstanceInput{
		Module:    mod,
		Name:      "myrel",
		Namespace: "default",
		Values:    []kernel.Source{mustSource(t, k, "values.cue", `sentinel: "from-values"`)},
	})
	require.Error(t, err)
	assert.Nil(t, inst)
	assert.Contains(t, err.Error(), "#ModuleInstance", "the build names the missing definition")
	assert.False(t, errors.Is(err, oerrors.ErrSchemaUnavailable), "the failure is the build's, not a pre-build schema check")
	assert.Empty(t, k.SchemaCache().ResolvedVersion(), "no schema load ran")
}

// synthesizedInstanceFile returns the instance.cue synthesis staged for inst.
func synthesizedInstanceFile(t *testing.T, inst *module.Instance) string {
	t.Helper()
	require.NotNil(t, inst.Source)
	data, ok := inst.Source.Overlay[filepath.Join(inst.Source.Root, inst.Source.Pkg, "instance.cue")]
	require.True(t, ok, "the synthesized instance.cue is in the staged overlay")
	return string(data)
}

// instance-synthesis spec, "Required inputs validated": each missing required
// field wraps the sentinel that names it, before any build runs.
func TestKernel_SynthesizeInstance_MissingInputsWrapSentinels(t *testing.T) {
	k, mod := publishSynthModule(t, "demo", "0.1.0", kernelSynthConfigBody)
	ctx := context.Background()

	sourceless, err := module.NewModuleFromValue(mod.Package)
	require.NoError(t, err)
	require.False(t, sourceless.HasSource())

	for name, tc := range map[string]struct {
		in       kernel.InstanceInput
		sentinel error
	}{
		"no module":    {kernel.InstanceInput{Name: "myrel", Namespace: "default"}, oerrors.ErrMissingModule},
		"no name":      {kernel.InstanceInput{Module: mod, Namespace: "default"}, oerrors.ErrMissingName},
		"no namespace": {kernel.InstanceInput{Module: mod, Name: "myrel"}, oerrors.ErrMissingNamespace},
		"no source":    {kernel.InstanceInput{Module: sourceless, Name: "myrel", Namespace: "default"}, oerrors.ErrMissingSource},
	} {
		t.Run(name, func(t *testing.T) {
			inst, err := k.SynthesizeInstance(ctx, tc.in)
			require.Error(t, err)
			assert.Nil(t, inst)
			assert.True(t, errors.Is(err, tc.sentinel), "want %v, got %v", tc.sentinel, err)
			assert.NotContains(t, err.Error(), "cannot find package", "the refusal precedes any build")
		})
	}
}

// instance-synthesis spec, "Layered values unify in order": a later source
// sets what an earlier one leaves open, and the merged stack reaches the
// synthesized package's values.
func TestKernel_SynthesizeInstance_LayeredSourcesUnifyInOrder(t *testing.T) {
	k, mod := publishSynthModule(t, "demo", "0.1.0",
		"#components: {}\n#config: {image: string, replicas: int | *1}\ndebugValues: {}\n")

	inst, err := k.SynthesizeInstance(context.Background(), kernel.InstanceInput{
		Module:    mod,
		Name:      "myrel",
		Namespace: "default",
		Values: []kernel.Source{
			mustSource(t, k, "/values/base.cue", `image: "nginx:1.27"`),
			mustSource(t, k, "/values/prod.cue", `replicas: 3`),
		},
	})
	require.NoError(t, err)

	image, err := inst.Package.LookupPath(cue.ParsePath("values.image")).String()
	require.NoError(t, err)
	assert.Equal(t, "nginx:1.27", image, "the first source's field")
	replicas, err := inst.Package.LookupPath(cue.ParsePath("values.replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(3), replicas, "the second source's field")
}

// kernel-runtime spec, "SynthesizeInstance attributes a values violation to
// its source": the post-build #config check runs at the sources' own
// positions, so the error names the Origin rather than the rendered values
// file.
func TestKernel_SynthesizeInstance_ViolationAttributedToSource(t *testing.T) {
	k, mod := publishSynthModule(t, "demo", "0.1.0",
		"#components: {}\n#config: {replicas: int | *1}\ndebugValues: {}\n")

	inst, err := k.SynthesizeInstance(context.Background(), kernel.InstanceInput{
		Module:    mod,
		Name:      "myrel",
		Namespace: "default",
		Values:    []kernel.Source{mustSource(t, k, "/values/bad.cue", `replicas: "three"`)},
	})
	require.Error(t, err)
	assert.Nil(t, inst)
	assert.Contains(t, err.Error(), "replicas")
	assert.True(t, positionsName(err, "/values/bad.cue"), "no position names the source: %v", err)
}

func TestKernel_SynthesizeInstance_UsesKernelContext(t *testing.T) {
	k, mod := publishSynthModule(t, "demo", "0.1.0", kernelSynthConfigBody)
	values := k.CueContext().CompileString(`sentinel: "from-values"`)
	require.NoError(t, values.Err())

	inst, err := k.SynthesizeInstance(context.Background(), kernel.InstanceInput{
		Module:    mod,
		Name:      "myrel",
		Namespace: "default",
		Values:    []kernel.Source{{Value: values, Origin: "values.cue"}},
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
// SynthesizeInstance stamps the tree synthesis built onto
// Instance.Source — overlay mode, rooted at the module's staged root, with
// the synthesized package under the reserved subdirectory.
func TestKernel_SynthesizeInstance_CarriesSource(t *testing.T) {
	k, mod := publishSynthModule(t, "demo", "0.1.0", kernelSynthConfigBody)
	values := k.CueContext().CompileString(`sentinel: "from-values"`)
	require.NoError(t, values.Err())

	inst, err := k.SynthesizeInstance(context.Background(), kernel.InstanceInput{
		Module:    mod,
		Name:      "myrel",
		Namespace: "default",
		Values:    []kernel.Source{{Value: values, Origin: "values.cue"}},
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

	inst, err := k.SynthesizeInstance(context.Background(), kernel.InstanceInput{
		Module: mod, Name: "myrel", Namespace: "default",
	})
	require.Error(t, err)
	assert.Nil(t, inst)
}
