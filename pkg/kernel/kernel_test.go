package kernel_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/open-platform-model/library/pkg/api"
	_ "github.com/open-platform-model/library/pkg/api/v1alpha2"
	"github.com/open-platform-model/library/pkg/apiversion"
	"github.com/open-platform-model/library/pkg/compile"
	"github.com/open-platform-model/library/pkg/kernel"
	"github.com/open-platform-model/library/pkg/loader"
	"github.com/open-platform-model/library/pkg/module"
	"github.com/open-platform-model/library/pkg/provider"
	"github.com/open-platform-model/library/pkg/validate"
)

// fakeClock returns a fixed time. Lets WithClock be observable from tests.
type fakeClock struct{ now time.Time }

func (f fakeClock) Now() time.Time { return f.now }

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

func TestNew_WithLogger(t *testing.T) {
	custom := slog.New(slog.NewTextHandler(io.Discard, nil))
	k := kernel.New(kernel.WithLogger(custom))
	require.NotNil(t, k)
	// Logger is intentionally not exposed; we exercise the option to confirm it
	// applies without panicking and the kernel is otherwise usable.
	assert.NotNil(t, k.CueContext())
}

func TestNew_WithTracer(t *testing.T) {
	tr := noop.NewTracerProvider().Tracer("kernel-test")
	k := kernel.New(kernel.WithTracer(tr))
	require.NotNil(t, k)
	assert.NotNil(t, k.CueContext())
}

func TestNew_WithClock(t *testing.T) {
	pinned := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	k := kernel.New(kernel.WithClock(fakeClock{now: pinned}))
	require.NotNil(t, k)
	// Clock is internal; we exercise the option to confirm it is accepted.
	assert.NotNil(t, k.CueContext())
}

func TestNew_NilOptionsAreIgnored(t *testing.T) {
	// Passing nil dependencies to options should not replace defaults.
	k := kernel.New(
		kernel.WithLogger(nil),
		kernel.WithTracer(nil),
		kernel.WithClock(nil),
	)
	require.NotNil(t, k)
	assert.NotNil(t, k.CueContext())
}

// --- Parity tests: each wrapper must produce results identical to the
// corresponding free function called with k.CueContext().

func writeTempModuleDir(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "module.cue"), []byte(content), 0o644))
	return dir
}

func writeTempReleaseFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "release.cue")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func writeTempValuesFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "values.cue")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func TestKernel_LoadModulePackage_Parity(t *testing.T) {
	dir := writeTempModuleDir(t, `
package mod
apiVersion: "opmodel.dev/v1alpha2"
kind: "Module"
`)

	k := kernel.New()
	gotVal, gotVer, gotErr := k.LoadModulePackage(context.Background(), dir)
	require.NoError(t, gotErr)

	wantVal, wantVer, wantErr := loader.LoadModulePackage(k.CueContext(), dir) //nolint:staticcheck // SA1019: parity test against deprecated free function
	require.NoError(t, wantErr)

	assert.Equal(t, wantVer, gotVer)
	assert.Equal(t, apiversion.V1alpha2, gotVer)
	assert.True(t, gotVal.Exists())
	assert.True(t, wantVal.Exists())
}

func TestKernel_LoadReleaseFile_Parity(t *testing.T) {
	path := writeTempReleaseFile(t, `
package release
apiVersion: "opmodel.dev/v1alpha2"
kind: "ModuleRelease"
metadata: {
	name: "demo"
	namespace: "ns"
}
`)

	k := kernel.New()
	gotVal, gotDir, gotVer, gotErr := k.LoadReleaseFile(context.Background(), path, loader.LoadOptions{})
	require.NoError(t, gotErr)

	wantVal, wantDir, wantVer, wantErr := loader.LoadReleaseFile(k.CueContext(), path, loader.LoadOptions{}) //nolint:staticcheck // SA1019: parity test against deprecated free function
	require.NoError(t, wantErr)

	assert.Equal(t, wantDir, gotDir)
	assert.Equal(t, wantVer, gotVer)
	assert.Equal(t, apiversion.V1alpha2, gotVer)
	assert.True(t, gotVal.Exists())
	assert.True(t, wantVal.Exists())
}

func TestKernel_LoadValuesFile_Parity(t *testing.T) {
	path := writeTempValuesFile(t, `
package values
values: {
	replicas: 3
	name: "demo"
}
`)

	k := kernel.New()
	gotVal, gotErr := k.LoadValuesFile(context.Background(), path)
	require.NoError(t, gotErr)

	wantVal, wantErr := loader.LoadValuesFile(k.CueContext(), path) //nolint:staticcheck // SA1019: parity test against deprecated free function
	require.NoError(t, wantErr)

	gotReplicas, err := gotVal.LookupPath(cue.ParsePath("replicas")).Int64()
	require.NoError(t, err)
	wantReplicas, err := wantVal.LookupPath(cue.ParsePath("replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, wantReplicas, gotReplicas)
	assert.Equal(t, int64(3), gotReplicas)
}

func TestKernel_LoadProvider_Parity(t *testing.T) {
	k := kernel.New()
	v := k.CueContext().CompileString(`
apiVersion: "opmodel.dev/v1alpha2"
kind: "Provider"
metadata: { name: "kubernetes", version: "v0" }
`)
	require.NoError(t, v.Err())
	providers := map[string]cue.Value{"kubernetes": v}

	gotP, gotErr := k.LoadProvider("kubernetes", providers)
	require.NoError(t, gotErr)

	wantP, wantErr := loader.LoadProvider("kubernetes", providers) //nolint:staticcheck // SA1019: parity test against deprecated free function
	require.NoError(t, wantErr)

	assert.Equal(t, wantP.APIVersion, gotP.APIVersion)
	assert.Equal(t, wantP.Metadata.Name, gotP.Metadata.Name)
}

func TestKernel_ValidateConfig_Parity(t *testing.T) {
	k := kernel.New()
	schema := k.CueContext().CompileString(`{ replicas: int & >0, name: string }`)
	require.NoError(t, schema.Err())
	values := []cue.Value{
		k.CueContext().CompileString(`{ replicas: 3, name: "demo" }`),
	}
	require.NoError(t, values[0].Err())

	gotMerged, gotErr := k.ValidateConfig(schema, values, "module", "demo")
	wantMerged, wantErr := validate.Config(schema, values, "module", "demo") //nolint:staticcheck // SA1019: parity test against deprecated free function

	assert.Equal(t, wantErr == nil, gotErr == nil)
	require.Nil(t, gotErr)
	require.True(t, gotMerged.Exists())
	require.True(t, wantMerged.Exists())

	gotName, err := gotMerged.LookupPath(cue.ParsePath("name")).String()
	require.NoError(t, err)
	wantName, err := wantMerged.LookupPath(cue.ParsePath("name")).String()
	require.NoError(t, err)
	assert.Equal(t, wantName, gotName)
}

func TestKernel_ValidateConfig_Parity_Error(t *testing.T) {
	k := kernel.New()
	schema := k.CueContext().CompileString(`{ replicas: int & >0 }`)
	require.NoError(t, schema.Err())
	bad := []cue.Value{k.CueContext().CompileString(`{ replicas: -1 }`)}
	require.NoError(t, bad[0].Err())

	_, gotErr := k.ValidateConfig(schema, bad, "module", "demo")
	_, wantErr := validate.Config(schema, bad, "module", "demo") //nolint:staticcheck // SA1019: parity test against deprecated free function

	require.NotNil(t, gotErr)
	require.NotNil(t, wantErr)
	assert.Equal(t, wantErr.Context, gotErr.Context)
	assert.Equal(t, wantErr.Name, gotErr.Name)
}

// --- ParseModuleRelease parity: build a minimal Module + spec, exercise both
// the kernel method and the free function and confirm both produce equivalent
// *module.Release values.

func minimalModule() module.Module {
	return module.Module{
		APIVersion: apiversion.V1alpha2,
		Metadata: &module.ModuleMetadata{
			Name:       "demo-mod",
			ModulePath: "example.com/m",
			Version:    "1.0.0",
			FQN:        "example.com/m/demo-mod:1.0.0",
			UUID:       "11111111-1111-1111-1111-111111111111",
		},
	}
}

func TestKernel_ParseModuleRelease_Parity(t *testing.T) {
	k := kernel.New()
	spec := k.CueContext().CompileString(`
apiVersion: "opmodel.dev/v1alpha2"
kind: "ModuleRelease"
metadata: {
	name: "demo"
	namespace: "ns"
	uuid: "u"
}
`)
	require.NoError(t, spec.Err())

	gotRel, gotErr := k.ParseModuleRelease(context.Background(), spec, minimalModule(), nil)
	require.NoError(t, gotErr)

	wantRel, wantErr := module.ParseModuleRelease(context.Background(), spec, minimalModule(), nil) //nolint:staticcheck // SA1019: parity test against deprecated free function
	require.NoError(t, wantErr)

	require.NotNil(t, gotRel)
	require.NotNil(t, wantRel)
	assert.Equal(t, wantRel.APIVersion, gotRel.APIVersion)
	assert.Equal(t, wantRel.Metadata.Name, gotRel.Metadata.Name)
	assert.Equal(t, wantRel.Metadata.Namespace, gotRel.Metadata.Namespace)
}

// --- Render parity: minimal release + provider that round-trip through both
// the kernel method and the free function.

func minimalReleaseValue(t *testing.T, k *kernel.Kernel) *module.Release {
	t.Helper()
	spec := k.CueContext().CompileString(`
apiVersion: "ignored-by-test"
kind: "ModuleRelease"
metadata: { name: "demo", namespace: "ns", uuid: "u" }
components: {}
`)
	require.NoError(t, spec.Err())
	return &module.Release{
		APIVersion: apiversion.V1alpha2,
		Metadata:   &module.ReleaseMetadata{Name: "demo", Namespace: "ns"},
		Package:    spec,
	}
}

func minimalProviderValue(t *testing.T, k *kernel.Kernel) *provider.Provider {
	t.Helper()
	pv := k.CueContext().CompileString(`
apiVersion: "ignored-by-test"
kind: "Provider"
metadata: { name: "kubernetes", version: "v0" }
#transformers: {}
`)
	require.NoError(t, pv.Err())
	return &provider.Provider{
		APIVersion: apiversion.V1alpha2,
		Metadata:   &provider.ProviderMetadata{Name: "kubernetes"},
		Data:       pv,
	}
}

func TestKernel_NewRenderModule_Parity(t *testing.T) {
	k := kernel.New()
	p := minimalProviderValue(t, k)

	got := k.NewRenderModule(p, "opm-cli")
	want := compile.NewModule(p, "opm-cli") //nolint:staticcheck // SA1019: parity test against deprecated free function

	require.NotNil(t, got)
	require.NotNil(t, want)
	// Both should produce non-nil *compile.Module values for the same provider.
	// Internal state is unexported; we round-trip through the public APIs in
	// the ProcessModuleRelease parity test below.
	assert.IsType(t, want, got)
}

func TestKernel_ProcessModuleRelease_Parity_VersionMismatch(t *testing.T) {
	k := kernel.New()
	rel := minimalReleaseValue(t, k)
	p := minimalProviderValue(t, k)
	// Force a mismatch.
	p.APIVersion = apiversion.Version("opmodel.dev/v1alpha-other")

	_, gotErr := k.ProcessModuleRelease(context.Background(), rel, p, "opm-cli")
	_, wantErr := compile.ProcessModuleRelease(context.Background(), rel, p, "opm-cli") //nolint:staticcheck // SA1019: parity test against deprecated free function

	require.Error(t, gotErr)
	require.Error(t, wantErr)
	assert.Contains(t, gotErr.Error(), "apiVersion mismatch")
	assert.Contains(t, wantErr.Error(), "apiVersion mismatch")
}

func TestKernel_ProcessModuleRelease_Parity_UnknownVersion(t *testing.T) {
	k := kernel.New()
	unknown := apiversion.Version("opmodel.dev/never-registered")
	rel := minimalReleaseValue(t, k)
	rel.APIVersion = unknown
	p := minimalProviderValue(t, k)
	p.APIVersion = unknown

	_, gotErr := k.ProcessModuleRelease(context.Background(), rel, p, "opm-cli")
	_, wantErr := compile.ProcessModuleRelease(context.Background(), rel, p, "opm-cli") //nolint:staticcheck // SA1019: parity test against deprecated free function

	require.Error(t, gotErr)
	require.Error(t, wantErr)
	assert.True(t, errors.Is(gotErr, apiversion.ErrUnknownAPIVersion))
	assert.True(t, errors.Is(wantErr, apiversion.ErrUnknownAPIVersion))
}

// Sanity check: the v1alpha2 binding is registered so api.Lookup succeeds.
func TestKernel_BindingRegistered(t *testing.T) {
	_, err := api.Lookup(apiversion.V1alpha2)
	require.NoError(t, err)
}

// --- Goroutine-safety regression: N kernels (one per goroutine) each run a
// basic Load + Process cycle. With -race enabled, this confirms no shared
// state leaks across kernels.

func TestKernel_GoroutineIsolation(t *testing.T) {
	const n = 8
	dir := writeTempModuleDir(t, `
package mod
apiVersion: "opmodel.dev/v1alpha2"
kind: "Module"
`)

	var wg sync.WaitGroup
	errCh := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			k := kernel.New() // one Kernel per goroutine
			ctx := context.Background()

			val, ver, err := k.LoadModulePackage(ctx, dir)
			if err != nil {
				errCh <- err
				return
			}
			if !val.Exists() {
				errCh <- errors.New("module value does not exist")
				return
			}
			if ver != apiversion.V1alpha2 {
				errCh <- errors.New("unexpected apiversion")
				return
			}

			rel := minimalReleaseValue(t, k)
			p := minimalProviderValue(t, k)
			_, perr := k.ProcessModuleRelease(ctx, rel, p, "opm-cli")
			if perr != nil {
				// ProcessModuleRelease may legitimately error on the minimal
				// fixture (no components/transformers); we only care that the
				// call returns deterministically without racing.
				_ = perr
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		require.NoError(t, err)
	}
}
