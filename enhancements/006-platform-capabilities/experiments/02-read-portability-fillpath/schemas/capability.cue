package v1alpha2

// #Capability — FQN-identified, schema-bearing context primitive.
// Sibling to #Resource and #Trait (same identity + schema pattern), but a
// render INPUT (a platform provides it, a module reads it) rather than a
// render OUTPUT (no transformer renders a #Capability).
//
// Copied from enhancements/006-platform-capabilities/03-schema.md §#Capability.
#Capability: {
	apiVersion: #ApiVersion
	kind:       "Capability"

	metadata: {
		name!: #NameType
		#definitionName: (#KebabToPascal & {"in": name}).out

		modulePath!: #ModulePathType
		version!:    #MajorVersionType
		fqn:         #FQNType & "\(modulePath)/\(name)@\(version)"

		description?: string
		labels?:      #LabelsAnnotationsType
		annotations?: #LabelsAnnotationsType
	}

	// The interface. MUST be an OpenAPIv3-compatible schema.
	// Unwrapped — capabilities are addressed by FQN in #consumes, never
	// flattened into a component's merged spec (006 D2).
	spec!: _
}

#CapabilityMap: [#FQNType]: #Capability
