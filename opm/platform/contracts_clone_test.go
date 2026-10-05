package platform

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// clone keeps nil and empty apart at every level, so a copy compares
// (reflect.DeepEqual, JSON null against []) exactly as the original does.
func TestContractInventoryClone_KeepsNilAndEmptyApart(t *testing.T) {
	for name, inv := range map[string]*ContractInventory{
		"nil": {},
		"empty": {
			DefinedBy:        map[string]string{},
			RequiredBy:       map[string][]string{"a": {}, "b": nil},
			ProvidedBy:       map[string][]string{},
			Unfulfilled:      []string{},
			OverSubscribed:   []string{},
			Comparable:       []ComparablePredicates{{Broader: "x", Narrower: "y"}, {Contracts: []string{}}},
			Collisions:       []string{},
			CollidingEntries: map[string][]string{},
		},
	} {
		t.Run(name, func(t *testing.T) {
			out := inv.clone()
			assert.Equal(t, inv, out)
			assert.Equal(t, inv.DefinedBy == nil, out.DefinedBy == nil)
			assert.Equal(t, inv.RequiredBy == nil, out.RequiredBy == nil)
			assert.Equal(t, inv.Unfulfilled == nil, out.Unfulfilled == nil)
			assert.Equal(t, inv.Comparable == nil, out.Comparable == nil)
			assert.Equal(t, inv.Collisions == nil, out.Collisions == nil)
			for k, v := range inv.RequiredBy {
				assert.Equal(t, v == nil, out.RequiredBy[k] == nil, "requiredBy[%s]", k)
			}
			for i, row := range inv.Comparable {
				assert.Equal(t, row.Contracts == nil, out.Comparable[i].Contracts == nil, "comparable[%d]", i)
			}
		})
	}
}

// clone shares nothing with its source: changing the copy's map entries,
// slice elements and a Comparable row's Contracts leaves the source as it was.
func TestContractInventoryClone_SharesNothing(t *testing.T) {
	src := func() *ContractInventory {
		return &ContractInventory{
			DefinedBy:        map[string]string{"a": "x"},
			RequiredBy:       map[string][]string{"a": {"t"}},
			ProvidedBy:       map[string][]string{"g": {"c1", "c2"}},
			Unfulfilled:      []string{"u"},
			OverSubscribed:   []string{"o"},
			Comparable:       []ComparablePredicates{{Broader: "x", Narrower: "y", Contracts: []string{"a"}}},
			Collisions:       []string{"c"},
			CollidingEntries: map[string][]string{"e": {"1"}},
		}
	}
	inv := src()
	out := inv.clone()
	assert.NotSame(t, inv, out)

	out.DefinedBy["a"] = "changed"
	out.RequiredBy["a"][0] = "changed"
	out.ProvidedBy["g"][1] = "changed"
	out.Unfulfilled[0] = "changed"
	out.OverSubscribed[0] = "changed"
	out.Comparable[0].Contracts[0] = "changed"
	out.Comparable[0].Broader = "changed"
	out.Collisions[0] = "changed"
	out.CollidingEntries["e"][0] = "changed"

	assert.Equal(t, src(), inv)
}
