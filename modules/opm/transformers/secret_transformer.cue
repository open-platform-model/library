package transformers

import (
	c "opmodel.dev/core/v1alpha2@v1"
	res "opmodel.dev/modules/opm/resources"
	k8scorev1 "opmodel.dev/modules/opm/schemas/kubernetes/core/v1@v1"
)

// SecretTransformer converts Secrets resources to Kubernetes Secrets.
//
// Variant dispatch per data entry:
//   #SecretLiteral -> include in K8s Secret stringData
//   #SecretK8sRef  -> skip (resource already exists in cluster)
//
// Mixed variants within a single secret group are supported: literal entries
// create a K8s Secret, K8s refs are skipped.
#SecretTransformer: c.#ComponentTransformer & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/transformers"
		version:     "v1"
		name:        "secret-transformer"
		description: "Converts Secrets resources to Kubernetes Secrets"

		labels: {
			"core.opmodel.dev/resource-category": "config"
			"core.opmodel.dev/resource-type":     "secret"
		}
	}

	requiredLabels: {}

	// Required resources - Secrets MUST be present
	requiredResources: {
		"opmodel.dev/modules/opm/resources/secrets@v1": res.#SecretsResource
	}

	optionalResources: {}
	requiredTraits: {}
	optionalTraits: {}

	#transform: {
		#component: _
		#context:   c.#TransformerContext

		_secrets: #component.spec.secrets

		output: {
			for _secretName, secret in _secrets {
				// Compute the deterministic K8s resource name.
				//
				// Two naming schemes based on whether this is the synthetic opm-secrets
				// component (which holds auto-discovered cross-component #Secret fields)
				// or a regular component that defines its own spec.secrets locally.
				//
				// opm-secrets: {releaseName}-{secretName}
				//   Auto-discovered secrets are referenced by env vars in OTHER components
				//   via {$secretName}. #ToK8sContainer resolves secretKeyRef as
				//   {releaseName}-{$secretName}, so the K8s name must match that scheme.
				//
				// other components: {releaseName}-{componentName}-{secretName}
				//   Component-local secrets are only mounted as volumes within the same
				//   component. #ToK8sVolumes resolves secretName as
				//   {releaseName}-{componentName}-{from.name}, so the K8s name must match.
				let _relName = #context.#moduleReleaseMetadata.name
				let _compName = #context.#componentMetadata.name

				// opm-secrets: {releaseName}-{secretName} (cross-component env var refs)
				// other:        {releaseName}-{componentName}-{secretName} (local volume refs)
				let _baseName = {
					if _compName == "opm-secrets" {out: "\(_relName)-\(secret.name)"}
					if _compName != "opm-secrets" {out: "\(_relName)-\(_compName)-\(secret.name)"}
				}.out

				let _k8sName = (res.#SecretImmutableName & {
					baseName:  _baseName
					data:      secret.data
					immutable: secret.immutable
				}).out

				// Collect literal entries for K8s Secret stringData.
				// Handles three cases:
				//   plain string    -> include directly
				//   #SecretLiteral  -> include via .value
				//   #SecretK8sRef   -> skip (resource pre-exists in cluster)
				let _literals = {
					for _dk, _entry in secret.data {
						// Plain string (manually defined secret, e.g. computed config)
						if (_entry & string) != _|_ {
							(_dk): _entry
						}

						// #SecretLiteral
						if (_entry & string) == _|_
						if _entry.value != _|_
						if _entry.secretName == _|_ {
							(_dk): _entry.value
						}
					}
				}

				// Emit K8s Secret if there are any literal entries
				if len(_literals) > 0 {
					"\(_k8sName)": k8scorev1.#Secret & {
						apiVersion: "v1"
						kind:       "Secret"
						metadata: {
							name:      _k8sName
							namespace: #context.#moduleReleaseMetadata.namespace
							labels:    #context.labels
							if len(#context.componentAnnotations) > 0 {
								annotations: #context.componentAnnotations
							}
						}
						type: secret.type
						if secret.immutable == true {
							immutable: true
						}
						stringData: _literals
					}
				}

				// #SecretK8sRef entries: nothing emitted (resource pre-exists)
			}
		}
	}
}
