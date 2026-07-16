package kernel_test

import (
	"path/filepath"
	"strings"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	cueerrors "cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/load"
	"github.com/stretchr/testify/require"
)

// TestCUEClosednessRegression_StillPresent is a toolchain canary, not a test of
// library behavior. It asserts that the pinned cuelang.org/go still exhibits the
// evaluator closedness regression that the OPM catalog works around by hoisting
// its `if … != _|_` propagation guards out of `spec`.
//
// WHEN THIS TEST FAILS, THAT IS THE GOOD NEWS: upstream has fixed the bug. The
// correct response is not to "fix" this test — it is to
//
//  1. confirm the fix is in a released CUE version the toolchain has adopted,
//  2. retire the hoisted-guard authoring rule (catalog_opm
//     docs/cue-guard-closedness-workaround.md and its CLAUDE.md pitfall entry),
//  3. delete this canary and docs/design/repro-cue-closedness/.
//
// Background: the regression arrived in v0.17.0-alpha.2 (commit 339485ddf008,
// "dependency-tracking comprehension pushdown") and is still present in v0.17.1.
// v0.17.1 fixed cue-lang/cue#4423 *as filed* — a different symptom ("adding
// field … already referenced") that OPM's report was conflated with. OPM's
// symptom survives and is not yet reported upstream. Full detail:
// docs/design/cue-closedness-regression-alpha2.md.
//
// The older canary (TestIntegration_Live_ValidateRealConfig) can no longer
// detect this: it resolves the published catalog, which now carries the
// workaround, so it passes on broken and fixed CUE alike. This one evaluates the
// trigger shape directly and has no registry or network dependency.
func TestCUEClosednessRegression_StillPresent(t *testing.T) {
	dir := filepath.Join(repoLibraryRoot(t), "docs", "design", "repro-cue-closedness")

	insts := load.Instances([]string{"."}, &load.Config{Dir: dir})
	require.Len(t, insts, 1)
	require.NoError(t, insts[0].Err, "loading the reproducer package")

	val := cuecontext.New().BuildInstance(insts[0])
	err := val.Validate()

	if err == nil {
		t.Fatalf(`the CUE closedness regression appears to be FIXED in this cuelang.org/go.

This canary deliberately fails when upstream fixes the bug. Do not "repair" it.
See the doc comment on this test: confirm the fix, then retire the hoisted-guard
rule in catalog_opm (docs/cue-guard-closedness-workaround.md + CLAUDE.md), delete
docs/design/repro-cue-closedness/, and delete this test.`)
	}

	// Pin the symptom, so an unrelated breakage in the fixture cannot masquerade
	// as "the bug is still there".
	got := cueerrors.Details(err, nil)
	require.Truef(t, strings.Contains(got, "field not allowed"),
		"expected the closedness symptom %q, got a different error:\n%s", "field not allowed", got)
	require.Truef(t, strings.Contains(got, "x.out.a.b"),
		"expected the error to name x.out.a.b, got:\n%s", got)
}
