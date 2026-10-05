package lifecycle_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-platform-model/library/opm/k8s/inventory"
	"github.com/open-platform-model/library/opm/k8s/lifecycle"
	"github.com/open-platform-model/library/opm/k8s/ownership"
)

func entriesOf(p lifecycle.DeletionPlan) []inventory.Entry {
	var out []inventory.Entry
	for _, s := range p.Steps() {
		out = append(out, s.Entry)
	}
	return out
}

func TestNewDeletionPlan(t *testing.T) {
	t.Run("entries are ordered for deletion", func(t *testing.T) {
		in := []inventory.Entry{cm, web, ns, hook}
		before := append([]inventory.Entry(nil), in...)
		p := lifecycle.NewDeletionPlan(in, prune, thisUUID)
		assert.Equal(t, []inventory.Entry{hook, web, cm, ns}, entriesOf(p))
		assert.Equal(t, before, in, "the caller's list is unchanged")
	})

	t.Run("equal weights keep their order", func(t *testing.T) {
		b := web
		b.Name = "b"
		a := web
		a.Name = "a"
		p := lifecycle.NewDeletionPlan([]inventory.Entry{b, a}, prune, thisUUID)
		assert.Equal(t, []inventory.Entry{b, a}, entriesOf(p))
	})

	t.Run("protected kinds are marked up front", func(t *testing.T) {
		p := lifecycle.NewDeletionPlan([]inventory.Entry{crd, ns, web}, prune, thisUUID)
		for _, s := range p.Steps() {
			if s.Entry == web {
				assert.Empty(t, s.Skip)
				continue
			}
			assert.Equal(t, ownership.SkipSafetyExcluded, s.Skip, s.Entry.Kind)
		}
	})

	t.Run("a same-named kind in another group is not protected", func(t *testing.T) {
		other := inventory.Entry{Group: "example.com", Kind: "Namespace", Name: "x", Version: "v1"}
		p := lifecycle.NewDeletionPlan([]inventory.Entry{other}, prune, thisUUID)
		assert.Empty(t, p.Steps()[0].Skip)
	})

	t.Run("a prune hands its stale set to the same plan", func(t *testing.T) {
		previous := []inventory.Entry{cm, web, api, ns, hook}
		current := []inventory.Entry{api, ns}
		stale := inventory.StaleSet(previous, current)
		p := lifecycle.NewDeletionPlan(stale, prune, thisUUID)
		assert.Equal(t, []inventory.Entry{hook, web, cm}, entriesOf(p))
	})

	t.Run("changing the listed steps does not change the plan", func(t *testing.T) {
		p := lifecycle.NewDeletionPlan([]inventory.Entry{cm, web}, prune, thisUUID)
		listed := p.Steps()
		listed[0], listed[1] = listed[1], listed[0]
		assert.Equal(t, []inventory.Entry{web, cm}, entriesOf(p))
	})

	t.Run("an empty inventory gives a plan with no steps", func(t *testing.T) {
		assert.Equal(t, 0, lifecycle.NewDeletionPlan(nil, prune, thisUUID).Len())
		assert.Empty(t, lifecycle.NewDeletionPlan([]inventory.Entry{}, prune, thisUUID).Steps())
	})

	t.Run("the plan keeps its policy and owner", func(t *testing.T) {
		pol := lifecycle.Policy{Prune: true, ForceOrphan: true}
		p := lifecycle.NewDeletionPlan([]inventory.Entry{web}, pol, thisUUID)
		assert.Equal(t, pol, p.Policy())
		assert.Equal(t, thisUUID, p.OwnerUUID())
		assert.Equal(t, 1, p.Len())
	})

	t.Run("duplicates are kept", func(t *testing.T) {
		p := lifecycle.NewDeletionPlan([]inventory.Entry{web, web}, prune, thisUUID)
		assert.Equal(t, 2, p.Len())
	})

	t.Run("the zero plan has no steps and does not prune", func(t *testing.T) {
		var p lifecycle.DeletionPlan
		assert.Equal(t, 0, p.Len())
		assert.False(t, p.Policy().Prune)
	})
}
