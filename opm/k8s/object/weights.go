// The kind-class weight table is ported from the cli's pkg/resourceorder
// (cli 0b37e3f2, last changed by cli#289, 75e5f268), values unchanged. Only
// GetWeight is renamed to Weight. The operator has no table of its own; it
// applies through Flux, whose order flux_order_test.go compares with this
// one.

package object

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Kind-class ordering weights for Kubernetes apply and delete. Lower weights
// are applied first and deleted last (0012:D5).
const (
	WeightCRD                = -100
	WeightNamespace          = 0
	WeightClusterRole        = 5
	WeightClusterRoleBinding = 5
	WeightServiceAccount     = 10
	WeightRole               = 10
	WeightRoleBinding        = 10
	WeightSecret             = 15
	WeightConfigMap          = 15
	WeightStorageClass       = 20
	WeightPersistentVolume   = 20
	WeightPVC                = 20
	WeightService            = 50
	WeightDeployment         = 100
	WeightStatefulSet        = 100
	WeightDaemonSet          = 100
	WeightJob                = 110
	WeightCronJob            = 110
	WeightIngress            = 150
	WeightNetworkPolicy      = 150
	WeightHPA                = 200
	WeightVPA                = 200
	WeightPDB                = 200
	WeightWebhook            = 500
	WeightDefault            = 1000
)

var gvkWeights = map[schema.GroupVersionKind]int{
	{Group: "apiextensions.k8s.io", Version: "v1", Kind: "CustomResourceDefinition"}: WeightCRD,

	{Group: "", Version: "v1", Kind: "Namespace"}:             WeightNamespace,
	{Group: "", Version: "v1", Kind: "ServiceAccount"}:        WeightServiceAccount,
	{Group: "", Version: "v1", Kind: "Secret"}:                WeightSecret,
	{Group: "", Version: "v1", Kind: "ConfigMap"}:             WeightConfigMap,
	{Group: "", Version: "v1", Kind: "PersistentVolume"}:      WeightPersistentVolume,
	{Group: "", Version: "v1", Kind: "PersistentVolumeClaim"}: WeightPVC,
	{Group: "", Version: "v1", Kind: "Service"}:               WeightService,

	{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"}:        WeightClusterRole,
	{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding"}: WeightClusterRoleBinding,
	{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role"}:               WeightRole,
	{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"}:        WeightRoleBinding,

	{Group: "storage.k8s.io", Version: "v1", Kind: "StorageClass"}: WeightStorageClass,

	{Group: "apps", Version: "v1", Kind: "Deployment"}:  WeightDeployment,
	{Group: "apps", Version: "v1", Kind: "StatefulSet"}: WeightStatefulSet,
	{Group: "apps", Version: "v1", Kind: "DaemonSet"}:   WeightDaemonSet,
	{Group: "apps", Version: "v1", Kind: "ReplicaSet"}:  WeightDeployment,

	{Group: "batch", Version: "v1", Kind: "Job"}:     WeightJob,
	{Group: "batch", Version: "v1", Kind: "CronJob"}: WeightCronJob,

	{Group: "networking.k8s.io", Version: "v1", Kind: "Ingress"}:       WeightIngress,
	{Group: "networking.k8s.io", Version: "v1", Kind: "NetworkPolicy"}: WeightNetworkPolicy,

	{Group: "autoscaling", Version: "v2", Kind: "HorizontalPodAutoscaler"}:      WeightHPA,
	{Group: "autoscaling", Version: "v1", Kind: "HorizontalPodAutoscaler"}:      WeightHPA,
	{Group: "autoscaling.k8s.io", Version: "v1", Kind: "VerticalPodAutoscaler"}: WeightVPA,

	{Group: "policy", Version: "v1", Kind: "PodDisruptionBudget"}: WeightPDB,

	{Group: "admissionregistration.k8s.io", Version: "v1", Kind: "ValidatingWebhookConfiguration"}: WeightWebhook,
	{Group: "admissionregistration.k8s.io", Version: "v1", Kind: "MutatingWebhookConfiguration"}:   WeightWebhook,
}

var kindWeights = map[string]int{
	"Namespace":                      WeightNamespace,
	"ServiceAccount":                 WeightServiceAccount,
	"Secret":                         WeightSecret,
	"ConfigMap":                      WeightConfigMap,
	"PersistentVolume":               WeightPersistentVolume,
	"PersistentVolumeClaim":          WeightPVC,
	"Service":                        WeightService,
	"ClusterRole":                    WeightClusterRole,
	"ClusterRoleBinding":             WeightClusterRoleBinding,
	"Role":                           WeightRole,
	"RoleBinding":                    WeightRoleBinding,
	"StorageClass":                   WeightStorageClass,
	"Deployment":                     WeightDeployment,
	"StatefulSet":                    WeightStatefulSet,
	"DaemonSet":                      WeightDaemonSet,
	"ReplicaSet":                     WeightDeployment,
	"Job":                            WeightJob,
	"CronJob":                        WeightCronJob,
	"Ingress":                        WeightIngress,
	"NetworkPolicy":                  WeightNetworkPolicy,
	"HorizontalPodAutoscaler":        WeightHPA,
	"VerticalPodAutoscaler":          WeightVPA,
	"PodDisruptionBudget":            WeightPDB,
	"ValidatingWebhookConfiguration": WeightWebhook,
	"MutatingWebhookConfiguration":   WeightWebhook,
	"CustomResourceDefinition":       WeightCRD,
}

// Weight returns the ordering weight for a GVK: the entry for the exact
// group, version and kind, else the entry for the kind alone, else
// [WeightDefault]. Lower weights are applied first.
func Weight(gvk schema.GroupVersionKind) int {
	if w, ok := gvkWeights[gvk]; ok {
		return w
	}
	if w, ok := kindWeights[gvk.Kind]; ok {
		return w
	}
	return WeightDefault
}
