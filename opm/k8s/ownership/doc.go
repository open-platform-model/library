// Package ownership decides, for one Kubernetes object at a time, whether a
// frontend may delete it. The verdict is the same for every frontend, so the
// cli and the operator never judge ownership in code of their own.
//
// The verdicts are pure functions of explicit inputs. The caller reads the
// live object with its own client and hands it in, nil when the read found
// nothing; the package reads no cluster, no clock and no environment, logs
// nothing, and never changes the object it is given. What a frontend does
// with a read error is the frontend's own policy.
//
// A frontend consults a verdict on every delete path, prune and instance
// deletion alike (0012:D4:R1/R2). A skip carries a reason, which is the
// contract's literal, and a message the library words, so both frontends
// report a skip in the same words.
//
// A proceed verdict from [CanDelete] carries the UID and resourceVersion of
// the live object it judged, and [DeleteVerdict.Preconditions] turns the UID
// into a DELETE precondition, so the DELETE removes the object that was
// judged and not one recreated under the same name since the read. A frontend
// reports a DELETE that fails its precondition as left behind or to retry,
// never as deleted.
package ownership
