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
