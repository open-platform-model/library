package object

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestWeightKnownGVK(t *testing.T) {
	gvk := schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}
	assert.Equal(t, WeightDeployment, Weight(gvk))
}

func TestWeightCoreService(t *testing.T) {
	gvk := schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Service"}
	assert.Equal(t, WeightService, Weight(gvk))
}

func TestWeightCRD(t *testing.T) {
	gvk := schema.GroupVersionKind{Group: "apiextensions.k8s.io", Version: "v1", Kind: "CustomResourceDefinition"}
	assert.Equal(t, WeightCRD, Weight(gvk))
}

func TestWeightUnknown(t *testing.T) {
	gvk := schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "Foo"}
	assert.Equal(t, WeightDefault, Weight(gvk))
}

func TestWeightKindFallback(t *testing.T) {
	gvk := schema.GroupVersionKind{Group: "unknown.io", Version: "v99", Kind: "Deployment"}
	assert.Equal(t, WeightDeployment, Weight(gvk))
}

// TestWeightUnknownVersionFallsBackToKind covers the kubernetes-tier scenario
// "An unknown version of a known kind falls back to the kind".
func TestWeightUnknownVersionFallsBackToKind(t *testing.T) {
	gvk := schema.GroupVersionKind{Group: "autoscaling", Version: "v2beta2", Kind: "HorizontalPodAutoscaler"}
	assert.Equal(t, 200, Weight(gvk))
}

// TestWeightTableGuard pins every constant and every table entry to the
// values ported from the cli (0b37e3f2, cli#289). The kind-class order is
// what every Kubernetes frontend applies by, so a change here is a
// deliberate, reviewed edit of this guard too.
func TestWeightTableGuard(t *testing.T) {
	constants := map[string]int{
		"CRD": WeightCRD, "Namespace": WeightNamespace,
		"ClusterRole": WeightClusterRole, "ClusterRoleBinding": WeightClusterRoleBinding,
		"ServiceAccount": WeightServiceAccount, "Role": WeightRole, "RoleBinding": WeightRoleBinding,
		"Secret": WeightSecret, "ConfigMap": WeightConfigMap,
		"StorageClass": WeightStorageClass, "PersistentVolume": WeightPersistentVolume, "PVC": WeightPVC,
		"Service":    WeightService,
		"Deployment": WeightDeployment, "StatefulSet": WeightStatefulSet, "DaemonSet": WeightDaemonSet,
		"Job": WeightJob, "CronJob": WeightCronJob,
		"Ingress": WeightIngress, "NetworkPolicy": WeightNetworkPolicy,
		"HPA": WeightHPA, "VPA": WeightVPA, "PDB": WeightPDB,
		"Webhook": WeightWebhook, "Default": WeightDefault,
	}
	assert.Equal(t, map[string]int{
		"CRD": -100, "Namespace": 0,
		"ClusterRole": 5, "ClusterRoleBinding": 5,
		"ServiceAccount": 10, "Role": 10, "RoleBinding": 10,
		"Secret": 15, "ConfigMap": 15,
		"StorageClass": 20, "PersistentVolume": 20, "PVC": 20,
		"Service":    50,
		"Deployment": 100, "StatefulSet": 100, "DaemonSet": 100,
		"Job": 110, "CronJob": 110,
		"Ingress": 150, "NetworkPolicy": 150,
		"HPA": 200, "VPA": 200, "PDB": 200,
		"Webhook": 500, "Default": 1000,
	}, constants)

	gvk := func(g, v, k string) schema.GroupVersionKind {
		return schema.GroupVersionKind{Group: g, Version: v, Kind: k}
	}
	assert.Equal(t, map[schema.GroupVersionKind]int{
		gvk("apiextensions.k8s.io", "v1", "CustomResourceDefinition"):               -100,
		gvk("", "v1", "Namespace"):                                                  0,
		gvk("", "v1", "ServiceAccount"):                                             10,
		gvk("", "v1", "Secret"):                                                     15,
		gvk("", "v1", "ConfigMap"):                                                  15,
		gvk("", "v1", "PersistentVolume"):                                           20,
		gvk("", "v1", "PersistentVolumeClaim"):                                      20,
		gvk("", "v1", "Service"):                                                    50,
		gvk("rbac.authorization.k8s.io", "v1", "ClusterRole"):                       5,
		gvk("rbac.authorization.k8s.io", "v1", "ClusterRoleBinding"):                5,
		gvk("rbac.authorization.k8s.io", "v1", "Role"):                              10,
		gvk("rbac.authorization.k8s.io", "v1", "RoleBinding"):                       10,
		gvk("storage.k8s.io", "v1", "StorageClass"):                                 20,
		gvk("apps", "v1", "Deployment"):                                             100,
		gvk("apps", "v1", "StatefulSet"):                                            100,
		gvk("apps", "v1", "DaemonSet"):                                              100,
		gvk("apps", "v1", "ReplicaSet"):                                             100,
		gvk("batch", "v1", "Job"):                                                   110,
		gvk("batch", "v1", "CronJob"):                                               110,
		gvk("networking.k8s.io", "v1", "Ingress"):                                   150,
		gvk("networking.k8s.io", "v1", "NetworkPolicy"):                             150,
		gvk("autoscaling", "v2", "HorizontalPodAutoscaler"):                         200,
		gvk("autoscaling", "v1", "HorizontalPodAutoscaler"):                         200,
		gvk("autoscaling.k8s.io", "v1", "VerticalPodAutoscaler"):                    200,
		gvk("policy", "v1", "PodDisruptionBudget"):                                  200,
		gvk("admissionregistration.k8s.io", "v1", "ValidatingWebhookConfiguration"): 500,
		gvk("admissionregistration.k8s.io", "v1", "MutatingWebhookConfiguration"):   500,
	}, gvkWeights)

	assert.Equal(t, map[string]int{
		"Namespace": 0, "ServiceAccount": 10, "Secret": 15, "ConfigMap": 15,
		"PersistentVolume": 20, "PersistentVolumeClaim": 20, "Service": 50,
		"ClusterRole": 5, "ClusterRoleBinding": 5, "Role": 10, "RoleBinding": 10,
		"StorageClass": 20, "Deployment": 100, "StatefulSet": 100, "DaemonSet": 100,
		"ReplicaSet": 100, "Job": 110, "CronJob": 110, "Ingress": 150, "NetworkPolicy": 150,
		"HorizontalPodAutoscaler": 200, "VerticalPodAutoscaler": 200, "PodDisruptionBudget": 200,
		"ValidatingWebhookConfiguration": 500, "MutatingWebhookConfiguration": 500,
		"CustomResourceDefinition": -100,
	}, kindWeights)
}
