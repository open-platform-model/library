package object

import (
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// fluxFirst and fluxLast are ReconcileOrder.First and .Last from
// github.com/fluxcd/pkg/ssa v0.77.0 sort.go, the version the operator pins.
// They are literals: Flux is the operator's engine and depguard keeps it out
// of the library.
var (
	fluxFirst = []string{
		"CustomResourceDefinition",
		"Namespace",
		"ClusterRole",
		"ClusterClass",
		"RuntimeClass",
		"PriorityClass",
		"StorageClass",
		"VolumeSnapshotClass",
		"IngressClass",
		"GatewayClass",
		"ClusterRoleBinding",
		"ResourceQuota",
		"ServiceAccount",
		"Role",
		"RoleBinding",
		"ConfigMap",
		"Secret",
		"Service",
		"LimitRange",
		"Deployment",
		"StatefulSet",
		"CronJob",
		"PodDisruptionBudget",
	}
	fluxLast = []string{
		"MutatingWebhookConfiguration",
		"ValidatingWebhookConfiguration",
	}
)

// fluxRank mirrors ssa v0.77.0 computeKind2index: First kinds rank
// -len(First)..-1, Last kinds 1..len(Last), and every kind Flux does not list
// ranks 0. Flux breaks ties by group, then kind; the comparison counts only
// pairs whose ranks differ, so it does not model the tie-break.
func fluxRank(kind string) int {
	if i := slices.Index(fluxFirst, kind); i >= 0 {
		return -len(fluxFirst) + i
	}
	if i := slices.Index(fluxLast, kind); i >= 0 {
		return 1 + i
	}
	return 0
}

// customKind stands for every custom resource: unlisted in Flux (rank 0) and
// at WeightDefault in the library.
const customKind = "Widget"

// comparisonUniverse is Flux's listed kinds, the library's kind table and one
// custom kind, sorted and deduplicated.
func comparisonUniverse() []string {
	var kinds []string
	kinds = append(kinds, fluxFirst...)
	kinds = append(kinds, fluxLast...)
	for k := range kindWeights {
		kinds = append(kinds, k)
	}
	kinds = append(kinds, customKind)
	slices.Sort(kinds)
	return slices.Compact(kinds)
}

func libraryWeight(kind string) int {
	return Weight(schema.GroupVersionKind{Kind: kind})
}

// fluxContradictions returns every pair "A before B" that Flux orders A
// strictly before B and the library orders B strictly before A.
func fluxContradictions() []string {
	kinds := comparisonUniverse()
	var out []string
	for _, a := range kinds {
		for _, b := range kinds {
			if fluxRank(a) < fluxRank(b) && libraryWeight(a) > libraryWeight(b) {
				out = append(out, fmt.Sprintf("%s before %s", a, b))
			}
		}
	}
	slices.Sort(out)
	return out
}

// TestFluxOrderContradictions records every pair of kinds the library orders
// strictly opposite to Flux v0.77.0. It decides nothing: under SD11 the
// operator submits one library stage per Flux ApplyAll, so none of these
// pairs ever meets inside one Flux sort. A weight edit that adds or removes a
// pair fails here until this list is edited in review (kubernetes-tier,
// "Differences from Flux's apply order are recorded").
func TestFluxOrderContradictions(t *testing.T) {
	want := []string{
		// Kinds Flux lists early that the library leaves at WeightDefault (1000):
		// the cluster-scoped *Class kinds, ResourceQuota and LimitRange. Flux
		// applies them before the workloads that name them; the library after.
		"ClusterClass before ClusterRoleBinding",
		"ClusterClass before ConfigMap",
		"ClusterClass before CronJob",
		"ClusterClass before DaemonSet",
		"ClusterClass before Deployment",
		"ClusterClass before HorizontalPodAutoscaler",
		"ClusterClass before Ingress",
		"ClusterClass before Job",
		"ClusterClass before MutatingWebhookConfiguration",
		"ClusterClass before NetworkPolicy",
		"ClusterClass before PersistentVolume",
		"ClusterClass before PersistentVolumeClaim",
		"ClusterClass before PodDisruptionBudget",
		"ClusterClass before ReplicaSet",
		"ClusterClass before Role",
		"ClusterClass before RoleBinding",
		"ClusterClass before Secret",
		"ClusterClass before Service",
		"ClusterClass before ServiceAccount",
		"ClusterClass before StatefulSet",
		"ClusterClass before StorageClass",
		"ClusterClass before ValidatingWebhookConfiguration",
		"ClusterClass before VerticalPodAutoscaler",
		"GatewayClass before ClusterRoleBinding",
		"GatewayClass before ConfigMap",
		"GatewayClass before CronJob",
		"GatewayClass before DaemonSet",
		"GatewayClass before Deployment",
		"GatewayClass before HorizontalPodAutoscaler",
		"GatewayClass before Ingress",
		"GatewayClass before Job",
		"GatewayClass before MutatingWebhookConfiguration",
		"GatewayClass before NetworkPolicy",
		"GatewayClass before PersistentVolume",
		"GatewayClass before PersistentVolumeClaim",
		"GatewayClass before PodDisruptionBudget",
		"GatewayClass before ReplicaSet",
		"GatewayClass before Role",
		"GatewayClass before RoleBinding",
		"GatewayClass before Secret",
		"GatewayClass before Service",
		"GatewayClass before ServiceAccount",
		"GatewayClass before StatefulSet",
		"GatewayClass before ValidatingWebhookConfiguration",
		"GatewayClass before VerticalPodAutoscaler",
		"IngressClass before ClusterRoleBinding",
		"IngressClass before ConfigMap",
		"IngressClass before CronJob",
		"IngressClass before DaemonSet",
		"IngressClass before Deployment",
		"IngressClass before HorizontalPodAutoscaler",
		"IngressClass before Ingress",
		"IngressClass before Job",
		"IngressClass before MutatingWebhookConfiguration",
		"IngressClass before NetworkPolicy",
		"IngressClass before PersistentVolume",
		"IngressClass before PersistentVolumeClaim",
		"IngressClass before PodDisruptionBudget",
		"IngressClass before ReplicaSet",
		"IngressClass before Role",
		"IngressClass before RoleBinding",
		"IngressClass before Secret",
		"IngressClass before Service",
		"IngressClass before ServiceAccount",
		"IngressClass before StatefulSet",
		"IngressClass before ValidatingWebhookConfiguration",
		"IngressClass before VerticalPodAutoscaler",
		"LimitRange before CronJob",
		"LimitRange before DaemonSet",
		"LimitRange before Deployment",
		"LimitRange before HorizontalPodAutoscaler",
		"LimitRange before Ingress",
		"LimitRange before Job",
		"LimitRange before MutatingWebhookConfiguration",
		"LimitRange before NetworkPolicy",
		"LimitRange before PersistentVolume",
		"LimitRange before PersistentVolumeClaim",
		"LimitRange before PodDisruptionBudget",
		"LimitRange before ReplicaSet",
		"LimitRange before StatefulSet",
		"LimitRange before ValidatingWebhookConfiguration",
		"LimitRange before VerticalPodAutoscaler",
		"PriorityClass before ClusterRoleBinding",
		"PriorityClass before ConfigMap",
		"PriorityClass before CronJob",
		"PriorityClass before DaemonSet",
		"PriorityClass before Deployment",
		"PriorityClass before HorizontalPodAutoscaler",
		"PriorityClass before Ingress",
		"PriorityClass before Job",
		"PriorityClass before MutatingWebhookConfiguration",
		"PriorityClass before NetworkPolicy",
		"PriorityClass before PersistentVolume",
		"PriorityClass before PersistentVolumeClaim",
		"PriorityClass before PodDisruptionBudget",
		"PriorityClass before ReplicaSet",
		"PriorityClass before Role",
		"PriorityClass before RoleBinding",
		"PriorityClass before Secret",
		"PriorityClass before Service",
		"PriorityClass before ServiceAccount",
		"PriorityClass before StatefulSet",
		"PriorityClass before StorageClass",
		"PriorityClass before ValidatingWebhookConfiguration",
		"PriorityClass before VerticalPodAutoscaler",
		"ResourceQuota before ConfigMap",
		"ResourceQuota before CronJob",
		"ResourceQuota before DaemonSet",
		"ResourceQuota before Deployment",
		"ResourceQuota before HorizontalPodAutoscaler",
		"ResourceQuota before Ingress",
		"ResourceQuota before Job",
		"ResourceQuota before MutatingWebhookConfiguration",
		"ResourceQuota before NetworkPolicy",
		"ResourceQuota before PersistentVolume",
		"ResourceQuota before PersistentVolumeClaim",
		"ResourceQuota before PodDisruptionBudget",
		"ResourceQuota before ReplicaSet",
		"ResourceQuota before Role",
		"ResourceQuota before RoleBinding",
		"ResourceQuota before Secret",
		"ResourceQuota before Service",
		"ResourceQuota before ServiceAccount",
		"ResourceQuota before StatefulSet",
		"ResourceQuota before ValidatingWebhookConfiguration",
		"ResourceQuota before VerticalPodAutoscaler",
		"RuntimeClass before ClusterRoleBinding",
		"RuntimeClass before ConfigMap",
		"RuntimeClass before CronJob",
		"RuntimeClass before DaemonSet",
		"RuntimeClass before Deployment",
		"RuntimeClass before HorizontalPodAutoscaler",
		"RuntimeClass before Ingress",
		"RuntimeClass before Job",
		"RuntimeClass before MutatingWebhookConfiguration",
		"RuntimeClass before NetworkPolicy",
		"RuntimeClass before PersistentVolume",
		"RuntimeClass before PersistentVolumeClaim",
		"RuntimeClass before PodDisruptionBudget",
		"RuntimeClass before ReplicaSet",
		"RuntimeClass before Role",
		"RuntimeClass before RoleBinding",
		"RuntimeClass before Secret",
		"RuntimeClass before Service",
		"RuntimeClass before ServiceAccount",
		"RuntimeClass before StatefulSet",
		"RuntimeClass before StorageClass",
		"RuntimeClass before ValidatingWebhookConfiguration",
		"RuntimeClass before VerticalPodAutoscaler",
		"VolumeSnapshotClass before ClusterRoleBinding",
		"VolumeSnapshotClass before ConfigMap",
		"VolumeSnapshotClass before CronJob",
		"VolumeSnapshotClass before DaemonSet",
		"VolumeSnapshotClass before Deployment",
		"VolumeSnapshotClass before HorizontalPodAutoscaler",
		"VolumeSnapshotClass before Ingress",
		"VolumeSnapshotClass before Job",
		"VolumeSnapshotClass before MutatingWebhookConfiguration",
		"VolumeSnapshotClass before NetworkPolicy",
		"VolumeSnapshotClass before PersistentVolume",
		"VolumeSnapshotClass before PersistentVolumeClaim",
		"VolumeSnapshotClass before PodDisruptionBudget",
		"VolumeSnapshotClass before ReplicaSet",
		"VolumeSnapshotClass before Role",
		"VolumeSnapshotClass before RoleBinding",
		"VolumeSnapshotClass before Secret",
		"VolumeSnapshotClass before Service",
		"VolumeSnapshotClass before ServiceAccount",
		"VolumeSnapshotClass before StatefulSet",
		"VolumeSnapshotClass before ValidatingWebhookConfiguration",
		"VolumeSnapshotClass before VerticalPodAutoscaler",
		// StorageClass: Flux applies it right after ClusterRole (rank -17); the
		// library weighs it 20, after RBAC, ServiceAccounts, Secrets and ConfigMaps.
		"StorageClass before ClusterRoleBinding",
		"StorageClass before ConfigMap",
		"StorageClass before Role",
		"StorageClass before RoleBinding",
		"StorageClass before Secret",
		"StorageClass before ServiceAccount",
		// PersistentVolume and PersistentVolumeClaim (and the other kinds Flux does
		// not list) rank 0 in Flux, after Service, Deployment, StatefulSet and
		// CronJob; the library weighs volumes 20, before all of those, and
		// DaemonSet and ReplicaSet 100, before CronJob at 110.
		"Service before PersistentVolume",
		"Service before PersistentVolumeClaim",
		"Deployment before PersistentVolume",
		"Deployment before PersistentVolumeClaim",
		"StatefulSet before PersistentVolume",
		"StatefulSet before PersistentVolumeClaim",
		"CronJob before DaemonSet",
		"CronJob before PersistentVolume",
		"CronJob before PersistentVolumeClaim",
		"CronJob before ReplicaSet",
		// PodDisruptionBudget is the last of Flux's First kinds (rank -1); the
		// library weighs it 200, after workloads, Jobs, Ingresses and volumes.
		"PodDisruptionBudget before DaemonSet",
		"PodDisruptionBudget before Ingress",
		"PodDisruptionBudget before Job",
		"PodDisruptionBudget before NetworkPolicy",
		"PodDisruptionBudget before PersistentVolume",
		"PodDisruptionBudget before PersistentVolumeClaim",
		"PodDisruptionBudget before ReplicaSet",
		// Webhook configurations are Flux's Last kinds (ranks 1 and 2), after every
		// custom resource at rank 0; the library weighs them 500, before custom
		// resources at 1000.
		"Widget before MutatingWebhookConfiguration",
		"Widget before ValidatingWebhookConfiguration",
	}
	slices.Sort(want)
	assert.Equal(t, want, fluxContradictions())
}

// TestFluxOrderDefinitionStage pins the pair the definition stage rests on:
// both orders put a CustomResourceDefinition before a Namespace, so the stage
// can go to Flux in one call although it spans two library weights.
func TestFluxOrderDefinitionStage(t *testing.T) {
	assert.Less(t, fluxRank("CustomResourceDefinition"), fluxRank("Namespace"))
	assert.Less(t, libraryWeight("CustomResourceDefinition"), libraryWeight("Namespace"))
	assert.NotContains(t, fluxContradictions(), "CustomResourceDefinition before Namespace")
}
