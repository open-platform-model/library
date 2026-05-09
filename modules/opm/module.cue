// OPM core catalog module — publishes Resources, Traits, Blueprints, and
// ComponentTransformers via #defines. Mirrors the modules-repo authoring
// pattern (bare m.#Module + metadata + #config + debugValues) plus the new
// 014 #defines slot.
package opm

import (
	m "opmodel.dev/core/v1alpha2@v1"
	resource_config "opmodel.dev/modules/opm/resources/config"
	resource_extension "opmodel.dev/modules/opm/resources/extension"
	resource_security "opmodel.dev/modules/opm/resources/security"
	resource_storage "opmodel.dev/modules/opm/resources/storage"
	resource_workload "opmodel.dev/modules/opm/resources/workload"
	trait_network "opmodel.dev/modules/opm/traits/network"
	trait_security "opmodel.dev/modules/opm/traits/security"
	trait_workload "opmodel.dev/modules/opm/traits/workload"
	opm_transformers "opmodel.dev/modules/opm/transformers"
)

m.#Module

metadata: {
	modulePath:  "opmodel.dev/modules"
	name:        "opm"
	version:     "1.0.0"
	description: "OPM core catalog — Resources, Traits, Blueprints, and ComponentTransformers published via #defines"
}

// opm is a pure catalog module — it publishes primitives but does not declare
// components or runtime values. Downstream modules reference the published
// FQNs from #defines.
#config: {}
debugValues: {}

#defines: {
	resources: {
		// config
		"opmodel.dev/modules/opm/resources/config/config-maps@v1": resource_config.#ConfigMapsResource
		"opmodel.dev/modules/opm/resources/config/secrets@v1":     resource_config.#SecretsResource
		// extension
		"opmodel.dev/modules/opm/resources/extension/crds@v1": resource_extension.#CRDsResource
		// security
		"opmodel.dev/modules/opm/resources/security/role@v1":            resource_security.#RoleResource
		"opmodel.dev/modules/opm/resources/security/service-account@v1": resource_security.#ServiceAccountResource
		// storage
		"opmodel.dev/modules/opm/resources/storage/volumes@v1": resource_storage.#VolumesResource
		// workload
		"opmodel.dev/modules/opm/resources/workload/container@v1": resource_workload.#ContainerResource
	}

	traits: {
		// network
		"opmodel.dev/modules/opm/traits/network/expose@v1":       trait_network.#ExposeTrait
		"opmodel.dev/modules/opm/traits/network/grpc-route@v1":   trait_network.#GrpcRouteTrait
		"opmodel.dev/modules/opm/traits/network/http-route@v1":   trait_network.#HttpRouteTrait
		"opmodel.dev/modules/opm/traits/network/tcp-route@v1":    trait_network.#TcpRouteTrait
		"opmodel.dev/modules/opm/traits/network/tls-route@v1":    trait_network.#TlsRouteTrait
		"opmodel.dev/modules/opm/traits/network/host-network@v1": trait_network.#HostNetworkTrait
		// security
		"opmodel.dev/modules/opm/traits/security/encryption@v1":         trait_security.#EncryptionConfigTrait
		"opmodel.dev/modules/opm/traits/security/host-ipc@v1":           trait_security.#HostIPCTrait
		"opmodel.dev/modules/opm/traits/security/host-pid@v1":           trait_security.#HostPIDTrait
		"opmodel.dev/modules/opm/traits/security/image-pull-secrets@v1": trait_security.#ImagePullSecretsTrait
		"opmodel.dev/modules/opm/traits/security/security-context@v1":   trait_security.#SecurityContextTrait
		"opmodel.dev/modules/opm/traits/security/workload-identity@v1":  trait_security.#WorkloadIdentityTrait
		// workload
		"opmodel.dev/modules/opm/traits/workload/scaling@v1":            trait_workload.#ScalingTrait
		"opmodel.dev/modules/opm/traits/workload/cron-job-config@v1":    trait_workload.#CronJobConfigTrait
		"opmodel.dev/modules/opm/traits/workload/job-config@v1":         trait_workload.#JobConfigTrait
		"opmodel.dev/modules/opm/traits/workload/disruption-budget@v1":  trait_workload.#DisruptionBudgetTrait
		"opmodel.dev/modules/opm/traits/workload/graceful-shutdown@v1":  trait_workload.#GracefulShutdownTrait
		"opmodel.dev/modules/opm/traits/workload/init-containers@v1":    trait_workload.#InitContainersTrait
		"opmodel.dev/modules/opm/traits/workload/placement@v1":          trait_workload.#PlacementTrait
		"opmodel.dev/modules/opm/traits/workload/restart-policy@v1":     trait_workload.#RestartPolicyTrait
		"opmodel.dev/modules/opm/traits/workload/sidecar-containers@v1": trait_workload.#SidecarContainersTrait
		"opmodel.dev/modules/opm/traits/workload/sizing@v1":             trait_workload.#SizingTrait
		"opmodel.dev/modules/opm/traits/workload/update-strategy@v1":    trait_workload.#UpdateStrategyTrait
	}

	transformers: {
		"opmodel.dev/modules/opm/transformers/deployment-transformer@v1":              opm_transformers.#DeploymentTransformer
		"opmodel.dev/modules/opm/transformers/statefulset-transformer@v1":             opm_transformers.#StatefulsetTransformer
		"opmodel.dev/modules/opm/transformers/daemonset-transformer@v1":               opm_transformers.#DaemonSetTransformer
		"opmodel.dev/modules/opm/transformers/job-transformer@v1":                     opm_transformers.#JobTransformer
		"opmodel.dev/modules/opm/transformers/cronjob-transformer@v1":                 opm_transformers.#CronJobTransformer
		"opmodel.dev/modules/opm/transformers/service-transformer@v1":                 opm_transformers.#ServiceTransformer
		"opmodel.dev/modules/opm/transformers/configmap-transformer@v1":               opm_transformers.#ConfigMapTransformer
		"opmodel.dev/modules/opm/transformers/secret-transformer@v1":                  opm_transformers.#SecretTransformer
		"opmodel.dev/modules/opm/transformers/role-transformer@v1":                    opm_transformers.#RoleTransformer
		"opmodel.dev/modules/opm/transformers/serviceaccount-resource-transformer@v1": opm_transformers.#ServiceAccountResourceTransformer
		"opmodel.dev/modules/opm/transformers/pvc-transformer@v1":                     opm_transformers.#PVCTransformer
		"opmodel.dev/modules/opm/transformers/crd-transformer@v1":                     opm_transformers.#CRDTransformer
	}
}
