package object

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Stage is one apply stage: the objects a frontend submits in one call to its
// apply engine before it moves on to the next stage.
type Stage[T any] struct {
	// ClusterDefinitions marks the stage that holds the CustomResourceDefinitions
	// and core Namespaces. A frontend waits for them (a CRD to be Established)
	// before it applies the next stage, whose objects may need them.
	ClusterDefinitions bool

	// Items are the stage's objects in stable [Sort] order.
	Items []T
}

// Stages returns a sorted copy of items cut into apply stages. The cluster
// definitions (every CustomResourceDefinition of apiextensions.k8s.io and
// every Namespace of the core group) come first as one stage, omitted when
// there are none. Then comes one stage per distinct [Weight] of the remaining
// items, in ascending weight. Order within a stage is the stable sort order.
// The input slice is not reordered, and no stage is empty.
//
// The definition stage spans two weights, -100 and 0, and Flux's order also
// puts a CustomResourceDefinition before a Namespace. The [Weight] table
// agrees with the staged apply order of Flux's ssa package wherever that
// orders two kinds, so an engine that re-sorts with Flux's order can take the
// whole set or any one stage in a call and only refine the library's order,
// never contradict it (0012:D5:R1).
func Stages[T any](items []T, gvkOf func(T) schema.GroupVersionKind) []Stage[T] {
	sorted := make([]T, len(items))
	copy(sorted, items)
	Sort(sorted, gvkOf, Ascending)

	var defs, rest []T
	for _, it := range sorted {
		if isClusterDefinition(gvkOf(it)) {
			defs = append(defs, it)
		} else {
			rest = append(rest, it)
		}
	}

	var stages []Stage[T]
	if len(defs) > 0 {
		stages = append(stages, Stage[T]{ClusterDefinitions: true, Items: defs})
	}
	for i := 0; i < len(rest); {
		w := Weight(gvkOf(rest[i]))
		j := i + 1
		for j < len(rest) && Weight(gvkOf(rest[j])) == w {
			j++
		}
		stages = append(stages, Stage[T]{Items: rest[i:j:j]})
		i = j
	}
	return stages
}

// isClusterDefinition reports whether a GVK is a CustomResourceDefinition of
// apiextensions.k8s.io or a Namespace of the core group, on group and kind
// whatever the version.
func isClusterDefinition(gvk schema.GroupVersionKind) bool {
	switch {
	case gvk.Group == "apiextensions.k8s.io" && gvk.Kind == "CustomResourceDefinition":
		return true
	case gvk.Group == "" && gvk.Kind == "Namespace":
		return true
	default:
		return false
	}
}
