package kernel_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loaderfile "github.com/open-platform-model/library/opm/helper/loader/file"
	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/kernel"
)

// TestCompile_ModuleInstanceFill is the kernel-only half of the parity
// harness's instance-probe plus the self-referential read (spec
// transform-input-fill: "#moduleInstance is filled with the whole evaluated
// instance"). A registrytest-served catalog carries one transformer that
// re-declares `#moduleInstance: _` and emits the instance's name and
// namespace, and reads its own component back through the instance's
// `components` map: the value filled into #moduleInstance contains the very
// component filled into #component, and that must render with no cycle
// (0019 D3). Before library-instance-fill the slot was never filled and the
// pair failed to render. Hermetic: no oracle, no GHCR.
func TestCompile_ModuleInstanceFill(t *testing.T) {
	const version = "0.1.0"
	catPath := registrytest.UniquePath(t, "cat")
	modPath := registrytest.UniquePath(t, "modules") + "/probe_app"
	const txName = "instance-regression"
	txFQN := fmt.Sprintf("%s/transformers/%s@%s", catPath, txName, version)
	containerFQN := resFQN(catPath, "container")

	transform := `{
		#moduleInstance: _
		#component:      _
		output: {
			kind:      "InstanceProbe"
			instance:  #moduleInstance.metadata.name
			namespace: #moduleInstance.metadata.namespace
			// Self-reference: the component this pair renders, read through
			// the instance rather than through #component.
			self: #moduleInstance.components[#component.metadata.name].metadata.name
		}
	}`

	mapping := registrytest.NewModuleRegistry(t,
		[]registrytest.ModuleFixture{
			{Path: modPath, Version: version, File: probeModuleFile(modPath, catPath, containerFQN, version)},
		},
		[]registrytest.CatalogFixture{{
			Path:    catPath,
			Version: version,
			Body:    probeCatalogBody(catPath, version, txName, txFQN, containerFQN, transform),
		}},
	)

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "cue.mod", "module.cue"), fmt.Sprintf(`module: "testing.opmodel.dev/library-instance-fill@v0"
language: version: "v0.17.0"
deps: {
	"opmodel.dev/core@v2": v: "v2.0.0-alpha.4"
	%q: v: %q
}
`, modPath+"@v0", "v"+version))
	writeFile(t, filepath.Join(dir, "instance", "instance.cue"), fmt.Sprintf(`package instance

import (
	core "opmodel.dev/core@v2"
	probe %q
)

core.#ModuleInstance

metadata: {
	name:      "probe-demo"
	namespace: "default"
}

#module: probe
values: {}
`, modPath+"@v0"))

	ctx := context.Background()
	k := kernel.New(kernel.WithRegistry(mapping))

	mod, err := k.AcquireModuleFromRegistry(ctx, modPath+"@v0", "v"+version)
	require.NoError(t, err, "acquiring the probe module")
	mp, err := materializePlatform(t, k, version, catPath)
	require.NoError(t, err, "materializing the probe platform")
	instVal, err := k.LoadInstancePackage(ctx, filepath.Join(dir, "instance"), loaderfile.LoadOptions{Registry: mapping})
	require.NoError(t, err, "loading the probe instance package")
	inst, err := k.ProcessModuleInstance(ctx, instVal, *mod, cue.Value{})
	require.NoError(t, err)

	out, err := k.Compile(ctx, kernel.CompileInput{ModuleInstance: inst, Platform: mp, RuntimeName: "opm-test"})
	require.NoError(t, err, "a transformer reading #moduleInstance must render through Kernel.Compile")
	require.Len(t, out.Compiled, 1)
	require.Equal(t, txFQN, out.Compiled[0].Transformer)

	obj := out.Compiled[0].Value
	require.NoError(t, obj.Validate(cue.Concrete(true)), "rendered object must be concrete")

	for field, want := range map[string]string{
		"instance":  "probe-demo",
		"namespace": "default",
		"self":      "web",
	} {
		got, err := obj.LookupPath(cue.ParsePath(field)).String()
		require.NoError(t, err, field)
		assert.Equal(t, want, got, field)
	}
}
