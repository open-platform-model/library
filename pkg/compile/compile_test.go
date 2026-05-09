package compile_test

import (
	"context"
	"errors"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/pkg/api"
	_ "github.com/open-platform-model/library/pkg/api/v1alpha2"
	"github.com/open-platform-model/library/pkg/apiversion"
	"github.com/open-platform-model/library/pkg/compile"
	"github.com/open-platform-model/library/pkg/module"
	"github.com/open-platform-model/library/pkg/platform"
)

// minimalRelease constructs a *module.Release with the given apiVersion. The
// CUE Spec is a synthesised value sufficient for CompileModuleRelease's early
// validations (APIVersion check, MatchComponents lookup).
func minimalRelease(t *testing.T, ver apiversion.Version) *module.Release {
	t.Helper()
	ctx := cuecontext.New()
	spec := ctx.CompileString(`
apiVersion: "ignored-by-test"
kind: "ModuleRelease"
metadata: { name: "demo", namespace: "ns", uuid: "u" }
components: {}
`)
	require.NoError(t, spec.Err())
	return &module.Release{
		APIVersion: ver,
		Metadata:   &module.ReleaseMetadata{Name: "demo", Namespace: "ns"},
		Package:    spec,
	}
}

// minimalPlatform constructs a *platform.Platform with the given apiVersion
// and an empty registry / matchers / composedTransformers index.
func minimalPlatform(t *testing.T, ver apiversion.Version) *platform.Platform {
	t.Helper()
	ctx := cuecontext.New()
	pv := ctx.CompileString(`
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
		APIVersion: ver,
		Metadata:   &platform.PlatformMetadata{Name: "kubernetes", Type: "kubernetes"},
		Package:    pv,
	}
}

func TestCompileModuleRelease_VersionMismatch(t *testing.T) {
	rel := minimalRelease(t, apiversion.V1alpha2)
	plat := minimalPlatform(t, apiversion.Version("opmodel.dev/v1alpha-other"))

	_, err := compile.CompileModuleRelease(context.Background(), rel, plat, "opm-cli") //nolint:staticcheck // SA1019: testing the deprecated free function
	require.Error(t, err)
	assert.Contains(t, err.Error(), "apiVersion mismatch")
}

func TestCompileModuleRelease_NoBindingRegistered(t *testing.T) {
	// Both release and platform declare the same unrecognised version. Mismatch
	// check passes; api.Lookup then fails.
	unknown := apiversion.Version("opmodel.dev/never-registered")
	rel := minimalRelease(t, unknown)
	plat := minimalPlatform(t, unknown)

	_, err := compile.CompileModuleRelease(context.Background(), rel, plat, "opm-cli") //nolint:staticcheck // SA1019: testing the deprecated free function
	require.Error(t, err)
	assert.True(t, errors.Is(err, apiversion.ErrUnknownAPIVersion), "want ErrUnknownAPIVersion in chain, got %v", err)
}

func TestMatch_RequiresBinding(t *testing.T) {
	ctx := cuecontext.New()
	components := ctx.CompileString(`{}`)
	require.NoError(t, components.Err())
	plat := minimalPlatform(t, apiversion.V1alpha2)

	_, err := compile.Match(components, plat, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "binding is required")
}

func TestMatch_RequiresPlatform(t *testing.T) {
	ctx := cuecontext.New()
	components := ctx.CompileString(`{}`)
	require.NoError(t, components.Err())
	b, err := api.Lookup(apiversion.V1alpha2)
	require.NoError(t, err)

	_, err = compile.Match(components, nil, b)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "platform is required")
}

func TestMatch_UsesBindingPaths(t *testing.T) {
	// Construct a Platform whose #matchers index points a resource FQN at a
	// transformer requiring a label that the component carries. Match should
	// mark the pair as matched, proving the path lookups go through the
	// binding rather than crashing on absent paths.
	ctx := cuecontext.New()
	pv := ctx.CompileString(`
apiVersion: "opmodel.dev/v1alpha2"
kind: "Platform"
metadata: { name: "k8s" }
type: "kubernetes"
#registry: {}
#knownResources: {}
#knownTraits: {}
#composedTransformers: {
	"opmodel.dev/p/k8s/x@v0": {
		requiredLabels: { tier: "web" }
		requiredResources: {}
		requiredTraits: {}
		optionalTraits: {}
	}
}
#matchers: {
	resources: {
		"opmodel.dev/r/echo@v0": ["opmodel.dev/p/k8s/x@v0"]
	}
	traits: {}
}
`)
	require.NoError(t, pv.Err())
	plat := &platform.Platform{
		APIVersion: apiversion.V1alpha2,
		Metadata:   &platform.PlatformMetadata{Name: "k8s", Type: "kubernetes"},
		Package:    pv,
	}
	components := ctx.CompileString(`
"web": {
	metadata: { labels: { tier: "web" } }
	#resources: { "opmodel.dev/r/echo@v0": {} }
}
`)
	require.NoError(t, components.Err())

	b, err := api.Lookup(apiversion.V1alpha2)
	require.NoError(t, err)

	plan, err := compile.Match(components, plat, b)
	require.NoError(t, err)
	require.NotNil(t, plan)
	pairs := plan.MatchedPairs()
	require.Len(t, pairs, 1)
	assert.Equal(t, "web", pairs[0].ComponentName)
	assert.Equal(t, "opmodel.dev/p/k8s/x@v0", pairs[0].TransformerFQN)
}

// TestMatch_AmbiguousFQN exercises the defensive multi-fulfiller check.
// Catalog 014 D13 forbids more than one transformer per FQN at the platform
// layer; if a hand-built Platform somehow violates that, the matcher must
// flag the FQN as ambiguous and not pair the component with any candidate.
func TestMatch_AmbiguousFQN(t *testing.T) {
	ctx := cuecontext.New()
	pv := ctx.CompileString(`
apiVersion: "opmodel.dev/v1alpha2"
kind: "Platform"
metadata: { name: "k8s" }
type: "kubernetes"
#registry: {}
#knownResources: {}
#knownTraits: {}
#composedTransformers: {
	"example.com/p/a@v0": {
		requiredLabels: {}
		requiredResources: {}
		requiredTraits: {}
		optionalTraits: {}
	}
	"example.com/p/b@v0": {
		requiredLabels: {}
		requiredResources: {}
		requiredTraits: {}
		optionalTraits: {}
	}
}
#matchers: {
	resources: {
		"example.com/r/echo@v0": ["example.com/p/a@v0", "example.com/p/b@v0"]
	}
	traits: {}
}
`)
	require.NoError(t, pv.Err())
	plat := &platform.Platform{
		APIVersion: apiversion.V1alpha2,
		Metadata:   &platform.PlatformMetadata{Name: "k8s", Type: "kubernetes"},
		Package:    pv,
	}
	components := ctx.CompileString(`
"web": {
	metadata: { labels: {} }
	#resources: { "example.com/r/echo@v0": {} }
}
`)
	require.NoError(t, components.Err())

	b, err := api.Lookup(apiversion.V1alpha2)
	require.NoError(t, err)

	plan, err := compile.Match(components, plat, b)
	require.NoError(t, err)
	require.NotNil(t, plan)
	assert.Contains(t, plan.Ambiguous, "example.com/r/echo@v0",
		"ambiguous FQN must surface in MatchPlan.Ambiguous")
	assert.Empty(t, plan.MatchedPairs(),
		"ambiguous FQN must not produce a matched pair")
	assert.Contains(t, plan.Unmatched, "web",
		"component with only ambiguous demand resolves to no matched transformer")
}

// keep a reference to cue.Value to silence unused-import warnings in builds
// where the helper structs change shape during refactors.
var _ = cue.Value{}

// TestReleaseImplementsReleaseView confirms that *module.Release satisfies
// api.ReleaseView so the v1alpha2 binding can call it without an adapter.
// This is the static guard for the moves added in group 7.
func TestReleaseImplementsReleaseView(t *testing.T) {
	var _ api.ReleaseView = (*module.Release)(nil)
}

// TestCompileModuleRelease_RendersContextViaBinding is a coarse snapshot test
// for the binding-driven context injection. It builds a minimal release+platform
// fixture with one transformer whose `output` echoes back the injected #context,
// then confirms the rendered value carries the binding-built fields. The test
// does not pin byte-stable serialised output — that would force a CUE→JSON
// encode step the renderer does not perform — but it does verify the new
// binding.BuildTransformerContext path produces the expected shape.
func TestCompileModuleRelease_RendersContextViaBinding(t *testing.T) {
	ctx := cuecontext.New()

	// Release spec carries a single component declaring an "echo" resource
	// and a tier=web label.
	releaseSpec := ctx.CompileString(`
apiVersion: "ignored-by-test"
kind: "ModuleRelease"
metadata: { name: "demo", namespace: "ns", uuid: "u-rel" }
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
`)
	require.NoError(t, releaseSpec.Err())

	rel := &module.Release{
		APIVersion: apiversion.V1alpha2,
		Metadata: &module.ReleaseMetadata{
			Name: "demo", Namespace: "ns", UUID: "u-rel",
			Labels: map[string]string{"k": "v"},
		},
		Package: releaseSpec,
	}

	// Platform with one transformer matching tier=web and indexed against the
	// echo resource FQN. #transform's output is a single-element list whose
	// only entry echoes #context.#runtimeName, providing a probe for the
	// binding's context injection.
	pv := ctx.CompileString(`
apiVersion: "ignored-by-test"
kind: "Platform"
metadata: { name: "k8s" }
type: "kubernetes"
#registry: {}
#knownResources: {}
#knownTraits: {}
#composedTransformers: {
	"example.com/p/echo@v0": {
		requiredLabels: { tier: "web" }
		requiredResources: {}
		requiredTraits: {}
		optionalTraits: {}
		#transform: {
			#component: _
			#context:   _
			output: [{
				kind: "echo"
				runtime: #context.#runtimeName
				release: #context.#moduleReleaseMetadata.name
				component: #context.#componentMetadata.name
			}]
		}
	}
}
#matchers: {
	resources: {
		"example.com/r/echo@v0": ["example.com/p/echo@v0"]
	}
	traits: {}
}
`)
	require.NoError(t, pv.Err())
	plat := &platform.Platform{
		APIVersion: apiversion.V1alpha2,
		Metadata:   &platform.PlatformMetadata{Name: "k8s", Type: "kubernetes"},
		Package:    pv,
	}

	out, err := compile.CompileModuleRelease(context.Background(), rel, plat, "opm-cli") //nolint:staticcheck // SA1019: testing the deprecated free function
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Len(t, out.Compiled, 1, "expected one compiled item")

	got := out.Compiled[0].Value
	runtime, err := got.LookupPath(cue.ParsePath("runtime")).String()
	require.NoError(t, err)
	assert.Equal(t, "opm-cli", runtime, "binding-built #runtimeName should reach the compiled output")
	release, err := got.LookupPath(cue.ParsePath("release")).String()
	require.NoError(t, err)
	assert.Equal(t, "demo", release)
	component, err := got.LookupPath(cue.ParsePath("component")).String()
	require.NoError(t, err)
	assert.Equal(t, "web", component)
}
