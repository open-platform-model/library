// Package repro is a standalone reproducer for the CUE evaluator closedness
// regression that the catalog's hoisted-guard authoring rule works around.
//
// It is deliberately INVALID on affected CUE versions: `cue vet ./...` here is
// expected to fail with "x.out.a.b: field not allowed". This module is the
// ready-to-file upstream report; the Go canary asserting the failure lives in
// opm/internal/cueregression/closedness_test.go with the same body embedded as
// a string (keep the two in sync). Nothing evaluates this module — it sits
// outside CUE_MODULE_GLOBS so `task cue:vet` does not discover it.
//
// Measured (fresh CUE_CACHE_DIR per run, 2026-07-16):
//
//	v0.16.1           clean
//	v0.17.0-alpha.1   clean
//	v0.17.0           FAIL  <- regression introduced (339485ddf008)
//	v0.17.1           FAIL  <- still unfixed
//
// Each element below is load-bearing; removing any one makes it evaluate
// cleanly on every version:
//
//   - `b` must be STRUCT-typed. A scalar `b: int` is exempt — which is exactly
//     why the real catalog reports `scaling`/`updateStrategy` but never the
//     scalar `restartPolicy`.
//   - `#Inner` must be a definition (closedness must come from somewhere).
//     `close()` is NOT required.
//   - `#Base` must build `out` via a field comprehension.
//   - `#Derived` must extend it via unification (`#Base & {…}`), not inline.
//   - The trigger is the comprehension's CONDITION, not its body — the body
//     never mentions `b`.
//   - A concrete usage (`x`) is required; definitions alone do not trigger it.
package repro

#Inner: {
	b?: {n: int}
}

#Base: {
	#parts: {...}
	out: {
		for _, p in #parts {p}
	}
}

#Derived: #Base & {
	#parts: only: a: #Inner

	out: {
		a: #Inner

		// The condition traverses out.a.b — a struct-typed field of the closed
		// definition #Inner. That traversal loses the closedness info for out.a.
		if out.a.b != _|_ {
			a: {}
		}
	}
}

x: {
	#Derived

	// `b` IS declared by #Inner, so this must be allowed. Affected versions
	// reject it as "field not allowed".
	out: a: b: n: 2
}
