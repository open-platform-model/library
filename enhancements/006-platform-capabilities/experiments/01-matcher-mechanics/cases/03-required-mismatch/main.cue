package case03

import (
	v1 "opmodel.dev/exp006-01/schemas:v1alpha2"
)

// Hypothesis: required FQN, provider present, value violates the
// capability's schema constraint (#Route.spec.domain regex requires a dot).
// Expected outcome: CUE bottom localized at the offending field.
// `cue vet -c` MUST fail and the diagnostic path MUST point at
// `result.consumes.required[fqn].spec.domain` (006 D6 alt).
//
// Note: this case is constructed to produce the mismatch INSIDE the
// matcher, not at provider authoring time. To do so the platform's
// #provides entry is typed as #Capability (open) and supplies a domain
// value that satisfies `string` but violates the route's regex.
//
// Why not type the provider entry as #Route directly? Because that would
// fail at platform-construction time — which is also a valid pathway, but
// not the one we are probing. We want to verify the matcher's output is
// a bottom at the FQN when the provider+consumer disagree on the spec
// constraint.

platform: v1.#Platform & {
	metadata: name: "kind-bad-domain"
	// Bypass the #Route constraint at the provider side by typing the entry
	// as bare #Capability with a free-form spec. The matcher will unify
	// this with #Route on the consumer side, and THE UNIFIED VALUE is the
	// bottom under test.
	#provides: (v1.RouteFQN): v1.#Capability & {
		metadata: {
			name:       "route"
			modulePath: "opmodel.dev/exp/caps/routing"
			version:    "v1"
		}
		spec: domain: "no-tld" // violates =~"^[a-z0-9.-]+\\.[a-z]+$"
	}
}

result: (v1.#ContextBuilder & {
	#platform: platform
	#consumes: {
		required: (v1.RouteFQN): v1.#Route
		optional: {}
	}
}).out
