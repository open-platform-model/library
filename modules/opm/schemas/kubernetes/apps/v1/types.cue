// Kubernetes apps/v1 API group re-exports
package v1

import appsv1 "cue.dev/x/k8s.io/api/apps/v1"

#ControllerRevision:                              appsv1.#ControllerRevision
#ControllerRevisionList:                          appsv1.#ControllerRevisionList
#DaemonSet:                                       appsv1.#DaemonSet
#DaemonSetCondition:                              appsv1.#DaemonSetCondition
#DaemonSetList:                                   appsv1.#DaemonSetList
#DaemonSetSpec:                                   appsv1.#DaemonSetSpec
#DaemonSetStatus:                                 appsv1.#DaemonSetStatus
#DaemonSetUpdateStrategy:                         appsv1.#DaemonSetUpdateStrategy
#Deployment:                                      appsv1.#Deployment
#DeploymentCondition:                             appsv1.#DeploymentCondition
#DeploymentList:                                  appsv1.#DeploymentList
#DeploymentSpec:                                  appsv1.#DeploymentSpec
#DeploymentStatus:                                appsv1.#DeploymentStatus
#DeploymentStrategy:                              appsv1.#DeploymentStrategy
#ReplicaSet:                                      appsv1.#ReplicaSet
#ReplicaSetCondition:                             appsv1.#ReplicaSetCondition
#ReplicaSetList:                                  appsv1.#ReplicaSetList
#ReplicaSetSpec:                                  appsv1.#ReplicaSetSpec
#ReplicaSetStatus:                                appsv1.#ReplicaSetStatus
#RollingUpdateDaemonSet:                          appsv1.#RollingUpdateDaemonSet
#RollingUpdateDeployment:                         appsv1.#RollingUpdateDeployment
#RollingUpdateStatefulSetStrategy:                appsv1.#RollingUpdateStatefulSetStrategy
#StatefulSet:                                     appsv1.#StatefulSet
#StatefulSetCondition:                            appsv1.#StatefulSetCondition
#StatefulSetList:                                 appsv1.#StatefulSetList
#StatefulSetOrdinals:                             appsv1.#StatefulSetOrdinals
#StatefulSetPersistentVolumeClaimRetentionPolicy: appsv1.#StatefulSetPersistentVolumeClaimRetentionPolicy
#StatefulSetSpec:                                 appsv1.#StatefulSetSpec
#StatefulSetStatus:                               appsv1.#StatefulSetStatus
#StatefulSetUpdateStrategy:                       appsv1.#StatefulSetUpdateStrategy
