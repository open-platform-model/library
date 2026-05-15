package case01

import (
	v1 "opmodel.dev/exp006-01/schemas:v1alpha2"
)

// Hypothesis: required FQN, provider present, schema matches.
// Expected outcome: out.consumes.required[fqn].spec is concrete with the
// provider's values. `cue vet -c` passes.

platform: v1.#Platform & {
	metadata: name: "kind-prod"
	#provides: (v1.RouteFQN): v1.#Route & {
		spec: domain: "apps.example.com"
	}
}

result: (v1.#ContextBuilder & {
	#platform: platform
	#consumes: {
		required: (v1.RouteFQN): v1.#Route
		optional: {}
	}
}).out

// Assertion — equality unification fails if the matcher did not produce
// the expected concrete value. `cue vet -c` reports the mismatch.
_assertDomain: "apps.example.com" & result.consumes.required[v1.RouteFQN].spec.domain
