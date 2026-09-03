package kernel_test

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/core"
	loaderfile "github.com/open-platform-model/library/opm/helper/loader/file"
	"github.com/open-platform-model/library/opm/helper/synth"
	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/kernel"
)

// TestFlow_ImportedModule_SynthToRender is the end-to-end imported-module
// render coverage (spec instance-synthesis, "Imported-module render coverage
// exists"): a real published module — referenced by IMPORT, named at its
// snake_case path leaf per core v2's D8 identity rule — is synthesized into
// a #ModuleInstance and rendered through Kernel.Render against a D5-shaped
// platform to concrete resources. An authored instance.cue importing the
// SAME module is rendered too, and both MUST yield the same rendered set
// (single-build parity through Kernel.Render, not merely at the
// instance-value level).
//
// Hermetic: the module + catalog are served from an in-memory registry;
// opmodel.dev/core resolves from the warm workspace cache at
// registrytest.DefaultCoreVersion, matching the other registrytest-backed
// integration tests in this package.
func TestFlow_ImportedModule_SynthToRender(t *testing.T) {
	const version = "0.1.0"
	catPath := registrytest.UniquePath(t, "cat")
	metaPath := registrytest.UniquePath(t, "modules")
	const snake = "web_app" // v2 module names are snake_case, and the path leaf
	modPath := metaPath + "/" + snake
	containerFQN := resFQN(catPath, "container")

	// Catalog: one transformer requiring the `container` resource, emitting a
	// single Deployment (core v2, the registry-writer default).
	cat := standardCatalog(catPath, version)

	// Module: a single component `web` declaring an inline #Resource keyed by the
	// catalog's container FQN (the matcher pairs on that key). No catalog import
	// is needed — the FQN is a plain key, exactly as an instance authors it.
	modBody := fmt.Sprintf(`#config: {}
debugValues: {}
#components: {
	web: {
		metadata: name: "web"
		#resources: %q: {
			kind: "Resource"
			metadata: {name: "container", modulePath: %q, apiVersion: %q, catalogVersion: %q, fqn: %q}
			spec: container: {image: "nginx"}
		}
	}
}
`, containerFQN, catPath+"/resources", registrytest.ContractAPIVersion, version, containerFQN)

	var modFile strings.Builder
	fmt.Fprintf(&modFile, "package %s\n\n", snake)
	modFile.WriteString("import core \"opmodel.dev/core@v2\"\n\n")
	modFile.WriteString("core.#Module\n")
	fmt.Fprintf(&modFile, "metadata: {\n\tname:       %q\n\tmodulePath: %q\n\tversion:    %q\n}\n", snake, modPath+"@v0", version)
	modFile.WriteString(modBody)

	registryMapping := registrytest.NewModuleRegistry(t,
		[]registrytest.ModuleFixture{{Path: modPath, Version: version, File: modFile.String()}},
		[]registrytest.CatalogFixture{cat},
	)

	k := kernel.New(kernel.WithRegistry(registryMapping))
	ctx := context.Background()

	mod, err := k.AcquireModuleFromRegistry(ctx, modPath+"@v0", "v"+version)
	require.NoErrorf(t, err, "acquiring published module %s", modPath)
	require.Equal(t, "web_app", mod.Metadata.Name)

	plat := acquireCatalogPlatform(t, k, registryMapping, catPath, version)

	// ── synth path ───────────────────────────────────────────────────────
	inst, err := k.SynthesizeInstance(ctx, synth.InstanceInput{
		Module:      mod,
		Name:        "web-inst",
		Namespace:   "default",
		Values:      k.CueContext().CompileString("{}"), // #config is empty; supply concrete (empty) values
		SchemaCache: k.SchemaCache(),
	})
	require.NoError(t, err, "synthesizing instance from the imported module")

	res, err := k.Render(ctx, kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
	require.NoError(t, err)
	pairs := pairsByComponent(res.Diagnostics.Pairs)
	require.Contains(t, pairs, "web", "web component must pair with a transformer")
	assertContainsFQNSub(t, pairs["web"], "transformers/deployment@", "web must match the deployment transformer")
	require.NotEmpty(t, res.Compiled, "synth instance must render to at least one resource")
	synthKinds := compiledKinds(t, res.Compiled)
	assert.Contains(t, synthKinds, "Deployment", "the container resource must render a Deployment")

	// ── authored path (single-build parity through Kernel.Render) ─────────
	instDir := writeImportedInstance(t, t.TempDir(), "authored.opmodel.dev/instance@v0", modPath, version, "web-inst", "default", "{}", nil)
	authored, err := k.AcquireInstanceFromDir(ctx, instDir, loaderfile.LoadOptions{Registry: registryMapping})
	require.NoError(t, err, "authored instance.cue importing the published module must acquire")

	authoredRes, err := k.Render(ctx, kernel.RenderInput{Instance: authored, Platform: plat, RuntimeName: "rt"})
	require.NoError(t, err)
	assert.Equal(t, synthKinds, compiledKinds(t, authoredRes.Compiled),
		"synth and authored imported-module instances must render to the same resources")
	assert.Equal(t, res.Diagnostics.Pairs, authoredRes.Diagnostics.Pairs)
	for i := range res.Compiled {
		a, err := res.Compiled[i].Value.MarshalJSON()
		require.NoError(t, err)
		b, err := authoredRes.Compiled[i].Value.MarshalJSON()
		require.NoError(t, err)
		assert.Equal(t, string(a), string(b), "object %d renders byte-identically on both paths", i)
	}
}

// compiledKinds returns the sorted `kind` strings of every rendered object,
// a stable fingerprint of a render's output for parity comparison.
func compiledKinds(t *testing.T, compiled []*core.Compiled) []string {
	t.Helper()
	kinds := make([]string, 0, len(compiled))
	for _, c := range compiled {
		k, err := c.Value.LookupPath(cue.ParsePath("kind")).String()
		require.NoError(t, err, "rendered object must carry a concrete kind")
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	return kinds
}
