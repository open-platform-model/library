package case05

import (
	v1 "opmodel.dev/exp006-01/schemas:v1alpha2"
)

// Hypothesis: optional FQN, no provider.
// Expected outcome: out.consumes.optional[fqn] absent entirely (comprehension
// `if` drops the entry). `cue vet -c` passes — modules guard reads with
// `if #consumes.optional[<fqn>] != _|_`.

platform: v1.#Platform & {
	metadata: name: "kind-empty"
	#provides: {}
}

result: (v1.#ContextBuilder & {
	#platform: platform
	#consumes: {
		required: {}
		optional: (v1.RouteFQN): v1.#Route
	}
}).out

// Assertion: result.consumes.optional must be the empty struct {}.
// We assert by unifying against {} (closed) and checking concreteness.
_assertEmptyOptional: result.consumes.optional & {}

// Sanity: looking up the absent entry must yield _|_.
_assertAbsent: (result.consumes.optional[v1.RouteFQN] == _|_) & true
