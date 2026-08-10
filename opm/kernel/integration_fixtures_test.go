package kernel_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/materialize"
	"github.com/open-platform-model/library/opm/module"
)

// This file holds the shared, fully hermetic builders for the kernel
// integration harness. Catalogs are served from an in-memory OCI registry
// (opm/internal/registrytest); the core schema resolves from the warm
// workspace cache. No localhost:5000, so these run in CI under any condition.
//
// Instances are hand-authored CUE values rather than loaded from disk: a
// component's #resources / #traits are keyed by the catalog's stamped FQNs
// (`<path>/resources/<name>@<version>` etc.), which is all the matcher and
// compile phases read. This keeps the harness independent of on-disk fixtures
// and of any module that imports the real catalog.

// compSpec describes one component to author into an instance: its name, the
// short names of the catalog resources/traits it declares, and its labels.
type compSpec struct {
	name      string
	resources []string
	traits    []string
	labels    map[string]string
}

// resFQN / traitFQN reproduce the FQNs registrytest.BuildCatalog stamps so
// instance components key against the same strings the matcher index uses.
func resFQN(path, name, version string) string {
	return fmt.Sprintf("%s/resources/%s@%s", path, name, version)
}

func traitFQN(path, name, version string) string {
	return fmt.Sprintf("%s/traits/%s@%s", path, name, version)
}

// newKernelWithCatalogs stands up an in-memory registry serving the given
// catalogs and returns a kernel wired to it. The kernel's context resolves
// both the core schema (warm cache) and the catalogs (in-memory host).
func newKernelWithCatalogs(t *testing.T, catalogs ...registrytest.CatalogFixture) *kernel.Kernel {
	t.Helper()
	registry := registrytest.NewCatalogRegistry(t, catalogs...)
	return kernel.New(kernel.WithRegistry(registry))
}

// subscribe builds a #registry body subscribing (enabled) to each path.
func subscribe(paths ...string) string {
	var b strings.Builder
	b.WriteString("{")
	for i, p := range paths {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%q: {enable: true}", p)
	}
	b.WriteString("}")
	return b.String()
}

// buildInstance assembles a hermetic *module.Instance: an embedded #module (with
// an optional #config schema), instance metadata, optional values, and the
// authored components. configSchema and valuesSrc are raw CUE struct bodies
// ("" to omit). It is constructed through k.NewInstanceFromValue so the harness
// exercises that constructor path.
func buildInstance(
	t *testing.T,
	k *kernel.Kernel,
	catPath, version string,
	configSchema, valuesSrc string,
	comps ...compSpec,
) *module.Instance {
	t.Helper()

	var cb strings.Builder
	cb.WriteString("{\n")
	for _, c := range comps {
		fmt.Fprintf(&cb, "\t%q: {\n", c.name)
		fmt.Fprintf(&cb, "\t\tmetadata: { name: %q, labels: %s }\n", c.name, labelsLiteral(c.labels))
		// Bodies are written open ("{...}"): #resources / #traits are CUE
		// definitions, which recursively close nested structs. The always-unify
		// matcher rung unifies each body with the transformer's required
		// (closed) #Resource / #Trait, so a closed empty body would fail
		// closedness. "{...}" stays open and absorbs the required shape.
		cb.WriteString("\t\t#resources: {\n")
		for _, r := range c.resources {
			fmt.Fprintf(&cb, "\t\t\t%q: {...}\n", resFQN(catPath, r, version))
		}
		cb.WriteString("\t\t}\n")
		if len(c.traits) > 0 {
			cb.WriteString("\t\t#traits: {\n")
			for _, tr := range c.traits {
				fmt.Fprintf(&cb, "\t\t\t%q: {...}\n", traitFQN(catPath, tr, version))
			}
			cb.WriteString("\t\t}\n")
		}
		cb.WriteString("\t}\n")
	}
	cb.WriteString("}")

	configField := ""
	if strings.TrimSpace(configSchema) != "" {
		configField = "\n\t#config: " + configSchema
	}
	valuesField := ""
	if strings.TrimSpace(valuesSrc) != "" {
		valuesField = "\nvalues: " + valuesSrc
	}

	src := fmt.Sprintf(`
kind: "ModuleInstance"
metadata: { name: "demo", namespace: "ns", uuid: "11111111-2222-5333-8444-555555555555" }
#module: {
	kind: "Module"
	metadata: { name: "demo", modulePath: "example.com/demo", version: "0.1.0" }%s
}%s
components: %s
`, configField, valuesField, cb.String())

	v := k.CueContext().CompileString(src, cue.Filename("instance.cue"))
	require.NoError(t, v.Err(), "compiling hermetic instance")
	inst, err := k.NewInstanceFromValue(v)
	require.NoError(t, err, "constructing instance from value")
	return inst
}

func labelsLiteral(labels map[string]string) string {
	if len(labels) == 0 {
		return "{}"
	}
	var b strings.Builder
	b.WriteString("{")
	first := true
	for k, v := range labels {
		if !first {
			b.WriteString(", ")
		}
		first = false
		fmt.Fprintf(&b, "%q: %q", k, v)
	}
	b.WriteString("}")
	return b.String()
}

// buildModule assembles a hermetic *module.Module with the given #config schema
// body (raw CUE, e.g. `{ replicas: int | *1 }`). Constructed via
// k.NewModuleFromValue so the harness exercises that path.
func buildModule(t *testing.T, k *kernel.Kernel, configSchema string) *module.Module {
	t.Helper()
	src := fmt.Sprintf(`
kind: "Module"
metadata: { name: "demo", modulePath: "example.com/demo", version: "0.1.0" }
#config: %s
`, configSchema)
	v := k.CueContext().CompileString(src, cue.Filename("module.cue"))
	require.NoError(t, v.Err(), "compiling hermetic module")
	m, err := k.NewModuleFromValue(v)
	require.NoError(t, err, "constructing module from value")
	return m
}

// cueVal compiles src in the kernel's context with a stable filename so
// per-source attribution (used by ValidateConfigDetailed) is meaningful.
func cueVal(t *testing.T, k *kernel.Kernel, src, filename string) cue.Value {
	t.Helper()
	v := k.CueContext().CompileString(src, cue.Filename(filename))
	require.NoError(t, v.Err(), "compiling %s", filename)
	return v
}

// materialize subscribes the kernel to the given catalog paths and materializes.
func materializePlatform(t *testing.T, k *kernel.Kernel, paths ...string) (*materialize.MaterializedPlatform, error) {
	t.Helper()
	plat := registrytest.BuildPlatform(t, k.CueContext(), subscribe(paths...))
	return k.Materialize(context.Background(), plat)
}
