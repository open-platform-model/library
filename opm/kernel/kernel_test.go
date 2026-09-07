package kernel_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/schema"
)

func TestNew_Default(t *testing.T) {
	k := kernel.New()
	require.NotNil(t, k)
	require.NotNil(t, k.CueContext(), "default Kernel must own a non-nil cue.Context")
}

func TestNew_CueContextStableAcrossCalls(t *testing.T) {
	k := kernel.New()
	first := k.CueContext()
	for range 5 {
		assert.Same(t, first, k.CueContext(), "CueContext must return the same *cue.Context for the lifetime of the Kernel")
	}
}

func TestNew_DistinctKernelsHaveDistinctContexts(t *testing.T) {
	a := kernel.New()
	b := kernel.New()
	assert.NotSame(t, a.CueContext(), b.CueContext(), "each Kernel owns its own *cue.Context")
}

// stubSchemaLoader is a [schema.Loader] that never touches a registry; it
// records the calls the Cache makes so a test can prove which loader ran.
type stubSchemaLoader struct{ calls *int }

func (s stubSchemaLoader) Load(ctx *cue.Context) (cue.Value, error) {
	*s.calls++
	return ctx.CompileString(`#ModuleInstance: {}`), nil
}

// unsetRegistry removes CUE_REGISTRY from the process environment for the
// duration of the test (t.Setenv registers the restore) and points the CUE
// module cache at an empty directory, so schema resolution must reach a
// registry rather than a warm cache entry.
func unsetRegistry(t *testing.T) {
	t.Helper()
	t.Setenv("CUE_REGISTRY", "")
	require.NoError(t, os.Unsetenv("CUE_REGISTRY"))
	t.Setenv("CUE_CACHE_DIR", t.TempDir())
}

// kernel-runtime spec, "Registry option seeds the schema loader": absent
// WithSchemaLoader, the schema cache resolves through the kernel's registry
// mapping rather than the process environment. The mapping is proved by the
// host it names — an unseeded loader would fall back to CUE's own default.
func TestNew_RegistryOptionSeedsSchemaLoader(t *testing.T) {
	unsetRegistry(t)

	k := kernel.New(kernel.WithRegistry("opmodel.dev=localhost:1+insecure"))
	_, err := k.SchemaCache().Get(k.CueContext())
	require.Error(t, err, "the seeded mapping serves nothing, so the fetch must fail through it")
	assert.Contains(t, err.Error(), "localhost:1", "the kernel's mapping, not the process environment, resolved the schema")
	assert.Empty(t, os.Getenv("CUE_REGISTRY"), "the process environment is not mutated")
}

// kernel-runtime spec, "Construction with options": an explicit schema
// loader wins over the registry-seeded default in either option order.
func TestNew_ExplicitSchemaLoaderWinsInEitherOrder(t *testing.T) {
	unsetRegistry(t)

	orders := map[string][]kernel.Option{
		"loader first":   nil,
		"registry first": nil,
	}
	calls := 0
	stub := stubSchemaLoader{calls: &calls}
	orders["loader first"] = []kernel.Option{kernel.WithSchemaLoader(stub), kernel.WithRegistry("opmodel.dev=localhost:1+insecure")}
	orders["registry first"] = []kernel.Option{kernel.WithRegistry("opmodel.dev=localhost:1+insecure"), kernel.WithSchemaLoader(stub)}

	for name, opts := range orders {
		t.Run(name, func(t *testing.T) {
			before := calls
			k := kernel.New(opts...)
			v, err := k.SchemaCache().Get(k.CueContext())
			require.NoError(t, err, "the explicit loader never contacts the unreachable mapping")
			assert.True(t, v.LookupPath(cue.ParsePath("#ModuleInstance")).Exists())
			assert.Equal(t, before+1, calls, "the explicit loader ran")
		})
	}
}

func writeTempModuleDir(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "module.cue"), []byte(content), 0o644))
	return dir
}

func writeTempInstanceDir(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "instance.cue"), []byte(content), 0o644))
	return dir
}

func TestKernel_ValidateConfigDetailed_HappyPath(t *testing.T) {
	k := kernel.New()
	schema := k.CueContext().CompileString(`{ replicas: int & >0, name: string }`)
	require.NoError(t, schema.Err())
	values := mustSource(t, k, "values.cue", `{ replicas: 3, name: "demo" }`)

	gotMerged, gotErr := k.ValidateConfigDetailed(schema, []kernel.Source{values})
	require.NoError(t, gotErr)
	require.True(t, gotMerged.Exists())

	gotName, err := gotMerged.LookupPath(cue.ParsePath("name")).String()
	require.NoError(t, err)
	assert.Equal(t, "demo", gotName)
}

// Instance processing is kernel-internal; its behaviour (concreteness on
// the built spec, metadata decoding, Source stamped) is covered through the
// acquirers in acquire_test.go and synth_test.go.

// --- Goroutine-safety regression: N kernels (one per goroutine) each drive
// the context-owning path (load + process). With -race enabled, this
// confirms no shared state leaks across kernels. The render side of the
// same claim (Render shares nothing, 0019 D8) is
// TestRender_ConcurrentKernelsShareNothing in render_test.go.

func TestKernel_GoroutineIsolation(t *testing.T) {
	const n = 8
	dir := writeTempModuleDir(t, `
package mod
kind: "Module"
metadata: {
	name:       "demo"
	modulePath: "example.com/modules"
	version:    "0.1.0"
}
`)
	instDir := writeTempInstanceDir(t, `
package instance
kind: "ModuleInstance"
metadata: {
	name: "demo"
	namespace: "ns"
}
#module: {kind: "Module"}
values: {replicas: 3}
`)

	var wg sync.WaitGroup
	errCh := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			k := kernel.New() // one Kernel per goroutine
			ctx := context.Background()

			mod, err := k.AcquireModuleFromDir(ctx, dir)
			if err != nil {
				errCh <- err
				return
			}
			if !mod.Package.Exists() {
				errCh <- errors.New("module value does not exist")
				return
			}
			inst, err := k.AcquireInstanceFromDir(ctx, instDir)
			if err != nil {
				errCh <- err
				return
			}
			if inst.Metadata.Name != "demo" {
				errCh <- errors.New("instance metadata not decoded")
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		require.NoError(t, err)
	}
}

// TestKernel_NoFinalizeMethod pins the absence of any finalization step on
// the kernel (spec kernel-runtime: "No finalization method on the Kernel";
// enhancement 0019 D1). Transformer inputs are bound as evaluated inside the
// render build; a Finalize method reappearing here is a deliberate act, not
// drift.
func TestKernel_NoFinalizeMethod(t *testing.T) {
	_, found := reflect.TypeOf(&kernel.Kernel{}).MethodByName("Finalize")
	assert.False(t, found, "*kernel.Kernel must not expose a Finalize method (0019 D1)")
}

// TestKernel_PrunedSurface pins the removals of library-phase-and-values-prune,
// library-render-cutover and cut-dead-surface: the kernel exposes exactly one
// render verb (Render) and one validation primitive (ValidateConfigDetailed),
// values enter through the acquire verbs and SynthesizeInstance, and the old
// pipeline's verbs (Match, Compile, Materialize, SynthesizePlatform), the
// typed validation wrappers, the single-value and partial validation
// variants, the exported instance processing step, the instance constructor
// wrapper and the string source loader are gone (spec single-build-render,
// "Old entry points are gone"; config-validation, "Single Kernel Validation
// Primitive"), and the removals of one-api-tier: the raw value tier
// (Load*Package plus the constructor wrappers) is gone, so an artifact comes
// from an acquire verb or from the package constructor a caller already holds
// a value for (kernel-runtime, "No raw-load methods on the Kernel";
// artifact-types, "No kernel constructor wrappers"). Any of these reappearing
// is a deliberate act, not drift.
func TestKernel_PrunedSurface(t *testing.T) {
	kt := reflect.TypeOf(&kernel.Kernel{})
	for _, name := range []string{
		"Plan", "Validate", "Match", "Compile", "Materialize", "SynthesizePlatform",
		"ValidateModuleValues", "ValidateModuleValuesPartial", "ValidateModuleValuesDetailed",
		"ValidateInstanceValues", "ValidateInstanceValuesPartial", "ValidateInstanceValuesDetailed",
		"ValidateConfig", "ValidateConfigPartial", "ProcessModuleInstance",
		"NewInstanceFromValue", "LoadSourceFromString",
		"LoadModulePackage", "LoadPlatformPackage", "LoadInstancePackage",
		"NewModuleFromValue", "NewPlatformFromValue",
	} {
		_, found := kt.MethodByName(name)
		assert.False(t, found, "*kernel.Kernel must not expose a %s method", name)
	}
	for _, name := range []string{
		"Render", "ValidateConfigDetailed", "SynthesizeInstance",
		"AcquireModuleFromDir", "AcquireModuleFromRegistry",
		"AcquirePlatformFromDir", "AcquireInstanceFromDir",
	} {
		_, found := kt.MethodByName(name)
		assert.True(t, found, "*kernel.Kernel exposes %s", name)
	}

	// artifact-types, "No option type for values": there is no AcquireOption
	// and no WithValues, because values are the variadic trailing argument.
	// A package-level function cannot be reflected on, so the shape of the
	// signature is what pins their absence.
	acquire, ok := kt.MethodByName("AcquireInstanceFromDir")
	require.True(t, ok)
	require.True(t, acquire.Type.IsVariadic(), "values are variadic, not an option list")
	assert.Equal(t, reflect.TypeFor[kernel.Source](), acquire.Type.In(acquire.Type.NumIn()-1).Elem(),
		"the variadic trailing argument is kernel.Source")

	_, found := reflect.TypeOf(kernel.RenderInput{}).FieldByName("Values")
	assert.False(t, found, "RenderInput must not carry a Values field; values enter through the acquire verbs and SynthesizeInstance")
	_, found = reflect.TypeOf(kernel.Source{}).FieldByName("Name")
	assert.False(t, found, "Source carries no display label; Origin is the attribution key")
}

// markerLoader is a schema.Loader that compiles a marker definition instead
// of resolving a registry, so a test can tell which loader backs a cache.
type markerLoader struct{ calls int }

func (l *markerLoader) Load(ctx *cue.Context) (cue.Value, error) {
	l.calls++
	return ctx.CompileString(`#Marker: true`), nil
}

// TestKernel_WithSchemaLoaderBacksTheCache pins the WithSchemaLoader option:
// the supplied Loader is what the kernel-owned cache resolves through, the
// cache memoizes one Load, and a nil Loader is ignored (the default
// OCILoader applies), so the option never yields a cache with no loader.
func TestKernel_WithSchemaLoaderBacksTheCache(t *testing.T) {
	ml := &markerLoader{}
	k := kernel.New(kernel.WithSchemaLoader(ml))
	require.NotNil(t, k.SchemaCache())

	val, err := k.SchemaCache().Get(k.CueContext())
	require.NoError(t, err)
	assert.True(t, val.LookupPath(cue.ParsePath("#Marker")).Exists(), "the cache resolves through the supplied loader")
	_, err = k.SchemaCache().Get(k.CueContext())
	require.NoError(t, err)
	assert.Equal(t, 1, ml.calls, "one Load per cache")

	var nilLoader schema.Loader
	k2 := kernel.New(kernel.WithSchemaLoader(nilLoader))
	require.NotNil(t, k2.SchemaCache(), "a nil loader is ignored, the default applies")
	assert.NotSame(t, k.SchemaCache(), k2.SchemaCache(), "one Cache per Kernel")
}
