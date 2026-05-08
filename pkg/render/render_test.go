package render_test

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
	"github.com/open-platform-model/library/pkg/module"
	"github.com/open-platform-model/library/pkg/provider"
	"github.com/open-platform-model/library/pkg/render"
)

// minimalRelease constructs a *module.Release with the given apiVersion. The
// CUE Spec is a synthesised value sufficient for ProcessModuleRelease's early
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

// minimalProvider constructs a *provider.Provider with the given apiVersion.
func minimalProvider(t *testing.T, ver apiversion.Version) *provider.Provider {
	t.Helper()
	ctx := cuecontext.New()
	pv := ctx.CompileString(`
apiVersion: "ignored-by-test"
kind: "Provider"
metadata: { name: "kubernetes", version: "v0" }
#transformers: {}
`)
	require.NoError(t, pv.Err())
	return &provider.Provider{
		APIVersion: ver,
		Metadata:   &provider.ProviderMetadata{Name: "kubernetes"},
		Data:       pv,
	}
}

func TestProcessModuleRelease_VersionMismatch(t *testing.T) {
	rel := minimalRelease(t, apiversion.V1alpha2)
	p := minimalProvider(t, apiversion.Version("opmodel.dev/v1alpha-other"))

	_, err := render.ProcessModuleRelease(context.Background(), rel, p, "opm-cli")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "apiVersion mismatch")
}

func TestProcessModuleRelease_NoBindingRegistered(t *testing.T) {
	// Both release and provider declare the same unrecognised version. Mismatch
	// check passes; api.Lookup then fails.
	unknown := apiversion.Version("opmodel.dev/never-registered")
	rel := minimalRelease(t, unknown)
	p := minimalProvider(t, unknown)

	_, err := render.ProcessModuleRelease(context.Background(), rel, p, "opm-cli")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apiversion.ErrUnknownAPIVersion), "want ErrUnknownAPIVersion in chain, got %v", err)
}

func TestMatch_RequiresBinding(t *testing.T) {
	ctx := cuecontext.New()
	components := ctx.CompileString(`{}`)
	require.NoError(t, components.Err())
	p := minimalProvider(t, apiversion.V1alpha2)

	_, err := render.Match(components, p, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "binding is required")
}

func TestMatch_UsesBindingPaths(t *testing.T) {
	// Construct a provider with a #transformers field requiring a label that
	// the component carries. Match should mark the pair as matched, proving the
	// path lookups go through the binding rather than crashing on an absent
	// transformers field.
	ctx := cuecontext.New()
	pv := ctx.CompileString(`
#transformers: {
	"opmodel.dev/p/k8s/x@v0": {
		requiredLabels: { tier: "web" }
		requiredResources: {}
		requiredTraits: {}
		optionalTraits: {}
	}
}
`)
	require.NoError(t, pv.Err())
	p := &provider.Provider{
		APIVersion: apiversion.V1alpha2,
		Metadata:   &provider.ProviderMetadata{Name: "k8s"},
		Data:       pv,
	}
	components := ctx.CompileString(`
"web": {
	metadata: { labels: { tier: "web" } }
}
`)
	require.NoError(t, components.Err())

	b, err := api.Lookup(apiversion.V1alpha2)
	require.NoError(t, err)

	plan, err := render.Match(components, p, b)
	require.NoError(t, err)
	require.NotNil(t, plan)
	pairs := plan.MatchedPairs()
	require.Len(t, pairs, 1)
	assert.Equal(t, "web", pairs[0].ComponentName)
	assert.Equal(t, "opmodel.dev/p/k8s/x@v0", pairs[0].TransformerFQN)
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

// TestProcessModuleRelease_RendersContextViaBinding is a coarse snapshot test
// for the group-7 refactor. It builds a minimal release+provider fixture with
// one transformer whose `output` echoes back the injected #context, then
// confirms the rendered value carries the binding-built fields. The test does
// not pin byte-stable serialised output — that would force a CUE→JSON encode
// step the renderer does not perform — but it does verify the new
// binding.BuildTransformerContext path produces the same shape the legacy
// injectContext used to.
func TestProcessModuleRelease_RendersContextViaBinding(t *testing.T) {
	ctx := cuecontext.New()

	// Module: one component with a label and a resource definition.
	moduleSpec := ctx.CompileString(`
apiVersion: "ignored-by-test"
kind: "Module"
metadata: {
	name: "demo-mod"
	modulePath: "example.com/m"
	version: "1.0.0"
	fqn: "example.com/m/demo-mod:1.0.0"
	uuid: "11111111-1111-1111-1111-111111111111"
}
`)
	require.NoError(t, moduleSpec.Err())

	// Release spec carries a single component with a tier=web label.
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

	// Provider with one transformer matching tier=web. #transform's output is
	// a single-element list whose only entry echoes #context.#runtimeName,
	// providing a probe for the binding's context injection.
	pv := ctx.CompileString(`
apiVersion: "ignored-by-test"
kind: "Provider"
metadata: { name: "k8s", version: "v0" }
#transformers: {
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
`)
	require.NoError(t, pv.Err())
	p := &provider.Provider{
		APIVersion: apiversion.V1alpha2,
		Metadata:   &provider.ProviderMetadata{Name: "k8s"},
		Data:       pv,
	}

	out, err := render.ProcessModuleRelease(context.Background(), rel, p, "opm-cli")
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Len(t, out.Rendered, 1, "expected one rendered item")

	got := out.Rendered[0].Value
	runtime, err := got.LookupPath(cue.ParsePath("runtime")).String()
	require.NoError(t, err)
	assert.Equal(t, "opm-cli", runtime, "binding-built #runtimeName should reach the rendered output")
	release, err := got.LookupPath(cue.ParsePath("release")).String()
	require.NoError(t, err)
	assert.Equal(t, "demo", release)
	component, err := got.LookupPath(cue.ParsePath("component")).String()
	require.NoError(t, err)
	assert.Equal(t, "web", component)
	_ = moduleSpec // moduleSpec is unused in this fixture but kept to document module shape
}
