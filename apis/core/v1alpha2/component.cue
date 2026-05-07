package v1alpha2

// Workload type label key
#LabelWorkloadType: "core.opmodel.dev/workload-type"

#Component: {
	apiVersion: #ApiVersion
	kind:       "Component"

	metadata: {
		name!: #NameType

		// Component labels - unified from all attached resources, traits, and blueprints
		// If definitions have conflicting labels, CUE unification will fail (automatic validation).
		labels?: #LabelsAnnotationsType

		// Component annotations - unified from all attached resources, traits, and blueprints
		// If definitions have conflicting annotations, CUE unification will fail (automatic validation).
		annotations?: #LabelsAnnotationsType
	}

	// Resources applied for this component
	#resources: #ResourceMap

	// Traits applied to this component
	#traits?: #TraitMap

	// Blueprints applied to this component
	#blueprints?: #BlueprintMap

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
