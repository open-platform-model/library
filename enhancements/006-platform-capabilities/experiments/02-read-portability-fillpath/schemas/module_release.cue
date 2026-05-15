package v1alpha2

// #ModuleRelease — REVISED in light of exp 02 finding F1.
//
// The 006/03-schema.md design specified that #ModuleRelease invokes
// #ContextBuilder inline and unifies `out.consumes` back into `#module.
// #consumes`. Exp 02 probe runs (see README.md "Finding F1") show this
// creates a self-referential evaluation cycle: `_builderOut` depends on
// `#module.#consumes`, and `#module.#consumes` is set to `_builderOut.
// consumes`. CUE 0.16.1 (both default and evalv3) does not resolve this
// fixed point — it freezes #consumes at the type-level constraint
// (spec.domain: string) regardless of provider value.
//
// The mechanism that DOES work (probe runs and probe26 in the experiment
// transcript): the kernel computes the matched #consumes externally (a
// top-level #ContextBuilder call with module + platform inputs) and
// FillPaths the resolved values into `#module.#consumes` BEFORE the release
// is evaluated. The release itself does no CB call.
//
// This file reflects the revised mechanism: #ModuleRelease has no inline CB
// invocation. End-user-authored releases ship without #consumes writeback;
// the kernel/CLI/operator does both:
//   1. FillPath #platform onto the release (006 D13 — unchanged).
//   2. FillPath the matched #consumes entries onto #module.#consumes
//      (the writeback the in-CUE design intended to perform but cannot).
#ModuleRelease: {
	apiVersion: #ApiVersion
	kind:       "ModuleRelease"

	metadata: {
		name!:      #NameType
		namespace!: string
	}

	#module!: #Module
	values:   _

	// Kernel-populated. End-users do not author this field.
	#platform: #Platform

	// Step 1 — unify values into #config (004 D34 — unchanged).
	let _withConfig = #module & {#config: values}

	// Step 2 — components are extracted from the post-config module. The
	// kernel is expected to have already FillPathed any matched #consumes
	// entries onto #module.#consumes by the time the release is evaluated.
	// References inside a component body to `#consumes.required[fqn].spec.X`
	// resolve through the lexical scope of the (authored) module, which is
	// now the (kernel-resolved) module. Verified by direct unification — see
	// experiments/02-read-portability-fillpath/cases/jellyfin/release-bound.cue
	// which simulates the kernel writeback inline.
	components: {
		for name, comp in _withConfig.#components {
			(name): comp
		}
	}
}
