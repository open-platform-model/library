package component

import (
	t "opmodel.dev/core/v1alpha1/types@v1"
	prim "opmodel.dev/core/v1alpha1/primitives@v1"
)

// Workload type label key
#LabelWorkloadType: "core.opmodel.dev/workload-type"

#Component: {
	apiVersion: "opmodel.dev/core/v1alpha1"
	kind:       "Component"

	metadata: {
		name!: t.#NameType

		// Component labels - unified from all attached resources, traits, and blueprints
		// If definitions have conflicting labels, CUE unification will fail (automatic validation).
		labels?: t.#LabelsAnnotationsType

		// Component annotations - unified from all attached resources, traits, and blueprints
		// If definitions have conflicting annotations, CUE unification will fail (automatic validation).
		annotations?: t.#LabelsAnnotationsType
	}

	// Resources applied for this component
	#resources: prim.#ResourceMap

	// Traits applied to this component
	#traits?: prim.#TraitMap

	// Blueprints applied to this component
	#blueprints?: prim.#BlueprintMap

	_allFields: {
		for _, resource in #resources {
			if resource.spec != _|_ {resource.spec}
		}
		if #traits != _|_ {
			for _, trait in #traits {
				if trait.spec != _|_ {trait.spec}
			}
		}
		if #blueprints != _|_ {
			for _, blueprint in #blueprints {
				if blueprint.spec != _|_ {blueprint.spec}
			}
		}
	}

	// Fields exposed by this component (merged from all resources, traits, and blueprints)
	// Automatically turned into a spec.
	// Must be made concrete by the user.
	// Have to do it this way because if we allowed the spec flattened in the root of the component
	// we would have to open the #Module definition which would make it impossible to properly validate.
	spec: close({
		_allFields
	})
}

#ComponentMap: [string]: #Component
