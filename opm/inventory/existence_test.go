package inventory

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func candidate() InventoryEntry {
	return InventoryEntry{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v1", Component: "web"}
}

// Spec: Pure Pre-Apply Collision Predicate — "Collision with a foreign-owned object".
func TestCollidesOnApply_ForeignOwned(t *testing.T) {
	observed := ObservedState{Exists: true, BeingDeleted: false, ManagedBy: "argocd"}
	assert.True(t, CollidesOnApply(candidate(), observed))
}

// Spec: Pure Pre-Apply Collision Predicate — "No live object".
func TestCollidesOnApply_NoLiveObject(t *testing.T) {
	observed := ObservedState{Exists: false}
	assert.False(t, CollidesOnApply(candidate(), observed))
}

// Spec: Pure Pre-Apply Collision Predicate — "Object owned by this release".
func TestCollidesOnApply_OPMOwned(t *testing.T) {
	for _, owner := range []string{ManagedByCLI, ManagedByController, ManagedByLegacy} {
		observed := ObservedState{Exists: true, ManagedBy: owner}
		assert.Falsef(t, CollidesOnApply(candidate(), observed), "OPM owner %q must not collide", owner)
	}
}

// An OPM-owned object that is terminating cannot be safely applied over.
func TestCollidesOnApply_TerminatingCollides(t *testing.T) {
	observed := ObservedState{Exists: true, BeingDeleted: true, ManagedBy: ManagedByCLI}
	assert.True(t, CollidesOnApply(candidate(), observed))
}

// An object with no managed-by attribution is foreign.
func TestCollidesOnApply_UnlabeledIsForeign(t *testing.T) {
	observed := ObservedState{Exists: true, ManagedBy: ""}
	assert.True(t, CollidesOnApply(candidate(), observed))
}

// Spec: Pure Pre-Apply Collision Predicate — "Predicate performs no I/O".
// The predicate reads only its arguments and does not mutate them.
func TestCollidesOnApply_Pure(t *testing.T) {
	entry := candidate()
	entryCopy := entry
	observed := ObservedState{Exists: true, BeingDeleted: false, ManagedBy: "argocd"}
	observedCopy := observed

	first := CollidesOnApply(entry, observed)
	second := CollidesOnApply(entry, observed)

	assert.Equal(t, first, second, "predicate must be deterministic")
	assert.Equal(t, entryCopy, entry, "predicate must not mutate the candidate entry")
	assert.Equal(t, observedCopy, observed, "predicate must not mutate the observed state")
}
