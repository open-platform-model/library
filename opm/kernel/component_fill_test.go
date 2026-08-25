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

// TestCompile_ComponentFillPreservesDefinitions is the kernel-only half of
// the parity harness's names-probe (spec transform-input-fill: "#component
// is filled with every field class preserved"). A registrytest-served
// catalog carries one transformer that re-declares `#component: _` and
// emits the names core computes for the component; the instance is
// import-authored so `#instance` and `#names` resolve. Rendering through
// Kernel.Compile must carry those definitions into the transformer: before
// library-component-fill the strip left `#component.#names` absent
// (0019 D1/D3). Hermetic: no oracle, no GHCR.
func TestCompile_ComponentFillPreservesDefinitions(t *testing.T) {
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
	writeFile(t, filepath.Join(dir, "cue.mod", "module.cue"), fmt.Sprintf(`module: "testing.opmodel.dev/library-component-fill@v0"
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
	require.NoError(t, err, "a transformer reading #component.#names must render through Kernel.Compile")
	require.Len(t, out.Compiled, 1)
	require.Equal(t, txFQN, out.Compiled[0].Transformer)

	obj := out.Compiled[0].Value
	require.NoError(t, obj.Validate(cue.Concrete(true)), "rendered object must be concrete")

	resourceName, err := obj.LookupPath(cue.ParsePath("resourceName")).String()
	require.NoError(t, err)
	fqdn, err := obj.LookupPath(cue.ParsePath("fqdn")).String()
	require.NoError(t, err)

	// The values core computes for component `web` of instance
	// probe-demo/default; the same the parity oracle renders (D3).
	assert.Equal(t, "web", resourceName)
	assert.Equal(t, "web.default.svc.cluster.local", fqdn)
}
