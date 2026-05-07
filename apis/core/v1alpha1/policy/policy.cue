package policy

import (
	t "opmodel.dev/core/v1alpha1/types@v1"
	prim "opmodel.dev/core/v1alpha1/primitives@v1"
	component "opmodel.dev/core/v1alpha1/component@v1"
)

// #Policy: Groups PolicyRules and Directives and targets them
// to a set of components via label matching or explicit references.
// Policies enable cross-cutting governance and operational behavior
// without coupling rules to individual components.
#Policy: {
	apiVersion: "opmodel.dev/core/v1alpha1"
	kind:       "Policy"

	metadata: {
		name!: t.#NameType

		labels?:      t.#LabelsAnnotationsType
		annotations?: t.#LabelsAnnotationsType
	}

	// PolicyRules grouped by this policy (governance)
	#rules: [RuleFQN=string]: prim.#PolicyRule

	// Directives grouped by this policy (operational behavior)
	#directives?: [DirectiveFQN=string]: prim.#Directive

	// Which components this policy applies to
	// At least one of matchLabels or components must be specified
	appliesTo: {
		// Label-based matching — select components whose labels are a superset
		matchLabels?: t.#LabelsAnnotationsType

		// Explicit component references
		components?: [...component.#Component]
	}

	_allFields: {
		if #rules != _|_ {
			for _, rule in #rules {
				if rule.#spec != _|_ {
					for k, v in rule.#spec {
						(k): v
					}
				}
			}
		}
		if #directives != _|_ {
			for _, directive in #directives {
				if directive.#spec != _|_ {
					for k, v in directive.#spec {
						(k): v
					}
				}
			}
		}
	}

	// Fields exposed by this policy
	// Automatically turned into a spec
	// Must be made concrete by the user
	spec: close(_allFields)
}

#PolicyMap: [string]: #Policy
