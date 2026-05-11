package compile_test

import (
	"context"
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

// Coverage for the version-mismatch and unknown-binding error paths
// previously lived here (testing compile.CompileModuleRelease). Those
// scenarios are now exercised against the canonical Kernel.Compile entry
// point in pkg/kernel/kernel_test.go (TestKernel_Compile_Parity_*).

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
		metadata: { fqn: "opmodel.dev/p/k8s/x@v0" }
		requiredLabels: { tier: "web" }
		requiredResources: { "opmodel.dev/r/echo@v0": {} }
		requiredTraits: {}
		optionalTraits: {}
	}
}
#matchers: {
	resources: {
		"opmodel.dev/r/echo@v0": [#composedTransformers["opmodel.dev/p/k8s/x@v0"]]
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

// TestMatch_MultiCandidateDisambiguatedByLabels exercises the predicate-bucket
// matcher: two transformers compete for the same resource FQN but are gated by
// different requiredLabels. Each component carries one of those labels and must
// match exactly one transformer.
func TestMatch_MultiCandidateDisambiguatedByLabels(t *testing.T) {
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
	"example.com/p/deployment@v0": {
		metadata: { fqn: "example.com/p/deployment@v0" }
		requiredLabels: { "workload-type": "stateless" }
		requiredResources: { "example.com/r/container@v0": {} }
		requiredTraits: {}
		optionalTraits: {}
	}
	"example.com/p/statefulset@v0": {
		metadata: { fqn: "example.com/p/statefulset@v0" }
		requiredLabels: { "workload-type": "stateful" }
		requiredResources: { "example.com/r/container@v0": {} }
		requiredTraits: {}
		optionalTraits: {}
	}
}
#matchers: {
	resources: {
		"example.com/r/container@v0": [
			#composedTransformers["example.com/p/deployment@v0"],
			#composedTransformers["example.com/p/statefulset@v0"],
		]
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
	metadata: { labels: { "workload-type": "stateless" } }
	#resources: { "example.com/r/container@v0": {} }
}
"db": {
	metadata: { labels: { "workload-type": "stateful" } }
	#resources: { "example.com/r/container@v0": {} }
}
`)
	require.NoError(t, components.Err())

	b, err := api.Lookup(apiversion.V1alpha2)
	require.NoError(t, err)

	plan, err := compile.Match(components, plat, b)
	require.NoError(t, err)
	require.NotNil(t, plan)

	assert.Empty(t, plan.Unmatched, "every component should match exactly one transformer")

	pairs := plan.MatchedPairs()
	require.Len(t, pairs, 2)
	got := map[string]string{}
	for _, p := range pairs {
		got[p.ComponentName] = p.TransformerFQN
	}
	assert.Equal(t, "example.com/p/statefulset@v0", got["db"])
	assert.Equal(t, "example.com/p/deployment@v0", got["web"])
}

// TestMatch_TwoTransformersPairBoth verifies that when two transformers
// share a required resource FQN and the component satisfies both predicates,
// the matcher pairs both. (Same-transformer-FQN collisions are caught
// upstream by CUE map unification on #composedTransformers; this test
// covers the legitimate multi-candidate case the schema now allows.)
func TestMatch_TwoTransformersPairBoth(t *testing.T) {
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
		metadata: { fqn: "example.com/p/a@v0" }
		requiredLabels: {}
		requiredResources: { "example.com/r/echo@v0": {} }
		requiredTraits: {}
		optionalTraits: {}
	}
	"example.com/p/b@v0": {
		metadata: { fqn: "example.com/p/b@v0" }
		requiredLabels: {}
		requiredResources: { "example.com/r/echo@v0": {} }
		requiredTraits: {}
		optionalTraits: {}
	}
}
#matchers: {
	resources: {
		"example.com/r/echo@v0": [
			#composedTransformers["example.com/p/a@v0"],
			#composedTransformers["example.com/p/b@v0"],
		]
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

	assert.Empty(t, plan.Unmatched, "component matched at least one transformer")

	got := map[string]struct{}{}
	for _, p := range plan.MatchedPairs() {
		if p.ComponentName == "web" {
			got[p.TransformerFQN] = struct{}{}
		}
	}
	assert.Contains(t, got, "example.com/p/a@v0",
		"transformer a should pair with web")
	assert.Contains(t, got, "example.com/p/b@v0",
		"transformer b should pair with web")
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
	plat := &platform.Platform{
		APIVersion: apiversion.V1alpha2,
		Metadata:   &platform.PlatformMetadata{Name: "k8s", Type: "kubernetes"},
		Package:    pv,
	}

	binding, err := api.Lookup(rel.APIVersion)
	require.NoError(t, err)
	schemaComponents := rel.MatchComponents()
	require.True(t, schemaComponents.Exists())
	dataComponents, err := compile.FinalizeValue(plat.Package.Context(), schemaComponents)
	require.NoError(t, err)
	plan, err := compile.Match(schemaComponents, plat, binding)
	require.NoError(t, err)
	out, err := compile.NewModule(plat, "opm-cli").Execute(context.Background(), rel, schemaComponents, dataComponents, plan) //nolint:staticcheck // SA1019: compile.NewModule is on its own deprecation arc, unrelated to this test
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
