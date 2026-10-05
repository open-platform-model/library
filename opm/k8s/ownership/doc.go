// Package ownership decides, for one Kubernetes object at a time, whether a
// frontend may apply over it ([CanApply]) and whether it may delete it
// ([CanDelete]). The verdicts are the same for every frontend, so the cli and
// the operator never judge ownership in code of their own.
//
// The verdicts are pure functions of explicit inputs. The caller reads the
// live object with its own client and hands it in, nil when the read found
// nothing; the package reads no cluster, no clock and no environment, logs
// nothing, and never changes the object it is given. What a frontend does
// with a read error is the frontend's own policy.
//
// A frontend consults a verdict on every apply, prune and delete path
// (0012:D4:R1/R2). A refusal or skip carries a reason, which is the
// contract's literal, and a message the library words, so both frontends
// report it in the same words.
//
// The apply verdict guards objects outside the applying instance's inventory:
// it refuses one OPM does not manage and one that belongs to another module
// instance. The one override is the adopt annotation,
// [labels.AnnotationAdopt], which a user sets on the live object with the
// adopting instance's UUID as its value; a refusal message names it and the
// UUID to set. An object being deleted is refused on every apply, and nothing
// lifts that.
//
// Inside the inventory the apply verdict judges only another instance's
// adoption. When a user annotates an object for another instance, the
// instance that held it refuses it. A frontend drops an object refused as
// adopted-elsewhere from the inventory it records next, keeps applying the
// instance's other objects, and never deletes the object for that refusal.
// The delete verdict leaves an object annotated for another instance in
// place, so the instance that let go of it never prunes it. Annotating the object back for the instance that held it reverses the
// hand-over.
//
// A proceed verdict from [CanDelete] carries the UID and resourceVersion of
// the live object it judged, and [DeleteVerdict.Preconditions] turns the UID
// into a DELETE precondition, so the DELETE removes the object that was
// judged and not one recreated under the same name since the read. A frontend
// reports a DELETE that fails its precondition as left behind or to retry,
// never as deleted.
package ownership

// Maintainer pointers, kept out of the package doc because it publishes into
// the Library reference: the purity rule follows ADR-008 (the kernel plans,
// the caller runs) and the package's place follows ADR-011 (the Kubernetes
// tier beside the kernel).
