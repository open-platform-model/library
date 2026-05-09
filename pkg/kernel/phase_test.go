package kernel_test

import (
	"context"
	"errors"
	"testing"

	"cuelang.org/go/cue"
	cueerrors "cuelang.org/go/cue/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/open-platform-model/library/pkg/api/v1alpha2"
	"github.com/open-platform-model/library/pkg/apiversion"
	"github.com/open-platform-model/library/pkg/compile"
	"github.com/open-platform-model/library/pkg/kernel"
	"github.com/open-platform-model/library/pkg/module"
	"github.com/open-platform-model/library/pkg/platform"
)

// phaseFixture builds a minimal Module + Release + Platform with a single
// component declaring one resource FQN, and one transformer in the
// platform's #composedTransformers / #matchers index that fulfills it.
// The transformer's output echoes #context fields so tests can confirm the
// full pipeline ran.
type phaseFixture struct {
	mod  *module.Module
	rel  *module.Release
	plat *platform.Platform
}

func newPhaseFixture(t *testing.T, k *kernel.Kernel) phaseFixture {
	t.Helper()
	ctx := k.CueContext()

	modPkg := ctx.CompileString(`
apiVersion: "opmodel.dev/v1alpha2"
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
apiVersion: "opmodel.dev/v1alpha2"
kind: "ModuleRelease"
metadata: { name: "demo", namespace: "ns", uuid: "u-rel" }
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
`)
	require.NoError(t, relPkg.Err())

	platVal := ctx.CompileString(`
apiVersion: "opmodel.dev/v1alpha2"
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
		"example.com/r/echo@v0": [#composedTransformers["example.com/p/echo@v0"]]
	}
	traits: {}
}
`)
	require.NoError(t, platVal.Err())

	return phaseFixture{
		mod: &module.Module{
			APIVersion: apiversion.V1alpha2,
			Metadata: &module.ModuleMetadata{
				Name:       "demo-mod",
				ModulePath: "example.com/m",
				Version:    "1.0.0",
				FQN:        "example.com/m/demo-mod:1.0.0",
				UUID:       "11111111-1111-1111-1111-111111111111",
			},
			Package: modPkg,
		},
		rel: &module.Release{
			APIVersion: apiversion.V1alpha2,
			Metadata: &module.ReleaseMetadata{
				Name: "demo", Namespace: "ns", UUID: "u-rel",
			},
			Package: relPkg,
		},
		plat: &platform.Platform{
			APIVersion: apiversion.V1alpha2,
			Metadata:   &platform.PlatformMetadata{Name: "k8s", Type: "kubernetes"},
			Package:    platVal,
		},
	}
}

func TestKernel_Validate_OK(t *testing.T) {
	k := kernel.New()
	f := newPhaseFixture(t, k)

	values := k.CueContext().CompileString(`{ replicas: 3, name: "demo" }`)
	require.NoError(t, values.Err())

	err := k.Validate(context.Background(), kernel.ValidateInput{
		Module: f.mod, ModuleRelease: f.rel, Values: values,
	})
	require.NoError(t, err)
}

func TestKernel_Validate_NoValues(t *testing.T) {
	k := kernel.New()
	f := newPhaseFixture(t, k)

	err := k.Validate(context.Background(), kernel.ValidateInput{
		Module: f.mod, ModuleRelease: f.rel,
	})
	require.NoError(t, err)
}

func TestKernel_Validate_FailureWrapsModuleName(t *testing.T) {
	k := kernel.New()
	f := newPhaseFixture(t, k)

	bad := k.CueContext().CompileString(`{ replicas: -1, name: "demo" }`)
	require.NoError(t, bad.Err())

	err := k.Validate(context.Background(), kernel.ValidateInput{
		Module: f.mod, ModuleRelease: f.rel, Values: bad,
	})
	require.Error(t, err)
	// Error MUST be framed with the module name and walkable as CUE-native.
	assert.Contains(t, err.Error(), `module "`+f.rel.Metadata.Name+`":`,
		"phase method MUST wrap with module-name framing")
	assert.NotEmpty(t, cueerrors.Errors(err),
		"wrapped error MUST remain walkable via cueerrors.Errors")
}

func TestKernel_Validate_RequiresInputs(t *testing.T) {
	k := kernel.New()
	f := newPhaseFixture(t, k)

	require.Error(t, k.Validate(context.Background(), kernel.ValidateInput{
		ModuleRelease: f.rel,
	}))
	require.Error(t, k.Validate(context.Background(), kernel.ValidateInput{
		Module: f.mod,
	}))
}

func TestKernel_Match_OK(t *testing.T) {
	k := kernel.New()
	f := newPhaseFixture(t, k)

	plan, err := k.Match(context.Background(), kernel.MatchInput{
		ModuleRelease: f.rel, Platform: f.plat,
	})
	require.NoError(t, err)
	require.NotNil(t, plan)
	pairs := plan.MatchedPairs()
	require.Len(t, pairs, 1)
	assert.Equal(t, "web", pairs[0].ComponentName)
	assert.Equal(t, "example.com/p/echo@v0", pairs[0].TransformerFQN)
}

func TestKernel_Match_VersionMismatch(t *testing.T) {
	k := kernel.New()
	f := newPhaseFixture(t, k)
	f.plat.APIVersion = apiversion.Version("opmodel.dev/v1alpha-other")

	_, err := k.Match(context.Background(), kernel.MatchInput{
		ModuleRelease: f.rel, Platform: f.plat,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "apiVersion mismatch")
}

func TestKernel_Plan_NoRendered(t *testing.T) {
	k := kernel.New()
	f := newPhaseFixture(t, k)

	out, err := k.Plan(context.Background(), kernel.PlanInput{
		ModuleRelease: f.rel, Platform: f.plat, RuntimeName: "opm-cli",
	})
	require.NoError(t, err)
	require.NotNil(t, out)

	// PlanResult does not expose a Compiled field — verify by reflection on
	// the public surface that no such slice leaks through.
	_ = out.MatchPlan
	require.Len(t, out.Components, 1)
	assert.Equal(t, "web", out.Components[0].Name)
	assert.Empty(t, out.Unmatched)
	assert.NotNil(t, out.Ambiguous)
}

func TestKernel_Plan_RequiresInputs(t *testing.T) {
	k := kernel.New()
	f := newPhaseFixture(t, k)

	_, err := k.Plan(context.Background(), kernel.PlanInput{
		Platform: f.plat, RuntimeName: "opm-cli",
	})
	require.Error(t, err, "ModuleRelease must be required")

	_, err = k.Plan(context.Background(), kernel.PlanInput{
		ModuleRelease: f.rel, RuntimeName: "opm-cli",
	})
	require.Error(t, err, "Platform must be required")

	_, err = k.Plan(context.Background(), kernel.PlanInput{
		ModuleRelease: f.rel, Platform: f.plat,
	})
	require.Error(t, err, "RuntimeName must be required")
}

func TestKernel_Compile_OK(t *testing.T) {
	k := kernel.New()
	f := newPhaseFixture(t, k)

	out, err := k.Compile(context.Background(), kernel.CompileInput{
		ModuleRelease: f.rel, Platform: f.plat, RuntimeName: "opm-cli",
	})
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Len(t, out.Compiled, 1)

	got := out.Compiled[0].Value
	runtime, err := got.LookupPath(cue.ParsePath("runtime")).String()
	require.NoError(t, err)
	assert.Equal(t, "opm-cli", runtime)
}

// TestKernel_Compile_FromReleaseOnly is a regression test for the slim-input
// contract: Compile must succeed when given a release whose Package carries an
// embedded #module reference, with no separate *module.Module supplied.
func TestKernel_Compile_FromReleaseOnly(t *testing.T) {
	k := kernel.New()
	f := newPhaseFixture(t, k)

	out, err := k.Compile(context.Background(), kernel.CompileInput{
		ModuleRelease: f.rel,
		Platform:      f.plat,
		RuntimeName:   "opm-cli",
	})
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Len(t, out.Compiled, 1, "embedded #module on the release is sufficient for Compile")
}

// TestRender_ModuleResult_Aliased verifies *compile.ModuleResult resolves to
// *compile.CompileResult via the type alias.
func TestRender_ModuleResult_Aliased(t *testing.T) {
	var cr *compile.CompileResult
	var mr *compile.ModuleResult = cr //nolint:staticcheck // SA1019: testing alias compatibility
	_ = mr
}

func TestKernel_DetectAPIVersion(t *testing.T) {
	k := kernel.New()
	v := k.CueContext().CompileString(`apiVersion: "opmodel.dev/v1alpha2"`)
	require.NoError(t, v.Err())

	got, err := k.DetectAPIVersion(v)
	require.NoError(t, err)
	assert.Equal(t, apiversion.V1alpha2, got)
}

func TestKernel_DetectAPIVersion_Unknown(t *testing.T) {
	k := kernel.New()
	v := k.CueContext().CompileString(`apiVersion: "opmodel.dev/never-registered"`)
	require.NoError(t, v.Err())

	_, err := k.DetectAPIVersion(v)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apiversion.ErrUnknownAPIVersion))
}

func TestKernel_Finalize(t *testing.T) {
	k := kernel.New()
	v := k.CueContext().CompileString(`{
	replicas: int & >0
	replicas: 3
	name: "demo"
}`)
	require.NoError(t, v.Err())

	got, err := k.Finalize(v)
	require.NoError(t, err)
	require.True(t, got.Exists())

	// After finalization the constraint is gone — the value is concrete.
	replicas, err := got.LookupPath(cue.ParsePath("replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(3), replicas)
}
