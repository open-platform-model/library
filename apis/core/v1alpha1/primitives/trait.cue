package primitives

import (
	"strings"
	t "opmodel.dev/core/v1alpha1/types@v1"
)

// #Trait: Defines additional behavior or characteristics that can be attached to components.
#Trait: {
	apiVersion: "opmodel.dev/core/v1alpha1"
	kind:       "Trait"

	metadata: {
		modulePath!: t.#ModulePathType   // Example: "opmodel.dev/opm/traits/workload"
		version!:    t.#MajorVersionType // Example: "v1"
		name!:       t.#NameType         // Example: "scaling"
		#definitionName: (t.#KebabToPascal & {"in": name}).out

		fqn: t.#FQNType & "\(modulePath)/\(name)@\(version)" // Example: "opmodel.dev/opm/traits/workload/scaling@v1"

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

	// Resources that this trait can be applied to (full references)
	appliesTo!: [...#Resource]
}

#TraitMap: [string]: _
