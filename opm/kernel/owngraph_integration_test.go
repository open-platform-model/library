package kernel_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/helper/synth"
	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/module"
)

// TestFlow_OwnGraphResolution is the 0010 D10 graduation gate (enhancement
// 0010, 04-graduation): a module's primitives resolve through the MODULE'S OWN
// dependency graph, never one shared with the platform. The platform's
// subscribed catalog build carries definitions that DIVERGE from the ones the
// module's own catalog dependency carries for the same contract keys; the test
// passes only while the two resolutions stay separate:
//
//   - The always-unify rung reports the divergence as a UnifyError — proving
//     the component's body (module-side resolution) and the transformer's
//     required copy (platform-side resolution) are distinct values that were
//     actually compared. Under a shared resolution the two would be one value,
//     the rung would trivially pass, and the assertion fails.
//   - Rendered output follows the module-side definition: the storage
//     retention default is stated ONLY by the module-side build (the
//     platform's copy constrains nothing there), so the compiled value's
//     "module-default" proves the render consumed the module's own
//     resolution of the schema.
//
// ownGraphCatalogs publishes the two builds of one catalog path:
//
//   - 0.1.0 — the module-side build. No transformers; it exports the plain
//     spec-schema definitions (#ContainerSchemaV1 / #StorageSchemaV1) whose
//     values modules pull through their own dependency graph.
//   - 0.2.0 — the platform-side build. Its transformers' embedded required
//     copies DIVERGE from the 0.1.0 definitions at the container contract key:
//     spec.container collapses struct→string, a same-apiVersion shape break
//     the rung must refuse. The storage copy stays agreeing (`_`), so the
//     storage module renders and proves output follows module-side data.
func ownGraphCatalogs(t *testing.T) (catPath string, moduleSide, platformSide registrytest.CatalogFixture) {
	t.Helper()
	catPath = registrytest.UniquePath(t, "cat")
	containerFQN := resFQN(catPath, "container")
	storageFQN := resFQN(catPath, "storage")

	// resourceCopy authors a platform-side embedded required copy.
	//
	// The copies keep their nested spec body
	// at `_` or a scalar (the shape registrytest.BuildCatalog emits): both
	// operands are #Resource-typed by their own build's core import, and
	// unifying two independently-loaded closed instances whose spec bodies
	// are BOTH nested struct literals trips a v0.17 evaluator closedness
	// quirk ("spec.<name>: field not allowed" pointing at both operands'
	// declarations) that would disqualify every pair regardless of
	// divergence and make this gate vacuous. Divergence therefore rides on
	// the spec.<name> position itself (struct vs scalar), which conflicts
	// without nested-literal admission.
	resourceCopy := func(name, catalogVersion, specBody string) string {
		fqn := resFQN(catPath, name)
		return fmt.Sprintf(`{
			kind: "Resource"
			metadata: {name: %q, modulePath: %q, apiVersion: %q, catalogVersion: %q, fqn: %q}
			spec: %s: %s
		}`, name, catPath+"/resources", registrytest.ContractAPIVersion, catalogVersion, fqn, name, specBody)
	}

	moduleSide = registrytest.CatalogFixture{
		Path:    catPath,
		Version: "0.1.0",
		Body: fmt.Sprintf(`metadata: {
	modulePath:  %q
	version:     "0.1.0"
	description: "module-side build"
}
#transformers: {}
#ContainerSchemaV1: { image: string, port: int | *8080 }
#StorageSchemaV1: { retention: string | *"module-default" }
`, catPath+"@v0"),
	}

	platformSide = registrytest.CatalogFixture{
		Path:    catPath,
		Version: "0.2.0",
		Body: fmt.Sprintf(`metadata: {
	modulePath:  %q
	version:     "0.2.0"
	description: "platform-side build"
}
#transformers: {
	%q: {
		kind: "ComponentTransformer"
		metadata: {name: "deployment", description: "renders containers", fqn: %q}
		requiredResources: %q: %s
		#transform: {
			#component: _
			#context:   _
			output: { kind: "Deployment" }
		}
	}
	%q: {
		kind: "ComponentTransformer"
		metadata: {name: "storage", description: "renders storage", fqn: %q}
		requiredResources: %q: %s
		#transform: {
			#component: _
			#context:   _
			output: {
				kind:      "Storage"
				retention: #component.spec.storage.retention
			}
		}
	}
}
`, catPath+"@v0",
			catPath+"/transformers/deployment@0.2.0", catPath+"/transformers/deployment@0.2.0",
			containerFQN, resourceCopy("container", "0.2.0", "string"),
			catPath+"/transformers/storage@0.2.0", catPath+"/transformers/storage@0.2.0",
			storageFQN, resourceCopy("storage", "0.2.0", "_")),
	}

	return catPath, moduleSide, platformSide
}

// ownGraphModule publishes a module (with source) at metaPath/<leaf> whose
// cue.mod declares the 0.1.0 catalog dep and whose single component attaches
// the given module-side definition, then acquires and synthesizes it.
func ownGraphModule(t *testing.T, k *kernel.Kernel, modPath, leaf string) *module.Instance {
	t.Helper()
	ctx := context.Background()

	mod, err := k.AcquireModuleFromRegistry(ctx, modPath+"@v0", "v0.1.0")
	require.NoErrorf(t, err, "acquiring module %s", modPath)
	require.True(t, mod.HasSource())

	inst, err := k.SynthesizeInstance(ctx, synth.InstanceInput{
		Module:      mod,
		Name:        strings.ReplaceAll(leaf, "_", "-") + "-inst",
		Namespace:   "default",
		Values:      k.CueContext().CompileString("{}"),
		SchemaCache: k.SchemaCache(),
	})
	require.NoErrorf(t, err, "synthesizing instance for %s", modPath)
	return inst
}

func ownGraphModuleFile(catPath, leaf, modPath, componentBody string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "package %s\n\n", leaf)
	b.WriteString("import (\n\tcore \"opmodel.dev/core@v2\"\n")
	fmt.Fprintf(&b, "\tcat %q\n)\n\n", catPath+"@v0")
	b.WriteString("core.#Module\n")
	fmt.Fprintf(&b, "metadata: {\n\tname:       %q\n\tmodulePath: %q\n\tversion:    \"0.1.0\"\n}\n", leaf, modPath+"@v0")
	b.WriteString("#config: {}\n")
	b.WriteString(componentBody)
	return b.String()
}

func TestFlow_OwnGraphResolution(t *testing.T) {
	catPath, moduleSideCat, platformSideCat := ownGraphCatalogs(t)
	containerFQN := resFQN(catPath, "container")

	metaPath := registrytest.UniquePath(t, "modules")
	webPath := metaPath + "/own_web"
	storePath := metaPath + "/own_store"

	// Component resource bodies are authored inline (typed by the MODULE's
	// #ResourceMap) with the spec schema referenced from the module's own
	// catalog dependency — the value under test resolves through the
	// module's dependency graph, never the platform's subscription.
	moduleResource := func(name, schemaRef, extra string) string {
		fqn := resFQN(catPath, name)
		return fmt.Sprintf(`%q: {
			kind: "Resource"
			metadata: {name: %q, modulePath: %q, apiVersion: %q, catalogVersion: "0.1.0", fqn: %q}
			spec: %s: %s%s
		}`, fqn, name, catPath+"/resources", registrytest.ContractAPIVersion, fqn, name, schemaRef, extra)
	}

	webFile := ownGraphModuleFile(catPath, "own_web", webPath, fmt.Sprintf(`#components: {
	web: {
		metadata: name: "web"
		#resources: %s
	}
}
`, moduleResource("container", "cat.#ContainerSchemaV1", ` & {image: "nginx"}`)))

	storeFile := ownGraphModuleFile(catPath, "own_store", storePath, fmt.Sprintf(`#components: {
	store: {
		metadata: name: "store"
		#resources: %s
	}
}
`, moduleResource("storage", "cat.#StorageSchemaV1", "")))

	registry := registrytest.NewModuleRegistry(t,
		[]registrytest.ModuleFixture{
			{Path: webPath, Version: "0.1.0", File: webFile, Deps: map[string]string{catPath + "@v0": "0.1.0"}},
			{Path: storePath, Version: "0.1.0", File: storeFile, Deps: map[string]string{catPath + "@v0": "0.1.0"}},
		},
		[]registrytest.CatalogFixture{moduleSideCat, platformSideCat},
	)

	k := kernel.New(kernel.WithRegistry(registry))
	ctx := context.Background()

	// The platform subscribes the 0.2.0 build — the divergent one.
	mp, err := materializePlatform(t, k, "0.2.0", catPath)
	require.NoError(t, err)

	t.Run("divergence is compared and refused", func(t *testing.T) {
		inst := ownGraphModule(t, k, webPath, "own_web")

		plan, err := k.Match(ctx, kernel.MatchInput{ModuleInstance: inst, Platform: mp})
		require.NoError(t, err)

		// The UnifyError is the own-graph proof: the component's container
		// body resolved through the MODULE's dependency (port: int | *8080)
		// and the transformer's required copy through the PLATFORM's
		// subscription (port: string). Two distinct values, actually
		// compared, genuinely divergent. Under a shared resolution they
		// would be one value and no UnifyError could exist.
		require.NotEmpty(t, plan.Unify, "own-graph divergence must surface at the unify rung")
		assert.Equal(t, containerFQN, plan.Unify[0].FQN)
		assert.Equal(t, "web", plan.Unify[0].Component)

		// The disqualified candidate leaves the demand unresolved (D28).
		require.NotEmpty(t, plan.Unresolved)
		assert.Equal(t, containerFQN, plan.Unresolved[0].FQN)
	})

	t.Run("render follows the module-side definition", func(t *testing.T) {
		inst := ownGraphModule(t, k, storePath, "own_store")

		out, err := k.Compile(ctx, kernel.CompileInput{ModuleInstance: inst, Platform: mp, RuntimeName: "rt"})
		require.NoError(t, err, "agreeing shapes with differing defaults must render")
		require.Len(t, out.Compiled, 1)

		retention, err := out.Compiled[0].Value.LookupPath(cue.ParsePath("retention")).String()
		require.NoError(t, err)
		assert.Equal(t, "module-default", retention,
			"the rendered default is the MODULE-side definition's — platform-side leakage means shared resolution")
	})
}
