package kernel_test

import (
	"context"
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/internal/renderstage"
	"github.com/open-platform-model/library/opm/kernel"
)

// The glue unifies each matched pair's transform once, in `rendered`: the
// failed pairs are named by the kernel from those outputs, so `diagnostics`
// carries no second evaluation of them. Every decoded verdict is the same
// either way, so this is asserted on the built value itself.
func TestRenderGlue_DiagnosticsCarryNoFailedPairs(t *testing.T) {
	k := newRenderKernel(t)
	plat := acquireRenderPlatform(t, k, "platform")
	inst := acquireRenderInstance(t, k, "scenarios", "failing")

	built, _, err := k.RenderForTest(context.Background(), kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
	require.Error(t, err)
	diagnostics := built.LookupPath(cue.ParsePath("diagnostics"))
	require.True(t, diagnostics.Exists())
	assert.False(t, diagnostics.LookupPath(cue.ParsePath("failedPairs")).Exists(),
		"diagnostics does not re-apply the matched pairs")
	assert.True(t, built.LookupPath(cue.MakePath(cue.Str("rendered"), cue.Str("crash :: "+renderTxPath+"/broken-transformer@0.1.0"), cue.Str("output"))).Exists(),
		"the failing pair's output is evaluated once, in rendered")
}

// fieldKeys lists the field labels of a struct value, hidden ones excluded.
func fieldKeys(t *testing.T, v cue.Value) []string {
	t.Helper()
	require.True(t, v.Exists())
	iter, err := v.Fields()
	require.NoError(t, err)
	var out []string
	for iter.Next() {
		out = append(out, iter.Selector().Unquoted())
	}
	return out
}

// The always-unify and predicate rungs are evaluated for a component's
// candidates only, never for every transformer on the platform. The decoded
// verdicts are the same either way, so this is asserted on the built value:
// each component's hidden rung structs carry exactly its candidate keys,
// on a platform whose transformers are not all candidates of it.
func TestRenderGlue_RungsCoverCandidatesOnly(t *testing.T) {
	k := newRenderKernel(t)
	plat := acquireRenderPlatform(t, k, "platform")
	inst := acquireRenderInstance(t, k, "instance")

	built, _, err := k.RenderForTest(context.Background(), kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
	require.NoError(t, err)

	hid := func(name string) cue.Selector { return cue.Hid(name, renderstage.RenderModulePath+":render") }
	match := built.LookupPath(cue.ParsePath("match"))
	all := fieldKeys(t, match.LookupPath(cue.MakePath(cue.Def("#transformers"))))

	for _, cid := range []string{"config", "web"} {
		v := match.LookupPath(cue.MakePath(cue.Str("verdicts"), cue.Str(cid)))
		candidates := fieldKeys(t, v.LookupPath(cue.MakePath(hid("_candidates"))))
		require.NotEmpty(t, candidates, "component %s has candidates", cid)
		assert.Less(t, len(candidates), len(all), "component %s: the platform carries transformers that are not its candidates", cid)
		assert.ElementsMatch(t, candidates, fieldKeys(t, v.LookupPath(cue.MakePath(hid("_unify")))),
			"component %s: the always-unify rung covers its candidates only", cid)
		assert.ElementsMatch(t, candidates, fieldKeys(t, v.LookupPath(cue.MakePath(hid("_pred")))),
			"component %s: the predicate rung covers its candidates only", cid)
	}
}

// The demand is a field of the build's own diagnostics, concrete beside the
// other verdicts, so it shares their fail-closed decode (0013:D24).
func TestRenderGlue_DiagnosticsCarryRequiredContracts(t *testing.T) {
	k := newRenderKernel(t)
	plat := acquireRenderPlatform(t, k, "platform")
	inst := acquireRenderInstance(t, k, "instance")

	built, _, err := k.RenderForTest(context.Background(), kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
	require.NoError(t, err)
	demand := built.LookupPath(cue.ParsePath("diagnostics.requiredContracts"))
	require.True(t, demand.Exists(), "diagnostics carries requiredContracts")
	require.NoError(t, demand.Validate(cue.Concrete(true)), "the demand is concrete")
	assert.Equal(t, cue.ListKind, demand.Kind(), "the demand is a list")
}
