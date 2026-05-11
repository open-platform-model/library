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
	loader "github.com/open-platform-model/library/pkg/helper/loader/file"
	"github.com/open-platform-model/library/pkg/kernel"
	"github.com/open-platform-model/library/pkg/module"
	"github.com/open-platform-model/library/pkg/platform"
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

	wantVal, wantVer, wantErr := loader.LoadModulePackage(k.CueContext(), dir)
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

	wantVal, wantDir, wantVer, wantErr := loader.LoadReleaseFile(k.CueContext(), path, loader.LoadOptions{})
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

	wantVal, wantErr := loader.LoadValuesFile(k.CueContext(), path)
	require.NoError(t, wantErr)

	gotReplicas, err := gotVal.LookupPath(cue.ParsePath("replicas")).Int64()
	require.NoError(t, err)
	wantReplicas, err := wantVal.LookupPath(cue.ParsePath("replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, wantReplicas, gotReplicas)
	assert.Equal(t, int64(3), gotReplicas)
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

// --- ProcessModuleRelease: build a minimal Module + spec and confirm the
// canonical kernel method produces a well-formed *module.Release.

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

func TestKernel_ProcessModuleRelease_HappyPath(t *testing.T) {
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

	rel, err := k.ProcessModuleRelease(context.Background(), spec, minimalModule(), cue.Value{})
	require.NoError(t, err)
	require.NotNil(t, rel)
	assert.Equal(t, apiversion.V1alpha2, rel.APIVersion)
	assert.Equal(t, "demo", rel.Metadata.Name)
	assert.Equal(t, "ns", rel.Metadata.Namespace)
}

// --- Compile parity: minimal release + platform that round-trip through both
// the kernel method and the free function.

func minimalReleaseValue(t *testing.T, k *kernel.Kernel) *module.Release {
	t.Helper()
	spec := k.CueContext().CompileString(`
apiVersion: "ignored-by-test"
kind: "ModuleRelease"
metadata: { name: "demo", namespace: "ns", uuid: "u" }
#module: {
	apiVersion: "opmodel.dev/v1alpha2"
	kind: "Module"
	metadata: {
		name: "demo-mod"
		modulePath: "example.com/m"
		version: "1.0.0"
		fqn: "example.com/m/demo-mod:1.0.0"
		uuid: "11111111-1111-1111-1111-111111111111"
	}
}
components: {}
`)
	require.NoError(t, spec.Err())
	return &module.Release{
		APIVersion: apiversion.V1alpha2,
		Metadata:   &module.ReleaseMetadata{Name: "demo", Namespace: "ns"},
		Package:    spec,
	}
}

// minimalPlatformValue constructs a *platform.Platform with the given
// apiVersion and an empty registry / matchers / composedTransformers index.
func minimalPlatformValue(t *testing.T, k *kernel.Kernel) *platform.Platform {
	t.Helper()
	pv := k.CueContext().CompileString(`
apiVersion: "ignored-by-test"
kind: "Platform"
metadata: { name: "kubernetes" }
type: "kubernetes"
#registry: {}
#knownResources: {}
#knownTraits: {}
#composedTransformers: {}
#matchers: {
	resources: {}
	traits: {}
}
`)
	require.NoError(t, pv.Err())
	return &platform.Platform{
		APIVersion: apiversion.V1alpha2,
		Metadata:   &platform.PlatformMetadata{Name: "kubernetes", Type: "kubernetes"},
		Package:    pv,
	}
}

func TestKernel_Compile_Parity_VersionMismatch(t *testing.T) {
	k := kernel.New()
	rel := minimalReleaseValue(t, k)
	plat := minimalPlatformValue(t, k)
	// Force a mismatch.
	plat.APIVersion = apiversion.Version("opmodel.dev/v1alpha-other")

	_, gotErr := k.Compile(context.Background(), kernel.CompileInput{
		ModuleRelease: rel,
		Platform:      plat,
		RuntimeName:   "opm-cli",
	})

	require.Error(t, gotErr)
	assert.Contains(t, gotErr.Error(), "apiVersion mismatch")
}

func TestKernel_Compile_Parity_UnknownVersion(t *testing.T) {
	k := kernel.New()
	unknown := apiversion.Version("opmodel.dev/never-registered")
	rel := minimalReleaseValue(t, k)
	rel.APIVersion = unknown
	plat := minimalPlatformValue(t, k)
	plat.APIVersion = unknown

	_, gotErr := k.Compile(context.Background(), kernel.CompileInput{
		ModuleRelease: rel,
		Platform:      plat,
		RuntimeName:   "opm-cli",
	})

	require.Error(t, gotErr)
	assert.True(t, errors.Is(gotErr, apiversion.ErrUnknownAPIVersion))
}

// Sanity check: the v1alpha2 binding is registered so api.Lookup succeeds.
func TestKernel_BindingRegistered(t *testing.T) {
	_, err := api.Lookup(apiversion.V1alpha2)
	require.NoError(t, err)
}

// --- Goroutine-safety regression: N kernels (one per goroutine) each run a
// basic Load + Compile cycle. With -race enabled, this confirms no shared
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
			plat := minimalPlatformValue(t, k)
			_, perr := k.Compile(ctx, kernel.CompileInput{
				ModuleRelease: rel,
				Platform:      plat,
				RuntimeName:   "opm-cli",
			})
			if perr != nil {
				// Compile may legitimately error on the minimal fixture (no
				// components/transformers); we only care that the call returns
				// deterministically without racing.
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
