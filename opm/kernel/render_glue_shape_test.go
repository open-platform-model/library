package kernel_test

import (
	"context"
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
