// OPM core catalog module — publishes Resources, Traits, Blueprints, and
// ComponentTransformers via #defines. Mirrors the modules-repo authoring
// pattern (bare m.#Module + metadata + #config + debugValues) plus the new
// 014 #defines slot.
package opm

import (
	m "opmodel.dev/core/v1alpha2@v1"
	res "opmodel.dev/modules/opm/resources"
	tr "opmodel.dev/modules/opm/traits"
	opm_transformers "opmodel.dev/modules/opm/transformers"
)

m.#Module

metadata: {
	modulePath:  "opmodel.dev/modules"
	name:        "opm"
	version:     "1.0.4"
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
		"opmodel.dev/modules/opm/resources/config-maps@v1": res.#ConfigMapsResource
		"opmodel.dev/modules/opm/resources/secrets@v1":     res.#SecretsResource
		// extension
		"opmodel.dev/modules/opm/resources/crds@v1": res.#CRDsResource
		// security
		"opmodel.dev/modules/opm/resources/role@v1":            res.#RoleResource
		"opmodel.dev/modules/opm/resources/service-account@v1": res.#ServiceAccountResource
		// storage
		"opmodel.dev/modules/opm/resources/volumes@v1": res.#VolumesResource
		// workload
		"opmodel.dev/modules/opm/resources/container@v1": res.#ContainerResource
	}

	traits: {
		// network
		"opmodel.dev/modules/opm/traits/expose@v1":       tr.#ExposeTrait
		"opmodel.dev/modules/opm/traits/grpc-route@v1":   tr.#GrpcRouteTrait
		"opmodel.dev/modules/opm/traits/http-route@v1":   tr.#HttpRouteTrait
		"opmodel.dev/modules/opm/traits/tcp-route@v1":    tr.#TcpRouteTrait
		"opmodel.dev/modules/opm/traits/tls-route@v1":    tr.#TlsRouteTrait
		"opmodel.dev/modules/opm/traits/host-network@v1": tr.#HostNetworkTrait
		// security
		"opmodel.dev/modules/opm/traits/encryption@v1":         tr.#EncryptionConfigTrait
		"opmodel.dev/modules/opm/traits/host-ipc@v1":           tr.#HostIPCTrait
		"opmodel.dev/modules/opm/traits/host-pid@v1":           tr.#HostPIDTrait
		"opmodel.dev/modules/opm/traits/image-pull-secrets@v1": tr.#ImagePullSecretsTrait
		"opmodel.dev/modules/opm/traits/security-context@v1":   tr.#SecurityContextTrait
		"opmodel.dev/modules/opm/traits/workload-identity@v1":  tr.#WorkloadIdentityTrait
		// workload
		"opmodel.dev/modules/opm/traits/scaling@v1":            tr.#ScalingTrait
		"opmodel.dev/modules/opm/traits/cron-job-config@v1":    tr.#CronJobConfigTrait
		"opmodel.dev/modules/opm/traits/job-config@v1":         tr.#JobConfigTrait
		"opmodel.dev/modules/opm/traits/disruption-budget@v1":  tr.#DisruptionBudgetTrait
		"opmodel.dev/modules/opm/traits/graceful-shutdown@v1":  tr.#GracefulShutdownTrait
		"opmodel.dev/modules/opm/traits/init-containers@v1":    tr.#InitContainersTrait
		"opmodel.dev/modules/opm/traits/placement@v1":          tr.#PlacementTrait
		"opmodel.dev/modules/opm/traits/restart-policy@v1":     tr.#RestartPolicyTrait
		"opmodel.dev/modules/opm/traits/sidecar-containers@v1": tr.#SidecarContainersTrait
		"opmodel.dev/modules/opm/traits/sizing@v1":             tr.#SizingTrait
		"opmodel.dev/modules/opm/traits/update-strategy@v1":    tr.#UpdateStrategyTrait
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
