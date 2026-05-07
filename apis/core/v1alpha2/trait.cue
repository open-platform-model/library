package v1alpha2

import (
	"strings"
)

// #Trait: Defines additional behavior or characteristics that can be attached to components.
#Trait: {
	apiVersion: #ApiVersion
	kind:       "Trait"

	metadata: {
		name!: #NameType // Example: "scaling"
		#definitionName: (#KebabToPascal & {"in": name}).out

		modulePath!: #ModulePathType                               // Example: "opmodel.dev/opm/traits/workload"
		version!:    #MajorVersionType                             // Example: "v1"	
		fqn:         #FQNType & "\(modulePath)/\(name)@\(version)" // Example: "opmodel.dev/opm/traits/workload/scaling@v1"

		// Human-readable description of the definition
		description?: string

		// Optional metadata labels for categorization and filtering
		// Labels are used by OPM for definition selection and matching
		// Example: {"core.opmodel.dev/workload-type": "stateless"}
		labels?: #LabelsAnnotationsType

		// Optional metadata annotations for definition behavior hints (not used for categorization)
		// Annotations provide additional metadata but are not used for selection
		annotations?: #LabelsAnnotationsType
	}

	// MUST be an OpenAPIv3 compatible schema
	// The field and schema exposed by this definition
	spec!: (strings.ToCamel(metadata.#definitionName)): _

	// Resources that this trait can be applied to (full references)
	appliesTo!: [...#Resource]
}

#TraitMap: [string]: #Trait
