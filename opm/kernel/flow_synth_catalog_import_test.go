package kernel_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/kernel"
)

// TestFlow_ImportedModule_CatalogSubpackageImport_SynthToRender is the
// HERMETIC construction + render smoke test for the module-root synth path: a
// published module whose SOURCE imports a catalog package is acquired WITH
// SOURCE, synthesized inside its own staged root, and rendered end-to-end to
// its expected Deployment. It exercises the full path
// (AcquireModuleFromRegistry → synth-in-module-root → Render) in CI without a
// real registry.
//
// This is the library#31 coverage. The GHCR-gated guard that once sat beside
// it (TestInstance_CatalogSubpackageImport_Regression, importing the real
// catalogs/opm blueprints subpackage) left with opm/helper/synth in #116 and
// was not re-created. The module here is fetched through modconfig and built
// by cue/load against the in-process OCI host, CUE's production resolver, so
// a synthesized package whose main cue.mod lacked the catalog dependency (the
// #31 failure) would fail here too; that path is gone, and the
// instance-synthesis spec forbids fabricating a cue.mod, so this test is the
// guard against its return.
func TestFlow_ImportedModule_CatalogSubpackageImport_SynthToRender(t *testing.T) {
	const version = "0.1.0"
	catPath := registrytest.UniquePath(t, "cat")
	metaPath := registrytest.UniquePath(t, "modules")
	const snake = "web_app" // v2 module names are snake_case, and the path leaf
	modPath := metaPath + "/" + snake
	containerFQN := resFQN(catPath, "container")

	// Catalog: a deployment transformer requiring `container`, emitting a
	// Deployment (same fixture the non-importing flow test uses).
	cat := standardCatalog(catPath, version)

	// Module: its SOURCE imports the catalog package (the #31 trigger) and
	// references it load-bearingly through debugValues (an open field, so the
	// import is not elided), AND declares a renderable `web` component.
	modBody := fmt.Sprintf(`#config: {}
debugValues: catalogModulePath: cat.metadata.modulePath
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
	modFile.WriteString("import (\n")
	modFile.WriteString("\tcore \"opmodel.dev/core@v2\"\n")
	fmt.Fprintf(&modFile, "\tcat %q\n", catPath+"@v0")
	modFile.WriteString(")\n\n")
	modFile.WriteString("core.#Module\n")
	fmt.Fprintf(&modFile, "metadata: {\n\tname:       %q\n\tmodulePath: %q\n\tversion:    %q\n}\n", snake, modPath+"@v0", version)
	modFile.WriteString(modBody)

	registryMapping := registrytest.NewModuleRegistry(t,
		[]registrytest.ModuleFixture{{
			Path:    modPath,
			Version: version,
			File:    modFile.String(),
			// The module's own cue.mod/module.cue declares the catalog dep — this
			// is the tidied closure synth now reuses.
			Deps: map[string]string{catPath + "@v0": version},
		}},
		[]registrytest.CatalogFixture{cat},
	)

	k := kernel.New(kernel.WithRegistry(registryMapping))
	ctx := context.Background()

	// Acquire WITH source so synth can build inside the module's own root.
	mod, err := k.AcquireModuleFromRegistry(ctx, modPath+"@v0", "v"+version)
	require.NoErrorf(t, err, "acquiring catalog-importing module %s", modPath)
	require.True(t, mod.HasSource(), "acquired module must carry staged source")

	plat := acquireCatalogPlatform(t, k, registryMapping, catPath, version)

	inst, err := k.SynthesizeInstance(ctx, kernel.InstanceInput{
		Module:    mod,
		Name:      "web-inst",
		Namespace: "default",
		Values:    []kernel.Source{mustSource(t, k, "values.cue", "{}")},
	})
	require.NoErrorf(t, err, "synthesizing an instance from a catalog-importing module (library#31 regression)")
	if err != nil {
		// Make the historical failure signature legible if this ever regresses.
		assert.NotContains(t, err.Error(), "cannot find module providing package",
			"synth must resolve the module's transitive catalog import via its own cue.mod/module.cue")
	}

	res, err := k.Render(ctx, kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
	require.NoError(t, err)
	require.NotEmpty(t, res.Compiled, "catalog-importing synth instance must render to at least one resource")
	assert.Contains(t, compiledKinds(t, res.Compiled), "Deployment",
		"the container resource must render a Deployment")
	// The instance module's own catalog dependency and the platform's are
	// the same build: a row, not skew.
	for _, r := range res.Diagnostics.ResolvedVersions {
		assert.False(t, r.Newer, "no path is newer than the platform's build: %s", r.Path)
	}
}
