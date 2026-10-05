// The kind-class weight table was ported from the cli's pkg/resourceorder
// (cli 0b37e3f2, last changed by cli#289, 75e5f268), with GetWeight renamed
// to Weight. Its values have since been aligned with the staged apply order of
// Flux's ssa package (github.com/fluxcd/pkg/ssa v0.77.0, the version the
// operator pins): the operator hands the whole set to Flux's ApplyAllStaged,
// which re-sorts it, and 0012:D5:R1 lets that engine refine the library's
// order and never contradict it. flux_order_test.go fails on any pair the two
// orders put the opposite way round.

package object

import (
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Kind-class ordering weights for Kubernetes apply and delete. Lower weights
// are applied first and deleted last (0012:D5). They follow Flux's staged
// apply order: the cluster definitions, then the class kinds, then Flux's
// ReconcileOrder. Every kind Flux does not list weighs [WeightDefault], so
// Flux's alphabetical fallback among those kinds only refines the order.
const (
	WeightCRD                = -100
	WeightNamespace          = 0
	WeightClusterRole        = 5
	WeightClass              = 6
	WeightStorageClass       = WeightClass
	WeightClusterRoleBinding = 7
	WeightResourceQuota      = 8
	WeightServiceAccount     = 10
	WeightRole               = 10
	WeightRoleBinding        = 10
	WeightSecret             = 15
	WeightConfigMap          = 15
	WeightService            = 50
	WeightLimitRange         = 60
	WeightDeployment         = 100
	WeightStatefulSet        = 100
	WeightCronJob            = 105
	WeightPDB                = 108
	WeightDefault            = 1000
	WeightWebhook            = 2000

	// The kinds below are not in Flux's ReconcileOrder, so they weigh
	// WeightDefault. The names stay for the callers that use them.
	WeightPersistentVolume = WeightDefault
	WeightPVC              = WeightDefault
	WeightDaemonSet        = WeightDefault
	WeightJob              = WeightDefault
	WeightIngress          = WeightDefault
	WeightNetworkPolicy    = WeightDefault
	WeightHPA              = WeightDefault
	WeightVPA              = WeightDefault
)

var gvkWeights = map[schema.GroupVersionKind]int{
	{Group: "apiextensions.k8s.io", Version: "v1", Kind: "CustomResourceDefinition"}: WeightCRD,

	{Group: "", Version: "v1", Kind: "Namespace"}:             WeightNamespace,
	{Group: "", Version: "v1", Kind: "ResourceQuota"}:         WeightResourceQuota,
	{Group: "", Version: "v1", Kind: "ServiceAccount"}:        WeightServiceAccount,
	{Group: "", Version: "v1", Kind: "Secret"}:                WeightSecret,
	{Group: "", Version: "v1", Kind: "ConfigMap"}:             WeightConfigMap,
	{Group: "", Version: "v1", Kind: "Service"}:               WeightService,
	{Group: "", Version: "v1", Kind: "LimitRange"}:            WeightLimitRange,
	{Group: "", Version: "v1", Kind: "PersistentVolume"}:      WeightPersistentVolume,
	{Group: "", Version: "v1", Kind: "PersistentVolumeClaim"}: WeightPVC,

	{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"}:        WeightClusterRole,
	{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding"}: WeightClusterRoleBinding,
	{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role"}:               WeightRole,
	{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"}:        WeightRoleBinding,

	{Group: "storage.k8s.io", Version: "v1", Kind: "StorageClass"}:     WeightStorageClass,
	{Group: "scheduling.k8s.io", Version: "v1", Kind: "PriorityClass"}: WeightClass,
	{Group: "node.k8s.io", Version: "v1", Kind: "RuntimeClass"}:        WeightClass,
	{Group: "networking.k8s.io", Version: "v1", Kind: "IngressClass"}:  WeightClass,

	{Group: "apps", Version: "v1", Kind: "Deployment"}:  WeightDeployment,
	{Group: "apps", Version: "v1", Kind: "StatefulSet"}: WeightStatefulSet,
	{Group: "apps", Version: "v1", Kind: "DaemonSet"}:   WeightDaemonSet,
	{Group: "apps", Version: "v1", Kind: "ReplicaSet"}:  WeightDefault,

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

// definitionWeights are Flux's cluster definitions, matched on group and kind
// in any version: Flux applies them in a stage of their own before anything
// else. Flux also requires apiVersion exactly v1 for the core Namespace; no
// other version of it exists, and Stages matches it on group and kind too.
var definitionWeights = map[schema.GroupKind]int{
	{Group: "apiextensions.k8s.io", Kind: "CustomResourceDefinition"}: WeightCRD,
	{Group: "", Kind: "Namespace"}:                                    WeightNamespace,
	{Group: "rbac.authorization.k8s.io", Kind: "ClusterRole"}:         WeightClusterRole,
}

var kindWeights = map[string]int{
	// A kind with a definition's name in another group is not a definition:
	// Flux applies it after the class kinds and before ClusterRoleBinding.
	"CustomResourceDefinition": WeightClass,
	"Namespace":                WeightClass,
	"ClusterRole":              WeightClass,

	"ClusterClass":        WeightClass,
	"RuntimeClass":        WeightClass,
	"PriorityClass":       WeightClass,
	"StorageClass":        WeightStorageClass,
	"VolumeSnapshotClass": WeightClass,
	"IngressClass":        WeightClass,
	"GatewayClass":        WeightClass,

	"ClusterRoleBinding":             WeightClusterRoleBinding,
	"ResourceQuota":                  WeightResourceQuota,
	"ServiceAccount":                 WeightServiceAccount,
	"Role":                           WeightRole,
	"RoleBinding":                    WeightRoleBinding,
	"Secret":                         WeightSecret,
	"ConfigMap":                      WeightConfigMap,
	"Service":                        WeightService,
	"LimitRange":                     WeightLimitRange,
	"Deployment":                     WeightDeployment,
	"StatefulSet":                    WeightStatefulSet,
	"CronJob":                        WeightCronJob,
	"PodDisruptionBudget":            WeightPDB,
	"ValidatingWebhookConfiguration": WeightWebhook,
	"MutatingWebhookConfiguration":   WeightWebhook,

	"PersistentVolume":        WeightPersistentVolume,
	"PersistentVolumeClaim":   WeightPVC,
	"DaemonSet":               WeightDaemonSet,
	"ReplicaSet":              WeightDefault,
	"Job":                     WeightJob,
	"Ingress":                 WeightIngress,
	"NetworkPolicy":           WeightNetworkPolicy,
	"HorizontalPodAutoscaler": WeightHPA,
	"VerticalPodAutoscaler":   WeightVPA,
}

// Weight returns the ordering weight for a GVK. Lower weights are applied
// first. The lookup follows Flux's staged apply: the entry for the exact
// group, version and kind; else a cluster definition (a CustomResourceDefinition
// of apiextensions.k8s.io, a core Namespace or a ClusterRole of
// rbac.authorization.k8s.io) by group and kind in any version; else the entry
// for the kind alone, where a definition's kind name in another group weighs
// [WeightClass]; else [WeightClass] for any kind whose name ends in "Class"
// (case-sensitive, as Flux's class stage is); else [WeightDefault].
func Weight(gvk schema.GroupVersionKind) int {
	if w, ok := gvkWeights[gvk]; ok {
		return w
	}
	if w, ok := definitionWeights[gvk.GroupKind()]; ok {
		return w
	}
	if w, ok := kindWeights[gvk.Kind]; ok {
		return w
	}
	if strings.HasSuffix(gvk.Kind, "Class") {
		return WeightClass
	}
	return WeightDefault
}
