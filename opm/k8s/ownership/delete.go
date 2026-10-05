package ownership

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
)

// SkipReason says why a delete verdict leaves an object in place. Each value
// is the contract's literal, so a frontend can report it as it stands. The
// empty value means proceed.
type SkipReason string

// The reasons a delete verdict skips an object.
const (
	// SkipSafetyExcluded: the kind is one OPM never deletes ([SafetyExcluded]).
	SkipSafetyExcluded SkipReason = "safety-excluded"
	// SkipAlreadyAbsent: the read found no live object.
	SkipAlreadyAbsent SkipReason = "already-absent"
	// SkipNotOPMManaged: the live object's managed-by label is not an OPM
	// runtime's value.
	SkipNotOPMManaged SkipReason = "not-opm-managed"
	// SkipOwnerMismatch: the live object belongs to another module instance.
	SkipOwnerMismatch SkipReason = "owner-mismatch"
	// SkipAdoptedElsewhere: the live object's adopt annotation names another
	// module instance, which is taking the object over.
	SkipAdoptedElsewhere SkipReason = "adopted-elsewhere"
)

// DeleteInput is what [CanDelete] judges.
type DeleteInput struct {
	// Object names the object to delete.
	Object Object
	// Live is the object as the caller read it, nil when the read found
	// nothing. CanDelete does not change it.
	Live *unstructured.Unstructured
	// InstanceUUID is the deleting instance's UUID. Empty disables the owner
	// comparison, as does a live object without a UUID label. Empty never
	// matches an adopt annotation, so any non-blank adopt annotation then
	// names another instance.
	InstanceUUID string
	// Admit is set only for an object the caller has proven came from an
	// earlier operator release's install manifest. It lifts the
	// not-opm-managed skip for a Deployment of apps and a RoleBinding or
	// ClusterRoleBinding of rbac.authorization.k8s.io, the kinds the
	// operator install may delete outside the inventory, and only when the
	// live object carries no UUID label at all (0012:D8:R6/R7). It lifts
	// nothing else, adopted-elsewhere included. The library cannot check the
	// proof.
	Admit bool
}

// DeleteVerdict is the outcome of [CanDelete]: proceed, or skip with a
// reason.
type DeleteVerdict struct {
	// Skip is the reason the object is left in place; empty means proceed.
	Skip SkipReason
	// Message words the skip for a user; empty on proceed.
	Message string
	// UID is the judged live object's UID, set on proceed.
	UID types.UID
	// ResourceVersion is the judged live object's resourceVersion, set on
	// proceed. A caller that wants the strict precondition adds it to
	// [DeleteVerdict.Preconditions] itself.
	ResourceVersion string
}

// Proceed reports whether the object may be deleted.
func (v DeleteVerdict) Proceed() bool { return v.Skip == "" }

// Preconditions returns the DELETE precondition for the judged object: its
// UID, and no resourceVersion, since any writer touching the object between
// the read and the DELETE (a status update, a finalizer) would fail a
// resourceVersion precondition. It returns nil when the verdict skips or the
// judged object had no UID, because a precondition on an empty UID never
// matches.
func (v DeleteVerdict) Preconditions() *metav1.Preconditions {
	if !v.Proceed() || v.UID == "" {
		return nil
	}
	uid := v.UID
	return &metav1.Preconditions{UID: &uid}
}

// CanDelete decides whether a frontend may delete one object. It checks, in
// order, and stops at the first match: a safety-excluded kind skips, whatever
// the live object; a missing live object skips as already absent; a live
// object OPM does not manage skips, unless admitted; a live object whose UUID
// label and the instance's UUID are both set and differ skips as another
// instance's; a live object whose adopt annotation (opmodel.dev/adopt)
// is set and does not name this instance skips as adopted elsewhere, so an
// instance never deletes an object it let go of (0012:D8:R8); otherwise it
// proceeds. An object already being deleted proceeds, since deleting it again
// changes nothing.
func CanDelete(in DeleteInput) DeleteVerdict {
	obj := in.Object
	if SafetyExcluded(obj.Group, obj.Kind) {
		return skip(SkipSafetyExcluded, obj.String()+" is a "+obj.Kind+", which OPM never deletes; left in place")
	}
	if in.Live == nil {
		return skip(SkipAlreadyAbsent, obj.String()+" no longer exists")
	}
	if !opmManaged(in.Live) && !admittedForDelete(in) {
		return skip(SkipNotOPMManaged, obj.String()+" is not managed by OPM; left in place")
	}
	if u := liveUUID(in.Live); u != "" && in.InstanceUUID != "" && u != in.InstanceUUID {
		return skip(SkipOwnerMismatch, obj.String()+" belongs to module instance "+u+", not this one; left in place")
	}
	if a := adoptAnnotation(in.Live); a != "" && a != in.InstanceUUID {
		return skip(SkipAdoptedElsewhere, obj.String()+" is being adopted by module instance "+a+", not this one; left in place")
	}
	return DeleteVerdict{UID: in.Live.GetUID(), ResourceVersion: in.Live.GetResourceVersion()}
}

func skip(r SkipReason, msg string) DeleteVerdict {
	return DeleteVerdict{Skip: r, Message: msg}
}

// admittedForDelete reports whether the install admission lifts the
// not-opm-managed skip: the caller admitted the object, it is of a kind the
// operator install may delete outside the inventory, and it carries no OPM
// instance identity at all. This is narrower than apply-side admission, which
// also admits an object carrying this instance's UUID: a deletion outside the
// inventory is for proven objects only, and a proven object carries no
// instance identity (0012:D8:R6/R7).
func admittedForDelete(in DeleteInput) bool {
	return in.Admit && installDeletable(in.Object.Group, in.Object.Kind) &&
		liveUUID(in.Live) == ""
}

// installDeletable reports whether the operator install may delete an object
// of this group and kind outside the inventory: the earlier operator
// Deployment and the superseded role bindings (0012:D8:R7).
func installDeletable(group, kind string) bool {
	switch {
	case group == "apps" && kind == "Deployment":
		return true
	case group == "rbac.authorization.k8s.io" && (kind == "RoleBinding" || kind == "ClusterRoleBinding"):
		return true
	default:
		return false
	}
}
