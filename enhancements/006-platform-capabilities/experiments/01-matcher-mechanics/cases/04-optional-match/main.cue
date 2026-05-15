package case04

import (
	v1 "opmodel.dev/exp006-01/schemas:v1alpha2"
)

// Hypothesis: optional FQN, provider present.
// Expected outcome: out.consumes.optional[fqn].spec concrete; `cue vet -c`
// passes.

platform: v1.#Platform & {
	metadata: name: "kind-prod"
	#provides: (v1.RouteFQN): v1.#Route & {
		spec: domain: "apps.example.com"
	}
}

result: (v1.#ContextBuilder & {
	#platform: platform
	#consumes: {
		required: {}
		optional: (v1.RouteFQN): v1.#Route
	}
}).out

_assertDomain: "apps.example.com" & result.consumes.optional[v1.RouteFQN].spec.domain
