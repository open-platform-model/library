// Package lifecycle is the deletion protocol every Kubernetes frontend runs
// when it removes an instance's objects, on an uninstall and on a prune. It
// plans and names actions; the frontend performs each one with its own
// client.
//
// [NewDeletionPlan] builds the plan from inventory entries: the persisted
// inventory for an uninstall, or the stale set [inventory.StaleSet] computes
// for a prune. Nothing is rendered and no plan is stored. The plan orders the
// entries for deletion, by descending kind weight ([object.Descending]), and
// marks each CustomResourceDefinition and Namespace as left in place before
// any live object is read ([ownership.SafetyExcluded]). It is the only way to
// build a plan, so every frontend deletes in the same order.
//
// The caller owns the [State]. It is a plain value with a fixed JSON
// encoding, and the zero State is the start of a plan. The package keeps
// nothing between calls, so a state written out and read back advances
// exactly as the one held in memory.
//
// Each call to [Advance] names one [Action]: read one step's live object,
// delete it, report it as skipped, or done. The frontend performs a read or a
// delete with its own client and hands back what it saw in the next [Event]:
// the object read (nil when not found) and the raw error. The package
// classifies the error itself. Every step it judges goes through
// [ownership.CanDelete], so a frontend has no way to skip the ownership guard.
// A failure is recorded and the plan moves on to the next step. Each call
// appends the outcomes of the steps it finished, in step order, so a frontend
// reports the outcomes appended since the state it passed in.
//
// Every delete carries Foreground propagation and a precondition on the UID of
// the object that was judged. A frontend reports a delete that fails its
// precondition as left behind or to retry, never as deleted.
//
// [MayReleaseHold] decides whether an instance's deletion hold may come off,
// from the plan's policy, the outcome and whether the caller could act as the
// deleting identity. Its inputs name no frontend and no finalizer, and how a
// frontend surfaces a hold is its own policy.
//
// The protocol has no hook semantics. Its actions are reads, deletions and
// skips of the plan's own inventory entries, and the hold verdict; no step a
// module declares runs as part of a deletion. The package reads no cluster,
// no clock and no environment, starts no goroutine, logs nothing, and never
// changes the entries, live objects or errors it is given.
package lifecycle

// Maintainer pointers, kept out of the package doc because it publishes into
// the Library reference: the package follows ADR-008 rules 1 to 3 (the
// library plans and the caller runs, one step per call from a caller-owned
// state, actions named and never performed) and its Deletion plans
// consequence (the plan comes from the persisted inventory plus the live
// objects, with no render and no stored plan); ADR-011 item 6 gives the
// library the whole deletion sequence; the contract is 0012:D4.
