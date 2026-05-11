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

		// Emit one K8s Secret per literal-bearing entry in spec.secrets.
		// #SecretK8sRef entries are skipped (resource pre-exists in cluster).
		// Output is a list of resources; the renderer dispatches on cue.Kind
		// and produces one Compiled per list element.
		let _relName = #context.#moduleReleaseMetadata.name
		let _compName = #context.#componentMetadata.name

		output: [
			for _, secret in _secrets

			// Naming scheme:
			//   opm-secrets component → {releaseName}-{secretName}
			//     (cross-component env var refs resolve to this shape)
			//   other components → {releaseName}-{componentName}-{secretName}
			//     (component-local volume refs resolve to this shape)
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
					if (_entry & string) != _|_ {
						(_dk): _entry
					}
					if (_entry & string) == _|_
					if _entry.value != _|_
					if _entry.secretName == _|_ {
						(_dk): _entry.value
					}
				}
			}

			if len(_literals) > 0 {
				k8scorev1.#Secret & {
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
			},
		]
	}
}
