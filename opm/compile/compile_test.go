package compile_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	cueerrors "cuelang.org/go/cue/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/compile"
	oerrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/materialize"
	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/platform"
	"github.com/open-platform-model/library/opm/schema"
)

// materialized wraps a platform CUE source (already carrying filled
// #composedTransformers / #matchers) into a *materialize.MaterializedPlatform —
// the realized form the matcher now consumes. Unit-level match tests build the
// realized package directly rather than round-tripping through an OCI registry;
// the materialize package's own tests cover the registry resolution path.
func materialized(t *testing.T, ctx *cue.Context, src string) *materialize.MaterializedPlatform {
	t.Helper()
	pv := ctx.CompileString(src)
	require.NoError(t, pv.Err())
	plat := &platform.Platform{
		Metadata: &platform.PlatformMetadata{Name: "k8s", Type: "kubernetes"},
		Package:  pv,
	}
	return &materialize.MaterializedPlatform{Source: plat, Package: pv}
}

func TestMatch_RequiresPlatform(t *testing.T) {
	ctx := cuecontext.New()
	components := ctx.CompileString(`{}`)
	require.NoError(t, components.Err())

	_, err := compile.Match(components, nil, "demo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "materialized platform is required")
}

func TestMatch_UsesSchemaPaths(t *testing.T) {
	// Construct a Platform whose #matchers index points a resource FQN at a
	// transformer requiring a label that the component carries. Match should
	// mark the pair as matched, proving the path lookups go through opm/schema
	// rather than crashing on absent paths.
	ctx := cuecontext.New()
	mp := materialized(t, ctx, `
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
	components := ctx.CompileString(`
"web": {
	metadata: { labels: { tier: "web" } }
	#resources: { "opmodel.dev/r/echo@v0": {} }
}
`)
	require.NoError(t, components.Err())

	plan, err := compile.Match(components, mp, "demo")
	require.NoError(t, err)
	require.NotNil(t, plan)
	pairs := plan.MatchedPairs()
	require.Len(t, pairs, 1)
	assert.Equal(t, "web", pairs[0].ComponentName)
	assert.Equal(t, "opmodel.dev/p/k8s/x@v0", pairs[0].TransformerFQN)
	assert.Empty(t, plan.Missing)
	assert.Empty(t, plan.Unify)
}

// TestMatch_UnifyAgreeingBodiesPairs covers the always-unify rung's happy path:
// the component and transformer carry non-empty primitive bodies that unify
// cleanly, so the pair survives to predicate evaluation and matches.
func TestMatch_UnifyAgreeingBodiesPairs(t *testing.T) {
	ctx := cuecontext.New()
	mp := materialized(t, ctx, `
kind: "Platform"
metadata: { name: "k8s" }
type: "kubernetes"
#registry: {}
#composedTransformers: {
	"example.com/p/c@v0": {
		metadata: { fqn: "example.com/p/c@v0" }
		requiredLabels: {}
		requiredResources: { "example.com/r/container@v0": { image: string } }
		requiredTraits: {}
		optionalTraits: {}
	}
}
#matchers: {
	resources: {
		"example.com/r/container@v0": [#composedTransformers["example.com/p/c@v0"]]
	}
	traits: {}
}
`)
	components := ctx.CompileString(`
"web": {
	metadata: { labels: {} }
	#resources: { "example.com/r/container@v0": { image: "nginx" } }
}
`)
	require.NoError(t, components.Err())

	plan, err := compile.Match(components, mp, "demo")
	require.NoError(t, err)
	require.Empty(t, plan.Unify, "agreeing bodies must not produce a unify error")
	pairs := plan.MatchedPairs()
	require.Len(t, pairs, 1)
	assert.Equal(t, "example.com/p/c@v0", pairs[0].TransformerFQN)
}

// TestMatch_DivergentBodyRecordsUnifyError covers the always-unify rung's
// failure path: the component's primitive body conflicts with the transformer's
// required body at the same FQN. The candidate is disqualified and a UnifyError
// carrying the verbatim CUE cause is recorded.
func TestMatch_DivergentBodyRecordsUnifyError(t *testing.T) {
	ctx := cuecontext.New()
	mp := materialized(t, ctx, `
kind: "Platform"
metadata: { name: "k8s" }
type: "kubernetes"
#registry: {}
#composedTransformers: {
	"example.com/p/c@v0": {
		metadata: { fqn: "example.com/p/c@v0" }
		requiredLabels: {}
		requiredResources: { "example.com/r/container@v0": { image: "redis" } }
		requiredTraits: {}
		optionalTraits: {}
	}
}
#matchers: {
	resources: {
		"example.com/r/container@v0": [#composedTransformers["example.com/p/c@v0"]]
	}
	traits: {}
}
`)
	components := ctx.CompileString(`
"web": {
	metadata: { labels: {} }
	#resources: { "example.com/r/container@v0": { image: "nginx" } }
}
`)
	require.NoError(t, components.Err())

	plan, err := compile.Match(components, mp, "demo")
	require.NoError(t, err)

	require.Len(t, plan.Unify, 1, "divergent body must record exactly one UnifyError")
	assert.Equal(t, "web", plan.Unify[0].Component)
	assert.Equal(t, "example.com/r/container@v0", plan.Unify[0].FQN)

	// The conflicting candidate is not paired; the component is unmatched.
	assert.Empty(t, plan.MatchedPairs())
	assert.Contains(t, plan.Unmatched, "web")

	// The CUE cause is reachable verbatim via errors.As.
	var cueErr cueerrors.Error
	require.True(t, errors.As(plan.Unify[0].Cause, &cueErr),
		"UnifyError.Cause must be walkable to a CUE error")
	assert.Contains(t, plan.Unify[0].Cause.Error(), "conflicting values")
}

// TestMatch_AbsentFQNRecordsMissingWithAlternatives covers the lookup rung: a
// demanded FQN whose #matchers bucket is empty produces a hard MissingFQN, and
// Alternatives surfaces the same-modulePath/name FQN materialized at another
// SemVer.
func TestMatch_AbsentFQNRecordsMissingWithAlternatives(t *testing.T) {
	ctx := cuecontext.New()
	mp := materialized(t, ctx, `
kind: "Platform"
metadata: { name: "k8s" }
type: "kubernetes"
#registry: {}
#composedTransformers: {
	"example.com/p/c@v0": {
		metadata: { fqn: "example.com/p/c@v0" }
		requiredLabels: {}
		requiredResources: { "example.com/r/container@1.1.0": {} }
		requiredTraits: {}
		optionalTraits: {}
	}
}
#matchers: {
	resources: {
		"example.com/r/container@1.1.0": [#composedTransformers["example.com/p/c@v0"]]
	}
	traits: {}
}
`)
	// Component demands the 1.0.0 version, which no transformer requires.
	components := ctx.CompileString(`
"web": {
	metadata: { labels: {} }
	#resources: { "example.com/r/container@1.0.0": {} }
}
`)
	require.NoError(t, components.Err())

	plan, err := compile.Match(components, mp, "demo")
	require.NoError(t, err)

	require.Len(t, plan.Missing, 1, "absent FQN must record exactly one MissingFQN")
	miss := plan.Missing[0]
	assert.Equal(t, "demo", miss.Release)
	assert.Equal(t, "web", miss.Component)
	assert.Equal(t, "example.com/r/container@1.0.0", miss.FQN)
	assert.Equal(t, []string{"example.com/r/container@1.1.0"}, miss.Alternatives)

	assert.Contains(t, plan.Unmatched, "web")
}

// TestMatch_SecondRequiredPrimitiveDivergesRejects covers D1: a transformer
// requires two primitives; the component agrees on the first and conflicts on
// the second. The unify rung walks the full intersection (not just the
// triggering FQN), so the conflict on the second primitive is caught, a single
// UnifyError is recorded for it, and the candidate is rejected.
func TestMatch_SecondRequiredPrimitiveDivergesRejects(t *testing.T) {
	ctx := cuecontext.New()
	mp := materialized(t, ctx, `
kind: "Platform"
metadata: { name: "k8s" }
type: "kubernetes"
#registry: {}
#composedTransformers: {
	"example.com/p/c@v0": {
		metadata: { fqn: "example.com/p/c@v0" }
		requiredLabels: {}
		requiredResources: {
			"example.com/r/container@v0": { image: string }
			"example.com/r/volume@v0":    { size: "10Gi" }
		}
		requiredTraits: {}
		optionalTraits: {}
	}
}
#matchers: {
	resources: {
		"example.com/r/container@v0": [#composedTransformers["example.com/p/c@v0"]]
		"example.com/r/volume@v0":    [#composedTransformers["example.com/p/c@v0"]]
	}
	traits: {}
}
`)
	// Agrees on container (image: string ∋ "nginx"), conflicts on volume (size).
	components := ctx.CompileString(`
"web": {
	metadata: { labels: {} }
	#resources: {
		"example.com/r/container@v0": { image: "nginx" }
		"example.com/r/volume@v0":    { size: "20Gi" }
	}
}
`)
	require.NoError(t, components.Err())

	plan, err := compile.Match(components, mp, "demo")
	require.NoError(t, err)

	// Exactly one UnifyError — for the conflicting second primitive — and no
	// duplicate even though the candidate sits in two buckets.
	require.Len(t, plan.Unify, 1, "only the divergent primitive records a UnifyError, once")
	assert.Equal(t, "web", plan.Unify[0].Component)
	assert.Equal(t, "example.com/r/volume@v0", plan.Unify[0].FQN)

	// The candidate is rejected; the component is unmatched.
	assert.Empty(t, plan.MatchedPairs())
	assert.Contains(t, plan.Unmatched, "web")
}

// TestMatch_MultipleMissesAccumulated covers the one-pass, no-fail-fast
// accumulation: two components each demand a distinct absent FQN, so the plan
// carries two MissingFQN entries (one per (release, component, fqn)).
func TestMatch_MultipleMissesAccumulated(t *testing.T) {
	ctx := cuecontext.New()
	mp := materialized(t, ctx, `
kind: "Platform"
metadata: { name: "k8s" }
type: "kubernetes"
#registry: {}
#composedTransformers: {}
#matchers: { resources: {}, traits: {} }
`)
	components := ctx.CompileString(`
"web": {
	metadata: { labels: {} }
	#resources: { "example.com/r/a@v0": {} }
}
"db": {
	metadata: { labels: {} }
	#resources: { "example.com/r/b@v0": {} }
}
`)
	require.NoError(t, components.Err())

	plan, err := compile.Match(components, mp, "demo")
	require.NoError(t, err)

	require.Len(t, plan.Missing, 2, "one MissingFQN per (component, fqn), accumulated in one pass")
	byComp := map[string]string{}
	for _, m := range plan.Missing {
		assert.Equal(t, "demo", m.Release)
		byComp[m.Component] = m.FQN
	}
	assert.Equal(t, "example.com/r/a@v0", byComp["web"])
	assert.Equal(t, "example.com/r/b@v0", byComp["db"])
	assert.ElementsMatch(t, []string{"web", "db"}, plan.Unmatched)
}

// TestMatch_MultiCandidateDisambiguatedByLabels exercises the predicate-bucket
// matcher: two transformers compete for the same resource FQN but are gated by
// different requiredLabels. Each component carries one of those labels and must
// match exactly one transformer — unchanged behavior under the unify rung (D17).
func TestMatch_MultiCandidateDisambiguatedByLabels(t *testing.T) {
	ctx := cuecontext.New()
	mp := materialized(t, ctx, `
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

	plan, err := compile.Match(components, mp, "demo")
	require.NoError(t, err)
	require.NotNil(t, plan)

	assert.Empty(t, plan.Unmatched, "every component should match exactly one transformer")
	assert.Empty(t, plan.Unify, "shared-FQN bodies agree; no unify error expected")

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
	mp := materialized(t, ctx, `
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
	components := ctx.CompileString(`
"web": {
	metadata: { labels: {} }
	#resources: { "example.com/r/echo@v0": {} }
}
`)
	require.NoError(t, components.Err())

	plan, err := compile.Match(components, mp, "demo")
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
	mp := materialized(t, ctx, `
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

	schemaComponents := rel.MatchComponents()
	require.True(t, schemaComponents.Exists())
	dataComponents, err := compile.FinalizeValue(mp.Package.Context(), schemaComponents)
	require.NoError(t, err)
	plan, err := compile.Match(schemaComponents, mp, rel.Metadata.Name)
	require.NoError(t, err)
	out, err := compile.NewModule(ctx, mp, "opm-cli").Execute(context.Background(), rel, schemaComponents, dataComponents, plan) //nolint:staticcheck // SA1019: compile.NewModule is on its own deprecation arc, unrelated to this test
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

// ---------------------------------------------------------------------------
// Execute-path tests.
//
// The match phase is covered above. The tests below exercise compile/execute.go
// and compile/module.go: output-kind dispatch (struct vs list vs unexpected),
// the per-pair error branches, context cancellation, the nil guards,
// UnmatchedComponentsError, and warning propagation. They drive the same entry
// the kernel uses (compile.NewModule(mp, runtime).Execute), so behaviour stays
// pinned to the real compile pipeline.
// ---------------------------------------------------------------------------

const (
	echoResFQN = "example.com/r/echo@v0"
	echoTfFQN  = "example.com/p/echo@v0"
)

// releaseWithComponents builds a minimal *module.Release whose `components`
// field is the given CUE struct source. Metadata is fixed; only the component
// shape varies between tests.
func releaseWithComponents(t *testing.T, ctx *cue.Context, componentsSrc string) *module.Release {
	t.Helper()
	spec := ctx.CompileString(`
kind: "ModuleRelease"
metadata: { name: "demo", namespace: "ns", uuid: "u-rel" }
components: ` + componentsSrc)
	require.NoError(t, spec.Err())
	return &module.Release{
		Metadata: &module.ReleaseMetadata{
			Name: "demo", Namespace: "ns", UUID: "u-rel",
			Labels: map[string]string{},
		},
		Package: spec,
	}
}

// echoRelease is the common single-component release: "web" declaring the echo
// resource and no labels.
func echoRelease(t *testing.T, ctx *cue.Context) *module.Release {
	t.Helper()
	return releaseWithComponents(t, ctx, `{
	web: {
		metadata: { name: "web", labels: {} }
		#resources: { "`+echoResFQN+`": {} }
	}
}`)
}

// echoPlatform builds a materialized platform with one transformer matched to
// the echo resource. transformField is spliced into the composed entry verbatim
// (e.g. `#transform: { ... }`); pass "" to omit #transform entirely and
// exercise the not-found branch.
func echoPlatform(t *testing.T, ctx *cue.Context, transformField string) *materialize.MaterializedPlatform {
	t.Helper()
	return materialized(t, ctx, `
kind: "Platform"
metadata: { name: "k8s" }
type: "kubernetes"
#registry: {}
#composedTransformers: {
	"`+echoTfFQN+`": {
		metadata: { fqn: "`+echoTfFQN+`" }
		requiredLabels: {}
		requiredResources: { "`+echoResFQN+`": {} }
		requiredTraits: {}
		optionalTraits: {}
		`+transformField+`
	}
}
#matchers: {
	resources: {
		"`+echoResFQN+`": [#composedTransformers["`+echoTfFQN+`"]]
	}
	traits: {}
}
`)
}

// runExecute drives MatchComponents → Finalize → Match → Execute against the
// given platform and release using a background context.
func runExecute(t *testing.T, mp *materialize.MaterializedPlatform, rel *module.Release) (*compile.CompileResult, error) {
	t.Helper()
	return runExecuteCtx(t, context.Background(), mp, rel)
}

func runExecuteCtx(t *testing.T, ctx context.Context, mp *materialize.MaterializedPlatform, rel *module.Release) (*compile.CompileResult, error) {
	t.Helper()
	sc := rel.MatchComponents()
	require.True(t, sc.Exists())
	dc, err := compile.FinalizeValue(mp.Package.Context(), sc)
	require.NoError(t, err)
	plan, err := compile.Match(sc, mp, rel.Metadata.Name)
	require.NoError(t, err)
	//nolint:staticcheck // SA1019: NewModule is on its own deprecation arc; this mirrors kernel/compile.go.
	return compile.NewModule(mp.Package.Context(), mp, "opm-cli").Execute(ctx, rel, sc, dc, plan)
}

// TestExecute_ListOutputEmitsOnePerItem covers the ListKind dispatch: a list
// output yields one *core.Compiled per element, each carrying identical
// provenance and a distinct value.
func TestExecute_ListOutputEmitsOnePerItem(t *testing.T) {
	ctx := cuecontext.New()
	mp := echoPlatform(t, ctx, `#transform: { #component: _, #context: _, output: [ {n: 1}, {n: 2}, {n: 3} ] }`)

	out, err := runExecute(t, mp, echoRelease(t, ctx))
	require.NoError(t, err)
	require.Len(t, out.Compiled, 3, "one Compiled per list element")

	seen := map[int64]bool{}
	for _, c := range out.Compiled {
		assert.Equal(t, "web", c.Component)
		assert.Equal(t, echoTfFQN, c.Transformer)
		assert.Equal(t, "demo", c.Release)
		n, err := c.Value.LookupPath(cue.ParsePath("n")).Int64()
		require.NoError(t, err)
		seen[n] = true
	}
	assert.Equal(t, map[int64]bool{1: true, 2: true, 3: true}, seen, "each element preserved")
}

// TestExecute_EmptyListEmitsZero covers an empty list output: zero Compiled, no
// error.
func TestExecute_EmptyListEmitsZero(t *testing.T) {
	ctx := cuecontext.New()
	mp := echoPlatform(t, ctx, `#transform: { #component: _, #context: _, output: [] }`)

	out, err := runExecute(t, mp, echoRelease(t, ctx))
	require.NoError(t, err)
	assert.Empty(t, out.Compiled)
}

// TestExecute_UnexpectedOutputKindErrors covers the default dispatch arm: a
// scalar output is neither struct nor list and must error.
func TestExecute_UnexpectedOutputKindErrors(t *testing.T) {
	ctx := cuecontext.New()
	mp := echoPlatform(t, ctx, `#transform: { #component: _, #context: _, output: "not-a-resource" }`)

	_, err := runExecute(t, mp, echoRelease(t, ctx))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected output kind")
}

// TestExecute_AbsentOutputFieldYieldsNoCompiledNoError covers a #transform with
// no output field: the pair contributes nothing but is not an error.
func TestExecute_AbsentOutputFieldYieldsNoCompiledNoError(t *testing.T) {
	ctx := cuecontext.New()
	mp := echoPlatform(t, ctx, `#transform: { #component: _, #context: _ }`)

	out, err := runExecute(t, mp, echoRelease(t, ctx))
	require.NoError(t, err)
	assert.Empty(t, out.Compiled)
}

// TestExecute_FillComponentConflictErrors covers the FillPath(#component) error
// branch: the transform unifies the injected #component with a conflicting
// scalar, so unification fails the moment the data component is filled in.
//
// Note: this is also the closest reachable proxy for the later
// outputVal.Err() branch (execute.go evaluating-output) — any structural bottom
// in the transform value surfaces at the first FillPath().Err() check, so the
// output-evaluation branch is effectively shadowed by this one.
func TestExecute_FillComponentConflictErrors(t *testing.T) {
	ctx := cuecontext.New()
	mp := echoPlatform(t, ctx, `#transform: { #component: _, #context: _, output: #component.metadata.name & 123 }`)

	_, err := runExecute(t, mp, echoRelease(t, ctx))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "filling #component")
}

// TestExecute_TransformNotFoundInComposed covers the missing-#transform branch:
// the transformer is indexed and pairs at match time but carries no #transform,
// so execution cannot find it.
func TestExecute_TransformNotFoundInComposed(t *testing.T) {
	ctx := cuecontext.New()
	mp := echoPlatform(t, ctx, "") // composed entry without #transform

	_, err := runExecute(t, mp, echoRelease(t, ctx))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "#transform not found")
}

// TestExecute_ComponentMissingInDataComponents covers the branch where a matched
// component is absent from the finalized data components passed to Execute.
func TestExecute_ComponentMissingInDataComponents(t *testing.T) {
	ctx := cuecontext.New()
	mp := echoPlatform(t, ctx, `#transform: { #component: _, #context: _, output: { ok: true } }`)
	rel := echoRelease(t, ctx)

	sc := rel.MatchComponents()
	plan, err := compile.Match(sc, mp, rel.Metadata.Name)
	require.NoError(t, err)

	// Deliberately pass empty data components — the plan still references "web".
	emptyDC := ctx.CompileString(`{}`)
	require.NoError(t, emptyDC.Err())

	//nolint:staticcheck // SA1019: NewModule deprecation arc, see runExecuteCtx.
	_, err = compile.NewModule(ctx, mp, "opm-cli").Execute(context.Background(), rel, sc, emptyDC, plan)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found in data components value")
}

// TestExecute_PerPairErrorsAccumulated covers executeTransforms collecting
// errors across pairs rather than failing fast: two components each match a
// transformer that lacks #transform, and both failures surface.
func TestExecute_PerPairErrorsAccumulated(t *testing.T) {
	ctx := cuecontext.New()
	mp := materialized(t, ctx, `
kind: "Platform"
metadata: { name: "k8s" }
type: "kubernetes"
#registry: {}
#composedTransformers: {
	"example.com/p/ta@v0": {
		metadata: { fqn: "example.com/p/ta@v0" }
		requiredLabels: {}
		requiredResources: { "example.com/r/ra@v0": {} }
		requiredTraits: {}
		optionalTraits: {}
	}
	"example.com/p/tb@v0": {
		metadata: { fqn: "example.com/p/tb@v0" }
		requiredLabels: {}
		requiredResources: { "example.com/r/rb@v0": {} }
		requiredTraits: {}
		optionalTraits: {}
	}
}
#matchers: {
	resources: {
		"example.com/r/ra@v0": [#composedTransformers["example.com/p/ta@v0"]]
		"example.com/r/rb@v0": [#composedTransformers["example.com/p/tb@v0"]]
	}
	traits: {}
}
`)
	rel := releaseWithComponents(t, ctx, `{
	a: { metadata: { name: "a", labels: {} }, #resources: { "example.com/r/ra@v0": {} } }
	b: { metadata: { name: "b", labels: {} }, #resources: { "example.com/r/rb@v0": {} } }
}`)

	_, err := runExecute(t, mp, rel)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"a"`, "first pair failure surfaced")
	assert.Contains(t, err.Error(), `"b"`, "second pair failure surfaced (no fail-fast)")
}

// TestExecute_ContextCancellationStops covers the ctx.Done() guard in
// executeTransforms: a cancelled context aborts execution with context.Canceled.
func TestExecute_ContextCancellationStops(t *testing.T) {
	ctx := cuecontext.New()
	mp := echoPlatform(t, ctx, `#transform: { #component: _, #context: _, output: { ok: true } }`)

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := runExecuteCtx(t, cancelled, mp, echoRelease(t, ctx))
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

// TestExecute_UnmatchedComponentsError covers module.Execute rejecting a plan
// with unmatched components, exposing per-component *oerrors.TransformError via
// errors.As.
func TestExecute_UnmatchedComponentsError(t *testing.T) {
	ctx := cuecontext.New()
	mp := echoPlatform(t, ctx, `#transform: { #component: _, #context: _, output: { ok: true } }`)
	// Component demands a resource the platform does not index → unmatched.
	rel := releaseWithComponents(t, ctx, `{
	web: { metadata: { name: "web", labels: {} }, #resources: { "example.com/r/missing@v0": {} } }
}`)

	_, err := runExecute(t, mp, rel)
	require.Error(t, err)

	var uce *compile.UnmatchedComponentsError
	require.ErrorAs(t, err, &uce)
	assert.Contains(t, uce.Components, "web")

	var te *oerrors.TransformError
	require.ErrorAs(t, err, &te, "per-component TransformError reachable via errors.As")
}

// TestExecute_NilGuards covers the three nil guards at the top of module.Execute.
func TestExecute_NilGuards(t *testing.T) {
	ctx := cuecontext.New()
	mp := echoPlatform(t, ctx, `#transform: { #component: _, #context: _, output: { ok: true } }`)
	rel := echoRelease(t, ctx)
	sc := rel.MatchComponents()
	dc, err := compile.FinalizeValue(mp.Package.Context(), sc)
	require.NoError(t, err)
	plan, err := compile.Match(sc, mp, rel.Metadata.Name)
	require.NoError(t, err)

	t.Run("nil release", func(t *testing.T) {
		//nolint:staticcheck // SA1019: NewModule deprecation arc.
		_, err := compile.NewModule(ctx, mp, "opm-cli").Execute(context.Background(), nil, sc, dc, plan)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "release is required")
	})
	t.Run("nil platform", func(t *testing.T) {
		//nolint:staticcheck // SA1019: NewModule deprecation arc.
		_, err := compile.NewModule(ctx, nil, "opm-cli").Execute(context.Background(), rel, sc, dc, plan)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "platform is required")
	})
	t.Run("nil plan", func(t *testing.T) {
		//nolint:staticcheck // SA1019: NewModule deprecation arc.
		_, err := compile.NewModule(ctx, mp, "opm-cli").Execute(context.Background(), rel, sc, dc, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "match plan is required")
	})
}

// TestExecute_WarningsPropagated covers warning aggregation: a component matches
// one transformer via its resource but also declares a trait whose only
// candidate transformer fails its label predicate, so the trait is unhandled and
// surfaces as a warning (without failing the compile).
func TestExecute_WarningsPropagated(t *testing.T) {
	ctx := cuecontext.New()
	const extraTraitFQN = "example.com/tr/extra@v0"
	mp := materialized(t, ctx, `
kind: "Platform"
metadata: { name: "k8s" }
type: "kubernetes"
#registry: {}
#composedTransformers: {
	"`+echoTfFQN+`": {
		metadata: { fqn: "`+echoTfFQN+`" }
		requiredLabels: {}
		requiredResources: { "`+echoResFQN+`": {} }
		requiredTraits: {}
		optionalTraits: {}
		#transform: { #component: _, #context: _, output: { ok: true } }
	}
	"example.com/p/needslabel@v0": {
		metadata: { fqn: "example.com/p/needslabel@v0" }
		requiredLabels: { need: "yes" }
		requiredResources: {}
		requiredTraits: { "`+extraTraitFQN+`": {} }
		optionalTraits: {}
		#transform: { #component: _, #context: _, output: {} }
	}
}
#matchers: {
	resources: {
		"`+echoResFQN+`": [#composedTransformers["`+echoTfFQN+`"]]
	}
	traits: {
		"`+extraTraitFQN+`": [#composedTransformers["example.com/p/needslabel@v0"]]
	}
}
`)
	// "web" carries the echo resource (matches) and the extra trait (its only
	// candidate transformer needs label need:"yes", which web lacks).
	rel := releaseWithComponents(t, ctx, `{
	web: {
		metadata: { name: "web", labels: {} }
		#resources: { "`+echoResFQN+`": {} }
		#traits: { "`+extraTraitFQN+`": {} }
	}
}`)

	out, err := runExecute(t, mp, rel)
	require.NoError(t, err, "unhandled trait is a warning, not a failure")
	require.Len(t, out.Compiled, 1, "the echo transformer still fired")
	require.NotEmpty(t, out.Warnings, "unhandled trait recorded as a warning")
	joined := strings.Join(out.Warnings, "\n")
	assert.Contains(t, joined, extraTraitFQN)
}
