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
)

// ApplyInput is what [CanApply] judges.
type ApplyInput struct {
	// Object names the object to apply.
	Object Object
	// Live is the object as the caller read it, nil when it does not exist.
	// CanApply does not change it.
	Live *unstructured.Unstructured
	// InInventory reports whether the object is in the applying instance's
	// recorded inventory. Only objects outside it are judged for ownership.
	InInventory bool
	// InstanceUUID is the applying instance's UUID, from the render. Empty
	// matches no adopt annotation, and every non-empty live UUID then counts
	// as another instance's.
	InstanceUUID string
	// Admit is set only for an object the caller has proven came from an
	// earlier operator release's install manifest. It lifts the
	// foreign-object refusal when the live object carries no UUID label or
	// InstanceUUID (0012:D8:R6). It lifts nothing else. The library cannot
	// check the proof.
	Admit bool
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
// admitted; an object in the instance's inventory applies; an object whose
// adopt annotation ([labels.AnnotationAdopt]) names this instance applies; a
// live object OPM does not manage is refused, unless admitted; a live object
// carrying another instance's UUID is refused; otherwise it applies
// (0012:D8:R1/R2/R5). The adopt annotation is the only override, and a
// refusal message names it with the UUID to set (0012:D8:R3).
func CanApply(in ApplyInput) ApplyVerdict {
	if in.Live == nil {
		return ApplyVerdict{}
	}
	obj := in.Object.String()
	if in.Live.GetDeletionTimestamp() != nil {
		return refuse(RefuseTerminating, obj+" is being deleted; wait for the deletion to finish, then apply again")
	}
	if in.InInventory {
		return ApplyVerdict{}
	}
	annotation := adoptAnnotation(in.Live)
	if annotation != "" && in.InstanceUUID != "" && annotation == in.InstanceUUID {
		return ApplyVerdict{}
	}
	if !opmManaged(in.Live) && !admittedForApply(in) {
		return refuse(RefuseForeignObject, obj+" exists and is not managed by OPM"+
			adoptsAnother(annotation)+adoptRemedy("to let this instance take it over,", in.InstanceUUID))
	}
	if u := liveUUID(in.Live); u != "" && u != in.InstanceUUID {
		return refuse(RefuseOtherInstance, obj+" belongs to module instance "+u+
			adoptsAnother(annotation)+adoptRemedy("to move it to this instance, remove it from module instance "+u+", then", in.InstanceUUID))
	}
	return ApplyVerdict{}
}

// admittedForApply reports whether the install admission lifts the
// foreign-object refusal: the caller admitted the object and it carries no
// other instance's UUID.
func admittedForApply(in ApplyInput) bool {
	return in.Admit && carriesNoOtherIdentity(in.Live, in.InstanceUUID)
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
