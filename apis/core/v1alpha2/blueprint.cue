package v1alpha2

import (
	"strings"
)

// #Blueprint: Defines a reusable blueprint
// that composes resources and traits into a higher-level abstraction.
// Blueprints enable standardized configurations for common use cases.
#Blueprint: {
	apiVersion: #ApiVersion
	kind:       "Blueprint"

	metadata: {
		name!: #NameType // Example: "stateless-workload"
		#definitionName: (#KebabToPascal & {"in": name}).out

		modulePath!: #ModulePathType                               // Example: "opmodel.dev/opm/blueprints/workload"
		version!:    #MajorVersionType                             // Example: "v1"
		fqn:         #FQNType & "\(modulePath)/\(name)@\(version)" // Example: "opmodel.dev/opm/blueprints/workload/stateless-workload@v1"

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

	// Resources that compose this blueprint (full references)
	composedResources!: [...#Resource]

	// Traits that compose this blueprint (full references)
	composedTraits?: [...#Trait]

	// MUST be an OpenAPIv3 compatible schema
	// The field and schema exposed by this definition
	spec!: (strings.ToCamel(metadata.#definitionName)): _
}

#BlueprintMap: [string]: #Blueprint
