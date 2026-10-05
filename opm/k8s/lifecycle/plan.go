package lifecycle

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/open-platform-model/library/opm/k8s/inventory"
	"github.com/open-platform-model/library/opm/k8s/object"
	"github.com/open-platform-model/library/opm/k8s/ownership"
)

// Policy is how an instance's objects are treated when it is deleted.
type Policy struct {
	// Prune removes the instance's objects. False orphans them: the plan
	// names done at once and nothing is read or deleted. The zero Policy
	// does not prune, so every caller states it: a frontend deleting or
	// pruning on a user's explicit request passes true.
	Prune bool
	// ForceOrphan lets the deletion hold come off when the deleting identity
	// is missing, leaving the objects in place. It lifts nothing else.
	ForceOrphan bool
}

// Step is one inventory entry in a deletion plan.
type Step struct {
	// Entry is the object to delete.
	Entry inventory.Entry
	// Skip is [ownership.SkipSafetyExcluded] when the plan leaves the object
	// in place without reading it, and empty otherwise.
	Skip ownership.SkipReason
}

// DeletionPlan is the ordered list of objects a deletion removes, with its
// policy and the deleting instance's UUID. Build it with [NewDeletionPlan];
// the zero DeletionPlan has no steps and does not prune.
type DeletionPlan struct {
	ownerUUID string
	policy    Policy
	steps     []Step
}

// NewDeletionPlan orders entries for deletion: by descending kind weight
// ([object.Weight]), keeping the relative order of entries of equal weight.
// It marks every step whose group and kind [ownership.SafetyExcluded] matches
// as skipped up front. It copies entries and never changes the caller's
// slice. Duplicate entries are kept; the later one reads the object again,
// which is harmless. ownerUUID is the deleting instance's UUID, compared with
// each live object's UUID label; empty disables that comparison.
func NewDeletionPlan(entries []inventory.Entry, policy Policy, ownerUUID string) DeletionPlan {
	steps := make([]Step, len(entries))
	for i, e := range entries {
		steps[i] = Step{Entry: e}
		if ownership.SafetyExcluded(e.Group, e.Kind) {
			steps[i].Skip = ownership.SkipSafetyExcluded
		}
	}
	object.Sort(steps, stepGVK, object.Descending)
	return DeletionPlan{ownerUUID: ownerUUID, policy: policy, steps: steps}
}

// Steps returns a copy of the plan's steps in deletion order, so a frontend
// can list the plan before it acts. Changing the copy does not change the
// plan.
func (p DeletionPlan) Steps() []Step {
	out := make([]Step, len(p.steps))
	copy(out, p.steps)
	return out
}

// Policy returns the plan's deletion policy.
func (p DeletionPlan) Policy() Policy { return p.policy }

// OwnerUUID returns the deleting instance's UUID the plan judges against.
func (p DeletionPlan) OwnerUUID() string { return p.ownerUUID }

// Len returns the number of steps in the plan.
func (p DeletionPlan) Len() int { return len(p.steps) }

func stepGVK(s Step) schema.GroupVersionKind {
	return schema.GroupVersionKind{Group: s.Entry.Group, Version: s.Entry.Version, Kind: s.Entry.Kind}
}

// objectOf names a step's entry the way the ownership verdicts do.
func objectOf(e inventory.Entry) ownership.Object {
	return ownership.Object{Group: e.Group, Kind: e.Kind, Namespace: e.Namespace, Name: e.Name}
}
