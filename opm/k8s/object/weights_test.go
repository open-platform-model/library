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
	gvk := schema.GroupVersionKind{Group: "apps", Version: "v1beta2", Kind: "Deployment"}
	assert.Equal(t, 100, Weight(gvk))
}

func kinds(gvks ...schema.GroupVersionKind) []item {
	out := make([]item, len(gvks))
	for i, gvk := range gvks {
		out[i] = item{gvk, gvk.Kind}
	}
	return out
}

var (
	gvkWidget   = schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "Widget"}
	gvkService  = schema.GroupVersionKind{Version: "v1", Kind: "Service"}
	gvkSA       = schema.GroupVersionKind{Version: "v1", Kind: "ServiceAccount"}
	gvkMutating = schema.GroupVersionKind{Group: "admissionregistration.k8s.io", Version: "v1", Kind: "MutatingWebhookConfiguration"}
	gvkValidate = schema.GroupVersionKind{Group: "admissionregistration.k8s.io", Version: "v1", Kind: "ValidatingWebhookConfiguration"}
)

// TestWebhooksAfterCustomResources covers the kubernetes-tier scenario
// "Webhook configurations come after custom resources": a webhook with
// failurePolicy Fail applied before its backend runs would refuse every
// later object it matches.
func TestWebhooksAfterCustomResources(t *testing.T) {
	in := kinds(gvkValidate, gvkMutating, gvkWidget, gvkService)
	Sort(in, itemGVK, Ascending)
	assert.Equal(t, []string{"Service", "Widget", "ValidatingWebhookConfiguration", "MutatingWebhookConfiguration"}, names(in))

	in = kinds(gvkValidate, gvkMutating, gvkWidget, gvkService)
	Sort(in, itemGVK, Descending)
	assert.Equal(t, []string{"ValidatingWebhookConfiguration", "MutatingWebhookConfiguration", "Widget", "Service"}, names(in))
}

// TestClassKindsAfterClusterRoles covers the kubernetes-tier scenario "Class
// kinds come right after ClusterRoles", including a custom class kind that
// only the "Class" suffix places.
func TestClassKindsAfterClusterRoles(t *testing.T) {
	in := kinds(
		schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding"},
		schema.GroupVersionKind{Group: "scheduling.k8s.io", Version: "v1", Kind: "PriorityClass"},
		schema.GroupVersionKind{Group: "karpenter.k8s.aws", Version: "v1", Kind: "EC2NodeClass"},
		schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"},
		gvkSA,
	)
	Sort(in, itemGVK, Ascending)
	assert.Equal(t, []string{"ClusterRole", "PriorityClass", "EC2NodeClass", "ClusterRoleBinding", "ServiceAccount"}, names(in))
}

// TestQuotasAndLimitsBeforeWorkloads covers the kubernetes-tier scenario
// "Quotas and limits come before the workloads they constrain".
func TestQuotasAndLimitsBeforeWorkloads(t *testing.T) {
	in := kinds(
		gvkDeployment,
		schema.GroupVersionKind{Version: "v1", Kind: "LimitRange"},
		gvkService,
		schema.GroupVersionKind{Version: "v1", Kind: "ResourceQuota"},
		gvkSA,
	)
	Sort(in, itemGVK, Ascending)
	assert.Equal(t, []string{"ResourceQuota", "ServiceAccount", "Service", "LimitRange", "Deployment"}, names(in))
}

// TestUnlistedKindsWeighTheDefault covers the kubernetes-tier scenario "Kinds
// Flux does not list weigh the default": Flux orders them only by group and
// kind, so one weight keeps that fallback a refinement.
func TestUnlistedKindsWeighTheDefault(t *testing.T) {
	for _, gvk := range []schema.GroupVersionKind{
		{Version: "v1", Kind: "PersistentVolumeClaim"},
		{Group: "apps", Version: "v1", Kind: "DaemonSet"},
		{Group: "batch", Version: "v1", Kind: "Job"},
		{Group: "networking.k8s.io", Version: "v1", Kind: "Ingress"},
		{Group: "autoscaling", Version: "v2", Kind: "HorizontalPodAutoscaler"},
		gvkWidget,
	} {
		assert.Equal(t, 1000, Weight(gvk), gvk.String())
	}
}

// TestDefinitionKindNameInAnotherGroup covers the kubernetes-tier scenario "A
// definition kind name in another group is not a definition": the cluster
// definitions match on group and kind in any version, and a same-named kind
// elsewhere sits with the class kinds, where Flux applies it.
func TestDefinitionKindNameInAnotherGroup(t *testing.T) {
	assert.Equal(t, 6, Weight(schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "Namespace"}))
	assert.Equal(t, 5, Weight(schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1beta1", Kind: "ClusterRole"}))
	assert.Equal(t, -100, Weight(schema.GroupVersionKind{Group: "apiextensions.k8s.io", Version: "v1beta1", Kind: "CustomResourceDefinition"}))
	assert.Equal(t, 6, Weight(schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "CustomResourceDefinition"}))
}

// TestWeightTableGuard pins every constant and every table entry. The
// kind-class order is what every Kubernetes frontend applies by and what the
// cli's order parity test pins, so a change here is a deliberate, reviewed
// edit of this guard too; flux_order_test.go then checks it against Flux.
func TestWeightTableGuard(t *testing.T) {
	constants := map[string]int{
		"CRD": WeightCRD, "Namespace": WeightNamespace,
		"ClusterRole": WeightClusterRole, "Class": WeightClass, "StorageClass": WeightStorageClass,
		"ClusterRoleBinding": WeightClusterRoleBinding, "ResourceQuota": WeightResourceQuota,
		"ServiceAccount": WeightServiceAccount, "Role": WeightRole, "RoleBinding": WeightRoleBinding,
		"Secret": WeightSecret, "ConfigMap": WeightConfigMap,
		"Service": WeightService, "LimitRange": WeightLimitRange,
		"Deployment": WeightDeployment, "StatefulSet": WeightStatefulSet,
		"CronJob": WeightCronJob, "PDB": WeightPDB,
		"Default": WeightDefault, "Webhook": WeightWebhook,
		"PersistentVolume": WeightPersistentVolume, "PVC": WeightPVC, "DaemonSet": WeightDaemonSet,
		"Job": WeightJob, "Ingress": WeightIngress, "NetworkPolicy": WeightNetworkPolicy,
		"HPA": WeightHPA, "VPA": WeightVPA,
	}
	assert.Equal(t, map[string]int{
		"CRD": -100, "Namespace": 0,
		"ClusterRole": 5, "Class": 6, "StorageClass": 6,
		"ClusterRoleBinding": 7, "ResourceQuota": 8,
		"ServiceAccount": 10, "Role": 10, "RoleBinding": 10,
		"Secret": 15, "ConfigMap": 15,
		"Service": 50, "LimitRange": 60,
		"Deployment": 100, "StatefulSet": 100,
		"CronJob": 105, "PDB": 108,
		"Default": 1000, "Webhook": 2000,
		"PersistentVolume": 1000, "PVC": 1000, "DaemonSet": 1000,
		"Job": 1000, "Ingress": 1000, "NetworkPolicy": 1000,
		"HPA": 1000, "VPA": 1000,
	}, constants)

	gvk := func(g, v, k string) schema.GroupVersionKind {
		return schema.GroupVersionKind{Group: g, Version: v, Kind: k}
	}
	assert.Equal(t, map[schema.GroupVersionKind]int{
		gvk("apiextensions.k8s.io", "v1", "CustomResourceDefinition"):               -100,
		gvk("", "v1", "Namespace"):                                                  0,
		gvk("", "v1", "ResourceQuota"):                                              8,
		gvk("", "v1", "ServiceAccount"):                                             10,
		gvk("", "v1", "Secret"):                                                     15,
		gvk("", "v1", "ConfigMap"):                                                  15,
		gvk("", "v1", "Service"):                                                    50,
		gvk("", "v1", "LimitRange"):                                                 60,
		gvk("", "v1", "PersistentVolume"):                                           1000,
		gvk("", "v1", "PersistentVolumeClaim"):                                      1000,
		gvk("rbac.authorization.k8s.io", "v1", "ClusterRole"):                       5,
		gvk("rbac.authorization.k8s.io", "v1", "ClusterRoleBinding"):                7,
		gvk("rbac.authorization.k8s.io", "v1", "Role"):                              10,
		gvk("rbac.authorization.k8s.io", "v1", "RoleBinding"):                       10,
		gvk("storage.k8s.io", "v1", "StorageClass"):                                 6,
		gvk("scheduling.k8s.io", "v1", "PriorityClass"):                             6,
		gvk("node.k8s.io", "v1", "RuntimeClass"):                                    6,
		gvk("networking.k8s.io", "v1", "IngressClass"):                              6,
		gvk("apps", "v1", "Deployment"):                                             100,
		gvk("apps", "v1", "StatefulSet"):                                            100,
		gvk("apps", "v1", "DaemonSet"):                                              1000,
		gvk("apps", "v1", "ReplicaSet"):                                             1000,
		gvk("batch", "v1", "Job"):                                                   1000,
		gvk("batch", "v1", "CronJob"):                                               105,
		gvk("networking.k8s.io", "v1", "Ingress"):                                   1000,
		gvk("networking.k8s.io", "v1", "NetworkPolicy"):                             1000,
		gvk("autoscaling", "v2", "HorizontalPodAutoscaler"):                         1000,
		gvk("autoscaling", "v1", "HorizontalPodAutoscaler"):                         1000,
		gvk("autoscaling.k8s.io", "v1", "VerticalPodAutoscaler"):                    1000,
		gvk("policy", "v1", "PodDisruptionBudget"):                                  108,
		gvk("admissionregistration.k8s.io", "v1", "ValidatingWebhookConfiguration"): 2000,
		gvk("admissionregistration.k8s.io", "v1", "MutatingWebhookConfiguration"):   2000,
	}, gvkWeights)

	assert.Equal(t, map[schema.GroupKind]int{
		{Group: "apiextensions.k8s.io", Kind: "CustomResourceDefinition"}: -100,
		{Group: "", Kind: "Namespace"}:                                    0,
		{Group: "rbac.authorization.k8s.io", Kind: "ClusterRole"}:         5,
	}, definitionWeights)

	assert.Equal(t, map[string]int{
		"CustomResourceDefinition": 6, "Namespace": 6, "ClusterRole": 6,
		"ClusterClass": 6, "RuntimeClass": 6, "PriorityClass": 6, "StorageClass": 6,
		"VolumeSnapshotClass": 6, "IngressClass": 6, "GatewayClass": 6,
		"ClusterRoleBinding": 7, "ResourceQuota": 8,
		"ServiceAccount": 10, "Role": 10, "RoleBinding": 10,
		"Secret": 15, "ConfigMap": 15, "Service": 50, "LimitRange": 60,
		"Deployment": 100, "StatefulSet": 100, "CronJob": 105, "PodDisruptionBudget": 108,
		"ValidatingWebhookConfiguration": 2000, "MutatingWebhookConfiguration": 2000,
		"PersistentVolume": 1000, "PersistentVolumeClaim": 1000, "DaemonSet": 1000,
		"ReplicaSet": 1000, "Job": 1000, "Ingress": 1000, "NetworkPolicy": 1000,
		"HorizontalPodAutoscaler": 1000, "VerticalPodAutoscaler": 1000,
	}, kindWeights)
}
