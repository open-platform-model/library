package kernel_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/helper/synth"
	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/kernel"
)

// TestFlow_ImportedModule_CatalogSubpackageImport_SynthToCompile is the
// HERMETIC construction + render smoke test for the module-root synth path: a
// published module whose SOURCE imports a catalog package is acquired WITH
// SOURCE, synthesized inside its own staged root, and compiled end-to-end to its
// expected Deployment. It exercises the full new path
// (AcquireModuleFromRegistry → synth-in-module-root → Match → Compile) in CI
// without a real registry.
//
// NOTE: this hermetic test does NOT by itself prove the library#31 fix is
// non-vacuous. The in-memory registrytest resolver walks a dependency's
// cue.mod/module.cue transitively, so the OLD fabricated-{core, module} synth
// would ALSO have resolved this fixture's catalog import. The real modconfig
// resolver does not transitively resolve, which is what actually broke #31. The
// faithful, non-vacuous #31 guard (proven to fail under the old path and pass
// under the new one) is TestFlow_Redis_CatalogSubpackage_Regression, which runs
// against a real registry. Keep both: this one guards the construction/compile
// wiring in CI; redis guards the actual resolution semantics.
//
// The Compile step also exercises design D4's within-major safety: the instance
// is built against the MODULE's own core (from its cue.mod/module.cue) and then
// processed/compiled under the kernel's SchemaCache core; both are core@v1, so
// they unify. (Cross-patch core skew is not constructible in this hermetic
// harness, which serves a single core version.)
func TestFlow_ImportedModule_CatalogSubpackageImport_SynthToCompile(t *testing.T) {
	const version = "0.1.0"
	catPath := registrytest.UniquePath(t, "cat")
	metaPath := registrytest.UniquePath(t, "modules")
	const modName = "web-app"
	const snake = "web_app"
	modPath := metaPath + "/" + snake
	containerFQN := fmt.Sprintf("%s/resources/container@%s", catPath, version)

	// Catalog: a deployment transformer requiring `container`, emitting a
	// Deployment (same fixture the non-importing flow test uses).
	cat := standardCatalog(catPath, version)
	cat.CoreVersion = "v1.0.0-alpha.1"

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
			metadata: {name: "container", modulePath: %q, version: %q}
			spec: container: {image: "nginx"}
		}
	}
}
`, containerFQN, catPath+"/resources", version)

	var modFile strings.Builder
	fmt.Fprintf(&modFile, "package %s\n\n", snake)
	modFile.WriteString("import (\n")
	modFile.WriteString("\tcore \"opmodel.dev/core@v1\"\n")
	fmt.Fprintf(&modFile, "\tcat %q\n", catPath+"@v0")
	modFile.WriteString(")\n\n")
	modFile.WriteString("core.#Module\n")
	fmt.Fprintf(&modFile, "metadata: {\n\tname:       %q\n\tmodulePath: %q\n\tversion:    %q\n}\n", modName, metaPath, version)
	modFile.WriteString(modBody)

	registryMapping := registrytest.NewModuleRegistry(t,
		[]registrytest.ModuleFixture{{
			Path:        modPath,
			Version:     version,
			File:        modFile.String(),
			CoreVersion: "v1.0.0-alpha.1",
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

	mp, err := materializePlatform(t, k, catPath)
	require.NoError(t, err, "materializing platform subscribed to the catalog")

	inst, err := k.SynthesizeInstance(ctx, synth.InstanceInput{
		Module:      mod,
		Name:        "web-inst",
		Namespace:   "default",
		Values:      k.CueContext().CompileString("{}"),
		SchemaCache: k.SchemaCache(),
	})
	require.NoErrorf(t, err, "synthesizing an instance from a catalog-importing module (library#31 regression)")
	if err != nil {
		// Make the historical failure signature legible if this ever regresses.
		assert.NotContains(t, err.Error(), "cannot find module providing package",
			"synth must resolve the module's transitive catalog import via its own cue.mod/module.cue")
	}

	out, err := k.Compile(ctx, kernel.CompileInput{ModuleInstance: inst, Platform: mp, RuntimeName: "rt"})
	require.NoError(t, err)
	require.NotEmpty(t, out.Compiled, "catalog-importing synth instance must compile to at least one resource")
	assert.Contains(t, compiledKinds(t, out.Compiled), "Deployment",
		"the container resource must render a Deployment")
}
