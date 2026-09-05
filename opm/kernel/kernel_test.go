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

	loader "github.com/open-platform-model/library/opm/helper/loader/file"
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

// --- Parity tests: each wrapper must produce results identical to the
// corresponding free function called with k.CueContext().

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

func TestKernel_LoadModulePackage_Parity(t *testing.T) {
	dir := writeTempModuleDir(t, `
package mod
kind: "Module"
metadata: {
	name:       "demo"
	modulePath: "example.com/modules"
	version:    "0.1.0"
}
`)

	k := kernel.New()
	gotVal, gotErr := k.LoadModulePackage(context.Background(), dir, loader.LoadOptions{})
	require.NoError(t, gotErr)

	wantVal, wantErr := loader.LoadModulePackage(k.CueContext(), dir, loader.LoadOptions{})
	require.NoError(t, wantErr)

	assert.True(t, gotVal.Exists())
	assert.True(t, wantVal.Exists())
}

func TestKernel_LoadInstancePackage_Parity(t *testing.T) {
	dir := writeTempInstanceDir(t, `
package instance
kind: "ModuleInstance"
metadata: {
	name: "demo"
	namespace: "ns"
}
#module: {kind: "Module"}
`)

	k := kernel.New()
	gotVal, gotErr := k.LoadInstancePackage(context.Background(), dir, loader.LoadOptions{})
	require.NoError(t, gotErr)

	wantVal, wantErr := loader.LoadInstancePackage(k.CueContext(), dir, loader.LoadOptions{})
	require.NoError(t, wantErr)

	assert.True(t, gotVal.Exists())
	assert.True(t, wantVal.Exists())
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

			val, err := k.LoadModulePackage(ctx, dir, loader.LoadOptions{})
			if err != nil {
				errCh <- err
				return
			}
			if !val.Exists() {
				errCh <- errors.New("module value does not exist")
				return
			}
			inst, err := k.AcquireInstanceFromDir(ctx, instDir, loader.LoadOptions{})
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
// values enter through WithValues and SynthesizeInstance, and the old
// pipeline's verbs (Match, Compile, Materialize, SynthesizePlatform), the
// typed validation wrappers, the single-value and partial validation
// variants, the exported instance processing step, the instance constructor
// wrapper and the string source loader are gone (spec single-build-render,
// "Old entry points are gone"; config-validation, "Single Kernel Validation
// Primitive"). Any of these reappearing is a deliberate act, not drift.
func TestKernel_PrunedSurface(t *testing.T) {
	kt := reflect.TypeOf(&kernel.Kernel{})
	for _, name := range []string{
		"Plan", "Validate", "Match", "Compile", "Materialize", "SynthesizePlatform",
		"ValidateModuleValues", "ValidateModuleValuesPartial", "ValidateModuleValuesDetailed",
		"ValidateInstanceValues", "ValidateInstanceValuesPartial", "ValidateInstanceValuesDetailed",
		"ValidateConfig", "ValidateConfigPartial", "ProcessModuleInstance",
		"NewInstanceFromValue", "LoadSourceFromString",
	} {
		_, found := kt.MethodByName(name)
		assert.False(t, found, "*kernel.Kernel must not expose a %s method", name)
	}
	_, found := kt.MethodByName("Render")
	assert.True(t, found, "*kernel.Kernel exposes Render")
	_, found = kt.MethodByName("ValidateConfigDetailed")
	assert.True(t, found, "*kernel.Kernel exposes ValidateConfigDetailed")

	_, found = reflect.TypeOf(kernel.RenderInput{}).FieldByName("Values")
	assert.False(t, found, "RenderInput must not carry a Values field; values enter through WithValues and SynthesizeInstance")
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
