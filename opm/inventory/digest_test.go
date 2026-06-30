package inventory

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Spec: Deterministic Content Digest — "Order independence".
func TestComputeDigest_OrderIndependent(t *testing.T) {
	entries := []InventoryEntry{
		{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v1", Component: "web"},
		{Group: "", Kind: "Service", Namespace: "ns", Name: "svc", Version: "v1", Component: "web"},
		{Group: "", Kind: "ConfigMap", Namespace: "ns", Name: "cfg", Version: "v1", Component: "data"},
	}
	reversed := []InventoryEntry{entries[2], entries[1], entries[0]}
	assert.Equal(t, ComputeDigest(entries), ComputeDigest(reversed),
		"digest must be identical regardless of input order")
}

// Spec: Deterministic Content Digest — "Content sensitivity".
// Every field that participates in the encoding must change the digest, so the
// serialization path of each one is exercised (including Component, whose
// omitempty tag means absent vs present is a structural JSON difference).
func TestComputeDigest_ContentSensitive(t *testing.T) {
	base := InventoryEntry{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v1", Component: "web"}
	baseDigest := ComputeDigest([]InventoryEntry{base})

	mutations := map[string]InventoryEntry{
		"group":            {Group: "extensions", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v1", Component: "web"},
		"kind":             {Group: "apps", Kind: "StatefulSet", Namespace: "ns", Name: "app", Version: "v1", Component: "web"},
		"namespace":        {Group: "apps", Kind: "Deployment", Namespace: "other", Name: "app", Version: "v1", Component: "web"},
		"name":             {Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "other", Version: "v1", Component: "web"},
		"version":          {Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v2", Component: "web"},
		"component":        {Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v1", Component: "data"},
		"component-absent": {Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v1"},
	}
	for field, mutated := range mutations {
		t.Run(field, func(t *testing.T) {
			assert.NotEqualf(t, baseDigest, ComputeDigest([]InventoryEntry{mutated}),
				"a change to %s must change the digest", field)
		})
	}
}

func TestComputeDigest_Empty(t *testing.T) {
	const emptySHA = "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	assert.Equal(t, emptySHA, ComputeDigest(nil))
	assert.Equal(t, emptySHA, ComputeDigest([]InventoryEntry{}))
}
