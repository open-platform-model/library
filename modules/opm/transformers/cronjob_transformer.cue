package transformers

import (
	"list"
	k8sbatchv1 "opmodel.dev/modules/opm/schemas/kubernetes/batch/v1@v1"
	c "opmodel.dev/core/v1alpha2@v1"
	res "opmodel.dev/modules/opm/resources"
	tr "opmodel.dev/modules/opm/traits"
)

// CronJobTransformer converts scheduled task components to Kubernetes CronJobs
#CronJobTransformer: c.#ComponentTransformer & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/transformers"
		version:     "v1"
		name:        "cronjob-transformer"
		description: "Converts scheduled task components to Kubernetes CronJobs"

		labels: {
			"core.opmodel.dev/workload-type": "scheduled-task"
			"core.opmodel.dev/resource-type": "cronjob"
		}
	}

	// Required label to match scheduled task workloads
	requiredLabels: {
		"core.opmodel.dev/workload-type": "scheduled-task"
	}

	// Required resources - Container MUST be present
	requiredResources: {
		(res.#ContainerResource.metadata.fqn): res.#ContainerResource
	}

	// Optional resources
	optionalResources: {
		(res.#VolumesResource.metadata.fqn): res.#VolumesResource
	}

	// Required traits - CronJobConfig is mandatory for CronJob
	requiredTraits: {
		(tr.#CronJobConfigTrait.metadata.fqn): tr.#CronJobConfigTrait
	}

	// Optional traits
	optionalTraits: {
		(tr.#RestartPolicyTrait.metadata.fqn):     tr.#RestartPolicyTrait
		(tr.#SidecarContainersTrait.metadata.fqn): tr.#SidecarContainersTrait
		(tr.#InitContainersTrait.metadata.fqn):    tr.#InitContainersTrait
		(tr.#SecurityContextTrait.metadata.fqn):   tr.#SecurityContextTrait
		(tr.#WorkloadIdentityTrait.metadata.fqn):  tr.#WorkloadIdentityTrait
		(tr.#ImagePullSecretsTrait.metadata.fqn):  tr.#ImagePullSecretsTrait
		(tr.#HostPIDTrait.metadata.fqn):           tr.#HostPIDTrait
		(tr.#HostIPCTrait.metadata.fqn):           tr.#HostIPCTrait
		(tr.#GracefulShutdownTrait.metadata.fqn):  tr.#GracefulShutdownTrait
	}

	#transform: {
		#component: _ // Unconstrained; validated by matching, not by transform signature
		#context:   c.#TransformerContext

		// Extract required Container resource
		_container: #component.spec.container

		// Extract required CronJobConfig trait
		_cronConfig: #component.spec.cronJobConfig

		// Apply defaults for optional RestartPolicy trait
		_restartPolicy: *"OnFailure" | string
		if #component.spec.restartPolicy != _|_ {
			_restartPolicy: #component.spec.restartPolicy
		}

		// Build main container: base conversion via helper, unified with trait fields
		_mainContainer: (#ToK8sContainer & {"in": _container, #releasePrefix: #context.#moduleReleaseMetadata.name}).out

		// Extract optional sidecar and init containers with defaults
		_sidecarContainers: [...]
		if #component.spec.sidecarContainers != _|_ {
			_sidecarContainers: #component.spec.sidecarContainers
		}

		_initContainers: [...]
		if #component.spec.initContainers != _|_ {
			_initContainers: #component.spec.initContainers
		}

		output: k8sbatchv1.#CronJob & {
			apiVersion: "batch/v1"
			kind:       "CronJob"
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
				schedule: _cronConfig.scheduleCron

				if _cronConfig.suspend != _|_ {
					suspend: _cronConfig.suspend
				}

				concurrencyPolicy: string | *"Allow"
				if _cronConfig.concurrencyPolicy != _|_ {
					concurrencyPolicy: _cronConfig.concurrencyPolicy
				}

				successfulJobsHistoryLimit: int | *3
				if _cronConfig.successfulJobsHistoryLimit != _|_ {
					successfulJobsHistoryLimit: _cronConfig.successfulJobsHistoryLimit
				}

				failedJobsHistoryLimit: int | *1
				if _cronConfig.failedJobsHistoryLimit != _|_ {
					failedJobsHistoryLimit: _cronConfig.failedJobsHistoryLimit
				}

				jobTemplate: {
					spec: {
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
	}
}
