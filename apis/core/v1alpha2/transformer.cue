package transformer

import (
	"strings"
)

// #Transformer: Declares how to convert OPM components into platform-specific resources.
//
// Transformers use label-based matching to determine which components they can handle.
// A transformer matches a component when ALL of the following are true:
//  1. ALL requiredLabels are present on the component with matching values
//  2. ALL requiredResources FQNs exist in component #resources
//  3. ALL requiredTraits FQNs exist in component #traits
//
// Component labels are inherited from the union of labels from all attached
// #resources, #traits, and #policies definitions.
#Transformer: {
	apiVersion: #ApiVersion
	kind:       "Transformer"

	metadata: {
		modulePath!: #ModulePathType   // Example: "opmodel.dev/opm/transformers/kubernetes"
		version!:    #MajorVersionType // Example: "v0"
		name!:       #NameType         // Example: "deployment-transformer"
		#definitionName: (#KebabToPascal & {"in": name}).out

		fqn: #FQNType & "\(modulePath)/\(name)@\(version)" // Example: "opmodel.dev/opm/transformers/kubernetes/deployment-transformer@v0"

		description!: string // A brief description of what this transformer produces

		// Labels for categorizing this transformer (not used for matching)
		labels?: #LabelsAnnotationsType

		// Annotations for additional transformer metadata
		annotations?: #LabelsAnnotationsType
	}

	// Labels that a component MUST have to match this transformer.
	// Component labels are inherited from the union of labels from all attached
	// #resources, #traits, and #policies.
	//
	// Example: A DeploymentTransformer requires stateless workloads:
	//   requiredLabels: {"core.opmodel.dev/workload-type": "stateless"}
	//
	// The Container resource defines this label, so components with Container
	// will have it. Transformers requiring "stateful" won't match.
	requiredLabels?: #LabelsAnnotationsType

	// Labels optionally used by this transformer - component MAY include these
	// If not provided, defaults from the definition can be used
	optionalLabels?: #LabelsAnnotationsType

	// Resources required by this transformer - component MUST include these
	// Map key is the FQN, value is the Resource definition (provides access to #defaults)
	requiredResources: [string]: #Resource

	// Resources optionally used by this transformer - component MAY include these
	// If not provided, defaults from the definition can be used
	optionalResources: [string]: #Resource

	// Traits required by this transformer - component MUST include these
	// Map key is the FQN, value is the Trait definition (provides access to #defaults)
	requiredTraits: [string]: #Trait

	// Traits optionally used by this transformer - component MAY include these
	// If not provided, defaults from the definition can be used
	optionalTraits: [string]: #Trait

	// Transform function
	// IMPORTANT: output must be a single resource
	#transform: {
		#component: _ // Unconstrained; validated by matching, not by the transform signature
		#context:   #TransformerContext

		output: {...} // Must be a single provider-specific resource
	}
}

// Map of transformers by fully qualified name
#TransformerMap: [#FQNType]: #Transformer

// Provider context passed to transformers
#TransformerContext: {
	#moduleReleaseMetadata: {
		name!:        #NameType
		namespace!:   #NameType // Required for releases (target environment)
		fqn:          string
		version:      string
		uuid:         #UUIDType
		labels?:      #LabelsAnnotationsType
		annotations?: #LabelsAnnotationsType
	}

	#componentMetadata: {
		name!:        #NameType
		labels?:      #LabelsAnnotationsType
		annotations?: #LabelsAnnotationsType
	}

	// Identity of the runtime that is executing this transform. Mandatory — CUE
	// evaluation fails if the runtime forgets to fill this. The value is stamped
	// verbatim onto every rendered resource as "app.kubernetes.io/managed-by".
	// Runtimes are expected to fill this with their own identity (e.g. "opm-cli"
	// for the CLI, "opm-controller" for the operator).
	#runtimeName!: #NameType

	// Labels and annotations. These are inherited from the component and module metadata.
	//
	// - moduleLabels: labels from #moduleReleaseMetadata.labels (if defined)
	// - moduleAnnotations: annotations from #moduleReleaseMetadata.annotations (if defined)
	// - componentLabels: labels from #componentMetadata.labels (if defined) + "app.kubernetes.io/name" = component name
	// - componentAnnotations: annotations from #componentMetadata.annotations (if defined)
	// - controllerLabels: standard controller labels (including managed-by = #runtimeName)
	moduleLabels: {
		if #moduleReleaseMetadata.labels != _|_ {
			for k, v in #moduleReleaseMetadata.labels {
				(k): "\(v)"
			}
		}
	}

	moduleAnnotations: {
		if #moduleReleaseMetadata.annotations != _|_ {
			for k, v in #moduleReleaseMetadata.annotations {
				(k): "\(v)"
			}
		}
	}

	componentLabels: {
		"app.kubernetes.io/name":          #componentMetadata.name
		"module-release.opmodel.dev/name": #moduleReleaseMetadata.name
		if #componentMetadata.labels != _|_ {
			for k, v in #componentMetadata.labels {
				if !strings.HasPrefix(k, "transformer.opmodel.dev/") {
					(k): "\(v)"
				}
			}
		}
	}

	componentAnnotations: {
		if #componentMetadata.annotations != _|_ {
			for k, v in #componentMetadata.annotations {
				if !strings.HasPrefix(k, "transformer.opmodel.dev/") {
					(k): "\(v)"
				}
			}
		}
	}

	controllerLabels: {
		"app.kubernetes.io/managed-by": #runtimeName
		"app.kubernetes.io/name":       #componentMetadata.name
		"app.kubernetes.io/instance":   #componentMetadata.name
	}

	// Final labels and annotations applied to the output resource
	labels: {[string]: string}
	labels: {
		for k, v in moduleLabels {
			(k): "\(v)"
		}
		for k, v in componentLabels {
			(k): "\(v)"
		}
		for k, v in controllerLabels {
			(k): "\(v)"
		}
		...
	}
	annotations: {[string]: string}
	annotations: {
		for k, v in moduleAnnotations {
			(k): "\(v)"
		}
		for k, v in componentAnnotations {
			(k): "\(v)"
		}
		...
	}
}
