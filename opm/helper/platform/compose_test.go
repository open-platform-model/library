package platform_test

import (
	"fmt"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	helperplatform "github.com/open-platform-model/library/opm/helper/platform"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/module"
	pkgplatform "github.com/open-platform-model/library/opm/platform"
)

// makeShell returns a Platform whose #registry is concrete-empty and whose
// computed views are concrete-empty stubs. FillPath into #registry succeeds
// without recomputing the views — used by the tests that check registry
// population in isolation from view recomputation.
func makeShell(t *testing.T, k *kernel.Kernel) *pkgplatform.Platform {
	t.Helper()
	v := k.CueContext().CompileString(`
kind: "Platform"
metadata: { name: "test-shell" }
type: "kubernetes"
#registry: {}
#composedTransformers: {}
#matchers: { resources: {}, traits: {} }
`)
	require.NoError(t, v.Err())
	p, err := pkgplatform.NewPlatformFromValue(k, v)
	require.NoError(t, err)
	return p
}

// makeComputingShell returns a Platform whose #composedTransformers and
// #matchers are CUE comprehensions reading from registered Modules'
// #defines.transformers map. The fixtures use a hand-rolled shape that
// mirrors what the matcher needs without depending on the schema's actual
// #Platform.
func makeComputingShell(t *testing.T, k *kernel.Kernel) *pkgplatform.Platform {
	t.Helper()
	v := k.CueContext().CompileString(computingShellSrc)
	require.NoError(t, v.Err())
	p, err := pkgplatform.NewPlatformFromValue(k, v)
	require.NoError(t, err)
	return p
}

// computingShellSrc carries the comprehension-driven views. Trait
// matchers are intentionally omitted — the resource path exercises the
// same machinery and keeps the fixture compact.
const computingShellSrc = `
kind: "Platform"
metadata: { name: "computing-shell" }
type: "kubernetes"

#registry: [Id=string]: {
	#module!: _
	enabled: bool | *true
}

#composedTransformers: {
	for _, reg in #registry
	if reg.enabled
	if reg.#module.#defines != _|_
	if reg.#module.#defines.transformers != _|_
	for fqn, v in reg.#module.#defines.transformers {
		(fqn): v
	}
}

#matchers: {
	let _resourceFqns = {
		for _, t in #composedTransformers
		if t.requiredResources != _|_
		for fqn, _ in t.requiredResources {
			(fqn): _
		}
	}
	let _resourceCandidates = {
		for fqn, _ in _resourceFqns {
			(fqn): [
				for tname, t in #composedTransformers
				if t.requiredResources != _|_
				if t.requiredResources[fqn] != _|_ {tname},
			]
		}
	}
	resources: _resourceCandidates
	traits: {}
}
`

// makeModule builds a *module.Module with metadata.name and an inlined
// #defines.transformers map. transformers maps a transformer FQN to the
// list of resource FQNs it claims via requiredResources.
func makeModule(t *testing.T, k *kernel.Kernel, name string, transformers map[string][]string) *module.Module {
	t.Helper()
	var b strings.Builder
	fmt.Fprintf(&b, `kind: "Module"
metadata: {
	name: %q
	modulePath: "example.com/m"
	version: "0.1.0"
	fqn: "example.com/m/%s:0.1.0"
	uuid: "00000000-0000-0000-0000-000000000000"
}
#defines: transformers: {
`, name, name)
	for tfqn, reqs := range transformers {
		fmt.Fprintf(&b, "\t%q: { requiredResources: {\n", tfqn)
		for _, r := range reqs {
			fmt.Fprintf(&b, "\t\t%q: {}\n", r)
		}
		b.WriteString("\t} }\n")
	}
	b.WriteString("}\n")

	v := k.CueContext().CompileString(b.String())
	require.NoError(t, v.Err())
	m, err := module.NewModuleFromValue(k, v)
	require.NoError(t, err)
	return m
}

// TestCompose_EmptyModules confirms that an empty modules slice returns a
// Platform identical to the shell. The shell's empty #registry stays empty.
func TestCompose_EmptyModules(t *testing.T) {
	k := kernel.New()
	shell := makeShell(t, k)

	got, err := helperplatform.Compose(k, shell, nil)
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, shell.Metadata.Name, got.Metadata.Name)

	registry := got.Package.LookupPath(cue.ParsePath("#registry"))
	require.True(t, registry.Exists())
	iter, err := registry.Fields()
	require.NoError(t, err)
	assert.False(t, iter.Next(), "#registry remains empty when no modules are passed")
}

// TestCompose_SingleModule checks that one module produces a #registry
// entry under metadata.name and the computing shell's views populate.
func TestCompose_SingleModule(t *testing.T) {
	k := kernel.New()
	shell := makeComputingShell(t, k)
	m := makeModule(t, k, "mod-a", map[string][]string{
		"example.com/p/echo@v0": {"example.com/r/echo@v0"},
	})

	got, err := helperplatform.Compose(k, shell, []*module.Module{m})
	require.NoError(t, err)

	entry := got.Package.LookupPath(cue.MakePath(cue.Def("registry"), cue.Str("mod-a")))
	require.True(t, entry.Exists(), "registry entry under metadata.name must exist")

	enabled, err := entry.LookupPath(cue.ParsePath("enabled")).Bool()
	require.NoError(t, err)
	assert.True(t, enabled, "enabled is set true explicitly")

	composed := got.Package.LookupPath(cue.ParsePath("#composedTransformers"))
	require.True(t, composed.Exists())
	tfEntry := composed.LookupPath(cue.MakePath(cue.Str("example.com/p/echo@v0")))
	require.True(t, tfEntry.Exists(), "#composedTransformers populates from the module")

	resources := got.Package.LookupPath(cue.ParsePath("#matchers.resources"))
	require.True(t, resources.Exists())
	candidate := resources.LookupPath(cue.MakePath(cue.Str("example.com/r/echo@v0")))
	require.True(t, candidate.Exists(), "#matchers.resources index populates from transformer requiredResources")
}

// TestCompose_TwoDisjointModules verifies both modules' transformers appear
// in #composedTransformers when their FQNs do not conflict.
func TestCompose_TwoDisjointModules(t *testing.T) {
	k := kernel.New()
	shell := makeComputingShell(t, k)
	a := makeModule(t, k, "mod-a", map[string][]string{
		"example.com/p/a@v0": {"example.com/r/foo@v0"},
	})
	b := makeModule(t, k, "mod-b", map[string][]string{
		"example.com/p/b@v0": {"example.com/r/bar@v0"},
	})

	got, err := helperplatform.Compose(k, shell, []*module.Module{a, b})
	require.NoError(t, err)

	composed := got.Package.LookupPath(cue.ParsePath("#composedTransformers"))
	require.True(t, composed.Exists())
	for _, tfqn := range []string{"example.com/p/a@v0", "example.com/p/b@v0"} {
		assert.True(t, composed.LookupPath(cue.MakePath(cue.Str(tfqn))).Exists(),
			"transformer %q must appear in #composedTransformers", tfqn)
	}
}

// TestCompose_TwoModulesSharingResourceFQN_Compose asserts that two
// modules whose transformers both require the same resource FQN compose
// successfully. The runtime matcher resolves which transformers actually
// fire by predicate at match time; the helper does not pre-empt that.
func TestCompose_TwoModulesSharingResourceFQN_Compose(t *testing.T) {
	k := kernel.New()
	shell := makeComputingShell(t, k)
	a := makeModule(t, k, "mod-a", map[string][]string{
		"example.com/p/a@v0": {"example.com/r/echo@v0"},
	})
	b := makeModule(t, k, "mod-b", map[string][]string{
		"example.com/p/b@v0": {"example.com/r/echo@v0"},
	})

	got, err := helperplatform.Compose(k, shell, []*module.Module{a, b})
	require.NoError(t, err)
	require.NotNil(t, got)

	composed := got.Package.LookupPath(cue.ParsePath("#composedTransformers"))
	require.True(t, composed.Exists())
	for _, tfqn := range []string{"example.com/p/a@v0", "example.com/p/b@v0"} {
		assert.True(t, composed.LookupPath(cue.MakePath(cue.Str(tfqn))).Exists(),
			"transformer %q must appear in #composedTransformers", tfqn)
	}
}

// TestCompose_Idempotency confirms two Compose calls with the same inputs
// produce equal Package values.
func TestCompose_Idempotency(t *testing.T) {
	k := kernel.New()
	shell := makeComputingShell(t, k)
	m := makeModule(t, k, "mod-a", map[string][]string{
		"example.com/p/echo@v0": {"example.com/r/echo@v0"},
	})

	got1, err := helperplatform.Compose(k, shell, []*module.Module{m})
	require.NoError(t, err)
	got2, err := helperplatform.Compose(k, shell, []*module.Module{m})
	require.NoError(t, err)

	assert.True(t, got1.Package.Equals(got2.Package),
		"two Compose calls with the same inputs must yield equal Package values")
}

// TestCompose_DoesNotMutateInputs verifies the shell Platform's Package
// and each input Module's Package are unchanged after Compose returns.
func TestCompose_DoesNotMutateInputs(t *testing.T) {
	k := kernel.New()
	shell := makeComputingShell(t, k)
	shellSnapshot := shell.Package
	m := makeModule(t, k, "mod-a", map[string][]string{
		"example.com/p/echo@v0": {"example.com/r/echo@v0"},
	})
	moduleSnapshot := m.Package

	_, err := helperplatform.Compose(k, shell, []*module.Module{m})
	require.NoError(t, err)

	assert.True(t, shell.Package.Equals(shellSnapshot),
		"shell.Package must be unchanged after Compose")
	assert.True(t, m.Package.Equals(moduleSnapshot),
		"module.Package must be unchanged after Compose")

	registry := shell.Package.LookupPath(cue.ParsePath("#registry"))
	iter, err := registry.Fields()
	require.NoError(t, err)
	assert.False(t, iter.Next(), "shell's #registry must remain empty")
}

func TestCompose_RejectsNilOwner(t *testing.T) {
	_, err := helperplatform.Compose(nil, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "kernel is required")
}

func TestCompose_RejectsNilShell(t *testing.T) {
	k := kernel.New()
	_, err := helperplatform.Compose(k, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "shell platform is required")
}

func TestCompose_RejectsModuleWithoutName(t *testing.T) {
	k := kernel.New()
	shell := makeShell(t, k)
	m := &module.Module{
		Metadata: &module.ModuleMetadata{Name: ""},
		Package:  k.CueContext().CompileString(`{}`),
	}
	_, err := helperplatform.Compose(k, shell, []*module.Module{m})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "metadata.name")
}

func TestComposePlatform_KernelWrapper(t *testing.T) {
	k := kernel.New()
	shell := makeComputingShell(t, k)
	m := makeModule(t, k, "mod-a", map[string][]string{
		"example.com/p/echo@v0": {"example.com/r/echo@v0"},
	})

	got, err := k.ComposePlatform(shell, []*module.Module{m})
	require.NoError(t, err)
	want, err := helperplatform.Compose(k, shell, []*module.Module{m})
	require.NoError(t, err)

	assert.True(t, got.Package.Equals(want.Package),
		"kernel wrapper must produce the same Package as the helper")
}
