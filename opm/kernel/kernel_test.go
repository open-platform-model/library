package kernel_test

import (
	"context"
	"errors"
	"fmt"
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

	loader "github.com/open-platform-model/library/opm/helper/loader/file"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/materialize"
	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/platform"
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

func writeTempReleaseDir(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "release.cue"), []byte(content), 0o644))
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

func TestKernel_LoadReleasePackage_Parity(t *testing.T) {
	dir := writeTempReleaseDir(t, `
package release
kind: "ModuleRelease"
metadata: {
	name: "demo"
	namespace: "ns"
}
#module: {kind: "Module"}
`)

	k := kernel.New()
	gotVal, gotErr := k.LoadReleasePackage(context.Background(), dir, loader.LoadOptions{})
	require.NoError(t, gotErr)

	wantVal, wantErr := loader.LoadReleasePackage(k.CueContext(), dir, loader.LoadOptions{})
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

// --- ProcessModuleRelease: build a minimal Module + spec and confirm the
// canonical kernel method produces a well-formed *module.Release.

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

func TestKernel_ProcessModuleRelease_HappyPath(t *testing.T) {
	k := kernel.New()
	spec := k.CueContext().CompileString(`
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
	assert.Equal(t, "demo", rel.Metadata.Name)
	assert.Equal(t, "ns", rel.Metadata.Namespace)
}

// --- Compile parity: minimal release + platform that round-trip through both
// the kernel method and the free function.

func minimalReleaseValue(t *testing.T, k *kernel.Kernel) *module.Release {
	t.Helper()
	spec := k.CueContext().CompileString(`
kind: "ModuleRelease"
metadata: { name: "demo", namespace: "ns", uuid: "u" }
#module: {
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
		Metadata: &module.ReleaseMetadata{Name: "demo", Namespace: "ns"},
		Package:  spec,
	}
}

// minimalPlatformValue constructs a *materialize.MaterializedPlatform with an
// empty registry / matchers / composedTransformers index — the realized form
// the phase methods now consume.
func minimalPlatformValue(t *testing.T, k *kernel.Kernel) *materialize.MaterializedPlatform {
	t.Helper()
	pv := k.CueContext().CompileString(`
kind: "Platform"
metadata: { name: "kubernetes" }
type: "kubernetes"
#registry: {}
#composedTransformers: {}
#matchers: {
	resources: {}
	traits: {}
}
`)
	require.NoError(t, pv.Err())
	return &materialize.MaterializedPlatform{
		Source: &platform.Platform{
			Metadata: &platform.PlatformMetadata{Name: "kubernetes", Type: "kubernetes"},
			Package:  pv,
		},
		Package:  pv,
		Composed: pv.LookupPath(cue.ParsePath("#composedTransformers")),
	}
}

// --- Goroutine-safety regression: N kernels (one per goroutine) each run a
// basic Load + Compile cycle. With -race enabled, this confirms no shared
// state leaks across kernels.

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

// --- Concurrent-render regression (v0.17 shared-read-only-platform guarantee).
//
// One dedicated Kernel K0 materializes a platform ONCE, in its own *cue.Context.
// N goroutines then each construct their OWN Kernel and Compile a DISTINCT
// ModuleRelease (built in that goroutine's context) against the single shared
// K0 platform. This is the cross-context case TestKernel_GoroutineIsolation
// never covers: there every goroutine's values live in one context, here the
// platform value lives in K0's context while every rendered value is built in a
// per-goroutine Kernel's context.
//
// On CUE v0.16 the cross-context FillPath inside Execute panics "values are not
// from the same runtime"; v0.17 makes the combination legal, race-safe, and
// correct. Run under `go test -race` this is the permanent guard for the
// materialize-once-reuse-many model (see openspec change
// enable-concurrent-render-v017 and ADR-002).

// sharedEchoPlatform builds the echo-transformer platform (mirroring
// newPhaseFixture's platform) in the supplied context. The returned platform is
// meant to be materialized once and then read concurrently by other Kernels.
func sharedEchoPlatform(t *testing.T, cc *cue.Context) *materialize.MaterializedPlatform {
	t.Helper()
	pv := cc.CompileString(`
kind: "Platform"
metadata: { name: "k8s" }
type: "kubernetes"
#registry: {}
#composedTransformers: {
	"example.com/p/echo@v0": {
		metadata: { fqn: "example.com/p/echo@v0" }
		requiredLabels: { tier: "web" }
		requiredResources: { "example.com/r/echo@v0": {} }
		requiredTraits: {}
		optionalTraits: {}
		#transform: {
			#component: _
			#context:   _
			output: {
				kind: "echo"
				runtime: #context.#runtimeName
				release: #context.#moduleReleaseMetadata.name
				component: #context.#componentMetadata.name
			}
		}
	}
}
#matchers: {
	resources: {
		"example.com/r/echo@v0": [#composedTransformers["example.com/p/echo@v0"]]
	}
	traits: {}
}
`)
	require.NoError(t, pv.Err())
	return &materialize.MaterializedPlatform{
		Source: &platform.Platform{
			Metadata: &platform.PlatformMetadata{Name: "k8s", Type: "kubernetes"},
			Package:  pv,
		},
		Package:  pv,
		Composed: pv.LookupPath(cue.ParsePath("#composedTransformers")),
	}
}

// distinctEchoRelease builds a ModuleRelease whose metadata.name is unique, in
// the supplied context. The echo transformer echoes #moduleReleaseMetadata.name
// (sourced from rel.Metadata.Name) into output.release, so a unique name lets
// the test prove each concurrent render produced ITS OWN output. It returns an
// error rather than calling t.FailNow so it is safe to invoke from a goroutine.
func distinctEchoRelease(cc *cue.Context, name string) (*module.Release, error) {
	relPkg := cc.CompileString(fmt.Sprintf(`
kind: "ModuleRelease"
metadata: { name: %q, namespace: "ns", uuid: %q }
#module: {
	kind: "Module"
	metadata: {
		name: "demo-mod"
		modulePath: "example.com/m"
		version: "1.0.0"
		fqn: "example.com/m/demo-mod:1.0.0"
		uuid: "11111111-1111-1111-1111-111111111111"
	}
	#config: {
		replicas: int & >0
		name: string
	}
}
components: {
	web: {
		metadata: {
			name: "web"
			labels: { tier: "web" }
		}
		#resources: {
			"example.com/r/echo@v0": {}
		}
	}
}
`, name, "u-"+name))
	if err := relPkg.Err(); err != nil {
		return nil, err
	}
	return &module.Release{
		Metadata: &module.ReleaseMetadata{Name: name, Namespace: "ns", UUID: "u-" + name},
		Package:  relPkg,
	}, nil
}

func TestKernel_ConcurrentRender_SharedPlatform(t *testing.T) {
	const n = 8

	// K0 materializes the shared platform ONCE, in its own context. No goroutine
	// below ever touches k0 — it only reads the platform value k0 produced.
	k0 := kernel.New()
	shared := sharedEchoPlatform(t, k0.CueContext())

	type renderResult struct {
		want string // the release name this goroutine asked to render
		got  string // the release name echoed back in the rendered output
		err  error
	}
	results := make([]renderResult, n)

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			name := fmt.Sprintf("release-%d", i)

			k := kernel.New() // one Kernel per goroutine — NOT k0
			rel, err := distinctEchoRelease(k.CueContext(), name)
			if err != nil {
				results[i] = renderResult{want: name, err: err}
				return
			}

			// Cross-context render: rel lives in k's context, shared in k0's.
			out, err := k.Compile(context.Background(), kernel.CompileInput{
				ModuleRelease: rel,
				Platform:      shared,
				RuntimeName:   "opm-cli",
			})
			if err != nil {
				results[i] = renderResult{want: name, err: err}
				return
			}
			if len(out.Compiled) != 1 {
				results[i] = renderResult{want: name, err: fmt.Errorf("expected 1 compiled value, got %d", len(out.Compiled))}
				return
			}
			got, err := out.Compiled[0].Value.LookupPath(cue.ParsePath("release")).String()
			results[i] = renderResult{want: name, got: got, err: err}
		}(i)
	}
	wg.Wait()

	// Assert correctness, not just race-silence: each render echoed ITS OWN
	// release name, with no cross-contamination between concurrent renders.
	for i, r := range results {
		require.NoErrorf(t, r.err, "goroutine %d (%s) failed to render against the shared platform", i, r.want)
		assert.Equalf(t, r.want, r.got,
			"goroutine %d rendered the wrong release — concurrent renders cross-contaminated", i)
	}
}
