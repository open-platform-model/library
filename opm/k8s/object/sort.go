package object

import (
	"sort"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Direction selects ascending (apply) or descending (delete) weight order.
type Direction int

const (
	// Ascending puts the lowest weight first: the order to apply in.
	Ascending Direction = iota
	// Descending puts the highest weight first: the order to delete in.
	Descending
)

// Sort orders items in place by [Weight] of each item's GVK, read through
// gvkOf. The sort is stable: items of equal weight keep their relative order.
func Sort[T any](items []T, gvkOf func(T) schema.GroupVersionKind, dir Direction) {
	sort.SliceStable(items, func(i, j int) bool {
		wi, wj := Weight(gvkOf(items[i])), Weight(gvkOf(items[j]))
		if dir == Descending {
			return wi > wj
		}
		return wi < wj
	})
}
