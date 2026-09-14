package kernel_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/internal/schematest"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/schema"
)

// writeValuesModule lays out a CUE module holding one values file that
// imports the served package at importPath (major-qualified) pinned at
// version, and returns the values file's absolute path. The file wraps its
// payload in the conventional top-level `values:` field.
func writeValuesModule(t *testing.T, importPath, version string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cue.mod"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cue.mod", "module.cue"), []byte(
		"module: \"values.example/site@v0\"\nlanguage: version: \"v0.17.0\"\n"+
			"deps: \""+importPath+"\": v: \""+version+"\"\n"), 0o644))
	path := filepath.Join(dir, "values.cue")
	require.NoError(t, os.WriteFile(path, []byte(
		"package values\n\nimport defaults \""+importPath+"\"\n\nvalues: sentinel: defaults.sentinel\n"), 0o644))
	return path
}

// config-validation spec, "File-Backed Sources Resolve Imports Through the
// Kernel Mapping", and kernel-runtime spec, "Values source compilation uses
// the kernel mapping": a values file importing a package served only through
// the kernel's WithRegistry mapping compiles on every path that takes Source
// values, the process environment stays untouched, and a kernel without the
// option fails at the import.
//
// The registry constructor sets the process CUE_REGISTRY to the mapping it
// returns; the test then points the process at schema.PublicRegistry so the
// served prefix is routed only by the option under test, while core keeps
// resolving from the shared cache. The negative case runs first: CUE serves
// a coordinate from the module cache before it consults any registry, so
// once a positive load has cached the served package, a kernel without the
// mapping would find it there and prove nothing.
func TestKernel_LoadSourceFromFile_ImportsResolveThroughKernelRegistry(t *testing.T) {
	defaultsPath := registrytest.UniquePath(t, "defaults")
	defaults := registrytest.ModuleFixture{
		Path: defaultsPath, Version: "0.0.1",
		File: "package defaults\n\nsentinel: \"from-registry\"\n",
	}
	modPath, modFixture := synthModuleFixture(t, "demo", "0.1.0",
		"#components: {}\n#config: {sentinel: string}\ndebugValues: {}\n")
	mapping := registrytest.NewModuleRegistry(t, []registrytest.ModuleFixture{defaults, modFixture}, nil)
	t.Setenv("CUE_REGISTRY", schema.PublicRegistry)

	valuesPath := writeValuesModule(t, defaultsPath+"@v0", "v0.0.1")
	ctx := context.Background()
	newConfigSchema := func(t *testing.T) cue.Value {
		t.Helper()
		configSchema := cuecontext.New().CompileString(`{ sentinel: string }`)
		require.NoError(t, configSchema.Err())
		return configSchema
	}

	k := kernel.New(kernel.WithRegistry(mapping))
	src, err := k.LoadSourceFromFile(valuesPath)
	require.NoError(t, err, "the loader parses and evaluates nothing, so the import is not resolved here")
	// The module coordinate is cached by this acquisition; the values file's
	// import is a different coordinate and stays uncached until a positive
	// case below resolves it.
	mod := acquireSynthModule(t, k, modPath, "0.1.0")

	t.Run("a kernel without the option fails at the import", func(t *testing.T) {
		plain := kernel.New()
		_, err := plain.ValidateConfigDetailed(newConfigSchema(t), []kernel.Source{src})
		require.Error(t, err)
		assert.Contains(t, err.Error(), defaultsPath, "the failure names the unresolvable import: %v", err)

		// Artifacts cross kernels: the module acquired above synthesizes on
		// the plain kernel, and only the values import is left unrouted.
		inst, err := plain.SynthesizeInstance(ctx, kernel.InstanceInput{
			Module:    mod,
			Name:      "myrel",
			Namespace: "default",
			Values:    []kernel.Source{src},
		})
		require.Error(t, err)
		assert.Nil(t, inst)
		assert.Contains(t, err.Error(), defaultsPath, "the failure names the unresolvable import: %v", err)
	})

	t.Run("ValidateConfigDetailed", func(t *testing.T) {
		merged, err := k.ValidateConfigDetailed(newConfigSchema(t), []kernel.Source{src})
		require.NoError(t, err, "the import resolves through the kernel mapping")
		assert.Equal(t, "from-registry", lookupString(t, merged, "sentinel"))
	})

	t.Run("AcquireInstanceFromDir with a trailing value", func(t *testing.T) {
		dir := schematest.WriteInstanceDir(t, acquireInstanceFixture)
		inst, err := k.AcquireInstanceFromDir(ctx, dir, src)
		require.NoError(t, err, "the import resolves through the kernel mapping")
		assert.Equal(t, "from-registry", lookupString(t, inst.Package, "values.sentinel"))
	})

	t.Run("SynthesizeInstance with InstanceInput.Values", func(t *testing.T) {
		inst, err := k.SynthesizeInstance(ctx, kernel.InstanceInput{
			Module:    mod,
			Name:      "myrel",
			Namespace: "default",
			Values:    []kernel.Source{src},
		})
		require.NoError(t, err, "the import resolves through the kernel mapping")
		assert.Equal(t, "from-registry", lookupString(t, inst.Package, "values.sentinel"))
	})

	assert.Equal(t, schema.PublicRegistry, os.Getenv("CUE_REGISTRY"), "the process environment is never mutated")
}
