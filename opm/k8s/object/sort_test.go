package object

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type item struct {
	gvk  schema.GroupVersionKind
	name string
}

func itemGVK(i item) schema.GroupVersionKind { return i.gvk }

var (
	gvkDeployment = schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}
	gvkCRD        = schema.GroupVersionKind{Group: "apiextensions.k8s.io", Version: "v1", Kind: "CustomResourceDefinition"}
	gvkConfigMap  = schema.GroupVersionKind{Version: "v1", Kind: "ConfigMap"}
)

func names(items []item) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.name
	}
	return out
}

func TestSort(t *testing.T) {
	tests := []struct {
		name string
		in   []item
		dir  Direction
		want []string
	}{
		{
			name: "ascending for apply",
			in:   []item{{gvkDeployment, "deploy"}, {gvkCRD, "crd"}, {gvkConfigMap, "cm"}},
			dir:  Ascending,
			want: []string{"crd", "cm", "deploy"},
		},
		{
			name: "descending for delete",
			in:   []item{{gvkDeployment, "deploy"}, {gvkCRD, "crd"}, {gvkConfigMap, "cm"}},
			dir:  Descending,
			want: []string{"deploy", "cm", "crd"},
		},
		{
			name: "equal weights keep input order ascending",
			in:   []item{{gvkConfigMap, "b"}, {gvkConfigMap, "a"}},
			dir:  Ascending,
			want: []string{"b", "a"},
		},
		{
			name: "equal weights keep input order descending",
			in:   []item{{gvkConfigMap, "b"}, {gvkDeployment, "d"}, {gvkConfigMap, "a"}},
			dir:  Descending,
			want: []string{"d", "b", "a"},
		},
		{
			name: "empty",
			in:   nil,
			dir:  Ascending,
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Sort(tt.in, itemGVK, tt.dir)
			assert.Equal(t, tt.want, names(tt.in))
		})
	}
}

// TestSortApplyOrder covers the kubernetes-tier scenario "Apply order puts
// definitions before their users".
func TestSortApplyOrder(t *testing.T) {
	in := []item{
		{gvkDeployment, "deploy"},
		{schema.GroupVersionKind{Version: "v1", Kind: "Namespace"}, "ns"},
		{gvkCRD, "crd"},
		{schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "Widget"}, "widget"},
		{gvkConfigMap, "cm"},
	}
	Sort(in, itemGVK, Ascending)
	assert.Equal(t, []string{"crd", "ns", "cm", "deploy", "widget"}, names(in))
}

// TestSortDeleteOrder covers the kubernetes-tier scenario "Delete order is
// the reverse and stable".
func TestSortDeleteOrder(t *testing.T) {
	svc := schema.GroupVersionKind{Version: "v1", Kind: "Service"}
	in := []item{{svc, "svc-a"}, {gvkDeployment, "deploy"}, {svc, "svc-b"}}
	Sort(in, itemGVK, Descending)
	assert.Equal(t, []string{"deploy", "svc-a", "svc-b"}, names(in))
}

// largeMixed returns 40 items, 25 ConfigMaps cm-00..cm-24 interleaved with 15
// Deployments d-00..d-14, with the names in input order per kind. The size is
// above the length at which Go's sort stops using insertion sort (12), so an
// unstable sort reorders equal weights and these tests see it.
func largeMixed() []item {
	var in []item
	cm, d := 0, 0
	for i := 0; i < 40; i++ {
		if i%8 == 1 || i%8 == 4 || i%8 == 6 {
			in = append(in, item{gvkDeployment, fmt.Sprintf("d-%02d", d)})
			d++
		} else {
			in = append(in, item{gvkConfigMap, fmt.Sprintf("cm-%02d", cm)})
			cm++
		}
	}
	return in
}

func seq(prefix string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("%s-%02d", prefix, i)
	}
	return out
}

// TestSortStableOnLargeInput pins the "stable sort" of the kubernetes-tier
// spec on an input large enough that sort.Slice would reorder equal weights.
func TestSortStableOnLargeInput(t *testing.T) {
	asc := largeMixed()
	Sort(asc, itemGVK, Ascending)
	assert.Equal(t, append(seq("cm", 25), seq("d", 15)...), names(asc))

	desc := largeMixed()
	Sort(desc, itemGVK, Descending)
	assert.Equal(t, append(seq("d", 15), seq("cm", 25)...), names(desc))
}
