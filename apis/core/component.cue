package core

// Workload type label key
#LabelWorkloadType: "core.opmodel.dev/workload-type"

#Component: {
	kind: "Component"

	metadata: {
		name!: #NameType

		// Per-component resource-name override. Defaults to metadata.name; an
		// explicit value wins via the disjunction-default cascade. Introduced by
		// enhancement 0001 (D2): #names reads from here to compute the rendered
		// resource name and its DNS variants.
		resourceName: *name | #NameType

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

	// Release context injected by the parent #Module via its #components
	// pattern constraint. Hidden definition slot — module authors never set
	// this directly. Introduced by enhancement 0001 (D3).
	#release: #ReleaseIdentity

	// Single source of truth for this component's computed names. `resourceName`
	// reads straight from metadata (cascade lives there); DNS variants derive
	// deterministically from resourceName + #release.namespace + #release.clusterDomain.
	// Introduced by enhancement 0001 (D2). #Module.#ctx.components projects this
	// block automatically; authors writing self-references inside a component's
	// `spec` body MUST go through `#ctx.components.<self-id>.dns.fqdn` because
	// `#names` is not in lexical scope under the spec definition block.
	#names: {
		resourceName: metadata.resourceName
		dns: {
			short: resourceName
			local: "\(resourceName).\(#release.namespace)"
			fqdn:  "\(resourceName).\(#release.namespace).svc.\(#release.clusterDomain)"
		}
	}

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
