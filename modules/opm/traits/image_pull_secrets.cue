package traits

import (
	id "opmodel.dev/catalogs/opm/identity"
	c "opmodel.dev/core@v0"
	res "opmodel.dev/catalogs/opm/resources"
)

// References pre-existing K8s Secrets (type kubernetes.io/dockerconfigjson)
// that the kubelet uses to authenticate to private container registries when
// pulling images for any container in the pod.
#ImagePullSecretsTrait: c.#Trait & {
	metadata: {
		modulePath:  "\(id.ModulePath)/traits"
		version:     id.Version
		name:        "image-pull-secrets"
		description: "Reference K8s Secrets used to authenticate to private container registries"
		labels: {
			"trait.opmodel.dev/category": "security"
		}
	}

	appliesTo: [res.#ContainerResource]

	spec: imagePullSecrets: #ImagePullSecretsSchema
}

#ImagePullSecrets: c.#Component & {
	#traits: (#ImagePullSecretsTrait.metadata.fqn): #ImagePullSecretsTrait
}

// References to pre-existing K8s Secrets. Each entry is a LocalObjectReference
// to a Secret of type kubernetes.io/dockerconfigjson in the pod's namespace.
// OPM does not create these — they must already exist (typically managed by
// an external secret operator or platform team).
#ImagePullSecretsSchema: [...{name!: string}]
