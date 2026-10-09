package ownership

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/library/opm/k8s/labels"
)

// ApplyRefusal says why an apply verdict refuses an object. Each value is the
// contract's literal, so a frontend can report it as it stands. The empty
// value means apply.
type ApplyRefusal string

// The reasons an apply verdict refuses an object.
const (
	// RefuseTerminating: the live object is being deleted. Nothing lifts it.
	RefuseTerminating ApplyRefusal = "terminating"
	// RefuseForeignObject: the live object outside the inventory is not
	// managed by OPM.
	RefuseForeignObject ApplyRefusal = "foreign-object"
	// RefuseOtherInstance: the live object outside the inventory belongs to
	// another module instance.
	RefuseOtherInstance ApplyRefusal = "other-instance"
	// RefuseAdoptedElsewhere: the live object's adopt annotation names
	// another module instance, which is taking the object over or has taken
	// it. A frontend never fails the apply on this refusal, in the inventory
	// or outside it; it drops an inventoried object refused for this from
	// its next inventory and never deletes it.
	RefuseAdoptedElsewhere ApplyRefusal = "adopted-elsewhere"
)

// ApplyInput is what [CanApply] judges.
type ApplyInput struct {
	// Object names the object to apply.
	Object Object
	// Live is the object as the caller read it, nil when it does not exist.
	// CanApply does not change it.
	Live *unstructured.Unstructured
	// InInventory reports whether the object is in the applying instance's
	// recorded inventory. An inventoried object is judged only for an adopt
	// annotation naming another instance; a UUID label naming another
	// instance alone does not refuse it, so the instance's objects still
	// apply after its UUID changes, except an object whose adopt annotation
	// still names the old UUID, which must be re-annotated with the new one.
	// A frontend never fails the apply on an adopted-elsewhere refusal: it
	// drops an inventoried object refused for that from the inventory it
	// records next, keeps applying the instance's other objects, and never
	// deletes the object for that refusal.
	InInventory bool
	// InstanceUUID is the applying instance's UUID, from the render. Empty
	// matches no adopt annotation. Outside the inventory every non-empty
	// live UUID and every non-blank adopt annotation then counts as another
	// instance's; inside it the object applies, since there is no identity to
	// compare against.
	InstanceUUID string
}

// ApplyVerdict is the outcome of [CanApply]: apply, or refuse with a reason.
type ApplyVerdict struct {
	// Refuse is the reason the apply is refused; empty means apply.
	Refuse ApplyRefusal
	// Message words the refusal for a user; empty when the object applies.
	// An ownership refusal ends with how to adopt the object.
	Message string
}

// Allowed reports whether the object may be applied.
func (v ApplyVerdict) Allowed() bool { return v.Refuse == "" }

// CanApply decides whether a frontend may apply over one object. It checks,
// in order, and stops at the first match: a missing live object applies; a
// live object being deleted is refused, in the inventory or not, adopted or
// not; an object whose adopt annotation ([labels.AnnotationAdopt])
// names this instance applies, in the inventory or not. In the instance's
// inventory: with no instance UUID it applies; an adopt annotation naming
// another instance is refused as adopted-elsewhere; otherwise it applies,
// whatever its UUID label says. Outside it: a live object OPM does not manage
// is refused; a live object whose adopt annotation and UUID
// label name the same other instance is refused as adopted-elsewhere, since
// that instance completed the hand-over; a live object carrying another
// instance's UUID is refused as other-instance; an adopt annotation naming
// another instance is refused as adopted-elsewhere; otherwise it applies
// (0012:D8:R1/R2/R5, 0012:D8:R8). The adopt annotation
// is the only override outside the inventory: no input field exists only to
// lift a refusal. InInventory and InstanceUUID are the caller's own ownership
// record, which CanApply trusts and cannot check. A refusal message names the
// annotation with the UUID to set (0012:D8:R3).
func CanApply(in ApplyInput) ApplyVerdict {
	if in.Live == nil {
		return ApplyVerdict{}
	}
	obj := in.Object.String()
	if in.Live.GetDeletionTimestamp() != nil {
		return refuse(RefuseTerminating, obj+" is being deleted; wait for the deletion to finish, then apply again")
	}
	annotation := adoptAnnotation(in.Live)
	if annotation != "" && in.InstanceUUID != "" && annotation == in.InstanceUUID {
		return ApplyVerdict{}
	}
	if in.InInventory {
		return judgeInventoried(in, obj, annotation)
	}
	if !opmManaged(in.Live) {
		return refuse(RefuseForeignObject, obj+" exists and is not managed by OPM"+
			adoptsAnother(annotation)+adoptRemedy("to let this instance take it over,", in.InstanceUUID))
	}
	u := liveUUID(in.Live)
	if annotation != "" && annotation == u {
		return refuse(RefuseAdoptedElsewhere, obj+" was adopted by module instance "+annotation+
			"; this instance does not apply it"+adoptRemedy("to let this instance take it back,", in.InstanceUUID))
	}
	if u != "" && u != in.InstanceUUID {
		return refuse(RefuseOtherInstance, obj+" belongs to module instance "+u+
			adoptsAnother(annotation)+adoptRemedy("to move it to this instance, remove it from module instance "+u+", then", in.InstanceUUID))
	}
	if annotation != "" {
		return refuse(RefuseAdoptedElsewhere, obj+" is being adopted by module instance "+annotation+
			"; this instance does not apply it"+adoptRemedy("to let this instance take it over,", in.InstanceUUID))
	}
	return ApplyVerdict{}
}

// judgeInventoried judges an object in the instance's inventory whose adopt
// annotation does not name this instance. With no instance UUID nothing can
// name another instance, so it applies: refusing would drop the object from
// the inventory, and the prune would then delete the instance's own object.
// Otherwise an adopt annotation naming another instance refuses it as
// adopted-elsewhere. A UUID label naming another instance alone does not, so
// an instance whose UUID changed (a module moved to a new path) keeps
// applying and relabels its own objects, except an object it once adopted:
// that object's annotation still names the old UUID and refuses it until it
// is re-annotated with the new one.
func judgeInventoried(in ApplyInput, obj, annotation string) ApplyVerdict {
	if in.InstanceUUID == "" || annotation == "" {
		return ApplyVerdict{}
	}
	return refuse(RefuseAdoptedElsewhere, obj+" was adopted by module instance "+annotation+
		"; this instance no longer applies it and drops it from its inventory"+
		adoptRemedy("to take it back,", in.InstanceUUID))
}

func refuse(r ApplyRefusal, msg string) ApplyVerdict {
	return ApplyVerdict{Refuse: r, Message: msg}
}

// adoptAnnotation returns the live object's adopt annotation with
// surrounding whitespace trimmed, "" when it has none or its value is blank.
// A UUID never contains whitespace, so trimming loses no safety.
func adoptAnnotation(live *unstructured.Unstructured) string {
	return strings.TrimSpace(live.GetAnnotations()[labels.AnnotationAdopt])
}

// adoptsAnother words an adopt annotation that did not lift the refusal,
// because it names another instance than this one.
func adoptsAnother(annotation string) string {
	if annotation == "" {
		return ""
	}
	return "; its " + labels.AnnotationAdopt + " annotation names another instance (" + annotation + ")"
}

// adoptRemedy words how to adopt the object into this instance. With no
// instance UUID there is nothing to set the annotation to, so no remedy.
func adoptRemedy(lead, instanceUUID string) string {
	if instanceUUID == "" {
		return ""
	}
	return "; " + lead + " annotate it " + labels.AnnotationAdopt + "=" + instanceUUID
}
