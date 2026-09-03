package kernel_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/require"

	loaderfile "github.com/open-platform-model/library/opm/helper/loader/file"
	"github.com/open-platform-model/library/opm/helper/synth"
	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/platform"
)

// This file holds the shared, fully hermetic builders for the kernel
// integration harness. Catalogs and modules are served from an in-memory OCI
// registry (opm/internal/registrytest); the core schema resolves from the
// warm workspace cache. No localhost:5000, so these run in CI under any
// condition.
//
// Render takes source-carrying inputs (0019 D9), so a hermetic test authors
// its inputs as modules on disk: a #CatalogEntry-form platform module
// importing a served catalog (writeCatalogPlatform), and an instance module
// importing a served module (writeImportedInstance), or a synthesized
// instance (synthesizeInstance). Nothing is built from a bare value.

// standardCatalog is the common two-transformer catalog: "deployment" requires
// the container resource and emits a single struct (→ 1 Compiled), "configmap"
// requires the config-maps resource and emits a two-element list (→ 2 Compiled).
func standardCatalog(path, version string) registrytest.CatalogFixture {
	return registrytest.CatalogFixture{
		Path:    path,
		Version: version,
		Body: registrytest.BuildCatalog(path, version,
			registrytest.TxFixture{
				Name:      "deployment",
				Resources: []string{"container"},
				Output:    `{ kind: "Deployment" }`,
			},
			registrytest.TxFixture{
				Name:      "configmap",
				Resources: []string{"config-maps"},
				Output:    `[ {kind: "ConfigMap", n: 1}, {kind: "ConfigMap", n: 2} ]`,
			},
		),
	}
}

// resFQN reproduces the resource contract FQN registrytest.BuildCatalog
// keys v2 members under — apiVersion-keyed (enhancement 0010 D4), so the
// catalog's build version does not appear in it.
func resFQN(path, name string) string {
	return fmt.Sprintf("%s/resources/%s@%s", path, name, registrytest.ContractAPIVersion)
}

// majorOf returns the major-qualified suffix a served fixture at version is
// published under: "0.1.0" → "v0".
func majorOf(version string) string {
	major, _, _ := strings.Cut(version, ".")
	return "v" + major
}

// newKernelWithCatalogs stands up an in-memory registry serving the given
// catalogs and returns a kernel wired to it, plus the registry mapping for
// the loaders.
func newKernelWithCatalogs(t *testing.T, catalogs ...registrytest.CatalogFixture) (*kernel.Kernel, string) {
	t.Helper()
	registry := registrytest.NewCatalogRegistry(t, catalogs...)
	return kernel.New(kernel.WithRegistry(registry)), registry
}

// writeCatalogPlatform writes, as a nested module under dir, a
// #CatalogEntry-form platform (core 0019 D5) importing the served catalog
// catPath at version, and returns the platform module directory.
func writeCatalogPlatform(t *testing.T, dir, catPath, version string) string {
	t.Helper()
	platDir := filepath.Join(dir, "platform")
	dep := catPath + "@" + majorOf(version)
	writeFile(t, filepath.Join(platDir, "cue.mod", "module.cue"), fmt.Sprintf(`module: "testing.opmodel.dev/library-kernel-test/platform@v0"
language: version: "v0.17.0"
deps: {
	"opmodel.dev/core@v2": v: %q
	%q: v: %q
}
`, registrytest.DefaultCoreVersion, dep, "v"+version))
	writeFile(t, filepath.Join(platDir, "platform.cue"), fmt.Sprintf(`package platform

import (
	core "opmodel.dev/core@v2"
	cat %q
)

core.#Platform
metadata: name: "hermetic"
type: "kubernetes"
#registry: %q: {
	enable:   true
	#catalog: cat
}
`, dep, dep))
	return platDir
}

// acquireCatalogPlatform writes a platform module for the served catalog and
// acquires it source-carrying through the kernel.
func acquireCatalogPlatform(t *testing.T, k *kernel.Kernel, mapping, catPath, version string) *platform.Platform {
	t.Helper()
	platDir := writeCatalogPlatform(t, t.TempDir(), catPath, version)
	plat, err := k.AcquirePlatformFromDir(context.Background(), platDir, loaderfile.LoadOptions{Registry: mapping})
	require.NoErrorf(t, err, "acquiring the #CatalogEntry-form platform for %s at %s", catPath, version)
	return plat
}

// writeImportedInstance writes, under root, a module (modulePath) whose
// cue.mod pins core, the served module modPath at version and extraDeps
// (major-qualified path → bare version), and an `instance` package importing
// the module with the given identity and values body. Returns the instance
// package directory; the module root is root itself.
func writeImportedInstance(t *testing.T, root, modulePath, modPath, version, name, namespace, values string, extraDeps map[string]string) string {
	t.Helper()
	var deps strings.Builder
	fmt.Fprintf(&deps, "\t\"opmodel.dev/core@v2\": v: %q\n", registrytest.DefaultCoreVersion)
	fmt.Fprintf(&deps, "\t%q: v: %q\n", modPath+"@"+majorOf(version), "v"+version)
	for p, v := range extraDeps {
		fmt.Fprintf(&deps, "\t%q: v: %q\n", p, "v"+strings.TrimPrefix(v, "v"))
	}
	writeFile(t, filepath.Join(root, "cue.mod", "module.cue"), fmt.Sprintf(`module: %q
language: version: "v0.17.0"
deps: {
%s}
`, modulePath, deps.String()))
	writeFile(t, filepath.Join(root, "instance", "instance.cue"), fmt.Sprintf(`package instance

import (
	core "opmodel.dev/core@v2"
	opmModule %q
)

core.#ModuleInstance

metadata: {
	name:      %q
	namespace: %q
}

#module: opmModule
values: %s
`, modPath+"@"+majorOf(version), name, namespace, values))
	return filepath.Join(root, "instance")
}

// synthesizeInstance acquires the served module with source and synthesizes
// an overlay-mode instance from it with empty values (the module's #config
// must be empty or fully defaulted).
func synthesizeInstance(t *testing.T, k *kernel.Kernel, modPath, version, name string) *module.Instance {
	t.Helper()
	ctx := context.Background()
	mod, err := k.AcquireModuleFromRegistry(ctx, modPath+"@"+majorOf(version), "v"+version)
	require.NoErrorf(t, err, "acquiring served module %s", modPath)
	require.True(t, mod.HasSource(), "acquired module must carry staged source")
	inst, err := k.SynthesizeInstance(ctx, synth.InstanceInput{
		Module:      mod,
		Name:        name,
		Namespace:   "default",
		Values:      k.CueContext().CompileString("{}"),
		SchemaCache: k.SchemaCache(),
	})
	require.NoErrorf(t, err, "synthesizing an instance from %s", modPath)
	require.NotNil(t, inst.Source)
	return inst
}

// twoComponentModuleFile authors a module with a `web` component declaring
// the catalog's container resource and a `config` component declaring its
// config-maps resource, keyed by the FQNs standardCatalog's transformers
// require.
func twoComponentModuleFile(modPath, catPath, version string) string {
	containerFQN := resFQN(catPath, "container")
	configFQN := resFQN(catPath, "config-maps")
	return fmt.Sprintf(`package two_app

import core "opmodel.dev/core@v2"

core.#Module
metadata: {
	name:       "two_app"
	modulePath: %q
	version:    %q
}
#config: {}
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
	config: {
		metadata: name: "config"
		#resources: %q: {
			kind: "Resource"
			metadata: {name: "config-maps", modulePath: %q, apiVersion: %q, catalogVersion: %q, fqn: %q}
			spec: configMaps: {app: {data: {}}}
		}
	}
}
`, modPath+"@"+majorOf(version), version,
		containerFQN, catPath+"/resources", registrytest.ContractAPIVersion, version, containerFQN,
		configFQN, catPath+"/resources", registrytest.ContractAPIVersion, version, configFQN)
}

// pairsByComponent groups matched pairs by component name for ergonomic
// containment assertions.
func pairsByComponent(pairs []kernel.RenderPair) map[string][]string {
	out := map[string][]string{}
	for _, p := range pairs {
		out[p.Component] = append(out[p.Component], p.Transformer)
	}
	return out
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

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
