package provider

import (
	"list"
	t "opmodel.dev/core/v1alpha1/types@v1"
	transformer "opmodel.dev/core/v1alpha1/transformer@v1"
)

#Provider: {
	apiVersion: "opmodel.dev/core/v1alpha1"
	kind:       "Provider"
	metadata: {
		name:        t.#NameType // The name of the provider
		description: string      // A brief description of the provider
		version:     string      // The version of the provider

		// Labels for provider categorization and compatibility
		// Example: {"core.opmodel.dev/format": "kubernetes"}
		labels?:      t.#LabelsAnnotationsType
		annotations?: t.#LabelsAnnotationsType
	}

	// Transformer registry - maps platform resources to transformers
	// Example:
	// #transformers: {
	// 	"k8s.io/api/apps/v1.Deployment": #DeploymentTransformer
	// 	"k8s.io/api/apps/v1.StatefulSet": #StatefulsetTransformer
	// }
	#transformers: transformer.#TransformerMap

	// All resources, traits declared by transformers
	// Extract FQNs from the map keys
	#declaredResources: list.FlattenN([
		for _, t in #transformers {
			list.Concat([
				[for fqn, _ in t.requiredResources {fqn}],
				[for fqn, _ in t.optionalResources {fqn}],
			])
		},
	], 1)

	#declaredTraits: list.FlattenN([
		for _, t in #transformers {
			list.Concat([
				[for fqn, _ in t.requiredTraits {fqn}],
				[for fqn, _ in t.optionalTraits {fqn}],
			])
		},
	], 1)

	#declaredDefinitions: list.Concat([#declaredResources, #declaredTraits])
	...
}
