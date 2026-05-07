package v1alpha2

import (
	"strings"
)

// #Resource: Defines a resource of deployment within the system.
// Resources represent deployable components, services or resources that can be instantiated and managed independently.
#Resource: {
	apiVersion: #ApiVersion
	kind:       "Resource"

	metadata: {
		name!: #NameType // Example: "container"
		#definitionName: (#KebabToPascal & {"in": name}).out

		modulePath!: #ModulePathType                               // Example: "opmodel.dev/opm/resources/workload"
		version!:    #MajorVersionType                             // Example: "v1"
		fqn:         #FQNType & "\(modulePath)/\(name)@\(version)" // Example: "opmodel.dev/opm/resources/workload/container@v1"

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
}

#ResourceMap: [string]: #Resource
