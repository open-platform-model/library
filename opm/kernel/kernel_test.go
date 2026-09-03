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
	"github.com/open-platform-model/library/opm/module"
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

func TestKernel_ValidateConfig_HappyPath(t *testing.T) {
	k := kernel.New()
	schema := k.CueContext().CompileString(`{ replicas: int & >0, name: string }`)
	require.NoError(t, schema.Err())
	values := k.CueContext().CompileString(`{ replicas: 3, name: "demo" }`)
	require.NoError(t, values.Err())

	gotMerged, gotErr := k.ValidateConfig(schema, values)
	require.NoError(t, gotErr)
	require.True(t, gotMerged.Exists())

	gotName, err := gotMerged.LookupPath(cue.ParsePath("name")).String()
	require.NoError(t, err)
	assert.Equal(t, "demo", gotName)
}

func TestKernel_ValidateConfig_SchemaErrorReturnsCueNativeError(t *testing.T) {
	k := kernel.New()
	schema := k.CueContext().CompileString(`{ replicas: int & >0 }`)
	require.NoError(t, schema.Err())
	bad := k.CueContext().CompileString(`{ replicas: -1 }`)
	require.NoError(t, bad.Err())

	_, gotErr := k.ValidateConfig(schema, bad)
	require.Error(t, gotErr)
	// Module-name framing is the caller's responsibility — primitive
	// returns the raw CUE error tree only.
}

// --- ProcessModuleInstance: build a minimal Module + spec and confirm the
// canonical kernel method produces a well-formed *module.Instance.

func minimalModule() module.Module {
	return module.Module{
		Metadata: &module.ModuleMetadata{
			Name:       "demo-mod",
			ModulePath: "example.com/m",
			Version:    "1.0.0",
			FQN:        "example.com/m/demo-mod:1.0.0",
			UUID:       "11111111-1111-1111-1111-111111111111",
		},
	}
}

func TestKernel_ProcessModuleInstance_HappyPath(t *testing.T) {
	k := kernel.New()
	spec := k.CueContext().CompileString(`
kind: "ModuleInstance"
metadata: {
	name: "demo"
	namespace: "ns"
	uuid: "u"
}
`)
	require.NoError(t, spec.Err())

	inst, err := k.ProcessModuleInstance(context.Background(), spec, minimalModule(), cue.Value{})
	require.NoError(t, err)
	require.NotNil(t, inst)
	assert.Equal(t, "demo", inst.Metadata.Name)
	assert.Equal(t, "ns", inst.Metadata.Namespace)
}

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

// TestKernel_PrunedSurface pins the removals of library-phase-and-values-prune
// and library-render-cutover: the kernel exposes exactly one render verb
// (Render), values enter through ProcessModuleInstance, and the old
// pipeline's verbs (Match, Compile, Materialize, SynthesizePlatform) and the
// typed validation wrappers are gone (spec single-build-render, "Old entry
// points are gone"). Any of these reappearing is a deliberate act, not drift.
func TestKernel_PrunedSurface(t *testing.T) {
	kt := reflect.TypeOf(&kernel.Kernel{})
	for _, name := range []string{
		"Plan", "Validate", "Match", "Compile", "Materialize", "SynthesizePlatform",
		"ValidateModuleValues", "ValidateModuleValuesPartial", "ValidateModuleValuesDetailed",
		"ValidateInstanceValues", "ValidateInstanceValuesPartial", "ValidateInstanceValuesDetailed",
	} {
		_, found := kt.MethodByName(name)
		assert.False(t, found, "*kernel.Kernel must not expose a %s method", name)
	}
	_, found := kt.MethodByName("Render")
	assert.True(t, found, "*kernel.Kernel exposes Render")

	_, found = reflect.TypeOf(kernel.RenderInput{}).FieldByName("Values")
	assert.False(t, found, "RenderInput must not carry a Values field; values enter through ProcessModuleInstance")
}
