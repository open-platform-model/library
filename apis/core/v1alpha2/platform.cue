package v1alpha2

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

// #Platform — registry of registered Modules and their computed projections.
//
// Collisions on transformer FQN are caught by CUE map unification at
// #composedTransformers (the map is keyed by FQN; two definitions at the
// same key unify, identical bodies are no-ops, divergent bodies fail
// evaluation). Multiple transformers legitimately requiring the same
// resource/trait FQN is allowed and resolved by the runtime matcher's
// predicate evaluation.
#Platform: {
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

	// ---- Match index ----
	//
	// Reverse index from primitive FQN → list of transformers that require
	// that FQN. Multiple candidates per FQN is normal — the runtime matcher
	// evaluates each candidate's predicate against the component and pairs
	// every survivor.
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
	}
}
