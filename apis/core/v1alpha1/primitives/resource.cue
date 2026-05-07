package primitives

import (
	"strings"
	t "opmodel.dev/core/v1alpha1/types@v1"
)

// #Resource: Defines a resource of deployment within the system.
// Resources represent deployable components, services or resources that can be instantiated and managed independently.
#Resource: {
	apiVersion: "opmodel.dev/core/v1alpha1"
	kind:       "Resource"

	metadata: {
		modulePath!: t.#ModulePathType   // Example: "opmodel.dev/opm/resources/workload"
		version!:    t.#MajorVersionType // Example: "v1"
		name!:       t.#NameType         // Example: "container"
		#definitionName: (t.#KebabToPascal & {"in": name}).out

		fqn: t.#FQNType & "\(modulePath)/\(name)@\(version)" // Example: "opmodel.dev/opm/resources/workload/container@v1"

		// Human-readable description of the definition
		description?: string

		// Optional metadata labels for categorization and filtering
		// Labels are used by OPM for definition selection and matching
		// Example: {"core.opmodel.dev/workload-type": "stateless"}
		labels?: t.#LabelsAnnotationsType

		// Optional metadata annotations for definition behavior hints (not used for categorization)
		// Annotations provide additional metadata but are not used for selection
		annotations?: t.#LabelsAnnotationsType
	}

	// MUST be an OpenAPIv3 compatible schema
	// The field and schema exposed by this definition
	spec!: (strings.ToCamel(metadata.#definitionName)): _
}

#ResourceMap: [string]: _
