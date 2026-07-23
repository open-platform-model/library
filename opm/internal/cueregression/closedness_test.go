// Package cueregression is a toolchain canary, not a test of library behavior.
// It pins the CUE evaluator closedness regression (spurious "field not allowed"
// on comprehension-guarded nested struct fields) that arrived in
// cuelang.org/go v0.17.0-alpha.2 (commit 339485ddf008, "dependency-tracking
// comprehension pushdown") and is still present in v0.17.1. v0.17.1 fixed
// cue-lang/cue#4423 *as filed* — a different symptom OPM's report was conflated
// with; OPM's symptom survives and is not yet reported upstream.
//
// The OPM catalog is only rendering cleanly because catalog_opm ships the
// hoisted-guard authoring rule (docs/cue-guard-closedness-workaround.md), which
// removed the trigger shape from the published catalog — and thereby also
// removed it from every registry-resolving integration test. These two tests
// restore hermetic detection in both directions:
//
//   - the trigger form is asserted to STILL FAIL — when a future CUE bump makes
//     it evaluate clean, upstream has fixed our shape and the test inverts
//     loudly with re-evaluation instructions;
//   - the hoisted form is asserted to stay CLEAN — a failure means the shipped
//     workaround itself broke, which blocks any CUE version bump.
//
// Both fixtures are embedded strings evaluated via cuecontext — no registry, no
// network, no filesystem module loading, no cue.mod. Full history and the
// bisected trigger rule: docs/design/cue-closedness-regression-alpha2.md. The
// ready-to-file upstream reproducer: docs/design/repro-cue-closedness/.
package cueregression

import (
	"runtime/debug"
	"strings"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	cueerrors "cuelang.org/go/cue/errors"
	"github.com/stretchr/testify/require"
)

// triggerForm is the minimal reproducer, transcribed verbatim from
// docs/design/cue-closedness-regression-alpha2.md ("Minimal in-package
// reproduction"). Six elements are individually load-bearing — removing any one
// makes it evaluate clean on every version: `b` struct-typed, #Inner a
// definition, the field comprehension in #Base, the `#Base & {…}` unification
// split, the guard CONDITION traversing out.a.b, and the concrete usage `x`.
const triggerForm = `
#Inner: {b?: {n: int}}
#Base: {#parts: {...}, out: {for _, p in #parts {p}}}
#Derived: #Base & {
	#parts: only: a: #Inner
	out: {
		a: #Inner
		if out.a.b != _|_ {a: {}}   // CONDITION traverses a struct-typed field of closed #Inner
	}
}
x: {#Derived, out: a: b: n: 2}     // v0.17.0/v0.17.1: "x.out.a.b: field not allowed"
`

// hoistedForm is the same reproducer with the shipped workaround applied: the
// guard is hoisted out of the struct it contributes to (workaround 1 in the
// design doc, the authoring rule catalog_opm encodes in
// docs/cue-guard-closedness-workaround.md). Semantics are unchanged; only the
// guard's lexical position moves.
const hoistedForm = `
#Inner: {b?: {n: int}}
#Base: {#parts: {...}, out: {for _, p in #parts {p}}}
#Derived: #Base & {
	#parts: only: a: #Inner
	out: {
		a: #Inner
	}
	if out.a.b != _|_ {out: a: {}}   // hoisted: the guard no longer sits inside out
}
x: {#Derived, out: a: b: n: 2}
`

// TestTriggerForm_StillFails asserts the pinned cuelang.org/go still exhibits
// the regression.
//
// WHEN THIS TEST FAILS, THAT IS THE GOOD NEWS: upstream has fixed the bug. The
// correct response is not to "fix" this test — it is to
//
//  1. confirm the fix is in a released CUE version the toolchain has adopted,
//  2. re-run the regression matrix in
//     docs/design/cue-closedness-regression-alpha2.md,
//  3. deliberately re-evaluate — do not silently revert — the hoisted-guard
//     authoring rule (catalog_opm docs/cue-guard-closedness-workaround.md and
//     its CLAUDE.md pitfall entry),
//  4. then retire this canary and docs/design/repro-cue-closedness/ together.
func TestTriggerForm_StillFails(t *testing.T) {
	val := cuecontext.New().CompileString(triggerForm)
	err := val.Validate()

	if err == nil {
		t.Fatalf(`the CUE closedness regression appears to be FIXED in cuelang.org/go %s.

This canary deliberately fails when upstream fixes the bug. Do not "repair" it.
Re-run the regression matrix in docs/design/cue-closedness-regression-alpha2.md,
re-evaluate (do NOT silently revert) the hoisted-guard rule in catalog_opm
(docs/cue-guard-closedness-workaround.md + CLAUDE.md), then retire this canary
and docs/design/repro-cue-closedness/ deliberately.

If validation is clean because CUE reworded the diagnostic rather than fixed the
bug, update the match below instead — the matrix run will tell the two apart.`,
			cueVersion())
	}

	// Pin the symptom narrowly, so an unrelated breakage in the fixture cannot
	// masquerade as "the bug is still there".
	got := cueerrors.Details(err, nil)
	require.Truef(t, strings.Contains(got, "field not allowed"),
		"expected the closedness symptom %q, got a different error:\n%s", "field not allowed", got)
	require.Truef(t, strings.Contains(got, "x.out.a.b"),
		"expected the error to name x.out.a.b, got:\n%s", got)
}

// TestHoistedForm_Clean asserts the shipped workaround pattern evaluates clean
// on the pinned cuelang.org/go. This is a RELEASE GATE for any CUE version
// bump: if it fails, the hoisted-guard form the published catalog relies on is
// broken on the new evaluator, and real rendering breaks with it.
func TestHoistedForm_Clean(t *testing.T) {
	val := cuecontext.New().CompileString(hoistedForm)
	err := val.Validate()
	if err != nil {
		t.Fatalf(`the hoisted-guard workaround form FAILS on cuelang.org/go %s:

%s

The published catalog depends on this exact shape rendering clean
(catalog_opm docs/cue-guard-closedness-workaround.md). Do not adopt this CUE
version until the workaround is repaired or replaced.`,
			cueVersion(), cueerrors.Details(err, nil))
	}
}

// cueVersion returns the running cuelang.org/go module version, or a
// placeholder when build info is unavailable (e.g. some test binaries).
func cueVersion() string {
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range bi.Deps {
			if dep.Path == "cuelang.org/go" {
				return dep.Version
			}
		}
	}
	return "(unknown version)"
}
