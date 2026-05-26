package compile_test

import (
	"context"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/compile"
	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/platform"
	"github.com/open-platform-model/library/opm/schema"
)

// minimalPlatform constructs a *platform.Platform with an empty registry /
// matchers / composedTransformers index.
func minimalPlatform(t *testing.T) *platform.Platform {
	t.Helper()
	ctx := cuecontext.New()
	pv := ctx.CompileString(`
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
	return &platform.Platform{
		Metadata: &platform.PlatformMetadata{Name: "kubernetes", Type: "kubernetes"},
		Package:  pv,
	}
}

func TestMatch_RequiresPlatform(t *testing.T) {
	ctx := cuecontext.New()
	components := ctx.CompileString(`{}`)
	require.NoError(t, components.Err())

	_, err := compile.Match(components, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "platform is required")
}

func TestMatch_UsesSchemaPaths(t *testing.T) {
	// Construct a Platform whose #matchers index points a resource FQN at a
	// transformer requiring a label that the component carries. Match should
	// mark the pair as matched, proving the path lookups go through opm/schema
	// rather than crashing on absent paths.
	ctx := cuecontext.New()
	pv := ctx.CompileString(`
kind: "Platform"
metadata: { name: "k8s" }
type: "kubernetes"
#registry: {}
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
		Metadata: &platform.PlatformMetadata{Name: "k8s", Type: "kubernetes"},
		Package:  pv,
	}
	components := ctx.CompileString(`
"web": {
	metadata: { labels: { tier: "web" } }
	#resources: { "opmodel.dev/r/echo@v0": {} }
}
`)
	require.NoError(t, components.Err())

	plan, err := compile.Match(components, plat)
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
kind: "Platform"
metadata: { name: "k8s" }
type: "kubernetes"
#registry: {}
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
		Metadata: &platform.PlatformMetadata{Name: "k8s", Type: "kubernetes"},
		Package:  pv,
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

	plan, err := compile.Match(components, plat)
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
kind: "Platform"
metadata: { name: "k8s" }
type: "kubernetes"
#registry: {}
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
		Metadata: &platform.PlatformMetadata{Name: "k8s", Type: "kubernetes"},
		Package:  pv,
	}
	components := ctx.CompileString(`
"web": {
	metadata: { labels: {} }
	#resources: { "example.com/r/echo@v0": {} }
}
`)
	require.NoError(t, components.Err())

	plan, err := compile.Match(components, plat)
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

// TestReleaseImplementsReleaseView confirms that *module.Release satisfies
// schema.ReleaseView so BuildTransformerContext can call it without an
// adapter.
func TestReleaseImplementsReleaseView(t *testing.T) {
	var _ schema.ReleaseView = (*module.Release)(nil)
}

// TestCompileModuleRelease_RendersContextViaSchema is a coarse snapshot test
// for the schema-driven context injection. It builds a minimal release+platform
// fixture with one transformer whose `output` echoes back the injected #context,
// then confirms the rendered value carries the schema-built fields.
func TestCompileModuleRelease_RendersContextViaSchema(t *testing.T) {
	ctx := cuecontext.New()

	// Release spec carries a single component declaring an "echo" resource
	// and a tier=web label.
	releaseSpec := ctx.CompileString(`
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
		Metadata: &module.ReleaseMetadata{
			Name: "demo", Namespace: "ns", UUID: "u-rel",
			Labels: map[string]string{"k": "v"},
		},
		Package: releaseSpec,
	}

	// Platform with one transformer matching tier=web and indexed against the
	// echo resource FQN. #transform's output echoes #context, providing a probe
	// for the schema's context injection.
	pv := ctx.CompileString(`
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
	plat := &platform.Platform{
		Metadata: &platform.PlatformMetadata{Name: "k8s", Type: "kubernetes"},
		Package:  pv,
	}

	schemaComponents := rel.MatchComponents()
	require.True(t, schemaComponents.Exists())
	dataComponents, err := compile.FinalizeValue(plat.Package.Context(), schemaComponents)
	require.NoError(t, err)
	plan, err := compile.Match(schemaComponents, plat)
	require.NoError(t, err)
	out, err := compile.NewModule(plat, "opm-cli").Execute(context.Background(), rel, schemaComponents, dataComponents, plan) //nolint:staticcheck // SA1019: compile.NewModule is on its own deprecation arc, unrelated to this test
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Len(t, out.Compiled, 1, "expected one compiled item")

	got := out.Compiled[0].Value
	runtime, err := got.LookupPath(cue.ParsePath("runtime")).String()
	require.NoError(t, err)
	assert.Equal(t, "opm-cli", runtime, "schema-built #runtimeName should reach the compiled output")
	release, err := got.LookupPath(cue.ParsePath("release")).String()
	require.NoError(t, err)
	assert.Equal(t, "demo", release)
	component, err := got.LookupPath(cue.ParsePath("component")).String()
	require.NoError(t, err)
	assert.Equal(t, "web", component)
}

// silence unused-import warnings that crop up during refactors
var _ = cue.Value{}
var _ = minimalPlatform
