package object

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// fluxFirst and fluxLast are ReconcileOrder.First and .Last from
// github.com/fluxcd/pkg/ssa v0.77.0 sort.go, the version the operator pins.
// They are literals: Flux is the operator's engine and depguard keeps it out
// of the library. A Flux bump in the operator updates them, and the guard
// below then shows any pair the new order contradicts.
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
// ranks 0.
func fluxRank(kind string) int {
	if i := slices.Index(fluxFirst, kind); i >= 0 {
		return -len(fluxFirst) + i
	}
	if i := slices.Index(fluxLast, kind); i >= 0 {
		return 1 + i
	}
	return 0
}

// fluxStage mirrors the stages of ssa v0.77.0 ApplyAllStaged, which applies
// one stage after the other: the cluster definitions (utils.IsClusterDefinition:
// a CustomResourceDefinition of apiextensions.k8s.io, the core Namespace and a
// ClusterRole of rbac.authorization.k8s.io), then the class definitions
// (utils.IsClassDefinition: a case-sensitive "Class" suffix on the kind), then
// the rest. The custom stage is not modelled: no frontend sets
// ApplyOptions.CustomStageKinds. Flux's IsNamespace also requires apiVersion
// exactly v1; the guard compares group and kind, so it treats every version
// of the core Namespace as a definition, as Weight and Stages do.
func fluxStage(gk schema.GroupKind) int {
	switch {
	case gk.Group == "apiextensions.k8s.io" && gk.Kind == "CustomResourceDefinition",
		gk.Group == "" && gk.Kind == "Namespace",
		gk.Group == "rbac.authorization.k8s.io" && gk.Kind == "ClusterRole":
		return 0
	case strings.HasSuffix(gk.Kind, "Class"):
		return 1
	default:
		return 2
	}
}

// fluxLess reports whether Flux v0.77.0's staged apply applies a strictly
// before b: by stage, then (ssa IsLessThan) by ReconcileOrder rank, then by
// API group, then by kind.
func fluxLess(a, b schema.GroupKind) bool {
	if sa, sb := fluxStage(a), fluxStage(b); sa != sb {
		return sa < sb
	}
	if ra, rb := fluxRank(a.Kind), fluxRank(b.Kind); ra != rb {
		return ra < rb
	}
	if a.Group != b.Group {
		return a.Group < b.Group
	}
	return a.Kind < b.Kind
}

// canonicalGVK is where each kind the guard knows by name lives in
// Kubernetes or its common extensions. Every kind of Flux's ReconcileOrder and
// of the library's kind table must have an entry (TestFluxUniverseIsComplete).
var canonicalGVK = map[string]schema.GroupVersionKind{
	"CustomResourceDefinition":       {Group: "apiextensions.k8s.io", Version: "v1", Kind: "CustomResourceDefinition"},
	"Namespace":                      {Version: "v1", Kind: "Namespace"},
	"ClusterRole":                    {Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"},
	"ClusterClass":                   {Group: "cluster.x-k8s.io", Version: "v1beta1", Kind: "ClusterClass"},
	"RuntimeClass":                   {Group: "node.k8s.io", Version: "v1", Kind: "RuntimeClass"},
	"PriorityClass":                  {Group: "scheduling.k8s.io", Version: "v1", Kind: "PriorityClass"},
	"StorageClass":                   {Group: "storage.k8s.io", Version: "v1", Kind: "StorageClass"},
	"VolumeSnapshotClass":            {Group: "snapshot.storage.k8s.io", Version: "v1", Kind: "VolumeSnapshotClass"},
	"IngressClass":                   {Group: "networking.k8s.io", Version: "v1", Kind: "IngressClass"},
	"GatewayClass":                   {Group: "gateway.networking.k8s.io", Version: "v1", Kind: "GatewayClass"},
	"ClusterRoleBinding":             {Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding"},
	"ResourceQuota":                  {Version: "v1", Kind: "ResourceQuota"},
	"ServiceAccount":                 {Version: "v1", Kind: "ServiceAccount"},
	"Role":                           {Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role"},
	"RoleBinding":                    {Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"},
	"ConfigMap":                      {Version: "v1", Kind: "ConfigMap"},
	"Secret":                         {Version: "v1", Kind: "Secret"},
	"Service":                        {Version: "v1", Kind: "Service"},
	"LimitRange":                     {Version: "v1", Kind: "LimitRange"},
	"Deployment":                     {Group: "apps", Version: "v1", Kind: "Deployment"},
	"StatefulSet":                    {Group: "apps", Version: "v1", Kind: "StatefulSet"},
	"CronJob":                        {Group: "batch", Version: "v1", Kind: "CronJob"},
	"PodDisruptionBudget":            {Group: "policy", Version: "v1", Kind: "PodDisruptionBudget"},
	"MutatingWebhookConfiguration":   {Group: "admissionregistration.k8s.io", Version: "v1", Kind: "MutatingWebhookConfiguration"},
	"ValidatingWebhookConfiguration": {Group: "admissionregistration.k8s.io", Version: "v1", Kind: "ValidatingWebhookConfiguration"},
	"PersistentVolume":               {Version: "v1", Kind: "PersistentVolume"},
	"PersistentVolumeClaim":          {Version: "v1", Kind: "PersistentVolumeClaim"},
	"DaemonSet":                      {Group: "apps", Version: "v1", Kind: "DaemonSet"},
	"ReplicaSet":                     {Group: "apps", Version: "v1", Kind: "ReplicaSet"},
	"Job":                            {Group: "batch", Version: "v1", Kind: "Job"},
	"Ingress":                        {Group: "networking.k8s.io", Version: "v1", Kind: "Ingress"},
	"NetworkPolicy":                  {Group: "networking.k8s.io", Version: "v1", Kind: "NetworkPolicy"},
	"HorizontalPodAutoscaler":        {Group: "autoscaling", Version: "v2", Kind: "HorizontalPodAutoscaler"},
	"VerticalPodAutoscaler":          {Group: "autoscaling.k8s.io", Version: "v1", Kind: "VerticalPodAutoscaler"},
}

// Groups that sort before every built-in group ("" aside) and after all of
// them, so a same-named or custom kind meets Flux's group tie-break on both
// sides.
const (
	groupBefore = "a.example"
	groupAfter  = "zz.example"
)

// fluxUniverse is every GVK the guard compares: the library's GVK table;
// every kind of its kind table and of Flux's ReconcileOrder in its canonical
// group; a same-named copy of each of those kinds in groupBefore and
// groupAfter; a custom kind in both groups; and a custom class kind.
func fluxUniverse() []schema.GroupVersionKind {
	var out []schema.GroupVersionKind
	for gvk := range gvkWeights {
		out = append(out, gvk)
	}
	kinds := slices.Concat(fluxFirst, fluxLast)
	for k := range kindWeights {
		kinds = append(kinds, k)
	}
	slices.Sort(kinds)
	kinds = slices.Compact(kinds)
	for _, k := range kinds {
		if gvk, ok := canonicalGVK[k]; ok {
			out = append(out, gvk)
		}
		out = append(out,
			schema.GroupVersionKind{Group: groupBefore, Version: "v1", Kind: k},
			schema.GroupVersionKind{Group: groupAfter, Version: "v1", Kind: k},
		)
	}
	out = append(out,
		schema.GroupVersionKind{Group: groupBefore, Version: "v1", Kind: "Widget"},
		schema.GroupVersionKind{Group: groupAfter, Version: "v1", Kind: "Widget"},
		schema.GroupVersionKind{Group: groupBefore, Version: "v1", Kind: "WidgetClass"},
	)
	slices.SortFunc(out, func(a, b schema.GroupVersionKind) int {
		return strings.Compare(a.String(), b.String())
	})
	return slices.Compact(out)
}

func gkString(gk schema.GroupKind) string { return gk.Group + "/" + gk.Kind }

// contradictions returns every pair of the universe that Flux's staged apply
// orders A strictly before B while weight orders B strictly before A, written
// "<group>/<kind> before <group>/<kind>" in Flux's order.
func contradictions(weight func(schema.GroupVersionKind) int) []string {
	universe := fluxUniverse()
	var out []string
	for _, a := range universe {
		for _, b := range universe {
			if fluxLess(a.GroupKind(), b.GroupKind()) && weight(a) > weight(b) {
				out = append(out, fmt.Sprintf("%s before %s", gkString(a.GroupKind()), gkString(b.GroupKind())))
			}
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// TestWeightNeverContradictsFlux is the guard of kubernetes-tier "The weight
// table never contradicts Flux's staged apply order": the operator hands the
// whole set to Flux's ApplyAllStaged, which re-sorts it, so Flux may refine
// the library's order and never contradict it (0012:D5:R1). A table edit or a
// Flux bump that reorders a pair fails here, naming the pair.
func TestWeightNeverContradictsFlux(t *testing.T) {
	assert.Empty(t, contradictions(Weight))
}

// TestFluxComparisonCatchesAContradiction proves the guard cannot pass
// vacuously: an injected contradiction between two ranked kinds is reported,
// and so is one between two kinds Flux orders only by its group tie-break.
func TestFluxComparisonCatchesAContradiction(t *testing.T) {
	t.Run("rank", func(t *testing.T) {
		weight := func(gvk schema.GroupVersionKind) int {
			if gvk.Group == "apps" && gvk.Kind == "Deployment" {
				return WeightService - 1
			}
			return Weight(gvk)
		}
		assert.Contains(t, contradictions(weight), "/Service before apps/Deployment")
	})
	t.Run("group tie-break", func(t *testing.T) {
		weight := func(gvk schema.GroupVersionKind) int {
			if gvk.Group == "autoscaling" && gvk.Kind == "HorizontalPodAutoscaler" {
				return WeightDefault + 1
			}
			return Weight(gvk)
		}
		assert.Contains(t, contradictions(weight), "autoscaling/HorizontalPodAutoscaler before batch/Job")
	})
}

// TestFluxOrderDefinitionStage pins the pair the definition stage rests on:
// both orders put a CustomResourceDefinition before a core Namespace, so the
// stage can go to Flux in one call although it spans two library weights.
func TestFluxOrderDefinitionStage(t *testing.T) {
	crd := canonicalGVK["CustomResourceDefinition"]
	ns := canonicalGVK["Namespace"]
	assert.True(t, fluxLess(crd.GroupKind(), ns.GroupKind()))
	assert.Less(t, Weight(crd), Weight(ns))
}

// TestFluxUniverseIsComplete keeps canonicalGVK in step with the two lists it
// serves, so no kind silently drops out of the guard's canonical groups.
func TestFluxUniverseIsComplete(t *testing.T) {
	for _, k := range slices.Concat(fluxFirst, fluxLast) {
		require.Contains(t, canonicalGVK, k)
	}
	for k := range kindWeights {
		if _, ok := canonicalGVK[k]; !ok {
			t.Errorf("kind table entry %q has no canonical GVK in the Flux guard", k)
		}
	}
}
