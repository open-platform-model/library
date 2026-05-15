package case02

import (
	v1 "opmodel.dev/exp006-01/schemas:v1alpha2"
)

// Hypothesis: required FQN, no provider.
// Expected outcome: out.consumes.required[fqn].spec stays incomplete.
// `cue vet -c` MUST fail with a diagnostic naming
// `result.consumes.required.<fqn>.spec.domain` (or similar precise path).
//
// This case is a NEGATIVE fixture — the expected verification is that
// `cue vet -c` fails. run.sh captures the diagnostic.

platform: v1.#Platform & {
	metadata: name: "kind-empty"
	#provides: {}
}

result: (v1.#ContextBuilder & {
	#platform: platform
	#consumes: {
		required: (v1.RouteFQN): v1.#Route
		optional: {}
	}
}).out
