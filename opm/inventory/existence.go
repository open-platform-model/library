package inventory

// Managed-by attribution values identifying OPM runtime actors. They mirror the
// constants in the operator and CLI core packages and are duplicated here to
// keep this package free of any runtime-specific dependency.
const (
	// LabelManagedBy is the standard Kubernetes label key indicating the manager.
	LabelManagedBy = "app.kubernetes.io/managed-by"
	// ManagedByCLI is the attribution value for the CLI actor.
	ManagedByCLI = "opm-cli"
	// ManagedByController is the attribution value for the controller actor.
	ManagedByController = "opm-controller"
	// ManagedByLegacy is the attribution value used before runtime-owned labels
	// were introduced. Recognized for backward compatibility.
	ManagedByLegacy = "open-platform-model"
)

// IsOPMManagedBy reports whether a managed-by attribution value identifies any
// OPM runtime actor (CLI, controller, or the legacy value). An object carrying
// any of these is considered owned by OPM and safe for a release to take over;
// any other value (including the empty string) is foreign.
func IsOPMManagedBy(value string) bool {
	switch value {
	case ManagedByCLI, ManagedByController, ManagedByLegacy:
		return true
	default:
		return false
	}
}

// ObservedState is the neutral, already-fetched state of the live cluster object
// at a candidate entry's identity. The caller populates it from its own cluster
// read; this package never fetches it. Its fields are exactly those the
// pre-apply collision decision inspects — no speculative additions.
type ObservedState struct {
	// Exists reports whether a live object was found at the candidate identity.
	Exists bool
	// BeingDeleted reports whether the live object is terminating (its
	// deletionTimestamp is set).
	BeingDeleted bool
	// ManagedBy is the value of the live object's app.kubernetes.io/managed-by
	// label, or "" if the label is absent.
	ManagedBy string
}

// CollidesOnApply reports whether applying the candidate entry would collide
// with the live cluster object whose already-observed state is supplied. A
// collision means the apply cannot proceed safely without operator intervention.
//
// The decision, mirroring the field usage of the CLI's pre-apply existence
// check, is:
//
//   - no live object exists                          → no collision
//   - the live object is terminating (being deleted) → collision
//   - the live object is foreign-owned               → collision
//   - the live object is OPM-owned and not deleting  → no collision
//
// The verdict currently derives entirely from observed; the candidate entry is
// part of the signature so the contract reads as "does applying THIS entry
// collide" at every call site and so identity-specific rules (e.g. never
// flagging a collision for a particular Kind) can be added later without a
// signature break. It is deliberately not consumed yet.
//
// The function is pure: it reads only its arguments and performs no cluster
// read, apply, delete, or logging. Fetching the live state into ObservedState
// and reacting to the verdict are caller responsibilities.
func CollidesOnApply(entry InventoryEntry, observed ObservedState) bool {
	_ = entry // see doc: part of the contract, intentionally not yet consumed
	if !observed.Exists {
		return false
	}
	if observed.BeingDeleted {
		return true
	}
	return !IsOPMManagedBy(observed.ManagedBy)
}
