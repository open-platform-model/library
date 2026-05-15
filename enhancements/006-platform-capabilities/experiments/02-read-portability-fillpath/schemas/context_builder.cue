package v1alpha2

// #ContextBuilder — isolated to the 006 capability-matching step.
//
// In the production design, #ContextBuilder also computes 004's runtime
// context (out.ctx, out.injections). Those are stubbed here — exp 01 is
// scoped to the matcher alone. Copy of design 03-schema.md §"#ContextBuilder
// matching step" with the 004 lets/outputs elided.
#ContextBuilder: {
	#platform: #Platform
	#consumes: {
		required: [#FQNType]: #Capability
		optional: [#FQNType]: #Capability
	}

	// Capability matching.
	//
	// required: for each consumed FQN, unify cap with the provider value if
	//   present. If absent, the conditional embedding contributes nothing —
	//   cap's spec! stays incomplete and `cue vet -c` reports the FQN path
	//   (006 D6).
	// optional: the outer `if` drops the entry entirely when no provider
	//   exists, so out.consumes.optional[fqn] is absent.
	//
	// Schema mismatches surface as CUE bottoms at the FQN — release-time
	// errors either way.
	let _matched = {
		required: {
			for fqn, cap in #consumes.required {
				(fqn): cap & {
					if #platform.#provides[fqn] != _|_ {
						#platform.#provides[fqn]
					}
				}
			}
		}
		optional: {
			for fqn, cap in #consumes.optional
			if #platform.#provides[fqn] != _|_ {
				(fqn): cap & #platform.#provides[fqn]
			}
		}
	}

	out: consumes: _matched
}
