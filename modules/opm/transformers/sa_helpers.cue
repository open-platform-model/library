package transformers

import (
	c "opmodel.dev/core/v1alpha2@v1"
	k8scorev1 "opmodel.dev/modules/opm/schemas/kubernetes/core/v1@v1"
)

// #ToK8sServiceAccount converts an OPM identity spec (either #WorkloadIdentitySchema
// or #ServiceAccountSchema — both share the same shape) to a Kubernetes ServiceAccount.
//
// Both schema types provide `name!: string` and `automountToken?: bool`, so either
// can be passed directly without conversion.
//
// Usage:
//   (#ToK8sServiceAccount & {"in": _identity, context: #context}).out
#ToK8sServiceAccount: {
	X="in": {
		name!:           string
		automountToken?: bool
	}
	context: c.#TransformerContext

	out: k8scorev1.#ServiceAccount & {
		apiVersion: "v1"
		kind:       "ServiceAccount"
		metadata: {
			name:      X.name
			namespace: context.#moduleReleaseMetadata.namespace
			labels:    context.labels
			if len(context.componentAnnotations) > 0 {
				annotations: context.componentAnnotations
			}
		}
		automountServiceAccountToken: X.automountToken
	}
}

/////////////////////////////////////////////////////////////////
//// Test Data
/////////////////////////////////////////////////////////////////

_testToK8sServiceAccount: (#ToK8sServiceAccount & {
	"in": {
		name:           "ci-bot"
		automountToken: false
	}
	context: {
		namespace: "ci"
		labels: app: "ci-bot"
		componentAnnotations: {}
	}
}).out
