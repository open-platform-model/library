package object

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func stageNames(stages []Stage[item]) ([][]string, []bool) {
	var got [][]string
	var defs []bool
	for _, s := range stages {
		got = append(got, names(s.Items))
		defs = append(defs, s.ClusterDefinitions)
	}
	return got, defs
}

// TestStagesTypicalModule covers the kubernetes-tier scenario "Stages for a
// typical module". The input slice is unchanged afterwards.
func TestStagesTypicalModule(t *testing.T) {
	in := []item{
		{gvkDeployment, "deploy"},
		{schema.GroupVersionKind{Version: "v1", Kind: "Service"}, "svc"},
		{gvkConfigMap, "cm"},
		{schema.GroupVersionKind{Version: "v1", Kind: "Namespace"}, "ns"},
		{schema.GroupVersionKind{Version: "v1", Kind: "Secret"}, "secret"},
		{gvkCRD, "crd"},
	}
	before := append([]item(nil), in...)

	got, defs := stageNames(Stages(in, itemGVK))
	assert.Equal(t, [][]string{{"crd", "ns"}, {"cm", "secret"}, {"svc"}, {"deploy"}}, got)
	assert.Equal(t, []bool{true, false, false, false}, defs)
	assert.Equal(t, before, in, "the input is not reordered")
}

// TestStagesNamespaceInOtherGroup covers the kubernetes-tier scenario "A
// Namespace kind in another group is not a cluster definition".
func TestStagesNamespaceInOtherGroup(t *testing.T) {
	in := []item{
		{schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "Namespace"}, "other-ns"},
		{schema.GroupVersionKind{Version: "v1", Kind: "Namespace"}, "core-ns"},
	}
	got, defs := stageNames(Stages(in, itemGVK))
	assert.Equal(t, [][]string{{"core-ns"}, {"other-ns"}}, got)
	assert.Equal(t, []bool{true, false}, defs)
}

// TestStagesNoClusterDefinitions covers the kubernetes-tier scenario "No
// cluster definitions, no definition stage".
func TestStagesNoClusterDefinitions(t *testing.T) {
	in := []item{
		{gvkDeployment, "deploy"},
		{schema.GroupVersionKind{Version: "v1", Kind: "Service"}, "svc"},
	}
	got, defs := stageNames(Stages(in, itemGVK))
	assert.Equal(t, [][]string{{"svc"}, {"deploy"}}, got)
	assert.Equal(t, []bool{false, false}, defs)
}

// TestStagesOneWeightPerStage checks that every stage after the definitions
// holds a single weight, and that kinds of equal weight keep their input
// order. The operator submits one stage per Flux ApplyAll call, so Flux's
// re-sort within a call can refine the library's order and never contradict
// it (0012:D4, ADR-011).
func TestStagesOneWeightPerStage(t *testing.T) {
	sts := schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "StatefulSet"}
	widget := schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "Widget"}
	in := []item{
		{widget, "w1"}, {sts, "sts"}, {gvkDeployment, "deploy"}, {widget, "w2"},
	}
	stages := Stages(in, itemGVK)
	got, _ := stageNames(stages)
	assert.Equal(t, [][]string{{"sts", "deploy"}, {"w1", "w2"}}, got)
	for _, s := range stages {
		w := Weight(itemGVK(s.Items[0]))
		for _, it := range s.Items {
			assert.Equal(t, w, Weight(itemGVK(it)))
		}
	}
}

func TestStagesEmpty(t *testing.T) {
	assert.Empty(t, Stages[item](nil, itemGVK))
}

// TestStagesStableOnLargeInput pins "Order within a stage SHALL be the stable
// sort order" on an input large enough that sort.Slice would reorder it. Two
// CRDs sit among the items, so the definition stage is checked too.
func TestStagesStableOnLargeInput(t *testing.T) {
	in := largeMixed()
	in = append(in[:7], append([]item{{gvkCRD, "crd-00"}}, in[7:]...)...)
	in = append(in, item{gvkCRD, "crd-01"})
	for i := 0; i < 15; i++ {
		in = append(in, item{gvkConfigMap, fmt.Sprintf("cm-%02d", 25+i)})
	}

	got, defs := stageNames(Stages(in, itemGVK))
	assert.Equal(t, [][]string{seq("crd", 2), seq("cm", 40), seq("d", 15)}, got)
	assert.Equal(t, []bool{true, false, false}, defs)
}
