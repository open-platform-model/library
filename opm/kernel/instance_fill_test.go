package kernel_test

import (
	"context"
	"fmt"
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loaderfile "github.com/open-platform-model/library/opm/helper/loader/file"
	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/kernel"
)

// TestRender_ModuleInstanceFill is the kernel-only half of the parity
// harness's instance-probe plus the self-referential read (spec
// transform-input-fill: "#moduleInstance is filled with the whole evaluated
// instance"). A registrytest-served catalog carries one transformer that
// re-declares `#moduleInstance: _` and emits the instance's name and
// namespace, and reads its own component back through the instance's
// `components` map: the value bound to #moduleInstance contains the very
// component bound to #component, and that must render with no cycle
// (0019 D3). Hermetic: no oracle, no GHCR.
func TestRender_ModuleInstanceFill(t *testing.T) {
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
	instDir := writeImportedInstance(t, dir, "testing.opmodel.dev/library-instance-fill@v0", modPath, version, "probe-demo", "default", "{}", nil)
	platDir := writeCatalogPlatform(t, dir, catPath, version)

	ctx := context.Background()
	k := kernel.New(kernel.WithRegistry(mapping))
	opts := loaderfile.LoadOptions{Registry: mapping}
	inst, err := k.AcquireInstanceFromDir(ctx, instDir, opts)
	require.NoError(t, err, "acquiring the probe instance package")
	plat, err := k.AcquirePlatformFromDir(ctx, platDir, opts)
	require.NoError(t, err, "acquiring the probe platform")

	res, err := k.Render(ctx, kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "opm-test"})
	require.NoError(t, err, "a transformer reading #moduleInstance must render through Kernel.Render")
	require.Len(t, res.Compiled, 1)
	require.Equal(t, txFQN, res.Compiled[0].Transformer)

	obj := res.Compiled[0].Value
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
