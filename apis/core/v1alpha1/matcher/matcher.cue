package matcher

import (
	"list"
	component "opmodel.dev/core/v1alpha1/component@v1"
	provider "opmodel.dev/core/v1alpha1/provider@v1"
)

// #MatchResult captures the outcome of evaluating a single (component, transformer) pair.
//
// When matched is true, missing* lists are all empty.
// When matched is false, missing* lists explain exactly why the transformer did not match —
// this is the structured diagnostic the Go engine surfaces to the operator.
#MatchResult: {
	matched: bool

	// Labels that the transformer required but the component did not have.
	// Format: "key=value" strings (same as the requiredLabels map entries).
	missingLabels: [...string]

	// Resource FQNs that the transformer required but the component did not declare.
	missingResources: [...string]

	// Trait FQNs that the transformer required but the component did not declare.
	missingTraits: [...string]
}

// #MatchPlan evaluates every (component, transformer) pair and computes:
//   - matches:         full result matrix — used both to drive rendering and for diagnostics
//   - unmatched:       component names that had zero matching transformers (error condition)
//   - unhandledTraits: per component, traits present but not handled by any matched transformer (warning)
//
// Inputs are raw values; #MatchPlan has no coupling to #ModuleRelease.
// The Go engine fills #provider and #components, evaluates, and decodes.
#MatchPlan: {
	#provider:   provider.#Provider
	#components: component.#ComponentMap

	// Full (component × transformer) match result matrix.
	//
	// matches["web"]["opmodel.dev/opm/transformers/kubernetes/deployment-transformer@v0"] = {
	//   matched: true, missingLabels: [], missingResources: [], missingTraits: []
	// }
	matches: {
		for compName, comp in #components {
			// Pre-compute the component's label pairs, resource FQNs, and trait FQNs as
			// flat lists so list.Contains can perform concrete boolean checks.
			// Use *field | {} so CUE picks the field value as default when present,
			// and falls back to an empty struct when the optional field is absent.
			let _labelPairs = [for k, v in (*comp.metadata.labels | {}) {"\(k)=\(v)"}]
			let _resourceFQNs = [for fqn, _ in comp.#resources {fqn}]
			let _traitFQNs = [for fqn, _ in (*comp.#traits | {}) {fqn}]

			(compName): {
				for tfFQN, tf in #provider.#transformers {
					// Compute what's missing — empty list means requirement satisfied.
					// requiredLabels is optional on #Transformer; use *field | {} fallback.
					let _missingLabels = [
						for k, v in (*tf.requiredLabels | {})
						if !list.Contains(_labelPairs, "\(k)=\(v)") {"\(k)=\(v)"},
					]
					let _missingResources = [
						for fqn, _ in tf.requiredResources
						if !list.Contains(_resourceFQNs, fqn) {fqn},
					]
					let _missingTraits = [
						for fqn, _ in tf.requiredTraits
						if !list.Contains(_traitFQNs, fqn) {fqn},
					]

					(tfFQN): #MatchResult & {
						matched:          len(_missingLabels) == 0 && len(_missingResources) == 0 && len(_missingTraits) == 0
						missingLabels:    _missingLabels
						missingResources: _missingResources
						missingTraits:    _missingTraits
					}
				}
			}
		}
	}

	// Component names for which no transformer matched.
	// A non-empty list is an error condition: the engine cannot produce output for these components.
	unmatched: [
		for compName, compResults in matches
		if len([for _, r in compResults if r.matched {true}]) == 0 {compName},
	]

	// Per-component list of trait FQNs that no matched transformer handles.
	// Non-empty entries are warnings: the trait's values are present but will be silently ignored.
	//
	// A trait is "handled" if it appears in the requiredTraits OR optionalTraits of
	// at least one transformer that matched this component.
	unhandledTraits: {
		for compName, comp in #components {
			// Collect FQNs of all transformers that matched this component.
			let _matchedTFQNs = [for tfFQN, r in matches[compName] if r.matched {tfFQN}]

			// Union of all traits handled (required or optional) by the matched transformers.
			let _handledTraits = list.FlattenN([
				for tfFQN in _matchedTFQNs {
					list.Concat([
						[for fqn, _ in #provider.#transformers[tfFQN].requiredTraits {fqn}],
						[for fqn, _ in #provider.#transformers[tfFQN].optionalTraits {fqn}],
					])
				},
			], 1)

			// Traits present on the component but absent from _handledTraits.
			(compName): [
				for fqn, _ in (*comp.#traits | {})
				if !list.Contains(_handledTraits, fqn) {fqn},
			]
		}
	}
}
