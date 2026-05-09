// Kubernetes batch/v1 API group re-exports
package v1

import batchv1 "cue.dev/x/k8s.io/api/batch/v1"

#CronJob:                                batchv1.#CronJob
#CronJobList:                            batchv1.#CronJobList
#CronJobSpec:                            batchv1.#CronJobSpec
#CronJobStatus:                          batchv1.#CronJobStatus
#Job:                                    batchv1.#Job
#JobCondition:                           batchv1.#JobCondition
#JobList:                                batchv1.#JobList
#JobSpec:                                batchv1.#JobSpec
#JobStatus:                              batchv1.#JobStatus
#JobTemplateSpec:                        batchv1.#JobTemplateSpec
#PodFailurePolicy:                       batchv1.#PodFailurePolicy
#PodFailurePolicyOnExitCodesRequirement: batchv1.#PodFailurePolicyOnExitCodesRequirement
#PodFailurePolicyOnPodConditionsPattern: batchv1.#PodFailurePolicyOnPodConditionsPattern
#PodFailurePolicyRule:                   batchv1.#PodFailurePolicyRule
#SuccessPolicy:                          batchv1.#SuccessPolicy
#SuccessPolicyRule:                      batchv1.#SuccessPolicyRule
#UncountedTerminatedPods:                batchv1.#UncountedTerminatedPods
