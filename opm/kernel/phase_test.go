package kernel_test

import (
	"context"
	"reflect"
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/materialize"
	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/platform"
)

// phaseFixture builds a minimal Module + Instance + Platform with a single
// component declaring one resource FQN, and one transformer in the
// platform's #composedTransformers / #matchers index that fulfills it.
// The transformer's output echoes #context fields so tests can confirm the
// full pipeline ran.
type phaseFixture struct {
	mod  *module.Module
	inst *module.Instance
	plat *materialize.MaterializedPlatform
}

func newPhaseFixture(t *testing.T, k *kernel.Kernel) phaseFixture {
	t.Helper()
	ctx := k.CueContext()

	modPkg := ctx.CompileString(`
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
`)
	require.NoError(t, modPkg.Err())

	relPkg := ctx.CompileString(`
kind: "ModuleInstance"
metadata: { name: "demo", namespace: "ns", uuid: "u-inst" }
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
		matchLabels: { tier: "web" }
		#resources: {
			"example.com/r/echo@v0": {}
		}
	}
}
`)
	require.NoError(t, relPkg.Err())

	platVal := ctx.CompileString(`
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
				instance: #context.#moduleInstanceMetadata.name
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
	require.NoError(t, platVal.Err())

	return phaseFixture{
		mod: &module.Module{
			Metadata: &module.ModuleMetadata{
				Name:       "demo-mod",
				ModulePath: "example.com/m",
				Version:    "1.0.0",
				FQN:        "example.com/m/demo-mod:1.0.0",
				UUID:       "11111111-1111-1111-1111-111111111111",
			},
			Package: modPkg,
		},
		inst: &module.Instance{
			Metadata: &module.InstanceMetadata{
				Name: "demo", Namespace: "ns", UUID: "u-inst",
			},
			Package: relPkg,
		},
		plat: &materialize.MaterializedPlatform{
			Source: &platform.Platform{
				Metadata: &platform.PlatformMetadata{Name: "k8s", Type: "kubernetes"},
				Package:  platVal,
			},
			Transformers: platVal.LookupPath(cue.ParsePath("#composedTransformers")),
			Matchers:     platVal.LookupPath(cue.ParsePath("#matchers")),
		},
	}
}

func TestKernel_Match_OK(t *testing.T) {
	k := kernel.New()
	f := newPhaseFixture(t, k)

	plan, err := k.Match(context.Background(), kernel.MatchInput{
		ModuleInstance: f.inst, Platform: f.plat,
	})
	require.NoError(t, err)
	require.NotNil(t, plan)
	pairs := plan.MatchedPairs()
	require.Len(t, pairs, 1)
	assert.Equal(t, "web", pairs[0].ComponentName)
	assert.Equal(t, "example.com/p/echo@v0", pairs[0].TransformerFQN)
}

func TestKernel_Compile_OK(t *testing.T) {
	k := kernel.New()
	f := newPhaseFixture(t, k)

	out, err := k.Compile(context.Background(), kernel.CompileInput{
		ModuleInstance: f.inst, Platform: f.plat, RuntimeName: "opm-cli",
	})
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Len(t, out.Compiled, 1)

	got := out.Compiled[0].Value
	runtime, err := got.LookupPath(cue.ParsePath("runtime")).String()
	require.NoError(t, err)
	assert.Equal(t, "opm-cli", runtime)

	// The compile pipeline builds every value in the caller Kernel's own
	// *cue.Context (k.cueCtx), consuming the materialized platform as read-only
	// input. The v0.16-era observable contract for this — asserting the rendered
	// value's context IS k.CueContext() via cue.Value.Context() — has been
	// retired: under CUE v0.17 that method is deprecated and its result is
	// undefined precisely because values from different contexts may now be
	// combined freely (the property the concurrent-render model relies on), so
	// context identity is no longer a meaningful invariant. Cross-Kernel context
	// behavior is instead verified by the concurrent -race regression test
	// deferred to the v0.17 follow-up (see openspec change
	// concurrent-render-recontract, design.md § Deferred).
}

// TestKernel_Compile_FromInstanceOnly is a regression test for the slim-input
// contract: Compile must succeed when given an instance whose Package carries an
// embedded #module reference, with no separate *module.Module supplied.
func TestKernel_Compile_FromInstanceOnly(t *testing.T) {
	k := kernel.New()
	f := newPhaseFixture(t, k)

	out, err := k.Compile(context.Background(), kernel.CompileInput{
		ModuleInstance: f.inst,
		Platform:       f.plat,
		RuntimeName:    "opm-cli",
	})
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Len(t, out.Compiled, 1, "embedded #module on the instance is sufficient for Compile")
}

// TestKernel_NoFinalizeMethod pins the absence of any finalization step on
// the kernel (spec kernel-runtime: "No finalization method on the Kernel";
// enhancement 0019 D1). Transformer inputs are filled as evaluated; a
// Finalize method reappearing here is a deliberate act, not drift.
func TestKernel_NoFinalizeMethod(t *testing.T) {
	_, found := reflect.TypeOf(&kernel.Kernel{}).MethodByName("Finalize")
	assert.False(t, found, "*kernel.Kernel must not expose a Finalize method (0019 D1)")
}

// TestKernel_PrunedPhaseSurface pins the removals of
// library-phase-and-values-prune: the kernel exposes exactly two phase verbs
// (Match, Compile), values enter through ProcessModuleInstance, and the six
// typed validation wrappers are gone. Any of these reappearing is a
// deliberate act, not drift.
func TestKernel_PrunedPhaseSurface(t *testing.T) {
	kt := reflect.TypeOf(&kernel.Kernel{})
	for _, name := range []string{
		"Plan", "Validate",
		"ValidateModuleValues", "ValidateModuleValuesPartial", "ValidateModuleValuesDetailed",
		"ValidateInstanceValues", "ValidateInstanceValuesPartial", "ValidateInstanceValuesDetailed",
	} {
		_, found := kt.MethodByName(name)
		assert.False(t, found, "*kernel.Kernel must not expose a %s method", name)
	}

	_, found := reflect.TypeOf(kernel.CompileInput{}).FieldByName("Values")
	assert.False(t, found, "CompileInput must not carry a Values field; values enter through ProcessModuleInstance")
}
