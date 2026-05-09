package security

import (
	c "opmodel.dev/core/v1alpha2@v1"
	schemas "opmodel.dev/modules/opm/schemas"
	res_workload "opmodel.dev/modules/opm/resources/workload"
)

// References pre-existing K8s Secrets (type kubernetes.io/dockerconfigjson)
// that the kubelet uses to authenticate to private container registries when
// pulling images for any container in the pod.
#ImagePullSecretsTrait: c.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/traits/security"
		version:     "v1"
		name:        "image-pull-secrets"
		description: "Reference K8s Secrets used to authenticate to private container registries"
		labels: {
			"trait.opmodel.dev/category": "security"
		}
	}

	appliesTo: [res_workload.#ContainerResource]

	spec: imagePullSecrets: schemas.#ImagePullSecretsSchema
}
