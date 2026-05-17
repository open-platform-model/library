package v1alpha2

// Trimmed #Component — only the surface 004 needs: metadata.resourceName?
// override (D13) and the per-component #names injection slot (D32).
// Production #Component carries #resources / #traits / #blueprints / spec;
// none of that is relevant to context-builder mechanics, so it's elided.
#Component: {
	apiVersion: #ApiVersion
	kind:       "Component"

	metadata: {
		name!:         #NameType
		resourceName?: #NameType
		labels?:       #LabelsAnnotationsType
		annotations?:  #LabelsAnnotationsType
	}

	// Per-component computed names; equal to #ctx.runtime.components[<key>]
	// after #ContextBuilder runs.
	#names: #ComponentNames

	// Stub spec slot so component bodies can write `spec: x: #names.dns.fqdn`
	// when an experiment case wants to demonstrate self-reads. Optional so
	// cases that don't use spec don't bottom out under `cue vet -c`.
	spec?: _
}
