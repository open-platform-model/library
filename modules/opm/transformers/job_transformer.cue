package transformers

import (
	"list"
	k8sbatchv1 "opmodel.dev/modules/opm/schemas/kubernetes/batch/v1@v1"
	c "opmodel.dev/core/v1alpha2@v1"
	res "opmodel.dev/modules/opm/resources"
	tr "opmodel.dev/modules/opm/traits"
)

// JobTransformer converts task workload components to Kubernetes Jobs
#JobTransformer: c.#ComponentTransformer & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/transformers"
		version:     "v1"
		name:        "job-transformer"
		description: "Converts task workload components to Kubernetes Jobs"

		labels: {
			"core.opmodel.dev/workload-type": "task"
			"core.opmodel.dev/resource-type": "job"
		}
	}

	// Required label to match task workloads
	requiredLabels: {
		"core.opmodel.dev/workload-type": "task"
	}

	// Required resources - Container MUST be present
	requiredResources: {
		"opmodel.dev/modules/opm/resources/container@v1": res.#ContainerResource
	}

	// Optional resources
	optionalResources: {
		"opmodel.dev/modules/opm/resources/volumes@v1": res.#VolumesResource
	}

	// Required traits - JobConfig is mandatory for Job
	requiredTraits: {
		"opmodel.dev/modules/opm/traits/job-config@v1": tr.#JobConfigTrait
	}

	// Optional traits
	optionalTraits: {
		"opmodel.dev/modules/opm/traits/restart-policy@v1":     tr.#RestartPolicyTrait
		"opmodel.dev/modules/opm/traits/sidecar-containers@v1": tr.#SidecarContainersTrait
		"opmodel.dev/modules/opm/traits/init-containers@v1":    tr.#InitContainersTrait
		"opmodel.dev/modules/opm/traits/security-context@v1":   tr.#SecurityContextTrait
		"opmodel.dev/modules/opm/traits/workload-identity@v1":  tr.#WorkloadIdentityTrait
		"opmodel.dev/modules/opm/traits/image-pull-secrets@v1": tr.#ImagePullSecretsTrait
		"opmodel.dev/modules/opm/traits/host-pid@v1":           tr.#HostPIDTrait
		"opmodel.dev/modules/opm/traits/host-ipc@v1":           tr.#HostIPCTrait
		"opmodel.dev/modules/opm/traits/graceful-shutdown@v1":  tr.#GracefulShutdownTrait
	}

	#transform: {
		#component: _ // Unconstrained; validated by matching, not by transform signature
		#context:   c.#TransformerContext

		// Extract required Container resource
		_container: #component.spec.container

		// Extract required JobConfig trait
		_jobConfig: #component.spec.jobConfig

		// Apply defaults for optional RestartPolicy trait
		_restartPolicy: *"OnFailure" | string
		if #component.spec.restartPolicy != _|_ {
			_restartPolicy: #component.spec.restartPolicy
		}

		// Build main container: base conversion via helper, unified with trait fields
		_mainContainer: (#ToK8sContainer & {"in": _container, #releasePrefix: #context.#moduleReleaseMetadata.name}).out

		// Extract optional sidecar and init containers (defaults inlined post-014).
		_sidecarContainers: [...] | *[]
		if #component.spec.sidecarContainers != _|_ {
			_sidecarContainers: #component.spec.sidecarContainers
		}

		_initContainers: [...] | *[]
		if #component.spec.initContainers != _|_ {
			_initContainers: #component.spec.initContainers
		}

		output: k8sbatchv1.#Job & {
			apiVersion: "batch/v1"
			kind:       "Job"
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
				completions: int | *1
				if _jobConfig.completions != _|_ {
					completions: _jobConfig.completions
				}

				parallelism: int | *1
				if _jobConfig.parallelism != _|_ {
					parallelism: _jobConfig.parallelism
				}

				backoffLimit: int | *6
				if _jobConfig.backoffLimit != _|_ {
					backoffLimit: _jobConfig.backoffLimit
				}

				activeDeadlineSeconds: int | *300
				if _jobConfig.activeDeadlineSeconds != _|_ {
					activeDeadlineSeconds: _jobConfig.activeDeadlineSeconds
				}

				ttlSecondsAfterFinished: int | *100
				if _jobConfig.ttlSecondsAfterFinished != _|_ {
					ttlSecondsAfterFinished: _jobConfig.ttlSecondsAfterFinished
				}

				template: {
					metadata: labels: #context.componentLabels
					spec: {
						_convertedSidecars: (#ToK8sContainers & {"in": _sidecarContainers, #releasePrefix: #context.#moduleReleaseMetadata.name}).out
						containers: list.Concat([[_mainContainer], _convertedSidecars])

						if len(_initContainers) > 0 {
							initContainers: (#ToK8sContainers & {"in": _initContainers, #releasePrefix: #context.#moduleReleaseMetadata.name}).out
						}

						restartPolicy: _restartPolicy

						if #component.spec.hostPid != _|_ {
							hostPID: #component.spec.hostPid
						}

						if #component.spec.hostIpc != _|_ {
							hostIPC: #component.spec.hostIpc
						}

						if #component.spec.securityContext != _|_ {
							let _sc = #component.spec.securityContext
							if _sc.runAsNonRoot != _|_ || _sc.runAsUser != _|_ || _sc.runAsGroup != _|_ || _sc.supplementalGroups != _|_ {
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

						// Volumes: map persistent claim volumes to PVC references
						if #component.spec.volumes != _|_ {
							volumes: [
								for vName, vol in #component.spec.volumes if vol.persistentClaim != _|_ {
									name: vol.name | *vName
									persistentVolumeClaim: claimName: vol.name | *vName
								},
							]
						}

						// Graceful shutdown: pod-level termination grace period
						if #component.spec.gracefulShutdown != _|_ {
							terminationGracePeriodSeconds: #component.spec.gracefulShutdown.terminationGracePeriodSeconds
						}
					}
				}
			}
		}
	}
}
