package lifecycle

import "strconv"

// Identity says whether the caller can act as the identity that deletes the
// instance's objects.
type Identity string

// The values of [Identity].
const (
	// IdentityAvailable: the caller can act as the deleting identity. A
	// frontend that deletes with its own identity always passes this.
	IdentityAvailable Identity = ""
	// IdentityMissing: the deleting identity does not exist.
	IdentityMissing Identity = "missing"
	// IdentityFailed: obtaining the deleting identity failed for another
	// reason.
	IdentityFailed Identity = "failed"
)

// HoldInput is what [MayReleaseHold] needs beyond the plan and the state.
type HoldInput struct {
	// Identity is whether the caller can act as the deleting identity.
	Identity Identity
}

// HoldReason names why a hold verdict releases or holds. Each value is the
// contract's literal, so a frontend can report it as it stands.
type HoldReason string

// The reasons of a hold verdict.
const (
	// HoldPruneDisabled releases: the policy does not prune, so the objects
	// are left in place.
	HoldPruneDisabled HoldReason = "prune-disabled"
	// HoldInventoryEmpty releases: the plan has no steps.
	HoldInventoryEmpty HoldReason = "inventory-empty"
	// HoldCleanupComplete releases: every step was deleted or skipped.
	HoldCleanupComplete HoldReason = "cleanup-complete"
	// HoldForceOrphan releases: the deleting identity is missing and the
	// policy sets force-orphan, so the objects are left in place.
	HoldForceOrphan HoldReason = "force-orphan"
	// HoldCleanupIncomplete holds: the plan is not finished, or a step failed
	// other than as Forbidden.
	HoldCleanupIncomplete HoldReason = "cleanup-incomplete"
	// HoldCleanupForbidden holds: a step failed as Forbidden.
	HoldCleanupForbidden HoldReason = "cleanup-forbidden"
	// HoldIdentityUnavailable holds: the deleting identity is missing without
	// force-orphan, or could not be obtained.
	HoldIdentityUnavailable HoldReason = "identity-unavailable"
)

// HoldVerdict is the outcome of [MayReleaseHold].
type HoldVerdict struct {
	// Release reports whether the hold may come off.
	Release bool
	// Because is the reason, the contract's literal.
	Because HoldReason
	// Message words the verdict for a user. It names no frontend, so each
	// frontend can add its own remedy.
	Message string
}

// MayReleaseHold decides whether an instance's deletion hold may come off,
// from the plan's policy and steps, the state's outcomes and whether the
// caller can act as the deleting identity. It checks, in order, and stops at
// the first match:
//
//  1. a policy that does not prune releases ([HoldPruneDisabled]);
//  2. a plan with no steps releases ([HoldInventoryEmpty]);
//  3. a missing identity with force-orphan releases ([HoldForceOrphan]);
//  4. a missing or failed identity holds ([HoldIdentityUnavailable]);
//  5. an unfinished plan holds ([HoldCleanupIncomplete]);
//  6. a step failed as Forbidden holds ([HoldCleanupForbidden]);
//  7. any other failed step holds ([HoldCleanupIncomplete]);
//  8. otherwise it releases ([HoldCleanupComplete]); skipped steps count as
//     complete.
//
// Force-orphan lifts only a missing identity, never a failed or unfinished
// cleanup.
func MayReleaseHold(plan DeletionPlan, state State, in HoldInput) HoldVerdict {
	switch {
	case !plan.policy.Prune:
		return release(HoldPruneDisabled, "pruning is disabled; the objects are left in place")
	case plan.Len() == 0:
		return release(HoldInventoryEmpty, "the inventory is empty; there is nothing to delete")
	case in.Identity == IdentityMissing && plan.policy.ForceOrphan:
		return release(HoldForceOrphan, "the deleting identity is missing and orphaning was forced; "+
			objects(plan.Len())+" left in place")
	case in.Identity == IdentityMissing:
		return hold(HoldIdentityUnavailable, "the deleting identity is missing; "+
			objects(plan.Len())+" not deleted")
	case in.Identity != IdentityAvailable:
		return hold(HoldIdentityUnavailable, "the deleting identity could not be obtained; "+
			objects(plan.Len())+" not deleted")
	case state.Next < plan.Len() || state.Awaiting != AwaitNothing:
		return hold(HoldCleanupIncomplete, "the deletion is not finished; "+
			objects(plan.Len()-state.Next)+" still to process")
	}
	forbidden, failedCount := 0, 0
	for _, o := range state.Outcomes {
		if o.Result != ResultFailed {
			continue
		}
		failedCount++
		if o.Failure == FailureForbidden {
			forbidden++
		}
	}
	switch {
	case forbidden > 0:
		return hold(HoldCleanupForbidden, "the deleting identity was forbidden to delete "+
			objects(forbidden)+"; "+objects(failedCount)+" left to retry")
	case failedCount > 0:
		return hold(HoldCleanupIncomplete, objects(failedCount)+" could not be deleted; left to retry")
	default:
		return release(HoldCleanupComplete, "every object was deleted or left in place")
	}
}

func release(r HoldReason, msg string) HoldVerdict {
	return HoldVerdict{Release: true, Because: r, Message: msg}
}

func hold(r HoldReason, msg string) HoldVerdict {
	return HoldVerdict{Because: r, Message: msg}
}

// objects words a count of objects.
func objects(n int) string {
	if n == 1 {
		return "1 object"
	}
	return strconv.Itoa(n) + " objects"
}
