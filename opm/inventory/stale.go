package inventory

// ComputeStaleSet returns the entries present in previous but absent from
// current, where presence is determined by K8sIdentityEqual — Kubernetes object
// identity (Group, Kind, Namespace, Name), component-agnostic. This is the
// single canonical stale-set base for every OPM frontend.
//
// Component moves are deliberately NOT treated as stale here: an entry whose
// Kubernetes identity matches a current entry is retained even if its Component
// differs, because the live object would be patched in place by an SSA apply
// rather than deleted and recreated.
//
// Because this base relation already ignores Component, a component move never
// reaches the returned set, so chaining ApplyComponentRenameSafetyCheck onto
// this function's output is a no-op for moves. The safety check exists for
// callers whose stale set was instead derived from a full, component-aware
// comparison (e.g. a migrated or externally computed inventory); see its doc.
func ComputeStaleSet(previous, current []InventoryEntry) []InventoryEntry {
	if len(previous) == 0 {
		return []InventoryEntry{}
	}

	stale := make([]InventoryEntry, 0, len(previous))
	for _, prev := range previous {
		found := false
		for _, cur := range current {
			if K8sIdentityEqual(prev, cur) {
				found = true
				break
			}
		}
		if !found {
			stale = append(stale, prev)
		}
	}

	return stale
}

// ApplyComponentRenameSafetyCheck returns the stale set with any entry removed
// when that entry shares Kubernetes object identity (K8sIdentityEqual) with a
// current entry under a different Component. The effect is that a resource which
// moves between components is neither pruned nor recreated; it is left in place
// to be re-owned by its new component.
//
// This is a safety net for stale sets produced by a component-aware comparison
// (a migrated inventory, or a caller that diffed on full identity). For a stale
// set produced by ComputeStaleSet — whose base relation already ignores
// Component — it is a no-op, since a moved entry never enters that set.
//
// The function is pure: no I/O, no logging. Callers that wish to observe a
// detected rename do so by diffing the input and output sets at their edge.
func ApplyComponentRenameSafetyCheck(stale, current []InventoryEntry) []InventoryEntry {
	if len(stale) == 0 {
		return stale
	}

	filtered := make([]InventoryEntry, 0, len(stale))
	for _, s := range stale {
		isRename := false
		for _, c := range current {
			if K8sIdentityEqual(s, c) && s.Component != c.Component {
				isRename = true
				break
			}
		}
		if !isRename {
			filtered = append(filtered, s)
		}
	}
	return filtered
}
