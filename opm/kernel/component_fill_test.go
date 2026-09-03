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

// TestRender_ComponentFillPreservesDefinitions is the kernel-only half of
// the parity harness's names-probe (spec transform-input-fill: "#component
// is filled with every field class preserved"). A registrytest-served
// catalog carries one transformer that re-declares `#component: _` and
// emits the names core computes for the component; the instance is
// import-authored so `#instance` and `#names` resolve. Rendering through
// Kernel.Render must carry those definitions into the transformer: the
// binding is plain unification inside the build (0019 D1/D3). Hermetic: no
// oracle, no GHCR.
func TestRender_ComponentFillPreservesDefinitions(t *testing.T) {
	const version = "0.1.0"
	catPath := registrytest.UniquePath(t, "cat")
	modPath := registrytest.UniquePath(t, "modules") + "/probe_app"
	const txName = "names-regression"
	txFQN := fmt.Sprintf("%s/transformers/%s@%s", catPath, txName, version)
	containerFQN := resFQN(catPath, "container")

	transform := `{
		#component: _
		output: {
			kind:         "NamesProbe"
			resourceName: #component.#names.resourceName
			fqdn:         #component.#names.dns.fqdn
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
	instDir := writeImportedInstance(t, dir, "testing.opmodel.dev/library-component-fill@v0", modPath, version, "probe-demo", "default", "{}", nil)
	platDir := writeCatalogPlatform(t, dir, catPath, version)

	ctx := context.Background()
	k := kernel.New(kernel.WithRegistry(mapping))
	opts := loaderfile.LoadOptions{Registry: mapping}
	inst, err := k.AcquireInstanceFromDir(ctx, instDir, opts)
	require.NoError(t, err, "acquiring the probe instance package")
	plat, err := k.AcquirePlatformFromDir(ctx, platDir, opts)
	require.NoError(t, err, "acquiring the probe platform")

	res, err := k.Render(ctx, kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "opm-test"})
	require.NoError(t, err, "a transformer reading #component.#names must render through Kernel.Render")
	require.Len(t, res.Compiled, 1)
	require.Equal(t, txFQN, res.Compiled[0].Transformer)

	obj := res.Compiled[0].Value
	require.NoError(t, obj.Validate(cue.Concrete(true)), "rendered object must be concrete")

	resourceName, err := obj.LookupPath(cue.ParsePath("resourceName")).String()
	require.NoError(t, err)
	fqdn, err := obj.LookupPath(cue.ParsePath("fqdn")).String()
	require.NoError(t, err)

	// The values core computes for component `web` of instance
	// probe-demo/default (resourceName is <instance>-<component> since core
	// 2.0.0-alpha.7); the same the parity oracle renders (D3).
	assert.Equal(t, "probe-demo-web", resourceName)
	assert.Equal(t, "probe-demo-web.default.svc.cluster.local", fqdn)
}
