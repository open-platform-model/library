package transformers

import (
	"list"
	k8sappsv1 "opmodel.dev/modules/opm/schemas/kubernetes/apps/v1@v1"
	c "opmodel.dev/core/v1alpha2@v1"
	res "opmodel.dev/modules/opm/resources"
	tr "opmodel.dev/modules/opm/traits"
)

// DaemonSetTransformer converts daemon workload components to Kubernetes DaemonSets
#DaemonSetTransformer: c.#ComponentTransformer & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/transformers"
		version:     "v1"
		name:        "daemonset-transformer"
		description: "Converts daemon workload components to Kubernetes DaemonSets"

		labels: {
			"core.opmodel.dev/workload-type": "daemon"
			"core.opmodel.dev/resource-type": "daemonset"
		}
	}

	// Required label to match daemon workloads
	requiredLabels: {
		"core.opmodel.dev/workload-type": "daemon"
	}

	// Required resources - Container MUST be present
	requiredResources: {
		"opmodel.dev/modules/opm/resources/container@v1": res.#ContainerResource
	}

	// Optional resources
	optionalResources: {
		"opmodel.dev/modules/opm/resources/volumes@v1": res.#VolumesResource
	}

	// No required traits
	requiredTraits: {}

	// Optional traits that enhance daemonset behavior
	// Note: NO Scaling trait - DaemonSets run one pod per node
	optionalTraits: {
		"opmodel.dev/modules/opm/traits/restart-policy@v1":     tr.#RestartPolicyTrait
		"opmodel.dev/modules/opm/traits/update-strategy@v1":    tr.#UpdateStrategyTrait
		"opmodel.dev/modules/opm/traits/sidecar-containers@v1": tr.#SidecarContainersTrait
		"opmodel.dev/modules/opm/traits/init-containers@v1":    tr.#InitContainersTrait
		"opmodel.dev/modules/opm/traits/security-context@v1":   tr.#SecurityContextTrait
		"opmodel.dev/modules/opm/traits/workload-identity@v1":  tr.#WorkloadIdentityTrait
		"opmodel.dev/modules/opm/traits/image-pull-secrets@v1": tr.#ImagePullSecretsTrait
		"opmodel.dev/modules/opm/traits/host-pid@v1":           tr.#HostPIDTrait
		"opmodel.dev/modules/opm/traits/host-ipc@v1":           tr.#HostIPCTrait
		"opmodel.dev/modules/opm/traits/host-network@v1":       tr.#HostNetworkTrait
		"opmodel.dev/modules/opm/traits/graceful-shutdown@v1":  tr.#GracefulShutdownTrait
	}

	#transform: {
		#component: _ // Unconstrained; validated by matching, not by transform signature
		#context:   c.#TransformerContext

		// Extract required Container resource
		_container: #component.spec.container

		// Apply defaults for optional traits (defaults inlined post-014).
		_restartPolicy: string | *"Always"
		if #component.spec.restartPolicy != _|_ {
			_restartPolicy: #component.spec.restartPolicy
		}

		// Extract update strategy with defaults
		_updateStrategy: *null | {
			if #component.spec.updateStrategy != _|_ {
				type: #component.spec.updateStrategy.type
				if #component.spec.updateStrategy.type == "RollingUpdate" {
					rollingUpdate: #component.spec.updateStrategy.rollingUpdate
				}
			}
		}

		// Build main container: base conversion via helper, unified with trait fields
		_mainContainer: (#ToK8sContainer & {"in": _container, #releasePrefix: #context.#moduleReleaseMetadata.name}).out

		// Build container list (main container + optional sidecars)
		_sidecarContainers: [...] | *[]
		if #component.spec.sidecarContainers != _|_ {
			_sidecarContainers: #component.spec.sidecarContainers
		}

		// Extract init containers with defaults
		_initContainers: [...]
		if #component.spec.initContainers != _|_ {
			_initContainers: #component.spec.initContainers
		}

		// Build DaemonSet resource
		output: k8sappsv1.#DaemonSet & {
			apiVersion: "apps/v1"
			kind:       "DaemonSet"
			metadata: {
				name:      "\(#context.#moduleReleaseMetadata.name)-\(#component.metadata.name)"
				namespace: #context.#moduleReleaseMetadata.namespace
				labels:    #context.labels
				// Include component annotations if present
				if len(#context.componentAnnotations) > 0 {
					annotations: #context.componentAnnotations
				}
			}
			spec: {
				selector: matchLabels: #context.componentLabels
				template: {
					metadata: labels: #context.componentLabels
					spec: {
						_convertedSidecars: (#ToK8sContainers & {"in": _sidecarContainers, #releasePrefix: #context.#moduleReleaseMetadata.name}).out
						containers: list.Concat([[_mainContainer], _convertedSidecars])

						if len(_initContainers) > 0 {
							initContainers: (#ToK8sContainers & {"in": _initContainers, #releasePrefix: #context.#moduleReleaseMetadata.name}).out
						}

						restartPolicy: _restartPolicy

						if #component.spec.hostNetwork != _|_ {
							hostNetwork: #component.spec.hostNetwork
						}

						if #component.spec.hostPid != _|_ {
							hostPID: #component.spec.hostPid
						}

						if #component.spec.hostIpc != _|_ {
							hostIPC: #component.spec.hostIpc
						}

						if #component.spec.securityContext != _|_ {
							let _sc = #component.spec.securityContext
							if _sc.runAsNonRoot != _|_ || _sc.runAsUser != _|_ || _sc.runAsGroup != _|_ || _sc.fsGroup != _|_ || _sc.supplementalGroups != _|_ {
								securityContext: {
									if _sc.runAsNonRoot != _|_ {
										runAsNonRoot: _sc.runAsNonRoot
									}
									if _sc.runAsUser != _|_ {
										runAsUser: _sc.runAsUser
									}
									if _sc.runAsGroup != _|_ {
										runAsGroup: _sc.runAsGroup
									}
									if _sc.fsGroup != _|_ {
										fsGroup: _sc.fsGroup
									}
									if _sc.supplementalGroups != _|_ {
										supplementalGroups: _sc.supplementalGroups
									}
								}
							}
						}

						if #component.spec.workloadIdentity != _|_ {
							serviceAccountName: #component.spec.workloadIdentity.name
						}

						// Image pull secrets: pod-level registry credentials
						if #component.spec.imagePullSecrets != _|_ {
							imagePullSecrets: #component.spec.imagePullSecrets
						}

						// Volumes: convert OPM volume specs to Kubernetes volume specs
						if #component.spec.volumes != _|_ {
							volumes: (#ToK8sVolumes & {"in": #component.spec.volumes, #releasePrefix: "\(#context.#moduleReleaseMetadata.name)-\(#context.#componentMetadata.name)"}).out
						}

						// Graceful shutdown: pod-level termination grace period
						if #component.spec.gracefulShutdown != _|_ {
							terminationGracePeriodSeconds: #component.spec.gracefulShutdown.terminationGracePeriodSeconds
						}
					}
				}

				if _updateStrategy != null {
					updateStrategy: _updateStrategy
				}
			}
		}
	}
}
