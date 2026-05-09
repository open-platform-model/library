package v1alpha2

import (
	"list"
	"strings"
)

// #ModuleRegistration — single entry in #Platform.#registry.
// Pure projection of "this Module's primitives are visible on this platform".
// Carries no install metadata (014 D11). enabled: false hides every projection
// (014 D14). presentation is flat (014 D14, post-D11 cleanup).
#ModuleRegistration: {
	#module!: #Module

	enabled: bool | *true

	presentation?: {
		description?: string
		category?:    string
		tags?: [...string]
		examples?: [Name=string]: {
			description?: string
			values:       _
		}
	}

	metadata?: {
		labels?:      #LabelsAnnotationsType
		annotations?: #LabelsAnnotationsType
	}
}

// #PlatformBase — every projection except the multi-fulfiller hard-fail
// constraint (D13). Used by tests that need to inspect #matchers._invalid as
// a diagnostic surface (the strict #Platform definition would short-circuit
// to _|_ before a test could read _invalid). Production callers use #Platform.
#PlatformBase: {
	apiVersion: #ApiVersion
	kind:       "Platform"

	metadata: {
		name!:        #NameType
		description?: string
		labels?:      #LabelsAnnotationsType
		annotations?: #LabelsAnnotationsType
	}

	// #Platform.type — kept as authored field. Future enhancement may enforce
	// type-vs-transformer compatibility; today informational (014 OQ2).
	type!: string

	// #registry — kebab-case Id key (D16). Static + runtime writes unify by
	// Id; concrete-value disagreement = _|_ surfaced by reconciler (D15).
	#registry: [Id=#NameType]: #ModuleRegistration

	// ---- Computed views over #registry ----
	// Each gates on `reg.enabled` (D14 — disabled entries hide everything,
	// types and transformers alike).

	#knownResources: {
		[FQN=string]: #Resource
		for _, reg in #registry
		if reg.enabled
		if reg.#module.#defines != _|_
		if reg.#module.#defines.resources != _|_
		for fqn, v in reg.#module.#defines.resources {
			(fqn): v
		}
	}

	#knownTraits: {
		[FQN=string]: #Trait
		for _, reg in #registry
		if reg.enabled
		if reg.#module.#defines != _|_
		if reg.#module.#defines.traits != _|_
		for fqn, v in reg.#module.#defines.traits {
			(fqn): v
		}
	}

	#composedTransformers: #TransformerMap & {
		for _, reg in #registry
		if reg.enabled
		if reg.#module.#defines != _|_
		if reg.#module.#defines.transformers != _|_
		for fqn, v in reg.#module.#defines.transformers {
			(fqn): v
		}
	}

	// ---- Match index (D12) ----
	//
	// Pre-compute candidate maps as `let` bindings so the _invalid projection
	// can iterate them directly. Iterating the published fields (resources,
	// traits) fails with "incomplete type list" because the field type is
	// `[FQN]: [...#ComponentTransformer]` — an open value-list — which CUE
	// refuses to range over.
	#matchers: {
		let _resourceFqns = {
			for _, t in #composedTransformers
			if t.requiredResources != _|_
			for fqn, _ in t.requiredResources {
				(fqn): _
			}
		}
		let _traitFqns = {
			for _, t in #composedTransformers
			if t.requiredTraits != _|_
			for fqn, _ in t.requiredTraits {
				(fqn): _
			}
		}

		let _resourceCandidates = {
			for fqn, _ in _resourceFqns {
				(fqn): [
					for _, t in #composedTransformers
					if t.requiredResources != _|_
					if t.requiredResources[fqn] != _|_ {t},
				]
			}
		}
		let _traitCandidates = {
			for fqn, _ in _traitFqns {
				(fqn): [
					for _, t in #composedTransformers
					if t.requiredTraits != _|_
					if t.requiredTraits[fqn] != _|_ {t},
				]
			}
		}

		resources: {[FQN=string]: [...#ComponentTransformer]} & _resourceCandidates
		traits: {[FQN=string]: [...#ComponentTransformer]} & _traitCandidates

		// Per-transformer match-predicate signature. Two transformers with the
		// same signature have indistinguishable match domains — a Component
		// matching one would match the other. Sorted "k=v" label pairs and
		// sorted required-trait FQNs make the signature deterministic and
		// equality-comparable as a plain string.
		let _predicateSignature = {
			for _, t in #composedTransformers {
				(t.metadata.fqn): {
					labelPart: strings.Join([
						if t.requiredLabels != _|_
						for k in list.Sort([for k, _ in t.requiredLabels {k}], list.Ascending) {
							"\(k)=\(t.requiredLabels[k])"
						},
					], ",")
					traitPart: strings.Join([
						if t.requiredTraits != _|_
						for fqn in list.Sort([for fqn, _ in t.requiredTraits {fqn}], list.Ascending) {
							fqn
						},
					], ",")
					sig: "\(labelPart);\(traitPart)"
				}
			}
		}

		// Diagnostic surface — flags FQNs where two candidate transformers
		// share an identical match predicate. Shared FQNs across candidates
		// with *different* predicates are fine (e.g. all workload transformers
		// require container@v1 but each is gated by a unique workload-type
		// label, so no Component can match more than one).
		_invalid: {
			resources: [
				for fqn, ts in _resourceCandidates if len(ts) > 1
				let _sigs = [for t in ts {_predicateSignature[t.metadata.fqn].sig}]
				if len([for i, a in _sigs for j, b in _sigs if j > i if a == b {1}]) > 0 {
					fqn
				},
			]
			traits: [
				for fqn, ts in _traitCandidates if len(ts) > 1
				let _sigs = [for t in ts {_predicateSignature[t.metadata.fqn].sig}]
				if len([for i, a in _sigs for j, b in _sigs if j > i if a == b {1}]) > 0 {
					fqn
				},
			]
		}
	}
}

// #Platform — strict form. Adds the multi-fulfiller hard-fail constraint (D13).
// Use this for production schemas; use #PlatformBase only when a test needs to
// inspect _invalid before the constraint short-circuits.
#Platform: #PlatformBase & {
	#matchers: _noMultiFulfiller: 0 & (len(#matchers._invalid.resources) +
		len(#matchers._invalid.traits))
}
