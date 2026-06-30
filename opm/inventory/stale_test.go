package inventory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Spec: Canonical Stale-Set Computation — "Removed resource is stale".
func TestComputeStaleSet_RemovedIsStale(t *testing.T) {
	previous := []InventoryEntry{
		{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v1", Component: "web"},
		{Group: "", Kind: "Service", Namespace: "ns", Name: "svc", Version: "v1", Component: "web"},
	}
	current := []InventoryEntry{
		{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v1", Component: "web"},
	}
	stale := ComputeStaleSet(previous, current)
	require.Len(t, stale, 1)
	assert.Equal(t, "svc", stale[0].Name)
}

// Spec: Canonical Stale-Set Computation — "Retained resource is not stale",
// including the component-changed case (K8s identity matches → not stale).
func TestComputeStaleSet_RetainedIsNotStale(t *testing.T) {
	previous := []InventoryEntry{
		{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v1", Component: "web"},
	}

	sameComponent := []InventoryEntry{
		{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v2", Component: "web"},
	}
	assert.Empty(t, ComputeStaleSet(previous, sameComponent), "version change must not be stale")

	movedComponent := []InventoryEntry{
		{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v1", Component: "frontend"},
	}
	assert.Empty(t, ComputeStaleSet(previous, movedComponent),
		"component move must not be stale (K8s identity matches)")
}

func TestComputeStaleSet_EmptyPrevious(t *testing.T) {
	stale := ComputeStaleSet(nil, []InventoryEntry{
		{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app"},
	})
	assert.Empty(t, stale)
}

// Spec: Component-Rename Prune Safety — "Component move is not pruned".
func TestApplyComponentRenameSafetyCheck_MoveRemovedFromStale(t *testing.T) {
	current := []InventoryEntry{
		{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v1", Component: "frontend"},
	}
	// A stale set computed on full identity would include the old-component entry;
	// the safety check must drop it because the K8s object still exists in current.
	stale := []InventoryEntry{
		{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v1", Component: "web"},
	}
	filtered := ApplyComponentRenameSafetyCheck(stale, current)
	assert.Empty(t, filtered, "a component move must be removed from the stale set")
}

// Spec: Component-Rename Prune Safety — "Genuinely removed resource still pruned".
func TestApplyComponentRenameSafetyCheck_RemovedStaysStale(t *testing.T) {
	current := []InventoryEntry{
		{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v1", Component: "frontend"},
	}
	stale := []InventoryEntry{
		{Group: "", Kind: "Service", Namespace: "ns", Name: "svc", Version: "v1", Component: "web"},
	}
	filtered := ApplyComponentRenameSafetyCheck(stale, current)
	require.Len(t, filtered, 1, "a genuinely removed resource must remain stale")
	assert.Equal(t, "svc", filtered[0].Name)
}

func TestApplyComponentRenameSafetyCheck_EmptyStale(t *testing.T) {
	assert.Empty(t, ApplyComponentRenameSafetyCheck(nil, []InventoryEntry{
		{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app"},
	}))
}
